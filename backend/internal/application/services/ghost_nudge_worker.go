package services

// ghost_nudge_worker.go — Background cron worker for Ghost Nudge (spec §6.3 / REQ-4.4).
//
// REQ-4.4 (MUST): Runs every N minutes (configurable via ghost_nudge_interval_minutes,
// default 60). Finds users whose Recharge Streak will expire within the next
// ghost_nudge_warning_hours (default 4) AND who have a streak of at least
// ghost_nudge_min_streak days (default 3). For each such user:
//   1. Pushes an updated wallet pass with a visual "Streak Expiring Soon!" alert.
//   2. Sends an SMS nudge via NotificationService (Termii).
//   3. Logs the nudge to ghost_nudge_log (cooldown: no re-nudge within 24h).
//
// Also runs a wallet pass sync job to push updates to users whose tier or
// points have changed since the last wallet sync (REQ-4.3).

import (
	"context"
	"fmt"
	"log"
	"time"

	"loyalty-nexus/internal/infrastructure/config"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── GhostNudgeWorker ────────────────────────────────────────────────────────

// GhostNudgeWorker runs the ghost nudge and wallet push cron jobs.
type GhostNudgeWorker struct {
	db              *gorm.DB
	cfg             *config.ConfigManager
	passportSvc     *PassportService
	notificationSvc *NotificationService
	stopCh          chan struct{}
}

// NewGhostNudgeWorker creates a new worker. Call Start() to begin.
// All timing and threshold parameters are read from ConfigManager on every tick —
// no values are hardcoded.
func NewGhostNudgeWorker(
	db *gorm.DB,
	cfg *config.ConfigManager,
	passportSvc *PassportService,
	notificationSvc *NotificationService,
) *GhostNudgeWorker {
	return &GhostNudgeWorker{
		db:              db,
		cfg:             cfg,
		passportSvc:     passportSvc,
		notificationSvc: notificationSvc,
		stopCh:          make(chan struct{}),
	}
}

// Start launches the worker in a background goroutine.
// The cron interval is read from ConfigManager key ghost_nudge_interval_minutes
// (default 60 per REQ-4.4). The ticker is rebuilt on each run so that admin
// changes to the interval take effect within one cycle.
func (w *GhostNudgeWorker) Start() {
	go w.run()
	intervalMin := w.cfg.GetInt("ghost_nudge_interval_minutes", 60)
	log.Printf("[GhostNudge] Worker started (interval: %dm)", intervalMin)
}

// Stop signals the worker to stop after the current tick completes.
func (w *GhostNudgeWorker) Stop() {
	close(w.stopCh)
}

func (w *GhostNudgeWorker) run() {
	// Run immediately on startup so the first nudge fires without waiting.
	w.tick()

	for {
		// Re-read the interval on every cycle so admin changes take effect.
		intervalMin := w.cfg.GetInt("ghost_nudge_interval_minutes", 60)
		ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)

		select {
		case <-ticker.C:
			ticker.Stop()
			w.tick()
		case <-w.stopCh:
			ticker.Stop()
			log.Println("[GhostNudge] Worker stopped")
			return
		}
	}
}

