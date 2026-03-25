package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/persistence"
	"loyalty-nexus/internal/infrastructure/queue"
	"loyalty-nexus/internal/infrastructure/external"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})

	userRepo := persistence.NewPostgresUserRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)
	hlrRepo := persistence.NewPostgresHLRRepository(db)
	fraudGuard := services.NewFraudGuard(db)
	hlrSvc := services.NewHLRService(hlrRepo)
	
	walletAdapter := &external.RebitesWalletAdapter{} 
	passportSvc := services.NewPassportService(walletAdapter) 

	cfg := config.NewConfigManager(db)
	cfg.Refresh(context.Background())

	ctx := context.Background()
	streamName := "recharge_stream"
	groupName := "nexus_processors"

	rdb.XGroupCreateMkStream(ctx, streamName, groupName, "0")

	log.Printf("Loyalty Nexus Worker started.")

	for {
		entries, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: "worker-1",
			Streams:  []string{streamName, ">"},
			Count:    10,
			Block:    0,
		}).Result()

		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		for _, stream := range entries {
			for _, msg := range stream.Messages {
				var event queue.RechargeEvent
				json.Unmarshal([]byte(msg.Values["payload"].(string)), &event)

				// 1. Fraud Check
				isFraud, reason, _ := fraudGuard.IsFraudulent(ctx, event.MSISDN, event.Amount)
				if isFraud {
					log.Printf("[Worker] Fraud Blocked: %s | Reason: %s", event.MSISDN, reason)
					rdb.XAck(ctx, streamName, groupName, msg.ID)
					continue
				}

				// 2. HLR Validation (if integrated)
				if os.Getenv("OPERATION_MODE") == "integrated" {
					hlrSvc.DetectNetwork(ctx, event.MSISDN, nil)
				}

				processRecharge(ctx, event, userRepo, txRepo, cfg, db, passportSvc)
				rdb.XAck(ctx, streamName, groupName, msg.ID)
			}
		}
	}
}

func processRecharge(ctx context.Context, event queue.RechargeEvent, ur repositories.UserRepository, tr repositories.TransactionRepository, cfg *config.ConfigManager, db *gorm.DB, ps *services.PassportService) {
	isFirstRecharge := false
	user, err := ur.FindByMSISDN(ctx, event.MSISDN)
	if err != nil {
		isFirstRecharge = true
		user = &entities.User{
			ID:        uuid.New(),
			MSISDN:    event.MSISDN,
			UserCode:  fmt.Sprintf("NEX%s", uuid.New().String()[:6]),
			Tier:      "BRONZE",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := ur.Create(ctx, user); err != nil {
			log.Printf("Failed to create user: %v", err)
			return
		}
	}

	// 1. Streak Calculation (REQ-2.5)
	streakWindow := time.Duration(cfg.GetInt("streak_window_hours", 36)) * time.Hour
	if !user.LastVisitAt.IsZero() && time.Since(user.LastVisitAt) <= streakWindow {
		user.StreakCount++
	} else {
		user.StreakCount = 1
	}
	user.LastVisitAt = time.Now()

	// 2. Dynamic Point Earning (REQ-5.2.3)
	var tier struct {
		PointsPerNaira float64
	}
	// Select highest tier rate (REQ-5.2.3)
	err = db.Table("recharge_tiers").
		Where("min_amount_kobo <= ? AND is_active = true", event.Amount).
		Order("min_amount_kobo DESC").
		Limit(1).
		Select("points_per_naira").
		Scan(&tier).Error
	
	rate := tier.PointsPerNaira
	if rate == 0 { rate = 0.004 } // Default N250 = 1pt

	// Multipliers (Global + Regional)
	globalMultiplier := cfg.GetFloat("global_points_multiplier", 1.0)
	regionalMultiplier := 1.0
	if user.State != "" {
		var reg struct {
			GoldenHourMultiplier float64
			IsGoldenHour bool
			BaseMultiplier float64
		}
		if err := db.Table("regional_settings").Where("region_name = ?", user.State).First(&reg).Error; err == nil {
			regionalMultiplier = reg.BaseMultiplier
			if reg.IsGoldenHour { regionalMultiplier = reg.GoldenHourMultiplier }
		}
	}

	nairaAmount := float64(event.Amount) / 100
	pointsEarned := int64(nairaAmount * rate * globalMultiplier * regionalMultiplier)

	// 3. Bonus Triggers (REQ-5.2.8, 5.2.9)
	var bonuses int64
	if isFirstRecharge {
		db.Table("program_bonuses").Where("event_type = 'first_recharge' AND is_active = true").Pluck("bonus_points", &bonuses)
		pointsEarned += bonuses
	}

	// Streak milestone bonus
	var streakBonus int64
	db.Table("program_bonuses").
		Where("event_type = 'streak_milestone' AND threshold = ? AND is_active = true", user.StreakCount).
		Pluck("bonus_points", &streakBonus)
	pointsEarned += streakBonus

	user.TotalPoints += pointsEarned
	user.TotalRechargeAmount += event.Amount

	// Check Spin Credit Threshold (REQ-2.3)
	spinThreshold := int64(cfg.GetInt("recharge_to_spin_naira", 1000) * 100)
	if user.TotalRechargeAmount >= spinThreshold {
		user.TotalRechargeAmount -= spinThreshold
		user.SpinCredits++
	}

	ur.Update(ctx, user)
	ps.SyncWallet(ctx, user.ID.String(), user.TotalPoints, user.StreakCount, 500) 

	tx := &entities.Transaction{
		ID:          uuid.New(),
		MSISDN:      event.MSISDN,
		UserID:      user.ID,
		Type:        entities.TxTypeVisit,
		Amount:      event.Amount,
		PointsDelta: pointsEarned,
		CreatedAt:   time.Now(),
		Metadata:    map[string]any{"ref": event.Ref, "tier_rate": rate, "first_recharge": isFirstRecharge},
	}
	tr.Save(ctx, tx)
}
