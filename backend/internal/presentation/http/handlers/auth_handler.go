package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
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
	PhoneNumber string `json:"phone_number"`
	Purpose     string `json:"purpose"` // login | momo_link | prize_claim
}

type VerifyOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
	Code        string `json:"code"`
	Purpose     string `json:"purpose"`
}

func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PhoneNumber == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_number is required"})
		return
	}
	if req.Purpose == "" {
		req.Purpose = "login"
	}
	// Normalise to E.164 before any DB/OTP operation so every new account
	// is created with the canonical "+234…" format and no duplicates arise.
	req.PhoneNumber = normalizeE164NG(req.PhoneNumber)

	devCode, err := h.authSvc.SendOTP(r.Context(), req.PhoneNumber, req.Purpose)
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
	// Non-production only: include plaintext OTP in response so tests don't need log access
	if devCode != "" {
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
	req.PhoneNumber = normalizeE164NG(req.PhoneNumber)

	token, isNew, err := h.authSvc.VerifyOTP(r.Context(), req.PhoneNumber, req.Code, req.Purpose)
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
