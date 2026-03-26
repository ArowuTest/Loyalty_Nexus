package external

// google_wallet.go — Google Wallet Loyalty Object generation and JWT signing.
//
// Google Wallet uses a "Skinny JWT" approach:
//   1. Build a LoyaltyObject JSON payload (tier, points, streak, QR code)
//   2. Wrap it in a JWT signed with a Google Service Account private key
//   3. Return the JWT — the frontend embeds it in a "Add to Google Wallet" button
//      URL: https://pay.google.com/gp/v/save/{jwt}
//
// No SDK required — pure Go using golang-jwt/jwt/v5.
//
// Environment variables required:
//   GOOGLE_WALLET_ISSUER_ID   — your Google Pay & Wallet Console issuer ID
//   GOOGLE_WALLET_CLASS_ID    — the loyalty class ID (created once in the console)
//   GOOGLE_WALLET_SA_EMAIL    — service account email
//   GOOGLE_WALLET_SA_KEY      — PEM-encoded RSA private key (newlines as \n)

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ─── Tier colour mapping ─────────────────────────────────────────────────────

var tierHexColour = map[string]string{
	"BRONZE":   "#CD7F32",
	"SILVER":   "#C0C0C0",
	"GOLD":     "#FFD700",
	"PLATINUM": "#E5E4E2",
}

// ─── Google Wallet data structures ──────────────────────────────────────────

type gwLocalizedString struct {
	DefaultValue gwTranslatedValue `json:"defaultValue"`
}

type gwTranslatedValue struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type gwImage struct {
	SourceURI gwSourceURI `json:"sourceUri"`
}

type gwSourceURI struct {
	URI string `json:"uri"`
}

type gwTextModuleData struct {
	Header string `json:"header"`
	Body   string `json:"body"`
	ID     string `json:"id"`
}

type gwBarcode struct {
	Type            string `json:"type"`
	Value           string `json:"value"`
	AlternateText   string `json:"alternateText"`
}

type gwLoyaltyPoints struct {
	Label  string              `json:"label"`
	Balance gwLoyaltyBalance   `json:"balance"`
}

type gwLoyaltyBalance struct {
	String string `json:"string"`
}

// GWLoyaltyObject is the Google Wallet LoyaltyObject payload.
type GWLoyaltyObject struct {
	ID                    string                `json:"id"`
	ClassID               string                `json:"classId"`
	State                 string                `json:"state"` // ACTIVE
	AccountID             string                `json:"accountId"`
	AccountName           string                `json:"accountName"`
	LoyaltyPoints         gwLoyaltyPoints       `json:"loyaltyPoints"`
	SecondaryLoyaltyPoints gwLoyaltyPoints      `json:"secondaryLoyaltyPoints"`
	Barcode               gwBarcode             `json:"barcode"`
	TextModulesData       []gwTextModuleData    `json:"textModulesData"`
	HexBackgroundColor    string                `json:"hexBackgroundColor"`
}

// GWSkinnyJWTClaims wraps the loyalty object in the Google Wallet JWT format.
type GWSkinnyJWTClaims struct {
	Iss     string          `json:"iss"`
	Aud     string          `json:"aud"`
	Typ     string          `json:"typ"`
	Iat     int64           `json:"iat"`
	Payload GWJWTPayload    `json:"payload"`
	jwt.RegisteredClaims
}

// GWJWTPayload holds the loyaltyObjects array inside the JWT.
type GWJWTPayload struct {
	LoyaltyObjects []GWLoyaltyObject `json:"loyaltyObjects"`
}

// ─── GoogleWalletAdapter ─────────────────────────────────────────────────────

// GoogleWalletAdapter generates Google Wallet loyalty pass JWTs.
type GoogleWalletAdapter struct {
	issuerID  string
	classID   string
	saEmail   string
	rsaKey    *rsa.PrivateKey
	baseURL   string
}

