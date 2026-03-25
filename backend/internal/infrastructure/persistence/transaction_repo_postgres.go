package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
)

type PostgresTransactionRepository struct {
	db *sql.DB
}

func NewPostgresTransactionRepository(db *sql.DB) repositories.TransactionRepository {
	return &PostgresTransactionRepository{db: db}
}

func (r *PostgresTransactionRepository) Save(ctx context.Context, tx *entities.Transaction) error {
	metadata, _ := json.Marshal(tx.Metadata)
	query := `INSERT INTO transactions (id, user_id, msisdn, type, points_delta, stamps_delta, amount, metadata, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query, tx.ID, tx.UserID, tx.MSISDN, tx.Type, tx.PointsDelta, tx.StampsDelta, tx.Amount, metadata, tx.CreatedAt)
	return err
}

func (r *PostgresTransactionRepository) SaveTx(ctx context.Context, dbtx *sql.Tx, tx *entities.Transaction) error {
	metadata, _ := json.Marshal(tx.Metadata)
	query := `INSERT INTO transactions (id, user_id, msisdn, type, points_delta, stamps_delta, amount, metadata, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := dbtx.ExecContext(ctx, query, tx.ID, tx.UserID, tx.MSISDN, tx.Type, tx.PointsDelta, tx.StampsDelta, tx.Amount, metadata, tx.CreatedAt)
	return err
}
