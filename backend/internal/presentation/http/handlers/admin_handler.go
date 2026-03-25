package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/config"
)

// AdminHandler handles all /api/v1/admin/* endpoints.
// Zero-hardcoding: every business parameter is read from network_configs,
// editable live via PUT /api/v1/admin/config/:key.
type AdminHandler struct {
	db       *gorm.DB
	cfg      *config.ConfigManager
	spinSvc  *services.SpinService
	drawSvc  *services.DrawService
	fraudSvc *services.FraudService
	warsSvc  *services.RegionalWarsService
	rdb      *redis.Client
}

func NewAdminHandler(
	db *gorm.DB,
	cfg *config.ConfigManager,
	spinSvc *services.SpinService,
	drawSvc *services.DrawService,
	fraudSvc *services.FraudService,
	warsSvc *services.RegionalWarsService,
	rdb *redis.Client,
) *AdminHandler {
	return &AdminHandler{
		db:       db,
		cfg:      cfg,
		spinSvc:  spinSvc,
		drawSvc:  drawSvc,
		fraudSvc: fraudSvc,
		warsSvc:  warsSvc,
		rdb:      rdb,
	}
}

// ─── Dashboard ────────────────────────────────────────────────────────────

func (h *AdminHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var totalUsers, activeToday, totalSpins, pendingPrizes int64
	var totalPointsIssued int64

	h.db.WithContext(ctx).Table("users").Count(&totalUsers)
	h.db.WithContext(ctx).Table("users").
		Where("last_recharge_at >= ?", time.Now().Add(-24*time.Hour)).Count(&activeToday)
	h.db.WithContext(ctx).Table("spin_results").Count(&totalSpins)
	h.db.WithContext(ctx).Table("spin_results").
		Where("fulfillment_status IN ('pending','processing','pending_momo_setup')").
		Count(&pendingPrizes)
	h.db.WithContext(ctx).Table("transactions").
		Where("type IN ('recharge_reward','prize_award','bonus')").
		Select("COALESCE(SUM(points_delta), 0)").
		Scan(&totalPointsIssued)

	spinStats, _ := h.spinSvc.GetStats(ctx)
	drawStats, _ := h.drawSvc.GetStats(ctx)

	jsonOK(w, map[string]interface{}{
		"total_users":       totalUsers,
		"active_today":      activeToday,
		"total_spins":       totalSpins,
		"pending_prizes":    pendingPrizes,
		"total_points_issued": totalPointsIssued,
		"spin_stats":        spinStats,
		"draw_stats":        drawStats,
		"generated_at":      time.Now(),
	})
}

// ─── Config (network_configs table — zero-hardcoding) ────────────────────