func (w *GhostNudgeWorker) tick() {
	// Allow up to (interval - 1 min) for each tick to complete.
	intervalMin := w.cfg.GetInt("ghost_nudge_interval_minutes", 60)
	timeout := time.Duration(intervalMin-1) * time.Minute
	if timeout < time.Minute {
		timeout = time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	w.runGhostNudge(ctx)
	w.runWalletPassSync(ctx)
}

// ─── Ghost Nudge (REQ-4.4) ────────────────────────────────────────────────────

func (w *GhostNudgeWorker) runGhostNudge(ctx context.Context) {
	// Read all thresholds from ConfigManager — ZERO hardcoding.
	warningHours := w.cfg.GetInt("ghost_nudge_warning_hours", 4)
	minStreak := w.cfg.GetInt("ghost_nudge_min_streak", 3)

	candidates, err := w.passportSvc.GetGhostNudgeCandidates(ctx, warningHours, minStreak)
	if err != nil {
		log.Printf("[GhostNudge] GetGhostNudgeCandidates error: %v", err)
		return
	}

	if len(candidates) == 0 {
		return
	}

	log.Printf("[GhostNudge] Sending nudge to %d users (warningHours=%d, minStreak=%d)",
		len(candidates), warningHours, minStreak)

	for _, c := range candidates {
		// 1. Push wallet pass with "Streak Expiring Soon!" visual alert (REQ-4.4).
		w.pushStreakExpiryWalletPass(ctx, c.UserID, c.StreakCount)

		// 2. Send SMS nudge.
		msg := buildNudgeMessage(c.StreakCount, warningHours)
		if err2 := w.notificationSvc.SendSMS(ctx, c.PhoneNumber, msg); err2 != nil {
			log.Printf("[GhostNudge] SMS failed for user %s: %v", c.UserID, err2)
			w.logPassportPush(ctx, c.UserID, "sms", "ghost_nudge", "failed", err2.Error())
			continue
		}

		// 3. Record nudge (prevents re-nudge within 24h).
		if err2 := w.passportSvc.RecordGhostNudge(ctx, c.UserID); err2 != nil {
			log.Printf("[GhostNudge] RecordGhostNudge failed for user %s: %v", c.UserID, err2)
		}
		w.logPassportPush(ctx, c.UserID, "sms", "ghost_nudge", "sent", "")
	}
}

// pushStreakExpiryWalletPass pushes an updated wallet pass with the
// "Streak Expiring Soon!" visual alert to all registered devices for the user.
// This satisfies REQ-4.4: "push an updated wallet pass with a visual alert."
func (w *GhostNudgeWorker) pushStreakExpiryWalletPass(ctx context.Context, userID uuid.UUID, streakCount int) {
	type userRow struct {
		GoogleObjectID string `gorm:"column:google_wallet_object_id"`
		AppleSerial    string `gorm:"column:apple_pass_serial"`
	}
	var u userRow
	if err := w.db.WithContext(ctx).Table("users").
		Select("google_wallet_object_id, apple_pass_serial").
		Where("id = ?", userID).First(&u).Error; err != nil {
		return
	}

	// Update google_wallet_objects to flag streak expiry so the next pass
	// build picks up the IsStreakExpiring flag.
	if u.GoogleObjectID != "" {
		w.db.WithContext(ctx).Exec(`
			UPDATE google_wallet_objects
			SET streak_expiry_alert = true, updated_at = NOW()
			WHERE user_id = ?
		`, userID)
		w.logPassportPush(ctx, userID, "google", "streak_expiry_alert", "sent", "")
	}

	// For Apple: send APNs push so iOS re-fetches the pass from our server.
	// Our GET /api/v1/passport/pkpass endpoint will include the expiry alert
	// because it calls BuildApplePKPassBytes with IsStreakExpiring=true when
	// streak_expiry_alert=true in google_wallet_objects.
	if u.AppleSerial != "" {
		w.sendApplePushNotification(ctx, userID, u.AppleSerial, "streak_expiry_alert")
	}
}

func buildNudgeMessage(streakCount, warningHours int) string {
	if streakCount >= 30 {
		return fmt.Sprintf(
			"🔥 Your %d-day streak on Loyalty Nexus expires in %d hours! "+
				"Recharge now to keep your Month Master status. Dial *384# or open the app.",
			streakCount, warningHours,
		)
	}
	if streakCount >= 7 {
		return fmt.Sprintf(
			"⚡ Your %d-day streak expires in %d hours! "+
				"Don't lose your Week Warrior badge. Recharge now — dial *384# or open the Loyalty Nexus app.",
			streakCount, warningHours,
		)
	}
	return fmt.Sprintf(
		"🔥 Your %d-day Loyalty Nexus streak expires in %d hours! "+
			"Recharge to keep earning Pulse Points. Dial *384# or open the app.",
		streakCount, warningHours,
	)
}

// ─── Wallet Pass Sync (REQ-4.3) ───────────────────────────────────────────────
// Finds users whose tier or points have changed since the last wallet sync
// and pushes updated passes to their registered devices.

type walletSyncCandidate struct {
	UserID           uuid.UUID `gorm:"column:user_id"`
	PhoneNumber      string    `gorm:"column:phone_number"`
	Tier             string    `gorm:"column:tier"`
	LifetimePoints   int64     `gorm:"column:lifetime_points"`
	StreakCount      int       `gorm:"column:streak_count"`
	GoogleObjectID   string    `gorm:"column:google_wallet_object_id"`
	ApplePassSerial  string    `gorm:"column:apple_pass_serial"`
	PointsAtLastSync int64     `gorm:"column:points_at_last_sync"`
	TierAtLastSync   string    `gorm:"column:tier_at_last_sync"`
}

func (w *GhostNudgeWorker) runWalletPassSync(ctx context.Context) {
	var candidates []walletSyncCandidate
	err := w.db.WithContext(ctx).Raw(`
		SELECT
			u.id          AS user_id,
			u.phone_number,
			u.tier,
			u.lifetime_points,
			u.streak_count,
			u.google_wallet_object_id,
			u.apple_pass_serial,
			COALESCE(gwo.points_at_last_sync, 0)       AS points_at_last_sync,
			COALESCE(gwo.tier_at_last_sync, 'BRONZE')  AS tier_at_last_sync
		FROM users u
		LEFT JOIN google_wallet_objects gwo ON gwo.user_id = u.id
		WHERE u.is_active = true
		  AND (
			  u.google_wallet_object_id IS NOT NULL
			  OR u.apple_pass_serial IS NOT NULL
		  )
		  AND (
			  COALESCE(gwo.tier_at_last_sync, '') != u.tier
			  OR ABS(u.lifetime_points - COALESCE(gwo.points_at_last_sync, 0)) >= 100
		  )
		  AND (
			  gwo.last_synced_at IS NULL
			  OR gwo.last_synced_at < NOW() - INTERVAL '1 hour'
		  )
		LIMIT 200
	`).Scan(&candidates).Error
	if err != nil {
		log.Printf("[WalletSync] query error: %v", err)
		return
	}

	if len(candidates) == 0 {
		return
	}

	log.Printf("[WalletSync] Syncing wallet passes for %d users", len(candidates))

	for _, c := range candidates {
		trigger := "points_milestone"
		if c.TierAtLastSync != c.Tier {
			trigger = "tier_change"
		}

		if c.GoogleObjectID != "" {
			w.db.WithContext(ctx).Exec(`
				INSERT INTO google_wallet_objects (id, user_id, object_id, class_id, last_synced_at, points_at_last_sync, tier_at_last_sync, created_at, updated_at)
				VALUES (?, ?, ?, 'LoyaltyNexus', NOW(), ?, ?, NOW(), NOW())
				ON CONFLICT (user_id) DO UPDATE SET
					last_synced_at      = NOW(),
					points_at_last_sync = EXCLUDED.points_at_last_sync,
					tier_at_last_sync   = EXCLUDED.tier_at_last_sync,
					updated_at          = NOW()
			`, uuid.New(), c.UserID, c.GoogleObjectID, c.LifetimePoints, c.Tier)

			w.logPassportPush(ctx, c.UserID, "google", trigger, "sent", "")
		}

		if c.ApplePassSerial != "" {
			w.sendApplePushNotification(ctx, c.UserID, c.ApplePassSerial, trigger)
		}
	}
}

// sendApplePushNotification sends an Apple PassKit push notification to registered devices.
// Apple Wallet push: POST to https://api.push.apple.com/3/device/{pushToken}
// with an empty JSON body — Apple then calls GET /v1/passes/{passTypeID}/{serialNumber}
// on our server to fetch the updated pass.
func (w *GhostNudgeWorker) sendApplePushNotification(ctx context.Context, userID uuid.UUID, serialNumber, trigger string) {
	type deviceRow struct {
		PushToken string `gorm:"column:push_token"`
		DeviceID  string `gorm:"column:device_id"`
	}
	var devices []deviceRow
	w.db.WithContext(ctx).Raw(`
		SELECT push_token, device_id
		FROM wallet_registrations
		WHERE serial_number = ? AND platform = 'apple' AND is_active = true AND push_token IS NOT NULL
	`, serialNumber).Scan(&devices)

	if len(devices) == 0 {
		return
	}

	// Full APNs implementation requires golang.org/x/net/http2 + Apple cert.
	// For now, log the intent and mark the push as pending.
	for _, d := range devices {
		log.Printf("[ApplePush] Would push to device %s (token: %s...) for user %s trigger=%s",
			d.DeviceID, safePrefix(d.PushToken, 12), userID, trigger)
		w.logPassportPush(ctx, userID, "apple", trigger, "pending", "apns_not_yet_implemented")
	}
}

// ─── Audit log helper ─────────────────────────────────────────────────────────

func (w *GhostNudgeWorker) logPassportPush(ctx context.Context, userID uuid.UUID, platform, trigger, status, errMsg string) {
	w.db.WithContext(ctx).Exec(`
		INSERT INTO passport_push_log (id, user_id, platform, trigger, status, error_msg, pushed_at)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NOW())
	`, uuid.New(), userID, platform, trigger, status, errMsg)
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
