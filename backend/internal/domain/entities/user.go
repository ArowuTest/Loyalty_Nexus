package entities

import (
	"time"
	"github.com/google/uuid"
)

type User struct {
	ID                  uuid.UUID `json:"id"`
	MSISDN              string    `json:"msisdn"` // Nigerian Phone Number
	UserCode            string    `json:"user_code"`
	TotalPoints         int64     `json:"total_points"`
	StampsCount         int       `json:"stamps_count"`
	TotalRechargeAmount int64     `json:"total_recharge_amount"`
	Tier                string    `json:"tier"` // Bronze, Silver, Gold, Platinum
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
