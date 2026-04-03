package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/pkg/safe"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// AdminHandler handles all /api/v1/admin/* endpoints.
// Zero-hardcoding: every business parameter is read from network_configs,
// editable live via PUT /api/v1/admin/config/:key.
type AdminHandler struct {
	db            *gorm.DB
	cfg           *config.ConfigManager
	spinSvc       *services.SpinService
	drawSvc       *services.DrawService
	drawWindowSvc *services.DrawWindowService
	fraudSvc      *services.FraudService
	warsSvc       *services.RegionalWarsService
	studioSvc     *services.StudioService
	claimSvc      *services.AdminClaimService
	csvSvc        *services.MTNPushCSVService  // nil-safe; set via WithCSVService
	bonusPulseSvc *services.BonusPulseService  // nil-safe; set via WithBonusPulseService
	notifySvc     *services.NotificationService // for winner SMS notifications
	rdb           *redis.Client
}

func NewAdminHandler(
	db *gorm.DB,
	cfg *config.ConfigManager,
	spinSvc *services.SpinService,
	drawSvc *services.DrawService,
	drawWindowSvc *services.DrawWindowService,
	fraudSvc *services.FraudService,
	warsSvc *services.RegionalWarsService,
	studioSvc *services.StudioService,
	claimSvc  *services.AdminClaimService,
	rdb *redis.Client,
) *AdminHandler {
	return &AdminHandler{
		db:            db,
		cfg:           cfg,
		spinSvc:       spinSvc,
		drawSvc:       drawSvc,
		drawWindowSvc: drawWindowSvc,
		fraudSvc:      fraudSvc,
		warsSvc:       warsSvc,
		studioSvc:     studioSvc,
		claimSvc:      claimSvc,
		rdb:           rdb,
	}
}

// WithCSVService attaches the MTN push CSV upload service to the handler.
// Called after construction in main.go to avoid changing the constructor signature.
// WithNotificationService injects the notification service for winner SMS.
func (h *AdminHandler) WithNotificationService(n *services.NotificationService) *AdminHandler {
	h.notifySvc = n
	return h
}

func (h *AdminHandler) WithCSVService(svc *services.MTNPushCSVService) *AdminHandler {
	h.csvSvc = svc
	return h
}

