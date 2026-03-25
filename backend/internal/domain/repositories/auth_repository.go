package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
)

type AuthRepository interface {
	CreateOTP(ctx context.Context, otp *entities.AuthOTP) error
	FindPendingOTP(ctx context.Context, msisdn string, code string, purpose entities.OTPPurpose) (*entities.AuthOTP, error)
	MarkOTPUsed(ctx context.Context, id uuid.UUID) error
}
