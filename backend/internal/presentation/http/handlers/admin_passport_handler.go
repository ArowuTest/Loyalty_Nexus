package handlers

// admin_passport_handler.go — Admin endpoints for Passport & USSD monitoring.
// These methods are defined on AdminHandler (same struct as admin_handler.go)
// but kept in a separate file for clarity.
//
// Routes registered in main.go:
//   GET /api/v1/admin/passport/stats
//   GET /api/v1/admin/passport/nudge-log
//   GET /api/v1/admin/ussd/sessions

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

	jsonOK(w, map[string]interface{}{
		"total_passports":        totalPassports,
		"apple_wallet_downloads": appleDownloads,
		"google_wallet_saves":    googleSaves,
		"qr_scans_today":         qrScansToday,
		"tier_breakdown":         tierBreakdown,
		"top_badge_earners":      topEarners,
	})
}

// ─── GET /api/v1/admin/passport/nudge-log ────────────────────────────────────

// GetGhostNudgeLog returns the most recent Ghost Nudge SMS log entries.
func (h *AdminHandler) GetGhostNudgeLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	type NudgeLog struct {
		ID          string    `json:"id"`
		UserID      string    `json:"user_id"`
		PhoneNumber string    `json:"phone_number"`
		NudgeType   string    `json:"nudge_type"`
		StreakCount int       `json:"streak_count"`
		SentAt      time.Time `json:"sent_at"`
		Delivered   bool      `json:"delivered"`
	}

	var logs []NudgeLog
	h.db.WithContext(ctx).Raw(`
		SELECT gnl.id::text,
		       gnl.user_id::text,
		       u.phone_number,
		       gnl.nudge_type,
		       gnl.streak_count,
		       gnl.sent_at,
		       gnl.delivered
		FROM ghost_nudge_log gnl
		JOIN users u ON u.id = gnl.user_id
		ORDER BY gnl.sent_at DESC
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
func (h *AdminHandler) GetUSSDSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	type USSDSessionRow struct {
		ID          string    `json:"id"`
		PhoneNumber string    `json:"phone_number"`
		SessionID   string    `json:"session_id"`
		CurrentMenu string    `json:"current_menu"`
		StartedAt   time.Time `json:"started_at"`
		LastActive  time.Time `json:"last_active"`
		IsActive    bool      `json:"is_active"`
		StepCount   int       `json:"step_count"`
	}

	var sessions []USSDSessionRow
	h.db.WithContext(ctx).Raw(`
		SELECT id::text,
		       phone_number,
		       session_id,
		       current_menu,
		       started_at,
		       last_active_at AS last_active,
		       is_active,
		       step_count
		FROM ussd_sessions
		ORDER BY last_active_at DESC
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
