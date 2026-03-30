package services

// mtn_push_service.go
//
// Handles the MTN-push recharge pipeline for Loyalty Nexus.
//
// MTN pushes recharge events directly to us via a signed HTTP POST.
// Unlike RechargeMax (where the recharge happens on-platform and is
// confirmed via Paystack/VTPass), here MTN is the authoritative source —
// the recharge has already happened on the MTN network.
//
// ─── Reward Rules ────────────────────────────────────────────────────────────
//
//  Spin Credits  — TIER-BASED on CUMULATIVE DAILY recharge (resets midnight WAT):
//    ₦1,000–₦4,999/day  → 1 spin  (Bronze)
//    ₦5,000–₦9,999/day  → 2 spins (Silver)
//    ₦10,000–₦19,999/day → 3 spins (Gold)
//    ₦20,000+/day        → 5 spins (Platinum)
//    The tier's spins_per_day is the DAILY CAP, not additive per transaction.
//    Each recharge that pushes the cumulative daily total into a higher tier
//    awards the DIFFERENCE (new_cap - already_awarded_today).
//
//  Draw Entries  — SIMPLE ACCUMULATOR per transaction:
//    Every ₦200 recharge = 1 Draw Entry (draw_counter tracks kobo remainder).
//    Admin-configurable via network_configs: draw_naira_per_entry (default 200).
//
//  Pulse Points  — SIMPLE ACCUMULATOR:
//    Every ₦250 recharge = 1 Pulse Point (AI Studio currency).
//    Admin-configurable via network_configs: pulse_naira_per_point (default 250).
//
// ─── Pipeline ─────────────────────────────────────────────────────────────────
//
//  1. Idempotency check — reject duplicates via transaction_ref
//  2. Minimum amount guard — configurable via network_configs
//  3. Audit log — write mtn_push_events row immediately (status=RECEIVED)
//  4. Resolve or auto-create user account
//  5. ATOMIC DB TRANSACTION:
//     a. Row-lock wallet (SELECT FOR UPDATE)
//     b. Reset daily counters if date has changed (midnight WAT rollover)
//     c. Calculate Spin Credits (tier-based: cumulative daily recharge → spin_tiers)
//     d. Calculate Draw Entries (₦200 accumulator, draw_counter)
//     e. Calculate Pulse Points (₦250 accumulator, pulse_counter)
//     f. Update wallet (spin_credits, pulse_points, lifetime_points,
//                       draw_counter, pulse_counter, daily_recharge_kobo,
//                       daily_recharge_date, daily_spins_awarded)
//     g. Write immutable ledger entries (recharge, spin_credit_award, draw_entry_award, pulse_points_award)
//     h. Update user streak + stats
//  6. POST-COMMIT (non-fatal, never rolls back the payment):
//     a. Create draw_entries rows for the active draw (1 per draw entry earned)
//     b. Send SMS notification
//     c. Update mtn_push_events row to status=PROCESSED

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
	"unicode"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/pkg/safe"
	"loyalty-nexus/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ─── Request / Response types ─────────────────────────────────────────────────

// MTNPushPayload is the raw event body that MTN sends to
// POST /api/v1/recharge/mtn-push.
// Fields match the MTN BSS notification spec.
type MTNPushPayload struct {
	// TransactionRef is MTN's unique transaction ID — used for idempotency.
	TransactionRef string `json:"transaction_ref"`
	// MSISDN is the subscriber's phone number (any format; normalised internally).
	MSISDN string `json:"msisdn"`
	// RechargeType is "AIRTIME", "DATA", or "BUNDLE".
	RechargeType string `json:"recharge_type"`
	// Amount is the recharge value in NAIRA (not kobo) — MTN sends naira.
	Amount float64 `json:"amount"`
	// Timestamp is the event time from MTN's system (ISO-8601 or RFC3339).
	Timestamp string `json:"timestamp"`
}

// MTNPushResult is returned to the caller after processing.
type MTNPushResult struct {
	EventID     uuid.UUID `json:"event_id"`
	MSISDN      string    `json:"msisdn"`
	PulsePoints int64     `json:"pulse_points_awarded"`
	DrawEntries int       `json:"draw_entries_created"`
	SpinCredits int       `json:"spin_credits_awarded"`
	SpinTier    string    `json:"spin_tier,omitempty"`
	IsDuplicate bool      `json:"is_duplicate"`
}

// ─── mtn_push_events DB model ─────────────────────────────────────────────────

