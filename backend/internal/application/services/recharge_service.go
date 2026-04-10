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
	"os"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/pkg/safe"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrDuplicateRecharge = errors.New("recharge already processed")

// PaystackEvent is the incoming webhook payload from Paystack.
type PaystackEvent struct {
	Event string         `json:"event"`
	Data  PaystackCharge `json:"data"`
}

type PaystackCharge struct {
	Reference string `json:"reference"`
	Amount    int64  `json:"amount"` // in kobo
	Status    string `json:"status"`
	Customer  struct {
		PhoneNumber string `json:"phone"`
	} `json:"customer"`
	Metadata json.RawMessage `json:"metadata"`
}

type RechargeService struct {
	userRepo       repositories.UserRepository
	txRepo         repositories.TransactionRepository
	notifySvc      *NotificationService
	cfg            *config.ConfigManager
	db             *gorm.DB
	drawWindowSvc  *DrawWindowService // resolves which draws a recharge qualifies for
}

func NewRechargeService(
	ur repositories.UserRepository,
	tr repositories.TransactionRepository,
	ns *NotificationService,
	cfg *config.ConfigManager,
	db *gorm.DB,
	dws *DrawWindowService,
) *RechargeService {
	return &RechargeService{
		userRepo:      ur,
		txRepo:        tr,
		notifySvc:     ns,
		cfg:           cfg,
		db:            db,
		drawWindowSvc: dws,
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

	// Minimum recharge check — reads min_recharge_naira from DB (set by Points Engine UI)
	minKobo := s.cfg.GetInt64("min_recharge_naira", 50) * 100
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

// processAwardTransaction is the core atomic reward function — executed inside a DB transaction.
//
// Reward logic (all thresholds are admin-configurable via the Points Engine UI):
//
//   - Pulse Points: flat accumulator — every pulse_naira_per_point naira = 1 point.
//     The kobo remainder carries forward in wallet.pulse_counter across transactions.
//
//   - Draw Entries: flat accumulator — every draw_naira_per_entry naira = 1 entry.
//     The kobo remainder carries forward in wallet.draw_counter, resetting daily (WAT).
//
//   - Spin Credits: the ONLY tiered reward — based on the user's CUMULATIVE daily
//     recharge total crossing spin_tiers thresholds. Max spin_max_per_day per day.
func (s *RechargeService) processAwardTransaction(ctx context.Context, user *entities.User, amountKobo int64, reference string, isIntegrated bool) error {
	return s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// --- Row-level lock on wallet ---
		wallet, err := s.userRepo.GetWalletForUpdate(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("wallet lock failed: %w", err)
		}

		// ── Pulse Points (flat accumulator) ──────────────────────────────────
		// Every pulse_naira_per_point naira = 1 Pulse Point.
		// pulse_counter carries the kobo remainder across transactions (never wasted).
		pulseKoboPerPoint := s.cfg.GetInt64("pulse_naira_per_point", 250) * 100
		newPulseCounter := wallet.PulseCounter + amountKobo
		ptsEarned := newPulseCounter / pulseKoboPerPoint
		newPulseCounter = newPulseCounter % pulseKoboPerPoint

		// ── Draw Entries (flat daily accumulator) ─────────────────────────────
		// Every draw_naira_per_entry naira = 1 Draw Entry.
		// draw_counter resets at midnight WAT; remainder within the day carries forward.
		wat := time.FixedZone("WAT", 3600)
		todayWAT := time.Now().In(wat).Truncate(24 * time.Hour)
		effectiveDrawCounter := wallet.DrawCounter
		if wallet.DailyRechargeDate == nil || wallet.DailyRechargeDate.Before(todayWAT) {
			effectiveDrawCounter = 0 // New day — discard yesterday's remainder
		}
		drawThresholdKobo := s.cfg.GetInt64("draw_naira_per_entry", 200) * 100
		newDrawCounter := effectiveDrawCounter + amountKobo
		drawEntriesEarned := int(newDrawCounter / drawThresholdKobo)
		newDrawCounter = newDrawCounter % drawThresholdKobo
		// drawEntriesEarned is accumulated in draw_entries_today so the dashboard can show it

		// ── Spin Credits (tiered daily cumulative) ────────────────────────────
		// The ONLY tiered reward. Based on cumulative daily recharge crossing
		// spin_tiers thresholds. Max spin_max_per_day spins per calendar day.
		spinCreditsEarned, newDailyKobo, newDailySpinsAwarded, newDailyDate := s.calculateDailySpinCredits(
			ctx, wallet, amountKobo,
		)

		// ── Update wallet atomically ──────────────────────────────────────────
		// Reset draw_entries_today if it's a new WAT day
		isNewDay := wallet.DailyRechargeDate == nil || wallet.DailyRechargeDate.Before(todayWAT)
		newDrawEntriesToday := gorm.Expr("draw_entries_today + ?", drawEntriesEarned)
		if isNewDay {
			newDrawEntriesToday = gorm.Expr("?", drawEntriesEarned) // reset to today's count
		}

		walletUpdates := map[string]interface{}{
			"pulse_points":        gorm.Expr("pulse_points + ?", ptsEarned),
			"lifetime_points":     gorm.Expr("lifetime_points + ?", ptsEarned),
			"pulse_counter":       newPulseCounter,
			"draw_counter":        newDrawCounter,
			"daily_recharge_kobo": newDailyKobo,
			"daily_recharge_date": newDailyDate,
			"daily_spins_awarded": newDailySpinsAwarded,
			"draw_entries_today":  newDrawEntriesToday,
			"draw_entries_date":   todayWAT,
		}
		if spinCreditsEarned > 0 {
			walletUpdates["spin_credits"] = gorm.Expr("spin_credits + ?", spinCreditsEarned)
		}
		if err := dbTx.Table("wallets").Where("user_id = ?", wallet.UserID).Updates(walletUpdates).Error; err != nil {
			return fmt.Errorf("wallet update failed: %w", err)
		}
		// Update in-memory struct for subsequent use in this function
		wallet.PulsePoints += ptsEarned
		wallet.LifetimePoints += ptsEarned
		wallet.PulseCounter = newPulseCounter
		wallet.SpinCredits += spinCreditsEarned
		wallet.DrawCounter = newDrawCounter
		wallet.DailyRechargeKobo = newDailyKobo
		wallet.DailySpinsAwarded = newDailySpinsAwarded
		wallet.DailyRechargeDate = &newDailyDate

		// ── Update streak ─────────────────────────────────────────────────────
		streakHours := s.cfg.GetInt("streak_expiry_hours", 36)
		newStreak := s.calculateNewStreak(user, streakHours)
		streakExpiresAt := time.Now().Add(time.Duration(streakHours) * time.Hour)
		if err := s.userRepo.UpdateStreak(ctx, user.ID, newStreak, streakExpiresAt); err != nil {
			return err
		}

		// ── Insert draw_entries rows for qualifying draws ────────────────────
		// Every draw_naira_per_entry naira recharge = 1 draw entry ticket.
		// Entries are written into each active draw whose window this recharge falls into.
		if drawEntriesEarned > 0 && s.drawWindowSvc != nil {
			qualifyingDraws, qErr := s.drawWindowSvc.ResolveQualifyingDraws(context.Background(), time.Now())
			if qErr == nil {
				for _, qd := range qualifyingDraws {
					entry := map[string]interface{}{
						"id":           uuid.New(),
						"draw_id":      qd.DrawID,
						"user_id":      user.ID,
						"msisdn":       user.PhoneNumber,
						"entry_source": "recharge",
						"amount":       amountKobo,
						"entries_count": drawEntriesEarned,
						"created_at":   time.Now(),
					}
					if err := dbTx.Table("draw_entries").Create(&entry).Error; err != nil {
						log.Printf("[Recharge] draw entry insert failed draw=%s user=%s: %v", qd.DrawID, user.ID, err)
						// non-fatal — continue processing
					}
				}
			}
		}

		// ── Update user recharge stats ────────────────────────────────────────
		now := time.Now()
		user.TotalRechargeAmount += amountKobo
		user.LastRechargeAt = &now
		if err := s.userRepo.Update(ctx, user); err != nil {
			return err
		}

		// ── Update user tier (based on lifetime points) ───────────────────────
		newTier := entities.TierFromLifetimePoints(wallet.LifetimePoints)
		if newTier != user.Tier {
			_ = s.userRepo.UpdateTier(ctx, user.ID, newTier)
		}

		// ── Write immutable ledger entries ────────────────────────────────────
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
				"amount_kobo":          amountKobo,
				"pulse_naira_per_point": pulseKoboPerPoint / 100,
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
		safe.Go(func() {
			s.checkFirstRechargeBonus(context.Background(), user, ptsEarned)
		})

		log.Printf("[RECHARGE] Processed %s: ₦%d -> +%d pts, +%d spins (streak: %d)",
			user.PhoneNumber, amountKobo/100, ptsEarned, spinCreditsEarned, newStreak)

		return nil
	})
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

// calculateDailySpinCredits implements the tier-based daily spin credit logic.
//
// Spin credits are the ONLY tiered reward. They are based on the user's CUMULATIVE
// daily recharge total (in kobo) crossing spin_tiers thresholds — NOT per-transaction.
//
// Algorithm:
//  1. Reset daily_recharge_kobo if daily_recharge_date != today (WAT).
//  2. Add amountKobo to the daily total.
//  3. Look up the matching spin tier for the new cumulative total.
//  4. spinCreditsEarned = tier.SpinsPerDay - wallet.DailySpinsAwarded (never negative).
//  5. Increment daily_spins_awarded by spinCreditsEarned.
//
// The WAT timezone (UTC+1) is used for the daily reset boundary.
func (s *RechargeService) calculateDailySpinCredits(
	ctx context.Context,
	wallet *entities.Wallet,
	amountKobo int64,
) (spinCreditsEarned int, newDailyKobo int64, newDailySpinsAwarded int, newDailyDate time.Time) {
	// WAT = UTC+1
	wat := time.FixedZone("WAT", 3600)
	todayWAT := time.Now().In(wat).Truncate(24 * time.Hour)

	// Step 1: Reset daily counters if date has changed
	currentDailyKobo := wallet.DailyRechargeKobo
	currentDailySpins := wallet.DailySpinsAwarded
	if wallet.DailyRechargeDate == nil || wallet.DailyRechargeDate.Before(todayWAT) {
		currentDailyKobo = 0
		currentDailySpins = 0
	}

	// Step 2: Add this recharge to the daily total
	newDailyKobo = currentDailyKobo + amountKobo
	newDailyDate = todayWAT

	// Step 3: Look up the spin tier for the new cumulative total from the DB.
	// Falls back to hardcoded thresholds if the spin_tiers table is empty.
	var tiers []entities.SpinTier
	s.db.WithContext(ctx).
		Table("spin_tiers").
		Where("is_active = true AND min_daily_amount <= ? AND max_daily_amount >= ?", newDailyKobo, newDailyKobo).
		Order("spins_per_day DESC").
		Limit(1).
		Find(&tiers)

	var spinsPerDay int
	if len(tiers) > 0 {
		spinsPerDay = tiers[0].SpinsPerDay
	} else {
		// Fallback: hardcoded thresholds matching migration 067 canonical tiers (in kobo)
		// Bronze:   ₦1,000–₦4,999  → 1 spin
		// Silver:   ₦5,000–₦9,999  → 2 spins
		// Gold:     ₦10,000–₦19,999 → 3 spins
		// Platinum: ₦20,000+        → 5 spins
		switch {
		case newDailyKobo >= 2000000: // ₦20,000+ → Platinum
			spinsPerDay = 5
		case newDailyKobo >= 1000000: // ₦10,000–₦19,999 → Gold
			spinsPerDay = 3
		case newDailyKobo >= 500000: // ₦5,000–₦9,999 → Silver
			spinsPerDay = 2
		case newDailyKobo >= 100000: // ₦1,000–₦4,999 → Bronze
			spinsPerDay = 1
		default:
			spinsPerDay = 0 // Below ₦1,000 — no spins
		}
	}

	// Respect the global daily spin cap from config
	spinMaxPerDay := s.cfg.GetInt("spin_max_per_day", 5)
	if spinsPerDay > spinMaxPerDay {
		spinsPerDay = spinMaxPerDay
	}

	// Step 4: Award the DIFFERENCE between the new tier cap and spins already awarded today.
	// This prevents double-awarding when the user makes multiple recharges in one day.
	spinCreditsEarned = spinsPerDay - currentDailySpins
	if spinCreditsEarned < 0 {
		spinCreditsEarned = 0
	}

	// Step 5: Update the daily spins awarded counter
	newDailySpinsAwarded = currentDailySpins + spinCreditsEarned

	return spinCreditsEarned, newDailyKobo, newDailySpinsAwarded, newDailyDate
}
