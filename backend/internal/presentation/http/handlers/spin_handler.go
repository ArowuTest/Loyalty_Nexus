package handlers

import (
	"net/http"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"

	"github.com/google/uuid"
)

type SpinHandler struct {
	spinSvc *services.SpinService
}

func NewSpinHandler(ss *services.SpinService) *SpinHandler {
	return &SpinHandler{spinSvc: ss}
}

func (h *SpinHandler) GetWheelConfig(w http.ResponseWriter, r *http.Request) {
	payload, err := h.spinSvc.GetWheelConfig(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load wheel"})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *SpinHandler) Play(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, err := uuid.Parse(uid)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user"})
		return
	}

	outcome, err := h.spinSvc.PlaySpin(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, outcome)
}

func (h *SpinHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, err := uuid.Parse(uid)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user"})
		return
	}
	results, err := h.spinSvc.GetSpinHistory(r.Context(), userID, 20, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load history"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"history": results})
}

// CheckEligibility returns the user's current spin eligibility state:
// spin credits available, daily cap from their recharge tier, spins used today,
// and a nudge toward the next tier if they have hit their daily cap.
//
// GET /api/v1/spin/eligibility
//
// Response shape (mirrors services.SpinEligibility):
//
//	{
//	  "eligible":           true,
//	  "spin_credits":       3,
//	  "spins_used_today":   1,
//	  "max_spins_today":    3,
//	  "message":            "You have 2 spins left today!",
//	  "trigger_naira":      1000,
//	  "next_tier_name":     "Gold",          // only when cap is reached
//	  "next_tier_min_amount": 1000000,       // kobo
//	  "amount_to_next_tier":  500000,        // kobo remaining to unlock next tier
//	  "next_tier_spins":    3
//	}
func (h *SpinHandler) CheckEligibility(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, err := uuid.Parse(uid)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user"})
		return
	}
	eligibility, err := h.spinSvc.CheckEligibility(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, eligibility)
}
