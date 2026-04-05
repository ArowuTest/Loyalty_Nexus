package entities

import (
	"time"
	"github.com/google/uuid"
)

type User struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MSISDN              string    `gorm:"uniqueIndex" json:"msisdn"`
	UserCode            string    `gorm:"uniqueIndex" json:"user_code"`
	State               string    `json:"state"`
	TotalPoints         int64     `json:"total_points"`
	SpinCredits         int       `json:"spin_credits"`
	StampsCount         int       `json:"stamps_count"`
	TotalRechargeAmount int64     `json:"total_recharge_amount"`
	Tier                string    `json:"tier" gorm:"default:'BRONZE'"`
	StreakCount         int       `json:"streak_count"`
	LastVisitAt         time.Time `json:"last_visit_at"`
	IsActive            bool      `json:"is_active" gorm:"default:true"`
	MoMoNumber          string    `json:"momo_number"`
	MoMoVerified        bool      `json:"momo_verified" gorm:"default:false"`
	MoMoVerifiedAt      time.Time `json:"momo_verified_at"`
	
	// NEW: REQ-5.2.13, REQ-5.2.14
	StreakFreezeGraceUsed int       `json:"streak_freeze_grace_used" gorm:"default:0"`
	PointsExpiryDate      time.Time `json:"points_expiry_date"`
	
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
