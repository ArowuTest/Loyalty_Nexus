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
