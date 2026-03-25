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

// MTNMomoAdapter implements MoMoPayer against the MTN MoMo Disbursement API v2.
// Ref: https://momodeveloper.mtn.com/docs/disbursement
type MTNMomoAdapter struct {
	subscriptionKey string
	apiUser         string
	apiKey          string
	targetEnv       string
	baseURL         string
	client          *http.Client
}

func NewMTNMomoAdapter() *MTNMomoAdapter {
	env := os.Getenv("MOMO_TARGET_ENV") // sandbox | production
	if env == "" {
		env = "sandbox"
	}
	baseURL := "https://sandbox.momodeveloper.mtn.com"
	if env == "production" {
		baseURL = "https://momodeveloper.mtn.com"
	}
	return &MTNMomoAdapter{
		subscriptionKey: os.Getenv("MOMO_SUBSCRIPTION_KEY"),
		apiUser:         os.Getenv("MOMO_API_USER"),
		apiKey:          os.Getenv("MOMO_API_KEY"),
		targetEnv:       env,
		baseURL:         baseURL,
		client:          &http.Client{Timeout: 30 * time.Second},
	}
}

// Disburse sends money to a MoMo wallet.
// Uses the X-Reference-Id header as the idempotency key.
func (m *MTNMomoAdapter) Disburse(ctx context.Context, phone string, amountNaira int64, ref string) (string, error) {
	token, err := m.getAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("MoMo auth failed: %w", err)
	}

	// Normalize phone: remove leading 0 or +, prepend 234
	phone = normalizeNigerianPhone(phone)

	payload := map[string]interface{}{
		"amount":   fmt.Sprintf("%d", amountNaira),
		"currency": "NGN",
		"externalId": ref,
		"payee": map[string]string{
			"partyIdType": "MSISDN",
			"partyId":     phone,
		},
		"payerMessage": "Loyalty Nexus Prize Payout",
		"payeeNote":    "Congratulations! Your prize has been disbursed.",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		m.baseURL+"/disbursement/v1_0/transfer", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Reference-Id", ref)
	req.Header.Set("X-Target-Environment", m.targetEnv)
	req.Header.Set("Ocp-Apim-Subscription-Key", m.subscriptionKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("MoMo disburse request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 202 {
		return ref, nil // Accepted — poll for completion
	}
	return "", fmt.Errorf("MoMo Disburse HTTP %d", resp.StatusCode)
}

// VerifyAccount checks if a MoMo number is valid before linking.
func (m *MTNMomoAdapter) VerifyAccount(ctx context.Context, phone string) (string, bool, error) {
	token, err := m.getAccessToken(ctx)
	if err != nil {
		return "", false, fmt.Errorf("MoMo auth failed: %w", err)
	}
	phone = normalizeNigerianPhone(phone)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/disbursement/v1_0/accountholder/msisdn/%s/basicuserinfo", m.baseURL, phone), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Target-Environment", m.targetEnv)
	req.Header.Set("Ocp-Apim-Subscription-Key", m.subscriptionKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var info struct {
			Name string `json:"name"`
		}
		json.NewDecoder(resp.Body).Decode(&info)
		return info.Name, true, nil
	}
	return "", false, fmt.Errorf("MoMo account not found (HTTP %d)", resp.StatusCode)
}

// GetTransactionStatus polls disbursement status.
func (m *MTNMomoAdapter) GetTransactionStatus(ctx context.Context, momoRef string) (string, error) {
	token, err := m.getAccessToken(ctx)
	if err != nil {
		return "", err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/disbursement/v1_0/transfer/%s", m.baseURL, momoRef), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Target-Environment", m.targetEnv)
	req.Header.Set("Ocp-Apim-Subscription-Key", m.subscriptionKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"` // SUCCESSFUL | FAILED | PENDING
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Status, nil
}

func (m *MTNMomoAdapter) getAccessToken(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		m.baseURL+"/disbursement/token/", bytes.NewBuffer([]byte{}))
	req.SetBasicAuth(m.apiUser, m.apiKey)
	req.Header.Set("Ocp-Apim-Subscription-Key", m.subscriptionKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func normalizeNigerianPhone(phone string) string {
	if len(phone) == 11 && phone[0] == '0' {
		return "234" + phone[1:]
	}
	if len(phone) == 13 && phone[:3] == "234" {
		return phone
	}
	if len(phone) == 14 && phone[:4] == "+234" {
		return phone[1:]
	}
	return phone
}
