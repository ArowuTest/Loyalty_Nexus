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
//  Every ₦200 recharge  → 1 Spin Credit  + 1 Draw Entry
//  Every ₦250 recharge  → 1 Pulse Point  (AI Studio currency)
//
//  Both thresholds are admin-configurable via network_configs:
//    spin_draw_naira_per_credit  (default 200)
//    pulse_naira_per_point       (default 250)
//
//  Accumulator model (same as RechargeMax ÷200 pattern):
//    SpinDrawCounter accumulates kobo until it crosses spin_draw_naira_per_credit×100.
//    PulseCounter    accumulates kobo until it crosses pulse_naira_per_point×100.
//    This ensures two ₦100 recharges correctly award 1 spin at ₦200 threshold.
//
// ─── Pipeline ─────────────────────────────────────────────────────────────────
//
//  1. Idempotency check — reject duplicates via transaction_ref
//  2. Minimum amount guard — configurable via network_configs
//  3. Audit log — write mtn_push_events row immediately (status=RECEIVED)
//  4. Resolve or auto-create user account
//  5. ATOMIC DB TRANSACTION:
//     a. Row-lock wallet (SELECT FOR UPDATE)
//     b. Calculate Spin Credits + Draw Entries  (₦200 accumulator)
//     c. Calculate Pulse Points                 (₦250 accumulator)
//     d. Update wallet (spin_credits, pulse_points, lifetime_points,
//                       spin_draw_counter, pulse_counter)
//     e. Write immutable ledger entries (recharge, spin_credit_award, pulse_points_award)
//     f. Update user streak + stats
//  6. POST-COMMIT (non-fatal, never rolls back the payment):
//     a. Create draw_entries rows for the active draw (1 per spin credit earned)
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

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	EventID       uuid.UUID `json:"event_id"`
	MSISDN        string    `json:"msisdn"`
	PulsePoints   int64     `json:"pulse_points_awarded"`
	DrawEntries   int       `json:"draw_entries_created"`
	SpinCredits   int       `json:"spin_credits_awarded"`
	IsDuplicate   bool      `json:"is_duplicate"`
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
	db        *gorm.DB
	userRepo  repositories.UserRepository
	txRepo    repositories.TransactionRepository
	drawSvc   drawService
	notifySvc *NotificationService
	cfg       *config.ConfigManager
}

// drawService is the subset of DrawService used here.
// Defined as an interface so tests can inject a mock.
type drawService interface {
	GetActiveDrawID(ctx context.Context) (uuid.UUID, error)
	AddEntry(ctx context.Context, drawID, userID uuid.UUID, phone, source string, amount int64, tickets int) error
}

