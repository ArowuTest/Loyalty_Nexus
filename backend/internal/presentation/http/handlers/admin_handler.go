package handlers

import (
	"encoding/json"
	"net/http"
	"loyalty-nexus/internal/application/services"
	"github.com/google/uuid"
)

type AdminHandler struct {
	dbService *services.ConfigService // hypothetical service for DB configs
}

func (h *AdminHandler) UpdateProgramConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var req struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	// Implementation to update program_configs table
	w.WriteHeader(204)
}

func (h *AdminHandler) ListPrizes(w http.ResponseWriter, r *http.Request) {
	// Query prize_pool
}

func (h *AdminHandler) UpdatePrizeWeight(w http.ResponseWriter, r *http.Request) {
	// Update win_probability_weight in prize_pool
}
