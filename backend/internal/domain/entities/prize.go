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

type ClaimStatus string
const (
	ClaimPending      ClaimStatus = "PENDING"
	ClaimPendingAdmin ClaimStatus = "PENDING_ADMIN_REVIEW"
	ClaimApproved     ClaimStatus = "APPROVED"
	ClaimRejected     ClaimStatus = "REJECTED"
	ClaimClaimed      ClaimStatus = "CLAIMED"
	ClaimExpired      ClaimStatus = "EXPIRED"
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

	// Claim lifecycle fields
	ClaimStatus       ClaimStatus       `db:"claim_status"       gorm:"column:claim_status;default:'PENDING'"    json:"claim_status"`
	ExpiresAt         time.Time         `db:"expires_at"         gorm:"column:expires_at"                        json:"expires_at"`
	MoMoClaimNumber   string            `db:"momo_claim_number"  gorm:"column:momo_claim_number;default:''"      json:"momo_claim_number,omitempty"`
	BankAccountNumber string            `db:"bank_account_number" gorm:"column:bank_account_number;default:''"   json:"bank_account_number,omitempty"`
	BankAccountName   string            `db:"bank_account_name"  gorm:"column:bank_account_name;default:''"      json:"bank_account_name,omitempty"`
	BankName          string            `db:"bank_name"          gorm:"column:bank_name;default:''"              json:"bank_name,omitempty"`
	ReviewedBy        *uuid.UUID        `db:"reviewed_by"        gorm:"column:reviewed_by"                       json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time        `db:"reviewed_at"        gorm:"column:reviewed_at"                       json:"reviewed_at,omitempty"`
	RejectionReason   string            `db:"rejection_reason"   gorm:"column:rejection_reason;default:''"       json:"rejection_reason,omitempty"`
	AdminNotes        string            `db:"admin_notes"        gorm:"column:admin_notes;default:''"            json:"admin_notes,omitempty"`
	PaymentReference  string            `db:"payment_reference"  gorm:"column:payment_reference;default:''"      json:"payment_reference,omitempty"`
}

func (SpinResult) TableName() string { return "spin_results" }

// PrizePoolEntry is the admin-configurable prize slot (read from prize_pool table).
type PrizePoolEntry struct {
	ID                  uuid.UUID  `db:"id"                     gorm:"column:id;primaryKey"                   json:"id"`
	Name                string     `db:"name"                   gorm:"column:name"                            json:"name"`
	PrizeCode           string     `db:"prize_code"             gorm:"column:prize_code;default:''"           json:"prize_code,omitempty"`
	VariationCode       string     `db:"variation_code"         gorm:"column:variation_code;default:''"       json:"variation_code,omitempty"`
	PrizeType           PrizeType  `db:"prize_type"             gorm:"column:prize_type"                      json:"prize_type"`
	BaseValue           float64    `db:"base_value"             gorm:"column:base_value"                      json:"base_value"`
	IsActive            bool       `db:"is_active"              gorm:"column:is_active"                       json:"is_active"`
	ProbWeight          int        `db:"win_probability_weight" gorm:"column:win_probability_weight"          json:"win_probability_weight"`
	DailyInventoryCap   *int       `db:"daily_inventory_cap"    gorm:"column:daily_inventory_cap"             json:"daily_inventory_cap,omitempty"`
	IsNoWin             bool       `db:"is_no_win"              gorm:"column:is_no_win;default:false"         json:"is_no_win"`
	NoWinMessage        string     `db:"no_win_message"         gorm:"column:no_win_message;default:''"       json:"no_win_message,omitempty"`
	ColorScheme         string     `db:"color_scheme"           gorm:"column:color_scheme;default:''"         json:"color_scheme,omitempty"`
	SortOrder           int        `db:"sort_order"             gorm:"column:sort_order;default:0"            json:"sort_order"`
	MinimumRecharge     int64      `db:"minimum_recharge"       gorm:"column:minimum_recharge;default:0"      json:"minimum_recharge"`
	IconName            string     `db:"icon_name"              gorm:"column:icon_name;default:''"            json:"icon_name,omitempty"`
	TermsAndConditions  string     `db:"terms_and_conditions"   gorm:"column:terms_and_conditions;default:''" json:"terms_and_conditions,omitempty"`
}

func (PrizePoolEntry) TableName() string { return "prize_pool" }

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
	IconName  string    `json:"icon_name,omitempty"`
	IsNoWin   bool      `json:"is_no_win"`
	NoWinMsg  string    `json:"no_win_message,omitempty"`
}
