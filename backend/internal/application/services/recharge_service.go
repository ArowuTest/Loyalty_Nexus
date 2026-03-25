package services

import (
	"context"
	"fmt"
	"log"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/external"
	"github.com/google/uuid"
)

type RechargeService struct {
	provisioner external.Provisioner
	userRepo    UserRepository // Assuming available
	txRepo      TransactionRepository // Assuming available
}

func NewRechargeService(p external.Provisioner) *RechargeService {
	return &RechargeService{provisioner: p}
}

func (s *RechargeService) ProcessSuccessfulPayment(ctx context.Context, msisdn string, amountKobo int64, network string, ref string) error {
	// 1. Provision via VTPass
	provRef, err := s.provisioner.PurchaseAirtime(ctx, msisdn, amountKobo, network)
	if err != nil {
		// Log error and handle compensating logic if needed
		return err
	}

	log.Printf("[RechargeService] Provisioned successfully: %s", provRef)

	// 2. Award Points & Spin Credits (handled by the common worker or ledger triggers)
	// In production, we'd emit a domain event here or call the ledger service directly.
	
	return nil
}