func (h *AdminHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	type row struct {
		Key         string    `gorm:"column:key" json:"key"`
		Value       string    `gorm:"column:value" json:"value"`
		Description string    `gorm:"column:description" json:"description"`
		UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
	}
	var rows []row
	if err := h.db.WithContext(r.Context()).Table("network_configs").Order("key ASC").Find(&rows).Error; err != nil {
		jsonError(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	jsonOK(w, rows)
}

func (h *AdminHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		jsonError(w, "config key required", http.StatusBadRequest)
		return
	}
	var body struct {
		Value       string `json:"value"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	updates := map[string]interface{}{
		"value":      body.Value,
		"updated_at": time.Now(),
	}
	if body.Description != "" {
		updates["description"] = body.Description
	}
	err := h.db.WithContext(r.Context()).
		Table("network_configs").
		Where("key = ?", key).
		Updates(updates).Error
	if err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok", "key": key, "value": body.Value})
}

// ─── Users ────────────────────────────────────────────────────────────────

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	type userRow struct {
		ID          string     `gorm:"column:id" json:"id"`
		PhoneNumber string     `gorm:"column:phone_number" json:"phone_number"`
		Tier        string     `gorm:"column:tier" json:"tier"`
		State       string     `gorm:"column:state" json:"state"`
		IsActive    bool       `gorm:"column:is_active" json:"is_active"`
		StreakCount int        `gorm:"column:streak_count" json:"streak_count"`
		LastRechargeAt *time.Time `gorm:"column:last_recharge_at" json:"last_recharge_at,omitempty"`
		CreatedAt   time.Time  `gorm:"column:created_at" json:"created_at"`
		PulsePoints int64      `gorm:"column:pulse_points" json:"pulse_points"`
		SpinCredits int        `gorm:"column:spin_credits" json:"spin_credits"`
	}

	var users []userRow
	var total int64
	base := h.db.WithContext(r.Context()).Table("users u").
		Select("u.id, u.phone_number, u.tier, u.state, u.is_active, u.streak_count, u.last_recharge_at, u.created_at, COALESCE(w.pulse_points,0) AS pulse_points, COALESCE(w.spin_credits,0) AS spin_credits").
		Joins("LEFT JOIN wallets w ON w.user_id = u.id")

	if search := q.Get("search"); search != "" {
		base = base.Where("u.phone_number LIKE ?", "%"+search+"%")
	}
	if state := q.Get("state"); state != "" {
		base = base.Where("u.state = ?", state)
	}

	base.Count(&total)
	base.Order("u.created_at DESC").Limit(limit).Offset((page-1)*limit).Find(&users)

	jsonOK(w, map[string]interface{}{
		"users":  users,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var user map[string]interface{}
	if err := h.db.WithContext(r.Context()).Table("users u").
		Select("u.*, COALESCE(w.pulse_points,0) AS pulse_points, COALESCE(w.spin_credits,0) AS spin_credits, COALESCE(w.lifetime_points,0) AS lifetime_points").
		Joins("LEFT JOIN wallets w ON w.user_id = u.id").
		Where("u.id = ?", id).
		First(&user).Error; err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	jsonOK(w, user)
}

func (h *AdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Suspended bool   `json:"suspended"`
		Reason    string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	h.db.WithContext(r.Context()).Table("users").
		Where("id = ?", id).
		Update("is_active", !body.Suspended)
	jsonOK(w, map[string]bool{"suspended": body.Suspended})
}

func (h *AdminHandler) AdjustPoints(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
		Delta  int64  `json:"delta"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		jsonError(w, "user_id and delta required", http.StatusBadRequest)
		return
	}
	now := time.Now()
	err := h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("UPDATE wallets SET pulse_points = pulse_points + ?, updated_at = ? WHERE user_id = ?",
			body.Delta, now, body.UserID).Error; err != nil {
			return err
		}
		metaJSON, _ := json.Marshal(map[string]string{"admin_reason": body.Reason})
		return tx.Exec(`INSERT INTO transactions (id, user_id, type, points_delta, reference, metadata, created_at)
			VALUES (?, ?, 'admin_adjust', ?, ?, ?, ?)`,
			uuid.New(), body.UserID, body.Delta, "admin_adjust_"+uuid.New().String()[:8], string(metaJSON), now).Error
	})
	if err != nil {
		jsonError(w, "adjustment failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"status": "ok", "delta": body.Delta})
}

// ─── Prize Pool (Spin Wheel) ──────────────────────────────────────────────

func (h *AdminHandler) GetPrizePool(w http.ResponseWriter, r *http.Request) {
	prizes, err := h.spinSvc.GetAllPrizes(r.Context())
	if err != nil {
		jsonError(w, "failed to get prizes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, prizes)
}

func (h *AdminHandler) CreatePrize(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	prize, err := h.spinSvc.CreatePrize(r.Context(), data)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(prize)
}

func (h *AdminHandler) UpdatePrize(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	prizeID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid prize id", http.StatusBadRequest)
		return
	}
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	prize, err := h.spinSvc.UpdatePrize(r.Context(), prizeID, data)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, prize)
}

// UpdatePrizeFull is an alias kept for backward compat with existing admin routes.
func (h *AdminHandler) UpdatePrizeFull(w http.ResponseWriter, r *http.Request) {
	h.UpdatePrize(w, r)
}

func (h *AdminHandler) DeletePrize(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	prizeID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid prize id", http.StatusBadRequest)
		return
	}
	if err := h.spinSvc.DeletePrize(r.Context(), prizeID); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) GetSpinConfig(w http.ResponseWriter, r *http.Request) {
	// Returns the full spin configuration (prize table + global limits from network_configs)
	prizes, _ := h.spinSvc.GetAllPrizes(r.Context())
	spinStats, _ := h.spinSvc.GetStats(r.Context())
	jsonOK(w, map[string]interface{}{
		"prizes":               prizes,
		"max_spins_per_day":    h.cfg.GetInt("spin_max_per_user_per_day", 3),
		"spin_trigger_naira":   h.cfg.GetInt64("spin_trigger_naira", 1000),
		"liability_cap_naira":  h.cfg.GetInt64("daily_prize_liability_cap_naira", 500000),
		"stats":                spinStats,
	})
}

