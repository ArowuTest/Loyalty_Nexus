package handlers

// wars_handler.go — HTTP presentation layer for Regional Wars (spec §3.5)
//
// Routes:
//   GET  /api/v1/wars/leaderboard         — top-37 states for current month
//   GET  /api/v1/wars/my-rank             — authenticated user's state rank
//   GET  /api/v1/wars/history             — list of completed wars (admin + users)
//   GET  /api/v1/wars/{period}/winners    — winners for a specific war period
//   POST /api/v1/admin/wars/resolve       — admin: resolve a war period
//   PUT  /api/v1/admin/wars/prize-pool    — admin: update prize pool for active war

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"
)

type WarsHandler struct {
	warsSvc *services.RegionalWarsService
	hub     *LeaderboardHub
}

func NewWarsHandler(ws *services.RegionalWarsService, hub *LeaderboardHub) *WarsHandler {
	return &WarsHandler{warsSvc: ws, hub: hub}
}

// GET /api/v1/wars/leaderboard
func (h *WarsHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 37 {
		limit = 37
	}

	entries, err := h.warsSvc.GetLeaderboard(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load leaderboard"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"leaderboard": entries,
		"count":       len(entries),
		"period":      currentWarPeriod(),
	})
}

// GET /api/v1/wars/my-rank
func (h *WarsHandler) GetMyRank(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, err := uuid.Parse(uid)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}

	entry, err := h.warsSvc.GetUserRank(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ranked":  false,
			"message": "Set your state in Settings to join Regional Wars",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ranked": true,
		"entry":  entry,
	})
}

// GET /api/v1/wars/history
func (h *WarsHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 12
	}

	wars, err := h.warsSvc.ListWars(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load war history"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"wars": wars, "count": len(wars)})
}

// GET /api/v1/wars/{period}/winners
func (h *WarsHandler) GetWinners(w http.ResponseWriter, r *http.Request) {
	period := r.PathValue("period")
	if period == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "period required (YYYY-MM)"})
		return
	}

	// Find war ID for this period
	wars, err := h.warsSvc.ListWars(r.Context(), 24)
	if err != nil || len(wars) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "war not found"})
		return
	}
	var warID uuid.UUID
	for _, war := range wars {
		if war.Period == period {
			warID = war.ID
			break
		}
	}
	if warID == uuid.Nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "war period not found"})
		return
	}

	winners, err := h.warsSvc.GetWinnersForWar(r.Context(), warID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load winners"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"winners": winners, "period": period})
}

// POST /api/v1/admin/wars/resolve
func (h *WarsHandler) AdminResolve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Period string `json:"period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Period == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "period required (YYYY-MM)"})
		return
	}

	winners, err := h.warsSvc.ResolveWar(r.Context(), body.Period)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "war resolved",
		"period":  body.Period,
		"winners": winners,
	})
}

// PUT /api/v1/admin/wars/prize-pool
func (h *WarsHandler) AdminUpdatePrizePool(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Period     string `json:"period"`
		PrizeKobo  int64  `json:"prize_kobo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Period == "" || body.PrizeKobo <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "period and prize_kobo > 0 required"})
		return
	}

	if err := h.warsSvc.UpdateWarPrizePool(r.Context(), body.Period, body.PrizeKobo); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "prize pool updated",
		"period":     body.Period,
		"prize_kobo": body.PrizeKobo,
	})
}

// currentWarPeriod returns "YYYY-MM" for the current UTC month.
func currentWarPeriod() string {
	return services.ExportedCurrentPeriod()
}

// ─── Secondary Draw (admin) ───────────────────────────────────────────────────

// RunSecondaryDraw executes the secondary draw for one winning state.
// POST /api/v1/admin/wars/{war_id}/secondary-draw
func (h *WarsHandler) RunSecondaryDraw(w http.ResponseWriter, r *http.Request) {
	warIDStr := r.PathValue("war_id")
	warID, err := uuid.Parse(warIDStr)
	if err != nil {
		jsonError(w, "invalid war_id", http.StatusBadRequest)
		return
	}

	var req services.SecondaryDrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.WarID = warID

	// Optionally attach the admin who triggered it
	if adminID, ok := r.Context().Value(middleware.ContextUserID).(string); ok {
		if pid, pErr := uuid.Parse(adminID); pErr == nil {
			req.TriggeredBy = &pid
		}
	}

	result, err := h.warsSvc.RunSecondaryDraw(r.Context(), req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, result)
}

// GetSecondaryDraws returns all secondary draws for a war.
// GET /api/v1/admin/wars/{war_id}/secondary-draws
func (h *WarsHandler) GetSecondaryDraws(w http.ResponseWriter, r *http.Request) {
	warIDStr := r.PathValue("war_id")
	warID, err := uuid.Parse(warIDStr)
	if err != nil {
		jsonError(w, "invalid war_id", http.StatusBadRequest)
		return
	}
	draws, err := h.warsSvc.GetSecondaryDrawsForWar(r.Context(), warID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"draws": draws})
}

// MarkSecondaryWinnerPaid records a MoMo payment for one secondary draw winner.
// POST /api/v1/admin/wars/secondary-draw/winners/{winner_id}/pay
func (h *WarsHandler) MarkSecondaryWinnerPaid(w http.ResponseWriter, r *http.Request) {
	winnerIDStr := r.PathValue("winner_id")
	winnerID, err := uuid.Parse(winnerIDStr)
	if err != nil {
		jsonError(w, "invalid winner_id", http.StatusBadRequest)
		return
	}

	var body struct {
		MoMoNumber string `json:"momo_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}

	var paidByID uuid.UUID
	if adminID, ok := r.Context().Value(middleware.ContextUserID).(string); ok {
		paidByID, _ = uuid.Parse(adminID)
	}

	if err := h.warsSvc.MarkSecondaryWinnerPaid(r.Context(), winnerID, body.MoMoNumber, paidByID); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]interface{}{"status": "paid"})
}

// GetWinnersByWarID returns the top-3 state winners for a war identified by UUID.
// GET /api/v1/admin/wars/{war_id}/winners
func (h *WarsHandler) GetWinnersByWarID(w http.ResponseWriter, r *http.Request) {
	warIDStr := r.PathValue("war_id")
	warID, err := uuid.Parse(warIDStr)
	if err != nil {
		jsonError(w, "invalid war_id", http.StatusBadRequest)
		return
	}
	winners, err := h.warsSvc.GetWinnersForWar(r.Context(), warID)
	if err != nil {
		jsonError(w, "failed to load winners: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"winners": winners})
}
