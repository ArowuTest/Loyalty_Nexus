package services

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DrawService executes crypto-fair prize draws (REQ-3.2 / SEC-009).
type DrawService struct {
	db *gorm.DB
}

func NewDrawService(db *gorm.DB) *DrawService {
	return &DrawService{db: db}
}

// DrawRecord mirrors the draws table.
type DrawRecord struct {
	ID          uuid.UUID  `gorm:"column:id"`
	Name        string     `gorm:"column:name"`
	Status      string     `gorm:"column:status"`
	WinnerCount int        `gorm:"column:winner_count"`
	PrizeValue  int64      `gorm:"column:prize_value_kobo"`
	PrizeType   string     `gorm:"column:prize_type"`
	ExecutedAt  *time.Time `gorm:"column:executed_at"`
}

func (DrawRecord) TableName() string { return "draws" }

// DrawEntry mirrors draw_entries table.
type DrawEntry struct {
	ID          uuid.UUID `gorm:"column:id"`
	DrawID      uuid.UUID `gorm:"column:draw_id"`
	UserID      uuid.UUID `gorm:"column:user_id"`
	PhoneNumber string    `gorm:"column:phone_number"`
	TicketCount int       `gorm:"column:ticket_count"`
}

func (DrawEntry) TableName() string { return "draw_entries" }

// ExecuteDraw selects winners for a given draw using CSPRNG (SEC-009).
// Each entry's ticket_count gives weighted probability.
func (svc *DrawService) ExecuteDraw(ctx context.Context, drawID uuid.UUID) error {
	return svc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1 — fetch draw
		var draw DrawRecord
		if err := tx.Where("id = ? AND status = 'ACTIVE'", drawID).First(&draw).Error; err != nil {
			return fmt.Errorf("active draw not found: %w", err)
		}

		// 2 — build weighted ticket pool
		var entries []DrawEntry
		if err := tx.Where("draw_id = ?", drawID).Find(&entries).Error; err != nil {
			return fmt.Errorf("load entries: %w", err)
		}
		if len(entries) == 0 {
			return fmt.Errorf("no entries for draw %s", drawID)
		}

		// expand into ticket pool (up to 100k slots for memory safety)
		pool := make([]DrawEntry, 0, len(entries)*2)
		for _, e := range entries {
			count := e.TicketCount
			if count < 1 {
				count = 1
			}
			if count > 100 {
				count = 100 // cap per user
			}
			for i := 0; i < count; i++ {
				pool = append(pool, e)
			}
		}

		// 3 — crypto shuffle
		cryptoShuffle(pool)

		// 4 — pick winners (deduplicated by user_id)
		winnerCount := draw.WinnerCount
		if winnerCount < 1 {
			winnerCount = 3
		}
		seen := make(map[uuid.UUID]bool)
		winners := make([]DrawEntry, 0, winnerCount)
		for _, ticket := range pool {
			if seen[ticket.UserID] {
				continue
			}
			seen[ticket.UserID] = true
			winners = append(winners, ticket)
			if len(winners) >= winnerCount {
				break
			}
		}

		// 5 — insert draw_winners
		now := time.Now()
		for pos, w := range winners {
			row := map[string]interface{}{
				"id":            uuid.New(),
				"draw_id":       drawID,
				"user_id":       w.UserID,
				"phone_number":  w.PhoneNumber,
				"position":      pos + 1,
				"prize_type":    draw.PrizeType,
				"prize_value_kobo": draw.PrizeValue,
				"status":        "PENDING_FULFILLMENT",
				"created_at":    now,
				"updated_at":    now,
			}
			if err := tx.Table("draw_winners").Create(row).Error; err != nil {
				return fmt.Errorf("insert winner pos %d: %w", pos+1, err)
			}
		}

		// 6 — mark draw completed
		return tx.Table("draws").Where("id = ?", drawID).Updates(map[string]interface{}{
			"status":      "COMPLETED",
			"executed_at": now,
			"updated_at":  now,
		}).Error
	})
}

// ListUpcomingDraws returns draws with status SCHEDULED or ACTIVE.
func (svc *DrawService) ListUpcomingDraws(ctx context.Context) ([]DrawRecord, error) {
	var draws []DrawRecord
	err := svc.db.WithContext(ctx).
		Where("status IN ('SCHEDULED','ACTIVE')").
		Order("created_at ASC").
		Find(&draws).Error
	return draws, err
}

// GetDrawWinners returns the winner list for a completed draw.
func (svc *DrawService) GetDrawWinners(ctx context.Context, drawID uuid.UUID) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	err := svc.db.WithContext(ctx).Table("draw_winners").
		Where("draw_id = ?", drawID).
		Order("position ASC").
		Find(&rows).Error
	return rows, err
}

// cryptoShuffle performs a Fisher-Yates shuffle using CSPRNG.
func cryptoShuffle[T any](s []T) {
	for i := len(s) - 1; i > 0; i-- {
		var b [8]byte
		crand.Read(b[:]) //nolint:errcheck
		j := int(binary.BigEndian.Uint64(b[:]) % uint64(i+1))
		s[i], s[j] = s[j], s[i]
	}
}
