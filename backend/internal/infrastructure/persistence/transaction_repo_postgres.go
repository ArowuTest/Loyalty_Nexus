package persistence

import (
	"context"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type postgresTransactionRepository struct{ db *gorm.DB }

func NewPostgresTransactionRepository(db *gorm.DB) repositories.TransactionRepository {
	return &postgresTransactionRepository{db: db}
}

func (r *postgresTransactionRepository) Save(ctx context.Context, tx *entities.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *postgresTransactionRepository) SaveTx(ctx context.Context, dbTx *gorm.DB, tx *entities.Transaction) error {
	return dbTx.WithContext(ctx).Create(tx).Error
}

func (r *postgresTransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Transaction, error) {
	var t entities.Transaction
	return &t, r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
}

func (r *postgresTransactionRepository) FindByReference(ctx context.Context, ref string) (*entities.Transaction, error) {
	var t entities.Transaction
	err := r.db.WithContext(ctx).Where("reference = ?", ref).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *postgresTransactionRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Transaction, error) {
	var txs []entities.Transaction
	r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&txs)
	return txs, nil
}

func (r *postgresTransactionRepository) CountByUserAndType(ctx context.Context, userID uuid.UUID, txType entities.TransactionType, sinceEpoch int64) (int64, error) {
	var count int64
	sinceTime := time.Unix(sinceEpoch, 0)
	r.db.WithContext(ctx).Table("transactions").
		Where("user_id = ? AND type = ? AND created_at >= ?", userID, txType, sinceTime).
		Count(&count)
	return count, nil
}

func (r *postgresTransactionRepository) CountByPhoneAndTypeSince(ctx context.Context, phone string, txType entities.TransactionType, sinceEpoch int64) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Table("transactions").Where("phone_number = ? AND type = ?", phone, txType)
	if sinceEpoch > 0 {
		sinceTime := time.Unix(sinceEpoch, 0)
		q = q.Where("created_at >= ?", sinceTime)
	}
	q.Count(&count)
	return count, nil
}

func (r *postgresTransactionRepository) SumAmountByUserSince(ctx context.Context, userID uuid.UUID, sinceEpoch int64) (int64, error) {
	var total int64
	sinceTime := time.Unix(sinceEpoch, 0)
	r.db.WithContext(ctx).Table("transactions").
		Where("user_id = ? AND created_at >= ?", userID, sinceTime).
		Select("COALESCE(SUM(amount), 0)").Scan(&total)
	return total, nil
}

func (r *postgresTransactionRepository) DailyLiabilityTotal(ctx context.Context) (int64, error) {
	var total int64
	today := time.Now().UTC().Truncate(24 * time.Hour)
	r.db.WithContext(ctx).Raw(
		"SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'prize_award' AND created_at >= ?", today,
	).Scan(&total)
	return total, nil
}
