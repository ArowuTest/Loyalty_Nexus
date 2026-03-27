package services

// draw_window_service.go — Draw eligibility window resolver
//
// Business rules (from product spec + spreadsheet):
//
//   Draw schedule (all times WAT = UTC+1):
//     Monday Draw    : Thu 17:00:01 → Sun 17:00:00
//     Tuesday Draw   : Sun 17:00:01 → Mon 17:00:00
//     Wednesday Draw : Mon 17:00:01 → Tue 17:00:00
//     Thursday Draw  : Tue 17:00:01 → Wed 17:00:00
//     Friday Draw    : Wed 17:00:01 → Thu 17:00:00
//     Saturday Mega  : Fri 17:00:01 → Fri 17:00:00 (full week)
//
//   When a recharge arrives at time T:
//     1. Load all active draw_schedules rows from DB (cached, refreshed on admin update)
//     2. For each schedule, compute the most recent eligibility window boundary
//        relative to T (in WAT timezone)
//     3. If T falls within [window_open, window_close), find the active draw
//        of that draw_type whose draw_time is the NEXT occurrence after T
//     4. Return all matching (drawID, drawType) pairs — typically 2:
//        one daily draw + the Saturday weekly mega draw
//
//   Spin credits are SEPARATE and IMMEDIATE — they are not affected by draw windows.
//   Draw entries reset per draw — they do NOT accumulate across draws.

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WAT is West Africa Time (UTC+1), used for all draw window boundary calculations.
var WAT = time.FixedZone("WAT", 1*60*60)

// DrawSchedule mirrors the draw_schedules table.
type DrawSchedule struct {
	ID              uuid.UUID `gorm:"column:id;primaryKey"`
	DrawName        string    `gorm:"column:draw_name"`
	DrawType        string    `gorm:"column:draw_type"`
	DrawDayOfWeek   int       `gorm:"column:draw_day_of_week"` // 0=Sun … 6=Sat
	DrawTimeWAT     string    `gorm:"column:draw_time_wat"`    // "HH:MM:SS"
	WindowOpenDOW   int       `gorm:"column:window_open_dow"`
	WindowOpenTime  string    `gorm:"column:window_open_time"`  // "HH:MM:SS"
	WindowCloseDOW  int       `gorm:"column:window_close_dow"`
	WindowCloseTime string    `gorm:"column:window_close_time"` // "HH:MM:SS"
	CutoffHourUTC   int       `gorm:"column:cutoff_hour_utc"`
	IsActive        bool      `gorm:"column:is_active"`
	SortOrder       int       `gorm:"column:sort_order"`
}

func (DrawSchedule) TableName() string { return "draw_schedules" }

// QualifyingDraw is a draw that a recharge at a given time qualifies for.
type QualifyingDraw struct {
	DrawID   uuid.UUID
	DrawType string // DAILY | WEEKLY
	DrawName string
}

// DrawWindowService resolves which active draws a recharge qualifies for
// based on the configurable window rules in draw_schedules.
type DrawWindowService struct {
	db *gorm.DB

	// In-memory cache of draw schedules — refreshed on admin update.
	mu        sync.RWMutex
	schedules []DrawSchedule
	loadedAt  time.Time
	cacheTTL  time.Duration
}

func NewDrawWindowService(db *gorm.DB) *DrawWindowService {
	return &DrawWindowService{
		db:       db,
		cacheTTL: 5 * time.Minute,
	}
}

// InvalidateCache forces the next call to ResolveQualifyingDraws to reload
// schedules from the DB. Called by admin update endpoints.
func (s *DrawWindowService) InvalidateCache() {
	s.mu.Lock()
	s.loadedAt = time.Time{}
	s.mu.Unlock()
}

// ─── Admin request/response types ────────────────────────────────────────

// CreateDrawScheduleRequest is the body for POST /api/v1/admin/draw/schedule.
type CreateDrawScheduleRequest struct {
	DrawName       string `json:"draw_name"`
	DrawType       string `json:"draw_type"`        // DAILY | WEEKLY
	DrawDayOfWeek  int    `json:"draw_day_of_week"` // 0=Sun … 6=Sat
	DrawTimeWAT    string `json:"draw_time_wat"`    // "HH:MM:SS"
	WindowOpenDOW  int    `json:"window_open_dow"`
	WindowOpenTime string `json:"window_open_time"`  // "HH:MM:SS"
	WindowCloseDOW  int   `json:"window_close_dow"`
	WindowCloseTime string `json:"window_close_time"` // "HH:MM:SS"
	CutoffHourUTC  int    `json:"cutoff_hour_utc"`
	SortOrder      int    `json:"sort_order"`
	IsActive       bool   `json:"is_active"`
}

