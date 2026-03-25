package services

import (
	"context"
	"database/sql"
	"log"
	"time"
	"gorm.io/gorm"
	"github.com/google/uuid"
)

type MonetizationService struct {
	db *gorm.DB
}

func NewMonetizationService(db *gorm.DB) *MonetizationService {
	return &MonetizationService{db: db}
}

// TrackRechargeActivity updates monetization metrics after a successful recharge
func (s *MonetizationService) TrackRechargeActivity(ctx context.Context, userID uuid.UUID, msisdn string, amount int64, lastVisit time.Time) error {
	// 1. Check for Churn Recovery Bounty (Innovation 6.3)
	// Definition of "At-Risk": No activity for > 14 days (configurable)
	atRiskThreshold := 14 * 24 * time.Hour
	if !lastVisit.IsZero() && time.Since(lastVisit) > atRiskThreshold {
		bounty := map[string]interface{}{
			"user_id": userID,
			"last_activity_before_reactivation": lastVisit,
			"bounty_amount_kobo": 15000, // ₦150
		}
		s.db.Table("churn_recovery_bounties").Create(bounty)
		log.Printf("[Monetization] Churn Recovery Bounty Tracked for %s", msisdn)
	}

	// 2. Track Monthly ARPU Uplift (Innovation 6.2)
	monthStart := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -time.Now().Day() + 1)
	
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		tx.Table("arpu_uplift_tracking").
			Where("user_id = ? AND month_period = ?", userID, monthStart).
			Count(&count)

		if count == 0 {
			// First recharge this month, initialize snapshot
			tx.Table("arpu_uplift_tracking").Create(map[string]interface{}{
				"user_id":      userID,
				"msisdn":       msisdn,
				"month_period": monthStart,
				"current_month_spend": amount,
			})
		} else {
			// Increment current spend
			tx.Table("arpu_uplift_tracking").
				Where("user_id = ? AND month_period = ?", userID, monthStart).
				Update("current_month_spend", gorm.Expr("current_month_spend + ?", amount))
		}
		return nil
	})

	return err
}
