package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"gorm.io/gorm"
)

type TransactionRepository interface {
	Save(ctx context.Context, tx *entities.Transaction) error
	SaveTx(ctx context.Context, dbtx *gorm.DB, tx *entities.Transaction) error
}
