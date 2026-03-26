package persistence

import (
	"context"
	"time"

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
	err := r.db.WithContext(ctx).Table("prize_pool").Where("is_active = ? OR is_active = 'true'", true).Find(&prizes).Error
	return prizes, err
}

func (r *postgresPrizeRepository) ListActivePrizesSorted(ctx context.Context) ([]entities.PrizePoolEntry, error) {
	var prizes []entities.PrizePoolEntry
	err := r.db.WithContext(ctx).Table("prize_pool").
		Where("is_active = ? OR is_active = 'true'", true).
		Order("sort_order ASC").
		Find(&prizes).Error
	return prizes, err
}

func (r *postgresPrizeRepository) ListActivePrizesMaxValue(ctx context.Context, maxValueKobo int64) ([]entities.PrizePoolEntry, error) {
	var prizes []entities.PrizePoolEntry
	err := r.db.WithContext(ctx).Table("prize_pool").
		Where("(is_active = ? OR is_active = 'true') AND base_value <= ?", true, maxValueKobo/100).Find(&prizes).Error
	return prizes, err
}

func (r *postgresPrizeRepository) GetDailyInventoryUsed(ctx context.Context, prizeID uuid.UUID) (int, error) {
	var count int64
	today := time.Now().UTC().Truncate(24 * time.Hour)
	r.db.WithContext(ctx).Table("spin_results").
		Joins("JOIN prize_pool ON prize_pool.name = spin_results.prize_type").
		Where("prize_pool.id = ? AND spin_results.created_at >= ?", prizeID, today).
		Count(&count)
	return int(count), nil
}

func (r *postgresPrizeRepository) CreateSpinResult(ctx context.Context, result *entities.SpinResult) error {
	return r.db.WithContext(ctx).Create(result).Error
}

func (r *postgresPrizeRepository) CreateSpinResultTx(ctx context.Context, tx *gorm.DB, result *entities.SpinResult) error {
	return tx.WithContext(ctx).Create(result).Error
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
	today := time.Now().UTC().Truncate(24 * time.Hour)
	r.db.WithContext(ctx).Table("spin_results").
		Where("user_id = ? AND created_at >= ?", userID, today).Count(&count)
	return int(count), nil
}

func (r *postgresPrizeRepository) UpdateSpinClaimStatus(ctx context.Context, id uuid.UUID, status entities.ClaimStatus, bankDetails map[string]string) error {
	updates := map[string]interface{}{
		"claim_status": status,
	}
	if status == entities.ClaimClaimed {
		now := time.Now()
		updates["claimed_at"] = &now
	}
	if bankDetails != nil {
		if v, ok := bankDetails["momo_claim_number"]; ok {
			updates["momo_claim_number"] = v
		}
		if v, ok := bankDetails["bank_account_number"]; ok {
			updates["bank_account_number"] = v
		}
		if v, ok := bankDetails["bank_account_name"]; ok {
			updates["bank_account_name"] = v
		}
		if v, ok := bankDetails["bank_name"]; ok {
			updates["bank_name"] = v
		}
	}
	return r.db.WithContext(ctx).Table("spin_results").Where("id = ?", id).Updates(updates).Error
}

func (r *postgresPrizeRepository) ListUserWins(ctx context.Context, userID uuid.UUID) ([]entities.SpinResult, error) {
	var results []entities.SpinResult
	err := r.db.WithContext(ctx).Table("spin_results").
		Where("user_id = ? AND prize_type != ?", userID, entities.PrizeTryAgain).
		Order("created_at DESC").
		Find(&results).Error
	return results, err
}

func (r *postgresPrizeRepository) ListAdminClaims(ctx context.Context, status string, limit, offset int) ([]entities.SpinResult, int64, error) {
	var results []entities.SpinResult
	var total int64

	query := r.db.WithContext(ctx).Table("spin_results").
		Where("prize_type != ?", entities.PrizeTryAgain)

	if status != "" {
		query = query.Where("claim_status = ?", status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&results).Error
	return results, total, err
}

func (r *postgresPrizeRepository) UpdateAdminClaimReview(ctx context.Context, id uuid.UUID, status entities.ClaimStatus, adminID uuid.UUID, notes, rejectionReason, paymentRef string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"claim_status":      status,
		"reviewed_by":       adminID,
		"reviewed_at":       now,
		"admin_notes":       notes,
		"rejection_reason":  rejectionReason,
		"payment_reference": paymentRef,
	}
	return r.db.WithContext(ctx).Table("spin_results").Where("id = ?", id).Updates(updates).Error
}

func (r *postgresPrizeRepository) AggregateClaimStats(ctx context.Context, dest interface{}) error {
	return r.db.WithContext(ctx).Raw(`
		SELECT claim_status AS status, COUNT(*) AS count, COALESCE(SUM(prize_value), 0) AS total
		FROM spin_results
		WHERE prize_type != 'try_again'
		GROUP BY claim_status
	`).Scan(dest).Error
}
