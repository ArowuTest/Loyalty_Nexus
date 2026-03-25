package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrDuplicateRecharge = errors.New("recharge already processed")

// PaystackEvent is the incoming webhook payload from Paystack.
type PaystackEvent struct {
	Event string          `json:"event"`
	Data  PaystackCharge  `json:"data"`
}

type PaystackCharge struct {
	Reference   string  `json:"reference"`
	Amount      int64   `json:"amount"` // in kobo
	Status      string  `json:"status"`
	Customer    struct {
		PhoneNumber string `json:"phone"`
	} `json:"customer"`
	Metadata json.RawMessage `json:"metadata"`
}

type RechargeService struct {
	userRepo repositories.UserRepository
	txRepo   repositories.TransactionRepository
	notifySvc *NotificationService
	cfg       *config.ConfigManager
	db        *gorm.DB
}

func NewRechargeService(
	ur repositories.UserRepository,
	tr repositories.TransactionRepository,
	ns *NotificationService,
	cfg *config.ConfigManager,
	db *gorm.DB,
) *RechargeService {
	return &RechargeService{
		userRepo:  ur,
		txRepo:    tr,
		notifySvc: ns,
		cfg:       cfg,
		db:        db,
	}
}

// VerifyPaystackSignature validates HMAC-SHA512 header (REQ-9.1.1).
func (s *RechargeService) VerifyPaystackSignature(payload []byte, signature string) bool {
	secret := os.Getenv("PAYSTACK_SECRET_KEY")
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ProcessRechargeWebhook handles an incoming Paystack charge.success event.
// Idempotent: duplicate references are silently ignored.
func (s *RechargeService) ProcessRechargeWebhook(ctx context.Context, event *PaystackEvent) error {
	if event.Data.Status != "success" {
		return nil // Only process successful charges
	}

	phone := event.Data.Customer.PhoneNumber
	amountKobo := event.Data.Amount
	reference := event.Data.Reference

	// Idempotency check
	existing, _ := s.txRepo.FindByReference(ctx, reference)
	if existing != nil {
		log.Printf("[RECHARGE] Duplicate webhook ignored: %s", reference)
		return ErrDuplicateRecharge
	}

	// Minimum recharge check (read from config — never hardcoded)
	minKobo := s.cfg.GetInt64("min_qualifying_recharge_naira", 50) * 100
	if amountKobo < minKobo {
		log.Printf("[RECHARGE] Below minimum (₦%d): %s", amountKobo/100, phone)
		return nil
	}

	user, err := s.userRepo.FindByPhoneNumber(ctx, phone)
	if err != nil {
		return fmt.Errorf("user not found for %s: %w", phone, err)
	}

	return s.processAwardTransaction(ctx, user, amountKobo, reference, false)
}

// ProcessMNOWebhook handles a raw BSS event (Mode 2 — MTN Integrated).
func (s *RechargeService) ProcessMNOWebhook(ctx context.Context, phone string, amountKobo int64, reference string) error {
	existing, _ := s.txRepo.FindByReference(ctx, reference)
	if existing != nil {
		return ErrDuplicateRecharge
	}
	user, err := s.userRepo.FindByPhoneNumber(ctx, phone)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	return s.processAwardTransaction(ctx, user, amountKobo, reference, true)
}

// processAwardTransaction is the core atomic function — executed inside a DB transaction.
func (s *RechargeService) processAwardTransaction(ctx context.Context, user *entities.User, amountKobo int64, reference string, isIntegrated bool) error {
	return s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// --- Row-level lock on wallet ---
		wallet, err := s.userRepo.GetWalletForUpdate(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("wallet lock failed: %w", err)
		}

		amountNaira := amountKobo / 100

		// --- Calculate Pulse Points ---
		baseRate := s.cfg.GetFloat("points_per_250_naira", 1.0) / 250.0
		tieredRate := s.getTieredRate(ctx, wallet.LifetimePoints)
		globalMultiplier := s.cfg.GetFloat("global_points_multiplier", 1.0)
		scheduledMultiplier := s.getActiveScheduledMultiplier(ctx, user.ID)
		segmentMultiplier := s.getSegmentMultiplier(ctx, user)

		effectiveRate := tieredRate * globalMultiplier * scheduledMultiplier * segmentMultiplier
		ptsEarned := int64(math.Floor(float64(amountNaira) * effectiveRate))

		// --- Calculate Spin Credits ---
		spinTriggerKobo := s.cfg.GetInt64("spin_trigger_naira", 1000) * 100
		newCounter := wallet.RechargeCounter + amountKobo
		spinCreditsEarned := int(newCounter / spinTriggerKobo)
		newCounter = newCounter % spinTriggerKobo

		// --- Update wallet ---
		wallet.PulsePoints += ptsEarned
		wallet.LifetimePoints += ptsEarned
		wallet.SpinCredits += spinCreditsEarned
		wallet.RechargeCounter = newCounter
		if err := s.userRepo.UpdateWallet(ctx, wallet); err != nil {
			return fmt.Errorf("wallet update failed: %w", err)
		}

		// --- Update streak ---
		streakHours := s.cfg.GetInt("streak_expiry_hours", 36)
		newStreak := s.calculateNewStreak(user, streakHours)
		streakExpiresAt := time.Now().Add(time.Duration(streakHours) * time.Hour)
		if err := s.userRepo.UpdateStreak(ctx, user.ID, newStreak, streakExpiresAt); err != nil {
			return err
		}

		// --- Update user recharge stats ---
		now := time.Now()
		user.TotalRechargeAmount += amountKobo
		user.LastRechargeAt = &now
		if err := s.userRepo.Update(ctx, user); err != nil {
			return err
		}

		// --- Update tier ---
		newTier := entities.TierFromLifetimePoints(wallet.LifetimePoints)
		if newTier != user.Tier {
			_ = s.userRepo.UpdateTier(ctx, user.ID, newTier)
		}

		// --- Write immutable ledger entries ---
		// 1. Recharge record
		rechargeTx := &entities.Transaction{
			ID:           uuid.New(),
			UserID:       user.ID,
			PhoneNumber:  user.PhoneNumber,
			Type:         entities.TxTypeRecharge,
			Amount:       amountKobo,
			BalanceAfter: wallet.PulsePoints,
			Reference:    reference,
			CreatedAt:    time.Now(),
		}
		if err := s.txRepo.SaveTx(ctx, dbTx, rechargeTx); err != nil {
			return err
		}

		// 2. Points award record
		if ptsEarned > 0 {
			awardMeta, _ := json.Marshal(map[string]interface{}{
				"amount_kobo": amountKobo,
				"rate":        effectiveRate,
				"multipliers": map[string]float64{
					"global":    globalMultiplier,
					"scheduled": scheduledMultiplier,
					"segment":   segmentMultiplier,
				},
			})
			ptsTx := &entities.Transaction{
				ID:           uuid.New(),
				UserID:       user.ID,
				PhoneNumber:  user.PhoneNumber,
				Type:         entities.TxTypePointsAward,
				PointsDelta:  ptsEarned,
				BalanceAfter: wallet.PulsePoints,
				Reference:    reference + "_pts",
				Metadata:     awardMeta,
				CreatedAt:    time.Now(),
			}
			if err := s.txRepo.SaveTx(ctx, dbTx, ptsTx); err != nil {
				return err
			}
		}

		// 3. Spin credit award record
		if spinCreditsEarned > 0 {
			spinTx := &entities.Transaction{
				ID:          uuid.New(),
				UserID:      user.ID,
				PhoneNumber: user.PhoneNumber,
				Type:        entities.TxTypeSpinCreditAward,
				SpinDelta:   spinCreditsEarned,
				Reference:   reference + "_spin",
				CreatedAt:   time.Now(),
			}
			if err := s.txRepo.SaveTx(ctx, dbTx, spinTx); err != nil {
				return err
			}
		}

		// First recharge bonus (async — outside DB tx to avoid blocking)
		go s.checkFirstRechargeBonus(context.Background(), user, ptsEarned)

		log.Printf("[RECHARGE] Processed %s: ₦%d -> +%d pts, +%d spins (streak: %d)",
			user.PhoneNumber, amountNaira, ptsEarned, spinCreditsEarned, newStreak)

		return nil
	})
}

