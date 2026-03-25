package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
)

type PrizeRepository interface {
	CreateClaim(ctx context.Context, claim *entities.PrizeClaim) error
	UpdateClaim(ctx context.Context, claim *entities.PrizeClaim) error
	FindClaimByID(ctx context.Context, id uuid.UUID) (*entities.PrizeClaim, error)
}