// UpdateDrawScheduleRequest is the body for PUT /api/v1/admin/draw/schedule/{id}.
// All fields are optional — only non-nil pointer fields are applied.
type UpdateDrawScheduleRequest struct {
	DrawName        *string `json:"draw_name"`
	DrawType        *string `json:"draw_type"`
	DrawDayOfWeek   *int    `json:"draw_day_of_week"`
	DrawTimeWAT     *string `json:"draw_time_wat"`
	WindowOpenDOW   *int    `json:"window_open_dow"`
	WindowOpenTime  *string `json:"window_open_time"`
	WindowCloseDOW  *int    `json:"window_close_dow"`
	WindowCloseTime *string `json:"window_close_time"`
	CutoffHourUTC   *int    `json:"cutoff_hour_utc"`
	SortOrder       *int    `json:"sort_order"`
	IsActive        *bool   `json:"is_active"`
}

// GetAllSchedules returns all draw schedules (for admin UI), including inactive ones.
func (s *DrawWindowService) GetAllSchedules(ctx context.Context) ([]DrawSchedule, error) {
	var schedules []DrawSchedule
	err := s.db.WithContext(ctx).
		Order("sort_order ASC").
		Find(&schedules).Error
	return schedules, err
}

// GetSchedules returns only active draw schedules.
// Kept for backward compatibility with internal callers.
func (s *DrawWindowService) GetSchedules(ctx context.Context) ([]DrawSchedule, error) {
	var schedules []DrawSchedule
	err := s.db.WithContext(ctx).
		Where("is_active = true").
		Order("sort_order ASC").
		Find(&schedules).Error
	return schedules, err
}

// UpdateSchedule updates a single draw schedule row by UUID and invalidates the cache.
func (s *DrawWindowService) UpdateSchedule(ctx context.Context, id uuid.UUID, req UpdateDrawScheduleRequest) (*DrawSchedule, error) {
	updates := map[string]interface{}{}
	if req.DrawName != nil        { updates["draw_name"] = *req.DrawName }
	if req.DrawType != nil        { updates["draw_type"] = *req.DrawType }
	if req.DrawDayOfWeek != nil   { updates["draw_day_of_week"] = *req.DrawDayOfWeek }
	if req.DrawTimeWAT != nil     { updates["draw_time_wat"] = *req.DrawTimeWAT }
	if req.WindowOpenDOW != nil   { updates["window_open_dow"] = *req.WindowOpenDOW }
	if req.WindowOpenTime != nil  { updates["window_open_time"] = *req.WindowOpenTime }
	if req.WindowCloseDOW != nil  { updates["window_close_dow"] = *req.WindowCloseDOW }
	if req.WindowCloseTime != nil { updates["window_close_time"] = *req.WindowCloseTime }
	if req.CutoffHourUTC != nil   { updates["cutoff_hour_utc"] = *req.CutoffHourUTC }
	if req.SortOrder != nil       { updates["sort_order"] = *req.SortOrder }
	if req.IsActive != nil        { updates["is_active"] = *req.IsActive }

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields provided to update")
	}

	if err := s.db.WithContext(ctx).
		Model(&DrawSchedule{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update draw schedule: %w", err)
	}
	s.InvalidateCache()

	// Return the updated row.
	var updated DrawSchedule
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&updated).Error; err != nil {
		return nil, fmt.Errorf("fetch updated schedule: %w", err)
	}
	return &updated, nil
}

