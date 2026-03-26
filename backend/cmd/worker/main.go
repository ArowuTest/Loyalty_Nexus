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

	// ─── Database ─────────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		log.Fatalf("[WORKER] DB connect failed: %v", err)
	}

	// ─── Redis ────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	_ = rdb

	// ─── Config ───────────────────────────────────────────────
	cfg := config.NewConfigManager(db)

	// ─── Repositories ─────────────────────────────────────────
	userRepo   := persistence.NewPostgresUserRepository(db)
	txRepo     := persistence.NewPostgresTransactionRepository(db)
	studioRepo := persistence.NewPostgresStudioRepository(db)
	prizeRepo  := persistence.NewPostgresPrizeRepository(db)
	authRepo   := persistence.NewPostgresAuthRepository(db)
	chatRepo   := persistence.NewPostgresChatRepository(db)
	warsRepo   := persistence.NewPostgresWarsRepository(db)

	// ─── Services ─────────────────────────────────────────────
	notifySvc  := services.NewNotificationService(os.Getenv("TERMII_API_KEY"))
	drawSvc    := services.NewDrawService(db)
	fulfillSvc := services.NewPrizeFulfillmentService(
		prizeRepo, userRepo,
		external.NewVTPassAdapter(),
		external.NewMTNMomoAdapter(),
		notifySvc, cfg,
	)
	winnerSvc := services.NewWinnerService(db, userRepo, prizeRepo, notifySvc)
	studioSvc := services.NewStudioService(studioRepo, userRepo, txRepo, notifySvc, nil, db)
	warsSvc   := services.NewRegionalWarsService(warsRepo, userRepo, txRepo, cfg, db)

	// Bootstrap current month's war
	if bootstrapErr := warsSvc.EnsureActiveWar(context.Background(), 50_000_000); bootstrapErr != nil {
		log.Printf("[WORKER] EnsureActiveWar: %v", bootstrapErr)
	}

	// ─── Ghost Nudge + Wallet Sync Worker (REQ-4.4) ───────────
	// Interval, warning hours, and min streak are all read from ConfigManager:
	//   ghost_nudge_interval_minutes  (default 60)
	//   ghost_nudge_warning_hours     (default 4)
	//   ghost_nudge_min_streak        (default 3)
	// Zero values are never hardcoded — all come from platform_config table.
	passportSvc      := services.NewPassportService(db)
	ghostNudgeWorker := services.NewGhostNudgeWorker(db, cfg, passportSvc, notifySvc)
	ghostNudgeWorker.Start()
	defer ghostNudgeWorker.Stop()

	// ─── Lifecycle Worker ─────────────────────────────────────
	worker := services.NewLifecycleWorker(
		db,
		userRepo, studioRepo, prizeRepo, authRepo, chatRepo,
		warsRepo,
		fulfillSvc, drawSvc, winnerSvc,
		warsSvc, studioSvc,
		notifySvc, cfg,
	)

	log.Println("[WORKER] Starting Loyalty Nexus background worker")
	worker.Run(ctx)
}