// WithBonusPulseService attaches the bonus pulse point award service.
func (h *AdminHandler) WithBonusPulseService(svc *services.BonusPulseService) *AdminHandler {
	h.bonusPulseSvc = svc
	return h
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
	// Transaction types from entities.TransactionType constants (migration 020):
	//   'points_award' = points earned from recharge (was 'recharge_reward' — does not exist)
	//   'bonus'        = admin/streak/referral bonus
	//   'prize_award'  = prize value awarded
	h.db.WithContext(ctx).Table("transactions").
		Where("type IN ('points_award','prize_award','bonus','spin_credit_award','draw_entry_award') AND points_delta > 0").
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
	jsonOK(w, map[string]interface{}{"configs": rows})
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
	desc := body.Description
	if desc == "" {
		desc = key
	}
	// Use cfg.Set() which does an upsert AND immediately updates the in-memory cache
	// so the new value is visible to all handlers without waiting for the 60s auto-refresh.
	if err := h.cfg.Set(r.Context(), key, body.Value); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	// Also update the description if provided (cfg.Set only handles key/value)
	if desc != key {
		h.db.WithContext(r.Context()).Exec(
			`UPDATE network_configs SET description = ? WHERE key = ?`, desc, key,
		)
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

	// Build base filter conditions
	search := q.Get("search")
	state := q.Get("state")

	// Count query (separate from find to avoid GORM SELECT clause conflict)
	countQ := h.db.WithContext(r.Context()).Table("users u").
		Joins("LEFT JOIN wallets w ON w.user_id = u.id")
	if search != "" {
		countQ = countQ.Where("u.phone_number LIKE ?", "%"+search+"%")
	}
	if state != "" {
		countQ = countQ.Where("u.state = ?", state)
	}
	countQ.Count(&total)

	// Data query
	dataQ := h.db.WithContext(r.Context()).Table("users u").
		Select("u.id, u.phone_number, u.tier, u.state, u.is_active, u.streak_count, u.last_recharge_at, u.created_at, COALESCE(w.pulse_points,0) AS pulse_points, COALESCE(w.spin_credits,0) AS spin_credits").
		Joins("LEFT JOIN wallets w ON w.user_id = u.id")
	if search != "" {
		dataQ = dataQ.Where("u.phone_number LIKE ?", "%"+search+"%")
	}
	if state != "" {
		dataQ = dataQ.Where("u.state = ?", state)
	}
	if dbErr := dataQ.Order("u.created_at DESC").Limit(limit).Offset((page-1)*limit).Find(&users).Error; dbErr != nil {
		log.Printf("[ListUsers] wallet join failed (%v), falling back to simple query", dbErr)
		// Fallback: query without wallet join in case wallets table has issues
		h.db.WithContext(r.Context()).Table("users u").
			Select("u.id, u.phone_number, u.tier, u.state, u.is_active, u.streak_count, u.last_recharge_at, u.created_at, 0 AS pulse_points, 0 AS spin_credits").
			Order("u.created_at DESC").Limit(limit).Offset((page-1)*limit).Find(&users)
	}
	if users == nil {
		users = []userRow{}
	}

	jsonOK(w, map[string]interface{}{
		"users":  users,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var users []map[string]interface{}
	if err := h.db.WithContext(r.Context()).Table("users u").
		Select("u.*, COALESCE(w.pulse_points,0) AS pulse_points, COALESCE(w.spin_credits,0) AS spin_credits, COALESCE(w.lifetime_points,0) AS lifetime_points").
		Joins("LEFT JOIN wallets w ON w.user_id = u.id").
		Where("u.id = ?", id).
		Limit(1).Find(&users).Error; err != nil || len(users) == 0 {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	jsonOK(w, users[0])
}

func (h *AdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Suspended bool   `json:"suspended"`
		Reason    string `json:"reason"`
	}
	// body is optional — default values (Suspended=false) apply if body omitted
	_ = json.NewDecoder(r.Body).Decode(&body)

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
	// Look up user's phone number — transactions.phone_number is NOT NULL
	var phoneNumber string
	h.db.WithContext(r.Context()).Table("users").Where("id = ?", body.UserID).Pluck("phone_number", &phoneNumber)
	if phoneNumber == "" {
		phoneNumber = "unknown"
	}
	err := h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		// Upsert wallet: create row if missing, otherwise increment
		if err := tx.Exec(`INSERT INTO wallets (id, user_id, pulse_points, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, ?, ?)
			ON CONFLICT (user_id) DO UPDATE SET pulse_points = wallets.pulse_points + EXCLUDED.pulse_points, updated_at = EXCLUDED.updated_at`,
			body.UserID, body.Delta, now, now).Error; err != nil {
			return err
		}
		metaJSON, _ := json.Marshal(map[string]string{"admin_reason": body.Reason})
		return tx.Exec(`INSERT INTO transactions (id, user_id, phone_number, type, points_delta, reference, metadata, created_at)
			VALUES (?, ?, ?, 'admin_adjust', ?, ?, ?, ?)`,
			uuid.New(), body.UserID, phoneNumber, body.Delta, "admin_adjust_"+uuid.New().String()[:8], string(metaJSON), now).Error
	})
	if err != nil {
		jsonError(w, "adjustment failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"status": "ok", "delta": body.Delta})
}

// ─── Prize Pool (Spin Wheel) ──────────────────────────────────────────────

func (h *AdminHandler) GetPrizePool(w http.ResponseWriter, r *http.Request) {
	// Admin always sees all prizes (active + inactive) unless ?active_only=true
	includeInactive := r.URL.Query().Get("active_only") != "true"
	prizes, err := h.spinSvc.GetAllPrizes(r.Context(), includeInactive)
	if err != nil {
		jsonError(w, "failed to get prizes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"prizes": prizes})
}

func (h *AdminHandler) GetPrize(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	prizeID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid prize id", http.StatusBadRequest)
		return
	}
	prize, err := h.spinSvc.GetPrize(r.Context(), prizeID)
	if err != nil {
		jsonError(w, "prize not found", http.StatusNotFound)
		return
	}
	jsonOK(w, prize)
}

func (h *AdminHandler) GetPrizeSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.spinSvc.GetPrizeProbabilitySummary(r.Context())
	if err != nil {
		jsonError(w, "failed to get prize summary: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, summary)
}

func (h *AdminHandler) ReorderPrizes(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.OrderedIDs) == 0 {
		jsonError(w, "ordered_ids array is required", http.StatusBadRequest)
		return
	}
	ids := make([]uuid.UUID, 0, len(body.OrderedIDs))
	for _, s := range body.OrderedIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			jsonError(w, "invalid prize id: "+s, http.StatusBadRequest)
			return
		}
		ids = append(ids, id)
	}
	if err := h.spinSvc.ReorderPrizes(r.Context(), ids); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "reordered"})
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
	if encErr := json.NewEncoder(w).Encode(prize); encErr != nil {
		log.Printf("[Admin] CreatePrize encode error: %v", encErr)
	}
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
	// Returns the full spin configuration (all prizes + tiers + stats)
	prizes, _ := h.spinSvc.GetAllPrizes(r.Context(), true) // include inactive
	tiers, _ := h.spinSvc.GetAllSpinTiers(r.Context())
	summary, _ := h.spinSvc.GetPrizeProbabilitySummary(r.Context())
	spinStats, _ := h.spinSvc.GetStats(r.Context())
	jsonOK(w, map[string]interface{}{
		"prizes":              prizes,
		"tiers":               tiers,
		"probability_summary": summary,
		"liability_cap_naira": h.cfg.GetInt64("daily_prize_liability_cap_naira", 500000),
		"stats":               spinStats,
	})
}