func (s *RechargeService) getTieredRate(ctx context.Context, lifetimePoints int64) float64 {
	basePerNaira := s.cfg.GetFloat("points_per_250_naira", 1.0) / 250.0
	// Tier rates from recharge_tiers table — simplified inline for now
	switch {
	case lifetimePoints >= 5000: // Platinum
		return basePerNaira * 1.5
	case lifetimePoints >= 1500: // Gold
		return basePerNaira * 1.25
	case lifetimePoints >= 500: // Silver
		return basePerNaira * 1.1
	default:
		return basePerNaira
	}
}

func (s *RechargeService) calculateNewStreak(user *entities.User, expiryHours int) int {
	if user.LastRechargeAt == nil {
		return 1
	}
	deadline := user.LastRechargeAt.Add(time.Duration(expiryHours) * time.Hour)
	if time.Now().Before(deadline) {
		return user.StreakCount + 1
	}
	// Check grace days (REQ-5.2.13)
	currentMonth := time.Now().Month()
	graceLimit := s.cfg.GetInt("streak_grace_days_per_month", 1)
	graceUsedThisMonth := 0
	if user.StreakGraceMonth != nil && *user.StreakGraceMonth == int(currentMonth) {
		graceUsedThisMonth = user.StreakGraceUsed
	}
	if graceUsedThisMonth < graceLimit {
		return user.StreakCount // Streak preserved via grace day
	}
	return 1 // Reset
}

