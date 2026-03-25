package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type FraudGuard struct {
	db *sql.DB
}

func NewFraudGuard(db *sql.DB) *FraudGuard {
	return &FraudGuard{db: db}
}

func (g *FraudGuard) IsFraudulent(ctx context.Context, msisdn string, amount int64) (bool, string, error) {
	// 1. Blacklist Check
	var count int
	err := g.db.QueryRowContext(ctx, "SELECT count(*) FROM msisdn_blacklist WHERE msisdn = $1 AND is_active = true", msisdn).Scan(&count)
	if err == nil && count > 0 {
		return true, "MSISDN Blacklisted", nil
	}

	// 2. Velocity Check (e.g., max 5 recharges in 1 hour)
	hourAgo := time.Now().Add(-1 * time.Hour)
	err = g.db.QueryRowContext(ctx, "SELECT count(*) FROM transactions WHERE msisdn = $1 AND created_at > $2", msisdn, hourAgo).Scan(&count)
	if err == nil && count >= 5 {
		return true, "Transaction velocity exceeded", nil
	}

	return false, "", nil
}
