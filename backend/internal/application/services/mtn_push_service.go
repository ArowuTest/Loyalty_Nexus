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
// Pipeline (mirrors RechargeMax ussd_recharge_service.go):
//
//  1. Idempotency check — reject duplicates via transaction_ref
//  2. Minimum amount guard — configurable via network_configs
//  3. Audit log — write mtn_push_events row immediately (status=RECEIVED)
//  4. Resolve or auto-create user account
//  5. ATOMIC DB TRANSACTION:
//     a. Row-lock wallet (SELECT FOR UPDATE)
//     b. Calculate Pulse Points (same rate as Paystack recharge)
//     c. Calculate Spin Credits via RechargeCounter accumulator
//     d. Update wallet (pulse_points, spin_credits, recharge_counter)
//     e. Write immutable ledger entries (recharge, points_award, spin_credit_award)
//     f. Update user streak + stats
//  6. POST-COMMIT (non-fatal, never rolls back the payment):
//     a. Create draw_entries rows for the active draw (1 entry per point)
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
	EventID         uuid.UUID `json:"event_id"`
	MSISDN          string    `json:"msisdn"`
	PointsAwarded   int64     `json:"points_awarded"`
	DrawEntries     int       `json:"draw_entries_created"`
	SpinCredits     int       `json:"spin_credits_awarded"`
	IsDuplicate     bool      `json:"is_duplicate"`
}

// ─── mtn_push_events DB model ─────────────────────────────────────────────────

