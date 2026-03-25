package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
)

type PrizeRepository interface {
	// Prize Pool (admin-configurable)
	ListActivePrizes(ctx context.Context) ([]entities.PrizePoolEntry, error)
	ListActivePrizesMaxValue(ctx context.Context, maxValueKobo int64) ([]entities.PrizePoolEntry, error)
	GetDailyInventoryUsed(ctx context.Context, prizeID uuid.UUID) (int, error)

	// Spin Results
	CreateSpinResult(ctx context.Context, result *entities.SpinResult) error
	FindSpinResult(ctx context.Context, id uuid.UUID) (*entities.SpinResult, error)
	UpdateSpinFulfillment(ctx context.Context, id uuid.UUID, status entities.FulfillmentStatus, ref string, errMsg string) error
	ListPendingFulfillments(ctx context.Context, limit int) ([]entities.SpinResult, error)
	CountUserSpinsToday(ctx context.Context, userID uuid.UUID) (int, error)
}
