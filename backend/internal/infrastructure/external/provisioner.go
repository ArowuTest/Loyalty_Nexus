package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// VTPassAdapter implements VTPassClient against the VTPass REST API.
// Ref: https://vtpass.com/documentation
//
// Design:
//   - baseURL is chosen at construction time based on VTPASS_SANDBOX env var.
//   - Credentials (api-key, public-key, secret-key) are read FRESH from env on
//     every call so that credential rotation takes effect without a restart.
//   - request_id is formatted as required by VTPass: YYYYMMDDHHIISS + truncated ref.
//   - bundleSvc (optional) enables DB-first variation code lookup for data prizes.
//     Set via SetBundleService after construction. Falls back to networkDataCode()
//     when nil or when no matching bundle is found in the DB.
type VTPassAdapter struct {
	baseURL   string
	isSandbox bool
	client    *http.Client
	bundleSvc *NetworkBundleService // optional; wired in main.go after both are constructed
}

// SetBundleService wires the NetworkBundleService into the adapter so that
// TopUpData uses real variation codes from the synced DB catalog instead of
// the hardcoded networkDataCode() fallback map.
func (v *VTPassAdapter) SetBundleService(svc *NetworkBundleService) {
	v.bundleSvc = svc
}

func NewVTPassAdapter() *VTPassAdapter {
	sandbox := os.Getenv("VTPASS_SANDBOX") == "true"
	baseURL := "https://vtpass.com/api"
	if sandbox {
		baseURL = "https://sandbox.vtpass.com/api"
	}
	return &VTPassAdapter{
		baseURL:   baseURL,
		isSandbox: sandbox,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// vtpassRequestID formats an idempotency key acceptable to VTPass.
// VTPass requires: YYYYMMDDHHIISS + alphanumeric suffix, max 50 chars.
func vtpassRequestID(ref string) string {
	ts := time.Now().Format("20060102150405")
	// Sanitise ref: strip non-alphanumeric, truncate
	safe := ""
	for _, c := range ref {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			safe += string(c)
		}
		if len(safe) >= 20 {
			break
		}
	}
	return ts + safe // e.g. "20260514143022LNabc123"
}

// credentials reads VTPass API credentials fresh from the environment.
// Called on every outbound request so rotation takes effect immediately.
func (v *VTPassAdapter) credentials() (apiKey, pubKey, secretKey string) {
	return os.Getenv("VTPASS_API_KEY"), os.Getenv("VTPASS_PUBLIC_KEY"), os.Getenv("VTPASS_SECRET_KEY")
}

func (v *VTPassAdapter) TopUpAirtime(ctx context.Context, phone, network string, amountNaira float64, ref string) (string, error) {
	reqID := vtpassRequestID(ref)
	payload := map[string]interface{}{
		"request_id": reqID,
		"serviceID":  networkToVTPassID(network),
		"amount":     int(amountNaira), // VTPass expects integer naira amount
		"phone":      phone,
	}
	return v.post(ctx, "/pay", payload)
}

func (v *VTPassAdapter) TopUpData(ctx context.Context, phone, network string, amountNaira float64, ref string) (string, error) {
	serviceID := networkToVTPassDataID(network)

	// ── DB-first variation code lookup ─────────────────────────────────────────
	// NetworkBundleService keeps network_data_bundles fresh (5×/day via DataBundleSyncJob).
	// Using the real catalog is more reliable than the hardcoded networkDataCode() map,
	// especially for networks where the ₦price→variation mapping changes frequently (MTN).
	var variationCode string
	if v.bundleSvc != nil {
		if bundle, err := v.bundleSvc.GetBestBundleForPrice(ctx, network, amountNaira); err == nil && bundle != nil {
			variationCode = bundle.ID // ID field holds variation_code from network_data_bundles
			log.Printf("[VTPassAdapter] TopUpData: DB-first code=%q name=%q price=₦%.0f (prize=₦%.0f) network=%s",
				variationCode, bundle.Name, bundle.Price, amountNaira, network)
		} else {
			log.Printf("[VTPassAdapter] TopUpData: DB lookup failed (%v) — falling back to hardcoded map", err)
		}
	}

	// ── Hardcoded fallback ─────────────────────────────────────────────────────
	// Used when: bundleSvc not wired, DB is empty (first deploy), or network_data_bundles
	// hasn't been synced yet. networkDataCode() is a best-effort static map.
	if variationCode == "" {
		variationCode = networkDataCode(network, amountNaira)
	}
	if variationCode == "" {
		return "", fmt.Errorf("vtpass: no data variation code for network=%s amount=₦%.0f (DB empty and no hardcoded fallback)", network, amountNaira)
	}

	reqID := vtpassRequestID(ref)
	payload := map[string]interface{}{
		"request_id":     reqID,
		"serviceID":      serviceID,
		"billersCode":    phone, // VTPass data: billersCode = phone number
		"variation_code": variationCode,
		"amount":         0, // VTPass ignores amount for data; variation_code determines the plan
		"phone":          phone,
	}
	return v.post(ctx, "/pay", payload)
}

func (v *VTPassAdapter) VerifyService(ctx context.Context, serviceID string) (bool, error) {
	return true, nil
}

func (v *VTPassAdapter) post(ctx context.Context, path string, payload map[string]interface{}) (string, error) {
	apiKey, pubKey, secretKey := v.credentials()
	_ = pubKey // public-key is for GET requests; POST uses api-key + secret-key
	if apiKey == "" {
		return "", fmt.Errorf("VTPass: VTPASS_API_KEY not set")
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", apiKey)
	req.Header.Set("secret-key", secretKey) // required for purchase endpoints

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("VTPass request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Code             string `json:"code"`
		ResponseDesc     string `json:"response_description"`
		Content struct {
			Transactions struct {
				TransactionID string `json:"transactionId"`
				Status        string `json:"status"`
			} `json:"transactions"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("VTPass decode error: %w", err)
	}
	// VTPass success codes: "000" = delivered, "099" = processing/initiated
	if result.Code != "000" && result.Code != "099" {
		return "", fmt.Errorf("VTPass error [%s]: %s", result.Code, result.ResponseDesc)
	}
	return result.Content.Transactions.TransactionID, nil
}

func networkToVTPassID(network string) string {
	m := map[string]string{
		"MTN": "mtn", "AIRTEL": "airtel", "GLO": "glo", "9MOBILE": "etisalat",
	}
	if v, ok := m[network]; ok {
		return v
	}
	return "mtn"
}

func networkToVTPassDataID(network string) string {
	m := map[string]string{
		"MTN": "mtn-data", "AIRTEL": "airtel-data", "GLO": "glo-data", "9MOBILE": "etisalat-data",
	}
	if v, ok := m[network]; ok {
		return v
	}
	return "mtn-data"
}

// networkDataCode returns the VTPass variation code for a data prize.
// amountNaira is the prize value in Naira (prize_value_kobo / 100).
// Codes verified against VTPass sandbox API (GET /api/service-variations?serviceID=<network>-data).
// NOTE: VTPass ignores the amount field for data — variation_code determines the plan and price.
func networkDataCode(network string, amountNaira float64) string {
	kobo := int64(amountNaira * 100) // convert back to kobo for exact matching
	switch strings.ToUpper(network) {
	case "MTN":
		switch kobo {
		case 10000: // ₦100 → 100MB (1 day)
			return "mtn-10mb-100"
		case 20000: // ₦200 → 200MB (2 days)
			return "mtn-50mb-200"
		case 50000: // ₦500 → no exact MTN plan; closest is ₦600 2.5GB (2 days)
			return "mtn-2-5gb-600"
		case 100000: // ₦1000 → 1.5GB (30 days)
			return "mtn-100mb-1000"
		case 150000: // ₦1500 → 3GB (30 days)
			return "mtn-3gb-1500"
		case 200000: // ₦2000 → 4.5GB (30 days)
			return "mtn-500mb-2000"
		}
	case "GLO":
		switch kobo {
		case 10000: // ₦100 → 105MB (2 days)
			return "glo100"
		case 20000: // ₦200 → 350MB (4 days)
			return "glo200"
		case 50000: // ₦500 → 1.05GB (14 days)
			return "glo500"
		case 100000: // ₦1000 → 2.5GB (30 days)
			return "glo1000"
		case 200000: // ₦2000 → 5.8GB (30 days)
			return "glo2000"
		}
	case "AIRTEL":
		switch kobo {
		case 10000: // ₦100 → 75MB (1 day)
			return "airt-100"
		case 20000: // ₦200 → 200MB (3 days)
			return "airt-200"
		case 50000: // ₦500 → 750MB (14 days)
			return "airt-500"
		case 100000: // ₦1000 → 1.5GB (30 days)
			return "airt-1000"
		case 200000: // ₦2000 → 4.5GB (30 days)
			return "airt-2000"
		}
	case "9MOBILE":
		switch kobo {
		case 10000: // ₦100 → 100MB (1 day)
			return "eti-100"
		case 50000: // ₦500 → 500MB (30 days)
			return "eti-500"
		case 100000: // ₦1000 → 1.5GB (30 days)
			return "eti-1000"
		case 200000: // ₦2000 → 4.5GB (30 days)
			return "eti-2000"
		}
	}
	return "" // no matching plan; caller should log and fall back to retry
}

// NetworkFromPhone returns the most likely Nigerian carrier for a given MSISDN using
// NCC prefix allocations. Returns "MTN" as a safe fallback for unrecognised prefixes.
// This is used for spin-wheel prize fulfillment where we need to route to the right
// VTPass serviceID — accuracy is best-effort (does not account for number portability).
func NetworkFromPhone(phone string) string {
	p := strings.TrimSpace(phone)
	// Normalise +234 → 0
	if strings.HasPrefix(p, "+234") {
		p = "0" + p[4:]
	} else if strings.HasPrefix(p, "234") && len(p) == 13 {
		p = "0" + p[3:]
	}
	if len(p) < 4 {
		return "MTN"
	}
	prefix4 := p[:4]
	switch prefix4 {
	// MTN prefixes
	case "0703", "0706", "0803", "0806", "0810", "0813", "0814", "0816", "0903", "0906", "0913", "0916":
		return "MTN"
	// Airtel prefixes
	case "0701", "0708", "0802", "0808", "0812", "0901", "0902", "0904", "0907", "0912":
		return "AIRTEL"
	// GLO prefixes
	case "0805", "0807", "0811", "0815", "0905", "0915":
		return "GLO"
	// 9Mobile (etisalat) prefixes
	case "0809", "0817", "0818", "0908", "0909":
		return "9MOBILE"
	}
	return "MTN" // safe fallback — most Nigerian users are on MTN
}
