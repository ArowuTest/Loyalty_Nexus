package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/infrastructure/persistence"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ─── Database ───────────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		log.Fatalf("[WORKER] DB connect failed: %v", err)
	}

	// ─── Redis ──────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	// ─── Config ─────────────────────────────────────────────────
	cfg := config.NewConfigManager(db)

	// ─── Repositories ───────────────────────────────────────────
	userRepo  := persistence.NewPostgresUserRepository(db)
	studioRepo := persistence.NewPostgresStudioRepository(db)
	prizeRepo  := persistence.NewPostgresPrizeRepository(db)
	authRepo   := persistence.NewPostgresAuthRepository(db)
	chatRepo   := persistence.NewPostgresChatRepository(db)
	_ = rdb // Used by usage tracker

	// ─── Services ───────────────────────────────────────────────
	notifySvc := services.NewNotificationService(os.Getenv("TERMII_API_KEY"))
	drawSvc   := services.NewDrawService(db)
	fulfillSvc := services.NewPrizeFulfillmentService(
		prizeRepo, userRepo,
		external.NewVTPassAdapter(),
		external.NewMTNMomoAdapter(),
		notifySvc, cfg,
	)

	worker := services.NewLifecycleWorker(
		db, userRepo, studioRepo, prizeRepo, authRepo, chatRepo,
		fulfillSvc, drawSvc, notifySvc, cfg,
	)

	log.Println("[WORKER] Starting Loyalty Nexus background worker")
	worker.Run(ctx)
}
