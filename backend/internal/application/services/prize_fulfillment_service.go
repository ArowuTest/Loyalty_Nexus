package services

import (
	"context"
	"fmt"
	"log"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/external"
)

type PrizeFulfillmentService struct {
	prizeRepo   repositories.PrizeRepository
	userRepo    repositories.UserRepository
	provisioner external.Provisioner
	momoService *MoMoService
}

func NewPrizeFulfillmentService(pr repositories.PrizeRepository, ur repositories.UserRepository, p external.Provisioner, ms *MoMoService) *PrizeFulfillmentService {
	return &PrizeFulfillmentService{
		prizeRepo:   pr,
		userRepo:    ur,
		provisioner: p,
		momoService: ms,
	}
}

func (s *PrizeFulfillmentService) Fulfill(ctx context.Context, claim *entities.PrizeClaim) error {
	user, err := s.userRepo.FindByID(ctx, claim.UserID)
	if err != nil {
		return err
	}

	// REQ-3.8: Idempotency Check
	if claim.Status == entities.StatusCompleted || claim.Status == entities.StatusProcessing {
		return nil
	}

	claim.Status = entities.StatusProcessing
	s.prizeRepo.UpdateClaim(ctx, claim)

	var ref string
	var fulfillErr error

	switch claim.PrizeType {
	case "airtime":
		ref, fulfillErr = s.provisioner.PurchaseAirtime(ctx, user.MSISDN, int64(claim.PrizeValue*100), "MTN")
	case "data":
		// ...
	case "momo_cash":
		// REQ-3.7: Hold if not verified
		if !user.MoMoVerified {
			claim.Status = entities.StatusPendingMoMoLink
			claim.ErrorMessage = "MoMo account not verified. Hold for 48h (REQ-3.7)"
			return s.prizeRepo.UpdateClaim(ctx, claim)
		}
		// REQ-3.4: Disburse via MoMo API using unique claim ID for idempotency
		ref, fulfillErr = s.momoService.DisburseCash(ctx, user.MoMoNumber, int64(claim.PrizeValue*100), claim.ID.String())
	// ...
	}
	case "bonus_points":
		// Already handled by transaction delta in SpinService
		claim.Status = entities.StatusCompleted
		return s.prizeRepo.UpdateClaim(ctx, claim)
	default:
		claim.Status = entities.StatusFailed
		claim.ErrorMessage = "Unknown prize type"
		return s.prizeRepo.UpdateClaim(ctx, claim)
	}

	if fulfillErr != nil {
		claim.Status = entities.StatusFailed
		claim.ErrorMessage = fulfillErr.Error()
	} else {
		claim.Status = entities.StatusCompleted
		claim.FulfillmentRef = ref
	}

	return s.prizeRepo.UpdateClaim(ctx, claim)
}
