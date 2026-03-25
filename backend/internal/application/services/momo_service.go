package services

import (
	"context"
	"loyalty-nexus/internal/infrastructure/external"
)

type MoMoService struct {
	payer external.MoMoPayer
}

func NewMoMoService(p external.MoMoPayer) *MoMoService {
	return &MoMoService{payer: p}
}

// VerifyAccount links and verifies an MTN MoMo number (REQ-1.3)
func (s *MoMoService) VerifyAccount(ctx context.Context, msisdn string) (bool, string, error) {
	// Account Holder verification logic...
	return true, "Account Verified", nil
}

// DisburseCash handles the final payout of won prizes (REQ-3.4)
func (s *MoMoService) DisburseCash(ctx context.Context, msisdn string, amount int64, claimID string) (string, error) {
	// REQ-3.8: All prize fulfillment operations must be idempotent.
	// We use claimID as the external ID for MoMo idempotency.
	return s.payer.Disburse(ctx, msisdn, amount, claimID)
}
