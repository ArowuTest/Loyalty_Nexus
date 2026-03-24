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
	// 1. Check Eligibility (Backend-Driven)
	minRecharge := s.cfg.GetInt("min_recharge_naira", 500) * 100 // convert to kobo
	user, err := s.userRepo.FindByMSISDN(ctx, msisdn)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if user.TotalRechargeAmount < int64(minRecharge) {
		return nil, fmt.Errorf("not enough recharge for a spin")
	}

	// 2. Select Prize (CSPRNG Probability)
	prize, err := s.selectPrize(ctx)
	if err != nil {
		return nil, err
	}

	// 3. Record Transaction (Atomic Ledger via DB Trigger)
	tx := &entities.Transaction{
		ID:          uuid.New(),
		UserID:      user.ID,
		MSISDN:      msisdn,
		Type:        entities.TxTypeBonus,
		PointsDelta: prize.Value,
		Amount:      0,
		Metadata:    map[string]any{"prize_name": prize.Name, "engine": "nexus-v1"},
		CreatedAt:   time.Now(),
	}

	if err := s.txRepo.Save(ctx, tx); err != nil {
		return nil, err
	}

	return tx, nil
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
