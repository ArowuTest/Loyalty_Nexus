package external

import (
	"context"
	"fmt"
	"log"
)

type WalletPassport interface {
	IssueApplePass(ctx context.Context, userID string, points int64) (string, error)
	IssueGooglePass(ctx context.Context, userID string, points int64) (string, error)
	PushUpdate(ctx context.Context, userID string, points int64, streak int, currentDataMB int) error
}

type RebitesWalletAdapter struct {
	IssuerID string
	APIKey   string
}

func (a *RebitesWalletAdapter) IssueApplePass(ctx context.Context, userID string, points int64) (string, error) {
	// In production: Generate .pkpass using signing certificates
	return fmt.Sprintf("https://cdn.loyalty-nexus.ai/passes/%s.pkpass", userID), nil
}

func (a *RebitesWalletAdapter) IssueGooglePass(ctx context.Context, userID string, points int64) (string, error) {
	// In production: Generate Google Save JWT
	return "https://pay.google.com/gp/v/save/mock-jwt", nil
}

func (a *RebitesWalletAdapter) PushUpdate(ctx context.Context, userID string, points int64, streak int, currentDataMB int) error {
	// Innovation: "Projected Value" (Strategic Audit Section 3)
	// Showing what user is missing out on to prevent churn.
	nudge := fmt.Sprintf("You are missing out on 200MB today. Your %d-day streak is at risk.", streak)
	if streak >= 5 {
		nudge = "Toggle MTN back on to save your N50,000 Jackpot entry."
	}

	log.Printf("[WalletPush] Updating User %s | Points: %d | Nudge: %s", userID, points, nudge)
	// In production: send push via APNS/Google with this dynamic metadata
	return nil
}
