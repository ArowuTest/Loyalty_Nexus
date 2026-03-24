package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	FindByMSISDN(ctx context.Context, msisdn string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
}
