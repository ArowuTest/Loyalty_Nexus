// Package utils provides internal utility helpers for the Loyalty Nexus backend,
// including spin-tier calculation logic.
package utils

import (
	"fmt"
	"gorm.io/gorm"
	"loyalty-nexus/internal/domain/entities"
)

// SpinTierCalculatorDB handles spin tier calculations using database
type SpinTierCalculatorDB struct {
	db *gorm.DB
}

// NewSpinTierCalculatorDB creates a new database-driven spin tier calculator
func NewSpinTierCalculatorDB(db *gorm.DB) *SpinTierCalculatorDB {
	return &SpinTierCalculatorDB{db: db}
}

// GetSpinTierFromDB returns the spin tier for a given daily recharge amount (in kobo)
func (c *SpinTierCalculatorDB) GetSpinTierFromDB(dailyRechargeAmountKobo int64) (*entities.SpinTier, error) {
	if dailyRechargeAmountKobo < 100000 { // ₦1,000 in kobo
		return nil, fmt.Errorf("minimum recharge amount for spins is ₦1,000")
	}

	var tier entities.SpinTier
	err := c.db.Where("is_active = ? AND min_daily_amount <= ? AND max_daily_amount >= ?",
		true, dailyRechargeAmountKobo, dailyRechargeAmountKobo).
		Order("sort_order ASC").
		First(&tier).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Convert kobo to Naira for error message
			nairaAmount := float64(dailyRechargeAmountKobo) / 100.0
			return nil, fmt.Errorf("unable to determine spin tier for amount: ₦%.2f", nairaAmount)
		}
		return nil, fmt.Errorf("database error while fetching spin tier: %v", err)
	}

	return &tier, nil
}

// GetAllTiersFromDB returns all active spin tiers ordered by sort_order
func (c *SpinTierCalculatorDB) GetAllTiersFromDB() ([]entities.SpinTier, error) {
	var tiers []entities.SpinTier
	err := c.db.Where("is_active = ?", true).
		Order("sort_order ASC").
		Find(&tiers).Error

	if err != nil {
		return nil, fmt.Errorf("database error while fetching all tiers: %v", err)
	}

	return tiers, nil
}
