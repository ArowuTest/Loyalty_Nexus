package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/application/usecases"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/persistence"
	"loyalty-nexus/internal/infrastructure/queue"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/presentation/http/handlers"
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

	// Repositories
	userRepo := persistence.NewPostgresUserRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)
	studioRepo := persistence.NewPostgresStudioRepository(db)

	// Infrastructure
	eq := queue.NewEventQueue(rdb, "recharge_stream")
	cfg := config.NewConfigManager(db)
	cfg.Refresh(context.Background())

	// External AI Clients (Proxies/Adapters)
	// In production, these are initialized with real API keys from env
	groq := &external.GroqAdapter{}
	gemini := &external.GeminiAdapter{}
	deepseek := &external.DeepSeekAdapter{}
	usageTracker := external.NewRedisUsageTracker(rdb)

	// AI Orchestrator
	llmOrchestrator := external.NewLLMOrchestrator(groq, gemini, deepseek, usageTracker, 10, 20)

	// Services & UseCases
	notifySvc := services.NewNotificationService(os.Getenv("TERMII_API_KEY"))
	userUC := usecases.NewUserUseCase(userRepo)
	spinSvc := services.NewSpinService(userRepo, txRepo, cfg, db)
	studioSvc := services.NewStudioService(studioRepo, userRepo, txRepo, notifySvc, db)

	// Knowledge / Async Engine
	notebookLM := &external.NotebookLMAdapter{APIKey: os.Getenv("NOTEBOOK_LM_KEY")}
	asyncWorker := handlers.NewAsyncStudioWorker(studioSvc, notebookLM)

	// Handlers
	studioHandler := handlers.NewStudioHandler(studioSvc, llmOrchestrator, asyncWorker, notebookLM)
	adminHandler := &handlers.AdminHandler{} // To be fully initialized
	ussdHandler := &handlers.USSDHandler{}

	// --- ROUTES ---

	// USSD Entry Point
	http.Handle("/api/v1/ussd", ussdHandler)

	// Admin Cockpit
	http.HandleFunc("/api/v1/admin/config/update", adminHandler.UpdateProgramConfig)
	http.HandleFunc("/api/v1/admin/prizes", adminHandler.ListPrizes)
	http.HandleFunc("/api/v1/admin/prizes/update", adminHandler.UpdatePrizeWeight)

	// Ingestor (MNO / Paystack Gateway Endpoint)
	http.HandleFunc("/api/v1/recharge/ingest", func(w http.ResponseWriter, r *http.Request) {
		mode := os.Getenv("OPERATION_MODE")
		msisdn := r.URL.Query().Get("msisdn")
		amountRaw := r.URL.Query().Get("amount")
		
		var amount int64
		fmt.Sscanf(amountRaw, "%d", &amount)

		// Verification logic based on mode
		if mode == "integrated" {
			// Validate MNO signature/header
		} else {
			// Independent Mode: Validate Paystack origin or ref
		}

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

	// Spin Wheel
	http.HandleFunc("/api/v1/spin/play", func(w http.ResponseWriter, r *http.Request) {
		msisdn := r.URL.Query().Get("msisdn")
		tx, err := spinSvc.PlaySpin(r.Context(), msisdn)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		json.NewEncoder(w).Encode(tx)
	})

	// Studio Routes
	http.HandleFunc("/api/v1/studio/tools", studioHandler.ListTools)
	http.HandleFunc("/api/v1/studio/chat", studioHandler.Chat)
	http.HandleFunc("/api/v1/studio/generate/image", studioHandler.GenerateImage)
	http.HandleFunc("/api/v1/studio/generate/knowledge", studioHandler.GenerateKnowledge)
	http.HandleFunc("/api/v1/studio/generate/build", studioHandler.GenerateBuild)
	http.HandleFunc("/api/v1/studio/gallery", studioHandler.GetGallery)

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	log.Printf("Loyalty Nexus API listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
