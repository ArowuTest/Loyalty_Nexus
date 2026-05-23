package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"loyalty-nexus/internal/application/services"
)

// normalizeE164NG converts any Nigerian phone format to canonical E.164 (+234XXXXXXXXX).
//
// Accepted inputs (all map to the same output, e.g. +2348027000000):
//   - +2348027000000   → +2348027000000  (already E.164, returned as-is)
//   - 2348027000000    → +2348027000000  (international without +)
//   - 08027000000      → +2348027000000  (local 0XX format)
//
// Any other format is returned stripped of spaces and dashes but otherwise unchanged.
func normalizeE164NG(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(p, " ", ""), "-", ""))
	if strings.HasPrefix(p, "+234") {
		// Already canonical E.164
		return p
	}
	if strings.HasPrefix(p, "234") && len(p) >= 13 {
		// International without leading +
		return "+" + p
	}
	if strings.HasPrefix(p, "0") && len(p) == 11 {
		// Local Nigerian format: 08XXXXXXXXX → +2348XXXXXXXXX
		return "+234" + p[1:]
	}
	// Unknown format — return as-is so downstream phoneVariants still handles it
	return p
}

type AuthHandler struct {
	authSvc *services.AuthService
}

func NewAuthHandler(as *services.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: as}
}

type SendOTPRequest struct {
	PhoneNumber    string `json:"phone_number"`
	PhoneLegacy    string `json:"phone"` // accepted for backwards-compat with older mobile clients
	Purpose        string `json:"purpose"` // login | momo_link | prize_claim
}

// effectivePhone returns phone_number, falling back to legacy "phone" field.
func (r SendOTPRequest) effectivePhone() string {
	if r.PhoneNumber != "" {
		return r.PhoneNumber
	}
	return r.PhoneLegacy
}

type VerifyOTPRequest struct {
	PhoneNumber    string `json:"phone_number"`
	PhoneLegacy    string `json:"phone"` // accepted for backwards-compat with older mobile clients
	Code           string `json:"code"`
	Purpose        string `json:"purpose"`
}

func (r VerifyOTPRequest) effectivePhone() string {
	if r.PhoneNumber != "" {
		return r.PhoneNumber
	}
	return r.PhoneLegacy
}

func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.effectivePhone() == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_number is required"})
		return
	}
	if req.Purpose == "" {
		req.Purpose = "login"
	}
	// Normalise to E.164 before any DB/OTP operation so every new account
	// is created with the canonical "+234…" format and no duplicates arise.
	req.PhoneNumber = normalizeE164NG(req.effectivePhone())

	devCode, err := h.authSvc.SendOTP(r.Context(), req.effectivePhone(), req.Purpose)
	if err != nil {
		statusCode := http.StatusBadRequest
		msg := err.Error()
		if errors.Is(err, services.ErrRateLimitExceeded) {
			statusCode = http.StatusTooManyRequests
		} else if msg == "failed to save OTP" || msg == "failed to generate OTP" || msg == "failed to check rate limit" {
			statusCode = http.StatusInternalServerError
			msg = "failed to send OTP"
		}
		writeJSON(w, statusCode, map[string]string{"error": msg})
		return
	}
	resp := map[string]interface{}{"message": "OTP sent"}
	// Only expose plaintext OTP when ENVIRONMENT is explicitly "development" or "staging".
	// Absence of the env var, or any other value (including "production"), means production
	// mode — the OTP is never returned in the response body.
	// BUG-001 fix: previously this leaked whenever devCode was non-empty regardless of env.
	env := strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT")))
	isNonProd := env == "development" || env == "staging"
	if devCode != "" && isNonProd {
		resp["dev_otp"] = devCode
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.Purpose == "" {
		req.Purpose = "login"
	}
	// Normalise to E.164 here too — must match the format used in SendOTP so the
	// OTP lookup (FindLatestPendingOTP WHERE phone_number = ?) hits the right row.
	req.PhoneNumber = normalizeE164NG(req.effectivePhone())

	token, isNew, err := h.authSvc.VerifyOTP(r.Context(), req.effectivePhone(), req.Code, req.Purpose)
	if err != nil {
		statusCode := http.StatusUnauthorized
		if errors.Is(err, services.ErrOTPExpired) {
			statusCode = http.StatusGone
		}
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":       token,
		"is_new_user": isNew,
	})
}
