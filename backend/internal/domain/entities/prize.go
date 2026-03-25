package entities

import (
	"time"
	"github.com/google/uuid"
)

type PrizeStatus string

const (
	StatusPendingMoMoLink   PrizeStatus = "pending_momo_link"
	StatusPendingFulfillment PrizeStatus = "pending_fulfillment"
	StatusProcessing        PrizeStatus = "processing"
	StatusCompleted         PrizeStatus = "completed"
	StatusFailed            PrizeStatus = "failed"
)

type PrizeClaim struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID   `json:"user_id"`
	TransactionID  uuid.UUID   `json:"transaction_id"`
	PrizeType      string      `json:"prize_type"`
	PrizeValue     float64     `json:"prize_value"`
	Status         PrizeStatus `json:"status"`
	MoMoNumber     string      `json:"momo_number"`
	FulfillmentRef string      `json:"fulfillment_ref"`
	ErrorMessage   string      `json:"error_message"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}
