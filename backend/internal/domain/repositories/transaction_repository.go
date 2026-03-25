package repositories

import (
	"context"
	"database/sql"
	"loyalty-nexus/internal/domain/entities"
)

type TransactionRepository interface {
	Save(ctx context.Context, tx *entities.Transaction) error
	SaveTx(ctx context.Context, dbtx *sql.Tx, tx *entities.Transaction) error
}
