package handlers

import (
	"net/http"

	"loyalty-nexus/internal/application/services"
	"github.com/google/uuid"
)

// FraudHandler exposes fraud management endpoints (admin only).
type FraudHandler struct {
	fraudSvc *services.FraudService
}

func NewFraudHandler(fs *services.FraudService) *FraudHandler {
	return &FraudHandler{fraudSvc: fs}
}

// GET /api/v1/admin/fraud-events
func (h *FraudHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.fraudSvc.ListOpenEvents(r.Context(), 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load events"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": events})
}

// PUT /api/v1/admin/fraud-events/{id}/resolve
func (h *FraudHandler) ResolveEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}
	if err := h.fraudSvc.ResolveEvent(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// PUT /api/v1/admin/users/{id}/suspend
func (h *FraudHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if decErr := decodeJSON(r, &body); decErr != nil {
		// Non-fatal: body is optional, Reason has a default
		body.Reason = ""
	}
	if body.Reason == "" {
		body.Reason = "Admin manual suspension"
	}
	if err := h.fraudSvc.SuspendUser(r.Context(), id, body.Reason); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "user suspended"})
}