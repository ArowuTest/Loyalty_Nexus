package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type postgresAuthRepository struct{ db *gorm.DB }

func NewPostgresAuthRepository(db *gorm.DB) repositories.AuthRepository {
	return &postgresAuthRepository{db: db}
}

func (r *postgresAuthRepository) CreateOTP(ctx context.Context, otp *entities.AuthOTP) error {
	return r.db.WithContext(ctx).Create(otp).Error
}

func (r *postgresAuthRepository) FindLatestPendingOTP(ctx context.Context, phone, purpose string) (*entities.AuthOTP, error) {
	var otp entities.AuthOTP
	err := r.db.WithContext(ctx).
		Where("phone_number = ? AND purpose = ? AND status = 'pending' AND expires_at > NOW()", phone, purpose).
		Order("created_at DESC").First(&otp).Error
	return &otp, err
}

func (r *postgresAuthRepository) MarkOTPUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Table("auth_otps").Where("id = ?", id).Update("status", "verified").Error
}

func (r *postgresAuthRepository) ExpireOTP(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Table("auth_otps").Where("id = ?", id).Update("status", "expired").Error
}

func (r *postgresAuthRepository) ExpireOldOTPs(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Table("auth_otps").
		Where("status = 'pending' AND expires_at < NOW()").Update("status", "expired")
	return result.RowsAffected, result.Error
}

func (r *postgresAuthRepository) FindAdminByUsername(ctx context.Context, username string) (*entities.AdminUser, error) {
	var admin entities.AdminUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error
	return &admin, err
}

func (r *postgresAuthRepository) FindAdminByID(ctx context.Context, id uuid.UUID) (*entities.AdminUser, error) {
	var admin entities.AdminUser
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&admin).Error
	return &admin, err
}
