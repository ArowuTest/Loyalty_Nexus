package handlers

import (
	"encoding/json"
	"net/http"

	"loyalty-nexus/internal/application/services"
	"context"

	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/pkg/safe"
	"loyalty-nexus/internal/presentation/http/middleware"

	"github.com/google/uuid"
)

type UserHandler struct {
	userRepo      repositories.UserRepository
	hlrSvc        *services.HLRService
	momoAdapter   external.MoMoPayer
	fulfillSvc    *services.PrizeFulfillmentService
	bonusPulseSvc *services.BonusPulseService
}

func NewUserHandler(ur repositories.UserRepository, hs *services.HLRService, ma external.MoMoPayer, fs *services.PrizeFulfillmentService) *UserHandler {
	return &UserHandler{userRepo: ur, hlrSvc: hs, momoAdapter: ma, fulfillSvc: fs}
}

// WithBonusPulseService attaches the BonusPulseService so the user-facing
// bonus awards endpoint can query the audit table.
func (h *UserHandler) WithBonusPulseService(svc *services.BonusPulseService) *UserHandler {
	h.bonusPulseSvc = svc
	return h
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
	safe.Go(func() {
		h.fulfillSvc.ReleaseMoMoHeldPrizes(context.Background(), userID)
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "MoMo number linked successfully"})
}

func (h *UserHandler) GetPassportURLs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"apple":  "#",
		"google": "#",
		"message": "Wallet integration coming soon",
	})
}

// GetBonusPulseAwards handles GET /api/v1/user/bonus-pulse
// Returns the user's bonus Pulse Point award history (most recent first).
func (h *UserHandler) GetBonusPulseAwards(w http.ResponseWriter, r *http.Request) {
	if h.bonusPulseSvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"total_bonus": 0, "awards": []interface{}{}})
		return
	}
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)

	total, err := h.bonusPulseSvc.GetUserBonusTotal(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not fetch bonus total"})
		return
	}
	awards, err := h.bonusPulseSvc.GetUserAwards(r.Context(), userID, 20)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not fetch awards"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_bonus": total,
		"awards":      awards,
	})
}

// UpdateProfileState handles POST /api/v1/user/profile/state
// REQ-1.5: User sets their Nigerian state for Regional Wars team assignment
func (h *UserHandler) UpdateProfileState(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)
	var req struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.State == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "state is required"})
		return
	}
	if err := h.userRepo.UpdateState(r.Context(), userID, req.State); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