// NewMTNPushService constructs the service.
func NewMTNPushService(
	db *gorm.DB,
	userRepo repositories.UserRepository,
	txRepo repositories.TransactionRepository,
	drawSvc drawService,
	notifySvc *NotificationService,
	cfg *config.ConfigManager,
) *MTNPushService {
	return &MTNPushService{
		db:        db,
		userRepo:  userRepo,
		txRepo:    txRepo,
		drawSvc:   drawSvc,
		notifySvc: notifySvc,
		cfg:       cfg,
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
	var pulsePointsEarned int64

	txErr := s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Row-level lock on wallet — prevents concurrent double-award.
		wallet, err := s.userRepo.GetWalletForUpdate(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("wallet lock failed: %w", err)
		}

		// ── Spin Credit + Draw Entry calculation (₦200 accumulator) ──────────
		// Every ₦200 recharge = 1 spin credit + 1 draw entry.
		// spinDrawNairaPerCredit is admin-configurable (default 200).
		spinDrawKoboPerCredit := s.cfg.GetInt64("spin_draw_naira_per_credit", 200) * 100
		newSpinDrawCounter := wallet.SpinDrawCounter + amountKobo
		spinCreditsEarned = int(newSpinDrawCounter / spinDrawKoboPerCredit)
		newSpinDrawCounter = newSpinDrawCounter % spinDrawKoboPerCredit

		// ── Pulse Point calculation (₦250 accumulator) ───────────────────────
		// Every ₦250 recharge = 1 Pulse Point (AI Studio currency).
		// pulseNairaPerPoint is admin-configurable (default 250).
		pulseKoboPerPoint := s.cfg.GetInt64("pulse_naira_per_point", 250) * 100
		newPulseCounter := wallet.PulseCounter + amountKobo
		pulsePointsEarned = newPulseCounter / pulseKoboPerPoint
		newPulseCounter = newPulseCounter % pulseKoboPerPoint

		// ── Update wallet atomically ──────────────────────────────────────────
		updates := map[string]interface{}{
			"spin_draw_counter": newSpinDrawCounter,
			"pulse_counter":     newPulseCounter,
		}
		if spinCreditsEarned > 0 {
			updates["spin_credits"] = gorm.Expr("spin_credits + ?", spinCreditsEarned)
		}
		if pulsePointsEarned > 0 {
			updates["pulse_points"]    = gorm.Expr("pulse_points + ?", pulsePointsEarned)
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
		wallet.SpinDrawCounter = newSpinDrawCounter
		wallet.PulseCounter = newPulseCounter

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
				"amount_kobo":   amountKobo,
				"recharge_type": rechargeType,
				"threshold":     s.cfg.GetInt64("spin_draw_naira_per_credit", 200),
				"source":        "mtn_push",
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

		// 3. Pulse Point award record.
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

		// ── User streak + stats ───────────────────────────────────────────────
		streakHours := s.cfg.GetInt("streak_expiry_hours", 36)
		newStreak := calcStreak(user, streakHours)
		expiresAt := time.Now().Add(time.Duration(streakHours) * time.Hour)
		if err := s.userRepo.UpdateStreak(ctx, user.ID, newStreak, expiresAt); err != nil {
			return err
		}
		now := time.Now()
		user.TotalRechargeAmount += amountKobo
		user.LastRechargeAt = &now
		if err := s.userRepo.Update(ctx, user); err != nil {
			return err
		}
		newTier := entities.TierFromLifetimePoints(wallet.LifetimePoints)
		if newTier != user.Tier {
			_ = s.userRepo.UpdateTier(ctx, user.ID, newTier)
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

		// 7a. Draw entries — 1 entry per spin credit earned.
		// (Spin credit and draw entry are always awarded together at the ₦200 threshold.)
		if spinCreditsEarned > 0 {
			drawID, err := s.drawSvc.GetActiveDrawID(bgCtx)
			if err == nil {
				addErr := s.drawSvc.AddEntry(
					bgCtx,
					drawID,
					user.ID,
					phone,
					"recharge",
					amountKobo,
					spinCreditsEarned, // 1 draw entry per spin credit
				)
				if addErr != nil {
					log.Printf("[MTN-PUSH] draw entry creation failed (non-fatal): %v", addErr)
				} else {
					drawEntriesCreated = spinCreditsEarned
				}
			}
			// No active draw is fine — entries will be imported at draw time.
		}

		// 7b. SMS notification.
		if s.notifySvc != nil {
			amountNaira := int64(payload.Amount)
			var parts []string
			if spinCreditsEarned > 0 {
				parts = append(parts, fmt.Sprintf("%d Spin Credit(s)", spinCreditsEarned))
			}
			if pulsePointsEarned > 0 {
				parts = append(parts, fmt.Sprintf("%d Pulse Point(s)", pulsePointsEarned))
			}
			msg := fmt.Sprintf("Your MTN recharge of ₦%d has been processed.", amountNaira)
			if len(parts) > 0 {
				msg += " You earned " + strings.Join(parts, " and ") + "."
			}
			msg += " Keep recharging to win big!"
			s.notifySvc.SendSMS(bgCtx, phone, msg)
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

	log.Printf("[MTN-PUSH] Processed %s: ₦%.2f %s -> +%d spin credits, +%d pulse pts, +%d draw entries",
		phone, payload.Amount, rechargeType, spinCreditsEarned, pulsePointsEarned, drawEntriesCreated)

	return &MTNPushResult{
		EventID:     event.ID,
		MSISDN:      phone,
		PulsePoints: pulsePointsEarned,
		DrawEntries: drawEntriesCreated,
		SpinCredits: spinCreditsEarned,
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
	referralCode := "MTN" + strings.ToUpper(uuid.New().String()[:6])
	newUser := &entities.User{
		ID:           uuid.New(),
		PhoneNumber:  phone,
		ReferralCode: referralCode,
		Tier:         "BRONZE",
		IsActive:     true,
		CreatedAt:    time.Now(),
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("auto-create user failed: %w", err)
	}
	// Create the wallet row.
	if err := s.db.WithContext(ctx).Create(&entities.Wallet{
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
//                   8012345678, 234-801-234-5678.
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
