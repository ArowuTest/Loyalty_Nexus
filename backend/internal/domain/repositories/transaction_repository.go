package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
)

type TransactionRepository interface {
	Save(ctx context.Context, tx *entities.Transaction) error
}
