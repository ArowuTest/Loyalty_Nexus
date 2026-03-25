package services

import (
	"context"
	"fmt"
	"time"
	"gorm.io/gorm"
)

type FraudGuard struct {
	db *gorm.DB
}

func NewFraudGuard(db *gorm.DB) *FraudGuard {
	return &FraudGuard{db: db}
}

// IsFraudulent performs multi-layer fraud checks (Strategic innovation Section 4)
func (g *FraudGuard) IsFraudulent(ctx context.Context, msisdn string, amount int64) (bool, string, error) {
	// 1. MSISDN Blacklist Check
	var blacklistCount int64
	err := g.db.WithContext(ctx).Table("msisdn_blacklist").
		Where("msisdn = ? AND is_active = true", msisdn).
		Count(&blacklistCount).Error
	if err == nil && blacklistCount > 0 {
		return true, "MSISDN is blacklisted", nil
	}

	// 2. Velocity Check: Max 15 transactions per hour (Strategy Section 4)
	windowStart := time.Now().Add(-1 * time.Hour)
	var txCount int64
	err = g.db.WithContext(ctx).Table("transactions").
		Where("msisdn = ? AND created_at >= ?", msisdn, windowStart).
		Count(&txCount).Error
	if err == nil && txCount >= 15 {
		return true, fmt.Sprintf("Velocity exceeded: %d transactions in 1 hour", txCount), nil
	}

	// 3. Daily Cumulative Cap: ₦200,000 per MSISDN
	dayStart := time.Now().Truncate(24 * time.Hour)
	var dailyTotal int64
	err = g.db.WithContext(ctx).Table("transactions").
		Select("COALESCE(SUM(amount), 0)").
		Where("msisdn = ? AND type = 'visit' AND created_at >= ?", msisdn, dayStart).
		Scan(&dailyTotal).Error
	if err == nil && (dailyTotal + amount) > 20000000 { // 200,000 Naira in Kobo
		return true, "Daily cumulative recharge limit exceeded", nil
	}

	return false, "", nil
}
