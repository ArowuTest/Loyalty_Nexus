package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"time"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
	"github.com/google/uuid"
)

type SpinService struct {
	userRepo repositories.UserRepository
	txRepo   repositories.TransactionRepository
	cfg      *config.ConfigManager
	db       *sql.DB
}

func NewSpinService(ur repositories.UserRepository, tr repositories.TransactionRepository, c *config.ConfigManager, db *sql.DB) *SpinService {
	return &SpinService{userRepo: ur, txRepo: tr, cfg: c, db: db}
}

func (s *SpinService) PlaySpin(ctx context.Context, msisdn string) (*entities.Transaction, error) {
	// 1. Check Daily Liability Cap (REQ-3.5)
	dailyCap := int64(s.cfg.GetInt("daily_prize_liability_cap_naira", 500000) * 100)
	currentLiability, _ := s.getCurrentDailyLiability(ctx)
	
	forceLowValue := currentLiability >= dailyCap

	// 2. Check Eligibility (Backend-Driven)
	// ... (Eligibility checks) ...

	// 3. Select Prize (CSPRNG Probability)
	// If cap reached, only allow points or "Try Again"
	prize, err := s.selectPrize(ctx, forceLowValue)
	// ...
}

func (s *SpinService) getCurrentDailyLiability(ctx context.Context) (int64, error) {
	var total int64
	query := "SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'prize_award' AND created_at >= CURRENT_DATE"
	err := s.db.QueryRowContext(ctx, query).Scan(&total)
	return total, err
}

type prizeRow struct {
	Name   string
	Value  int64
	Weight int
}

func (s *SpinService) selectPrize(ctx context.Context) (*prizeRow, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name, base_value, win_probability_weight FROM prize_pool WHERE is_active = true")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prizes []prizeRow
	totalWeight := 0
	for rows.Next() {
		var p prizeRow
		if err := rows.Scan(&p.Name, &p.Value, &p.Weight); err != nil {
			return nil, err
		}
		prizes = append(prizes, p)
		totalWeight += p.Weight
	}

	if totalWeight == 0 {
		return nil, fmt.Errorf("no active prizes configured in database")
	}

	randomWeight, err := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	if err != nil {
		return nil, err
	}

	current := int64(0)
	for _, p := range prizes {
		current += int64(p.Weight)
		if randomWeight.Int64() < current {
			return &p, nil
		}
	}

	return &prizes[0], nil
}
