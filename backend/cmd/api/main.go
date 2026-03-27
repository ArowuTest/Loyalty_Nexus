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
	warsRepo   := persistence.NewPostgresWarsRepository(db)

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
	warssSvc   := services.NewRegionalWarsService(warsRepo, userRepo, txRepo, cfg, db)
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
	llmOrch := external.NewLLMOrchestrator(groq, gemini, deepseek, usageTracker, chatRepo, rdb, groqLimit, geminiLimit)

	// ─── Asset Storage (S3 / GCS / local — driven by STORAGE_BACKEND env var) ──
	assetStorage := external.NewAssetStorageFromEnv()

	// ─── AI Studio Orchestrator (4-tier provider) ──────────────
	aiStudioOrch := services.NewAIStudioOrchestrator(cfg, studioRepo, studioSvc, userRepo, assetStorage)
	aiStudioOrch.SetLLMOrch(llmOrch) // wire health tracking

	// ─── AI Provider Registry (DB-backed dynamic dispatch) ────────────
	aiProviderRepo := persistence.NewAIProviderRepository(db)
	aiStudioOrch.SetProviderDB(aiProviderRepo) // enables DB-driven fallback chains
	aiProviderH := handlers.NewAIProviderAdminHandler(aiProviderRepo)

	// ─── Knowledge Worker (dispatches studio jobs) ─────────────
	kbWorker := handlers.NewAsyncStudioWorker(studioSvc, aiStudioOrch)

	// ─── Session Summariser Worker (compresses idle chat sessions → memory) ──
	// Runs every 30 minutes; stores compressed summaries in session_summaries table.
	// The LLMOrchestrator.Chat() reads these back on next message to reconstruct context.
	summariserWorker := services.NewSummariserWorker(db, llmOrch)
	go summariserWorker.Run(ctx)

	// ─── HTTP Handlers ────────────────────────────────────────
	authH    := handlers.NewAuthHandler(authSvc)
	rechargeH := handlers.NewRechargeHandler(rechargeSvc, eq)
	spinH    := handlers.NewSpinHandler(spinSvc)
	studioH  := handlers.NewStudioHandler(studioSvc, llmOrch, kbWorker, cfg)
	studioH.SetAssetStorage(assetStorage) // enables /studio/upload endpoint
	// kbWorker no longer needs a back-link — orch is injected directly
	userH    := handlers.NewUserHandler(userRepo, hlrSvc, momoSvc, fulfillSvc)
	adminH   := handlers.NewAdminHandler(db, cfg, spinSvc, drawSvc, fraudSvc, warssSvc, studioSvc, rdb)
	ussdH    := handlers.NewUSSDHandler(spinSvc, rechargeSvc, userRepo, cfg)
	// ─── WebSocket Hub (Regional Wars real-time leaderboard) ────
	leaderboardHub := handlers.NewLeaderboardHub()
	handlers.StartLeaderboardPoller(ctx, leaderboardHub, warssSvc, 30*time.Second)

	warsH    := handlers.NewWarsHandler(warssSvc, leaderboardHub)
	drawH    := handlers.NewDrawHandler(drawSvc)
	passportH := handlers.NewPassportHandler(passportSvc)
	fraudH   := handlers.NewFraudHandler(fraudSvc)

	// ─── Router ───────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "1.0.0"}); err != nil {
			log.Printf("[health] encode error: %v", err)
		}
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


	// ── Nexus Studio ────────────────────────────────────────────────────────
	mux.Handle("GET  /api/v1/studio/tools",                  auth(http.HandlerFunc(studioH.ListTools)))
	mux.Handle("GET  /api/v1/studio/tools/{slug}",           auth(http.HandlerFunc(studioH.GetTool)))
	mux.Handle("POST /api/v1/studio/generate",               auth(http.HandlerFunc(studioH.Generate)))
	mux.Handle("GET  /api/v1/studio/generate/{id}",          auth(http.HandlerFunc(studioH.GetGenerationStatus)))
	mux.Handle("GET  /api/v1/studio/gallery",                auth(http.HandlerFunc(studioH.GetGallery)))
	mux.Handle("POST /api/v1/studio/generate/{id}/dispute",  auth(http.HandlerFunc(studioH.DisputeGeneration)))
	mux.Handle("GET  /api/v1/studio/session",                auth(http.HandlerFunc(studioH.GetSessionUsage)))
	mux.Handle("POST /api/v1/studio/upload",                 auth(http.HandlerFunc(studioH.UploadAsset)))
	// ── Nexus Chat ───────────────────────────────────────────────────────────
	mux.Handle("POST /api/v1/studio/chat",                   auth(http.HandlerFunc(studioH.Chat)))
	mux.Handle("GET  /api/v1/studio/chat/usage",             auth(http.HandlerFunc(studioH.GetChatUsage)))

	// Wars routes
	mux.Handle("GET  /api/v1/wars/leaderboard",         auth(http.HandlerFunc(warsH.GetLeaderboard)))
	mux.Handle("GET  /api/v1/wars/my-rank",             auth(http.HandlerFunc(warsH.GetMyRank)))
	mux.Handle("GET  /api/v1/wars/history",             auth(http.HandlerFunc(warsH.GetHistory)))
	mux.Handle("GET  /api/v1/wars/{period}/winners",    auth(http.HandlerFunc(warsH.GetWinners)))
	// WebSocket: real-time leaderboard (spec §3.5 Phase 3)
	mux.Handle("GET  /api/v1/wars/live",                auth(http.HandlerFunc(warsH.LiveLeaderboard)))

	// Passport routes (spec §6)
	mux.Handle("GET  /api/v1/passport",              auth(http.HandlerFunc(passportH.GetPassport)))
	mux.Handle("GET  /api/v1/passport/badges",       auth(http.HandlerFunc(passportH.GetBadges)))
	mux.Handle("GET  /api/v1/passport/qr",           auth(http.HandlerFunc(passportH.GetQR)))
	mux.Handle("POST /api/v1/passport/qr/verify",    auth(http.HandlerFunc(passportH.VerifyQR)))
	mux.Handle("GET  /api/v1/passport/pkpass",       auth(http.HandlerFunc(passportH.DownloadPKPass)))
	mux.Handle("GET  /api/v1/passport/events",       auth(http.HandlerFunc(passportH.GetEvents)))
	mux.Handle("GET  /api/v1/passport/share",        auth(http.HandlerFunc(passportH.GetShareCard)))

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
	mux.Handle("POST   /api/v1/admin/studio-tools",              adminAuth(http.HandlerFunc(adminH.CreateStudioTool)))
	mux.Handle("DELETE /api/v1/admin/studio-tools/{id}",         adminAuth(http.HandlerFunc(adminH.DisableStudioTool)))
	mux.Handle("GET    /api/v1/admin/studio-tools/stats",        adminAuth(http.HandlerFunc(adminH.GetStudioToolStats)))
	mux.Handle("GET    /api/v1/admin/studio-tools/{id}/errors",  adminAuth(http.HandlerFunc(adminH.GetStudioToolErrors)))
	mux.Handle("GET    /api/v1/admin/studio-generations",        adminAuth(http.HandlerFunc(adminH.GetStudioGenerations)))
	mux.Handle("GET    /api/v1/admin/users",              adminAuth(http.HandlerFunc(adminH.ListUsers)))
	mux.Handle("GET    /api/v1/admin/regional-wars",      adminAuth(http.HandlerFunc(adminH.GetRegionalWars)))
	mux.Handle("POST /api/v1/admin/wars/resolve",          adminAuth(http.HandlerFunc(warsH.AdminResolve)))
	mux.Handle("PUT  /api/v1/admin/wars/prize-pool",       adminAuth(http.HandlerFunc(warsH.AdminUpdatePrizePool)))
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
	// Studio tool enable/reprice — handled by PUT /api/v1/admin/studio-tools/{id} above
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
	// AI Provider management (dynamic provider registry)
	mux.Handle("GET    /api/v1/admin/ai-providers",                    adminAuth(http.HandlerFunc(aiProviderH.ListProviders)))
	mux.Handle("GET    /api/v1/admin/ai-providers/meta",               adminAuth(http.HandlerFunc(aiProviderH.GetProviderMeta)))
	mux.Handle("POST   /api/v1/admin/ai-providers",                    adminAuth(http.HandlerFunc(aiProviderH.CreateProvider)))
	mux.Handle("PUT    /api/v1/admin/ai-providers/{id}",               adminAuth(http.HandlerFunc(aiProviderH.UpdateProvider)))
	mux.Handle("DELETE /api/v1/admin/ai-providers/{id}",               adminAuth(http.HandlerFunc(aiProviderH.DeleteProvider)))
	mux.Handle("POST   /api/v1/admin/ai-providers/{id}/activate",      adminAuth(http.HandlerFunc(aiProviderH.ActivateProvider)))
	mux.Handle("POST   /api/v1/admin/ai-providers/{id}/deactivate",    adminAuth(http.HandlerFunc(aiProviderH.DeactivateProvider)))
	mux.Handle("POST   /api/v1/admin/ai-providers/{id}/test",          adminAuth(http.HandlerFunc(aiProviderH.TestProvider)))

	// System health (REQ-5.8.3)
	mux.Handle("GET    /api/v1/admin/health",                 adminAuth(http.HandlerFunc(adminH.GetHealth)))
	mux.Handle("GET    /api/v1/admin/ai-health",              adminAuth(http.HandlerFunc(adminH.GetAIHealth)))

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

	// ─── Recharge-event → leaderboard broadcast ─────────────────────────
	// Subscribe to the recharge stream; on every successful recharge re-fetch
	// the leaderboard and push to all connected WebSocket clients (≤1s latency).
	go eq.Subscribe(ctx, "wars-ws-group", "api-server", func(event map[string]interface{}) error {
		if leaderboardHub.ConnectedClients() == 0 {
			return nil // no one watching — skip expensive query
		}
		entries, err := warssSvc.GetLeaderboard(ctx, 37)
		if err != nil {
			return nil // non-fatal; poller will catch it in ≤30s
		}
		leaderboardHub.BroadcastLeaderboard(entries, currentWarPeriodStr(), "update")
		return nil
	})

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

// currentWarPeriodStr returns "YYYY-MM" for the current UTC month.
func currentWarPeriodStr() string {
	return time.Now().UTC().Format("2006-01")
}