func (s *RechargeService) getActiveScheduledMultiplier(ctx context.Context, userID uuid.UUID) float64 {
	var multiplier float64
	s.db.WithContext(ctx).
		Table("scheduled_multipliers").
		Where("is_active = true AND start_at <= NOW() AND end_at >= NOW()").
		Select("COALESCE(MAX(multiplier), 1.0)").
		Scan(&multiplier)
	if multiplier < 1.0 {
		return 1.0
	}
	return multiplier
}

func (s *RechargeService) getSegmentMultiplier(ctx context.Context, user *entities.User) float64 {
	// Check for state-based or tier-based overrides
	var multiplier float64
	s.db.WithContext(ctx).
		Table("segment_multipliers").
		Where("is_active = true AND (start_at IS NULL OR start_at <= NOW()) AND (end_at IS NULL OR end_at >= NOW())").
		Where("(segment_type = 'state' AND segment_value = ?) OR (segment_type = 'tier' AND segment_value = ?)",
			user.State, user.Tier).
		Select("COALESCE(MAX(multiplier), 1.0)").
		Scan(&multiplier)
	if multiplier < 1.0 {
		return 1.0
	}
	return multiplier
}

func (s *RechargeService) checkFirstRechargeBonus(ctx context.Context, user *entities.User, _ int64) {
	count, err := s.txRepo.CountByPhoneAndTypeSince(ctx, user.PhoneNumber, entities.TxTypeRecharge, 0)
	if err != nil || count != 1 {
		return // Not first recharge
	}
	bonus := s.cfg.GetInt64("first_recharge_bonus_points", 20)
	ref := "first_recharge_bonus_" + user.ID.String()
	tx := &entities.Transaction{
		ID:          uuid.New(),
		UserID:      user.ID,
		PhoneNumber: user.PhoneNumber,
		Type:        entities.TxTypeBonus,
		PointsDelta: bonus,
		Reference:   ref,
		CreatedAt:   time.Now(),
	}
	if err := s.txRepo.Save(ctx, tx); err != nil {
		log.Printf("[BONUS] First recharge bonus failed for %s: %v", user.PhoneNumber, err)
	}
	welcomeMsg := fmt.Sprintf("Welcome to Loyalty Nexus! You have earned %d bonus Pulse Points. Start exploring Nexus Studio now.", bonus)
	_ = s.notifySvc.SendSMS(ctx, user.PhoneNumber, welcomeMsg)
}
