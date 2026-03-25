package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// PassportHandler serves Digital Passport endpoints (spec §6).
type PassportHandler struct {
	passportSvc *services.PassportService
}

func NewPassportHandler(ps *services.PassportService) *PassportHandler {
	return &PassportHandler{passportSvc: ps}
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
// Returns a .pkpass file for Apple Wallet (spec §6.2).
// NOTE: In production, pass.json must be signed with Apple cert.
// This endpoint returns the unsigned manifest for dev/staging.

func (h *PassportHandler) DownloadPKPass(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	passData, err := h.passportSvc.BuildPKPass(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pkpass build failed"})
		return
	}

	passJSON, err := json.MarshalIndent(passData, "", "  ")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pkpass marshal failed"})
		return
	}

	// Build a minimal .pkpass zip (pass.json only; icon/logo need CDN assets in prod)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	pf, createErr := zw.Create("pass.json")
	if createErr != nil {
		log.Printf("[Passport] zip Create error: %v", createErr)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, writeErr := pf.Write(passJSON); writeErr != nil {
		log.Printf("[Passport] zip Write error: %v", writeErr)
	}
	zw.Close()

	w.Header().Set("Content-Type", "application/vnd.apple.pkpass")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=nexus_passport_%s.pkpass", userID.String()[:8]))
	w.WriteHeader(http.StatusOK)
	if _, writeErr := w.Write(buf.Bytes()); writeErr != nil {
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
