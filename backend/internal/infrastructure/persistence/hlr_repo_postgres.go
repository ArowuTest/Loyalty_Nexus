package persistence

import (
	"context"
	"errors"
	"time"
	"loyalty-nexus/internal/domain/repositories"
	"gorm.io/gorm"
)

type postgresHLRRepository struct{ db *gorm.DB }

func NewPostgresHLRRepository(db *gorm.DB) repositories.HLRRepository {
	return &postgresHLRRepository{db: db}
}

func (r *postgresHLRRepository) GetCached(ctx context.Context, phone string) (*repositories.HLRResult, error) {
	var row struct {
		Network      string    `gorm:"column:network"`
		IsValid      bool      `gorm:"column:is_valid"`
		LookupSource string    `gorm:"column:lookup_source"`
		CacheExpires time.Time `gorm:"column:cache_expires"`
	}
	err := r.db.WithContext(ctx).Table("network_cache").
		Where("phone_number = ? AND cache_expires > NOW()", phone).First(&row).Error
	if err != nil {
		return nil, errors.New("cache miss")
	}
	return &repositories.HLRResult{
		PhoneNumber:  phone,
		Network:      row.Network,
		IsValid:      row.IsValid,
		LookupSource: row.LookupSource,
	}, nil
}

func (r *postgresHLRRepository) Cache(ctx context.Context, result *repositories.HLRResult, ttlHours int) error {
	return r.db.WithContext(ctx).Table("network_cache").
		Where("phone_number = ?", result.PhoneNumber).
		Assign(map[string]interface{}{
			"network":       result.Network,
			"is_valid":      result.IsValid,
			"lookup_source": result.LookupSource,
			"cache_expires": time.Now().Add(time.Duration(ttlHours) * time.Hour),
			"last_verified": time.Now(),
		}).FirstOrCreate(&result).Error
}

func (r *postgresHLRRepository) Invalidate(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Table("network_cache").Where("phone_number = ?", phone).
		Update("cache_expires", time.Now().Add(-1*time.Second)).Error
}
