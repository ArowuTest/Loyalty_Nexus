package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresPrizeRepository struct {
	db *gorm.DB
}

func NewPostgresPrizeRepository(db *gorm.DB) repositories.PrizeRepository {
	return &PostgresPrizeRepository{db: db}
}

func (r *PostgresPrizeRepository) CreateClaim(ctx context.Context, claim *entities.PrizeClaim) error {
	return r.db.WithContext(ctx).Create(claim).Error
}

func (r *PostgresPrizeRepository) UpdateClaim(ctx context.Context, claim *entities.PrizeClaim) error {
	return r.db.WithContext(ctx).Save(claim).Error
}

func (r *PostgresPrizeRepository) FindClaimByID(ctx context.Context, id uuid.UUID) (*entities.PrizeClaim, error) {
	var claim entities.PrizeClaim
	if err := r.db.WithContext(ctx).First(&claim, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &claim, nil
}
