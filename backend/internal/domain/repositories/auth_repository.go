package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
)

type AuthRepository interface {
	CreateOTP(ctx context.Context, otp *entities.AuthOTP) error
	FindLatestPendingOTP(ctx context.Context, phone, purpose string) (*entities.AuthOTP, error)
	MarkOTPUsed(ctx context.Context, id uuid.UUID) error
	ExpireOTP(ctx context.Context, id uuid.UUID) error
	ExpireOldOTPs(ctx context.Context) (int64, error) // Called by cron worker

	// Admin
	FindAdminByUsername(ctx context.Context, username string) (*entities.AdminUser, error)
	FindAdminByID(ctx context.Context, id uuid.UUID) (*entities.AdminUser, error)
}
