package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"gorm.io/gorm"
)

type PostgresTransactionRepository struct {
	db *gorm.DB
}

func NewPostgresTransactionRepository(db *gorm.DB) repositories.TransactionRepository {
	return &PostgresTransactionRepository{db: db}
}

func (r *PostgresTransactionRepository) Save(ctx context.Context, tx *entities.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *PostgresTransactionRepository) SaveTx(ctx context.Context, dbtx *gorm.DB, tx *entities.Transaction) error {
	return dbtx.WithContext(ctx).Create(tx).Error
}
