package main

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
)

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})

	userRepo := persistence.NewPostgresUserRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)

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
