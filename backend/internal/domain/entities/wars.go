package entities

// wars.go — Domain entities for Regional Wars (spec §3.5 / SRS REQ-5.1–5.5)
//
// Data model:
//   regional_wars        — one row per calendar month (period "YYYY-MM")
//   regional_war_winners — top-3 state winners after ResolveWar
//
// Leaderboard is computed live from transactions.points_delta WHERE type='points_award'.
// This avoids a separate denormalised table and stays consistent with the
// immutable transaction ledger principle.

import (
	"time"

	"github.com/google/uuid"
)

// ─── RegionalWar ─────────────────────────────────────────────────────────────

// RegionalWar represents one monthly war cycle.
type RegionalWar struct {
	ID             uuid.UUID  `gorm:"column:id;primaryKey"        json:"id"`
	Period         string     `gorm:"column:period;uniqueIndex"   json:"period"`           // "YYYY-MM"
	Status         string     `gorm:"column:status"               json:"status"`           // ACTIVE|COMPLETED
	TotalPrizeKobo int64      `gorm:"column:total_prize_kobo"     json:"total_prize_kobo"` // ₦500k default
	StartsAt       time.Time  `gorm:"column:starts_at"            json:"starts_at"`
	EndsAt         time.Time  `gorm:"column:ends_at"              json:"ends_at"`
	ResolvedAt     *time.Time `gorm:"column:resolved_at"          json:"resolved_at,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RegionalWar) TableName() string { return "regional_wars" }

const (
	WarStatusActive    = "ACTIVE"
	WarStatusCompleted = "COMPLETED"
)

// ─── RegionalWarWinner ───────────────────────────────────────────────────────

// RegionalWarWinner is written once per top-3 state when a war is resolved.
type RegionalWarWinner struct {
	ID          uuid.UUID `gorm:"column:id;primaryKey"     json:"id"`
	WarID       uuid.UUID `gorm:"column:war_id;index"      json:"war_id"`
	State       string    `gorm:"column:state"             json:"state"`
	Rank        int       `gorm:"column:rank"              json:"rank"`
	TotalPoints int64     `gorm:"column:total_points"      json:"total_points"`
	PrizeKobo   int64     `gorm:"column:prize_kobo"        json:"prize_kobo"`
	Status      string    `gorm:"column:status"            json:"status"` // PENDING|PAID
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RegionalWarWinner) TableName() string { return "regional_war_winners" }

// ─── LeaderboardEntry ────────────────────────────────────────────────────────

// LeaderboardEntry is a computed read-model — never persisted.
type LeaderboardEntry struct {
	State         string `json:"state"`
	TotalPoints   int64  `json:"total_points"`
	ActiveMembers int    `json:"active_members"`
	Rank          int    `json:"rank"`
	PrizeKobo     int64  `json:"prize_kobo"`
	Period        string `json:"period"`
}