// CreateSchedule inserts a new draw schedule row.
func (s *DrawWindowService) CreateSchedule(ctx context.Context, req CreateDrawScheduleRequest) (*DrawSchedule, error) {
	if req.DrawName == "" {
		return nil, fmt.Errorf("draw_name is required")
	}
	if req.DrawType == "" {
		return nil, fmt.Errorf("draw_type is required")
	}
	if req.DrawTimeWAT == "" {
		return nil, fmt.Errorf("draw_time_wat is required (HH:MM:SS)")
	}
	if req.WindowOpenTime == "" || req.WindowCloseTime == "" {
		return nil, fmt.Errorf("window_open_time and window_close_time are required (HH:MM:SS)")
	}
	// Validate time format.
	for _, t := range []string{req.DrawTimeWAT, req.WindowOpenTime, req.WindowCloseTime} {
		if _, err := parseTimeOfDay(t); err != nil {
			return nil, fmt.Errorf("invalid time format %q: use HH:MM:SS", t)
		}
	}

	sched := &DrawSchedule{
		ID:              uuid.New(),
		DrawName:        req.DrawName,
		DrawType:        req.DrawType,
		DrawDayOfWeek:   req.DrawDayOfWeek,
		DrawTimeWAT:     req.DrawTimeWAT,
		WindowOpenDOW:   req.WindowOpenDOW,
		WindowOpenTime:  req.WindowOpenTime,
		WindowCloseDOW:  req.WindowCloseDOW,
		WindowCloseTime: req.WindowCloseTime,
		CutoffHourUTC:   req.CutoffHourUTC,
		SortOrder:       req.SortOrder,
		IsActive:        req.IsActive,
	}
	if err := s.db.WithContext(ctx).Create(sched).Error; err != nil {
		return nil, fmt.Errorf("create draw schedule: %w", err)
	}
	s.InvalidateCache()
	return sched, nil
}

// DeleteSchedule soft-deletes a draw schedule by setting is_active=false.
func (s *DrawWindowService) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	if err := s.db.WithContext(ctx).
		Model(&DrawSchedule{}).
		Where("id = ?", id).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("delete draw schedule: %w", err)
	}
	s.InvalidateCache()
	return nil
}

