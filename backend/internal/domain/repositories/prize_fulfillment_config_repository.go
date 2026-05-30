package repositories

import (
	"context"

	"loyalty-nexus/internal/domain/entities"
)

// PrizeFulfillmentConfigRepository manages the admin-configurable fulfillment
// policy per prize type. One row per prize type (seeded by migration 126).
type PrizeFulfillmentConfigRepository interface {
	// GetByPrizeType returns the config for a specific prize type.
	// Returns nil, nil if no row exists (treat as MANUAL in callers).
	GetByPrizeType(ctx context.Context, prizeType entities.PrizeType) (*entities.PrizeFulfillmentConfig, error)

	// GetAll returns configs for all prize types, ordered by prize_type.
	GetAll(ctx context.Context) ([]entities.PrizeFulfillmentConfig, error)

	// Update persists changes to an existing config row (identified by PrizeType).
	Update(ctx context.Context, config *entities.PrizeFulfillmentConfig) error
}