// UpdateSpinConfig updates global spin configuration keys.
// Note: daily spin caps are now controlled per-tier via /admin/spin/tiers.
// This endpoint only manages the daily prize liability cap.
func (h *AdminHandler) UpdateSpinConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LiabilityCapNaira *int64 `json:"liability_cap_naira"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.LiabilityCapNaira != nil {
		h.db.WithContext(r.Context()).Exec(
			"UPDATE network_configs SET value = ?, updated_at = NOW() WHERE key = ?",
			strconv.FormatInt(*body.LiabilityCapNaira, 10), "daily_prize_liability_cap_naira")
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
		// draw_time is the correct DB column name (migration 024 ADD COLUMN draw_time).
		// The legacy JSON key draw_date is also accepted for backwards compatibility.
		DrawTime       time.Time `json:"draw_time"`
		DrawDateLegacy time.Time `json:"draw_date"` // deprecated alias — use draw_time
		PrizePool      float64   `json:"prize_pool"`
		WinnerCount    int       `json:"winner_count"`
		RunnerUpsCount int       `json:"runner_ups_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	// Prefer draw_time; fall back to legacy draw_date if draw_time was not provided.
	effectiveDrawTime := body.DrawTime
	if effectiveDrawTime.IsZero() {
		effectiveDrawTime = body.DrawDateLegacy
	}
	if effectiveDrawTime.IsZero() {
		effectiveDrawTime = time.Now().Add(30 * 24 * time.Hour)
	}
	draw, err := h.drawSvc.CreateDraw(
		r.Context(),
		body.Name, body.Description, body.DrawType, body.Recurrence,
		effectiveDrawTime, body.PrizePool, body.WinnerCount, body.RunnerUpsCount,
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	if encErr := json.NewEncoder(w).Encode(draw); encErr != nil {
		log.Printf("[Admin] CreateDraw encode error: %v", encErr)
	}
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

	// Notify winners asynchronously (non-blocking — draw execution already committed)
	if h.notifySvc != nil {
		go h.notifyDrawWinners(drawID)
	}

	jsonOK(w, map[string]string{"status": "completed"})
}

// notifyDrawWinners fetches the draw's winners from DB and sends an SMS to each.
func (h *AdminHandler) notifyDrawWinners(drawID uuid.UUID) {
	ctx := context.Background()
	winners, err := h.drawSvc.GetDrawWinners(ctx, drawID)
	if err != nil {
		log.Printf("[Draw] notifyDrawWinners: failed to fetch winners for %s: %v", drawID, err)
		return
	}
	for _, w := range winners {
		if w.IsRunnerUp || w.PhoneNumber == "" {
			continue
		}
		msg := fmt.Sprintf(
			"🎉 Congratulations! You won the Loyalty Nexus draw! "+
				"Prize: ₦%s. Your winnings will be disbursed within 24 hours. "+
				"Ref: %s",
			formatNaira(int64(w.PrizeValue * 100)),
			drawID.String()[:8],
		)
		if err := h.notifySvc.SendSMS(ctx, w.PhoneNumber, msg); err != nil {
			log.Printf("[Draw] SMS notification failed for %s: %v", w.PhoneNumber, err)
		}
	}
	log.Printf("[Draw] ✅ Notified %d winners for draw %s", len(winners), drawID)
}

func formatNaira(kobo int64) string {
	naira := kobo / 100
	if naira == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", naira)
	// Insert commas every 3 digits from right
	var result []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, c)
	}
	return string(result)
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
		     LEFT JOIN ai_generations g ON g.tool_id = t.id
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

// ─── Studio Tools — extended admin CRUD & analytics ──────────────────────────

// CreateStudioTool creates a new studio tool.
// POST /api/v1/admin/studio-tools
func (h *AdminHandler) CreateStudioTool(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		Description  string `json:"description"`
		Category     string `json:"category"`
		PointCost    int64  `json:"point_cost"`
		Provider     string `json:"provider"`
		ProviderTool string `json:"provider_tool"`
		Icon         string `json:"icon"`
		SortOrder    int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.Slug == "" || body.Category == "" {
		jsonError(w, "name, slug, and category are required", http.StatusBadRequest)
		return
	}
	if body.PointCost < 0 {
		jsonError(w, "point_cost must be >= 0", http.StatusBadRequest)
		return
	}

	tool := entities.StudioTool{
		ID:           uuid.New(),
		Name:         body.Name,
		Slug:         body.Slug,
		Description:  body.Description,
		Category:     entities.ToolCategory(body.Category),
		PointCost:    body.PointCost,
		Provider:     body.Provider,
		ProviderTool: body.ProviderTool,
		Icon:         body.Icon,
		SortOrder:    body.SortOrder,
		IsActive:     true,
	}

	if err := h.studioSvc.UpsertTool(r.Context(), &tool); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			jsonError(w, "slug already exists", http.StatusConflict)
			return
		}
		jsonError(w, "failed to create tool: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if encErr := json.NewEncoder(w).Encode(tool); encErr != nil {
		log.Printf("[Admin] CreateStudioTool encode error: %v", encErr)
	}
}

