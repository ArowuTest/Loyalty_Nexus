package handlers

import (
	"net/http"

	"loyalty-nexus/internal/application/services"
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