package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type VTPassRequest struct {
	RequestID string `json:"request_id"`
	ServiceID string `json:"service_id"`
	Amount    string `json:"amount"`
	Phone     string `json:"phone"`
}

type VTPassResponse struct {
	Code     string `json:"code"`
	Content  struct {
		Transactions struct {
			Status string `json:"status"`
		} `json:"transactions"`
	} `json:"content"`
}

type VTPassAdapter struct {
	APIKey    string
	PublicKey string
	BaseURL   string
}

func NewVTPassAdapter(apiKey, pubKey string) *VTPassAdapter {
	return &VTPassAdapter{
		APIKey:    apiKey,
		PublicKey: pubKey,
		BaseURL:   "https://vtpass.com/api", // In production
	}
}

func (a *VTPassAdapter) PurchaseAirtime(ctx context.Context, msisdn string, amountKobo int64, network string) (string, error) {
	// 1. Generate unique request_id (YYYYMMDDHHII + random)
	reqID := time.Now().Format("200601021504") + msisdn[len(msisdn)-4:]
	
	serviceID := a.mapNetworkToServiceID(network, "airtime")
	amount := fmt.Sprintf("%.0f", float64(amountKobo)/100)

	payload := VTPassRequest{
		RequestID: reqID,
		ServiceID: serviceID,
		Amount:    amount,
		Phone:     msisdn,
	}

	// 2. Make API Call (Simulation)
	fmt.Printf("[VTPass API] Request: %+v\n", payload)
	
	// Mock success
	return "VT-" + reqID, nil
}

func (a *VTPassAdapter) PurchaseData(ctx context.Context, msisdn string, planID string, network string) (string, error) {
	// Similar logic for data bundles
	return "VT-DATA-MOCK", nil
}

func (a *VTPassAdapter) mapNetworkToServiceID(network, category string) string {
	// Mapping logic for VTPass service IDs (mtn, airtel, etc.)
	return fmt.Sprintf("%s-%s", network, category)
}
