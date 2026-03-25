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
	userRepo      repositories.UserRepository
	txRepo        repositories.TransactionRepository
	prizeRepo     repositories.PrizeRepository
	fulfillmentSvc *PrizeFulfillmentService
	cfg           *config.ConfigManager
	db            *gorm.DB
}

func NewSpinService(ur repositories.UserRepository, tr repositories.TransactionRepository, pr repositories.PrizeRepository, fs *PrizeFulfillmentService, c *config.ConfigManager, db *gorm.DB) *SpinService {
	return &SpinService{
		userRepo:      ur,
		txRepo:        tr,
		prizeRepo:     pr,
		fulfillmentSvc: fs,
		cfg:           c,
		db:            db,
	}
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

	// 3. Check Eligibility (Spin Credits) - REQ-3.1
	user, err := s.userRepo.FindByMSISDN(ctx, msisdn)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if user.SpinCredits < 1 {
		return nil, fmt.Errorf("insufficient spin credits: recharge ₦1,000 to earn one")
	}

	// 4. Select Prize (CSPRNG Probability) - REQ-3.2
	prize, err := s.selectPrize(ctx, forceLowValue)
	if err != nil {
		return nil, err
	}

	// 5. Atomic deduction of 1 Spin Credit + Record Prize (Transaction)
	var finalTx *entities.Transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Deduct Credit
		user.SpinCredits--
		if err := s.userRepo.Update(ctx, user); err != nil {
			return err
		}

		// Record Transaction
		ledgerTx := &entities.Transaction{
			ID:          uuid.New(),
			UserID:      user.ID,
			MSISDN:      msisdn,
			Type:        entities.TxTypeBonus,
			PointsDelta: 0,
			CreatedAt:   time.Now(),
			Metadata:    map[string]any{"prize_name": prize.Name, "type": "spin_win"},
		}
		if prize.Name == "Bonus Points" {
			ledgerTx.PointsDelta = prize.Value
		}

		if err := s.txRepo.SaveTx(ctx, tx, ledgerTx); err != nil {
			return err
		}

		// Create Claim
		if prize.Name != "Try Again" {
			claim := &entities.PrizeClaim{
				UserID:        user.ID,
				TransactionID: ledgerTx.ID,
				PrizeType:     prize.Type,
				PrizeValue:    float64(prize.Value),
				Status:        entities.StatusPendingFulfillment,
			}
			if err := s.prizeRepo.CreateClaim(ctx, claim); err != nil {
				return err
			}
			go s.fulfillmentSvc.Fulfill(context.Background(), claim)
		}

		finalTx = ledgerTx
		return nil
	})

	return finalTx, err
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
	Type   string
	Value  int64
	Weight int
}

func (s *SpinService) selectPrize(ctx context.Context, forceLowValue bool) (*prizeRow, error) {
	var prizes []prizeRow
	query := s.db.WithContext(ctx).Table("prize_pool").Where("is_active = ?", true)
	if forceLowValue {
		query = query.Where("base_value <= ?", 50000) 
	}
	
	err := query.Select("name, prize_type as type, base_value as value, win_probability_weight as weight").Find(&prizes).Error
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
