package handlers

import (
	"encoding/json"
	"net/http"
	"loyalty-nexus/internal/application/services"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	valJSON, _ := json.Marshal(req.Value)
	err := h.db.Table("program_configs").
		Where("config_key = ?", req.Key).
		Update("config_value", valJSON).Error
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(204)
}

func (h *AdminHandler) ListPrizes(w http.ResponseWriter, r *http.Request) {
	var prizes []struct {
		ID      uuid.UUID `json:"id"`
		Name    string    `json:"name"`
		Type    string    `json:"prize_type"`
		Value   float64   `json:"base_value"`
		Weight  int       `json:"win_probability_weight"`
		Active  bool      `json:"is_active"`
	}
	if err := h.db.Table("prize_pool").Find(&prizes).Error; err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(prizes)
}

func (h *AdminHandler) UpdatePrizeWeight(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     uuid.UUID `json:"id"`
		Weight int       `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	err := h.db.Table("prize_pool").Where("id = ?", req.ID).Update("win_probability_weight", req.Weight).Error
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(204)
}

// UpdateSMSTemplate handles editing of SMS content (REQ-5.7.1)
func (h *AdminHandler) UpdateSMSTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slug    string `json:"slug"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	err := h.db.Table("notification_templates").Where("slug = ?", req.Slug).Update("content_template", req.Content).Error
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(204)
}

// ListFraudAlerts shows users breaching velocity thresholds (REQ-5.6.2)
func (h *AdminHandler) ListFraudAlerts(w http.ResponseWriter, r *http.Request) {
	// Simple view of transactions where velocity might be high
	var alerts []map[string]interface{}
	h.db.Raw(`
		SELECT msisdn, count(*) as tx_count 
		FROM transactions 
		WHERE created_at > now() - interval '1 hour'
		GROUP BY msisdn 
		HAVING count(*) > 10
	`).Scan(&alerts)
	json.NewEncoder(w).Encode(alerts)
}
