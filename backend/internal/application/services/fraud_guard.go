package services

import (
	"context"
	"time"
	"gorm.io/gorm"
)

type FraudGuard struct {
	db *gorm.DB
}

func NewFraudGuard(db *gorm.DB) *FraudGuard {
	return &FraudGuard{db: db}
}

func (g *FraudGuard) IsFraudulent(ctx context.Context, msisdn string, amount int64) (bool, string, error) {
	// 1. Blacklist Check
	var count int64
	err := g.db.WithContext(ctx).Table("msisdn_blacklist").Where("msisdn = ? AND is_active = true", msisdn).Count(&count).Error
	if err == nil && count > 0 {
		return true, "MSISDN Blacklisted", nil
	}

	// 2. Velocity Check
	hourAgo := time.Now().Add(-1 * time.Hour)
	err = g.db.WithContext(ctx).Table("transactions").Where("msisdn = ? AND created_at > ?", msisdn, hourAgo).Count(&count).Error
	if err == nil && count >= 5 {
		return true, "Transaction velocity exceeded", nil
	}

	return false, "", nil
}
