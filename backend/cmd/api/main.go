package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"loyalty-nexus/internal/infrastructure/queue"
)

func main() {
	// 1. Initialize Infrastructure
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})
	eq := queue.NewEventQueue(rdb, "recharge_stream")

	// 2. HTTP Handlers
	http.HandleFunc("/api/v1/recharge/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Mock Ingestor: In production, this receives pings from MTN Gateway
		msisdn := r.URL.Query().Get("msisdn")
		amount := r.URL.Query().Get("amount") // simplified for demo

		event := queue.RechargeEvent{
			MSISDN: msisdn,
			Amount: 100000, // mock N1000
			Ref:    "MTN-REF-123",
		}

		if err := eq.PushRecharge(context.Background(), event); err != nil {
			http.Error(w, "Failed to queue event", 500)
			return
		}

		fmt.Fprintf(w, "OK: Queued recharge for %s", msisdn)
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	log.Printf("Loyalty Nexus Ingestor listening on port %s", port)
	http.ListenAndServe(":"+port, nil)
}
