package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"
)

type ClaimHandler struct {
	claimSvc *services.ClaimService
}

func NewClaimHandler(claimSvc *services.ClaimService) *ClaimHandler {
	return &ClaimHandler{claimSvc: claimSvc}
}

func (h *ClaimHandler) GetMyWins(w http.ResponseWriter, r *http.Request) {
	uidStr, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		jsonError(w, "invalid user id", http.StatusUnauthorized)
		return
	}

	wins, err := h.claimSvc.GetMyWins(r.Context(), userID)
	if err != nil {
		jsonError(w, "failed to get wins: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, wins)
}

func (h *ClaimHandler) ClaimPrize(w http.ResponseWriter, r *http.Request) {
	uidStr, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		jsonError(w, "invalid user id", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	claimID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid claim id", http.StatusBadRequest)
		return
	}

	var req services.ClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.claimSvc.ClaimPrize(r.Context(), userID, claimID, req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, result)
}

func (h *ClaimHandler) CheckMoMoAccount(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	if phone == "" {
		jsonError(w, "phone parameter is required", http.StatusBadRequest)
		return
	}

	hasAccount, name, err := h.claimSvc.CheckMoMoAccount(r.Context(), phone)
	if err != nil {
		jsonError(w, "failed to check account: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]interface{}{
		"has_momo_account": hasAccount,
		"account_name":     name,
	})
}
