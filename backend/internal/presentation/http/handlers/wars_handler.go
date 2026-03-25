package handlers

import (
	"fmt"
	"net/http"
	"time"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"
	"github.com/google/uuid"
)

// WarsHandler serves Regional Wars endpoints.
type WarsHandler struct {
	warsSvc *services.RegionalWarsService
}

func NewWarsHandler(ws *services.RegionalWarsService) *WarsHandler {
	return &WarsHandler{warsSvc: ws}
}

// GET /api/v1/wars/leaderboard
func (h *WarsHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	rows, err := h.warsSvc.GetLeaderboard(r.Context(), 37)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load leaderboard"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"leaderboard": rows,
		"period":      warsPeriod(),
	})
}

// GET /api/v1/wars/my-rank
func (h *WarsHandler) GetMyRank(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)
	entry, err := h.warsSvc.GetUserRank(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ranked":  false,
			"message": "Update your state in Settings to join Regional Wars",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ranked": true,
		"entry":  entry,
	})
}

// POST /api/v1/admin/wars/resolve  (admin only)
func (h *WarsHandler) AdminResolve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Period string `json:"period"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Period == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "period required (YYYY-MM)"})
		return
	}
	if err := h.warsSvc.ResolveWar(r.Context(), body.Period); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "war resolved", "period": body.Period})
}

// warsPeriod returns "YYYY-MM" for the current UTC month.
func warsPeriod() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%d-%02d", t.Year(), t.Month())
}
