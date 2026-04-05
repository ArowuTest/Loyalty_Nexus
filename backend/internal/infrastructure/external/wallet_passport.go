package external

// wallet_passport.go — Apple Wallet pass generation + unified WalletPassportAdapter.
// Google Wallet JWT generation lives in google_wallet.go (GoogleWalletAdapter).
// This file owns: ApplePassAdapter, WalletPassportAdapter, RebitesWalletAdapter.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// ─── ApplePassAdapter ─────────────────────────────────────────────────────────

// ApplePassAdapter generates Apple Wallet .pkpass descriptors.
// In development (no certificate configured) it returns a base64-encoded JSON
// data URI representing the pass structure.
type ApplePassAdapter struct {
	CertPath     string // APPLE_PASS_CERTIFICATE_PATH
	CertPassword string // APPLE_PASS_CERTIFICATE_PASSWORD
	TeamID       string // APPLE_TEAM_ID
	S3Uploader   S3Uploader
	client       *http.Client
}

// NewApplePassAdapter reads credentials from environment variables.
func NewApplePassAdapter(uploader S3Uploader) *ApplePassAdapter {
	return &ApplePassAdapter{
		CertPath:     os.Getenv("APPLE_PASS_CERTIFICATE_PATH"),
		CertPassword: os.Getenv("APPLE_PASS_CERTIFICATE_PASSWORD"),
		TeamID:       os.Getenv("APPLE_TEAM_ID"),
		S3Uploader:   uploader,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// IssueApplePass builds a Loyalty Card pass descriptor for the user.
// When no signing certificate is configured it returns a base64 data URI
// (development mode). In production the pass JSON is uploaded to S3.
func (a *ApplePassAdapter) IssueApplePass(ctx context.Context, userID string, points int64) (string, error) {
	teamID := a.TeamID
	if teamID == "" {
		teamID = "NEXUSTEAM01"
	}

	passObj := map[string]interface{}{
		"formatVersion":      1,
		"passTypeIdentifier": "pass.ai.nexus.loyalty",
		"serialNumber":       userID,
		"teamIdentifier":     teamID,
		"organizationName":   "Loyalty Nexus",
		"description":        "Loyalty Nexus Digital Passport",
		"logoText":           "Nexus",
		"backgroundColor":    "rgb(255, 200, 0)",
		"storeCard": map[string]interface{}{
			"primaryFields": []map[string]interface{}{
				{"key": "points", "label": "PULSE POINTS", "value": points},
			},
			"secondaryFields": []map[string]interface{}{
				{"key": "tier", "label": "TIER", "value": "Member"},
			},
			"headerFields": []map[string]interface{}{
				{"key": "name", "label": "LOYALTY NEXUS", "value": "Digital Passport"},
			},
		},
	}

	passJSON, err := json.Marshal(passObj)
	if err != nil {
		return "", fmt.Errorf("apple pass: marshal: %w", err)
	}

	// Development mode — no signing certificate
	if a.CertPath == "" {
		encoded := base64.StdEncoding.EncodeToString(passJSON)
		return "data:application/vnd.apple.pkpass;base64," + encoded, nil
	}

	// Production: upload pass descriptor to S3
	if a.S3Uploader != nil {
		key := fmt.Sprintf("passes/%s.pkpass.json", userID)
		cdnURL, uploadErr := a.S3Uploader.Upload(ctx, key, passJSON, "application/json")
		if uploadErr == nil {
			return cdnURL, nil
		}
	}

	// Fallback: return data URI
	encoded := base64.StdEncoding.EncodeToString(passJSON)
	return "data:application/vnd.apple.pkpass;base64," + encoded, nil
}

// ─── WalletPassportAdapter (unified) ─────────────────────────────────────────

// WalletPassportAdapter unifies Apple and Google pass issuance and implements
// the WalletPassport interface. Google Wallet is handled by GoogleWalletAdapter
// (defined in google_wallet.go) which uses the full loyalty object JWT approach.
type WalletPassportAdapter struct {
	apple  *ApplePassAdapter
	google *GoogleWalletAdapter
}

// NewWalletPassportAdapter constructs the unified adapter.
// GoogleWalletAdapter is initialised via NewGoogleWalletAdapter (google_wallet.go).
// If Google Wallet env vars are not set, the adapter gracefully degrades.
func NewWalletPassportAdapter(uploader S3Uploader) *WalletPassportAdapter {
	gwa, err := NewGoogleWalletAdapter()
	if err != nil {
		log.Printf("[WalletPassport] Google Wallet not configured (%v) — degraded mode", err)
		gwa = nil
	}
	return &WalletPassportAdapter{
		apple:  NewApplePassAdapter(uploader),
		google: gwa,
	}
}

// IssueApplePass delegates to ApplePassAdapter.
func (w *WalletPassportAdapter) IssueApplePass(ctx context.Context, userID string, points int64) (string, error) {
	return w.apple.IssueApplePass(ctx, userID, points)
}

// IssueGooglePass delegates to GoogleWalletAdapter.BuildSaveURL.
// Falls back to a placeholder URL if Google Wallet is not configured.
func (w *WalletPassportAdapter) IssueGooglePass(ctx context.Context, userID string, points int64) (string, error) {
	if w.google == nil || !w.google.IsConfigured() {
		return fmt.Sprintf("https://pay.google.com/gp/v/save/%s", userID), nil
	}
	// Parse the userID string to uuid.UUID
	uid, parseErr := uuid.Parse(userID)
	if parseErr != nil {
		return fmt.Sprintf("https://pay.google.com/gp/v/save/%s", userID), nil
	}
	saveURL, _, err := w.google.BuildSaveURL(GoogleWalletPassInput{
		UserID:         uid,
		LifetimePoints: points,
		Tier:           "member",
		StreakCount:    0,
	})
	if err != nil {
		log.Printf("[WalletPassport] IssueGooglePass error: %v", err)
		return fmt.Sprintf("https://pay.google.com/gp/v/save/%s", userID), nil
	}
	return saveURL, nil
}

// PushUpdate sends a points-updated push payload to a configurable webhook
// endpoint (WALLET_PUSH_ENDPOINT) and logs the event. The endpoint is optional;
// if not set the call is a structured no-op.
func (w *WalletPassportAdapter) PushUpdate(ctx context.Context, userID string, points int64) error {
	log.Printf("[WalletPassport] PushUpdate: userID=%s points=%d", userID, points)

	endpoint := os.Getenv("WALLET_PUSH_ENDPOINT")
	if endpoint == "" {
		return nil // not configured — skip HTTP call
	}

	body := newPushRequestBody(userID, points)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		log.Printf("[WalletPassport] PushUpdate: build request error: %v", err)
		return nil // non-fatal
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[WalletPassport] PushUpdate: HTTP error: %v", err)
		return nil // non-fatal
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		log.Printf("[WalletPassport] PushUpdate: upstream returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// ─── RebitesWalletAdapter (legacy — kept for backwards compatibility) ─────────

// RebitesWalletAdapter is the original stub adapter. It now delegates to
// WalletPassportAdapter for real pass generation.
type RebitesWalletAdapter struct {
	IssuerID string
	APIKey   string
	inner    *WalletPassportAdapter
}

// NewRebitesWalletAdapter returns the legacy adapter backed by real pass generation.
func NewRebitesWalletAdapter(uploader S3Uploader) *RebitesWalletAdapter {
	return &RebitesWalletAdapter{
		IssuerID: os.Getenv("GOOGLE_WALLET_ISSUER_ID"),
		APIKey:   os.Getenv("APPLE_PASS_CERTIFICATE_PASSWORD"),
		inner:    NewWalletPassportAdapter(uploader),
	}
}

// IssueApplePass delegates to the real Apple pass adapter.
func (a *RebitesWalletAdapter) IssueApplePass(ctx context.Context, userID string, points int64) (string, error) {
	return a.inner.IssueApplePass(ctx, userID, points)
}

// IssueGooglePass delegates to the real Google Wallet adapter.
func (a *RebitesWalletAdapter) IssueGooglePass(ctx context.Context, userID string, points int64) (string, error) {
	return a.inner.IssueGooglePass(ctx, userID, points)
}

// PushUpdate logs the wallet push event. streak and currentDataMB are legacy
// parameters retained for call-site compatibility.
func (a *RebitesWalletAdapter) PushUpdate(ctx context.Context, userID string, points int64, streak int, currentDataMB int) error {
	nudge := fmt.Sprintf("You are missing out on 200MB today. Your %d-day streak is at risk.", streak)
	if streak >= 5 {
		nudge = "Toggle MTN back on to save your N50,000 Jackpot entry."
	}
	log.Printf("[WalletPush] Updating User %s | Points: %d | Nudge: %s", userID, points, nudge)

	// Also trigger unified push
	_ = a.inner.PushUpdate(ctx, userID, points)
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// newPushRequestBody builds the push notification payload sent to
// WALLET_PUSH_ENDPOINT when a user's points balance changes.
func newPushRequestBody(userID string, points int64) []byte {
	payload := map[string]interface{}{
		"userID": userID,
		"points": points,
		"message": fmt.Sprintf(
			"Your Loyalty Nexus balance is now %d Pulse Points!", points),
	}
	b, _ := json.Marshal(payload)
	return b
}
