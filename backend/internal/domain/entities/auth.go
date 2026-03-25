package entities

import (
	"time"
	"github.com/google/uuid"
)

type OTPPurpose string

const (
	PurposeLogin      OTPPurpose = "login"
	PurposeMoMoLink   OTPPurpose = "momo_link"
	PurposePrizeClaim OTPPurpose = "prize_claim"
)

type AuthOTP struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MSISDN    string     `json:"msisdn"`
	Code      string     `json:"code"`
	Purpose   OTPPurpose `json:"purpose"`
	Status    string     `json:"status"` // pending, verified, expired
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}
