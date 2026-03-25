package handlers

import (
	"encoding/json"
	"net/http"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/presentation/http/middleware"

	"github.com/google/uuid"
)

type UserHandler struct {
	userRepo    repositories.UserRepository
	hlrSvc      *services.HLRService
	momoAdapter external.MoMoPayer
	fulfillSvc  *services.PrizeFulfillmentService
}

func NewUserHandler(ur repositories.UserRepository, hs *services.HLRService, ma external.MoMoPayer, fs *services.PrizeFulfillmentService) *UserHandler {
	return &UserHandler{userRepo: ur, hlrSvc: hs, momoAdapter: ma, fulfillSvc: fs}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)
	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)
	wallet, err := h.userRepo.GetWallet(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "wallet not found"})
		return
	}
	writeJSON(w, http.StatusOK, wallet)
}

func (h *UserHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []interface{}{})
}

type MoMoLinkRequest struct {
	MoMoNumber string `json:"momo_number"`
}

func (h *UserHandler) RequestMoMoLink(w http.ResponseWriter, r *http.Request) {
	var req MoMoLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MoMoNumber == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "momo_number is required"})
		return
	}

	name, valid, err := h.momoAdapter.VerifyAccount(r.Context(), req.MoMoNumber)
	if err != nil || !valid {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "MoMo account not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"verified":     true,
		"account_name": name,
		"momo_number":  req.MoMoNumber,
		"message":      "MoMo account verified. It will be linked to your profile.",
	})
}

func (h *UserHandler) VerifyMoMo(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)

	var req MoMoLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.userRepo.UpdateMoMo(r.Context(), userID, req.MoMoNumber, true); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update MoMo"})
		return
	}

	// Release any held MoMo prizes
	go h.fulfillSvc.ReleaseMoMoHeldPrizes(r.Context(), userID)

	writeJSON(w, http.StatusOK, map[string]string{"message": "MoMo number linked successfully"})
}

func (h *UserHandler) GetPassportURLs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"apple":  "#",
		"google": "#",
		"message": "Wallet integration coming soon",
	})
}
