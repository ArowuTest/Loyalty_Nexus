package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SpinService struct {
	userRepo repositories.UserRepository
	txRepo   repositories.TransactionRepository
	cfg      *config.ConfigManager
	db       *gorm.DB
}

func NewSpinService(ur repositories.UserRepository, tr repositories.TransactionRepository, c *config.ConfigManager, db *gorm.DB) *SpinService {
	return &SpinService{userRepo: ur, txRepo: tr, cfg: c, db: db}
}

func (s *SpinService) PlaySpin(ctx context.Context, msisdn string) (*entities.Transaction, error) {
	// 1. Daily Spin Limit (REQ-3.6)
	spinCount, _ := s.getDailySpinCount(ctx, msisdn)
	if spinCount >= 3 {
		return nil, fmt.Errorf("daily spin limit reached (max 3)")
	}

	// 2. Check Daily Liability Cap (REQ-3.5)
	dailyCap := int64(s.cfg.GetInt("daily_prize_liability_cap_naira", 500000) * 100)
	currentLiability, _ := s.getCurrentDailyLiability(ctx)
	forceLowValue := currentLiability >= dailyCap

	// 3. Check Eligibility (Backend-Driven)
	minRecharge := s.cfg.GetInt("min_recharge_naira", 500) * 100 // convert to kobo
	user, err := s.userRepo.FindByMSISDN(ctx, msisdn)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if user.TotalRechargeAmount < int64(minRecharge) {
		return nil, fmt.Errorf("not enough recharge for a spin")
	}

	// 4. Select Prize (CSPRNG Probability)
	prize, err := s.selectPrize(ctx, forceLowValue)
	if err != nil {
		return nil, err
	}

	// 5. Record Transaction (Atomic Ledger)
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

func (s *SpinService) getDailySpinCount(ctx context.Context, msisdn string) (int, error) {
	var count int64
	err := s.db.WithContext(ctx).Table("transactions").
		Where("msisdn = ? AND type = 'spin_play' AND created_at >= CURRENT_DATE", msisdn).
		Count(&count).Error
	return int(count), err
}

func (s *SpinService) getCurrentDailyLiability(ctx context.Context) (int64, error) {
	var total int64
	query := "SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'prize_award' AND created_at >= CURRENT_DATE"
	err := s.db.WithContext(ctx).Raw(query).Scan(&total).Error
	return total, err
}

type prizeRow struct {
	Name   string
	Value  int64
	Weight int
}

func (s *SpinService) selectPrize(ctx context.Context, forceLowValue bool) (*prizeRow, error) {
	var prizes []prizeRow
	query := s.db.WithContext(ctx).Table("prize_pool").Where("is_active = ?", true)
	if forceLowValue {
		// Logic to restrict high-value prizes if cap is reached
		query = query.Where("base_value <= ?", 50000) // e.g., max N500 prizes
	}
	
	err := query.Select("name, base_value, win_probability_weight").Find(&prizes).Error
	if err != nil {
		return nil, err
	}

	totalWeight := 0
	for _, p := range prizes {
		totalWeight += p.Weight
	}

	if totalWeight == 0 {
		return nil, fmt.Errorf("no active prizes configured")
	}

	randomWeight, _ := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	current := int64(0)
	for _, p := range prizes {
		current += int64(p.Weight)
		if randomWeight.Int64() < current {
			return &p, nil
		}
	}

	return &prizes[0], nil
}
