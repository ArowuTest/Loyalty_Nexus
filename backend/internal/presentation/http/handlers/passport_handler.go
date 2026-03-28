package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// PassportHandler serves Digital Passport endpoints (spec §6).
type PassportHandler struct {
	passportSvc *services.PassportService
	cfg         *config.ConfigManager
}

func NewPassportHandler(ps *services.PassportService) *PassportHandler {
	return &PassportHandler{passportSvc: ps}
}

func (h *PassportHandler) WithConfig(cfg *config.ConfigManager) *PassportHandler {
	h.cfg = cfg
	return h
}

// ─── GET /api/v1/passport/banner-config (public) ─────────────────────────
// Returns the admin-configured banner message for the dashboard passport banner.
// This is a public endpoint (no auth required) so the frontend can fetch it
// before the user logs in.
func (h *PassportHandler) GetBannerConfig(w http.ResponseWriter, r *http.Request) {
	getStr := func(key, def string) string {
		if h.cfg == nil {
			return def
		}
		return h.cfg.GetString(key, def)
	}
	getBool := func(key string, def bool) bool {
		if h.cfg == nil {
			return def
		}
		return h.cfg.GetBool(key, def)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"banner_title":           getStr("passport_banner_title", "Your Digital Passport is ready"),
		"banner_subtitle":        getStr("passport_banner_subtitle", "Track your Pulse Points and streak right from your lock screen — no app needed."),
		"banner_cta_ios":         getStr("passport_banner_cta_ios", "Add to Apple Wallet"),
		"banner_cta_android":     getStr("passport_banner_cta_android", "Save to Google Wallet"),
		"banner_enabled":         getBool("passport_banner_enabled", true),
		// Wallet card messages
		"wallet_streak_expiry_message":  getStr("wallet_streak_expiry_message", "Streak expiring soon!"),
		"wallet_spin_ready_message":     getStr("wallet_spin_ready_message", "You have a free spin!"),
		"wallet_tier_upgrade_message":   getStr("wallet_tier_upgrade_message", "You've been promoted!"),
		"wallet_prize_won_message":      getStr("wallet_prize_won_message", "Prize waiting — open app"),
		"wallet_broadcast_enabled":      getBool("wallet_broadcast_enabled", false),
		"wallet_broadcast_label":        getStr("wallet_broadcast_label", "📢 ANNOUNCEMENT"),
		"wallet_broadcast_message":      getStr("wallet_broadcast_message", ""),
		"wallet_streak_expiry_enabled":  getBool("wallet_streak_expiry_enabled", true),
		"wallet_spin_ready_enabled":     getBool("wallet_spin_ready_enabled", true),
		"wallet_tier_upgrade_enabled":   getBool("wallet_tier_upgrade_enabled", true),
		"wallet_prize_won_enabled":      getBool("wallet_prize_won_enabled", true),
	})
}

// ─── GET /api/v1/passport ─────────────────────────────────────────────────

func (h *PassportHandler) GetPassport(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	passport, err := h.passportSvc.GetPassport(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load passport"})
		return
	}
	writeJSON(w, http.StatusOK, passport)
}

// ─── GET /api/v1/passport/badges ─────────────────────────────────────────

func (h *PassportHandler) GetBadges(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	passport, err := h.passportSvc.GetPassport(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load badges"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"earned":  passport.Badges,
		"tier":    passport.Tier,
		"streak":  passport.StreakCount,
	})
}

// ─── GET /api/v1/passport/qr ─────────────────────────────────────────────
// Returns a signed QR payload string. The mobile app renders this as a QR image.
// QR is valid for 5 minutes (spec §6.1).

func (h *PassportHandler) GetQR(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	qr, err := h.passportSvc.GenerateQRPayload(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "qr generation failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"qr_payload":  qr,
		"expires_in":  300, // seconds
		"format":      "base64url_json",
	})
}

// ─── POST /api/v1/passport/qr/verify ─────────────────────────────────────
// Partner merchant endpoint: verify a QR scan and return user_id if valid.

func (h *PassportHandler) VerifyQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		QRPayload string `json:"qr_payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.QRPayload == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "qr_payload required"})
		return
	}
	uid, err := h.passportSvc.VerifyQRPayload(body.QRPayload)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	// Log the scan event
	_ = h.passportSvc.LogPassportEvent(r.Context(), uid, services.PassportEventQRScanned,
		map[string]interface{}{"scanned_by": r.RemoteAddr})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":   true,
		"user_id": uid.String(),
	})
}

// ─── GET /api/v1/passport/pkpass ─────────────────────────────────────────
// Returns a signed .pkpass file for Apple Wallet (REQ-4.1).
// In production (APPLE_CERT_PEM + APPLE_CERT_KEY set) the pass is signed.
// In dev it is unsigned (iOS will reject but useful for testing).
// The isStreakExpiring flag is read from google_wallet_objects.streak_expiry_alert
// so that Ghost Nudge APNs pushes trigger the correct visual alert (REQ-4.4).

func (h *PassportHandler) DownloadPKPass(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	// Check whether a streak expiry alert is active for this user.
	// This is set by GhostNudgeWorker.pushStreakExpiryWalletPass.
	isStreakExpiring := h.passportSvc.IsStreakExpiryAlertActive(r.Context(), userID)

	pkpassBytes, serialNumber, err := h.passportSvc.BuildApplePKPassBytes(r.Context(), userID, isStreakExpiring)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pkpass build failed"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.pkpass")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=nexus_passport_%s.pkpass", serialNumber[:8]))
	w.WriteHeader(http.StatusOK)
	if _, writeErr := w.Write(pkpassBytes); writeErr != nil {
		log.Printf("[Passport] response Write error: %v", writeErr)
	}
}

// ─── GET /api/v1/passport/events ─────────────────────────────────────────
// Returns the user's passport event history (tier changes, badge earns, etc.)

func (h *PassportHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	events, err := h.passportSvc.GetPassportEvents(r.Context(), userID, 30)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load events"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": events})
}

// ─── GET /api/v1/passport/share ──────────────────────────────────────────
// Returns the shareable card data (spec §6.5)

func (h *PassportHandler) GetShareCard(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	card, err := h.passportSvc.GetShareableCard(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build share card"})
		return
	}
	writeJSON(w, http.StatusOK, card)
}

// ─── Helper ───────────────────────────────────────────────────────────────

func mustUserID(r *http.Request) uuid.UUID {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	id, _ := uuid.Parse(uid)
	return id
}
