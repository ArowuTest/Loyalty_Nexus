package entities

import (
	"time"
	"github.com/google/uuid"
)

// User represents a Loyalty Nexus subscriber.
// phone_number is the canonical identifier (E.164 format: 2348XXXXXXXXX).
type User struct {
	ID                  uuid.UUID  `db:"id" json:"id"`
	PhoneNumber         string     `db:"phone_number" json:"phone_number"`
	UserCode            string     `db:"user_code" json:"user_code"`
	State               string     `db:"state" json:"state"`
	Tier                string     `db:"tier" json:"tier"`
	StreakCount         int        `db:"streak_count" json:"streak_count"`
	StreakExpiresAt     *time.Time `db:"streak_expires_at" json:"streak_expires_at,omitempty"`
	StreakGraceUsed     int        `db:"streak_grace_used" json:"streak_grace_used"`
	StreakGraceMonth    *int       `db:"streak_grace_month" json:"-"`
	TotalRechargeAmount int64      `db:"total_recharge_amount" json:"total_recharge_amount"` // Kobo
	LastRechargeAt      *time.Time `db:"last_recharge_at" json:"last_recharge_at,omitempty"`
	MoMoNumber          string     `db:"momo_number" json:"momo_number,omitempty"`
	MoMoVerified        bool       `db:"momo_verified" json:"momo_verified"`
	MoMoVerifiedAt      *time.Time `db:"momo_verified_at" json:"momo_verified_at,omitempty"`
	WalletPassID        string     `db:"wallet_pass_id" json:"-"`
	DeviceType          string     `db:"device_type" json:"device_type"` // smartphone | feature_phone
	SubscriptionTier    string     `db:"subscription_tier" json:"subscription_tier"`
	ReferralCode        string     `db:"referral_code" json:"referral_code"`
	ReferredBy          *uuid.UUID `db:"referred_by" json:"referred_by,omitempty"`
	KYCStatus           string     `db:"kyc_status" json:"kyc_status"`
	PointsExpireAt      *time.Time `db:"points_expire_at" json:"points_expire_at,omitempty"`
	IsActive            bool       `db:"is_active" json:"is_active"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updated_at"`
}

// Wallet holds the two-pool ledger for a user.
// PulsePoints  → AI Studio currency  (earned by recharging)
// SpinCredits  → Spin Wheel currency (1 per ₦1,000 cumulative recharge)
type Wallet struct {
	ID              uuid.UUID `db:"id" json:"id"`
	UserID          uuid.UUID `db:"user_id" json:"user_id"`
	PulsePoints     int64     `db:"pulse_points" json:"pulse_points"`
	SpinCredits     int       `db:"spin_credits" json:"spin_credits"`
	LifetimePoints  int64     `db:"lifetime_points" json:"lifetime_points"`
	RechargeCounter int64     `db:"recharge_counter" json:"recharge_counter"` // Kobo accumulator
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// UserTier thresholds (based on LifetimePoints)
const (
	TierBronze   = "BRONZE"
	TierSilver   = "SILVER"
	TierGold     = "GOLD"
	TierPlatinum = "PLATINUM"
)

func TierFromLifetimePoints(pts int64) string {
	switch {
	case pts >= 5000:
		return TierPlatinum
	case pts >= 1500:
		return TierGold
	case pts >= 500:
		return TierSilver
	default:
		return TierBronze
	}
}
