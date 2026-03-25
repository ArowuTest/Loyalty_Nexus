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
	if err := h.authSvc.SendOTP(r.Context(), req.PhoneNumber, req.Purpose); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to send OTP"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "OTP sent"})
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
