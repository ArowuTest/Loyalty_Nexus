package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/external"
)

type ClaimService struct {
	prizeRepo  repositories.PrizeRepository
	userRepo   repositories.UserRepository
	momoAdapter external.MoMoPayer
	fulfillSvc *PrizeFulfillmentService
}

func NewClaimService(
	prizeRepo repositories.PrizeRepository,
	userRepo repositories.UserRepository,
	momoAdapter external.MoMoPayer,
	fulfillSvc *PrizeFulfillmentService,
) *ClaimService {
	return &ClaimService{
		prizeRepo:   prizeRepo,
		userRepo:    userRepo,
		momoAdapter: momoAdapter,
		fulfillSvc:  fulfillSvc,
	}
}

// GetMyWins returns all non-TryAgain prizes won by the user.
func (s *ClaimService) GetMyWins(ctx context.Context, userID uuid.UUID) ([]entities.SpinResult, error) {
	return s.prizeRepo.ListUserWins(ctx, userID)
}

// CheckMoMoAccount checks if the user's phone number has an active MoMo account.
func (s *ClaimService) CheckMoMoAccount(ctx context.Context, phone string) (bool, string, error) {
	name, valid, err := s.momoAdapter.VerifyAccount(ctx, phone)
	if err != nil || !valid {
		return false, "", nil // Treat errors as "no account" for the dashboard check
	}
	return true, name, nil
}

type ClaimRequest struct {
	MoMoNumber        string `json:"momo_number"`
	BankAccountNumber string `json:"bank_account_number"`
	BankAccountName   string `json:"bank_account_name"`
	BankName          string `json:"bank_name"`
}

// ClaimPrize processes a user's claim for a specific prize.
func (s *ClaimService) ClaimPrize(ctx context.Context, userID, claimID uuid.UUID, req ClaimRequest) (*entities.SpinResult, error) {
	result, err := s.prizeRepo.FindSpinResult(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("claim not found: %w", err)
	}

	if result.UserID != userID {
		return nil, fmt.Errorf("unauthorized to claim this prize")
	}

	if result.ClaimStatus != entities.ClaimPending {
		return nil, fmt.Errorf("prize is already in status: %s", result.ClaimStatus)
	}

	if time.Now().After(result.ExpiresAt) {
		_ = s.prizeRepo.UpdateSpinClaimStatus(ctx, claimID, entities.ClaimExpired, nil)
		return nil, fmt.Errorf("claim has expired")
	}

	bankDetails := map[string]string{}

	switch result.PrizeType {
	case entities.PrizeMoMoCash:
		if req.MoMoNumber == "" && req.BankAccountNumber == "" {
			return nil, fmt.Errorf("momo_number or bank details required for cash prize")
		}
		if req.MoMoNumber != "" {
			bankDetails["momo_claim_number"] = req.MoMoNumber
		} else {
			bankDetails["bank_account_number"] = req.BankAccountNumber
			bankDetails["bank_account_name"] = req.BankAccountName
			bankDetails["bank_name"] = req.BankName
		}
		
		// Move to pending admin review
		err = s.prizeRepo.UpdateSpinClaimStatus(ctx, claimID, entities.ClaimPendingAdmin, bankDetails)
		if err != nil {
			return nil, err
		}
		result.ClaimStatus = entities.ClaimPendingAdmin

	case entities.PrizeAirtime, entities.PrizeDataBundle:
		// Auto-fulfill digital prizes
		err = s.prizeRepo.UpdateSpinClaimStatus(ctx, claimID, entities.ClaimClaimed, nil)
		if err != nil {
			return nil, err
		}
		result.ClaimStatus = entities.ClaimClaimed
		
		// Trigger fulfillment asynchronously
		// The fulfillment service will pick it up if it's pending fulfillment
		// We just need to ensure it's marked as ready for fulfillment
		if result.FulfillmentStatus == entities.FulfillPendingClaim {
			_ = s.prizeRepo.UpdateSpinFulfillment(ctx, claimID, entities.FulfillPending, "", "")
		}

	case entities.PrizePulsePoints:
		// Points are auto-credited at spin time, just mark as claimed
		err = s.prizeRepo.UpdateSpinClaimStatus(ctx, claimID, entities.ClaimClaimed, nil)
		if err != nil {
			return nil, err
		}
		result.ClaimStatus = entities.ClaimClaimed

	default:
		return nil, fmt.Errorf("cannot claim prize type: %s", result.PrizeType)
	}

	return result, nil
}
