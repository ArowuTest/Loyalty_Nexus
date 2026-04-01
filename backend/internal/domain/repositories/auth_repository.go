package repositories

import (
	"context"
	"github.com/google/uuid"
	"loyalty-nexus/internal/domain/entities"
	"time"
)

type AuthRepository interface {
	CreateOTP(ctx context.Context, otp *entities.AuthOTP) error
	FindLatestPendingOTP(ctx context.Context, phone, purpose string) (*entities.AuthOTP, error)
	MarkOTPUsed(ctx context.Context, id uuid.UUID) error
	ExpireOTP(ctx context.Context, id uuid.UUID) error
	ExpireOldOTPs(ctx context.Context) (int64, error) // Called by cron worker
	CountRecentOTPs(ctx context.Context, phone string, since time.Time) (int64, error)

	// Admin
	FindAdminByUsername(ctx context.Context, username string) (*entities.AdminUser, error)
	FindAdminByID(ctx context.Context, id uuid.UUID) (*entities.AdminUser, error)
}
