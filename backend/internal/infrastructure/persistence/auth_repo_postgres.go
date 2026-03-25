package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type PostgresAuthRepository struct {
	db *gorm.DB
}

func NewPostgresAuthRepository(db *gorm.DB) repositories.AuthRepository {
	return &PostgresAuthRepository{db: db}
}

func (r *PostgresAuthRepository) CreateOTP(ctx context.Context, otp *entities.AuthOTP) error {
	return r.db.WithContext(ctx).Create(otp).Error
}

func (r *PostgresAuthRepository) FindPendingOTP(ctx context.Context, msisdn string, code string, purpose entities.OTPPurpose) (*entities.AuthOTP, error) {
	var otp entities.AuthOTP
	err := r.db.WithContext(ctx).Where("msisdn = ? AND code = ? AND purpose = ? AND status = 'pending' AND expires_at > ?", 
		msisdn, code, purpose, time.Now()).First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

func (r *PostgresAuthRepository) MarkOTPUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entities.AuthOTP{}).Where("id = ?", id).Update("status", "verified").Error
}
