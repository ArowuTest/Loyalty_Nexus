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

// IsFraudulent performs multi-layer fraud checks (Strategic innovation Section 4).
//
// Schema alignment (fixed 2026-03-31):
//   Migration 020 renamed msisdn → phone_number in msisdn_blacklist and transactions.
//   Migration 060 recreated msisdn_blacklist with phone_number; there is no is_active
//   column — every row in the table is an active block.
//   Transaction type 'visit' does not exist; daily cap now uses type = 'recharge'.
func (g *FraudGuard) IsFraudulent(ctx context.Context, phoneNumber string, amount int64) (bool, string, error) {
	// 1. Phone Number Blacklist Check
	// msisdn_blacklist schema (migration 060): id, phone_number, reason, created_at
	// No is_active column — every row is an active block.
	var blacklistCount int64
	err := g.db.WithContext(ctx).Table("msisdn_blacklist").
		Where("phone_number = ?", phoneNumber).
		Count(&blacklistCount).Error
	if err == nil && blacklistCount > 0 {
		return true, "Phone number is blacklisted", nil
	}

	// 2. Velocity Check: Max 15 recharge transactions per hour (Strategy Section 4).
	// transactions.phone_number is the correct column post migration 020.
	windowStart := time.Now().Add(-1 * time.Hour)
	var txCount int64
	err = g.db.WithContext(ctx).Table("transactions").
		Where("phone_number = ? AND created_at >= ?", phoneNumber, windowStart).
		Count(&txCount).Error
	if err == nil && txCount >= 15 {
		return true, fmt.Sprintf("Velocity exceeded: %d transactions in 1 hour", txCount), nil
	}

	// 3. Daily Cumulative Cap: ₦200,000 per phone number.
	// type = 'recharge' matches TxTypeRecharge constant (migration 020 renamed 'visit').
	dayStart := time.Now().Truncate(24 * time.Hour)
	var dailyTotal int64
	err = g.db.WithContext(ctx).Table("transactions").
		Select("COALESCE(SUM(amount), 0)").
		Where("phone_number = ? AND type = 'recharge' AND created_at >= ?", phoneNumber, dayStart).
		Scan(&dailyTotal).Error
	if err == nil && (dailyTotal+amount) > 20000000 { // 200,000 Naira in Kobo
		return true, "Daily cumulative recharge limit exceeded", nil
	}

	return false, "", nil
}
