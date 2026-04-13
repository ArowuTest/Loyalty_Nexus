package services

// draw_service.go — full draw management engine
// Ported from RechargeMax draw_service.go and adapted for Loyalty Nexus:
//   - Uses loyalty-nexus module paths
//   - Uses net/http (not Gin)
//   - Uses Nexus entities (User, Wallet, Transaction)
//   - Adds recurrence + next_draw_at fields (spec §4)
//   - crypto/rand Fisher-Yates shuffle for winner selection (SEC-009)

import (
	"context"
	"encoding/csv"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── Draw entities (local to service — no separate repo interface needed) ────

// DrawRecord mirrors the draws table.
type DrawRecord struct {
	ID              uuid.UUID  `gorm:"column:id;primaryKey" json:"id"`
	Name            string     `gorm:"column:name" json:"name"`
	DrawCode        string     `gorm:"column:draw_code;uniqueIndex" json:"draw_code"`
	DrawType        string     `gorm:"column:draw_type" json:"draw_type"` // DAILY | WEEKLY | MONTHLY | SPECIAL
	Status          string     `gorm:"column:status" json:"status"`   // UPCOMING | ACTIVE | COMPLETED | CANCELLED
	PrizePool       float64    `gorm:"column:prize_pool" json:"prize_pool_kobo"`
	WinnerCount     int        `gorm:"column:winner_count" json:"winner_count"`
	RunnerUpsCount  int        `gorm:"column:runner_ups_count" json:"runner_ups_count"`
	TotalEntries    int        `gorm:"column:total_entries" json:"entry_count"`
	TotalWinners    int        `gorm:"column:total_winners" json:"total_winners"`
	Recurrence      string     `gorm:"column:recurrence" json:"recurrence"` // none | daily | weekly | monthly
	NextDrawAt      *time.Time `gorm:"column:next_draw_at" json:"next_draw_at,omitempty"`
	StartTime       time.Time  `gorm:"column:start_time" json:"start_time"`
	EndTime         time.Time  `gorm:"column:end_time" json:"end_time"`
	DrawTime        *time.Time `gorm:"column:draw_time" json:"draw_date,omitempty"`
	ExecutedAt      *time.Time `gorm:"column:executed_at" json:"executed_at,omitempty"`
	CompletedAt     *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (DrawRecord) TableName() string { return "draws" }

// DrawEntry mirrors draw_entries table.
type DrawEntry struct {
	ID           uuid.UUID  `gorm:"column:id;primaryKey"`
	DrawID       uuid.UUID  `gorm:"column:draw_id;index"`
	UserID       uuid.UUID  `gorm:"column:user_id;index"`
	MSISDN       string     `gorm:"column:msisdn"`        // writable column (migration 016)
	PhoneNumber  string     `gorm:"-"`                    // GENERATED ALWAYS AS (msisdn) — never write
	EntrySource  string     `gorm:"-"`                    // may not exist in all schemas — use raw SQL
	Amount       int64      `gorm:"-"`                    // may not exist in all schemas — use raw SQL
	EntriesCount int        `gorm:"column:entries_count"` // writable
	TicketCount  int        `gorm:"-"`                    // GENERATED ALWAYS AS (entries_count) — never write
	CreatedAt    *time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (DrawEntry) TableName() string { return "draw_entries" }

// DrawWinner mirrors draw_winners table (schema from migration 060).
type DrawWinner struct {
	ID             uuid.UUID  `gorm:"column:id;primaryKey"`
	DrawID         uuid.UUID  `gorm:"column:draw_id;index"`
	UserID         uuid.UUID  `gorm:"column:user_id;index"`
	PhoneNumber    string     `gorm:"column:phone_number"`
	Position       int        `gorm:"column:position"`
	PrizeValueKobo int64      `gorm:"column:prize_value_kobo"`
	Status         string     `gorm:"column:status"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (DrawWinner) TableName() string { return "draw_winners" }

// ─── Response types ────────────────────────────────────────────────────────

type DrawEntryExport struct {
	PhoneNumber string
	Points      int64
}

type WinnerImport struct {
	PhoneNumber string
	Position    int
	Prize       string
	Amount      int64
}

type DrawWinnerResponse struct {
	ID          uuid.UUID `json:"id"`
	DrawID      uuid.UUID `json:"draw_id"`
	UserID      uuid.UUID `json:"user_id"`
	PhoneNumber string    `json:"phone_number"`
	Position    int       `json:"position"`
	PrizeType   string    `json:"prize_type"`
	PrizeValue  float64   `json:"prize_value"`
	IsRunnerUp  bool      `json:"is_runner_up"`
	Status      string    `json:"status"`
	WonAt       time.Time `json:"won_at"`
}

// ─── Service ──────────────────────────────────────────────────────────────

type DrawService struct {
	db *gorm.DB
}

func NewDrawService(db *gorm.DB) *DrawService {
	return &DrawService{db: db}
}

func (svc *DrawService) DB() *gorm.DB { return svc.db }

// ─── Draw code generator ──────────────────────────────────────────────────

func generateDrawCode() string {
	var b [8]byte
	crand.Read(b[:]) //nolint:errcheck
	n := int(binary.BigEndian.Uint64(b[:]) % 9000)
	return fmt.Sprintf("DRAW-%s-%04d", time.Now().Format("20060102"), n+1000)
}

// ─── CRUD ─────────────────────────────────────────────────────────────────

// CreateDraw creates a new draw record.
func (svc *DrawService) CreateDraw(
	ctx context.Context,
	name, description, drawType, recurrence string,
	drawDate time.Time,
	prizePool float64,
	winnerCount, runnerUpsCount int,
) (*DrawRecord, error) {
	if name == "" {
		return nil, fmt.Errorf("draw name is required")
	}
	if drawType == "" {
		drawType = "MONTHLY"
	}
	validStatuses := map[string]bool{"DAILY": true, "WEEKLY": true, "MONTHLY": true, "SPECIAL": true}
	if !validStatuses[drawType] {
		drawType = "MONTHLY"
	}
	if winnerCount < 1 {
		winnerCount = 1
	}
	if recurrence == "" {
		recurrence = "none"
	}

	draw := &DrawRecord{
		ID:             uuid.New(),
		DrawCode:       generateDrawCode(),
		Name:           name,
		DrawType:       drawType,
		Status:         "UPCOMING", // DB CHECK constraint: UPCOMING | ACTIVE | COMPLETED | CANCELLED
		PrizePool:      prizePool,
		WinnerCount:    winnerCount,
		RunnerUpsCount: runnerUpsCount,
		Recurrence:     recurrence,
		StartTime:      drawDate.Add(-24 * time.Hour),
		EndTime:        drawDate,
		DrawTime:       timePtr(drawDate),
	}

	// If recurring, set NextDrawAt
	if recurrence != "none" {
		next := nextDrawTime(drawDate, recurrence)
		draw.NextDrawAt = timePtr(next)
	}

	if err := svc.db.WithContext(ctx).Create(draw).Error; err != nil {
		return nil, fmt.Errorf("failed to create draw: %w", err)
	}
	return draw, nil
}

// GetDraws returns all draws with pagination.
func (svc *DrawService) GetDraws(ctx context.Context, page, limit int) ([]DrawRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var draws []DrawRecord
	var total int64

	if err := svc.db.WithContext(ctx).Model(&DrawRecord{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count draws: %w", err)
	}
	if err := svc.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&draws).Error; err != nil {
		return nil, 0, fmt.Errorf("list draws: %w", err)
	}
	return draws, total, nil
}

// GetDrawByID returns a draw by ID.
func (svc *DrawService) GetDrawByID(ctx context.Context, drawID uuid.UUID) (*DrawRecord, error) {
	var draw DrawRecord
	if err := svc.db.WithContext(ctx).Where("id = ?", drawID).First(&draw).Error; err != nil {
		return nil, fmt.Errorf("draw not found: %w", err)
	}
	return &draw, nil
}

// UpdateDraw updates mutable fields of a draw.
func (svc *DrawService) UpdateDraw(ctx context.Context, drawID uuid.UUID, updates map[string]interface{}) (*DrawRecord, error) {
	draw, err := svc.GetDrawByID(ctx, drawID)
	if err != nil {
		return nil, err
	}

	// Build safe update map (never touch draw_code, id, created_at)
	safe := map[string]interface{}{}
	if v, ok := updates["name"].(string); ok && v != "" {
		safe["name"] = v
		draw.Name = v
	}
	if v, ok := updates["status"].(string); ok {
		allowed := map[string]bool{"UPCOMING": true, "ACTIVE": true, "COMPLETED": true, "CANCELLED": true}
		if allowed[v] {
			safe["status"] = v
			draw.Status = v
		}
	}
	if v, ok := updates["prize_pool"].(float64); ok {
		safe["prize_pool"] = v
		draw.PrizePool = v
	}
	if v, ok := updates["winner_count"].(float64); ok {
		safe["winner_count"] = int(v)
		draw.WinnerCount = int(v)
	}
	if v, ok := updates["runner_ups_count"].(float64); ok {
		safe["runner_ups_count"] = int(v)
		draw.RunnerUpsCount = int(v)
	}
	if v, ok := updates["draw_time"].(string); ok {
		if t, err2 := time.Parse(time.RFC3339, v); err2 == nil {
			safe["draw_time"] = t
			draw.DrawTime = timePtr(t)
		}
	}
	if v, ok := updates["recurrence"].(string); ok {
		safe["recurrence"] = v
		draw.Recurrence = v
	}

	if len(safe) == 0 {
		return draw, nil
	}
	if err := svc.db.WithContext(ctx).Model(draw).Updates(safe).Error; err != nil {
		return nil, fmt.Errorf("update draw: %w", err)
	}
	return draw, nil
}

// DeleteDraw cancels (soft-deletes) a draw.
func (svc *DrawService) DeleteDraw(ctx context.Context, drawID uuid.UUID) error {
	return svc.db.WithContext(ctx).
		Table("draws").
		Where("id = ? AND status NOT IN ('COMPLETED')", drawID).
		Updates(map[string]interface{}{
			"status":     "CANCELLED",
			"updated_at": time.Now(),
		}).Error
}

// ─── Execution ────────────────────────────────────────────────────────────

// ExecuteDraw selects winners for a draw using CSPRNG Fisher-Yates (SEC-009).
func (svc *DrawService) ExecuteDraw(ctx context.Context, drawID uuid.UUID) error {
	return svc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1 — fetch draw
		var draw DrawRecord
		if err := tx.Where("id = ? AND status IN ('UPCOMING','ACTIVE')", drawID).First(&draw).Error; err != nil {
			return fmt.Errorf("draw not found or not executable: %w", err)
		}

		// 2 — load entries (select only universally-present columns)
		// Schema A: msisdn + entries_count; Schema B: phone_number + ticket_count
		var entries []DrawEntry
		loadErr := tx.Select("id, draw_id, user_id, msisdn, entries_count, created_at").Where("draw_id = ?", drawID).Find(&entries).Error
		if loadErr != nil {
			// Schema B fallback: map phone_number -> MSISDN, ticket_count -> EntriesCount
			type entryB struct {
				ID          string `gorm:"column:id"`
				DrawID      string `gorm:"column:draw_id"`
				UserID      string `gorm:"column:user_id"`
				PhoneNumber string `gorm:"column:phone_number"`
				TicketCount int    `gorm:"column:ticket_count"`
			}
			var rawB []entryB
			if err2 := tx.Table("draw_entries").Select("id, draw_id, user_id, phone_number, ticket_count").Where("draw_id = ?", drawID).Find(&rawB).Error; err2 != nil {
				return fmt.Errorf("load entries: %w", err2)
			}
			for _, e := range rawB {
				uid, _ := uuid.Parse(e.UserID)
				did, _ := uuid.Parse(e.DrawID)
				entries = append(entries, DrawEntry{UserID: uid, DrawID: did, MSISDN: e.PhoneNumber, EntriesCount: e.TicketCount})
			}
		}
		if len(entries) == 0 {
			return fmt.Errorf("no entries for draw %s", drawID)
		}

		// 3 — build weighted ticket pool (each entry.TicketCount multiplied)
		pool := make([]DrawEntry, 0, len(entries)*2)
		for _, e := range entries {
			count := e.EntriesCount
			if count < 1 {
				count = 1
			}
			if count > 100 {
				count = 100 // cap to prevent pool explosion
			}
			for i := 0; i < count; i++ {
				pool = append(pool, e)
			}
		}

		// 4 — crypto shuffle
		drawShuffleCrypto(pool)

		// 5 — pick deduplicated winners
		winnerCount := draw.WinnerCount
		if winnerCount < 1 {
			winnerCount = 1
		}
		runnerUpCount := draw.RunnerUpsCount

		seen := make(map[uuid.UUID]bool)
		mainWinners := make([]DrawEntry, 0, winnerCount)
		runnerUps := make([]DrawEntry, 0, runnerUpCount)

		for _, ticket := range pool {
			if seen[ticket.UserID] {
				continue
			}
			seen[ticket.UserID] = true
			if len(mainWinners) < winnerCount {
				mainWinners = append(mainWinners, ticket)
			} else if len(runnerUps) < runnerUpCount {
				runnerUps = append(runnerUps, ticket)
			} else {
				break
			}
		}

		now := time.Now()

		// 6 — insert winners
		position := 1
		for _, w := range mainWinners {
			row := DrawWinner{
				ID:             uuid.New(),
				DrawID:         drawID,
				UserID:         w.UserID,
				PhoneNumber:    w.MSISDN,
				Position:       position,
				PrizeValueKobo: int64(draw.PrizePool * 100), // convert to kobo
				Status:         "PENDING_FULFILLMENT",
				CreatedAt:      now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("insert winner position %d: %w", position, err)
			}
			position++
		}
		for _, w := range runnerUps {
			row := DrawWinner{
				ID:             uuid.New(),
				DrawID:         drawID,
				UserID:         w.UserID,
				PhoneNumber:    w.MSISDN,
				Position:       position,
				PrizeValueKobo: 0,
				Status:         "RUNNER_UP",
				CreatedAt:      now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("insert runner-up position %d: %w", position, err)
			}
			position++
		}

		// 7 — update draw record
		updates := map[string]interface{}{
			"status":       "COMPLETED",
			"total_winners": len(mainWinners) + len(runnerUps),
			"executed_at":  now,
			"completed_at": now,
			"updated_at":   now,
		}
		// If recurring: schedule next draw
		if draw.Recurrence != "" && draw.Recurrence != "none" {
			baseTime := now
			if draw.DrawTime != nil {
				baseTime = *draw.DrawTime
			}
			next := nextDrawTime(baseTime, draw.Recurrence)
			updates["next_draw_at"] = next
			// Spawn a new draw for the next cycle
				nextDraw := DrawRecord{
					ID:             uuid.New(),
					DrawCode:       generateDrawCode(),
					Name:           draw.Name,
					DrawType:       draw.DrawType,
					Status:         "UPCOMING",
					PrizePool:      draw.PrizePool,
					WinnerCount:    draw.WinnerCount,
					RunnerUpsCount: draw.RunnerUpsCount,
					Recurrence:     draw.Recurrence,
					StartTime:      next.Add(-24 * time.Hour),
					EndTime:        next,
					DrawTime:       timePtr(next),
					CreatedAt:      now,
				}
			_ = tx.Create(&nextDraw).Error // non-fatal if this fails
		}
		return tx.Table("draws").Where("id = ?", drawID).Updates(updates).Error
	})
}

// ─── Entries ──────────────────────────────────────────────────────────────

// AddEntry adds a single draw entry for a user.
// Uses raw INSERT to handle schema variance across deployments:
//   - Guaranteed columns (migration 016 + 060): id, draw_id, user_id, msisdn, entries_count, created_at
//   - Optional columns (migration 045+): entry_source, amount
func (svc *DrawService) AddEntry(ctx context.Context, drawID, userID uuid.UUID, phone, source string, amount int64, tickets int) error {
	if tickets < 1 {
		tickets = 1
	}
	now := time.Now()
	// Try inserts in order of richest → minimal schema.
	// Different deployments have different draw_entries schemas:
	//   Schema A (016+045+049): msisdn, entries_count, entry_source, amount (phone_number/ticket_count are GENERATED)
	//   Schema B (060):         phone_number, ticket_count (regular columns, no msisdn, no entry_source)
	var insertErr error
	// Attempt 1: Schema A — full columns including entry_source
	insertErr = svc.db.WithContext(ctx).Exec(`
		INSERT INTO draw_entries (id, draw_id, user_id, msisdn, entries_count, entry_source, amount, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New(), drawID, userID, phone, tickets, source, amount, now,
	).Error
	if insertErr != nil {
		// Attempt 2: Schema A minimal (no entry_source/amount)
		insertErr = svc.db.WithContext(ctx).Exec(`
			INSERT INTO draw_entries (id, draw_id, user_id, msisdn, entries_count, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.New(), drawID, userID, phone, tickets, now,
		).Error
	}
	if insertErr != nil {
		// Attempt 3: Schema B (migration 060) — phone_number + ticket_count are regular columns
		insertErr = svc.db.WithContext(ctx).Exec(`
			INSERT INTO draw_entries (id, draw_id, user_id, phone_number, ticket_count, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.New(), drawID, userID, phone, tickets, now,
		).Error
	}
	if insertErr != nil {
		return fmt.Errorf("add entry: %w", insertErr)
	}
	// Update total_entries on the draw
	return svc.db.Exec("UPDATE draws SET total_entries = total_entries + ? WHERE id = ?", tickets, drawID).Error
}

// GetDrawEntries returns all entries for a draw (paginated).
func (svc *DrawService) GetDrawEntries(ctx context.Context, drawID uuid.UUID, page, limit int) ([]DrawEntry, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	var entries []DrawEntry
	var total int64
	svc.db.WithContext(ctx).Model(&DrawEntry{}).Where("draw_id = ?", drawID).Count(&total)
	err := svc.db.WithContext(ctx).
		Where("draw_id = ?", drawID).
		Order("created_at ASC").
		Limit(limit).
		Offset((page-1)*limit).
		Find(&entries).Error
	return entries, total, err
}

// ─── Winners ──────────────────────────────────────────────────────────────

// GetDrawWinners returns winners for a completed draw.
func (svc *DrawService) GetDrawWinners(ctx context.Context, drawID uuid.UUID) ([]DrawWinnerResponse, error) {
	var winners []DrawWinner
	if err := svc.db.WithContext(ctx).
		Where("draw_id = ?", drawID).
		Order("position ASC").
		Find(&winners).Error; err != nil {
		return nil, fmt.Errorf("get winners: %w", err)
	}

	resp := make([]DrawWinnerResponse, len(winners))
	for i, w := range winners {
		resp[i] = DrawWinnerResponse{
			ID:          w.ID,
			DrawID:      w.DrawID,
			UserID:      w.UserID,
			PhoneNumber: w.PhoneNumber,
			Position:    w.Position,
			PrizeType:   "CASH",
			PrizeValue:  float64(w.PrizeValueKobo) / 100, // convert kobo → naira
			IsRunnerUp:  w.Status == "RUNNER_UP",
			Status:      w.Status,
			WonAt:       w.CreatedAt,
		}
	}
	return resp, nil
}

// ─── CSV Export / Import ──────────────────────────────────────────────────

// ExportDrawEntries exports draw entries as CSV.
func (svc *DrawService) ExportDrawEntries(ctx context.Context, drawID uuid.UUID, outputPath string) (string, error) {
	type row struct {
		PhoneNumber string `gorm:"column:phone_number"`
		TotalTickets int64 `gorm:"column:total_tickets"`
	}
	var rows []row
	err := svc.db.WithContext(ctx).
		Table("draw_entries").
		Select("phone_number, COALESCE(SUM(ticket_count), 0) AS total_tickets").
		Where("draw_id = ?", drawID).
		Group("phone_number").
		Order("total_tickets DESC").
		Scan(&rows).Error
	if err != nil {
		return "", fmt.Errorf("aggregate entries: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create CSV: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("[DrawService] ExportEligibleEntries: file close: %v", err)
		}
	}()

	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"PhoneNumber", "TotalTickets"})
	for _, r := range rows {
		_ = w.Write([]string{r.PhoneNumber, strconv.FormatInt(r.TotalTickets, 10)})
	}
	return outputPath, nil
}

// ProcessCSVEntries bulk-imports entries from a CSV reader.
// CSV format: PhoneNumber,Tickets
func (svc *DrawService) ProcessCSVEntries(ctx context.Context, drawID uuid.UUID, r io.Reader) (int, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	created := 0
	lineNum := 0
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return created, fmt.Errorf("read CSV line %d: %w", lineNum, err)
		}
		lineNum++
		if lineNum == 1 && (strings.ToLower(rec[0]) == "phonenumber" || strings.ToLower(rec[0]) == "phone_number") {
			continue
		}
		if len(rec) < 2 {
			continue
		}
		phone := strings.TrimSpace(rec[0])
		tickets, _ := strconv.Atoi(strings.TrimSpace(rec[1]))
		if tickets < 1 {
			tickets = 1
		}
		now := time.Now()
		entry := DrawEntry{
			ID:          uuid.New(),
			DrawID:      drawID,
			PhoneNumber: phone,
			EntrySource: "csv_import",
			EntriesCount: tickets,
			CreatedAt:   &now,
		}
		if err := svc.db.WithContext(ctx).Create(&entry).Error; err != nil {
			return created, fmt.Errorf("insert entry at line %d: %w", lineNum, err)
		}
		created += tickets
	}
	// Update total_entries
	_ = svc.db.Exec("UPDATE draws SET total_entries = total_entries + ? WHERE id = ?", created, drawID).Error
	return created, nil
}

