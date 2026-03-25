package handlers

import (
	"encoding/json"
	"net/http"
	"loyalty-nexus/internal/application/services"
	"github.com/google/uuid"
)

type MoMoHandler struct {
	momoService *services.MoMoService
	authService *services.AuthService
}

func NewMoMoHandler(ms *services.MoMoService, as *services.AuthService) *MoMoHandler {
	return &MoMoHandler{
		momoService: ms,
		authService: as,
	}
}

func (h *MoMoHandler) RequestLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     uuid.UUID `json:"user_id"`
		MoMoNumber string    `json:"momo_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	// 1. Verify if account exists at MTN
	verified, msg, err := h.momoService.VerifyAccount(r.Context(), req.MoMoNumber)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if !verified {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": msg})
		return
	}

	// 2. Send OTP to confirm ownership
	// (Using login service helper for now)
	if err := h.authService.SendLoginOTP(r.Context(), req.MoMoNumber); err != nil {
		http.Error(w, "Failed to send confirmation code", 500)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{"message": "Verification code sent to MoMo number"})
}
