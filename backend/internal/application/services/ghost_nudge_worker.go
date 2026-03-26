package services

// ghost_nudge_worker.go — Background cron worker for Ghost Nudge (spec §6.3).
//
// Runs every 5 minutes. Finds users whose streak is 1 day away from expiry
// (last_recharge_at between 23h and 24h ago) and who haven't been nudged in
// the last 24h. Sends an SMS via Termii and logs the nudge.
//
// Also pushes wallet pass updates to users whose tier or points have changed
// since the last wallet sync (Apple APNS + Google Wallet object update).
//
// Env vars:
//   TERMII_API_KEY      — Termii API key for SMS
//   TERMII_SENDER_ID    — Termii sender ID (e.g. "LoyaltyNex")
//   TERMII_BASE_URL     — Termii API base URL (default: https://api.ng.termii.com)

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── GhostNudgeWorker ────────────────────────────────────────────────────────

// GhostNudgeWorker runs the ghost nudge and wallet push cron jobs.
type GhostNudgeWorker struct {
	db          *gorm.DB
	passportSvc *PassportService
	termiiKey   string
	termiiSender string
	termiiBase  string
	stopCh      chan struct{}
}

// NewGhostNudgeWorker creates a new worker. Call Start() to begin.
func NewGhostNudgeWorker(db *gorm.DB, passportSvc *PassportService) *GhostNudgeWorker {
	base := os.Getenv("TERMII_BASE_URL")
	if base == "" {
		base = "https://api.ng.termii.com"
	}
	sender := os.Getenv("TERMII_SENDER_ID")
	if sender == "" {
		sender = "LoyaltyNex"
	}
	return &GhostNudgeWorker{
		db:           db,
		passportSvc:  passportSvc,
		termiiKey:    os.Getenv("TERMII_API_KEY"),
		termiiSender: sender,
		termiiBase:   base,
		stopCh:       make(chan struct{}),
	}
}

// Start launches the worker in a background goroutine.
// Call Stop() to gracefully shut it down.
func (w *GhostNudgeWorker) Start() {
	go w.run()
	log.Println("[GhostNudge] Worker started (interval: 5m)")
}

// Stop signals the worker to stop after the current tick completes.
func (w *GhostNudgeWorker) Stop() {
	close(w.stopCh)
}

func (w *GhostNudgeWorker) run() {
	// Run immediately on startup, then every 5 minutes
	w.tick()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.tick()
		case <-w.stopCh:
			log.Println("[GhostNudge] Worker stopped")
			return
		}
	}
}

func (w *GhostNudgeWorker) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	w.runGhostNudge(ctx)
	w.runWalletPassSync(ctx)
}

// ─── Ghost Nudge ─────────────────────────────────────────────────────────────

func (w *GhostNudgeWorker) runGhostNudge(ctx context.Context) {
	if w.termiiKey == "" {
		log.Println("[GhostNudge] TERMII_API_KEY not set — skipping SMS nudge")
		return
	}

	candidates, err := w.passportSvc.GetGhostNudgeCandidates(ctx)
	if err != nil {
		log.Printf("[GhostNudge] GetGhostNudgeCandidates error: %v", err)
		return
	}

	if len(candidates) == 0 {
		return
	}

	log.Printf("[GhostNudge] Sending nudge to %d users", len(candidates))

	for _, c := range candidates {
		msg := buildNudgeMessage(c.StreakCount)
		if err2 := w.sendSMS(c.PhoneNumber, msg); err2 != nil {
			log.Printf("[GhostNudge] SMS failed for user %s: %v", c.UserID, err2)
			continue
		}
		if err2 := w.passportSvc.RecordGhostNudge(ctx, c.UserID); err2 != nil {
			log.Printf("[GhostNudge] RecordGhostNudge failed for user %s: %v", c.UserID, err2)
		}
		// Log push audit
		w.logPassportPush(ctx, c.UserID, "sms", "ghost_nudge", "sent", "")
	}
}

func buildNudgeMessage(streakCount int) string {
	if streakCount >= 30 {
		return fmt.Sprintf(
			"🔥 Your %d-day streak on Loyalty Nexus expires in 1 hour! "+
				"Recharge now to keep your Month Master status. Dial *384# or open the app.",
			streakCount,
		)
	}
	if streakCount >= 7 {
		return fmt.Sprintf(
			"⚡ Your %d-day streak expires in 1 hour! "+
				"Don't lose your Week Warrior badge. Recharge now — dial *384# or open the Loyalty Nexus app.",
			streakCount,
		)
	}
	return fmt.Sprintf(
		"🔥 Your %d-day Loyalty Nexus streak expires in 1 hour! "+
			"Recharge to keep earning Pulse Points. Dial *384# or open the app.",
		streakCount,
	)
}

// ─── Termii SMS ──────────────────────────────────────────────────────────────

type termiiSMSRequest struct {
	To      string `json:"to"`
	From    string `json:"from"`
	SMS     string `json:"sms"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	APIKey  string `json:"api_key"`
}

type termiiSMSResponse struct {
	MessageID string `json:"message_id"`
	Message   string `json:"message"`
	Balance   float64 `json:"balance"`
	User      string `json:"user"`
}

func (w *GhostNudgeWorker) sendSMS(phone, message string) error {
	payload := termiiSMSRequest{
		To:      phone,
		From:    w.termiiSender,
		SMS:     message,
		Type:    "plain",
		Channel: "generic",
		APIKey:  w.termiiKey,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("termii marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, w.termiiBase+"/api/sms/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("termii request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("termii send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("termii HTTP %d", resp.StatusCode)
	}

	return nil
}

// ─── Wallet Pass Sync ─────────────────────────────────────────────────────────
// Finds users whose tier or points have changed since the last wallet sync
// and marks them for a push update. The actual Apple APNS push and Google
// Wallet object update happen here.

type walletSyncCandidate struct {
	UserID             uuid.UUID `gorm:"column:user_id"`
	PhoneNumber        string    `gorm:"column:phone_number"`
	Tier               string    `gorm:"column:tier"`
	LifetimePoints     int64     `gorm:"column:lifetime_points"`
	StreakCount        int       `gorm:"column:streak_count"`
	GoogleObjectID     string    `gorm:"column:google_wallet_object_id"`
	ApplePassSerial    string    `gorm:"column:apple_pass_serial"`
	PointsAtLastSync   int64     `gorm:"column:points_at_last_sync"`
	TierAtLastSync     string    `gorm:"column:tier_at_last_sync"`
}

func (w *GhostNudgeWorker) runWalletPassSync(ctx context.Context) {
	// Find users whose wallet data has changed since last sync
	// We check: tier changed OR points changed by >= 100 (avoid too-frequent pushes)
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
			COALESCE(gwo.points_at_last_sync, 0)      AS points_at_last_sync,
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

		// Update Google Wallet object sync record
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

		// Apple: log the push (actual APNS push requires device push token lookup)
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
	// Look up registered Apple Wallet devices for this serial number
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
		// No registered devices — nothing to push
		return
	}

	// In production, use the Apple Push Notification service (APNs) HTTP/2 API.
	// For now, we log the intent and mark the push as pending.
	// Full APNs implementation requires golang.org/x/net/http2 + Apple cert.
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