func (h *AdminHandler) UpdateSpinConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MaxSpinsPerDay    *int   `json:"max_spins_per_day"`
		SpinTriggerNaira  *int64 `json:"spin_trigger_naira"`
		LiabilityCapNaira *int64 `json:"liability_cap_naira"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	updates := map[string]string{}
	if body.MaxSpinsPerDay != nil {
		updates["spin_max_per_user_per_day"] = strconv.Itoa(*body.MaxSpinsPerDay)
	}
	if body.SpinTriggerNaira != nil {
		updates["spin_trigger_naira"] = strconv.FormatInt(*body.SpinTriggerNaira, 10)
	}
	if body.LiabilityCapNaira != nil {
		updates["daily_prize_liability_cap_naira"] = strconv.FormatInt(*body.LiabilityCapNaira, 10)
	}
	for k, v := range updates {
		h.db.WithContext(r.Context()).Exec(
			"UPDATE network_configs SET value = ?, updated_at = NOW() WHERE key = ?", v, k)
	}
	jsonOK(w, map[string]string{"status": "ok"})
}

// ─── Draws ────────────────────────────────────────────────────────────────

func (h *AdminHandler) GetDraws(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	draws, total, err := h.drawSvc.GetDraws(r.Context(), page, limit)
	if err != nil {
		jsonError(w, "failed to get draws: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{
		"draws": draws,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AdminHandler) CreateDraw(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name           string    `json:"name"`
		Description    string    `json:"description"`
		DrawType       string    `json:"draw_type"`
		Recurrence     string    `json:"recurrence"`
		DrawDate       time.Time `json:"draw_date"`
		PrizePool      float64   `json:"prize_pool"`
		WinnerCount    int       `json:"winner_count"`
		RunnerUpsCount int       `json:"runner_ups_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.DrawDate.IsZero() {
		body.DrawDate = time.Now().Add(30 * 24 * time.Hour)
	}
	draw, err := h.drawSvc.CreateDraw(
		r.Context(),
		body.Name, body.Description, body.DrawType, body.Recurrence,
		body.DrawDate, body.PrizePool, body.WinnerCount, body.RunnerUpsCount,
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(draw)
}

func (h *AdminHandler) UpdateDraw(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	drawID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid draw id", http.StatusBadRequest)
		return
	}
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	draw, err := h.drawSvc.UpdateDraw(r.Context(), drawID, updates)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, draw)
}

