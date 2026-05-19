package services

import (
	"context"
	"fmt"
	"log"
	"time"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/pkg/safe"
	"github.com/google/uuid"
)

// PrizeFulfillmentService handles airtime, data, and MoMo cash prize delivery.
// All operations are idempotent — using SpinResult.ID as external reference.
type PrizeFulfillmentService struct {
	prizeRepo   repositories.PrizeRepository
	userRepo    repositories.UserRepository
	vtpass      external.VTPassClient
	momo        external.MoMoPayer
	notifySvc   *NotificationService
	cfg         *config.ConfigManager
}

func NewPrizeFulfillmentService(
	pr repositories.PrizeRepository,
	ur repositories.UserRepository,
	vt external.VTPassClient,
	mm external.MoMoPayer,
	ns *NotificationService,
	cfg *config.ConfigManager,
) *PrizeFulfillmentService {
	return &PrizeFulfillmentService{
		prizeRepo: pr,
		userRepo:  ur,
		vtpass:    vt,
		momo:      mm,
		notifySvc: ns,
		cfg:       cfg,
	}
}

// Fulfill dispatches a won prize based on its type.
// This is called in a goroutine — all errors are logged and retried by the lifecycle worker.
func (s *PrizeFulfillmentService) Fulfill(ctx context.Context, result *entities.SpinResult) error {
	ref := "LN_" + result.ID.String() // Stable idempotency key

	switch result.PrizeType {
	case entities.PrizeAirtime:
		return s.fulfillAirtime(ctx, result, ref)
	case entities.PrizeDataBundle:
		return s.fulfillData(ctx, result, ref)
	case entities.PrizeMoMoCash:
		return s.fulfillMoMo(ctx, result, ref)
	case entities.PrizePulsePoints, entities.PrizeTryAgain:
		return nil // Already handled in SpinService
	default:
		return fmt.Errorf("unknown prize type: %s", result.PrizeType)
	}
}

func (s *PrizeFulfillmentService) fulfillAirtime(ctx context.Context, result *entities.SpinResult, ref string) error {
	user, err := s.userRepo.FindByID(ctx, result.UserID)
	if err != nil {
		return s.markFailed(ctx, result.ID, fmt.Sprintf("user lookup: %v", err))
	}

	if err := s.prizeRepo.UpdateSpinFulfillment(ctx, result.ID, entities.FulfillProcessing, ref, ""); err != nil {
		return err
	}

	// base_value (PrizeValue) is stored in KOBO — divide by 100 for Naira
	amountNaira := result.PrizeValue / 100.0
	vtRef, err := s.vtpass.TopUpAirtime(ctx, user.PhoneNumber, "MTN", amountNaira, ref)
	if err != nil {
		log.Printf("[FULFILL] VTPass airtime failed (will retry): %v", err)
		return s.markFailed(ctx, result.ID, err.Error())
	}

	if err := s.prizeRepo.UpdateSpinFulfillment(ctx, result.ID, entities.FulfillCompleted, vtRef, ""); err != nil {
		return err
	}

	s.notifySvc.NotifyPrizeWon(ctx, user.PhoneNumber,
		fmt.Sprintf("You won ₦%.0f airtime! It has been credited to %s.", amountNaira, user.PhoneNumber))
	return nil
}

func (s *PrizeFulfillmentService) fulfillData(ctx context.Context, result *entities.SpinResult, ref string) error {
	user, err := s.userRepo.FindByID(ctx, result.UserID)
	if err != nil {
		return s.markFailed(ctx, result.ID, fmt.Sprintf("user lookup: %v", err))
	}

	_ = s.prizeRepo.UpdateSpinFulfillment(ctx, result.ID, entities.FulfillProcessing, ref, "")

	// base_value (PrizeValue) is in KOBO — pass Naira-equivalent to VTPass
	dataValueNaira := result.PrizeValue / 100.0
	vtRef, err := s.vtpass.TopUpData(ctx, user.PhoneNumber, "MTN", dataValueNaira, ref)
	if err != nil {
		return s.markFailed(ctx, result.ID, err.Error())
	}

	_ = s.prizeRepo.UpdateSpinFulfillment(ctx, result.ID, entities.FulfillCompleted, vtRef, "")
	s.notifySvc.NotifyPrizeWon(ctx, user.PhoneNumber,
		fmt.Sprintf("You won a ₦%.0f data bundle! It has been added to %s.", dataValueNaira, user.PhoneNumber))
	return nil
}

