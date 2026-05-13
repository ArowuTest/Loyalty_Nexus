package external

import (
	"bytes"
	"context"
	"log"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ─── VTPass Client ───────────────────────────────────────────────────────────
// Handles all VTPass API calls: airtime purchase, data purchase, requery,
// and live variation (bundle) catalog fetch.
//
// Sandbox vs. live is controlled by a single VTPASS_SANDBOX env var.
// GAP-7 fix: one flag controls both the base URL and the service mode — no
// divergence between DB provider_mode and isSandbox env var.
// GAP-5 fix: credentials are validated at construction time in NewVTPassHTTPClient().

type VTPassHTTPClient struct {
	apiKey    string
	publicKey string
	secretKey string
	baseURL   string
	isSandbox bool
	http      *http.Client
}

// VTPassPurchaseResult is the normalised result of an airtime/data purchase.
type VTPassPurchaseResult struct {
	RequestID   string
	Success     bool
	Pending     bool // VTPass returned code=011 or code=000+initiated — needs requery
	Failed      bool
	Status      string
	Description string
	ProviderRef string // VTPass internal transactionId
}

// VTPassVariation is one data bundle option from the VTPass catalog.
type VTPassVariation struct {
	Code   string  `json:"variation_code"`
	Name   string  `json:"name"`
	Amount float64 `json:"variation_amount_parsed"` // parsed from string
}

// ── Service IDs ──────────────────────────────────────────────────────────────
var vtpassAirtimeIDs = map[string]string{
	"MTN":     "mtn",
	"GLO":     "glo",
	"AIRTEL":  "airtel",
	"9MOBILE": "etisalat",
}
var vtpassDataIDs = map[string]string{
	"MTN":     "mtn-data",
	"GLO":     "glo-data",
	"AIRTEL":  "airtel-data",
	"9MOBILE": "etisalat-data",
}

// ── Constructor ──────────────────────────────────────────────────────────────

func NewVTPassHTTPClient() (*VTPassHTTPClient, error) {
	sandbox := os.Getenv("VTPASS_SANDBOX") == "true"

	apiKey    := os.Getenv("VTPASS_API_KEY")
	publicKey := os.Getenv("VTPASS_PUBLIC_KEY")
	secretKey := os.Getenv("VTPASS_SECRET_KEY")

	// GAP-5: fail fast in production if credentials are missing
	if !sandbox && (apiKey == "" || publicKey == "" || secretKey == "") {
		return nil, fmt.Errorf("VTPass: VTPASS_API_KEY, VTPASS_PUBLIC_KEY and VTPASS_SECRET_KEY must all be set in production (VTPASS_SANDBOX!=true)")
	}

	baseURL := "https://vtpass.com/api"
	if sandbox {
		baseURL = "https://sandbox.vtpass.com/api"
	}
	// Allow full override (e.g. for mock server in tests)
	if override := os.Getenv("VTPASS_BASE_URL"); override != "" {
		baseURL = override
	}

	return &VTPassHTTPClient{
		apiKey:    apiKey,
		publicKey: publicKey,
		secretKey: secretKey,
		baseURL:   baseURL,
		isSandbox: sandbox,
		http:      &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// NewVTPassHTTPClientUnchecked returns a VTPassHTTPClient without credential validation.
// Used as a fallback when NewVTPassHTTPClient fails in sandbox mode — the client
// will still work for network listing (uses DB, not VTPass), and will return an
// error on actual purchase attempts if credentials are missing.
func NewVTPassHTTPClientUnchecked() (*VTPassHTTPClient, error) {
	sandbox := true // always sandbox when unchecked
	baseURL := "https://sandbox.vtpass.com/api"
	if override := os.Getenv("VTPASS_BASE_URL"); override != "" {
		baseURL = override
	}
	return &VTPassHTTPClient{
		apiKey:    os.Getenv("VTPASS_API_KEY"),
		publicKey: os.Getenv("VTPASS_PUBLIC_KEY"),
		secretKey: os.Getenv("VTPASS_SECRET_KEY"),
		baseURL:   baseURL,
		isSandbox: sandbox,
		http:      &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// ── Airtime purchase ─────────────────────────────────────────────────────────

func (c *VTPassHTTPClient) PurchaseAirtime(ctx context.Context, network, phone string, amountNaira int) (*VTPassPurchaseResult, error) {
	svcID, ok := vtpassAirtimeIDs[network]
	if !ok {
		return nil, fmt.Errorf("vtpass: unsupported network %q", network)
	}
	reqID := c.newRequestID()
	phoneLocal := formatPhoneLocal(phone)
	log.Printf("[VTPass] PurchaseAirtime: svcID=%s phone=%s->%s amount=%d reqID=%s sandbox=%v url=%s",
		svcID, phone, phoneLocal, amountNaira, reqID, c.isSandbox, c.baseURL)
	body := map[string]interface{}{
		"request_id": reqID,
		"serviceID":  svcID,
		"amount":     amountNaira,
		"phone":      phoneLocal,
	}
	return c.doPurchase(ctx, reqID, body)
}

// ── Data purchase ─────────────────────────────────────────────────────────────

func (c *VTPassHTTPClient) PurchaseData(ctx context.Context, network, phone, variationCode string, amountNaira int) (*VTPassPurchaseResult, error) {
	svcID, ok := vtpassDataIDs[network]
	if !ok {
		return nil, fmt.Errorf("vtpass: unsupported network %q", network)
	}
	local := formatPhoneLocal(phone)
	reqID := c.newRequestID()
	log.Printf("[VTPass] PurchaseData: svcID=%s phone=%s->%s variation=%s amount=%d reqID=%s sandbox=%v url=%s",
		svcID, phone, local, variationCode, amountNaira, reqID, c.isSandbox, c.baseURL)
	body := map[string]interface{}{
		"request_id":     reqID,
		"serviceID":      svcID,
		"amount":         amountNaira,
		"phone":          local,
		"billersCode":    local,
		"variation_code": variationCode,
	}
	return c.doPurchase(ctx, reqID, body)
}

// ── Requery (for PENDING transactions) ───────────────────────────────────────

func (c *VTPassHTTPClient) RequeryTransaction(ctx context.Context, requestID string) (*VTPassPurchaseResult, error) {
	url := c.baseURL + "/requery"
	payload, _ := json.Marshal(map[string]string{"request_id": requestID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("secret-key", c.secretKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	return c.parseResponse(requestID, raw), nil
}

// ── GetVariations: live bundle catalog from VTPass ────────────────────────────
// Called by NetworkBundleService (backed by 1-hour cache). GAP data-bundle fix.

func (c *VTPassHTTPClient) GetVariations(ctx context.Context, network string) ([]VTPassVariation, error) {
	svcID, ok := vtpassDataIDs[network]
	if !ok {
		return nil, fmt.Errorf("vtpass: unsupported network %q", network)
	}
	url := fmt.Sprintf("%s/service-variations?serviceID=%s", c.baseURL, svcID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("secret-key", c.secretKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)

	var parsed struct {
		Code    string `json:"code"`                 // live API uses "code"
		RespDesc string `json:"response_description"` // sandbox uses "response_description"
		Content struct {
			Variations []struct {
				Code   string `json:"variation_code"`
				Name   string `json:"name"`
				Amount string `json:"variation_amount"` // VTPass sends as string e.g. "100.00"
			} `json:"varations"` // Note: VTPass has a typo — "varations" not "variations"
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("vtpass: parse variations: %w", err)
	}
	// VTPass sandbox returns "response_description":"000"; live API returns "code":"000"
	statusCode := parsed.Code
	if statusCode == "" {
		statusCode = parsed.RespDesc
	}
	if statusCode != "000" {
		return nil, fmt.Errorf("vtpass: GetVariations error code %q", statusCode)
	}

	out := make([]VTPassVariation, 0, len(parsed.Content.Variations))
	for _, v := range parsed.Content.Variations {
		var amount float64
		_, _ = fmt.Sscanf(v.Amount, "%f", &amount) // parse "100.00" → 100.0
		out = append(out, VTPassVariation{
			Code:   v.Code,
			Name:   v.Name,
			Amount: amount,
		})
	}
	return out, nil
}

// ── Internal helpers ─────────────────────────────────────────────────────────

func (c *VTPassHTTPClient) doPurchase(ctx context.Context, reqID string, body map[string]interface{}) (*VTPassPurchaseResult, error) {
	url := c.baseURL + "/pay"
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("secret-key", c.secretKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	log.Printf("[VTPass] response: httpStatus=%d body=%s", resp.StatusCode, string(raw))
	return c.parseResponse(reqID, raw), nil
}

func (c *VTPassHTTPClient) parseResponse(reqID string, raw []byte) *VTPassPurchaseResult {
	var r struct {
		Code                string `json:"code"`
		ResponseDescription string `json:"response_description"`
		RequestID           string `json:"requestId"`
		Content             struct {
			Transactions struct {
				Status        string `json:"status"`
				TransactionID string `json:"transactionId"`
			} `json:"transactions"`
		} `json:"content"`
	}
	_ = json.Unmarshal(raw, &r)

	if r.RequestID == "" {
		r.RequestID = reqID // VTPass omits requestId on PROCESSING responses
	}

	res := &VTPassPurchaseResult{
		RequestID:   r.RequestID,
		Description: r.ResponseDescription,
		ProviderRef: r.Content.Transactions.TransactionID,
		Status:      r.Content.Transactions.Status,
	}

	switch {
	case r.Code == "000" && (r.Content.Transactions.Status == "delivered" || r.Content.Transactions.Status == "success"):
		res.Success = true
	case r.Code == "011" || r.Code == "099":
		res.Pending = true
	case r.Code == "000" && (r.Content.Transactions.Status == "initiated" || r.Content.Transactions.Status == "pending" || r.Content.Transactions.Status == ""):
		res.Pending = true
	case r.Code == "015":
		res.Failed = true
	case r.Code == "016": // reversed — treat as pending for requery
		res.Pending = true
	default:
		res.Failed = true
	}
	return res
}

func (c *VTPassHTTPClient) newRequestID() string {
	return time.Now().Format("20060102150405") + uuid.New().String()[:8]
}

// formatPhoneLocal converts 234XXXXXXXXXX → 0XXXXXXXXXX (VTPass expects local format)
func formatPhoneLocal(phone string) string {
	digits := ""
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			digits += string(ch)
		}
	}
	if len(digits) == 13 && digits[:3] == "234" {
		return "0" + digits[3:]
	}
	return digits
}

// ─── NetworkBundleService ────────────────────────────────────────────────────
// Live VTPass bundle catalog with per-network 1-hour in-memory cache.
// Implements the GAP data-bundle fix: DataBundleResponse.ID = variation_code.

type NetworkBundleService struct {
	vtpass *VTPassHTTPClient
	mu     sync.RWMutex
	cache  map[string]bundleCacheEntry
}

type bundleCacheEntry struct {
	bundles   []DataBundleResponse
	expiresAt time.Time
}

// DataBundleResponse is the public DTO returned by GET /api/v1/recharge/networks/{code}/bundles
type DataBundleResponse struct {
	ID       string  `json:"id"`       // VTPass variation_code e.g. "mtn-10mb-100"
	Name     string  `json:"name"`
	Network  string  `json:"network"`
	Price    float64 `json:"price"`    // naira
	DataSize string  `json:"data_size"`// derived from name if not explicit
}

// NetworkResponse is the public DTO returned by GET /api/v1/recharge/networks
type NetworkResponse struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Logo           string `json:"logo"`
	BrandColor     string `json:"brand_color"`
	IsActive       bool   `json:"is_active"`
	AirtimeEnabled bool   `json:"airtime_enabled"`
	DataEnabled    bool   `json:"data_enabled"`
	SortOrder      int    `json:"sort_order"`
}

func NewNetworkBundleService(vtpass *VTPassHTTPClient) *NetworkBundleService {
	return &NetworkBundleService{
		vtpass: vtpass,
		cache:  make(map[string]bundleCacheEntry),
	}
}

const bundleCacheTTL = 1 * time.Hour

// GetBundles returns live data bundles for the given network, using a 1h cache.
func (s *NetworkBundleService) GetBundles(ctx context.Context, networkCode string) ([]DataBundleResponse, error) {
	s.mu.RLock()
	entry, ok := s.cache[networkCode]
	s.mu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.bundles, nil
	}

	// Cache miss — fetch live
	variations, err := s.vtpass.GetVariations(ctx, networkCode)
	if err != nil {
		// Return cached (stale) if available rather than a hard error
		if ok {
			return entry.bundles, nil
		}
		return nil, err
	}

	bundles := make([]DataBundleResponse, 0, len(variations))
	for _, v := range variations {
		bundles = append(bundles, DataBundleResponse{
			ID:       v.Code,   // ← variation_code is the ID — GAP fix
			Name:     v.Name,
			Network:  networkCode,
			Price:    v.Amount,
			DataSize: extractDataSize(v.Name),
		})
	}

	s.mu.Lock()
	s.cache[networkCode] = bundleCacheEntry{bundles: bundles, expiresAt: time.Now().Add(bundleCacheTTL)}
	s.mu.Unlock()

	return bundles, nil
}

// extractDataSize attempts to pull a human-readable size from the plan name.
// e.g. "MTN 500MB Data (30 days)" → "500MB"
func extractDataSize(name string) string {
	units := []string{"TB", "GB", "MB", "KB"}
	for _, u := range units {
		idx := -1
		for i := 0; i < len(name)-len(u); i++ {
			if name[i+len(u)-1:i+len(u)] == string(u[len(u)-1]) {
				match := true
				for j := 0; j < len(u); j++ {
					if i+j >= len(name) || name[i+j] != u[j] {
						match = false
						break
					}
				}
				if match {
					idx = i
					break
				}
			}
		}
		if idx > 0 {
			start := idx - 1
			for start > 0 && (name[start-1] >= '0' && name[start-1] <= '9' || name[start-1] == '.') {
				start--
			}
			return name[start : idx+len(u)]
		}
	}
	return ""
}