// NewGoogleWalletAdapter creates a new adapter from environment variables.
// Returns nil (not an error) if env vars are not set — callers should check.
func NewGoogleWalletAdapter() (*GoogleWalletAdapter, error) {
	issuerID := os.Getenv("GOOGLE_WALLET_ISSUER_ID")
	classID  := os.Getenv("GOOGLE_WALLET_CLASS_ID")
	saEmail  := os.Getenv("GOOGLE_WALLET_SA_EMAIL")
	saKey    := os.Getenv("GOOGLE_WALLET_SA_KEY")

	if issuerID == "" || classID == "" || saEmail == "" || saKey == "" {
		return nil, errors.New("google wallet env vars not configured (GOOGLE_WALLET_ISSUER_ID, GOOGLE_WALLET_CLASS_ID, GOOGLE_WALLET_SA_EMAIL, GOOGLE_WALLET_SA_KEY)")
	}

	// Support \n literal in env var (common in Docker/k8s secrets)
	saKey = strings.ReplaceAll(saKey, `\n`, "\n")

	block, _ := pem.Decode([]byte(saKey))
	if block == nil {
		return nil, errors.New("GOOGLE_WALLET_SA_KEY: failed to decode PEM block")
	}

	var rsaKey *rsa.PrivateKey
	var err error

	switch block.Type {
	case "RSA PRIVATE KEY":
		rsaKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			return nil, fmt.Errorf("GOOGLE_WALLET_SA_KEY: PKCS8 parse error: %w", parseErr)
		}
		var ok bool
		rsaKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("GOOGLE_WALLET_SA_KEY: not an RSA key")
		}
	default:
		return nil, fmt.Errorf("GOOGLE_WALLET_SA_KEY: unsupported PEM type %q", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("GOOGLE_WALLET_SA_KEY: parse error: %w", err)
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "https://app.loyaltynexus.ng"
	}

	return &GoogleWalletAdapter{
		issuerID: issuerID,
		classID:  classID,
		saEmail:  saEmail,
		rsaKey:   rsaKey,
		baseURL:  baseURL,
	}, nil
}

// IsConfigured returns true if the adapter was successfully initialised.
func (a *GoogleWalletAdapter) IsConfigured() bool {
	return a != nil && a.rsaKey != nil
}

// GoogleWalletPassInput holds the user data needed to build the pass.
type GoogleWalletPassInput struct {
	UserID         uuid.UUID
	PhoneNumber    string // masked to last 4 digits on the pass
	Tier           string
	StreakCount    int
	LifetimePoints int64
	QRPayload      string // signed QR string from PassportService.GenerateQRPayload
}

// BuildSaveURL generates a "Add to Google Wallet" URL for the given user.
// The URL embeds a signed JWT that Google Wallet uses to create the loyalty object.
func (a *GoogleWalletAdapter) BuildSaveURL(input GoogleWalletPassInput) (string, string, error) {
	objectID := fmt.Sprintf("%s.nexus_%s", a.issuerID, input.UserID.String())

	displayName := "****"
	if len(input.PhoneNumber) >= 4 {
		displayName = "****" + input.PhoneNumber[len(input.PhoneNumber)-4:]
	}

	bgColour := tierHexColour[input.Tier]
	if bgColour == "" {
		bgColour = "#5F72F9" // nexus-600 default
	}

	streakLabel := fmt.Sprintf("🔥 %d-day streak", input.StreakCount)
	if input.StreakCount == 0 {
		streakLabel = "No active streak"
	}

	obj := GWLoyaltyObject{
		ID:      objectID,
		ClassID: fmt.Sprintf("%s.%s", a.issuerID, a.classID),
		State:   "ACTIVE",
		AccountID:   input.UserID.String(),
		AccountName: displayName,
		LoyaltyPoints: gwLoyaltyPoints{
			Label: "Pulse Points",
			Balance: gwLoyaltyBalance{
				String: fmt.Sprintf("%d pts", input.LifetimePoints),
			},
		},
		SecondaryLoyaltyPoints: gwLoyaltyPoints{
			Label: "Tier",
			Balance: gwLoyaltyBalance{
				String: input.Tier,
			},
		},
		Barcode: gwBarcode{
			Type:          "QR_CODE",
			Value:         input.QRPayload,
			AlternateText: displayName,
		},
		TextModulesData: []gwTextModuleData{
			{ID: "streak",  Header: "Streak",         Body: streakLabel},
			{ID: "tier",    Header: "Loyalty Tier",   Body: input.Tier},
			{ID: "support", Header: "Support",        Body: "support@loyaltynexus.ng"},
		},
		HexBackgroundColor: bgColour,
	}

	claims := GWSkinnyJWTClaims{
		Iss: a.saEmail,
		Aud: "google",
		Typ: "savetowallet",
		Iat: time.Now().Unix(),
		Payload: GWJWTPayload{
			LoyaltyObjects: []GWLoyaltyObject{obj},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(a.rsaKey)
	if err != nil {
		return "", "", fmt.Errorf("google wallet jwt sign: %w", err)
	}

	saveURL := "https://pay.google.com/gp/v/save/" + signed
	return saveURL, objectID, nil
}
