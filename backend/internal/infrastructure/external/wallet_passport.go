package external

import (
	"context"
	"fmt"
	"log"
)

type WalletPassport interface {
	IssueApplePass(ctx context.Context, userID string, points int64) (string, error)
	IssueGooglePass(ctx context.Context, userID string, points int64) (string, error)
	PushUpdate(ctx context.Context, userID string, points int64) error
}

type RebitesWalletAdapter struct {
	// Inspired by the Rebites reference implementation
	IssuerID string
	APIKey   string
}

func (a *RebitesWalletAdapter) IssueApplePass(ctx context.Context, userID string, points int64) (string, error) {
	// In production: 
	// 1. Load Apple Signing Certificates (WWDR + Pass Certificate)
	// 2. Build pass.json (standard Apple format)
	// 3. Create manifest.json (SHA1 hashes of all assets)
	// 4. Create detached PKCS#7 signature
	// 5. Zip everything into .pkpass file
	// 6. Upload to S3 and return pre-signed URL
	return fmt.Sprintf("https://cdn.loyalty-nexus.ai/passes/%s.pkpass", userID), nil
}

func (a *RebitesWalletAdapter) IssueGooglePass(ctx context.Context, userID string, points int64) (string, error) {
	// In production: Generate Google Save JWT
	return "https://pay.google.com/gp/v/save/mock-jwt", nil
}

func (a *RebitesWalletAdapter) BuildPassContent(userID string, points int64, streak int, currentDataMB int) map[string]interface{} {
	// Innovation: "Projected Value" (Strategy Doc Section 3)
	message := fmt.Sprintf("You are missing out on bonus points today. Your %d-day streak is at risk!", streak)
	if streak >= 5 {
		message = "Maintain your streak to save your N50,000 Jackpot entry."
	}

	return map[string]interface{}{
		"points": points,
		"streak": streak,
		"nudge_message": message,
		"data_balance": fmt.Sprintf("%dMB", currentDataMB),
	}
}
