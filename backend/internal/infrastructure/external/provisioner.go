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
type VTPassAdapter struct {
	apiKey    string
	pubKey    string
	baseURL   string
	client    *http.Client
}

func NewVTPassAdapter() *VTPassAdapter {
	return &VTPassAdapter{
		apiKey:  os.Getenv("VTPASS_API_KEY"),
		pubKey:  os.Getenv("VTPASS_PUBLIC_KEY"),
		baseURL: "https://vtpass.com/api",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (v *VTPassAdapter) TopUpAirtime(ctx context.Context, phone, network string, amountNaira float64, ref string) (string, error) {
	payload := map[string]interface{}{
		"request_id":   ref,
		"serviceID":    networkToVTPassID(network),
		"amount":       amountNaira,
		"phone":        phone,
	}
	return v.post(ctx, "/pay", payload)
}

func (v *VTPassAdapter) TopUpData(ctx context.Context, phone, network string, dataMB float64, ref string) (string, error) {
	serviceID := networkToVTPassDataID(network)
	billersCode := networkDataCode(network, dataMB)
	payload := map[string]interface{}{
		"request_id":   ref,
		"serviceID":    serviceID,
		"billersCode":  billersCode,
		"variation_code": billersCode,
		"amount":       0, // Determined by variation
		"phone":        phone,
	}
	return v.post(ctx, "/pay", payload)
}

func (v *VTPassAdapter) VerifyService(ctx context.Context, serviceID string) (bool, error) {
	return true, nil
}

func (v *VTPassAdapter) post(ctx context.Context, path string, payload map[string]interface{}) (string, error) {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", v.apiKey)
	req.Header.Set("public-key", v.pubKey)

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("VTPass request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code    string `json:"code"`
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
	if result.Code != "000" {
		return "", fmt.Errorf("VTPass error code: %s", result.Code)
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
