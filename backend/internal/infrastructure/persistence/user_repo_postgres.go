package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresUserRepository struct {
	db *gorm.DB
}

func NewPostgresUserRepository(db *gorm.DB) repositories.UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *entities.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) FindByMSISDN(ctx context.Context, msisdn string) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).First(&user, "msisdn = ?", msisdn).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *entities.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}
