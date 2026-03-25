package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionRepository interface {
	// Immutable write — never updates
	Save(ctx context.Context, tx *entities.Transaction) error
	SaveTx(ctx context.Context, dbTx *gorm.DB, tx *entities.Transaction) error

	// Read
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Transaction, error)
	FindByReference(ctx context.Context, reference string) (*entities.Transaction, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Transaction, error)
	CountByUserAndType(ctx context.Context, userID uuid.UUID, txType entities.TransactionType, sinceEpoch int64) (int64, error)
	CountByPhoneAndTypeSince(ctx context.Context, phone string, txType entities.TransactionType, sinceEpoch int64) (int64, error)
	SumAmountByUserSince(ctx context.Context, userID uuid.UUID, sinceEpoch int64) (int64, error)

	// Reporting
	DailyLiabilityTotal(ctx context.Context) (int64, error) // Sum of prize_award amounts today
}
