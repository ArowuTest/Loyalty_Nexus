package entities

import (
	"time"
	"github.com/google/uuid"
)

// SpinTier represents a cumulative daily recharge tier that grants a daily spin cap.
type SpinTier struct {
	ID              uuid.UUID `db:"id" gorm:"column:id;primaryKey" json:"id"`
	TierName        string    `db:"tier_name" gorm:"column:tier_name" json:"tier_name"`
	TierDisplayName string    `db:"tier_display_name" gorm:"column:tier_display_name" json:"tier_display_name"`
	MinDailyAmount  int64     `db:"min_daily_amount" gorm:"column:min_daily_amount" json:"min_daily_amount"` // in kobo
	MaxDailyAmount  int64     `db:"max_daily_amount" gorm:"column:max_daily_amount" json:"max_daily_amount"` // in kobo
	SpinsPerDay     int       `db:"spins_per_day" gorm:"column:spins_per_day" json:"spins_per_day"`
	TierColor       string    `db:"tier_color" gorm:"column:tier_color" json:"tier_color,omitempty"`
	TierIcon        string    `db:"tier_icon" gorm:"column:tier_icon" json:"tier_icon,omitempty"`
	TierBadge       string    `db:"tier_badge" gorm:"column:tier_badge" json:"tier_badge,omitempty"`
	Description     string    `db:"description" gorm:"column:description" json:"description,omitempty"`
	SortOrder       int       `db:"sort_order" gorm:"column:sort_order" json:"sort_order"`
	IsActive        bool      `db:"is_active" gorm:"column:is_active" json:"is_active"`
	CreatedAt       time.Time `db:"created_at" gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SpinTier) TableName() string {
	return "spin_tiers"
}
