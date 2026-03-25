package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type postgresPrizeRepository struct{ db *gorm.DB }

func NewPostgresPrizeRepository(db *gorm.DB) repositories.PrizeRepository {
	return &postgresPrizeRepository{db: db}
}

func (r *postgresPrizeRepository) ListActivePrizes(ctx context.Context) ([]entities.PrizePoolEntry, error) {
	var prizes []entities.PrizePoolEntry
	err := r.db.WithContext(ctx).Table("prize_pool").Where("is_active = true").Find(&prizes).Error
	return prizes, err
}

func (r *postgresPrizeRepository) ListActivePrizesMaxValue(ctx context.Context, maxValueKobo int64) ([]entities.PrizePoolEntry, error) {
	var prizes []entities.PrizePoolEntry
	err := r.db.WithContext(ctx).Table("prize_pool").
		Where("is_active = true AND base_value <= ?", maxValueKobo/100).Find(&prizes).Error
	return prizes, err
}

func (r *postgresPrizeRepository) GetDailyInventoryUsed(ctx context.Context, prizeID uuid.UUID) (int, error) {
	var count int64
	r.db.WithContext(ctx).Table("spin_results").
		Joins("JOIN prize_pool ON prize_pool.name = spin_results.prize_type").
		Where("prize_pool.id = ? AND spin_results.created_at >= CURRENT_DATE", prizeID).
		Count(&count)
	return int(count), nil
}

func (r *postgresPrizeRepository) CreateSpinResult(ctx context.Context, result *entities.SpinResult) error {
	return r.db.WithContext(ctx).Create(result).Error
}

func (r *postgresPrizeRepository) FindSpinResult(ctx context.Context, id uuid.UUID) (*entities.SpinResult, error) {
	var result entities.SpinResult
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&result).Error
	return &result, err
}

func (r *postgresPrizeRepository) UpdateSpinFulfillment(ctx context.Context, id uuid.UUID, status entities.FulfillmentStatus, ref, errMsg string) error {
	updates := map[string]interface{}{"fulfillment_status": status}
	if ref != "" {
		updates["fulfillment_ref"] = ref
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	return r.db.WithContext(ctx).Table("spin_results").Where("id = ?", id).Updates(updates).Error
}

func (r *postgresPrizeRepository) ListPendingFulfillments(ctx context.Context, limit int) ([]entities.SpinResult, error) {
	var results []entities.SpinResult
	r.db.WithContext(ctx).Table("spin_results").
		Where("fulfillment_status IN ('pending', 'processing')").
		Order("created_at ASC").Limit(limit).Find(&results)
	return results, nil
}

func (r *postgresPrizeRepository) CountUserSpinsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	r.db.WithContext(ctx).Table("spin_results").
		Where("user_id = ? AND created_at >= CURRENT_DATE", userID).Count(&count)
	return int(count), nil
}
