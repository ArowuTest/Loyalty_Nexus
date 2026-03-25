package entities

import (
	"time"
	"github.com/google/uuid"
)

type User struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MSISDN              string    `gorm:"uniqueIndex" json:"msisdn"`
	UserCode            string    `gorm:"uniqueIndex" json:"user_code"`
	State               string    `json:"state"` // REQ-1.5
	TotalPoints         int64     `json:"total_points"`
	StampsCount         int       `json:"stamps_count"`
	TotalRechargeAmount int64     `json:"total_recharge_amount"`
	Tier                string    `json:"tier"`
	StreakCount         int       `json:"streak_count"`
	LastVisitAt         time.Time `json:"last_visit_at"`
	IsActive            bool      `json:"is_active"`
	MoMoNumber          string    `json:"momo_number"`
	MoMoVerified        bool      `json:"momo_verified"`
	MoMoVerifiedAt      time.Time `json:"momo_verified_at"`
	SpinCredits         int       `json:"spin_credits"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
