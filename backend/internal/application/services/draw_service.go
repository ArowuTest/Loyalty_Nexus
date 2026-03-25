package services

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DrawService struct {
	db *gorm.DB
}

func NewDrawService(db *gorm.DB) *DrawService {
	return &DrawService{db: db}
}

// ExecuteDraw selects winners for a given draw using CSPRNG (REQ-3.2)
func (s *DrawService) ExecuteDraw(ctx context.Context, drawID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Fetch Draw Details
		var draw struct {
			ID     uuid.UUID
			Status string
		}
		if err := tx.Table("draws").Where("id = ? AND status = 'ACTIVE'", drawID).First(&draw).Error; err != nil {
			return fmt.Errorf("active draw not found")
		}

		// 2. Fetch Unique Entries (MSISDNs)
		var entries []string
		if err := tx.Table("draw_entries").Where("draw_id = ?", drawID).Pluck("msisdn", &entries).Error; err != nil {
			return err
		}

		if len(entries) == 0 {
			return fmt.Errorf("no entries for draw")
		}

		// 3. SEC-009: Crypto-Random Shuffle (RECHARGEMAX_AUDIT.go)
		s.cryptoShuffle(entries)

		// 4. Select Winners (Mocking 3 winners for now)
		winnerCount := 3
		if len(entries) < winnerCount {
			winnerCount = len(entries)
		}

		for i := 0; i < winnerCount; i++ {
			msisdn := entries[i]
			
			// Find User ID
			var userID uuid.UUID
			tx.Table("users").Where("msisdn = ?", msisdn).Pluck("id", &userID)

			winner := map[string]interface{}{
				"draw_id":     drawID,
				"user_id":     userID,
				"msisdn":      msisdn,
				"position":    i + 1,
				"prize_name":  "MoMo Cash", // Assuming MoMo Cash for Daily Draw
				"prize_value": 50000,      // ₦50,000 Jackpot (Strategy Sec 5)
			}

			if err := tx.Table("draw_winners").Create(winner).Error; err != nil {
				return err
			}
		}

		// 5. Finalize Draw
		return tx.Table("draws").Where("id = ?", drawID).Updates(map[string]interface{}{
			"status":      "COMPLETED",
			"executed_at": time.Now(),
		}).Error
	})
}

func (s *DrawService) cryptoShuffle(s []string) {
	for i := len(s) - 1; i > 0; i-- {
		var b [8]byte
		crand.Read(b[:])
		j := int(binary.BigEndian.Uint64(b[:]) % uint64(i+1))
		s[i], s[j] = s[j], s[i]
	}
}
