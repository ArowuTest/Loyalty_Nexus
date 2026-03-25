package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationHandler serves in-app notification endpoints.
type NotificationHandler struct {
	db *gorm.DB
}

func NewNotificationHandler(db *gorm.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

type Notification struct {
	ID        string     `json:"id"         gorm:"column:id"`
	UserID    string     `json:"user_id"    gorm:"column:user_id"`
	Title     string     `json:"title"      gorm:"column:title"`
	Body      string     `json:"body"       gorm:"column:body"`
	Type      string     `json:"type"       gorm:"column:type"`
	DeepLink  string     `json:"deep_link"  gorm:"column:deep_link"`
	ImageURL  string     `json:"image_url"  gorm:"column:image_url"`
	IsRead    bool       `json:"is_read"    gorm:"column:is_read"`
	ReadAt    *time.Time `json:"read_at"    gorm:"column:read_at"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at"`
}

// ListNotifications GET /api/v1/notifications
func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	cursor := r.URL.Query().Get("cursor") // ISO timestamp for pagination

	var notifs []Notification
	q := h.db.WithContext(r.Context()).Table("notifications").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit)
	if cursor != "" {
		q = q.Where("created_at < ?", cursor)
	}
	if err := q.Find(&notifs).Error; err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}

	var unreadCount int64
	h.db.WithContext(r.Context()).Table("notifications").
		Where("user_id = ? AND is_read = false", userID).Count(&unreadCount)

	resp := map[string]interface{}{
		"notifications": notifs,
		"unread_count":  unreadCount,
	}
	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil { log.Printf("[Notify] encode error: %v", encErr) }
}

// MarkRead PATCH /api/v1/notifications/{id}/read
func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	idStr := r.PathValue("id")
	if _, err := uuid.Parse(idStr); err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	now := time.Now().UTC()
	h.db.WithContext(r.Context()).Table("notifications").
		Where("id = ? AND user_id = ?", idStr, userID).
		Updates(map[string]interface{}{"is_read": true, "read_at": now})
	w.WriteHeader(http.StatusNoContent)
}

// MarkAllRead POST /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	now := time.Now().UTC()
	result := h.db.WithContext(r.Context()).Table("notifications").
		Where("user_id = ? AND is_read = false", userID).
		Updates(map[string]interface{}{"is_read": true, "read_at": now})
	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(map[string]int64{"marked": result.RowsAffected}); encErr != nil { log.Printf("[Notify] encode error: %v", encErr) }
}

// RegisterPushToken POST /api/v1/notifications/push-token
func (h *NotificationHandler) RegisterPushToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	var body struct {
		Token    string `json:"token"`
		Platform string `json:"platform"` // android | ios | web
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}
	if body.Platform == "" {
		body.Platform = "android"
	}
	now := time.Now().UTC()
	// Upsert push token
	h.db.WithContext(r.Context()).Exec(`
		INSERT INTO push_tokens (id, user_id, token, platform, is_active, last_seen_at, created_at, updated_at)
		VALUES (gen_random_uuid(), ?, ?, ?, true, ?, ?, ?)
		ON CONFLICT (user_id, token) DO UPDATE SET
			is_active = true, last_seen_at = EXCLUDED.last_seen_at, updated_at = EXCLUDED.updated_at
	`, userID, body.Token, body.Platform, now, now, now)

	// Also update users.fcm_token for quick access
	h.db.WithContext(r.Context()).Table("users").
		Where("id = ?", userID).
		Update("fcm_token", body.Token)

	w.WriteHeader(http.StatusNoContent)
}

// GetPreferences GET /api/v1/notifications/preferences
func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	var prefs map[string]interface{}
	h.db.WithContext(r.Context()).Table("notification_preferences").
		Where("user_id = ?", userID).
		Take(&prefs)
	if len(prefs) == 0 {
		// Return defaults
		prefs = map[string]interface{}{
			"push_enabled":      true,
			"sms_enabled":       true,
			"marketing_enabled": true,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(prefs); encErr != nil { log.Printf("[Notify] encode error: %v", encErr) }
}

// UpdatePreferences PATCH /api/v1/notifications/preferences
func (h *NotificationHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	now := time.Now().UTC()
	updates["updated_at"] = now

	h.db.WithContext(r.Context()).Exec(`
		INSERT INTO notification_preferences (user_id, created_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, now, now)
	h.db.WithContext(r.Context()).Table("notification_preferences").
		Where("user_id = ?", userID).Updates(updates)

	w.WriteHeader(http.StatusNoContent)
}