// mtnPushEvent mirrors the mtn_push_events table added in migration 045.
type mtnPushEvent struct {
	ID                 uuid.UUID  `gorm:"column:id;primaryKey"`
	TransactionRef     string     `gorm:"column:transaction_ref"`
	MSISDN             string     `gorm:"column:msisdn"`
	RechargeType       string     `gorm:"column:recharge_type"`
	AmountKobo         int64      `gorm:"column:amount_kobo"`
	EventTimestamp     time.Time  `gorm:"column:event_timestamp"`
	RawPayload         []byte     `gorm:"column:raw_payload"`
	Status             string     `gorm:"column:status"`
	ProcessingError    string     `gorm:"column:processing_error"`
	PointsAwarded      int64      `gorm:"column:points_awarded"`
	DrawEntriesCreated int        `gorm:"column:draw_entries_created"`
	SpinCreditsAwarded int        `gorm:"column:spin_credits_awarded"`
	ProcessedAt        *time.Time `gorm:"column:processed_at"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (mtnPushEvent) TableName() string { return "mtn_push_events" }

// ─── Service ──────────────────────────────────────────────────────────────────

// MTNPushService processes inbound MTN recharge push events.
type MTNPushService struct {
	db          *gorm.DB
	userRepo    repositories.UserRepository
	txRepo      repositories.TransactionRepository
	drawSvc     drawService
	drawWindows drawWindowResolver
	notifySvc   *NotificationService
	cfg         *config.ConfigManager
	tierCalc    *utils.SpinTierCalculatorDB
}

// drawService is the subset of DrawService used here.
// Defined as an interface so tests can inject a mock.
type drawService interface {
	AddEntry(ctx context.Context, drawID, userID uuid.UUID, phone, source string, amount int64, tickets int) error
}

// drawWindowResolver resolves which draws a recharge qualifies for.
// Defined as an interface so tests can inject a mock.
type drawWindowResolver interface {
	ResolveQualifyingDraws(ctx context.Context, rechargeTime time.Time) ([]QualifyingDraw, error)
}

// NewMTNPushService constructs the service.
func NewMTNPushService(
	db *gorm.DB,
	userRepo repositories.UserRepository,
	txRepo repositories.TransactionRepository,
	drawSvc drawService,
	drawWindows drawWindowResolver,
	notifySvc *NotificationService,
	cfg *config.ConfigManager,
) *MTNPushService {
	return &MTNPushService{
		db:          db,
		userRepo:    userRepo,
		txRepo:      txRepo,
		drawSvc:     drawSvc,
		drawWindows: drawWindows,
		notifySvc:   notifySvc,
		cfg:         cfg,
		tierCalc:    utils.NewSpinTierCalculatorDB(db),
	}
}

// ProcessMTNPush is the main entry point. It is safe to call concurrently.
func (s *MTNPushService) ProcessMTNPush(ctx context.Context, payload MTNPushPayload) (*MTNPushResult, error) {
	// ── 0. Normalise inputs ───────────────────────────────────────────────────
	phone := normalisePhone(payload.MSISDN)
	rechargeType := strings.ToUpper(strings.TrimSpace(payload.RechargeType))
	if rechargeType == "" {
		rechargeType = "AIRTIME"
	}
	amountKobo := int64(math.Round(payload.Amount * 100))

	// ── 1. Idempotency ────────────────────────────────────────────────────────
	var existing mtnPushEvent
	if err := s.db.WithContext(ctx).
		Where("transaction_ref = ?", payload.TransactionRef).
		First(&existing).Error; err == nil {
		// Already processed — return the cached result.
		return &MTNPushResult{
			EventID:     existing.ID,
			MSISDN:      existing.MSISDN,
			PulsePoints: existing.PointsAwarded,
			DrawEntries: existing.DrawEntriesCreated,
			SpinCredits: existing.SpinCreditsAwarded,
			IsDuplicate: true,
		}, nil
	}

	// ── 2. Minimum amount guard ───────────────────────────────────────────────
	minNaira := s.cfg.GetInt64("mtn_push_min_amount_naira", 50)
	if payload.Amount < float64(minNaira) {
		return nil, fmt.Errorf("recharge amount ₦%.2f is below minimum ₦%d", payload.Amount, minNaira)
	}

	// ── 3. Parse event timestamp ──────────────────────────────────────────────
	eventTime := time.Now()
	if payload.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, payload.Timestamp); err == nil {
			eventTime = t
		} else if t, err := time.Parse("2006-01-02T15:04:05", payload.Timestamp); err == nil {
			eventTime = t
		}
	}

	// ── 4. Write audit record (status=RECEIVED) ───────────────────────────────
	rawJSON, _ := json.Marshal(payload)
	event := &mtnPushEvent{
		ID:             uuid.New(),
		TransactionRef: payload.TransactionRef,
		MSISDN:         phone,
		RechargeType:   rechargeType,
		AmountKobo:     amountKobo,
		EventTimestamp: eventTime,
		RawPayload:     rawJSON,
		Status:         "RECEIVED",
	}
	if err := s.db.WithContext(ctx).Create(event).Error; err != nil {
		return nil, fmt.Errorf("failed to write mtn_push_events audit row: %w", err)
	}

	// ── 5. Resolve user (auto-create if first recharge) ───────────────────────
	user, err := s.resolveOrCreateUser(ctx, phone)
	if err != nil {
		s.markEventFailed(ctx, event, err)
		return nil, fmt.Errorf("user resolution failed: %w", err)
	}

	// ── 6. Atomic DB transaction ──────────────────────────────────────────────
	var spinCreditsEarned int
	var drawEntriesEarned int
	var pulsePointsEarned int64
	var spinTierName string

	txErr := s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Row-level lock on wallet — prevents concurrent double-award.
		// IMPORTANT: use dbTx for ALL operations inside this callback so they
		// run on the same connection and participate in the same transaction.
		var wallet entities.Wallet
		if err := dbTx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", user.ID).First(&wallet).Error; err != nil {
			return fmt.Errorf("wallet lock failed: %w", err)
		}

		// ── Daily counter reset (midnight WAT rollover) ───────────────────────
		// WAT = UTC+1. We compare the calendar date of the event against the
		// date stored in daily_recharge_date. If they differ, reset the daily
		// counters so each day starts fresh.
		watLocation := time.FixedZone("WAT", 1*60*60)
		eventDateWAT := eventTime.In(watLocation).Truncate(24 * time.Hour)

		if wallet.DailyRechargeDate == nil || wallet.DailyRechargeDate.In(watLocation).Truncate(24*time.Hour).Before(eventDateWAT) {
			// New day — reset daily tracking
			wallet.DailyRechargeKobo = 0
			wallet.DailySpinsAwarded = 0
			wallet.DailyRechargeDate = &eventDateWAT
		}

		// ── Spin Credit calculation (tier-based, cumulative daily) ────────────
		//
		// Algorithm:
		//   1. Add this recharge to today's cumulative total.
		//   2. Look up the spin_tiers table to find the tier for the new total.
		//   3. The tier's spins_per_day is the DAILY CAP for this user today.
		//   4. Award = max(0, tier.spins_per_day - wallet.DailySpinsAwarded).
		//   5. Cap the total at spin_max_per_day (admin-configurable, default 5).
		//
		// This means:
		//   - First recharge of ₦1,000 → Bronze tier → 1 spin cap → award 1 spin
		//   - Second recharge of ₦4,000 (total ₦5,000) → Silver tier → 2 spin cap → award 1 more
		//   - Third recharge of ₦5,000 (total ₦10,000) → Gold tier → 3 spin cap → award 1 more
		//   - Recharges below ₦1,000 cumulative → no tier → 0 spins
		newDailyTotal := wallet.DailyRechargeKobo + amountKobo
		spinMaxPerDay := s.cfg.GetInt("spin_max_per_day", 5)

		tier, tierErr := s.tierCalc.GetSpinTierFromDB(newDailyTotal)
		if tierErr == nil && tier != nil {
			// User qualifies for a tier — calculate incremental spin award
			tierCap := tier.SpinsPerDay
			if tierCap > spinMaxPerDay {
				tierCap = spinMaxPerDay
			}
			spinCreditsEarned = tierCap - wallet.DailySpinsAwarded
			if spinCreditsEarned < 0 {
				spinCreditsEarned = 0
			}
			spinTierName = tier.TierDisplayName
		}
		// If tierErr != nil (e.g., below ₦1,000 threshold), spinCreditsEarned stays 0

		// ── Draw Entry calculation (₦200 simple accumulator) ─────────────────
		// Every ₦200 of the CURRENT TRANSACTION = 1 draw entry.
		// draw_counter carries the kobo remainder across transactions.
		drawKoboPerEntry := s.cfg.GetInt64("draw_naira_per_entry", 200) * 100
		newDrawCounter := wallet.DrawCounter + amountKobo
		drawEntriesEarned = int(newDrawCounter / drawKoboPerEntry)
		newDrawCounter = newDrawCounter % drawKoboPerEntry

		// ── Pulse Point calculation (₦250 accumulator) ───────────────────────
		// Every ₦250 recharge = 1 Pulse Point (AI Studio currency).
		pulseKoboPerPoint := s.cfg.GetInt64("pulse_naira_per_point", 250) * 100
		newPulseCounter := wallet.PulseCounter + amountKobo
		pulsePointsEarned = newPulseCounter / pulseKoboPerPoint
		newPulseCounter = newPulseCounter % pulseKoboPerPoint

		// ── Update wallet atomically ──────────────────────────────────────────
		updates := map[string]interface{}{
			"draw_counter":          newDrawCounter,
			"pulse_counter":         newPulseCounter,
			"daily_recharge_kobo":   newDailyTotal,
			"daily_recharge_date":   wallet.DailyRechargeDate,
			"daily_spins_awarded":   wallet.DailySpinsAwarded + spinCreditsEarned,
		}
		if spinCreditsEarned > 0 {
			updates["spin_credits"] = gorm.Expr("spin_credits + ?", spinCreditsEarned)
		}
		if pulsePointsEarned > 0 {
			updates["pulse_points"] = gorm.Expr("pulse_points + ?", pulsePointsEarned)
			updates["lifetime_points"] = gorm.Expr("lifetime_points + ?", pulsePointsEarned)
		}
		if err := dbTx.Table("wallets").
			Where("user_id = ?", wallet.UserID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("wallet update failed: %w", err)
		}
		wallet.SpinCredits += spinCreditsEarned
		wallet.PulsePoints += pulsePointsEarned
		wallet.LifetimePoints += pulsePointsEarned
		wallet.DrawCounter = newDrawCounter
		wallet.PulseCounter = newPulseCounter
		wallet.DailyRechargeKobo = newDailyTotal
		wallet.DailySpinsAwarded += spinCreditsEarned

		// ── Immutable ledger entries ──────────────────────────────────────────
		ref := "MTN-" + payload.TransactionRef

		// 1. Recharge record — always written, even if no rewards earned.
		rechargeTx := &entities.Transaction{
			ID:           uuid.New(),
			UserID:       user.ID,
			PhoneNumber:  phone,
			Type:         entities.TxTypeRecharge,
			Amount:       amountKobo,
			BalanceAfter: wallet.PulsePoints,
			Reference:    ref,
			CreatedAt:    time.Now(),
		}
		if err := s.txRepo.SaveTx(ctx, dbTx, rechargeTx); err != nil {
			return err
		}

		// 2. Spin credit award record.
		if spinCreditsEarned > 0 {
			meta, _ := json.Marshal(map[string]interface{}{
				"amount_kobo":         amountKobo,
				"recharge_type":       rechargeType,
				"daily_total_kobo":    newDailyTotal,
				"spin_tier":           spinTierName,
				"daily_spins_cap":     wallet.DailySpinsAwarded,
				"source":              "mtn_push",
			})
			spinTx := &entities.Transaction{
				ID:          uuid.New(),
				UserID:      user.ID,
				PhoneNumber: phone,
				Type:        entities.TxTypeSpinCreditAward,
				SpinDelta:   spinCreditsEarned,
				Reference:   ref + "_spin",
				Metadata:    meta,
				CreatedAt:   time.Now(),
			}
			if err := s.txRepo.SaveTx(ctx, dbTx, spinTx); err != nil {
				return err
			}
		}

		// 3. Draw entry award record.
		if drawEntriesEarned > 0 {
			meta, _ := json.Marshal(map[string]interface{}{
				"amount_kobo":   amountKobo,
				"recharge_type": rechargeType,
				"threshold":     s.cfg.GetInt64("draw_naira_per_entry", 200),
				"source":        "mtn_push",
			})
			drawTx := &entities.Transaction{
				ID:          uuid.New(),
				UserID:      user.ID,
				PhoneNumber: phone,
				Type:        entities.TxTypeDrawEntryAward,
				SpinDelta:   drawEntriesEarned, // reuse SpinDelta field for entry count
				Reference:   ref + "_draw",
				Metadata:    meta,
				CreatedAt:   time.Now(),
			}
			if err := s.txRepo.SaveTx(ctx, dbTx, drawTx); err != nil {
				return err
			}
		}

		// 4. Pulse Point award record.
		if pulsePointsEarned > 0 {
			meta, _ := json.Marshal(map[string]interface{}{
				"amount_kobo":   amountKobo,
				"recharge_type": rechargeType,
				"threshold":     s.cfg.GetInt64("pulse_naira_per_point", 250),
				"source":        "mtn_push",
			})
			pulseTx := &entities.Transaction{
				ID:           uuid.New(),
				UserID:       user.ID,
				PhoneNumber:  phone,
				Type:         entities.TxTypePointsAward,
				PointsDelta:  pulsePointsEarned,
				BalanceAfter: wallet.PulsePoints,
				Reference:    ref + "_pulse",
				Metadata:     meta,
				CreatedAt:    time.Now(),
			}
			if err := s.txRepo.SaveTx(ctx, dbTx, pulseTx); err != nil {
				return err
			}
		}

		// ── User streak + stats (all on dbTx — same connection as the wallet lock) ──
		streakHours := s.cfg.GetInt("streak_expiry_hours", 36)
		newStreak := calcStreak(user, streakHours)
		expiresAt := time.Now().Add(time.Duration(streakHours) * time.Hour)
		if err := dbTx.Table("users").Where("id = ?", user.ID).
			Updates(map[string]interface{}{
				"streak_count":      newStreak,
				"streak_expires_at": expiresAt,
			}).Error; err != nil {
			return err
		}
		now := time.Now()
		user.TotalRechargeAmount += amountKobo
		user.LastRechargeAt = &now
		if err := dbTx.Save(user).Error; err != nil {
			return err
		}
		newTier := entities.TierFromLifetimePoints(wallet.LifetimePoints)
		if newTier != user.Tier {
			_ = dbTx.Table("users").Where("id = ?", user.ID).Update("tier", newTier).Error
		}
		return nil // commit
	})
	if txErr != nil {
		s.markEventFailed(ctx, event, txErr)
		return nil, fmt.Errorf("MTN push processing failed: %w", txErr)
	}

	// ── 7. Post-commit side effects (non-fatal) ───────────────────────────────
	// These MUST NOT be inside the DB transaction — a failure here must never
	// roll back the wallet update (same pattern as RechargeMax).
	drawEntriesCreated := 0
	safe.Go(func() {
		bgCtx := context.Background()

		// 7a. Draw entries — inserted into EACH qualifying draw window.
		//
		// Draw entries are based on the ₦200 accumulator (drawEntriesEarned),
		// NOT on spin credits. These are separate currencies.
		//
		// Window rules (from draw_schedules table, admin-configurable):
		//   Recharges from 17:00:01 WAT yesterday → 17:00:00 WAT today qualify
		//   for TOMORROW's daily draw AND for the Saturday weekly mega draw.
		if drawEntriesEarned > 0 && s.drawWindows != nil {
			qualifyingDraws, wErr := s.drawWindows.ResolveQualifyingDraws(bgCtx, eventTime)
			if wErr != nil {
				log.Printf("[MTN-PUSH] draw window resolution failed (non-fatal): %v", wErr)
			} else {
				for _, qd := range qualifyingDraws {
					addErr := s.drawSvc.AddEntry(
						bgCtx,
						qd.DrawID,
						user.ID,
						phone,
						"recharge",
						amountKobo,
						drawEntriesEarned, // 1 draw entry per ₦200, per qualifying draw
					)
					if addErr != nil {
						log.Printf("[MTN-PUSH] draw entry creation failed for draw %s (%s) (non-fatal): %v",
							qd.DrawID, qd.DrawName, addErr)
					} else {
						drawEntriesCreated += drawEntriesEarned
					}
				}
			}
			if len(qualifyingDraws) == 0 {
				// No active draws in window — entries will be imported at draw creation time.
				log.Printf("[MTN-PUSH] no qualifying draws for recharge at %s (non-fatal)", eventTime.Format(time.RFC3339))
			}
		}

		// 7b. SMS notification.
		if s.notifySvc != nil {
			amountNaira := int64(payload.Amount)
			var parts []string
			if spinCreditsEarned > 0 {
				parts = append(parts, fmt.Sprintf("%d Spin Credit(s) [%s tier]", spinCreditsEarned, spinTierName))
			}
			if drawEntriesEarned > 0 {
				parts = append(parts, fmt.Sprintf("%d Draw Entr%s", drawEntriesEarned, map[bool]string{true: "y", false: "ies"}[drawEntriesEarned == 1]))
			}
			if pulsePointsEarned > 0 {
				parts = append(parts, fmt.Sprintf("%d Pulse Point(s)", pulsePointsEarned))
			}
			msg := fmt.Sprintf("Your MTN recharge of ₦%d has been processed.", amountNaira)
			if len(parts) > 0 {
				msg += " You earned " + strings.Join(parts, ", ") + "."
			}
			msg += " Keep recharging to win big!"
			if smsErr := s.notifySvc.SendSMS(bgCtx, phone, msg); smsErr != nil {
				log.Printf("[MTN-PUSH] SMS notification failed for %s (non-fatal): %v", phone, smsErr)
			}
		}

		// 7c. Mark audit event as PROCESSED.
		now := time.Now()
		s.db.WithContext(bgCtx).Model(event).Updates(map[string]interface{}{
			"status":               "PROCESSED",
			"points_awarded":       pulsePointsEarned,
			"draw_entries_created": drawEntriesCreated,
			"spin_credits_awarded": spinCreditsEarned,
			"processed_at":         &now,
		})
	})

	log.Printf("[MTN-PUSH] Processed %s: ₦%.2f %s -> +%d spin credits (%s tier), +%d draw entries, +%d pulse pts",
		phone, payload.Amount, rechargeType, spinCreditsEarned, spinTierName, drawEntriesEarned, pulsePointsEarned)

	return &MTNPushResult{
		EventID:     event.ID,
		MSISDN:      phone,
		PulsePoints: pulsePointsEarned,
		DrawEntries: drawEntriesCreated,
		SpinCredits: spinCreditsEarned,
		SpinTier:    spinTierName,
	}, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// resolveOrCreateUser finds the user by phone number.
// If no account exists yet (first recharge before app registration),
// a minimal account is auto-created — matching the RechargeMax guest pattern.
func (s *MTNPushService) resolveOrCreateUser(ctx context.Context, phone string) (*entities.User, error) {
	user, err := s.userRepo.FindByPhoneNumber(ctx, phone)
	if err == nil {
		return user, nil
	}
	// Auto-create a minimal account.
	// user_code has a UNIQUE constraint — use UUID-derived value to guarantee
	// uniqueness even under concurrent auto-creates.
	uid := uuid.New()
	userCode := "MTN" + strings.ToUpper(uid.String()[:8])
	newUser := &entities.User{
		ID:          uid,
		PhoneNumber: phone,
		UserCode:    userCode,
		Tier:        "BRONZE",
		IsActive:    true,
		CreatedAt:   time.Now(),
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("auto-create user failed: %w", err)
	}
	// Create the wallet row. Explicitly set ID so GORM doesn't try to insert
	// a zero UUID (which would violate the NOT NULL constraint on id).
	if err := s.db.WithContext(ctx).Create(&entities.Wallet{
		ID:          uuid.New(),
		UserID:      newUser.ID,
		PulsePoints: 0,
		SpinCredits: 0,
	}).Error; err != nil {
		return nil, fmt.Errorf("wallet creation failed: %w", err)
	}
	return newUser, nil
}

// markEventFailed updates the mtn_push_events row to status=FAILED.
func (s *MTNPushService) markEventFailed(ctx context.Context, event *mtnPushEvent, err error) {
	s.db.WithContext(ctx).Model(event).Updates(map[string]interface{}{
		"status":           "FAILED",
		"processing_error": err.Error(),
	})
}

// normalisePhone converts any Nigerian phone format to 0XXXXXXXXXX.
// Supported inputs: 2348012345678, +2348012345678, 08012345678,
//
//	8012345678, 234-801-234-5678.
func normalisePhone(raw string) string {
	// Strip non-digit characters.
	var digits strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	switch {
	case strings.HasPrefix(d, "234") && len(d) == 13:
		return "0" + d[3:]
	case strings.HasPrefix(d, "0") && len(d) == 11:
		return d
	case len(d) == 10:
		return "0" + d
	default:
		return d
	}
}

// calcStreak returns the new streak count for a user based on their last recharge.
func calcStreak(user *entities.User, windowHours int) int {
	if user.LastRechargeAt == nil {
		return 1
	}
	window := time.Duration(windowHours) * time.Hour
	if time.Since(*user.LastRechargeAt) <= window {
		return user.StreakCount + 1
	}
	return 1
}
