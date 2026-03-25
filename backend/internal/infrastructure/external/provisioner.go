package external

import (
	"context"
	"fmt"
	"log"
)

type Provisioner interface {
	PurchaseAirtime(ctx context.Context, msisdn string, amountKobo int64, network string) (string, error)
	PurchaseData(ctx context.Context, msisdn string, planID string, network string) (string, error)
}

type VTPassAdapter struct {
	APIKey    string
	PublicKey string
}

func (a *VTPassAdapter) PurchaseAirtime(ctx context.Context, msisdn string, amountKobo int64, network string) (string, error) {
	log.Printf("[VTPass] Airtime: %d Kobo to %s (%s)", amountKobo, msisdn, network)
	// In production: POST to https://api-service.vtpass.com/api/pay
	return "vt-mock-ref-123", nil
}

func (a *VTPassAdapter) PurchaseData(ctx context.Context, msisdn string, planID string, network string) (string, error) {
	log.Printf("[VTPass] Data: %s to %s (%s)", planID, msisdn, network)
	return "vt-mock-ref-456", nil
}
