package entities

import (
	"time"
	"github.com/google/uuid"
)

type OTPPurpose string
const (
	OTPLogin      OTPPurpose = "login"
	OTPMoMoLink   OTPPurpose = "momo_link"
	OTPPrizeClaim OTPPurpose = "prize_claim"
)

type OTPStatus string
const (
	OTPPending  OTPStatus = "pending"
	OTPVerified OTPStatus = "verified"
	OTPExpired  OTPStatus = "expired"
)

type AuthOTP struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	PhoneNumber string     `db:"phone_number" json:"-"`
	Code        string     `db:"code" json:"-"`     // AES-256 encrypted at rest
	Purpose     OTPPurpose `db:"purpose" json:"purpose"`
	Status      OTPStatus  `db:"status" json:"status"`
	ExpiresAt   time.Time  `db:"expires_at" json:"expires_at"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

type AdminUser struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"` // platform_admin | mno_executive
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// JWTClaims used for both user and admin tokens.
type JWTClaims struct {
	UserID      string `json:"uid"`
	PhoneNumber string `json:"phone,omitempty"`
	Role        string `json:"role,omitempty"` // empty = subscriber
	IsAdmin     bool   `json:"is_admin"`
}