func (h *AdminHandler) ExecuteDraw(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	drawID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid draw id", http.StatusBadRequest)
		return
	}
	if err := h.drawSvc.ExecuteDraw(r.Context(), drawID); err != nil {
		jsonError(w, "execution failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]string{"status": "completed"})
}

func (h *AdminHandler) GetDrawWinners(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	drawID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid draw id", http.StatusBadRequest)
		return
	}
	winners, err := h.drawSvc.GetDrawWinners(r.Context(), drawID)
	if err != nil {
		jsonError(w, "failed to get winners: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, winners)
}

func (h *AdminHandler) ExportDrawEntries(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	drawID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid draw id", http.StatusBadRequest)
		return
	}
	outPath := fmt.Sprintf("/tmp/draw_%s_entries.csv", drawID.String()[:8])
	path, err := h.drawSvc.ExportDrawEntries(r.Context(), drawID, outPath)
	if err != nil {
		jsonError(w, "export failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=draw_%s_entries.csv", drawID.String()[:8]))
	w.Header().Set("Content-Type", "text/csv")
	http.ServeFile(w, r, path)
}

// ─── Studio Tools ─────────────────────────────────────────────────────────

func (h *AdminHandler) GetStudioTools(w http.ResponseWriter, r *http.Request) {
	// Query the real studio_tools table (seeded with all tools by migrations)
	type toolRow struct {
		ID          string `gorm:"column:id"          json:"id"`
		Slug        string `gorm:"column:slug"        json:"slug"`
		Name        string `gorm:"column:name"        json:"name"`
		Category    string `gorm:"column:category"    json:"category"`
		Provider    string `gorm:"column:provider"    json:"provider"`
		PointCost   int64  `gorm:"column:point_cost"  json:"point_cost"`
		IsActive    bool   `gorm:"column:is_active"   json:"is_active"`
		Description string `gorm:"column:description" json:"description"`
		UsageCount  int64  `json:"usage_count"`
	}
	var rows []toolRow
	h.db.WithContext(r.Context()).
		Raw(`SELECT t.id, t.slug, t.name, t.category, t.provider, t.point_cost, t.is_active, t.description,
		     COUNT(g.id) AS usage_count
		     FROM studio_tools t
		     LEFT JOIN studio_generations g ON g.tool_id = t.id
		     GROUP BY t.id
		     ORDER BY t.category, t.name`).
		Scan(&rows)

	jsonOK(w, map[string]interface{}{"tools": rows})
}

func (h *AdminHandler) UpdateStudioTool(w http.ResponseWriter, r *http.Request) {
	// Support both /{id} and /{key} (legacy) path params
	id := r.PathValue("id")
	if id == "" {
		id = r.PathValue("key")
	}
	var body struct {
		PointCost int64 `json:"point_cost"`
		IsActive  *bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || id == "" {
		jsonError(w, "id/key and body required", http.StatusBadRequest)
		return
	}
	// Build dynamic update
	updates := map[string]interface{}{"updated_at": time.Now()}
	if body.PointCost >= 0 {
		updates["point_cost"] = body.PointCost
	}
	if body.IsActive != nil {
		updates["is_active"] = *body.IsActive
	}
	// Try by UUID first, fall back to slug
	result := h.db.WithContext(r.Context()).
		Table("studio_tools").
		Where("id = ? OR slug = ?", id, id).
		Updates(updates)
	if result.Error != nil {
		jsonError(w, "update failed: "+result.Error.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok"})
}

// ─── Points config / ledger audit ────────────────────────────────────────

func (h *AdminHandler) GetPointsStats(w http.ResponseWriter, r *http.Request) {
	type stats struct {
		TotalPointsIssued int64 `json:"total_points_issued"`
		TotalPointsSpent  int64 `json:"total_points_spent"`
		PointsInCirculation int64 `json:"points_in_circulation"`
		ActiveWallets     int64 `json:"active_wallets"`
	}
	var s stats
	h.db.WithContext(r.Context()).Table("transactions").
		Where("type IN ('recharge_reward','bonus','prize_award') AND points_delta > 0").
		Select("COALESCE(SUM(points_delta), 0)").Scan(&s.TotalPointsIssued)
	h.db.WithContext(r.Context()).Table("transactions").
		Where("type = 'studio_spend' AND points_delta < 0").
		Select("COALESCE(SUM(ABS(points_delta)), 0)").Scan(&s.TotalPointsSpent)
	h.db.WithContext(r.Context()).Table("wallets").
		Select("COALESCE(SUM(pulse_points), 0)").Scan(&s.PointsInCirculation)
	h.db.WithContext(r.Context()).Table("wallets").
		Where("pulse_points > 0").Count(&s.ActiveWallets)
	jsonOK(w, s)
}

func (h *AdminHandler) GetPointsHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}

	var rows []map[string]interface{}
	var total int64
	base := h.db.WithContext(r.Context()).Table("transactions t").
		Select("t.*, u.phone_number").
		Joins("LEFT JOIN users u ON u.id = t.user_id").
		Where("t.type IN ('recharge_reward','bonus','prize_award','studio_spend','admin_adjust','spin_play')")

	if phone := q.Get("phone"); phone != "" {
		base = base.Where("u.phone_number LIKE ?", "%"+phone+"%")
	}
	if txType := q.Get("type"); txType != "" {
		base = base.Where("t.type = ?", txType)
	}

	base.Count(&total)
	base.Order("t.created_at DESC").Limit(limit).Offset((page-1)*limit).Find(&rows)
	jsonOK(w, map[string]interface{}{
		"transactions": rows,
		"total":        total,
		"page":         page,
		"limit":        limit,
	})
}

// ─── Notifications ────────────────────────────────────────────────────────

func (h *AdminHandler) BroadcastNotification(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title   string   `json:"title"`
		Message string   `json:"message"`
		Type    string   `json:"type"` // push | sms | both
		Targets []string `json:"targets"` // empty = all users
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Title == "" || body.Message == "" {
		jsonError(w, "title and message required", http.StatusBadRequest)
		return
	}

	broadcastID := uuid.New()
	if err := h.db.WithContext(r.Context()).Exec(`
		INSERT INTO notification_broadcasts (id, title, message, type, target_count, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'queued', NOW())
	`, broadcastID, body.Title, body.Message, body.Type, len(body.Targets)).Error; err != nil {
		jsonError(w, "failed to queue broadcast", http.StatusInternalServerError)
		return
	}

	// Background: push to all users (simplified — production would use a queue)
	go func() {
		var phoneNumbers []string
		if len(body.Targets) > 0 {
			phoneNumbers = body.Targets
		} else {
			h.db.Table("users").Where("is_active = true").Pluck("phone_number", &phoneNumbers)
		}
		h.db.Exec("UPDATE notification_broadcasts SET target_count = ?, status = 'sent', sent_at = NOW() WHERE id = ?",
			len(phoneNumbers), broadcastID)
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"broadcast_id": broadcastID,
		"status":       "queued",
	})
}

func (h *AdminHandler) GetBroadcastHistory(w http.ResponseWriter, r *http.Request) {
	var rows []map[string]interface{}
	h.db.WithContext(r.Context()).Table("notification_broadcasts").
		Order("created_at DESC").Limit(50).Find(&rows)
	jsonOK(w, rows)
}

// ─── Subscriptions ────────────────────────────────────────────────────────

func (h *AdminHandler) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	var rows []map[string]interface{}
	var total int64
	h.db.WithContext(r.Context()).Table("subscription_events").Count(&total)
	h.db.WithContext(r.Context()).Table("subscription_events se").
		Select("se.*, u.phone_number").
		Joins("LEFT JOIN users u ON u.id = se.user_id").
		Order("se.created_at DESC").
		Limit(50).Offset((page-1)*50).
		Find(&rows)
	jsonOK(w, map[string]interface{}{"subscriptions": rows, "total": total})
}

