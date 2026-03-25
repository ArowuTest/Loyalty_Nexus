package services

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SpinService struct {
	userRepo    repositories.UserRepository
	txRepo      repositories.TransactionRepository
	prizeRepo   repositories.PrizeRepository
	fulfillSvc  *PrizeFulfillmentService
	notifySvc   *NotificationService
	cfg         *config.ConfigManager
	db          *gorm.DB
}

func NewSpinService(
	ur repositories.UserRepository,
	tr repositories.TransactionRepository,
	pr repositories.PrizeRepository,
	fs *PrizeFulfillmentService,
	ns *NotificationService,
	cfg *config.ConfigManager,
	db *gorm.DB,
) *SpinService {
	return &SpinService{
		userRepo:   ur,
		txRepo:     tr,
		prizeRepo:  pr,
		fulfillSvc: fs,
		notifySvc:  ns,
		cfg:        cfg,
		db:         db,
	}
}

type SpinOutcome struct {
	SpinResult  *entities.SpinResult `json:"spin_result"`
	PrizeLabel  string               `json:"prize_label"`
	SlotIndex   int                  `json:"slot_index"`
	Message     string               `json:"message"`
	NeedsMoMo   bool                 `json:"needs_momo_setup"` // Prompt user to link MoMo
}

// PlaySpin executes a single spin:
// 1. Validate daily limit (REQ-3.6)
// 2. Check daily liability cap (REQ-3.5) — force low value if hit
// 3. Verify user has ≥1 spin credit (REQ-3.1)
// 4. Select prize via CSPRNG (REQ-3.2)
// 5. Atomically deduct credit + write spin result + ledger entry
// 6. Dispatch fulfillment in background goroutine
func (s *SpinService) PlaySpin(ctx context.Context, userID uuid.UUID) (*SpinOutcome, error) {
	// --- Step 1: Daily spin limit ---
	dailySpins, err := s.prizeRepo.CountUserSpinsToday(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("spin count check failed: %w", err)
	}
	maxSpins := s.cfg.GetInt("spin_max_per_user_per_day", 3)
	if dailySpins >= maxSpins {
		return nil, fmt.Errorf("daily spin limit reached (max %d). Come back tomorrow!", maxSpins)
	}

	// --- Step 2: Daily liability cap ---
	capNaira := s.cfg.GetInt64("daily_prize_liability_cap_naira", 500000)
	capKobo := capNaira * 100
	currentLiability, _ := s.txRepo.DailyLiabilityTotal(ctx)
	forceLowValue := currentLiability >= capKobo

	// --- Step 3: Wallet check (row-level lock) ---
	wallet, err := s.userRepo.GetWalletForUpdate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("wallet not found")
	}
	if wallet.SpinCredits < 1 {
		return nil, fmt.Errorf("no spin credits. Recharge ₦%d to earn one!",
			s.cfg.GetInt64("spin_trigger_naira", 1000))
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// --- Step 4: Select prize via CSPRNG (REQ-3.2) ---
	prize, slotIdx, err := s.selectPrize(ctx, forceLowValue)
	if err != nil {
		return nil, fmt.Errorf("prize selection failed: %w", err)
	}

	// --- Step 5: Atomic DB transaction ---
	var spinResult *entities.SpinResult
	err = s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Deduct 1 spin credit
		wallet.SpinCredits--
		if err := s.userRepo.UpdateWallet(ctx, wallet); err != nil {
			return fmt.Errorf("credit deduction failed: %w", err)
		}

		// Determine initial fulfillment status
		fulfillStatus := entities.FulfillPending
		if prize.PrizeType == entities.PrizeTryAgain {
			fulfillStatus = entities.FulfillNA
		} else if prize.PrizeType == entities.PrizeMoMoCash && !user.MoMoVerified {
			fulfillStatus = entities.FulfillPendingMoMo
		}

		spinResult = &entities.SpinResult{
			ID:                uuid.New(),
			UserID:            userID,
			PrizeType:         prize.PrizeType,
			PrizeValue:        prize.BaseValue,
			SlotIndex:         slotIdx,
			FulfillmentStatus: fulfillStatus,
			CreatedAt:         time.Now(),
		}
		if err := s.prizeRepo.CreateSpinResult(ctx, spinResult); err != nil {
			return fmt.Errorf("spin result write failed: %w", err)
		}

		// Ledger entry: spin play
		spinMeta, _ := json.Marshal(map[string]interface{}{
			"prize_type":  prize.PrizeType,
			"prize_value": prize.BaseValue,
			"spin_id":     spinResult.ID.String(),
		})
		spinTx := &entities.Transaction{
			ID:          uuid.New(),
			UserID:      userID,
			PhoneNumber: user.PhoneNumber,
			Type:        entities.TxTypeSpinPlay,
			SpinDelta:   -1,
			Reference:   "spin_" + spinResult.ID.String(),
			Metadata:    spinMeta,
			CreatedAt:   time.Now(),
		}
		if err := s.txRepo.SaveTx(ctx, dbTx, spinTx); err != nil {
			return err
		}

		// For Pulse Points prizes, award immediately via ledger
		if prize.PrizeType == entities.PrizePulsePoints {
			pts := int64(prize.BaseValue)
			wallet.PulsePoints += pts
			wallet.LifetimePoints += pts
			if err := s.userRepo.UpdateWallet(ctx, wallet); err != nil {
				return err
			}
			ptsTx := &entities.Transaction{
				ID:           uuid.New(),
				UserID:       userID,
				PhoneNumber:  user.PhoneNumber,
				Type:         entities.TxTypePrizeAward,
				PointsDelta:  pts,
				BalanceAfter: wallet.PulsePoints,
				Reference:    "prize_pts_" + spinResult.ID.String(),
				CreatedAt:    time.Now(),
			}
			if err := s.txRepo.SaveTx(ctx, dbTx, ptsTx); err != nil {
				return err
			}
			if err := s.prizeRepo.UpdateSpinFulfillment(ctx, spinResult.ID, entities.FulfillCompleted, "", ""); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// --- Step 6: Background fulfillment for physical prizes ---
	if spinResult.FulfillmentStatus == entities.FulfillPending {
		go func() {
			if err := s.fulfillSvc.Fulfill(context.Background(), spinResult); err != nil {
				log.Printf("[SPIN] Fulfillment failed for %s: %v", spinResult.ID, err)
			}
		}()
	}

	// Build outcome response
	outcome := &SpinOutcome{
		SpinResult: spinResult,
		PrizeLabel: s.buildPrizeLabel(prize),
		SlotIndex:  slotIdx,
		NeedsMoMo:  spinResult.FulfillmentStatus == entities.FulfillPendingMoMo,
	}
	outcome.Message = s.buildWinMessage(outcome, user.PhoneNumber)

	return outcome, nil
}

// GetWheelConfig returns the assembled spin wheel for the frontend.
func (s *SpinService) GetWheelConfig(ctx context.Context) (*entities.SpinWheelPayload, error) {
	prizes, err := s.prizeRepo.ListActivePrizes(ctx)
	if err != nil {
		return nil, err
	}

	colors := []string{"#FF6B35","#FFD700","#00B4D8","#06D6A0","#EF476F","#118AB2","#073B4C","#FFB703"}
	slots := make([]entities.SpinSlot, len(prizes))
	for i, p := range prizes {
		slots[i] = entities.SpinSlot{
			Index:     i,
			PrizeType: p.PrizeType,
			Label:     s.buildPrizeLabel(&p),
			Color:     colors[i%len(colors)],
		}
	}
	return &entities.SpinWheelPayload{
		Slots:           slots,
		RequiredCredits: 1,
	}, nil
}

// selectPrize uses CSPRNG weighted random selection.
func (s *SpinService) selectPrize(ctx context.Context, forceLowValue bool) (*entities.PrizePoolEntry, int, error) {
	var prizes []entities.PrizePoolEntry
	var err error
	if forceLowValue {
		prizes, err = s.prizeRepo.ListActivePrizesMaxValue(ctx, 5000) // Max ₦50 when cap hit
	} else {
		prizes, err = s.prizeRepo.ListActivePrizes(ctx)
	}
	if err != nil || len(prizes) == 0 {
		return nil, 0, fmt.Errorf("no active prizes available")
	}

	// Check daily inventory caps
	eligible := make([]entities.PrizePoolEntry, 0, len(prizes))
	for _, p := range prizes {
		if p.DailyInventoryCap != nil {
			used, _ := s.prizeRepo.GetDailyInventoryUsed(ctx, p.ID)
			if used >= *p.DailyInventoryCap {
				continue // Inventory exhausted for this prize today
			}
		}
		eligible = append(eligible, p)
	}
	if len(eligible) == 0 {
		eligible = prizes // Fallback: no inventory caps remain, use all
	}

	// Weighted CSPRNG selection
	totalWeight := int64(0)
	for _, p := range eligible {
		totalWeight += int64(p.ProbWeight)
	}
	if totalWeight == 0 {
		return nil, 0, fmt.Errorf("all prizes have zero weight")
	}

	roll, _ := rand.Int(rand.Reader, big.NewInt(totalWeight))
	cursor := int64(0)
	for i, p := range eligible {
		cursor += int64(p.ProbWeight)
		if roll.Int64() < cursor {
			return &eligible[i], i, nil
		}
	}
	return &eligible[0], 0, nil
}

func (s *SpinService) buildPrizeLabel(p *entities.PrizePoolEntry) string {
	switch p.PrizeType {
	case entities.PrizeTryAgain:
		return "Try Again"
	case entities.PrizePulsePoints:
		return fmt.Sprintf("+%.0f Points", p.BaseValue)
	case entities.PrizeAirtime:
		return fmt.Sprintf("₦%.0f Airtime", p.BaseValue)
	case entities.PrizeDataBundle:
		return fmt.Sprintf("%.0fMB Data", p.BaseValue)
	case entities.PrizeMoMoCash:
		return fmt.Sprintf("₦%.0f MoMo Cash", p.BaseValue)
	default:
		return p.Name
	}
}

func (s *SpinService) buildWinMessage(o *SpinOutcome, phone string) string {
	if o.SpinResult.PrizeType == entities.PrizeTryAgain {
		return "Better luck next time! Keep recharging to earn more spins."
	}
	if o.NeedsMoMo {
		return fmt.Sprintf("You won %s! Link your MTN MoMo number to claim your cash prize.", o.PrizeLabel)
	}
	return fmt.Sprintf("Congratulations! You won %s!", o.PrizeLabel)
}
