import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/persistence"
	"loyalty-nexus/internal/infrastructure/queue"
	"loyalty-nexus/internal/application/services"
)

func main() {
	// ... existing db/redis setup ...
	db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL")})

	userRepo := persistence.NewPostgresUserRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)
	hlrRepo := persistence.NewPostgresHLRRepository(db)
	fraudGuard := services.NewFraudGuard(db)
	hlrSvc := services.NewHLRService(hlrRepo) 

	ctx := context.Background()
	// ... stream setup ...

	for {
		// ... read group ...
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

				// 2. HLR Validation (Integrated Mode)
				if os.Getenv("OPERATION_MODE") == "integrated" {
					res, err := hlrSvc.DetectNetwork(ctx, event.MSISDN, nil)
					if err != nil {
						log.Printf("[Worker] HLR Failed for %s: %v", event.MSISDN, err)
						// Handle error (e.g., dead letter queue or retry)
					} else {
						log.Printf("[Worker] Validated MSISDN %s on %s", res.MSISDN, res.Network)
					}
				}

				processRecharge(ctx, event, userRepo, txRepo)
				rdb.XAck(ctx, streamName, groupName, msg.ID)
			}
		}
	}
}

func processRecharge(ctx context.Context, event queue.RechargeEvent, ur repositories.UserRepository, tr repositories.TransactionRepository, cfg *config.ConfigManager) {
	user, err := ur.FindByMSISDN(ctx, event.MSISDN)
	// ... (user creation logic) ...

	// 1. Streak Calculation (REQ-2.5)
	streakWindow := time.Duration(cfg.GetInt("streak_window_hours", 36)) * time.Hour
	if !user.LastVisitAt.IsZero() && time.Since(user.LastVisitAt) <= streakWindow {
		user.StreakCount++
	} else {
		user.StreakCount = 1
	}
	user.LastVisitAt = time.Now()

	// 2. Points & Spin Credit Earning (REQ-2.2, REQ-2.3)
	// Apply global multiplier (REQ-2.6)
	multiplier := cfg.GetFloat("global_points_multiplier", 1.0)
	pointsRate := cfg.GetInt("base_points_rate", 250) // 1 pt per N250
	pointsEarned := int64(float64(event.Amount/int64(pointsRate*100)) * multiplier)

	user.TotalPoints += pointsEarned
	user.TotalRechargeAmount += event.Amount

	// Check Spin Credit Threshold
	spinThreshold := int64(cfg.GetInt("recharge_to_spin_naira", 1000) * 100)
	if user.TotalRechargeAmount >= spinThreshold {
		// Award Spin Credit
		// (Implementation would add a row to transactions with spin_credit_delta: 1)
		user.TotalRechargeAmount -= spinThreshold
	}

	// ... (Save user and transaction) ...
}
