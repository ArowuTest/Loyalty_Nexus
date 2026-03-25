package services

import (
	"context"
	"fmt"
	"log"
)

type MoMoService struct {
	// In production: momoClient external.MoMoClient
}

func NewMoMoService() *MoMoService {
	return &MoMoService{}
}

// VerifyAccount links and verifies an MTN MoMo number (REQ-1.3)
func (s *MoMoService) VerifyAccount(ctx context.Context, msisdn string) (bool, string, error) {
	// 1. Call MTN MoMo Account Holder API
	// Simulation:
	log.Printf("[MTN MoMo] Verifying account for %s", msisdn)
	
	// Mock: If number ends in '0', assume no MoMo account
	if msisdn[len(msisdn)-1] == '0' {
		return false, "Dial *671# on your MTN line to open a MoMo account in 2 minutes.", nil
	}

	return true, "Account Verified", nil
}
