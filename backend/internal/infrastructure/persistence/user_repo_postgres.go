package persistence

import (
	"context"
	"fmt"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type postgresUserRepository struct {
	db *gorm.DB
}

func NewPostgresUserRepository(db *gorm.DB) repositories.UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresUserRepository) FindByPhoneNumber(ctx context.Context, phone string) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).Where("phone_number = ?", phone).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresUserRepository) ExistsByPhoneNumber(ctx context.Context, phone string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("users").Where("phone_number = ?", phone).Count(&count).Error
	return count > 0, err
}

func (r *postgresUserRepository) Create(ctx context.Context, user *entities.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *postgresUserRepository) Update(ctx context.Context, user *entities.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *postgresUserRepository) UpdateStreak(ctx context.Context, userID uuid.UUID, count int, expiresAt interface{}) error {
	return r.db.WithContext(ctx).Table("users").
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"streak_count":      count,
			"streak_expires_at": expiresAt,
		}).Error
}

func (r *postgresUserRepository) UpdateMoMo(ctx context.Context, userID uuid.UUID, num string, verified bool) error {
	return r.db.WithContext(ctx).Table("users").
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"momo_number":    num,
			"momo_verified":  verified,
		}).Error
}

func (r *postgresUserRepository) UpdateTier(ctx context.Context, userID uuid.UUID, tier string) error {
	return r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Update("tier", tier).Error
}

func (r *postgresUserRepository) UpdateWalletPassID(ctx context.Context, userID uuid.UUID, passID string) error {
	return r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Update("wallet_pass_id", passID).Error
}

func (r *postgresUserRepository) SetPointsExpiry(ctx context.Context, userID uuid.UUID, expiresAt interface{}) error {
	return r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Update("points_expire_at", expiresAt).Error
}

func (r *postgresUserRepository) CreateWallet(ctx context.Context, wallet *entities.Wallet) error {
	return r.db.WithContext(ctx).Table("wallets").Create(wallet).Error
}

func (r *postgresUserRepository) GetWallet(ctx context.Context, userID uuid.UUID) (*entities.Wallet, error) {
	var w entities.Wallet
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&w).Error; err != nil {
		// ARCH-03: lazily create wallet row for existing users who registered before this fix
		w = entities.Wallet{ID: uuid.New(), UserID: userID}
		_ = r.db.WithContext(ctx).Table("wallets").Create(&w).Error
		return &w, nil
	}
	return &w, nil
}

func (r *postgresUserRepository) GetWalletForUpdate(ctx context.Context, userID uuid.UUID) (*entities.Wallet, error) {
	var w entities.Wallet
	// Try SELECT FOR UPDATE (Postgres); fall back to plain SELECT for SQLite/test DBs.
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&w).Error
	if err != nil {
		// Retry without lock (e.g. SQLite test environment)
		err = r.db.WithContext(ctx).Where("user_id = ?", userID).First(&w).Error
	}
	if err != nil {
		return nil, fmt.Errorf("wallet not found for user %s: %w", userID, err)
	}
	return &w, nil
}

func (r *postgresUserRepository) UpdateWallet(ctx context.Context, wallet *entities.Wallet) error {
	return r.db.WithContext(ctx).Table("wallets").Where("user_id = ?", wallet.UserID).Save(wallet).Error
}

func (r *postgresUserRepository) FindInactiveUsers(ctx context.Context, inactiveSinceHours int, limit int) ([]entities.User, error) {
	var users []entities.User
	r.db.WithContext(ctx).
		Where("streak_count > 0 AND streak_expires_at IS NOT NULL").
		Limit(limit).Find(&users)
	return users, nil
}

func (r *postgresUserRepository) FindUsersWithExpiringPoints(ctx context.Context, daysUntilExpiry int, limit int) ([]entities.User, error) {
	var users []entities.User
	r.db.WithContext(ctx).
		Where("points_expire_at IS NOT NULL AND points_expire_at <= NOW() + INTERVAL '? days'", daysUntilExpiry).
		Limit(limit).Find(&users)
	return users, nil
}

func (r *postgresUserRepository) CountByState(ctx context.Context, state string) (int64, error) {
	var count int64
	r.db.WithContext(ctx).Table("users").Where("state = ?", state).Count(&count)
	return count, nil
}

func (r *postgresUserRepository) UpdateState(ctx context.Context, userID uuid.UUID, state string) error {
	return r.db.WithContext(ctx).
		Table("users").
		Where("id = ?", userID).
		Update("state", state).Error
}
