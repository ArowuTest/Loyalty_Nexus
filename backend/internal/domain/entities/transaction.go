package entities

import (
	"time"
	"encoding/json"
	"github.com/google/uuid"
)

// Transaction is the immutable ledger record for all balance changes.
// Once written, a transaction MUST never be updated — compensate with a new row.
type Transaction struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	UserID       uuid.UUID       `db:"user_id" json:"user_id"`
	PhoneNumber  string          `db:"phone_number" json:"phone_number"`
	Type         TransactionType `db:"type" json:"type"`
	PointsDelta  int64           `db:"points_delta" json:"points_delta"`
	SpinDelta    int             `db:"spins_delta" json:"spins_delta"` // For spin credit events
	Amount       int64           `db:"amount" json:"amount"`           // Kobo
	BalanceAfter int64           `db:"balance_after" json:"balance_after"`
	Reference    string          `db:"reference" json:"reference"` // External idempotency key
	Metadata     json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
}

type TransactionType string

const (
	TxTypeRecharge       TransactionType = "recharge"       // User recharged airtime/data
	TxTypePointsAward    TransactionType = "points_award"   // Points earned from recharge
	TxTypeSpinCreditAward TransactionType = "spin_credit_award" // Spin credit earned
	TxTypeSpinPlay       TransactionType = "spin_play"      // Spin credit consumed
	TxTypePrizeAward     TransactionType = "prize_award"    // Prize value awarded (Kobo)
	TxTypeStudioSpend    TransactionType = "studio_spend"   // Points spent on AI Studio
	TxTypeStudioRefund   TransactionType = "studio_refund"  // Points refunded on failure
	TxTypeBonus          TransactionType = "bonus"          // Admin/streak/referral bonus
	TxTypeSubscription   TransactionType = "subscription"   // Daily draw subscription debit
	TxTypeExpiry         TransactionType = "expiry"         // Points expired
)
