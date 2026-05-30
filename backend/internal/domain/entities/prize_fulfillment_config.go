package entities

import (
	"time"

	"github.com/google/uuid"
)

// FulfillmentMode controls how a prize is fulfilled after a winning spin.
type FulfillmentMode string

const (
	// FulfillmentModeManual — user must click "Claim" from their dashboard.
	// This is the safe default: existing behaviour, no VTPass call at spin time.
	FulfillmentModeManual FulfillmentMode = "MANUAL"

	// FulfillmentModeAuto — background goroutine fires VTPass (or equivalent)
	// immediately at spin time. Only meaningful for prize types that can be
	// digitally provisioned: PrizeAirtime and PrizeDataBundle.
	FulfillmentModeAuto FulfillmentMode = "AUTO"
)

// PrizeFulfillmentConfig stores the admin-configurable fulfillment policy
// for each prize type. One row per prize type, unique on prize_type.
// All rows default to MANUAL — admin must explicitly enable AUTO.
type PrizeFulfillmentConfig struct {
	ID                 uuid.UUID       `gorm:"column:id;primaryKey;default:gen_random_uuid()"  json:"id"`
	PrizeType          PrizeType       `gorm:"column:prize_type;uniqueIndex;not null"           json:"prize_type"`
	FulfillmentMode    FulfillmentMode `gorm:"column:fulfillment_mode;not null;default:'MANUAL'" json:"fulfillment_mode"`
	MaxRetryAttempts   int             `gorm:"column:max_retry_attempts;not null;default:3"     json:"max_retry_attempts"`
	RetryDelaySeconds  int             `gorm:"column:retry_delay_seconds;not null;default:30"   json:"retry_delay_seconds"`
	FallbackToManual   bool            `gorm:"column:fallback_to_manual;not null;default:true"  json:"fallback_to_manual"`
	UpdatedAt          time.Time       `gorm:"column:updated_at;autoUpdateTime"                 json:"updated_at"`
}

func (PrizeFulfillmentConfig) TableName() string { return "prize_fulfillment_config" }

// IsAutoProvisionable returns true for prize types that can be auto-provisioned
// digitally without any admin or user action (airtime and data bundles only).
func IsAutoProvisionable(pt PrizeType) bool {
	return pt == PrizeAirtime || pt == PrizeDataBundle
}
