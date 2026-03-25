package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/external"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RechargeService struct {
	provisioner external.Provisioner
	userRepo    repositories.UserRepository
	txRepo      repositories.TransactionRepository
	db          *gorm.DB
}

func NewRechargeService(p external.Provisioner, ur repositories.UserRepository, tr repositories.TransactionRepository, db *gorm.DB) *RechargeService {
	return &RechargeService{
		provisioner: p,
		userRepo:    ur,
		txRepo:      tr,
		db:          db,
	}
}

func (s *RechargeService) ProcessSuccessfulPayment(ctx context.Context, msisdn string, amountKobo int64, network string, ref string) error {
	mode := os.Getenv("OPERATION_MODE")

	if mode == "independent" {
		// 1. Provision via VTPass
		provRef, err := s.provisioner.PurchaseAirtime(ctx, msisdn, amountKobo, network)
		if err != nil {
			log.Printf("[RechargeService] Provisioning failed for %s: %v", msisdn, err)
			// In production: trigger refund or manual review
			return fmt.Errorf("provisioning failure: %w", err)
		}
		log.Printf("[RechargeService] Independent Provisioned: %s for %s", provRef, msisdn)
	} else {
		// Integrated Mode: Managed by MNO BSS
		log.Printf("[RechargeService] Integrated Mode: Ledger update only for %s", msisdn)
	}

	// 2. Ledger Update (Atomic Transaction)
	// (Note: The worker handles the actual stream event, but for direct payments,
	// we might handle it here synchronously or push to the same Redis stream).
	
	return nil
}
