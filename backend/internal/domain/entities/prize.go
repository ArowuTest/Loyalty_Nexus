package entities

import (
	"time"
	"github.com/google/uuid"
)

type PrizeType string
const (
	PrizeTryAgain      PrizeType = "try_again"
	PrizePulsePoints   PrizeType = "pulse_points"
	PrizeAirtime       PrizeType = "airtime"
	PrizeDataBundle    PrizeType = "data_bundle"
	PrizeMoMoCash      PrizeType = "momo_cash"
)

type FulfillmentStatus string
const (
	FulfillNA              FulfillmentStatus = "na"
	FulfillPending         FulfillmentStatus = "pending"
	FulfillPendingMoMo     FulfillmentStatus = "pending_momo_setup"
	FulfillPendingClaim    FulfillmentStatus = "pending_claim"
	FulfillProcessing      FulfillmentStatus = "processing"
	FulfillCompleted       FulfillmentStatus = "completed"
	FulfillFailed          FulfillmentStatus = "failed"
	FulfillHeld            FulfillmentStatus = "held"
)

// SpinResult is the authoritative record of a spin play.
type SpinResult struct {
	ID                uuid.UUID         `db:"id"                 gorm:"column:id;primaryKey"                     json:"id"`
	UserID            uuid.UUID         `db:"user_id"            gorm:"column:user_id;index"                     json:"user_id"`
	PrizeType         PrizeType         `db:"prize_type"         gorm:"column:prize_type"                        json:"prize_type"`
	PrizeValue        float64           `db:"prize_value"        gorm:"column:prize_value"                       json:"prize_value"`
	SlotIndex         int               `db:"slot_index"         gorm:"column:slot_index"                        json:"slot_index"`
	FulfillmentStatus FulfillmentStatus `db:"fulfillment_status" gorm:"column:fulfillment_status"               json:"fulfillment_status"`
	FulfillmentRef    string            `db:"fulfillment_ref"    gorm:"column:fulfillment_ref;default:''"        json:"fulfillment_ref,omitempty"`
	MoMoNumber        string            `db:"momo_number"        gorm:"column:mo_mo_number;default:''"           json:"momo_number,omitempty"`
	ErrorMessage      string            `db:"error_message"      gorm:"column:error_message;default:''"          json:"error_message,omitempty"`
	RetryCount        int               `db:"retry_count"        gorm:"column:retry_count;default:0"             json:"retry_count"`
	ClaimedAt         *time.Time        `db:"claimed_at"         gorm:"column:claimed_at"                        json:"claimed_at,omitempty"`
	FulfilledAt       *time.Time        `db:"fulfilled_at"       gorm:"column:fulfilled_at"                      json:"fulfilled_at,omitempty"`
	CreatedAt         time.Time         `db:"created_at"         gorm:"column:created_at;autoCreateTime"         json:"created_at"`
}

func (SpinResult) TableName() string { return "spin_results" }

// PrizePoolEntry is the admin-configurable prize slot (read from prize_pool table).
type PrizePoolEntry struct {
	ID               uuid.UUID `db:"id" json:"id"`
	Name             string    `db:"name" json:"name"`
	PrizeType        PrizeType `db:"prize_type" json:"prize_type"`
	BaseValue        float64   `db:"base_value" json:"base_value"`
	IsActive         bool      `db:"is_active" gorm:"column:is_active" json:"is_active"`
	ProbWeight       int       `db:"win_probability_weight" gorm:"column:win_probability_weight" json:"win_probability_weight"`
	DailyInventoryCap *int     `db:"daily_inventory_cap" gorm:"column:daily_inventory_cap" json:"daily_inventory_cap,omitempty"`
}

// SpinWheel is the assembled payload sent to the frontend.
type SpinWheelPayload struct {
	Slots        []SpinSlot `json:"slots"`
	RequiredCredits int      `json:"required_credits"`
}

type SpinSlot struct {
	Index     int       `json:"index"`
	PrizeType PrizeType `json:"prize_type"`
	Label     string    `json:"label"`
	Color     string    `json:"color"`
}
