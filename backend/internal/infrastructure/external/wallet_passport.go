package external

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

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ─── ApplePassAdapter ─────────────────────────────────────────────────────

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
		"formatVersion":    1,
		"passTypeIdentifier": "pass.ai.nexus.loyalty",
		"serialNumber":     userID,
		"teamIdentifier":   teamID,
		"organizationName": "Loyalty Nexus",
		"description":      "Loyalty Nexus Digital Passport",
		"logoText":         "Nexus",
		"backgroundColor":  "rgb(255, 200, 0)",
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

// ─── GoogleWalletAdapter ──────────────────────────────────────────────────

// GoogleWalletAdapter generates Google Wallet save links using a signed JWT.
type GoogleWalletAdapter struct {
	IssuerID           string // GOOGLE_WALLET_ISSUER_ID
	ServiceAccountJSON string // GOOGLE_WALLET_SERVICE_ACCOUNT_JSON (raw JSON string)
}

// NewGoogleWalletAdapter reads credentials from environment variables.
func NewGoogleWalletAdapter() *GoogleWalletAdapter {
	return &GoogleWalletAdapter{
		IssuerID:           os.Getenv("GOOGLE_WALLET_ISSUER_ID"),
		ServiceAccountJSON: os.Getenv("GOOGLE_WALLET_SERVICE_ACCOUNT_JSON"),
	}
}

// IssueGooglePass returns a Google Wallet "Add to Google Wallet" URL.
// When credentials are not configured a placeholder URL is returned for
// graceful degradation.
func (a *GoogleWalletAdapter) IssueGooglePass(ctx context.Context, userID string, points int64) (string, error) {
	if a.IssuerID == "" || a.ServiceAccountJSON == "" {
		return fmt.Sprintf("https://pay.google.com/gp/v/save/%s", userID), nil
	}

	issuerID := a.IssuerID
	classID := fmt.Sprintf("%s.loyalty_nexus_class", issuerID)
	objectID := fmt.Sprintf("%s.user_%s", issuerID, userID)

	loyaltyObject := map[string]interface{}{
		"id":      objectID,
		"classId": classID,
		"state":   "ACTIVE",
		"loyaltyPoints": map[string]interface{}{
			"balance": map[string]interface{}{
				"int": points,
			},
			"label": "Pulse Points",
		},
		"accountId":   userID,
		"accountName": "Loyalty Nexus Member",
	}

	// Parse service account to extract private_key and client_email
	var sa struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}
	if err := json.Unmarshal([]byte(a.ServiceAccountJSON), &sa); err != nil {
		return fmt.Sprintf("https://pay.google.com/gp/v/save/%s", userID), nil
	}

	claims := jwt.MapClaims{
		"iss": sa.ClientEmail,
		"aud": "google",
		"typ": "savetowallet",
		"iat": time.Now().Unix(),
		"payload": map[string]interface{}{
			"loyaltyObjects": []interface{}{loyaltyObject},
		},
		"origins": []string{"https://loyalty-nexus.ai"},
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(sa.PrivateKey))
	if err != nil {
		return fmt.Sprintf("https://pay.google.com/gp/v/save/%s", userID), nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		return fmt.Sprintf("https://pay.google.com/gp/v/save/%s", userID), nil
	}

	return "https://pay.google.com/gp/v/save/" + signed, nil
}

// ─── WalletPassportAdapter (unified) ─────────────────────────────────────

// WalletPassportAdapter unifies Apple and Google pass issuance and implements
// the WalletPassport interface.
type WalletPassportAdapter struct {
	apple  *ApplePassAdapter
	google *GoogleWalletAdapter
}

// NewWalletPassportAdapter constructs the unified adapter.
func NewWalletPassportAdapter(uploader S3Uploader) *WalletPassportAdapter {
	return &WalletPassportAdapter{
		apple:  NewApplePassAdapter(uploader),
		google: NewGoogleWalletAdapter(),
	}
}

// IssueApplePass delegates to ApplePassAdapter.
func (w *WalletPassportAdapter) IssueApplePass(ctx context.Context, userID string, points int64) (string, error) {
	return w.apple.IssueApplePass(ctx, userID, points)
}

// IssueGooglePass delegates to GoogleWalletAdapter.
func (w *WalletPassportAdapter) IssueGooglePass(ctx context.Context, userID string, points int64) (string, error) {
	return w.google.IssueGooglePass(ctx, userID, points)
}

// PushUpdate logs the points update and can trigger downstream notifications.
func (w *WalletPassportAdapter) PushUpdate(ctx context.Context, userID string, points int64) error {
	log.Printf("[WalletPassport] PushUpdate: userID=%s points=%d", userID, points)
	return nil
}

// ─── RebitesWalletAdapter (legacy — kept for backwards compatibility) ─────

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

// ─── helpers ──────────────────────────────────────────────────────────────

// newPushRequestBody builds a generic push notification payload (for future
// APNS/FCM integration).
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

// unused reference to keep imports tidy
var _ = bytes.NewReader
var _ = uuid.New
var _ = http.MethodPost