// ImportWinners imports winners from CSV (for external draw systems).
func (svc *DrawService) ImportWinners(ctx context.Context, drawID uuid.UUID, csvPath string) ([]*WinnerImport, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("open CSV: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("[DrawService] ImportWinners: file close: %v", err)
		}
	}()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	expected := []string{"PhoneNumber", "Position", "Prize", "Amount"}
	if len(header) != len(expected) {
		return nil, fmt.Errorf("expected CSV header: %v", expected)
	}

	var winners []*WinnerImport
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) != 4 {
			continue
		}
		pos, _ := strconv.Atoi(rec[1])
		amount, _ := strconv.ParseInt(rec[3], 10, 64)
		winners = append(winners, &WinnerImport{
			PhoneNumber: rec[0],
			Position:    pos,
			Prize:       rec[2],
			Amount:      amount,
		})
	}
	return winners, nil
}

// ─── Stats ────────────────────────────────────────────────────────────────

// ListUpcomingDraws returns draws with status UPCOMING or ACTIVE.
func (svc *DrawService) ListUpcomingDraws(ctx context.Context) ([]DrawRecord, error) {
	var draws []DrawRecord
	err := svc.db.WithContext(ctx).
		Where("status IN ('UPCOMING','ACTIVE')").
		Order("draw_time ASC").
		Find(&draws).Error
	return draws, err
}

