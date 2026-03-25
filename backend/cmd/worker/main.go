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

				processRecharge(ctx, event, userRepo, txRepo, cfg)
				rdb.XAck(ctx, streamName, groupName, msg.ID)
			}
		}
	}
}

func processRecharge(ctx context.Context, event queue.RechargeEvent, ur repositories.UserRepository, tr repositories.TransactionRepository, cfg *config.ConfigManager) {
	user, err := ur.FindByMSISDN(ctx, event.MSISDN)
	if err != nil {
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

	// 2. Points & Spin Credit Earning
	multiplier := cfg.GetFloat("global_points_multiplier", 1.0)
	pointsRate := cfg.GetInt("base_points_rate", 250)
	pointsEarned := int64(float64(event.Amount/int64(pointsRate*100)) * multiplier)

	user.TotalPoints += pointsEarned
	user.TotalRechargeAmount += event.Amount

	// Check Spin Credit Threshold
	spinThreshold := int64(cfg.GetInt("recharge_to_spin_naira", 1000) * 100)
	if user.TotalRechargeAmount >= spinThreshold {
		user.TotalRechargeAmount -= spinThreshold
		// Add logic to award spin credit in DB
	}

	ur.Update(ctx, user)

	tx := &entities.Transaction{
		ID:          uuid.New(),
		MSISDN:      event.MSISDN,
		UserID:      user.ID,
		Type:        entities.TxTypeVisit,
		Amount:      event.Amount,
		PointsDelta: pointsEarned,
		CreatedAt:   time.Now(),
		Metadata:    map[string]any{"source": "gateway", "ref": event.Ref},
	}

	tr.Save(ctx, tx)
}
