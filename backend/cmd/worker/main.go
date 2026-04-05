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
	monetizationSvc := services.NewMonetizationService(db)
	
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

				// 2. HLR Validation
				if os.Getenv("OPERATION_MODE") == "integrated" {
					hlrSvc.DetectNetwork(ctx, event.MSISDN, nil)
				}

				processRecharge(ctx, event, userRepo, txRepo, cfg, db, passportSvc, monetizationSvc)
				rdb.XAck(ctx, streamName, groupName, msg.ID)
			}
		}
	}
}

func processRecharge(ctx context.Context, event queue.RechargeEvent, ur repositories.UserRepository, tr repositories.TransactionRepository, cfg *config.ConfigManager, db *gorm.DB, ps *services.PassportService, ms *services.MonetizationService) {
	isFirstRecharge := false
	user, err := ur.FindByMSISDN(ctx, event.MSISDN)
	
	lastActivity := time.Time{}
	if err == nil {
		lastActivity = user.LastVisitAt
	}

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

	// 1. Streak Calculation (REQ-2.5) with Grace Period (REQ-5.2.13)
	streakWindow := time.Duration(cfg.GetInt("streak_window_hours", 36)) * time.Hour
	graceLimit := cfg.GetInt("streak_freeze_grace_days_per_month", 1)
	
	now := time.Now()
	if !user.LastVisitAt.IsZero() {
		timeSinceLast := time.Since(user.LastVisitAt)
		if timeSinceLast <= streakWindow {
			user.StreakCount++
		} else {
			// Check if we can apply grace day (REQ-5.2.13)
			// Simple logic: if within (36h + 24h) and grace remains
			if timeSinceLast <= (streakWindow + 24*time.Hour) && user.StreakFreezeGraceUsed < graceLimit {
				user.StreakCount++
				user.StreakFreezeGraceUsed++
				log.Printf("[Worker] Grace Day Applied for %s", user.MSISDN)
			} else {
				user.StreakCount = 1
			}
		}
	} else {
		user.StreakCount = 1
	}
	user.LastVisitAt = now

	// 2. Dynamic Point Earning
	var tier struct {
		PointsPerNaira float64
	}
	db.Table("recharge_tiers").
		Where("min_amount_kobo <= ? AND is_active = true", event.Amount).
		Order("min_amount_kobo DESC").
		Limit(1).
		Select("points_per_naira").
		Scan(&tier)
	
	rate := tier.PointsPerNaira
	if rate == 0 { rate = 0.004 } 

	globalMultiplier := cfg.GetFloat("global_points_multiplier", 1.0)
	nairaAmount := float64(event.Amount) / 100
	pointsEarned := int64(nairaAmount * rate * globalMultiplier)

	// 3. Bonus Triggers
	var firstBonus int64
	if isFirstRecharge {
		db.Table("program_bonuses").Where("event_type = 'first_recharge' AND is_active = true").Pluck("bonus_points", &firstBonus)
		pointsEarned += firstBonus
	}

	var streakBonus int64
	db.Table("program_bonuses").
		Where("event_type = 'streak_milestone' AND threshold = ? AND is_active = true", user.StreakCount).
		Pluck("bonus_points", &streakBonus)
	pointsEarned += streakBonus

	user.TotalPoints += pointsEarned
	user.TotalRechargeAmount += event.Amount

	// REQ-5.2.14: Update Points Expiry (Rolling 90 days default)
	expiryDays := cfg.GetInt("points_expiry_days", 90)
	user.PointsExpiryDate = now.Add(time.Duration(expiryDays) * 24 * time.Hour)

	// Spin Credit Threshold
	spinThreshold := int64(cfg.GetInt("recharge_to_spin_naira", 1000) * 100)
	if user.TotalRechargeAmount >= spinThreshold {
		user.TotalRechargeAmount -= spinThreshold
		user.SpinCredits++
	}

	ur.Update(ctx, user)
	ms.TrackRechargeActivity(ctx, user.ID, user.MSISDN, event.Amount, lastActivity)
	ps.SyncWallet(ctx, user.ID.String(), user.TotalPoints, user.StreakCount, 500) 

	tx := &entities.Transaction{
		ID:          uuid.New(),
		MSISDN:      event.MSISDN,
		UserID:      user.ID,
		Type:        entities.TxTypeVisit,
		Amount:      event.Amount,
		PointsDelta: pointsEarned,
		CreatedAt:   time.Now(),
		Metadata:    map[string]any{"ref": event.Ref, "first_recharge": isFirstRecharge},
	}
	tr.Save(ctx, tx)
}