// GetActiveDraws returns ACTIVE draws.
func (svc *DrawService) GetActiveDraws(ctx context.Context) ([]DrawRecord, error) {
	var draws []DrawRecord
	err := svc.db.WithContext(ctx).
		Where("status = 'ACTIVE'").
		Order("draw_time ASC").
		Find(&draws).Error
	return draws, err
}

// GetStats returns aggregate draw statistics.
func (svc *DrawService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	var totalDraws, completedDraws, scheduledDraws int64
	var totalWinners int64
	svc.db.Model(&DrawRecord{}).Count(&totalDraws)
	svc.db.Model(&DrawRecord{}).Where("status = 'COMPLETED'").Count(&completedDraws)
	svc.db.Model(&DrawRecord{}).Where("status = 'UPCOMING'").Count(&scheduledDraws)
	svc.db.Model(&DrawWinner{}).Count(&totalWinners)
	return map[string]interface{}{
		"total_draws":     totalDraws,
		"completed_draws": completedDraws,
		"scheduled_draws": scheduledDraws,
		"total_winners":   totalWinners,
	}, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────

// nextDrawTime computes the next draw time for a recurring schedule.
func nextDrawTime(current time.Time, recurrence string) time.Time {
	switch recurrence {
	case "daily":
		return current.Add(24 * time.Hour)
	case "weekly":
		return current.Add(7 * 24 * time.Hour)
	case "monthly":
		return current.AddDate(0, 1, 0)
	default:
		return current.Add(30 * 24 * time.Hour)
	}
}

// drawShuffleCrypto performs a Fisher-Yates shuffle using CSPRNG.
func drawShuffleCrypto(s []DrawEntry) {
	for i := len(s) - 1; i > 0; i-- {
		var b [8]byte
		crand.Read(b[:]) //nolint:errcheck
		j := int(binary.BigEndian.Uint64(b[:]) % uint64(i+1))
		s[i], s[j] = s[j], s[i]
	}
}

// timePtr returns a pointer to a time.Time — used when assigning optional
// *time.Time fields on draw and schedule entities.
func timePtr(t time.Time) *time.Time { return &t }
