package handlers

import (
	"net/http"
	"time"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"
	"github.com/google/uuid"
)

// DrawHandler exposes draw engine endpoints.
type DrawHandler struct {
	drawSvc *services.DrawService
}

func NewDrawHandler(ds *services.DrawService) *DrawHandler {
	return &DrawHandler{drawSvc: ds}
}

// GET /api/v1/draws
func (h *DrawHandler) ListUpcoming(w http.ResponseWriter, r *http.Request) {
	draws, err := h.drawSvc.ListUpcomingDraws(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load draws"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"draws": draws})
}

// GET /api/v1/draws/{id}/winners
func (h *DrawHandler) GetWinners(w http.ResponseWriter, r *http.Request) {
	drawID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid draw id"})
		return
	}
	winners, err := h.drawSvc.GetDrawWinners(r.Context(), drawID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load winners"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"winners": winners})
}

// GET /api/v1/user/draw-wins
func (h *DrawHandler) GetMyWins(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok || userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	type myWin struct {
		DrawID      string    `json:"draw_id"`
		DrawName    string    `json:"draw_name"`
		Position    int       `json:"position"`
		PrizeValue  float64   `json:"prize_value"`
		IsRunnerUp  bool      `json:"is_runner_up"`
		WonAt       time.Time `json:"won_at"`
	}

	var wins []myWin
	if err := h.drawSvc.DB().WithContext(r.Context()).Raw(`
		SELECT dw.draw_id::text AS draw_id, d.name as draw_name, dw.position,
		       dw.prize_value_kobo::float / 100.0 as prize_value,
		       (dw.status = 'RUNNER_UP') as is_runner_up,
		       dw.created_at as won_at
		FROM draw_winners dw
		JOIN draws d ON d.id = dw.draw_id
		WHERE dw.user_id = ?
		ORDER BY dw.created_at DESC
		LIMIT 20
	`, userID).Scan(&wins).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load draw wins"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"wins": wins})
}

// POST /api/v1/admin/draws/{id}/execute
func (h *DrawHandler) Execute(w http.ResponseWriter, r *http.Request) {
	drawID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid draw id"})
		return
	}
	if err := h.drawSvc.ExecuteDraw(r.Context(), drawID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "draw executed successfully"})
}
