package handlers

import (
	"net/http"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"
	"github.com/google/uuid"
)

// PassportHandler serves Digital Passport endpoints.
type PassportHandler struct {
	passportSvc *services.PassportService
}

func NewPassportHandler(ps *services.PassportService) *PassportHandler {
	return &PassportHandler{passportSvc: ps}
}

// GET /api/v1/passport
func (h *PassportHandler) GetPassport(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)
	passport, err := h.passportSvc.GetPassport(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load passport"})
		return
	}
	writeJSON(w, http.StatusOK, passport)
}

// GET /api/v1/passport/badges
func (h *PassportHandler) GetBadges(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)
	passport, err := h.passportSvc.GetPassport(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load badges"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"earned":  passport.Badges,
		"tier":    passport.Tier,
		"streak":  passport.StreakCount,
	})
}