func (s *PrizeFulfillmentService) fulfillMoMo(ctx context.Context, result *entities.SpinResult, ref string) error {
	user, err := s.userRepo.FindByID(ctx, result.UserID)
	if err != nil {
		return s.markFailed(ctx, result.ID, fmt.Sprintf("user lookup: %v", err))
	}

	if !user.MoMoVerified || user.MoMoNumber == "" {
		// Hold the prize — user needs to link MoMo first
		_ = s.prizeRepo.UpdateSpinFulfillment(ctx, result.ID, entities.FulfillPendingMoMo, "", "")
		s.notifySvc.NotifyPrizeWon(ctx, user.PhoneNumber,
			fmt.Sprintf("You won ₦%.0f MoMo Cash! Link your MTN MoMo number in the app to receive it.", result.PrizeValue/100.0))
		return nil
	}

	_ = s.prizeRepo.UpdateSpinFulfillment(ctx, result.ID, entities.FulfillProcessing, ref, "")

	// base_value (PrizeValue) is in KOBO — convert to Naira for MoMo disbursement
	amountNairaMoMo := int64(result.PrizeValue / 100.0)
	momoRef, err := s.momo.Disburse(ctx, user.MoMoNumber, amountNairaMoMo, ref)
	if err != nil {
		return s.markFailed(ctx, result.ID, err.Error())
	}

	_ = s.prizeRepo.UpdateSpinFulfillment(ctx, result.ID, entities.FulfillCompleted, momoRef, "")
	// Mark claimed_at — MoMo cash is auto-claimed on successful Disburse
	_ = s.updateClaimedAt(ctx, result.ID, time.Now())

	s.notifySvc.NotifyPrizeWon(ctx, user.PhoneNumber,
		fmt.Sprintf("₦%d MoMo Cash has been sent to %s! Check your MoMo wallet.", amountNairaMoMo, user.MoMoNumber))
	return nil
}

// ReleaseMoMoHeldPrizes is called when a user links their MoMo number.
// It picks up all held prizes and dispatches them.
func (s *PrizeFulfillmentService) ReleaseMoMoHeldPrizes(ctx context.Context, userID interface{}) {
	pendingResults, err := s.prizeRepo.ListPendingFulfillments(ctx, 50)
	if err != nil {
		return
	}
	for _, result := range pendingResults {
		if result.FulfillmentStatus == entities.FulfillPendingMoMo {
			r := result // capture loop variable
			safe.Go(func() {
				if err := s.Fulfill(context.Background(), &r); err != nil {
					log.Printf("[FULFILL] MoMo release failed for %s: %v", r.ID, err)
				}
			})
		}
	}
}

// markFailed writes FulfillFailed + error message to the DB so the retry worker
// can detect the failure and retry up to the configured max attempts.
func (s *PrizeFulfillmentService) markFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	log.Printf("[FULFILL] Marking failed: %s — %s", id, errMsg)
	if dbErr := s.prizeRepo.UpdateSpinFulfillment(ctx, id, entities.FulfillFailed, "", errMsg); dbErr != nil {
		log.Printf("[FULFILL] markFailed DB write error: %v", dbErr)
	}
	return fmt.Errorf("%s", errMsg)
}

// updateClaimedAt records claimed_at via UpdateSpinClaimStatus (ClaimClaimed sets claimed_at = NOW()).
// Used for MoMo cash after a successful Disburse call.
func (s *PrizeFulfillmentService) updateClaimedAt(ctx context.Context, id uuid.UUID, _ time.Time) error {
	if err := s.prizeRepo.UpdateSpinClaimStatus(ctx, id, entities.ClaimClaimed, nil); err != nil {
		log.Printf("[FULFILL] updateClaimedAt DB write error for %s: %v", id, err)
		return err
	}
	return nil
}
