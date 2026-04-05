package handlers

import (
	"encoding/json"
	"net/http"
	"loyalty-nexus/internal/application/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(as *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: as}
}

func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MSISDN string `json:"msisdn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	if err := h.authService.SendLoginOTP(r.Context(), req.MSISDN); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{"message": "OTP Sent"})
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MSISDN       string `json:"msisdn"`
		Code         string `json:"code"`
		ReferralCode string `json:"referral_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	token, err := h.authService.VerifyLogin(r.Context(), req.MSISDN, req.Code, req.ReferralCode)
	if err != nil {
		http.Error(w, err.Error(), 401)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
