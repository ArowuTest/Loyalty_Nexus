package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/application/usecases"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/infrastructure/persistence"
	"loyalty-nexus/internal/infrastructure/queue"
	"loyalty-nexus/internal/presentation/http/handlers"
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

	// Repositories
	userRepo := persistence.NewPostgresUserRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)
	studioRepo := persistence.NewPostgresStudioRepository(db)
	hlrRepo := persistence.NewPostgresHLRRepository(db)
	chatRepo := persistence.NewPostgresChatRepository(db)
	authRepo := persistence.NewPostgresAuthRepository(db)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)

	// Infrastructure
	// ...
	provisioner := &external.VTPassAdapter{APIKey: os.Getenv("VTPASS_KEY")}

	// Services & UseCases
	notifySvc := services.NewNotificationService(os.Getenv("TERMII_API_KEY"))
	momoPayer := &external.MTNMomoAdapter{}
	momoSvc := services.NewMoMoService(momoPayer)
	authSvc := services.NewAuthService(authRepo, userRepo, notifySvc, os.Getenv("JWT_SECRET"))
	
	fulfillSvc := services.NewPrizeFulfillmentService(prizeRepo, userRepo, provisioner, momoSvc)
	spinSvc := services.NewSpinService(userRepo, txRepo, prizeRepo, fulfillSvc, cfg, db)
	studioSvc := services.NewStudioService(studioRepo, userRepo, txRepo, notifySvc, db)

	// Knowledge / Async Engine
	notebookLM := &external.NotebookLMAdapter{APIKey: os.Getenv("NOTEBOOK_LM_KEY")}
	asyncWorker := handlers.NewAsyncStudioWorker(studioSvc, notebookLM)

	// Handlers
	studioHandler := handlers.NewStudioHandler(studioSvc, llmOrchestrator, asyncWorker, notebookLM)
	authHandler := handlers.NewAuthHandler(authSvc)
	momoHandler := handlers.NewMoMoHandler(momoSvc, authSvc)
	mnoHandler := handlers.NewMNOWebhookHandler(eq)
	adminHandler := handlers.NewAdminHandler(db)
	ussdHandler := &handlers.USSDHandler{}

	// --- ROUTES ---

	// Auth Routes (REQ-1.1, REQ-1.2)
	http.HandleFunc("/api/v1/auth/otp/send", authHandler.SendOTP)
	http.HandleFunc("/api/v1/auth/otp/verify", authHandler.VerifyOTP)

	// MoMo Routes (REQ-1.3)
	http.HandleFunc("/api/v1/user/momo/link", momoHandler.RequestLink)

	// Integrated Mode Route (REQ-2.1)
	http.HandleFunc("/api/v1/recharge/mno-webhook", mnoHandler.BSSRechargeWebhook)

	// USSD Entry Point
	http.Handle("/api/v1/ussd", ussdHandler)

	// Ingestor (MNO / Paystack Gateway Endpoint)
	http.HandleFunc("/api/v1/recharge/ingest", func(w http.ResponseWriter, r *http.Request) {
		msisdn := r.URL.Query().Get("msisdn")
		amountRaw := r.URL.Query().Get("amount")
		var amount int64
		fmt.Sscanf(amountRaw, "%d", &amount)

		event := queue.RechargeEvent{
			MSISDN: msisdn,
			Amount: amount,
			Ref:    "NEX-" + time.Now().Format("150405"),
		}
		eq.PushRecharge(r.Context(), event)
		w.WriteHeader(202)
		fmt.Fprintf(w, "Accepted")
	})

	// User Profile
	http.HandleFunc("/api/v1/user/profile", func(w http.ResponseWriter, r *http.Request) {
		msisdn := r.URL.Query().Get("msisdn") // In production, get from JWT
		user, err := userUC.GetProfile(r.Context(), msisdn)
		if err != nil {
			http.Error(w, "User not found", 404)
			return
		}
		json.NewEncoder(w).Encode(user)
	})

	// Studio Routes
	http.HandleFunc("/api/v1/studio/tools", studioHandler.ListTools)
	http.HandleFunc("/api/v1/studio/chat", studioHandler.Chat)
	http.HandleFunc("/api/v1/studio/generate/image", studioHandler.GenerateImage)
	http.HandleFunc("/api/v1/studio/generate/knowledge", studioHandler.GenerateKnowledge)
	http.HandleFunc("/api/v1/studio/generate/build", studioHandler.GenerateBuild)
	http.HandleFunc("/api/v1/studio/gallery", studioHandler.GetGallery)

	// Admin Routes
	http.HandleFunc("/api/v1/admin/config/update", adminHandler.UpdateProgramConfig)

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	log.Printf("Loyalty Nexus API listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
