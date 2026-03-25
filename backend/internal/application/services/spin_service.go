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
	// ... (Existing limit and eligibility checks) ...

	// 2. Select Prize (CSPRNG Probability)
	prize, err := s.selectPrize(ctx, forceLowValue)
	if err != nil {
		return nil, err
	}

	// 3. Record Transaction
	tx := &entities.Transaction{
		ID:          uuid.New(),
		UserID:      user.ID,
		MSISDN:      msisdn,
		Type:        entities.TxTypeBonus,
		PointsDelta: 0, // Delta depends on prize type
		CreatedAt:   time.Now(),
	}

	if prize.Name == "Bonus Points" {
		tx.PointsDelta = prize.Value
	}

	if err := s.txRepo.Save(ctx, tx); err != nil {
		return nil, err
	}

	// 4. Create Prize Claim & Trigger Fulfillment
	if prize.Name != "Try Again" {
		claim := &entities.PrizeClaim{
			UserID:        user.ID,
			TransactionID: tx.ID,
			PrizeType:     prize.Type, // need to add Type to prizeRow
			PrizeValue:    float64(prize.Value),
			Status:        entities.StatusPendingFulfillment,
		}
		s.prizeRepo.CreateClaim(ctx, claim)
		go s.fulfillmentSvc.Fulfill(context.Background(), claim)
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
