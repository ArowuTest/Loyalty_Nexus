package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"loyalty-nexus/internal/application/services"
)

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
	devCode, err := h.authSvc.SendOTP(r.Context(), req.PhoneNumber, req.Purpose)
	if err != nil {
		// Surface the actual error so the frontend can show a meaningful message.
		// Rate-limit errors get 429; everything else gets 400 (client-side issue)
		// or 500 for genuine server failures.
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
		"token":    token,
		"is_new_user": isNew,
	})
}

