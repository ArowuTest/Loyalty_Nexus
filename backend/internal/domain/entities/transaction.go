package entities

import (
	"time"
	"encoding/json"
	"github.com/google/uuid"
)

// Transaction is the immutable ledger record for all balance changes.
// Once written, a transaction MUST never be updated — compensate with a new row.
type Transaction struct {
	ID           uuid.UUID       `db:"id"            gorm:"column:id;primaryKey"         json:"id"`
	UserID       uuid.UUID       `db:"user_id"       gorm:"column:user_id;index"         json:"user_id"`
	PhoneNumber  string          `db:"phone_number"  gorm:"column:phone_number"          json:"phone_number"`
	Type         TransactionType `db:"type"          gorm:"column:type"                  json:"type"`
	PointsDelta  int64           `db:"points_delta"  gorm:"column:points_delta"          json:"points_delta"`
	SpinDelta    int             `db:"spin_delta"    gorm:"column:spin_delta"            json:"spins_delta"` // For spin credit events
	Amount       int64           `db:"amount"        gorm:"column:amount"                json:"amount"`
	BalanceAfter int64           `db:"balance_after" gorm:"column:balance_after"         json:"balance_after"`
	Reference    string          `db:"reference"     gorm:"column:reference"             json:"reference"`
	Metadata     json.RawMessage `db:"metadata"      gorm:"column:metadata;serializer:json" json:"metadata,omitempty"`
	CreatedAt    time.Time       `db:"created_at"    gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Transaction) TableName() string { return "transactions" }

type TransactionType string

const (
	TxTypeRecharge       TransactionType = "recharge"       // User recharged airtime/data
	TxTypePointsAward    TransactionType = "points_award"   // Points earned from recharge
	TxTypeSpinCreditAward TransactionType = "spin_credit_award" // Spin credit earned (tier-based, daily cap)
	TxTypeDrawEntryAward  TransactionType = "draw_entry_award"  // Draw entry earned (₦200 accumulator)
	TxTypeSpinPlay        TransactionType = "spin_play"         // Spin credit consumed
	TxTypePrizeAward     TransactionType = "prize_award"    // Prize value awarded (Kobo)
	TxTypeStudioSpend    TransactionType = "studio_spend"   // Points spent on AI Studio
	TxTypeStudioRefund   TransactionType = "studio_refund"  // Points refunded on failure
	TxTypeBonus          TransactionType = "bonus"          // Admin/streak/referral bonus
	TxTypeExpiry         TransactionType = "expiry"         // Points expired
)
