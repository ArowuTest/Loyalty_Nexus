package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
)

type UserRepository interface {
	// Read
	FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	FindByPhoneNumber(ctx context.Context, phone string) (*entities.User, error)
	ExistsByPhoneNumber(ctx context.Context, phone string) (bool, error)

	// Write
	Create(ctx context.Context, user *entities.User) error
	Update(ctx context.Context, user *entities.User) error
	UpdateStreak(ctx context.Context, userID uuid.UUID, streakCount int, expiresAt interface{}) error
	UpdateMoMo(ctx context.Context, userID uuid.UUID, momoNumber string, verified bool) error
	UpdateTier(ctx context.Context, userID uuid.UUID, tier string) error
	UpdateWalletPassID(ctx context.Context, userID uuid.UUID, passID string) error
	SetPointsExpiry(ctx context.Context, userID uuid.UUID, expiresAt interface{}) error

	// Wallet (two-pool ledger)
	CreateWallet(ctx context.Context, wallet *entities.Wallet) error // ARCH-03: create wallet row on user registration
	GetWallet(ctx context.Context, userID uuid.UUID) (*entities.Wallet, error)
	GetWalletForUpdate(ctx context.Context, userID uuid.UUID) (*entities.Wallet, error) // SELECT FOR UPDATE
	UpdateWallet(ctx context.Context, wallet *entities.Wallet) error

	// Operational
	FindInactiveUsers(ctx context.Context, inactiveSinceHours int, limit int) ([]entities.User, error)
	FindUsersWithExpiringPoints(ctx context.Context, daysUntilExpiry int, limit int) ([]entities.User, error)
	CountByState(ctx context.Context, state string) (int64, error)
	UpdateState(ctx context.Context, userID uuid.UUID, state string) error
}
