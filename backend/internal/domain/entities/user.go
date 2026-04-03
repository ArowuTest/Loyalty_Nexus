package entities

import (
	"time"
	"github.com/google/uuid"
)

// User represents a Loyalty Nexus subscriber.
// phone_number is the canonical identifier (E.164 format: 2348XXXXXXXXX).
// IMPORTANT: Every field carries both a `db:` tag (for sqlx) AND a
// `gorm:"column:..."` tag (for GORM). Without the gorm tag, GORM derives
// the column name from the Go field name using its own snake_case converter,
// which produces wrong names for acronym-prefixed fields like MoMoNumber
// (→ mo_mo_number instead of momo_number).
type User struct {
	ID                    uuid.UUID  `db:"id"                     gorm:"column:id;primaryKey;default:gen_random_uuid()"  json:"id"`
	PhoneNumber           string     `db:"phone_number"            gorm:"column:phone_number"             json:"phone_number"`
	DisplayName           string     `db:"display_name"            gorm:"column:display_name"             json:"display_name,omitempty"`
	Email                 string     `db:"email"                   gorm:"column:email"                    json:"email,omitempty"`
	UserCode              string     `db:"user_code"               gorm:"column:user_code"                json:"user_code"`
	State                 string     `db:"state"                   gorm:"column:state"                    json:"state"`
	Tier                  string     `db:"tier"                    gorm:"column:tier"                     json:"tier"`
	StreakCount           int        `db:"streak_count"            gorm:"column:streak_count"             json:"streak_count"`
	StreakExpiresAt       *time.Time `db:"streak_expires_at"       gorm:"column:streak_expires_at"        json:"streak_expires_at,omitempty"`
	StreakGraceUsed       int        `db:"streak_grace_used"       gorm:"column:streak_grace_used"        json:"streak_grace_used"`
	StreakGraceMonth      *int       `db:"streak_grace_month"      gorm:"column:streak_grace_month"       json:"-"`
	TotalRechargeAmount   int64      `db:"total_recharge_amount"   gorm:"column:total_recharge_amount"    json:"total_recharge_amount"` // Kobo
	LastRechargeAt        *time.Time `db:"last_recharge_at"        gorm:"column:last_recharge_at"         json:"last_recharge_at,omitempty"`
	MoMoNumber            string     `db:"momo_number"             gorm:"column:momo_number"              json:"momo_number,omitempty"`
	MoMoVerified          bool       `db:"momo_verified"           gorm:"column:momo_verified"            json:"momo_verified"`
	MoMoVerifiedAt        *time.Time `db:"momo_verified_at"        gorm:"column:momo_verified_at"         json:"momo_verified_at,omitempty"`
	WalletPassID          string     `db:"wallet_pass_id"          gorm:"column:wallet_pass_id"           json:"-"`
	DeviceType            string     `db:"device_type"             gorm:"column:device_type"              json:"device_type"`
	// Deprecated: subscription billing removed. Columns retained for zero-downtime migration; hidden from API.
	SubscriptionTier      string     `db:"subscription_tier"       gorm:"column:subscription_tier"        json:"-"`
	SubscriptionStatus    string     `db:"subscription_status"     gorm:"column:subscription_status"      json:"-"`
	SubscriptionExpiresAt *time.Time `db:"subscription_expires_at" gorm:"column:subscription_expires_at" json:"-"`
	KYCStatus             string     `db:"kyc_status"              gorm:"column:kyc_status"               json:"kyc_status"`
	PointsExpireAt        *time.Time `db:"points_expire_at"        gorm:"column:points_expire_at"         json:"points_expire_at,omitempty"`
	TotalPoints           int64      `db:"total_points"            gorm:"column:total_points"             json:"total_points"`
	StampsCount           int        `db:"stamps_count"            gorm:"column:stamps_count"             json:"stamps_count"`
	LifetimePoints        int64      `db:"lifetime_points"         gorm:"column:lifetime_points"          json:"lifetime_points"`
	TotalSpins            int        `db:"total_spins"             gorm:"column:total_spins"              json:"total_spins"`
	StudioUseCount        int        `db:"studio_use_count"        gorm:"column:studio_use_count"         json:"studio_use_count"`
	GoogleWalletObjectID  string     `db:"google_wallet_object_id" gorm:"column:google_wallet_object_id" json:"google_wallet_object_id,omitempty"`
	ApplePassSerial       string     `db:"apple_pass_serial"       gorm:"column:apple_pass_serial"        json:"apple_pass_serial,omitempty"`
	SpinCredits           int        `db:"spin_credits"            gorm:"column:spin_credits"             json:"spin_credits"`
	IsActive              bool       `db:"is_active"               gorm:"column:is_active"                json:"is_active"`
	CreatedAt             time.Time  `db:"created_at"              gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at"              gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// Wallet holds the multi-pool ledger for a user.
//
// Currency pools:
//   PulsePoints     — AI Studio currency. Earned at ₦250 per point (configurable).
//   SpinCredits     — Spin Wheel currency. Tier-based on cumulative daily recharge.
//
// Accumulators (kobo remainder between awards):
//   DrawCounter     — tracks kobo remainder for draw entry awards (₦200 threshold).
//   PulseCounter    — tracks kobo remainder for Pulse Point awards (₦250 threshold).
//
// Daily spin tracking (for tier-based spin logic):
//   DailyRechargeKobo — cumulative recharge today in kobo (resets at midnight WAT).
//   DailyRechargeDate — the calendar date DailyRechargeKobo was last reset.
//   DailySpinsAwarded — spins already awarded today (prevents double-awarding on tier upgrade).
//
// Deprecated:
//   SpinDrawCounter — old shared counter for spin+draw, replaced by DrawCounter.
//   SpinCounter     — reserved, not currently used.
//   RechargeCounter — legacy field, kept for backwards compatibility.
type Wallet struct {
	ID                uuid.UUID  `db:"id"                   gorm:"column:id;primaryKey;default:gen_random_uuid()"  json:"id"`
	UserID            uuid.UUID  `db:"user_id"               gorm:"column:user_id;uniqueIndex" json:"user_id"`
	PulsePoints       int64      `db:"pulse_points"          gorm:"column:pulse_points"          json:"pulse_points"`
	SpinCredits       int        `db:"spin_credits"          gorm:"column:spin_credits"          json:"spin_credits"`
	LifetimePoints    int64      `db:"lifetime_points"       gorm:"column:lifetime_points"       json:"lifetime_points"`
	RechargeCounter   int64      `db:"recharge_counter"      gorm:"column:recharge_counter"      json:"recharge_counter"`
	SpinDrawCounter   int64      `db:"spin_draw_counter"     gorm:"column:spin_draw_counter"     json:"spin_draw_counter"`  // deprecated
	SpinCounter       int64      `db:"spin_counter"          gorm:"column:spin_counter"          json:"spin_counter"`          // reserved
	DrawCounter       int64      `db:"draw_counter"          gorm:"column:draw_counter"          json:"draw_counter"`
	PulseCounter      int64      `db:"pulse_counter"         gorm:"column:pulse_counter"         json:"pulse_counter"`
	DailyRechargeKobo int64      `db:"daily_recharge_kobo"   gorm:"column:daily_recharge_kobo"   json:"daily_recharge_kobo"`
	DailyRechargeDate *time.Time `db:"daily_recharge_date"   gorm:"column:daily_recharge_date"   json:"daily_recharge_date,omitempty"`
	DailySpinsAwarded int        `db:"daily_spins_awarded"   gorm:"column:daily_spins_awarded"   json:"daily_spins_awarded"`
	UpdatedAt         time.Time  `db:"updated_at"            gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Wallet) TableName() string { return "wallets" }

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
