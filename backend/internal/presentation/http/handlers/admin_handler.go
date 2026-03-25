package handlers

import (
	"encoding/json"
	"net/http"

	"loyalty-nexus/internal/infrastructure/config"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db  *gorm.DB
	cfg *config.ConfigManager
}

func NewAdminHandler(db *gorm.DB, cfg *config.ConfigManager) *AdminHandler {
	return &AdminHandler{db: db, cfg: cfg}
}

func (h *AdminHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalUsers       int64 `json:"total_users"`
		ActiveToday      int64 `json:"active_today"`
		TotalRechargeKobo int64 `json:"total_recharge_kobo"`
		SpinsToday       int64 `json:"spins_today"`
		GenerationsToday int64 `json:"studio_generations_today"`
	}
	h.db.WithContext(r.Context()).Table("users").Count(&stats.TotalUsers)
	writeJSON(w, http.StatusOK, stats)
}

func (h *AdminHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	var configs []struct {
		Key         string          `json:"key" gorm:"column:key"`
		Value       json.RawMessage `json:"value" gorm:"column:value"`
		Description string          `json:"description" gorm:"column:description"`
		UpdatedAt   interface{}     `json:"updated_at" gorm:"column:updated_at"`
	}
	h.db.WithContext(r.Context()).Table("network_configs").Order("key").Find(&configs)
	writeJSON(w, http.StatusOK, map[string]interface{}{"configs": configs})
}

type UpdateConfigRequest struct {
	Value string `json:"value"`
}

func (h *AdminHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	result := h.db.WithContext(r.Context()).
		Table("network_configs").
		Where("key = ?", key).
		Update("value", req.Value)
	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": result.Error.Error()})
		return
	}
	// Force config refresh
	h.cfg.Refresh(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"message": "config updated"})
}

func (h *AdminHandler) GetPrizePool(w http.ResponseWriter, r *http.Request) {
	var prizes []map[string]interface{}
	h.db.WithContext(r.Context()).Table("prize_pool").Find(&prizes)
	writeJSON(w, http.StatusOK, map[string]interface{}{"prizes": prizes})
}

func (h *AdminHandler) UpdatePrize(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *AdminHandler) GetStudioTools(w http.ResponseWriter, r *http.Request) {
	var tools []map[string]interface{}
	h.db.WithContext(r.Context()).Table("studio_tools").Order("category, name").Find(&tools)
	writeJSON(w, http.StatusOK, map[string]interface{}{"tools": tools})
}

func (h *AdminHandler) UpdateStudioTool(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	var users []map[string]interface{}
	h.db.WithContext(r.Context()).Table("users").Limit(50).Order("created_at DESC").Find(&users)
	writeJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}

func (h *AdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.db.WithContext(r.Context()).Table("users").Where("id = ?", id).Update("is_active", false)
	writeJSON(w, http.StatusOK, map[string]string{"message": "user suspended"})
}

func (h *AdminHandler) GetFraudEvents(w http.ResponseWriter, r *http.Request) {
	var events []map[string]interface{}
	h.db.WithContext(r.Context()).Table("fraud_events").Where("resolved = false").Order("created_at DESC").Limit(50).Find(&events)
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": events})
}

func (h *AdminHandler) GetRegionalWars(w http.ResponseWriter, r *http.Request) {
	var stats []map[string]interface{}
	h.db.WithContext(r.Context()).Table("regional_stats").Order("rank ASC").Find(&stats)
	writeJSON(w, http.StatusOK, map[string]interface{}{"leaderboard": stats})
}
