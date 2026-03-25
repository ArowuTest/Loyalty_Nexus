package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/infrastructure/persistence"
	"loyalty-nexus/internal/infrastructure/queue"
	"loyalty-nexus/internal/presentation/http/handlers"
	"loyalty-nexus/internal/presentation/http/middleware"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ─── Database ─────────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		log.Fatalf("DB connect: %v", err)
	}

	// ─── Redis ────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	// ─── Config Manager (reads all rules from network_configs) ──
	cfg := config.NewConfigManager(db)

	// ─── Repositories ─────────────────────────────────────────
	userRepo   := persistence.NewPostgresUserRepository(db)
	txRepo     := persistence.NewPostgresTransactionRepository(db)
	studioRepo := persistence.NewPostgresStudioRepository(db)
	hlrRepo    := persistence.NewPostgresHLRRepository(db)
	chatRepo   := persistence.NewPostgresChatRepository(db)
	authRepo   := persistence.NewPostgresAuthRepository(db)
	prizeRepo  := persistence.NewPostgresPrizeRepository(db)

	// ─── External Adapters ────────────────────────────────────
	vtpass    := external.NewVTPassAdapter()
	momoSvc   := external.NewMTNMomoAdapter()
	usageTracker := external.NewRedisUsageTracker(rdb)

	// ─── AI Clients ───────────────────────────────────────────
	groq     := external.NewGroqAdapter(os.Getenv("GROQ_API_KEY"))
	gemini   := external.NewGeminiAdapter(os.Getenv("GEMINI_API_KEY"))
	deepseek := external.NewDeepSeekAdapter(os.Getenv("DEEPSEEK_API_KEY"))

	// ─── NATS Event Queue ─────────────────────────────────────
	eq := queue.NewEventQueue(rdb, "recharge_stream")

	// ─── Services ─────────────────────────────────────────────
	notifySvc  := services.NewNotificationService(os.Getenv("TERMII_API_KEY"))
	authSvc    := services.NewAuthService(authRepo, userRepo, notifySvc, cfg)
	notifyH    := handlers.NewNotificationHandler(db)

	fulfillSvc := services.NewPrizeFulfillmentService(prizeRepo, userRepo, vtpass, momoSvc, notifySvc, cfg)
	rechargeSvc := services.NewRechargeService(userRepo, txRepo, notifySvc, cfg, db)
	spinSvc    := services.NewSpinService(userRepo, txRepo, prizeRepo, fulfillSvc, notifySvc, cfg, db)
	studioSvc  := services.NewStudioService(studioRepo, userRepo, txRepo, notifySvc, nil, db)
	hlrSvc     := services.NewHLRService(hlrRepo)
	warssSvc   := services.NewRegionalWarsService(db)
	drawSvc    := services.NewDrawService(db)
	passportSvc := services.NewPassportService(db)
	fraudSvc   := services.NewFraudService(db)

	// Bootstrap current month's war if none exists
	if err := warssSvc.EnsureActiveWar(context.Background(), 50_000_000); err != nil {
		log.Printf("[main] EnsureActiveWar: %v", err)
	}

	// ─── LLM Orchestrator (Groq → Gemini → DeepSeek) ─────────
	groqLimit   := cfg.GetInt("chat_groq_daily_limit", 1000)
	geminiLimit := cfg.GetInt("chat_gemini_daily_limit", 2000)
	llmOrch := external.NewLLMOrchestrator(groq, gemini, deepseek, usageTracker, chatRepo, groqLimit, geminiLimit)

	// ─── Knowledge Worker (NotebookLM) ────────────────────────
	kbWorker := handlers.NewAsyncStudioWorker(studioSvc, nil)

	// ─── HTTP Handlers ────────────────────────────────────────
	authH    := handlers.NewAuthHandler(authSvc)
	rechargeH := handlers.NewRechargeHandler(rechargeSvc, eq)
	spinH    := handlers.NewSpinHandler(spinSvc)
	studioH  := handlers.NewStudioHandler(studioSvc, llmOrch, kbWorker, cfg)
	kbWorker.LinkHandler(studioH) // bidirectional link for background dispatch
	userH    := handlers.NewUserHandler(userRepo, hlrSvc, momoSvc, fulfillSvc)
	adminH   := handlers.NewAdminHandler(db, cfg, spinSvc, drawSvc, fraudSvc)
	ussdH    := handlers.NewUSSDHandler(spinSvc, rechargeSvc, userRepo, cfg)
	warsH    := handlers.NewWarsHandler(warssSvc)
	drawH    := handlers.NewDrawHandler(drawSvc)
	passportH := handlers.NewPassportHandler(passportSvc)
	fraudH   := handlers.NewFraudHandler(fraudSvc)

	// ─── Router ───────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "1.0.0"})
	})

	// Auth (public)
	mux.HandleFunc("POST /api/v1/auth/otp/send",   authH.SendOTP)
	mux.HandleFunc("POST /api/v1/auth/otp/verify", authH.VerifyOTP)

	// Webhooks (public — signature-verified internally)
	mux.HandleFunc("POST /api/v1/recharge/paystack-webhook", rechargeH.PaystackWebhook)
	mux.HandleFunc("POST /api/v1/recharge/mno-webhook",      rechargeH.MNOWebhook)

	// USSD (public — HMAC-verified)
	mux.HandleFunc("POST /api/v1/ussd", ussdH.Handle)

	// Protected routes
	auth := middleware.AuthMiddleware(authSvc)

	mux.Handle("GET  /api/v1/user/profile",       auth(http.HandlerFunc(userH.GetProfile)))
	mux.Handle("GET  /api/v1/user/wallet",        auth(http.HandlerFunc(userH.GetWallet)))
	mux.Handle("POST /api/v1/user/momo/request",  auth(http.HandlerFunc(userH.RequestMoMoLink)))
	mux.Handle("POST /api/v1/user/profile/state",  auth(http.HandlerFunc(userH.UpdateProfileState)))
	mux.Handle("POST /api/v1/user/momo/verify",   auth(http.HandlerFunc(userH.VerifyMoMo)))
	mux.Handle("GET  /api/v1/user/transactions",  auth(http.HandlerFunc(userH.GetTransactions)))
	mux.Handle("GET  /api/v1/user/passport",      auth(http.HandlerFunc(userH.GetPassportURLs)))

	mux.Handle("GET  /api/v1/spin/wheel",         auth(http.HandlerFunc(spinH.GetWheelConfig)))
	mux.Handle("POST /api/v1/spin/play",          auth(http.HandlerFunc(spinH.Play)))
	mux.Handle("GET  /api/v1/spin/history",       auth(http.HandlerFunc(spinH.GetHistory)))

	// ─── Notifications ────────────────────────────────────────────────────────
	mux.Handle("GET /api/v1/notifications",                auth(http.HandlerFunc(notifyH.ListNotifications)))
	mux.Handle("PATCH /api/v1/notifications/{id}/read",    auth(http.HandlerFunc(notifyH.MarkRead)))
	mux.Handle("POST /api/v1/notifications/read-all",      auth(http.HandlerFunc(notifyH.MarkAllRead)))
	mux.Handle("POST /api/v1/notifications/push-token",    auth(http.HandlerFunc(notifyH.RegisterPushToken)))
	mux.Handle("GET /api/v1/notifications/preferences",    auth(http.HandlerFunc(notifyH.GetPreferences)))
	mux.Handle("PATCH /api/v1/notifications/preferences",  auth(http.HandlerFunc(notifyH.UpdatePreferences)))


	mux.Handle("GET  /api/v1/studio/tools",              auth(http.HandlerFunc(studioH.ListTools)))
	mux.Handle("POST /api/v1/studio/chat",               auth(http.HandlerFunc(studioH.Chat)))
	mux.Handle("POST /api/v1/studio/generate",           auth(http.HandlerFunc(studioH.Generate)))
	mux.Handle("GET  /api/v1/studio/generate/{id}/status", auth(http.HandlerFunc(studioH.GetGenerationStatus)))
	mux.Handle("GET  /api/v1/studio/gallery",            auth(http.HandlerFunc(studioH.GetGallery)))

	// Wars routes
	mux.Handle("GET  /api/v1/wars/leaderboard",  auth(http.HandlerFunc(warsH.GetLeaderboard)))
	mux.Handle("GET  /api/v1/wars/my-rank",      auth(http.HandlerFunc(warsH.GetMyRank)))

	// Passport routes
	mux.Handle("GET  /api/v1/passport",          auth(http.HandlerFunc(passportH.GetPassport)))
	mux.Handle("GET  /api/v1/passport/badges",   auth(http.HandlerFunc(passportH.GetBadges)))

	// Draws (public results)
	mux.Handle("GET  /api/v1/draws",             auth(http.HandlerFunc(drawH.ListUpcoming)))
	mux.Handle("GET  /api/v1/draws/{id}/winners", auth(http.HandlerFunc(drawH.GetWinners)))

	// Admin routes (admin JWT required)
	adminAuth := middleware.AdminAuthMiddleware(authSvc)
	mux.Handle("GET    /api/v1/admin/dashboard",          adminAuth(http.HandlerFunc(adminH.GetDashboard)))
	mux.Handle("GET    /api/v1/admin/config",             adminAuth(http.HandlerFunc(adminH.GetConfig)))
	mux.Handle("PUT    /api/v1/admin/config/{key}",       adminAuth(http.HandlerFunc(adminH.UpdateConfig)))
	mux.Handle("GET    /api/v1/admin/prize-pool",         adminAuth(http.HandlerFunc(adminH.GetPrizePool)))
	mux.Handle("PUT    /api/v1/admin/prize-pool/{id}",    adminAuth(http.HandlerFunc(adminH.UpdatePrize)))
	mux.Handle("GET    /api/v1/admin/studio-tools",       adminAuth(http.HandlerFunc(adminH.GetStudioTools)))
	mux.Handle("PUT    /api/v1/admin/studio-tools/{id}",  adminAuth(http.HandlerFunc(adminH.UpdateStudioTool)))
	mux.Handle("GET    /api/v1/admin/users",              adminAuth(http.HandlerFunc(adminH.ListUsers)))
	mux.Handle("GET    /api/v1/admin/regional-wars",      adminAuth(http.HandlerFunc(adminH.GetRegionalWars)))
	mux.Handle("POST   /api/v1/admin/wars/resolve",        adminAuth(http.HandlerFunc(warsH.AdminResolve)))
	mux.Handle("GET    /api/v1/admin/fraud-events",        adminAuth(http.HandlerFunc(fraudH.ListEvents)))
	mux.Handle("PUT    /api/v1/admin/fraud-events/{id}/resolve", adminAuth(http.HandlerFunc(fraudH.ResolveEvent)))
	mux.Handle("PUT    /api/v1/admin/users/{id}/suspend",  adminAuth(http.HandlerFunc(fraudH.SuspendUser)))
	// Draws admin — full CRUD + execute + CSV export
	mux.Handle("GET    /api/v1/admin/draws",                  adminAuth(http.HandlerFunc(adminH.GetDraws)))
	mux.Handle("POST   /api/v1/admin/draws",                  adminAuth(http.HandlerFunc(adminH.CreateDraw)))
	mux.Handle("PUT    /api/v1/admin/draws/{id}",             adminAuth(http.HandlerFunc(adminH.UpdateDraw)))
	mux.Handle("POST   /api/v1/admin/draws/{id}/execute",     adminAuth(http.HandlerFunc(adminH.ExecuteDraw)))
	mux.Handle("GET    /api/v1/admin/draws/{id}/winners",     adminAuth(http.HandlerFunc(adminH.GetDrawWinners)))
	mux.Handle("GET    /api/v1/admin/draws/{id}/export",      adminAuth(http.HandlerFunc(adminH.ExportDrawEntries)))
	// Prize / Spin Wheel CRUD
	mux.Handle("GET    /api/v1/admin/prizes",                 adminAuth(http.HandlerFunc(adminH.GetPrizePool)))
	mux.Handle("POST   /api/v1/admin/prizes",                 adminAuth(http.HandlerFunc(adminH.CreatePrize)))
	mux.Handle("PUT    /api/v1/admin/prizes/{id}",            adminAuth(http.HandlerFunc(adminH.UpdatePrize)))
	mux.Handle("DELETE /api/v1/admin/prizes/{id}",            adminAuth(http.HandlerFunc(adminH.DeletePrize)))
	// Spin configuration
	mux.Handle("GET    /api/v1/admin/spin/config",            adminAuth(http.HandlerFunc(adminH.GetSpinConfig)))
	mux.Handle("PUT    /api/v1/admin/spin/config",            adminAuth(http.HandlerFunc(adminH.UpdateSpinConfig)))
	// Points ledger audit
	mux.Handle("GET    /api/v1/admin/points/stats",           adminAuth(http.HandlerFunc(adminH.GetPointsStats)))
	mux.Handle("GET    /api/v1/admin/points/history",         adminAuth(http.HandlerFunc(adminH.GetPointsHistory)))
	mux.Handle("POST   /api/v1/admin/points/adjust",          adminAuth(http.HandlerFunc(adminH.AdjustPoints)))
	// User management
	mux.Handle("GET    /api/v1/admin/users/{id}",             adminAuth(http.HandlerFunc(adminH.GetUser)))
	mux.Handle("PUT    /api/v1/admin/users/{id}/suspend",     adminAuth(http.HandlerFunc(adminH.SuspendUser)))
	// Studio tool enable/reprice
	mux.Handle("PUT    /api/v1/admin/studio-tools/{key}",     adminAuth(http.HandlerFunc(adminH.UpdateStudioTool)))
	// Notification broadcast
	mux.Handle("POST   /api/v1/admin/notifications/broadcast", adminAuth(http.HandlerFunc(adminH.BroadcastNotification)))
	mux.Handle("GET    /api/v1/admin/notifications/broadcasts",adminAuth(http.HandlerFunc(adminH.GetBroadcastHistory)))
	// Subscription management
	mux.Handle("GET    /api/v1/admin/subscriptions",          adminAuth(http.HandlerFunc(adminH.GetSubscriptions)))
	mux.Handle("PUT    /api/v1/admin/subscriptions/{id}",     adminAuth(http.HandlerFunc(adminH.UpdateSubscription)))
	// Fraud management
	mux.Handle("POST   /api/v1/admin/fraud/{id}/resolve",     adminAuth(http.HandlerFunc(adminH.ResolveFraudEvent)))
	// Regional Wars control
	mux.Handle("POST   /api/v1/admin/wars/cycle/reset",       adminAuth(http.HandlerFunc(adminH.ResetWarsCycle)))
	// System health (REQ-5.8.3)
	mux.Handle("GET    /api/v1/admin/health",                 adminAuth(http.HandlerFunc(adminH.GetHealth)))

	// ─── HTTP Server ──────────────────────────────────────────
	port := cfg.GetString("port", "8080")
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      middleware.CORS(middleware.RequestLogger(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("[API] Loyalty Nexus API starting on :%s (mode: %s)", port, cfg.GetString("operation_mode", "independent"))

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[API] Server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("[API] Shutting down gracefully...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
	log.Println("[API] Shutdown complete")
}
