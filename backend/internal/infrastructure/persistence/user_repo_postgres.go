package persistence

import (
	"context"
	"database/sql"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
)

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) repositories.UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *entities.User) error {
	query := `INSERT INTO users (id, msisdn, user_code, tier, is_active, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, user.ID, user.MSISDN, user.UserCode, user.Tier, user.IsActive, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	u := &entities.User{}
	query := `SELECT id, msisdn, user_code, total_points, stamps_count, total_recharge_amount, tier, streak_count, last_visit_at, is_active, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.MSISDN, &u.UserCode, &u.TotalPoints, &u.StampsCount, &u.TotalRechargeAmount, &u.Tier, &u.StreakCount, &u.LastVisitAt, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *PostgresUserRepository) FindByMSISDN(ctx context.Context, msisdn string) (*entities.User, error) {
	u := &entities.User{}
	query := `SELECT id, msisdn, user_code, total_points, stamps_count, total_recharge_amount, tier, streak_count, last_visit_at, is_active, created_at, updated_at FROM users WHERE msisdn = $1`
	err := r.db.QueryRowContext(ctx, query, msisdn).Scan(&u.ID, &u.MSISDN, &u.UserCode, &u.TotalPoints, &u.StampsCount, &u.TotalRechargeAmount, &u.Tier, &u.StreakCount, &u.LastVisitAt, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *entities.User) error {
	query := `UPDATE users SET tier = $1, is_active = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, user.Tier, user.IsActive, user.UpdatedAt, user.ID)
	return err
}
