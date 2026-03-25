package external

import (
	"context"
	"fmt"
	"log"
)

type MoMoPayer interface {
	Disburse(ctx context.Context, msisdn string, amountKobo int64, externalID string) (string, error)
}

type MTNMomoAdapter struct {
	SubscriptionKey string
	APIUser         string
	APIKey          string
	Environment     string // 'sandbox' or 'production'
}

func (a *MTNMomoAdapter) Disburse(ctx context.Context, msisdn string, amountKobo int64, externalID string) (string, error) {
	log.Printf("[MTN MoMo API] Disbursing %d Kobo to %s (Ref: %s)", amountKobo, msisdn, externalID)
	// In production: 
	// 1. POST /token (Get OAuth2 token)
	// 2. POST /disbursement/v1_0/transfer (Use X-Reference-Id for idempotency)
	// 3. GET /disbursement/v1_0/transfer/{id} (Verify status)
	
	// Mock success
	return "momo-transfer-ref-999", nil
}
