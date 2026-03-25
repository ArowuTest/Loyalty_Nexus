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
	mode := os.Getenv("OPERATION_MODE")

	if mode == "independent" {
		// 1. Provision via VTPass
		provRef, err := s.provisioner.PurchaseAirtime(ctx, msisdn, amountKobo, network)
		if err != nil {
			return err
		}
		log.Printf("[RechargeService] Independent Provisioned: %s", provRef)
	} else {
		// Integrated Mode: Provisioning handled by MNO BSS
		log.Printf("[RechargeService] Integrated Mode: Awaiting MNO BSS Confirmation for %s", msisdn)
	}

	return nil
}
