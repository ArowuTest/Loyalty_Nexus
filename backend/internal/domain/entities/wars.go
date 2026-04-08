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
	ID             uuid.UUID  `db:"id" gorm:"column:id;primaryKey"        json:"id"`
	Period         string     `db:"period" gorm:"column:period;uniqueIndex"   json:"period"`           // "YYYY-MM"
	Status         string     `db:"status" gorm:"column:status"               json:"status"`           // ACTIVE|COMPLETED
	TotalPrizeKobo int64      `db:"total_prize_kobo" gorm:"column:total_prize_kobo"     json:"total_prize_kobo"` // ₦500k default
	StartsAt       time.Time  `db:"starts_at" gorm:"column:starts_at"            json:"starts_at"`
	EndsAt         time.Time  `db:"ends_at" gorm:"column:ends_at"              json:"ends_at"`
	ResolvedAt     *time.Time `db:"resolved_at" gorm:"column:resolved_at"          json:"resolved_at,omitempty"`
	CreatedAt      time.Time  `db:"created_at" gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RegionalWar) TableName() string { return "regional_wars" }

const (
	WarStatusActive    = "ACTIVE"
	WarStatusCompleted = "COMPLETED"
)

// ─── RegionalWarWinner ───────────────────────────────────────────────────────

// RegionalWarWinner is written once per top-3 state when a war is resolved.
type RegionalWarWinner struct {
	ID          uuid.UUID `db:"id" gorm:"column:id;primaryKey"     json:"id"`
	WarID       uuid.UUID `db:"war_id" gorm:"column:war_id;index"      json:"war_id"`
	State       string    `db:"state" gorm:"column:state"             json:"state"`
	Rank        int       `db:"rank" gorm:"column:rank"              json:"rank"`
	TotalPoints int64     `db:"total_points" gorm:"column:total_points"      json:"total_points"`
	PrizeKobo   int64     `db:"prize_kobo" gorm:"column:prize_kobo"        json:"prize_kobo"`
	Status      string    `db:"status" gorm:"column:status"            json:"status"` // PENDING|PAID
	CreatedAt   time.Time `db:"created_at" gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
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

// ─── WarSecondaryDraw ────────────────────────────────────────────────────────

// WarSecondaryDraw is created when admin triggers the secondary draw for a
// winning state.  At most one may exist per (war, state).
type WarSecondaryDraw struct {
	ID                 uuid.UUID  `db:"id" gorm:"column:id;primaryKey"                       json:"id"`
	WarID              uuid.UUID  `db:"war_id" gorm:"column:war_id;index"                        json:"war_id"`
	State              string     `db:"state" gorm:"column:state"                               json:"state"`
	WinnerCount        int        `db:"winner_count" gorm:"column:winner_count;default:1"              json:"winner_count"`
	PrizePerWinnerKobo int64      `db:"prize_per_winner_kobo" gorm:"column:prize_per_winner_kobo"               json:"prize_per_winner_kobo"`
	TotalPoolKobo      int64      `db:"total_pool_kobo" gorm:"column:total_pool_kobo"                     json:"total_pool_kobo"`
	ParticipantCount   int        `db:"participant_count" gorm:"column:participant_count"                   json:"participant_count"`
	Status             string     `db:"status" gorm:"column:status;default:'PENDING'"            json:"status"` // PENDING|COMPLETED|CANCELLED
	TriggeredBy        *uuid.UUID `db:"triggered_by" gorm:"column:triggered_by"                        json:"triggered_by,omitempty"`
	ExecutedAt         *time.Time `db:"executed_at" gorm:"column:executed_at"                         json:"executed_at,omitempty"`
	CreatedAt          time.Time  `db:"created_at" gorm:"column:created_at;autoCreateTime"           json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at" gorm:"column:updated_at;autoUpdateTime"           json:"updated_at"`
	// Preloaded
	Winners            []WarSecondaryDrawWinner `gorm:"-" json:"winners,omitempty"`
}

func (WarSecondaryDraw) TableName() string { return "war_secondary_draws" }

// ─── WarSecondaryDrawWinner ──────────────────────────────────────────────────

// WarSecondaryDrawWinner is one participant selected by the secondary draw engine.
type WarSecondaryDrawWinner struct {
	ID              uuid.UUID  `db:"id" gorm:"column:id;primaryKey"                      json:"id"`
	SecondaryDrawID uuid.UUID  `db:"secondary_draw_id" gorm:"column:secondary_draw_id;index"            json:"secondary_draw_id"`
	WarID           uuid.UUID  `db:"war_id" gorm:"column:war_id"                             json:"war_id"`
	State           string     `db:"state" gorm:"column:state"                              json:"state"`
	UserID          uuid.UUID  `db:"user_id" gorm:"column:user_id"                            json:"user_id"`
	PhoneNumber     string     `db:"phone_number" gorm:"column:phone_number"                       json:"phone_number"`
	Position        int        `db:"position" gorm:"column:position"                           json:"position"`
	PrizeKobo       int64      `db:"prize_kobo" gorm:"column:prize_kobo"                         json:"prize_kobo"`
	MoMoNumber      string     `db:"momo_number" gorm:"column:momo_number"                        json:"momo_number,omitempty"`
	PaymentStatus   string     `db:"payment_status" gorm:"column:payment_status;default:'PENDING_PAYMENT'" json:"payment_status"` // PENDING_PAYMENT|PAID|FAILED
	PaidAt          *time.Time `db:"paid_at" gorm:"column:paid_at"                            json:"paid_at,omitempty"`
	PaidBy          *uuid.UUID `db:"paid_by" gorm:"column:paid_by"                            json:"paid_by,omitempty"`
	Notes           string     `db:"notes" gorm:"column:notes"                              json:"notes,omitempty"`
	CreatedAt       time.Time  `db:"created_at" gorm:"column:created_at;autoCreateTime"          json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" gorm:"column:updated_at;autoUpdateTime"          json:"updated_at"`
}

func (WarSecondaryDrawWinner) TableName() string { return "war_secondary_draw_winners" }

const (
	SecondaryDrawStatusPending   = "PENDING"
	SecondaryDrawStatusCompleted = "COMPLETED"
	SecondaryDrawStatusCancelled = "CANCELLED"

	SecondaryWinnerPending = "PENDING_PAYMENT"
	SecondaryWinnerPaid    = "PAID"
	SecondaryWinnerFailed  = "FAILED"
)

// UserRef is a minimal read-model used by the secondary draw participant pool.
type UserRef struct {
	ID          uuid.UUID `json:"id"`
	PhoneNumber string    `json:"phone_number"`
}