// ResolveQualifyingDraws returns the list of active draws that a recharge
// arriving at rechargeTime qualifies for.
//
// A recharge qualifies for a draw if:
//   1. The recharge timestamp falls within the draw's eligibility window
//   2. There is an active (ACTIVE or SCHEDULED) draw of that draw_type
//      whose draw_time is in the future relative to rechargeTime
//
// Typically returns 2 draws: one DAILY + one WEEKLY (Saturday mega).
// Returns an empty slice (not an error) if no active draws are found —
// the caller should log this and continue without draw entries.
func (s *DrawWindowService) ResolveQualifyingDraws(ctx context.Context, rechargeTime time.Time) ([]QualifyingDraw, error) {
	schedules, err := s.loadSchedules(ctx)
	if err != nil {
		return nil, fmt.Errorf("load draw schedules: %w", err)
	}

	// Convert recharge time to WAT for day-of-week and time comparisons.
	rechargeWAT := rechargeTime.In(WAT)

	var qualifying []QualifyingDraw
	seen := make(map[string]bool) // deduplicate by draw_type

	for _, sched := range schedules {
		if !sched.IsActive {
			continue
		}

		inWindow, err := s.isInWindow(rechargeWAT, sched)
		if err != nil {
			log.Printf("[DrawWindow] schedule %q window check error: %v", sched.DrawName, err)
			continue
		}
		if !inWindow {
			continue
		}

		// Deduplicate: only one draw per draw_type per recharge.
		// (e.g. if two DAILY schedules both match, only take the first.)
		if seen[sched.DrawType] {
			continue
		}

		// Find the next active draw of this type.
		draw, err := s.findNextActiveDraw(ctx, sched.DrawType, rechargeTime)
		if err != nil || draw == nil {
			// No active draw of this type — skip silently.
			log.Printf("[DrawWindow] no active %s draw found for recharge at %s", sched.DrawType, rechargeWAT.Format(time.RFC3339))
			continue
		}

		qualifying = append(qualifying, QualifyingDraw{
			DrawID:   draw.ID,
			DrawType: draw.DrawType,
			DrawName: draw.Name,
		})
		seen[sched.DrawType] = true
	}

	return qualifying, nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────

// isInWindow returns true if rechargeWAT falls within the schedule's
// eligibility window for the current week.
//
// The window is defined by (window_open_dow, window_open_time) →
// (window_close_dow, window_close_time) in WAT.
//
// Because windows can span the week boundary (e.g. Thu → Sun for Monday draw),
// we compute the absolute window boundaries relative to the current week
// and handle the wrap-around case.
func (s *DrawWindowService) isInWindow(rechargeWAT time.Time, sched DrawSchedule) (bool, error) {
	openTime, err := parseTimeOfDay(sched.WindowOpenTime)
	if err != nil {
		return false, fmt.Errorf("parse window_open_time %q: %w", sched.WindowOpenTime, err)
	}
	closeTime, err := parseTimeOfDay(sched.WindowCloseTime)
	if err != nil {
		return false, fmt.Errorf("parse window_close_time %q: %w", sched.WindowCloseTime, err)
	}

	// Find the most recent occurrence of window_open_dow at window_open_time
	// that is on or before rechargeWAT.
	windowOpen := mostRecentDOW(rechargeWAT, sched.WindowOpenDOW, openTime)

	// Find the most recent occurrence of window_close_dow at window_close_time
	// that is AFTER windowOpen.
	windowClose := nextDOWAfter(windowOpen, sched.WindowCloseDOW, closeTime)

	// rechargeWAT must be strictly after windowOpen and on or before windowClose.
	return rechargeWAT.After(windowOpen) && !rechargeWAT.After(windowClose), nil
}

// mostRecentDOW returns the most recent datetime (on or before ref) that
// falls on dayOfWeek at timeOfDay (in WAT).
func mostRecentDOW(ref time.Time, dayOfWeek int, tod timeOfDay) time.Time {
	// Start from today at the given time.
	candidate := time.Date(ref.Year(), ref.Month(), ref.Day(),
		tod.hour, tod.minute, tod.second, 0, WAT)

	// Walk backwards until we hit the right day of week.
	for int(candidate.Weekday()) != dayOfWeek {
		candidate = candidate.AddDate(0, 0, -1)
	}

	// If candidate is after ref, go back one more week.
	if candidate.After(ref) {
		candidate = candidate.AddDate(0, 0, -7)
	}

	return candidate
}

// nextDOWAfter returns the next datetime strictly after `after` that falls
// on dayOfWeek at timeOfDay (in WAT).
func nextDOWAfter(after time.Time, dayOfWeek int, tod timeOfDay) time.Time {
	candidate := time.Date(after.Year(), after.Month(), after.Day(),
		tod.hour, tod.minute, tod.second, 0, WAT)

	// Walk forward until we hit the right day of week.
	for int(candidate.Weekday()) != dayOfWeek {
		candidate = candidate.AddDate(0, 0, 1)
	}

	// Must be strictly after `after`.
	if !candidate.After(after) {
		candidate = candidate.AddDate(0, 0, 7)
	}

	return candidate
}

// findNextActiveDraw finds the next ACTIVE or SCHEDULED draw of the given
// draw_type whose draw_time is in the future (or whose end_time is in the future).
func (s *DrawWindowService) findNextActiveDraw(ctx context.Context, drawType string, after time.Time) (*DrawRecord, error) {
	var draw DrawRecord
	err := s.db.WithContext(ctx).
		Where("draw_type = ? AND status IN ('ACTIVE','SCHEDULED') AND end_time > ?", drawType, after).
		Order("draw_time ASC NULLS LAST, end_time ASC").
		First(&draw).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &draw, nil
}

// loadSchedules returns cached schedules, reloading from DB if the cache is stale.
func (s *DrawWindowService) loadSchedules(ctx context.Context) ([]DrawSchedule, error) {
	s.mu.RLock()
	if time.Since(s.loadedAt) < s.cacheTTL && len(s.schedules) > 0 {
		schedules := s.schedules
		s.mu.RUnlock()
		return schedules, nil
	}
	s.mu.RUnlock()

	// Reload from DB.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock.
	if time.Since(s.loadedAt) < s.cacheTTL && len(s.schedules) > 0 {
		return s.schedules, nil
	}

	// Use context.Background() for the DB reload — schedule data is global config
	// and must not be tied to a request context (which may be a rolled-back txdb
	// in tests, or a short-lived HTTP context in production).
	var schedules []DrawSchedule
	if err := s.db.WithContext(context.Background()).
		Where("is_active = true").
		Order("sort_order ASC").
		Find(&schedules).Error; err != nil {
		return nil, err
	}

	s.schedules = schedules
	s.loadedAt = time.Now()
	return schedules, nil
}

// ─── Time parsing helpers ─────────────────────────────────────────────────

type timeOfDay struct {
	hour, minute, second int
}

func parseTimeOfDay(s string) (timeOfDay, error) {
	var h, m, sec int
	if _, err := fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec); err != nil {
		return timeOfDay{}, fmt.Errorf("invalid time %q: %w", s, err)
	}
	return timeOfDay{h, m, sec}, nil
}

// DrawRecord.Name field — add a Name field alias since the struct uses Name
// but the DB column is 'name'. The DrawRecord struct in draw_service.go
// already has gorm:"column:name" so this is just a convenience accessor.
// (No changes needed to DrawRecord — it already has Name string.)