// DisableStudioTool soft-deletes a tool by setting is_active = false.
// DELETE /api/v1/admin/studio-tools/{id}
func (h *AdminHandler) DisableStudioTool(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	toolID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid tool id", http.StatusBadRequest)
		return
	}
	if err := h.studioSvc.SetToolEnabled(r.Context(), toolID, false); err != nil {
		jsonError(w, "failed to disable tool: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"message": "tool disabled"})
}

// GetStudioToolErrors returns recent failed generations for a specific tool.
// GET /api/v1/admin/studio-tools/{id}/errors?limit=20
func (h *AdminHandler) GetStudioToolErrors(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	toolID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid tool id", http.StatusBadRequest)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, parseErr := strconv.Atoi(l); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}

	gens, err := h.studioSvc.GetToolErrors(r.Context(), toolID, limit)
	if err != nil {
		jsonError(w, "failed to fetch errors: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Project only the fields the spec requests
	type errorRow struct {
		ID           uuid.UUID `json:"id"`
		UserID       uuid.UUID `json:"user_id"`
		Prompt       string    `json:"prompt"`
		ErrorMessage string    `json:"error_message"`
		Provider     string    `json:"provider"`
		CreatedAt    time.Time `json:"created_at"`
	}
	rows := make([]errorRow, 0, len(gens))
	for _, g := range gens {
		rows = append(rows, errorRow{
			ID:           g.ID,
			UserID:       g.UserID,
			Prompt:       g.Prompt,
			ErrorMessage: g.ErrorMessage,
			Provider:     g.Provider,
			CreatedAt:    g.CreatedAt,
		})
	}

	jsonOK(w, map[string]interface{}{"errors": rows, "count": len(rows)})
}

// GetStudioToolStats returns 30-day aggregated usage stats per tool.
// GET /api/v1/admin/studio-tools/stats
func (h *AdminHandler) GetStudioToolStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.studioSvc.GetToolStats(r.Context())
	if err != nil {
		jsonError(w, "failed to fetch stats: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"stats": stats})
}

// GetStudioGenerations lists all ai_generations for admin oversight.
// GET /api/v1/admin/studio-generations?limit=50&offset=0&status=failed&tool_slug=...
func (h *AdminHandler) GetStudioGenerations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 50
	if l := q.Get("limit"); l != "" {
		if parsed, parseErr := strconv.Atoi(l); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if parsed, parseErr := strconv.Atoi(o); parseErr == nil && parsed >= 0 {
			offset = parsed
		}
	}

	filter := repositories.GenerationFilter{
		Status:   q.Get("status"),
		ToolSlug: q.Get("tool_slug"),
		Limit:    limit,
		Offset:   offset,
	}

	gens, total, err := h.studioSvc.ListGenerations(r.Context(), filter)
	if err != nil {
		jsonError(w, "failed to fetch generations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Project and truncate prompt to 80 chars
	type genRow struct {
		ID             uuid.UUID `json:"id"`
		UserID         uuid.UUID `json:"user_id"`
		ToolSlug       string    `json:"tool_slug"`
		Status         string    `json:"status"`
		Provider       string    `json:"provider"`
		Prompt         string    `json:"prompt"`
		PointsDeducted int64     `json:"points_deducted"`
		CreatedAt      time.Time `json:"created_at"`
	}
	rows := make([]genRow, 0, len(gens))
	for _, g := range gens {
		prompt := g.Prompt
		if len(prompt) > 80 {
			prompt = prompt[:80]
		}
		rows = append(rows, genRow{
			ID:             g.ID,
			UserID:         g.UserID,
			ToolSlug:       g.ToolSlug,
			Status:         g.Status,
			Provider:       g.Provider,
			Prompt:         prompt,
			PointsDeducted: g.PointsDeducted,
			CreatedAt:      g.CreatedAt,
		})
	}

	jsonOK(w, map[string]interface{}{
		"generations": rows,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
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
	// 'recharge_reward' does not exist; correct types are 'points_award' and 'bonus'.
	h.db.WithContext(r.Context()).Table("transactions").
		Where("type IN ('points_award','bonus','prize_award','spin_credit_award','draw_entry_award') AND points_delta > 0").
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
		// 'recharge_reward' does not exist; replaced with 'points_award' and 'recharge'.
		// 'admin_adjust' is written by the admin points adjustment handler (valid).
		Where("t.type IN ('recharge','points_award','bonus','prize_award','studio_spend','studio_refund','admin_adjust','spin_play','spin_credit_award','draw_entry_award')")

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
	safe.Go(func() {
		var phoneNumbers []string
		if len(body.Targets) > 0 {
			phoneNumbers = body.Targets
		} else {
			h.db.Table("users").Where("is_active = true").Pluck("phone_number", &phoneNumbers)
		}
		h.db.Exec("UPDATE notification_broadcasts SET target_count = ?, status = 'sent', sent_at = NOW() WHERE id = ?",
			len(phoneNumbers), broadcastID)
	})

	w.WriteHeader(http.StatusAccepted)
	if encErr := json.NewEncoder(w).Encode(map[string]interface{}{
		"broadcast_id": broadcastID,
		"status":       "queued",
	}); encErr != nil {
		log.Printf("[Admin] BroadcastSMS encode error: %v", encErr)
	}
}

func (h *AdminHandler) GetBroadcastHistory(w http.ResponseWriter, r *http.Request) {
	var rows []map[string]interface{}
	h.db.WithContext(r.Context()).Table("notification_broadcasts").
		Order("created_at DESC").Limit(50).Find(&rows)
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	jsonOK(w, map[string]interface{}{"broadcasts": rows, "total": len(rows)})
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
	jsonOK(w, map[string]interface{}{"events": rows})
}

func (h *AdminHandler) ResolveFraudEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Action string `json:"action"` // resolve | freeze | clear
		Notes  string `json:"notes"`
	}
	// body is optional — default values apply if body omitted
	_ = json.NewDecoder(r.Body).Decode(&body)
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
	dbStart := time.Now()
	dbOK := true
	if err := h.db.WithContext(r.Context()).Exec("SELECT 1").Error; err != nil {
		dbOK = false
	}
	dbLatency := time.Since(dbStart).Milliseconds()

	var dbPoolUsed, dbPoolMax int
	if sqlDB, err := h.db.DB(); err == nil {
		stats := sqlDB.Stats()
		dbPoolUsed = stats.InUse
		dbPoolMax = stats.MaxOpenConnections
	}

	// Check Redis
	redisOK := true
	var redisLatency int64
	if h.rdb != nil {
		rStart := time.Now()
		if err := h.rdb.Ping(r.Context()).Err(); err != nil {
			redisOK = false
		}
		redisLatency = time.Since(rStart).Milliseconds()
	} else {
		redisOK = false
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

	overall := "healthy"
	if !dbOK || !redisOK || pendingPrizes > 100 || openFraudEvents > 50 {
		overall = "degraded"
	}
	if !dbOK && !redisOK {
		overall = "outage"
	}

	services := []map[string]interface{}{
		{
			"name":         "PostgreSQL",
			"status":       map[bool]string{true: "up", false: "down"}[dbOK],
			"latency_ms":   dbLatency,
			"uptime_pct":   100.0, // Real uptime would require external monitoring
			"last_checked": time.Now(),
		},
		{
			"name":         "Redis",
			"status":       map[bool]string{true: "up", false: "down"}[redisOK],
			"latency_ms":   redisLatency,
			"uptime_pct":   100.0,
			"last_checked": time.Now(),
		},
	}

	jsonOK(w, map[string]interface{}{
		"overall":                   overall,
		"services":                  services,
		"webhook_success_rate_24h":  100.0, // Placeholder for real metrics
		"paystack_success_rate_24h": 100.0, // Placeholder for real metrics
		"api_p99_ms":                50,    // Placeholder for real metrics
		"db_pool_used":              dbPoolUsed,
		"db_pool_max":               dbPoolMax,
		"redis_hit_rate":            100.0, // Placeholder for real metrics
		"checked_at":                time.Now(),
		"pending_prizes":            pendingPrizes,
		"open_fraud_events":         openFraudEvents,
		"version":                   "phase-8",
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(data); encErr != nil {
		log.Printf("[Admin] jsonOK encode error: %v", encErr)
	}
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if encErr := json.NewEncoder(w).Encode(map[string]string{"error": msg}); encErr != nil {
		log.Printf("[Admin] jsonError encode failure: %v", encErr)
	}
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
			if _, scanErr := fmt.Sscanf(reqStr, "%d", &reqsToday); scanErr != nil {
				log.Printf("[Admin] reqsToday parse error: %v", scanErr)
			}
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
		if _, scanErr := fmt.Sscanf(reqStr, "%d", &reqs); scanErr != nil && reqStr != "" {
			log.Printf("[Admin] reqs parse error for %s: %v", slug, scanErr)
		}

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

// ─── Spin Tiers ───────────────────────────────────────────────────────────

func (h *AdminHandler) GetSpinTiers(w http.ResponseWriter, r *http.Request) {
	tiers, err := h.spinSvc.GetAllSpinTiers(r.Context())
	if err != nil {
		jsonError(w, "failed to get spin tiers: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, tiers)
}

func (h *AdminHandler) CreateSpinTier(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	tier, err := h.spinSvc.CreateSpinTier(r.Context(), data)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tier)
}

func (h *AdminHandler) UpdateSpinTier(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	tierID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid tier id", http.StatusBadRequest)
		return
	}
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	tier, err := h.spinSvc.UpdateSpinTier(r.Context(), tierID, data)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, tier)
}

func (h *AdminHandler) DeleteSpinTier(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	tierID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid tier id", http.StatusBadRequest)
		return
	}
	if err := h.spinSvc.DeleteSpinTier(r.Context(), tierID); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

// ─── Claims ───────────────────────────────────────────────────────────────

func (h *AdminHandler) ListClaims(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := 50
	offset := 0

	claims, total, err := h.claimSvc.ListClaims(r.Context(), status, limit, offset)
	if err != nil {
		jsonError(w, "failed to list claims: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]interface{}{
		"data":  claims,
		"total": total,
	})
}

func (h *AdminHandler) GetClaimDetails(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	claimID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid claim id", http.StatusBadRequest)
		return
	}

	claim, err := h.claimSvc.GetClaimDetails(r.Context(), claimID)
	if err != nil {
		jsonError(w, "claim not found", http.StatusNotFound)
		return
	}

	jsonOK(w, claim)
}

func (h *AdminHandler) ApproveClaim(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	claimID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid claim id", http.StatusBadRequest)
		return
	}

	uidStr, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	adminID, err := uuid.Parse(uidStr)
	if err != nil {
		jsonError(w, "invalid admin id", http.StatusUnauthorized)
		return
	}

	var req services.ApproveClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	claim, err := h.claimSvc.ApproveClaim(r.Context(), claimID, adminID, req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, claim)
}

func (h *AdminHandler) RejectClaim(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	claimID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid claim id", http.StatusBadRequest)
		return
	}

	uidStr, ok := r.Context().Value(middleware.ContextUserID).(string)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	adminID, err := uuid.Parse(uidStr)
	if err != nil {
		jsonError(w, "invalid admin id", http.StatusUnauthorized)
		return
	}

	var req services.RejectClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	claim, err := h.claimSvc.RejectClaim(r.Context(), claimID, adminID, req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, claim)
}

// GetPendingClaims returns all claims in PENDING_ADMIN_REVIEW status.
func (h *AdminHandler) GetPendingClaims(w http.ResponseWriter, r *http.Request) {
	claims, err := h.claimSvc.GetPendingClaims(r.Context())
	if err != nil {
		jsonError(w, "failed to get pending claims: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{
		"data":  claims,
		"total": len(claims),
	})
}

// GetClaimStatistics returns aggregate claim stats for the admin dashboard.
func (h *AdminHandler) GetClaimStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.claimSvc.GetStatistics(r.Context())
	if err != nil {
		jsonError(w, "failed to get claim statistics: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, stats)
}

// ExportClaims returns a CSV export of claims, optionally filtered by status.
func (h *AdminHandler) ExportClaims(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	csv, err := h.claimSvc.ExportCSV(r.Context(), status)
	if err != nil {
		jsonError(w, "export failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"claims_export.csv\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(csv))
}

// ─── Recharge Points Config ───────────────────────────────────────────────────

// rechargeConfigResponse is the shape returned by GET /api/v1/admin/recharge/config.
// All reward currencies are independently configurable (no shared counter):
//   - spin_naira_per_credit : minimum daily recharge (naira) to qualify for Bronze spin tier
//   - draw_naira_per_entry  : naira per Draw Entry (flat per-transaction accumulator)
//   - pulse_naira_per_point : naira per Pulse Point (flat accumulator, no tier multiplier)
//   - spin_max_per_day      : maximum spin credits per calendar day (Platinum tier cap)
//   - min_amount_naira      : minimum qualifying recharge amount
type rechargeConfigResponse struct {
	SpinNairaPerCredit int64 `json:"spin_naira_per_credit"` // ₦ minimum daily recharge for Bronze spin tier
	DrawNairaPerEntry  int64 `json:"draw_naira_per_entry"`  // ₦ per Draw Entry (flat per-transaction)
	PulseNairaPerPoint int64 `json:"pulse_naira_per_point"` // ₦ per Pulse Point (no tier multiplier)
	SpinMaxPerDay      int64 `json:"spin_max_per_day"`      // max spin credits per calendar day
	MinAmountNaira     int64 `json:"min_amount_naira"`      // minimum qualifying recharge amount
}

// GetRechargeConfig returns the admin-configurable recharge reward thresholds.
//
// GET /api/v1/admin/recharge/config
func (h *AdminHandler) GetRechargeConfig(w http.ResponseWriter, r *http.Request) {
	resp := rechargeConfigResponse{
		SpinNairaPerCredit: h.cfg.GetInt64("spin_naira_per_credit", 1000),
		DrawNairaPerEntry:  h.cfg.GetInt64("draw_naira_per_entry", 200),
		PulseNairaPerPoint: h.cfg.GetInt64("pulse_naira_per_point", 250),
		SpinMaxPerDay:      h.cfg.GetInt64("spin_max_per_day", 5),
		MinAmountNaira:     h.cfg.GetInt64("mtn_push_min_amount_naira", 50),
	}
	jsonOK(w, resp)
}

// UpdateRechargeConfig updates one or more recharge reward thresholds.
//
// PUT /api/v1/admin/recharge/config
// Body (all fields optional — only provided fields are updated):
//
//	{
//	  "spin_naira_per_credit": 1000,
//	  "draw_naira_per_entry":  200,
//	  "pulse_naira_per_point": 250,
//	  "spin_max_per_day":      5,
//	  "min_amount_naira":      50
//	}
func (h *AdminHandler) UpdateRechargeConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SpinNairaPerCredit *int64 `json:"spin_naira_per_credit"`
		DrawNairaPerEntry  *int64 `json:"draw_naira_per_entry"`
		PulseNairaPerPoint *int64 `json:"pulse_naira_per_point"`
		SpinMaxPerDay      *int64 `json:"spin_max_per_day"`
		MinAmountNaira     *int64 `json:"min_amount_naira"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	type kv struct {
		key string
		val int64
	}
	var updates []kv

	if req.SpinNairaPerCredit != nil {
		if *req.SpinNairaPerCredit < 1 {
			jsonError(w, "spin_naira_per_credit must be at least 1", http.StatusBadRequest)
			return
		}
		updates = append(updates, kv{"spin_naira_per_credit", *req.SpinNairaPerCredit})
	}
	if req.DrawNairaPerEntry != nil {
		if *req.DrawNairaPerEntry < 1 {
			jsonError(w, "draw_naira_per_entry must be at least 1", http.StatusBadRequest)
			return
		}
		updates = append(updates, kv{"draw_naira_per_entry", *req.DrawNairaPerEntry})
	}
	if req.PulseNairaPerPoint != nil {
		if *req.PulseNairaPerPoint < 1 {
			jsonError(w, "pulse_naira_per_point must be at least 1", http.StatusBadRequest)
			return
		}
		updates = append(updates, kv{"pulse_naira_per_point", *req.PulseNairaPerPoint})
	}
	if req.SpinMaxPerDay != nil {
		if *req.SpinMaxPerDay < 1 || *req.SpinMaxPerDay > 100 {
			jsonError(w, "spin_max_per_day must be between 1 and 100", http.StatusBadRequest)
			return
		}
		updates = append(updates, kv{"spin_max_per_day", *req.SpinMaxPerDay})
	}
	if req.MinAmountNaira != nil {
		if *req.MinAmountNaira < 0 {
			jsonError(w, "min_amount_naira must be non-negative", http.StatusBadRequest)
			return
		}
		updates = append(updates, kv{"mtn_push_min_amount_naira", *req.MinAmountNaira})
	}

	if len(updates) == 0 {
		jsonError(w, "no fields provided to update", http.StatusBadRequest)
		return
	}

	for _, u := range updates {
		if err := h.cfg.Set(r.Context(), u.key, fmt.Sprintf("%d", u.val)); err != nil {
			jsonError(w, fmt.Sprintf("failed to update %s: %s", u.key, err.Error()), http.StatusInternalServerError)
			return
		}
	}

	// Return the new effective config.
	resp := rechargeConfigResponse{
		SpinNairaPerCredit: h.cfg.GetInt64("spin_naira_per_credit", 1000),
		DrawNairaPerEntry:  h.cfg.GetInt64("draw_naira_per_entry", 200),
		PulseNairaPerPoint: h.cfg.GetInt64("pulse_naira_per_point", 250),
		SpinMaxPerDay:      h.cfg.GetInt64("spin_max_per_day", 5),
		MinAmountNaira:     h.cfg.GetInt64("mtn_push_min_amount_naira", 50),
	}
	jsonOK(w, resp)
}

// ─── Draw Schedule Config ─────────────────────────────────────────────────
//
// These endpoints let admin view and update the draw eligibility window rules
// stored in the draw_schedules table. All fields are admin-configurable at
// runtime with no deployment needed.

// GetDrawSchedule returns all draw window rules.
//
// GET /api/v1/admin/draw/schedule
func (h *AdminHandler) GetDrawSchedule(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.drawWindowSvc.GetAllSchedules(r.Context())
	if err != nil {
		jsonError(w, "failed to load draw schedules: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, schedules)
}

// UpdateDrawSchedule updates a single draw window rule by ID.
//
// PUT /api/v1/admin/draw/schedule/{id}
// Body (all fields optional — only provided fields are updated):
//
//	{
//	  "draw_type":            "DAILY",
//	  "draw_day_of_week":     2,
//	  "window_open_time":     "17:00:01",
//	  "window_close_time":    "17:00:00",
//	  "is_active":            true,
//	  "draw_name":            "Tuesday Daily Draw"
//	}
func (h *AdminHandler) UpdateDrawSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid schedule id — must be a UUID", http.StatusBadRequest)
		return
	}
	var req services.UpdateDrawScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updated, err := h.drawWindowSvc.UpdateSchedule(r.Context(), id, req)
	if err != nil {
		jsonError(w, "update failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, updated)
}

// CreateDrawSchedule adds a new draw window rule.
//
// POST /api/v1/admin/draw/schedule
func (h *AdminHandler) CreateDrawSchedule(w http.ResponseWriter, r *http.Request) {
	var req services.CreateDrawScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	created, err := h.drawWindowSvc.CreateSchedule(r.Context(), req)
	if err != nil {
		jsonError(w, "create failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, created)
}

// DeleteDrawSchedule soft-deletes a draw window rule.
//
// DELETE /api/v1/admin/draw/schedule/{id}
func (h *AdminHandler) DeleteDrawSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid schedule id — must be a UUID", http.StatusBadRequest)
		return
	}
	if err := h.drawWindowSvc.DeleteSchedule(r.Context(), id); err != nil {
		jsonError(w, "delete failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

// PreviewDrawWindow shows which draws a recharge at a given time would qualify for.
// Useful for admin to test window rules before going live.
//
// GET /api/v1/admin/draw/schedule/preview?at=2025-05-14T16:30:00+01:00
func (h *AdminHandler) PreviewDrawWindow(w http.ResponseWriter, r *http.Request) {
	atStr := r.URL.Query().Get("at")
	var at time.Time
	if atStr == "" {
		at = time.Now()
	} else {
		var err error
		at, err = time.Parse(time.RFC3339, atStr)
		if err != nil {
			jsonError(w, "invalid 'at' timestamp — use RFC3339 format e.g. 2025-05-14T16:30:00+01:00", http.StatusBadRequest)
			return
		}
	}
	qualifying, err := h.drawWindowSvc.ResolveQualifyingDraws(r.Context(), at)
	if err != nil {
		jsonError(w, "preview failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{
		"recharge_time":    at,
		"qualifying_draws": qualifying,
		"count":            len(qualifying),
	})
}

// ─── MTN Push CSV Bulk Upload ─────────────────────────────────────────────────
//
// When the MTN push webhook API is unavailable, admins can upload a CSV file
// to manually trigger the full recharge pipeline (spin credits, pulse points,
// draw entries) for each subscriber row.
//
// CSV format (header row required):
//
//	msisdn,date,time,amount[,recharge_type]
//
// date:          YYYY-MM-DD
// time:          HH:MM or HH:MM:SS  (WAT assumed)
// amount:        naira value (e.g. 1000 or 1000.00)
// recharge_type: optional; defaults to AIRTIME
//
// Routes:
//   POST   /api/v1/admin/mtn-push/csv-upload          — upload & process
//   GET    /api/v1/admin/mtn-push/csv-upload           — list batches
//   GET    /api/v1/admin/mtn-push/csv-upload/{id}      — batch summary
//   GET    /api/v1/admin/mtn-push/csv-upload/{id}/rows — per-row results

// UploadMTNPushCSV handles multipart/form-data CSV uploads.
// POST /api/v1/admin/mtn-push/csv-upload
func (h *AdminHandler) UploadMTNPushCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.csvSvc == nil {
		jsonError(w, "CSV upload service not configured", http.StatusServiceUnavailable)
		return
	}

	// Extract admin identity from JWT context.
	uploadedBy, _ := r.Context().Value(middleware.ContextUserID).(string)
	if uploadedBy == "" {
		uploadedBy = "unknown"
	}

	// Parse multipart form — limit to 10 MB.
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "field 'file' is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	note := r.FormValue("note")

	result, err := h.csvSvc.ProcessCSVUpload(ctx, services.CSVUploadRequest{
		UploadedBy: uploadedBy,
		Filename:   fileHeader.Filename,
		Reader:     file,
		Note:       note,
	})
	if err != nil {
		jsonError(w, "upload processing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return 207 Multi-Status when some rows failed, 200 when all succeeded.
	code := http.StatusOK
	if result.Status == "PARTIAL" || result.Status == "FAILED" {
		code = http.StatusMultiStatus
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if encErr := json.NewEncoder(w).Encode(result); encErr != nil {
		log.Printf("[Admin] UploadMTNPushCSV encode error: %v", encErr)
	}
}

// ListMTNPushCSVUploads returns recent upload batches.
// GET /api/v1/admin/mtn-push/csv-upload
func (h *AdminHandler) ListMTNPushCSVUploads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.csvSvc == nil {
		jsonError(w, "CSV upload service not configured", http.StatusServiceUnavailable)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	uploads, total, err := h.csvSvc.ListUploads(ctx, limit, offset)
	if err != nil {
		jsonError(w, "failed to list uploads: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{
		"total":   total,
		"uploads": uploads,
	})
}

// GetMTNPushCSVUpload returns the summary for a single upload batch.
// GET /api/v1/admin/mtn-push/csv-upload/{id}
func (h *AdminHandler) GetMTNPushCSVUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.csvSvc == nil {
		jsonError(w, "CSV upload service not configured", http.StatusServiceUnavailable)
		return
	}

	idStr := r.PathValue("id")
	uploadID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid upload id", http.StatusBadRequest)
		return
	}

	summary, err := h.csvSvc.GetUpload(ctx, uploadID)
	if err != nil {
		jsonError(w, "upload not found", http.StatusNotFound)
		return
	}
	jsonOK(w, summary)
}

// GetMTNPushCSVUploadRows returns per-row results for a batch.
// GET /api/v1/admin/mtn-push/csv-upload/{id}/rows
func (h *AdminHandler) GetMTNPushCSVUploadRows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.csvSvc == nil {
		jsonError(w, "CSV upload service not configured", http.StatusServiceUnavailable)
		return
	}

	idStr := r.PathValue("id")
	uploadID, err := uuid.Parse(idStr)
	if err != nil {
		jsonError(w, "invalid upload id", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	rows, total, err := h.csvSvc.GetUploadRows(ctx, uploadID, limit, offset)
	if err != nil {
		jsonError(w, "failed to get rows: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{
		"total": total,
		"rows":  rows,
	})
}

// ─── Bonus Pulse Point Awards ────────────────────────────────────────────────

// AwardBonusPulse awards bonus Pulse Points to a user by phone number.
// POST /api/v1/admin/bonus-pulse
//
// Request body:
//
//	{
//	  "phone_number": "08012345678",
//	  "points":       500,
//	  "campaign":     "Ramadan 2025",   // optional
//	  "note":         "Top-up for VIP"  // optional
//	}
func (h *AdminHandler) AwardBonusPulse(w http.ResponseWriter, r *http.Request) {
	if h.bonusPulseSvc == nil {
		jsonError(w, "bonus pulse service not configured", http.StatusServiceUnavailable)
		return
	}
	ctx := r.Context()

	// Extract admin identity from JWT claims.
	uidStr, ok := ctx.Value(middleware.ContextUserID).(string)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	adminID, err := uuid.Parse(uidStr)
	if err != nil {
		jsonError(w, "invalid admin id", http.StatusUnauthorized)
		return
	}

	// Resolve admin display name for the audit trail.
	// Note: users table has no full_name column — use phone_number as display name.
	var adminName string
	if dbErr := h.db.WithContext(ctx).
		Table("users").
		Where("id = ?", adminID).
		Select("COALESCE(phone_number, id::text)").
		Scan(&adminName).Error; dbErr != nil || adminName == "" {
		adminName = adminID.String()
	}

	var req services.AwardBonusPulseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.PhoneNumber == "" {
		jsonError(w, "phone_number is required", http.StatusBadRequest)
		return
	}
	if req.Points <= 0 {
		jsonError(w, "points must be greater than zero", http.StatusBadRequest)
		return
	}

	req.AwardedByID = adminID
	req.AwardedByName = adminName

	result, err := h.bonusPulseSvc.AwardBonusPulse(ctx, req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, result)
}

// ListBonusPulseAwards returns a paginated audit log of all bonus pulse point
// awards, optionally filtered by phone number and/or campaign name.
// GET /api/v1/admin/bonus-pulse?phone=&campaign=&limit=&offset=
func (h *AdminHandler) ListBonusPulseAwards(w http.ResponseWriter, r *http.Request) {
	if h.bonusPulseSvc == nil {
		jsonError(w, "bonus pulse service not configured", http.StatusServiceUnavailable)
		return
	}
	ctx := r.Context()
	phone := r.URL.Query().Get("phone")
	campaign := r.URL.Query().Get("campaign")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	records, total, err := h.bonusPulseSvc.ListAwards(ctx, phone, campaign, limit, offset)
	if err != nil {
		jsonError(w, "failed to list awards: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{
		"total":   total,
		"records": records,
	})
}
