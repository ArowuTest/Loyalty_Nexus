package services

import (
	"context"
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MonetizationService struct {
	db *gorm.DB
}

func NewMonetizationService(db *gorm.DB) *MonetizationService {
	return &MonetizationService{db: db}
}

// TrackRechargeActivity updates monetization metrics after a successful recharge (Section 6)
func (s *MonetizationService) TrackRechargeActivity(ctx context.Context, userID uuid.UUID, msisdn string, amount int64, lastVisit time.Time) error {
	// 1. Churn Recovery Bounty (Section 6.3)
	// Definition of "At-Risk": No activity for > 14 days (configurable)
	atRiskThreshold := 14 * 24 * time.Hour
	if !lastVisit.IsZero() && time.Since(lastVisit) > atRiskThreshold {
		bounty := map[string]interface{}{
			"user_id": userID,
			"last_activity_before_reactivation": lastVisit,
			"bounty_amount_kobo": 15000, // ₦150
		}
		s.db.WithContext(ctx).Table("churn_recovery_bounties").Create(bounty)
	}

	// 2. Monthly ARPU Uplift Tracking (Section 6.2)
	monthStart := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -time.Now().Day()+1)
	
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		tx.Table("arpu_uplift_tracking").
			Where("user_id = ? AND month_period = ?", userID, monthStart).
			Count(&count)

		if count == 0 {
			// Initialize monthly snapshot
			return tx.Table("arpu_uplift_tracking").Create(map[string]interface{}{
				"user_id":      userID,
				"msisdn":       msisdn,
				"month_period": monthStart,
				"current_month_spend": amount,
			}).Error
		}
		
		// Increment monthly spend
		return tx.Table("arpu_uplift_tracking").
			Where("user_id = ? AND month_period = ?", userID, monthStart).
			Update("current_month_spend", gorm.Expr("current_month_spend + ?", amount)).Error
	})
}

// TrackStudioUsage tracks GPU/API costs for AI rendering (Section 6.4)
func (s *MonetizationService) TrackStudioUsage(ctx context.Context, genID uuid.UUID, provider string, costMicros int) error {
	usage := map[string]interface{}{
		"generation_id":      genID,
		"provider":           provider,
		"compute_cost_micros": costMicros,
	}
	return s.db.WithContext(ctx).Table("studio_usage_metrics").Create(usage).Error
}
