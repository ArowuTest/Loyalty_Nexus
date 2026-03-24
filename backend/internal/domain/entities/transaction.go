package entities

import (
	"time"
	"github.com/google/uuid"
)

type TransactionType string

const (
	TxTypeVisit         TransactionType = "visit"
	TxTypeRewardRedeem  TransactionType = "reward_redeem"
	TxTypeBonus         TransactionType = "bonus"
	TxTypeStudioSpend   TransactionType = "studio_spend"
)

type Transaction struct {
	ID            uuid.UUID       `json:"id"`
	UserID        uuid.UUID       `json:"user_id"`
	MSISDN        string          `json:"msisdn"`
	Type          TransactionType `json:"type"`
	PointsDelta   int64           `json:"points_delta"`
	StampsDelta   int             `json:"stamps_delta"`
	Amount        int64           `json:"amount"` // in Kobo
	BalanceAfter  int64           `json:"balance_after"`
	Metadata      map[string]any  `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
}