// mtnPushEvent mirrors the mtn_push_events table added in migration 045.
type mtnPushEvent struct {
	ID                  uuid.UUID  `gorm:"column:id;primaryKey"`
	TransactionRef      string     `gorm:"column:transaction_ref"`
	MSISDN              string     `gorm:"column:msisdn"`
	RechargeType        string     `gorm:"column:recharge_type"`
	AmountKobo          int64      `gorm:"column:amount_kobo"`
	EventTimestamp      time.Time  `gorm:"column:event_timestamp"`
	RawPayload          []byte     `gorm:"column:raw_payload"`
	Status              string     `gorm:"column:status"`
	ProcessingError     string     `gorm:"column:processing_error"`
	PointsAwarded       int64      `gorm:"column:points_awarded"`
	DrawEntriesCreated  int        `gorm:"column:draw_entries_created"`
	SpinCreditsAwarded  int        `gorm:"column:spin_credits_awarded"`
	ProcessedAt         *time.Time `gorm:"column:processed_at"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime"`
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
			EventID:       existing.ID,
			MSISDN:        existing.MSISDN,
			PointsAwarded: existing.PointsAwarded,
			DrawEntries:   existing.DrawEntriesCreated,
			SpinCredits:   existing.SpinCreditsAwarded,
			IsDuplicate:   true,
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
	var ptsEarned int64
	var spinCreditsEarned int

	txErr := s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Row-level lock on wallet — prevents concurrent double-award.
		wallet, err := s.userRepo.GetWalletForUpdate(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("wallet lock failed: %w", err)
		}

		amountNaira := payload.Amount

		// ── Points calculation (same formula as processAwardTransaction) ──────
		baseRate := s.cfg.GetFloat("points_per_250_naira", 1.0) / 250.0
		tieredRate := s.getTieredRate(wallet.LifetimePoints)
		globalMult := s.cfg.GetFloat("global_points_multiplier", 1.0)
		effectiveRate := baseRate * tieredRate * globalMult
		ptsEarned = int64(math.Floor(amountNaira * effectiveRate))

		// ── Spin credit calculation via RechargeCounter accumulator ──────────
		// RechargeCounter accumulates kobo until it crosses spin_trigger_naira*100.
		// This ensures that e.g. two ₦500 recharges correctly award 1 spin
		// (not 0 spins each).
		spinTriggerKobo := s.cfg.GetInt64("spin_trigger_naira", 1000) * 100
		newCounter := wallet.RechargeCounter + amountKobo
		spinCreditsEarned = int(newCounter / spinTriggerKobo)
		newCounter = newCounter % spinTriggerKobo

		// ── Update wallet atomically ──────────────────────────────────────────
		if err := dbTx.Table("wallets").
			Where("user_id = ?", wallet.UserID).
			Updates(map[string]interface{}{
				"pulse_points":     gorm.Expr("pulse_points + ?", ptsEarned),
				"lifetime_points":  gorm.Expr("lifetime_points + ?", ptsEarned),
				"spin_credits":     gorm.Expr("spin_credits + ?", spinCreditsEarned),
				"recharge_counter": newCounter,
			}).Error; err != nil {
			return fmt.Errorf("wallet update failed: %w", err)
		}
		wallet.PulsePoints += ptsEarned
		wallet.LifetimePoints += ptsEarned
		wallet.SpinCredits += spinCreditsEarned
		wallet.RechargeCounter = newCounter

		// ── Immutable ledger entries ──────────────────────────────────────────
		ref := "MTN-" + payload.TransactionRef

		// 1. Recharge record
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

		// 2. Points award record
		if ptsEarned > 0 {
			meta, _ := json.Marshal(map[string]interface{}{
				"amount_kobo":    amountKobo,
				"recharge_type":  rechargeType,
				"effective_rate": effectiveRate,
				"source":         "mtn_push",
			})
			ptsTx := &entities.Transaction{
				ID:           uuid.New(),
				UserID:       user.ID,
				PhoneNumber:  phone,
				Type:         entities.TxTypePointsAward,
				PointsDelta:  ptsEarned,
				BalanceAfter: wallet.PulsePoints,
				Reference:    ref + "_pts",
				Metadata:     meta,
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
				PhoneNumber: phone,
				Type:        entities.TxTypeSpinCreditAward,
				SpinDelta:   spinCreditsEarned,
				Reference:   ref + "_spin",
				CreatedAt:   time.Now(),
			}
			if err := s.txRepo.SaveTx(ctx, dbTx, spinTx); err != nil {
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

		// 7a. Draw entries — 1 entry per Pulse Point earned (configurable).
		if ptsEarned > 0 {
			entriesPerPoint := s.cfg.GetInt("draw_entries_per_point", 1)
			totalEntries := int(ptsEarned) * entriesPerPoint
			if totalEntries > 0 {
				drawID, err := s.drawSvc.GetActiveDrawID(bgCtx)
				if err == nil {
					// AddEntry creates a single row with ticket_count = totalEntries.
					// This matches the RechargeMax pattern (one row, entries_count field).
					addErr := s.drawSvc.AddEntry(
						bgCtx,
						drawID,
						user.ID,
						phone,
						"recharge",   // entry_source
						amountKobo,
						totalEntries,
					)
					if addErr != nil {
						log.Printf("[MTN-PUSH] draw entry creation failed (non-fatal): %v", addErr)
					} else {
						drawEntriesCreated = totalEntries
					}
				}
				// No active draw is expected — entries will be imported at draw time.
			}
		}

		// 7b. SMS notification.
		if s.notifySvc != nil {
			amountNaira := int64(payload.Amount)
			msg := fmt.Sprintf(
				"Your MTN recharge of ₦%d has been processed. You earned %d Pulse Points",
				amountNaira, ptsEarned,
			)
			if spinCreditsEarned > 0 {
				msg += fmt.Sprintf(" and %d Spin Credit(s)", spinCreditsEarned)
			}
			msg += ". Keep recharging to win big!"
			s.notifySvc.SendSMS(bgCtx, phone, msg)
		}

		// 7c. Mark audit event as PROCESSED.
		now := time.Now()
		s.db.WithContext(bgCtx).Model(event).Updates(map[string]interface{}{
			"status":               "PROCESSED",
			"points_awarded":       ptsEarned,
			"draw_entries_created": drawEntriesCreated,
			"spin_credits_awarded": spinCreditsEarned,
			"processed_at":         &now,
		})
	})

	log.Printf("[MTN-PUSH] Processed %s: ₦%.2f %s -> +%d pts, +%d spins, +%d draw entries",
		phone, payload.Amount, rechargeType, ptsEarned, spinCreditsEarned, drawEntriesCreated)

	return &MTNPushResult{
		EventID:       event.ID,
		MSISDN:        phone,
		PointsAwarded: ptsEarned,
		DrawEntries:   drawEntriesCreated,
		SpinCredits:   spinCreditsEarned,
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
		ID:          uuid.New(),
		PhoneNumber: phone,
		ReferralCode: referralCode,
		Tier:        "bronze",
		CreatedAt:   time.Now(),
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

// getTieredRate returns the points multiplier based on lifetime points.
// Mirrors recharge_service.go getTieredRate.
func (s *MTNPushService) getTieredRate(lifetimePoints int64) float64 {
	switch {
	case lifetimePoints >= 5000:
		return 1.5
	case lifetimePoints >= 1500:
		return 1.25
	case lifetimePoints >= 500:
		return 1.1
	default:
		return 1.0
	}
}

// markEventFailed updates the audit row with the error message.
func (s *MTNPushService) markEventFailed(ctx context.Context, event *mtnPushEvent, err error) {
	s.db.WithContext(ctx).Model(event).Updates(map[string]interface{}{
		"status":           "FAILED",
		"processing_error": err.Error(),
	})
}

// normalisePhone converts any MTN phone format to 0XXXXXXXXXX.
// Handles: 2348012345678, +2348012345678, 08012345678, 8012345678.
func normalisePhone(raw string) string {
	// Strip all non-digit characters.
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
		return d // return as-is if unrecognised
	}
}

// calcStreak mirrors recharge_service.go calculateNewStreak.
func calcStreak(user *entities.User, expiryHours int) int {
	if user.LastRechargeAt == nil {
		return 1
	}
	deadline := user.LastRechargeAt.Add(time.Duration(expiryHours) * time.Hour)
	if time.Now().Before(deadline) {
		return user.StreakCount + 1
	}
	return 1
}

// ─── GetActiveDrawID helper on DrawService ────────────────────────────────────
// DrawService doesn't expose GetActiveDrawID yet — add it here as a thin
// wrapper so MTNPushService doesn't need to import gorm directly for this query.
// This is added to draw_service.go via an extension file to keep concerns separate.
// See: draw_service_active_draw.go
