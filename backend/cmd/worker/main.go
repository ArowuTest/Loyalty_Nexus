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
	fraudGuard := services.NewFraudGuard(db)
	hlrSvc := services.NewHLRService(nil, os.Getenv("TERMII_API_KEY")) // assuming cache repo optional

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

func processRecharge(ctx context.Context, event queue.RechargeEvent, ur repositories.UserRepository, tr repositories.TransactionRepository) {
	user, err := ur.FindByMSISDN(ctx, event.MSISDN)
	if err != nil {
		// Create Guest User
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

	pointsEarned := event.Amount / 20000 

	tx := &entities.Transaction{
		ID:          uuid.New(),
		MSISDN:      event.MSISDN,
		UserID:      user.ID,
		Type:        entities.TxTypeVisit,
		Amount:      event.Amount,
		PointsDelta: pointsEarned,
		CreatedAt:   time.Now(),
		Metadata:    map[string]any{"source": "mtn_gateway", "ref": event.Ref},
	}

	if err := tr.Save(ctx, tx); err != nil {
		log.Printf("Failed to save transaction: %v", err)
	}
}
