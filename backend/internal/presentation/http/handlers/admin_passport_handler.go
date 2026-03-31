package handlers

// admin_passport_handler.go — Admin endpoints for Passport & USSD monitoring.
// These methods are defined on AdminHandler (same struct as admin_handler.go)
// but kept in a separate file for clarity.
//
// Routes registered in main.go:
//   GET /api/v1/admin/passport/stats
//   GET /api/v1/admin/passport/nudge-log
//   GET /api/v1/admin/ussd/sessions
//
// Schema alignment (fixed 2026-03-31):
//   ghost_nudge_log columns: id, user_id, nudged_at, channel
//     (nudge_type, streak_count, sent_at, delivered do NOT exist — removed)
//   ussd_sessions columns: id, session_id, phone_number, menu_state,
//     input_buffer, pending_spin_id, expires_at, created_at, updated_at
//     (current_menu, started_at, last_active_at, is_active, step_count do NOT exist — removed)

import (
	"net/http"
	"strconv"
	"time"
)

// ─── GET /api/v1/admin/passport/stats ────────────────────────────────────────

// GetPassportStats returns aggregate passport statistics for the admin Passport panel.
func (h *AdminHandler) GetPassportStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Total unique users who have generated at least one passport event
	var totalPassports int64
	h.db.WithContext(ctx).
		Raw(`SELECT COUNT(DISTINCT user_id) FROM passport_events`).
		Scan(&totalPassports)

	// Apple Wallet downloads
	var appleDownloads int64
	h.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM passport_events WHERE event_type = 'apple_wallet_download'`).
		Scan(&appleDownloads)

	// Google Wallet saves
	var googleSaves int64
	h.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM passport_events WHERE event_type = 'google_wallet_save'`).
		Scan(&googleSaves)

	// QR scans today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	var qrScansToday int64
	h.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM passport_events WHERE event_type = 'qr_scanned' AND created_at >= ?`, today).
		Scan(&qrScansToday)

	// Tier breakdown
	type TierCount struct {
		Tier  string `json:"tier"`
		Count int64  `json:"count"`
	}
	var tierBreakdown []TierCount
	h.db.WithContext(ctx).
		Raw(`SELECT tier, COUNT(*) AS count FROM users
		     WHERE tier IS NOT NULL AND tier != ''
		     GROUP BY tier ORDER BY count DESC`).
		Scan(&tierBreakdown)
	if tierBreakdown == nil {
		tierBreakdown = []TierCount{}
	}

	// Top badge earners
	type BadgeEarner struct {
		UserID     string `json:"user_id"`
		Phone      string `json:"phone"`
		BadgeCount int64  `json:"badge_count"`
		Tier       string `json:"tier"`
	}
	var topEarners []BadgeEarner
	h.db.WithContext(ctx).Raw(`
		SELECT pe.user_id::text, u.phone_number AS phone,
		       COUNT(*) AS badge_count, u.tier
		FROM passport_events pe
		JOIN users u ON u.id = pe.user_id
		WHERE pe.event_type = 'badge_earned'
		GROUP BY pe.user_id, u.phone_number, u.tier
		ORDER BY badge_count DESC
		LIMIT 10
	`).Scan(&topEarners)
	if topEarners == nil {
		topEarners = []BadgeEarner{}
	}

	// ── Active installs: Apple (wallet_registrations, is_active=true) ──────────
	var activeAppleInstalls int64
	h.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM wallet_registrations WHERE platform = 'apple' AND is_active = true`).
		Scan(&activeAppleInstalls)

	// ── Active installs: Google (google_wallet_objects) ──────────────────────
	// google_wallet_objects has no state column; every row represents a live object.
	var activeGoogleInstalls int64
	h.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM google_wallet_objects`).
		Scan(&activeGoogleInstalls)

	// ── Device type breakdown from users table ───────────────────────────────
	type DeviceTypeCount struct {
		DeviceType string `json:"device_type"`
		Count      int64  `json:"count"`
	}
	var deviceBreakdown []DeviceTypeCount
	h.db.WithContext(ctx).
		Raw(`SELECT
		       CASE
		         WHEN device_type IS NULL OR device_type = '' THEN 'unknown'
		         ELSE LOWER(device_type)
		       END AS device_type,
		       COUNT(*) AS count
		     FROM users
		     GROUP BY 1
		     ORDER BY count DESC`).
		Scan(&deviceBreakdown)
	if deviceBreakdown == nil {
		deviceBreakdown = []DeviceTypeCount{}
	}

	// ── Pass removal rate (unregistered / total ever registered, Apple only) ──
	var totalEverApple int64
	h.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM wallet_registrations WHERE platform = 'apple'`).
		Scan(&totalEverApple)
	var removedApple int64
	h.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM wallet_registrations WHERE platform = 'apple' AND is_active = false`).
		Scan(&removedApple)
	removalRatePct := 0.0
	if totalEverApple > 0 {
		removalRatePct = float64(removedApple) / float64(totalEverApple) * 100
	}

	jsonOK(w, map[string]interface{}{
		"total_passports":        totalPassports,
		"apple_wallet_downloads": appleDownloads,
		"google_wallet_saves":    googleSaves,
		"qr_scans_today":         qrScansToday,
		"active_apple_installs":  activeAppleInstalls,
		"active_google_installs": activeGoogleInstalls,
		"total_active_installs":  activeAppleInstalls + activeGoogleInstalls,
		"removal_rate_pct":       removalRatePct,
		"device_breakdown":       deviceBreakdown,
		"tier_breakdown":         tierBreakdown,
		"top_badge_earners":      topEarners,
	})
}

// ─── GET /api/v1/admin/passport/nudge-log ────────────────────────────────────

// GetGhostNudgeLog returns the most recent Ghost Nudge log entries.
//
// Actual ghost_nudge_log schema (migrations 025, 036, 060):
//   id UUID, user_id UUID, nudged_at TIMESTAMPTZ, channel TEXT
//
// The legacy fields nudge_type, streak_count, sent_at, delivered were never
// added to the DB and have been removed from this query.
func (h *AdminHandler) GetGhostNudgeLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// NudgeLog reflects the real ghost_nudge_log columns only.
	type NudgeLog struct {
		ID          string    `json:"id"`
		UserID      string    `json:"user_id"`
		PhoneNumber string    `json:"phone_number"`
		NudgedAt    time.Time `json:"nudged_at"`
		Channel     string    `json:"channel"` // "sms" | "push" | "both"
	}

	var logs []NudgeLog
	h.db.WithContext(ctx).Raw(`
		SELECT gnl.id::text,
		       gnl.user_id::text,
		       u.phone_number,
		       gnl.nudged_at,
		       gnl.channel
		FROM ghost_nudge_log gnl
		JOIN users u ON u.id = gnl.user_id
		ORDER BY gnl.nudged_at DESC
		LIMIT ?
	`, limit).Scan(&logs)
	if logs == nil {
		logs = []NudgeLog{}
	}

	jsonOK(w, map[string]interface{}{
		"logs":  logs,
		"total": len(logs),
	})
}

// ─── GET /api/v1/admin/ussd/sessions ─────────────────────────────────────────

// GetUSSDSessions returns recent USSD session records for the admin monitor.
//
// Actual ussd_sessions schema (migrations 025, 056, 060):
//   id, session_id, phone_number, menu_state, input_buffer,
//   pending_spin_id, expires_at, created_at, updated_at
//
// The legacy fields current_menu, started_at, last_active_at, is_active,
// step_count were never added to the DB and have been removed from this query.
func (h *AdminHandler) GetUSSDSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// USSDSessionRow reflects the real ussd_sessions columns only.
	type USSDSessionRow struct {
		ID             string    `json:"id"`
		PhoneNumber    string    `json:"phone_number"`
		SessionID      string    `json:"session_id"`
		MenuState      string    `json:"menu_state"`      // renamed from current_menu
		InputBuffer    string    `json:"input_buffer"`
		PendingSpinID  *string   `json:"pending_spin_id,omitempty"`
		ExpiresAt      time.Time `json:"expires_at"`
		CreatedAt      time.Time `json:"created_at"`      // renamed from started_at
		UpdatedAt      time.Time `json:"updated_at"`      // renamed from last_active
	}

	var sessions []USSDSessionRow
	h.db.WithContext(ctx).Raw(`
		SELECT id::text,
		       phone_number,
		       session_id,
		       menu_state,
		       input_buffer,
		       pending_spin_id::text,
		       expires_at,
		       created_at,
		       updated_at
		FROM ussd_sessions
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit).Scan(&sessions)
	if sessions == nil {
		sessions = []USSDSessionRow{}
	}

	jsonOK(w, map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}
