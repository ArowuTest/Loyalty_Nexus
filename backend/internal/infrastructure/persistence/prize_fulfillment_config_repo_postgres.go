package persistence

import (
	"context"

	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
)

type postgresPrizeFulfillmentConfigRepository struct{ db *gorm.DB }

// NewPostgresPrizeFulfillmentConfigRepository creates a GORM-backed
// PrizeFulfillmentConfigRepository. The underlying table is seeded by
// migration 126 with all prize types defaulted to MANUAL.
func NewPostgresPrizeFulfillmentConfigRepository(db *gorm.DB) repositories.PrizeFulfillmentConfigRepository {
	return &postgresPrizeFulfillmentConfigRepository{db: db}
}

func (r *postgresPrizeFulfillmentConfigRepository) GetByPrizeType(
	ctx context.Context, prizeType entities.PrizeType,
) (*entities.PrizeFulfillmentConfig, error) {
	var cfg entities.PrizeFulfillmentConfig
	err := r.db.WithContext(ctx).
		Where("prize_type = ?", prizeType).
		First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *postgresPrizeFulfillmentConfigRepository) GetAll(
	ctx context.Context,
) ([]entities.PrizeFulfillmentConfig, error) {
	var cfgs []entities.PrizeFulfillmentConfig
	err := r.db.WithContext(ctx).
		Order("prize_type ASC").
		Find(&cfgs).Error
	return cfgs, err
}

func (r *postgresPrizeFulfillmentConfigRepository) Update(
	ctx context.Context, config *entities.PrizeFulfillmentConfig,
) error {
	return r.db.WithContext(ctx).
		Model(&entities.PrizeFulfillmentConfig{}).
		Where("prize_type = ?", config.PrizeType).
		Updates(map[string]interface{}{
			"fulfillment_mode":    config.FulfillmentMode,
			"max_retry_attempts":  config.MaxRetryAttempts,
			"retry_delay_seconds": config.RetryDelaySeconds,
			"fallback_to_manual":  config.FallbackToManual,
		}).Error
}
