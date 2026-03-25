package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/repositories"
	"gorm.io/gorm"
	"time"
)

type PostgresHLRRepository struct {
	db *gorm.DB
}

func NewPostgresHLRRepository(db *gorm.DB) repositories.HLRRepository {
	return &PostgresHLRRepository{db: db}
}

func (r *PostgresHLRRepository) FindByMSISDN(ctx context.Context, msisdn string) (*repositories.NetworkCache, error) {
	var c repositories.NetworkCache
	if err := r.db.WithContext(ctx).Table("network_cache").Where("msisdn = ?", msisdn).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresHLRRepository) Save(ctx context.Context, cache *repositories.NetworkCache) error {
	// Note: You might need to add gorm tags or map to an entity if network_cache table structure differs from repository struct
	// Assuming simple mapping for now.
	return r.db.WithContext(ctx).Table("network_cache").Save(cache).Error
}

func (r *PostgresHLRRepository) Invalidate(ctx context.Context, msisdn string) error {
	return r.db.WithContext(ctx).Table("network_cache").Where("msisdn = ?", msisdn).Update("is_valid", false).Error
}
