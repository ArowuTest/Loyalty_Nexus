package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
type VTPassAdapter struct {
	baseURL   string
	isSandbox bool
	client    *http.Client
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

func (v *VTPassAdapter) TopUpData(ctx context.Context, phone, network string, dataMB float64, ref string) (string, error) {
	serviceID := networkToVTPassDataID(network)
	billersCode := networkDataCode(network, dataMB)
	reqID := vtpassRequestID(ref)
	payload := map[string]interface{}{
		"request_id":     reqID,
		"serviceID":      serviceID,
		"billersCode":    phone, // VTPass data: billersCode = phone number
		"variation_code": billersCode,
		"amount":         0, // Determined by variation
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

func networkDataCode(network string, dataMB float64) string {
	// MTN data variation codes
	switch {
	case dataMB <= 100:
		return "mtn-10mb-200"
	case dataMB <= 500:
		return "mtn-100mb-200"
	case dataMB <= 1024:
		return "mtn-1gb-300"
	default:
		return "mtn-2gb-500"
	}
}
