package handlers

import (
	"encoding/json"
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

// Play executes a spin for the authenticated user.
// Extracts phone from JWT context (set by AuthMiddleware → ContextPhone).
// Falls back to body-provided MSISDN for guest support.
//
// POST /api/v1/spin/play
func (h *SpinHandler) Play(w http.ResponseWriter, r *http.Request) {
	// Extract phone from context (preferred — from JWT)
	phone, _ := r.Context().Value(middleware.ContextPhone).(string)

	// Also extract userID — needed for wallet/ledger operations inside PlaySpin
	uidStr, _ := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uidStr)

	// If no phone from context, check request body (guest recharge flow)
	if phone == "" {
		var body struct {
			MSISDN string `json:"msisdn"`
			Phone  string `json:"phone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			if body.MSISDN != "" {
				phone = body.MSISDN
			} else if body.Phone != "" {
				phone = body.Phone
			}
		}
	}

	if phone == "" && userID == uuid.Nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required to spin"})
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

// CheckEligibility returns the user's current spin eligibility state.
// Uses live-query approach: queries the recharges table by MSISDN for today's
// sum, then looks up the matching spin tier and counts spins played today.
// This mirrors the RechargeMax architecture and is immune to stale wallet counters.
//
// GET /api/v1/spin/eligibility
func (h *SpinHandler) CheckEligibility(w http.ResponseWriter, r *http.Request) {
	// Extract phone from JWT context
	phone, _ := r.Context().Value(middleware.ContextPhone).(string)

	// Fallback: if no phone in context, use UUID-based lookup
	if phone == "" {
		uidStr, _ := r.Context().Value(middleware.ContextUserID).(string)
		userID, err := uuid.Parse(uidStr)
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
		return
	}

	// Phone-based live-query eligibility check (preferred path)
	eligibility, err := h.spinSvc.CheckEligibilityByPhone(r.Context(), phone)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, eligibility)
}