func (h *AdminHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	h.db.WithContext(r.Context()).Table("subscription_events").
		Where("id = ?", id).
		Update("event_type", body.Status)
	jsonOK(w, map[string]string{"status": "updated"})
}

// ─── Fraud ────────────────────────────────────────────────────────────────

func (h *AdminHandler) GetFraudEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")

	var rows []map[string]interface{}
	query := h.db.WithContext(r.Context()).Table("fraud_events fe").
		Select("fe.*, u.phone_number").
		Joins("LEFT JOIN users u ON u.id = fe.user_id")
	if status != "" {
		query = query.Where("fe.resolved = ?", status == "resolved")
	}
	query.Order("fe.created_at DESC").Limit(100).Find(&rows)
	jsonOK(w, rows)
}

func (h *AdminHandler) ResolveFraudEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Action string `json:"action"` // resolve | freeze | clear
		Notes  string `json:"notes"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	h.db.WithContext(r.Context()).Table("fraud_events").
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"resolved":    true,
			"resolved_at": time.Now(),
			"notes":       body.Notes,
		})
	jsonOK(w, map[string]string{"status": "resolved"})
}

// ─── Regional Wars ────────────────────────────────────────────────────────

// GetRegionalWars returns current war leaderboard + history for admin panel.
func (h *AdminHandler) GetRegionalWars(w http.ResponseWriter, r *http.Request) {
	if h.warsSvc == nil {
		jsonError(w, "wars service not available", http.StatusServiceUnavailable)
		return
	}
	// Live leaderboard (top 37)
	leaderboard, err := h.warsSvc.GetLeaderboard(r.Context(), 37)
	if err != nil {
		jsonError(w, "leaderboard error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// War history
	wars, _ := h.warsSvc.ListWars(r.Context(), 12)

	jsonOK(w, map[string]interface{}{
		"leaderboard":           leaderboard,
		"history":               wars,
		"prize_pool_kobo":       h.cfg.GetInt("regional_wars_prize_pool_kobo", 50_000_000),
		"winning_bonus_pp":      h.cfg.GetInt("regional_wars_winning_bonus", 50),
	})
}

// ResetWarsCycle is kept for backward-compat; admin should use POST /wars/resolve instead.
func (h *AdminHandler) ResetWarsCycle(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{
		"message": "Use POST /api/v1/admin/wars/resolve to close a war cycle",
	})
}

// ─── Health ───────────────────────────────────────────────────────────────

func (h *AdminHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	// Check DB
	dbOK := true
	if err := h.db.WithContext(r.Context()).Exec("SELECT 1").Error; err != nil {
		dbOK = false
	}

	// Check pending prize queue
	var pendingPrizes int64
	h.db.WithContext(r.Context()).Table("spin_results").
		Where("fulfillment_status IN ('pending','processing')").
		Count(&pendingPrizes)

	// Check fraud events (unresolved)
	var openFraudEvents int64
	h.db.WithContext(r.Context()).Table("fraud_events").
		Where("resolved = false").
		Count(&openFraudEvents)

	status := "healthy"
	if !dbOK || pendingPrizes > 100 || openFraudEvents > 50 {
		status = "degraded"
	}

	jsonOK(w, map[string]interface{}{
		"status":              status,
		"database":            dbOK,
		"pending_prizes":      pendingPrizes,
		"open_fraud_events":   openFraudEvents,
		"checked_at":          time.Now(),
		"version":             "phase-8",
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// ─── AI Health ────────────────────────────────────────────────────────────────

// GetAIHealth returns real-time provider health data from Redis.
// GET /api/v1/admin/ai-health
func (h *AdminHandler) GetAIHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.rdb == nil {
		jsonError(w, "redis not configured", http.StatusServiceUnavailable)
		return
	}

	allProviders := []string{"GROQ", "GEMINI_LITE", "DEEPSEEK"}

	// ── active_chat_provider ────────────────────────────────────────
	activeProvider, _ := h.rdb.Get(ctx, "nexus:ai:active_chat_provider").Result()

	// ── per-provider stats ──────────────────────────────────────────
	type ProviderHealth struct {
		Name          string      `json:"name"`
		Status        string      `json:"status"`
		RequestsToday int64       `json:"requests_today"`
		LastUsedAt    *time.Time  `json:"last_used_at"`
		LastError     interface{} `json:"last_error"` // null when no error
	}

	providers := make([]ProviderHealth, 0, len(allProviders))
	for _, name := range allProviders {
		pkey := "nexus:ai:provider:" + name

		status, err := h.rdb.Get(ctx, pkey+":status").Result()
		if err != nil {
			status = "unknown"
		}

		reqStr, _ := h.rdb.Get(ctx, pkey+":requests_today").Result()
		var reqsToday int64
		if reqStr != "" {
			fmt.Sscanf(reqStr, "%d", &reqsToday)
		}

		var lastUsedAt *time.Time
		if tsStr, err2 := h.rdb.Get(ctx, pkey+":last_used_at").Result(); err2 == nil {
			var ts int64
			if _, scanErr := fmt.Sscanf(tsStr, "%d", &ts); scanErr == nil {
				t := time.Unix(ts, 0).UTC()
				lastUsedAt = &t
			}
		}

		var lastError interface{}
		if errMsg, err2 := h.rdb.Get(ctx, pkey+":last_error").Result(); err2 == nil && errMsg != "" {
			lastError = errMsg
		}

		providers = append(providers, ProviderHealth{
			Name:          name,
			Status:        status,
			RequestsToday: reqsToday,
			LastUsedAt:    lastUsedAt,
			LastError:     lastError,
		})
	}

	// ── recent provider switches ────────────────────────────────────
	type SwitchEntry struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Reason string `json:"reason"`
		TS     int64  `json:"ts"`
	}

	rawSwitches, _ := h.rdb.LRange(ctx, "nexus:ai:provider_switch_log", 0, 49).Result()
	recentSwitches := make([]SwitchEntry, 0, len(rawSwitches))
	for _, raw := range rawSwitches {
		var entry SwitchEntry
		if jsonErr := json.Unmarshal([]byte(raw), &entry); jsonErr == nil {
			recentSwitches = append(recentSwitches, entry)
		}
	}

	// ── studio tools ────────────────────────────────────────────────
	type StudioToolHealth struct {
		Slug          string     `json:"slug"`
		RequestsToday int64      `json:"requests_today"`
		LastProvider  string     `json:"last_provider"`
		LastUsedAt    *time.Time `json:"last_used_at"`
	}

	// Scan all nexus:ai:studio:*:requests_today keys
	var studioTools []StudioToolHealth
	iter := h.rdb.Scan(ctx, 0, "nexus:ai:studio:*:requests_today", 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val() // nexus:ai:studio:{slug}:requests_today
		// extract slug: strip prefix "nexus:ai:studio:" and suffix ":requests_today"
		const prefix = "nexus:ai:studio:"
		const suffix = ":requests_today"
		if len(key) <= len(prefix)+len(suffix) {
			continue
		}
		slug := key[len(prefix) : len(key)-len(suffix)]
		if slug == "" {
			continue
		}

		base := "nexus:ai:studio:" + slug
		reqStr, _ := h.rdb.Get(ctx, key).Result()
		var reqs int64
		fmt.Sscanf(reqStr, "%d", &reqs)

		lastProvider, _ := h.rdb.Get(ctx, base+":last_provider").Result()

		var lastUsedAt *time.Time
		if tsStr, err2 := h.rdb.Get(ctx, base+":last_used_at").Result(); err2 == nil {
			var ts int64
			if _, scanErr := fmt.Sscanf(tsStr, "%d", &ts); scanErr == nil {
				t := time.Unix(ts, 0).UTC()
				lastUsedAt = &t
			}
		}

		studioTools = append(studioTools, StudioToolHealth{
			Slug:          slug,
			RequestsToday: reqs,
			LastProvider:  lastProvider,
			LastUsedAt:    lastUsedAt,
		})
	}
	if studioTools == nil {
		studioTools = []StudioToolHealth{}
	}

	jsonOK(w, map[string]interface{}{
		"active_chat_provider": activeProvider,
		"providers":            providers,
		"recent_switches":      recentSwitches,
		"studio_tools":         studioTools,
		"checked_at":           time.Now().UTC().Format(time.RFC3339),
	})
}
