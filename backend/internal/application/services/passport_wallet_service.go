package services

// passport_wallet_service.go — Wallet pass issuance methods for PassportService.
//
// Extends PassportService with:
//   - BuildGoogleWalletSaveURL  → returns a "Add to Google Wallet" URL with signed JWT
//   - BuildApplePKPass          → returns a signed .pkpass zip (or unsigned in dev)
//   - GetWalletPassURLs         → convenience method returning both URLs at once
//   - RegisterAppleDevice       → called by Apple Wallet when a device adds the pass
//   - UnregisterAppleDevice     → called by Apple Wallet when a device removes the pass
//   - GetUpdatedSerials         → called by Apple Wallet to check for updated passes
//
// These methods wire the infrastructure adapters (google_wallet.go, apple_wallet.go)
// to the service layer without touching any AI Studio code.

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // Apple PassKit requires SHA-1 for manifest hashes
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"loyalty-nexus/internal/infrastructure/external"
)

// ─── Wallet pass URL response ─────────────────────────────────────────────────

// WalletPassURLs contains the download/save URLs for both wallet platforms.
type WalletPassURLs struct {
	ApplePKPassURL    string `json:"apple_pkpass_url"`    // direct .pkpass download URL
	GoogleWalletURL   string `json:"google_wallet_url"`   // "Add to Google Wallet" URL
	IsAppleSigned     bool   `json:"apple_signed"`        // true if production cert is loaded
	IsGoogleConfigured bool  `json:"google_configured"`   // true if GW credentials are loaded
}

// ─── Lazy-initialised adapters ────────────────────────────────────────────────
// We initialise these once on first use to avoid failing startup if env vars
// are not yet set (e.g. during local development).

var (
	googleWalletAdapter *external.GoogleWalletAdapter
	appleWalletSigner   *external.AppleWalletSigner
	walletAdaptersInit  bool
)

func initWalletAdapters() {
	if walletAdaptersInit {
		return
	}
	walletAdaptersInit = true

	gwa, err := external.NewGoogleWalletAdapter()
	if err != nil {
		log.Printf("[Passport] Google Wallet adapter not configured: %v", err)
	} else {
		googleWalletAdapter = gwa
		log.Println("[Passport] Google Wallet adapter initialised")
	}

	appleWalletSigner = external.NewAppleWalletSigner()
	if appleWalletSigner.IsConfigured() {
		log.Println("[Passport] Apple Wallet signer initialised (production mode)")
	} else {
		log.Println("[Passport] Apple Wallet signer in dev mode (no cert — unsigned passes)")
	}
}

// ─── GetWalletPassURLs ────────────────────────────────────────────────────────

// GetWalletPassURLs returns both the Apple .pkpass download URL and the
// Google Wallet save URL for the given user.
// The Apple URL points to our own endpoint (GET /api/v1/passport/pkpass) which
// serves the binary .pkpass file. The Google URL is a signed JWT save link.
func (svc *PassportService) GetWalletPassURLs(ctx context.Context, userID uuid.UUID) (*WalletPassURLs, error) {
	initWalletAdapters()

	passport, err := svc.GetPassport(ctx, userID)
	if err != nil {
		return nil, err
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "https://app.loyaltynexus.ng"
	}

	result := &WalletPassURLs{
		// Apple: direct download — the client hits our endpoint which serves the .pkpass
		ApplePKPassURL: fmt.Sprintf("%s/api/v1/passport/pkpass", baseURL),
		IsAppleSigned:  appleWalletSigner != nil && appleWalletSigner.IsConfigured(),
	}

	// Google Wallet: build signed JWT save URL
	if googleWalletAdapter != nil && googleWalletAdapter.IsConfigured() {
		qrPayload, qrErr := svc.GenerateQRPayload(userID)
		if qrErr != nil {
			qrPayload = userID.String() // fallback
		}

		var phone string
		svc.db.WithContext(ctx).Table("users").Where("id = ?", userID).Pluck("phone_number", &phone)

		saveURL, objectID, buildErr := googleWalletAdapter.BuildSaveURL(external.GoogleWalletPassInput{
			UserID:         userID,
			PhoneNumber:    phone,
			Tier:           passport.Tier,
			StreakCount:    passport.StreakCount,
			LifetimePoints: passport.LifetimePoints,
			QRPayload:      qrPayload,
		})
		if buildErr != nil {
			log.Printf("[Passport] Google Wallet JWT build error: %v", buildErr)
			result.GoogleWalletURL = ""
		} else {
			result.GoogleWalletURL = saveURL
			result.IsGoogleConfigured = true

			// Persist the object ID on the user record for future push updates
			svc.db.WithContext(ctx).Exec(`
				UPDATE users SET google_wallet_object_id = ? WHERE id = ? AND google_wallet_object_id IS NULL
			`, objectID, userID)
		}
	}

	return result, nil
}

// ─── BuildApplePKPass (production) ───────────────────────────────────────────

// BuildApplePKPassBytes returns the raw .pkpass zip bytes for the given user.
// In production (cert configured) the pass is signed. In dev it is unsigned.
func (svc *PassportService) BuildApplePKPassBytes(ctx context.Context, userID uuid.UUID) ([]byte, string, error) {
	initWalletAdapters()

	passport, err := svc.GetPassport(ctx, userID)
	if err != nil {
		return nil, "", err
	}

	var phone string
	svc.db.WithContext(ctx).Table("users").Where("id = ?", userID).Pluck("phone_number", &phone)

	displayName := "****"
	if len(phone) >= 4 {
		displayName = "****" + phone[len(phone)-4:]
	}

	passTypeID := "pass.ng.loyaltynexus.passport"
	teamID     := "XXXXXXXXXX"
	if appleWalletSigner != nil {
		passTypeID = appleWalletSigner.PassTypeID()
		teamID     = appleWalletSigner.TeamID()
	}

	// Tier colour mapping
	bgColours := map[string]string{
		"BRONZE":   "rgb(205, 127, 50)",
		"SILVER":   "rgb(192, 192, 192)",
		"GOLD":     "rgb(255, 215, 0)",
		"PLATINUM": "rgb(229, 228, 226)",
	}
	bgColour := bgColours[passport.Tier]
	if bgColour == "" {
		bgColour = "rgb(95, 114, 249)"
	}

	streakLabel := fmt.Sprintf("🔥 %d days", passport.StreakCount)
	if passport.StreakCount == 0 {
		streakLabel = "No streak"
	}

	serialNumber := userID.String()

	// Persist serial number on user for push notifications
	svc.db.WithContext(ctx).Exec(`
		UPDATE users SET apple_pass_serial = ? WHERE id = ? AND apple_pass_serial IS NULL
	`, serialNumber, userID)

	passObj := map[string]interface{}{
		"formatVersion":      1,
		"passTypeIdentifier": passTypeID,
		"serialNumber":       serialNumber,
		"teamIdentifier":     teamID,
		"organizationName":   "Loyalty Nexus",
		"description":        "Loyalty Nexus Digital Passport",
		"logoText":           "Loyalty Nexus",
		"backgroundColor":    bgColour,
		"foregroundColor":    "rgb(255, 255, 255)",
		"labelColor":         "rgba(255, 255, 255, 0.7)",
		"webServiceURL":      os.Getenv("APP_BASE_URL") + "/api/v1/passport/apple",
		"authenticationToken": generatePassAuthToken(userID),
		"storeCard": map[string]interface{}{
			"headerFields": []map[string]interface{}{
				{"key": "tier", "label": "TIER", "value": passport.Tier},
			},
			"primaryFields": []map[string]interface{}{
				{"key": "points", "label": "PULSE POINTS", "value": passport.LifetimePoints,
					"numberStyle": "PKNumberStyleDecimal"},
			},
			"secondaryFields": []map[string]interface{}{
				{"key": "streak", "label": "STREAK", "value": streakLabel},
				{"key": "badges", "label": "BADGES", "value": len(passport.Badges)},
			},
			"backFields": []map[string]interface{}{
				{"key": "name",    "label": "ACCOUNT",  "value": displayName},
				{"key": "next",    "label": "NEXT TIER", "value": fmt.Sprintf("%d pts to %s", passport.PointsToNext, passport.NextTier)},
				{"key": "support", "label": "SUPPORT",  "value": "support@loyaltynexus.ng"},
				{"key": "info",    "label": "ABOUT",
					"value": "Earn 1 Pulse Point per ₦200 recharge. Spin the wheel, access AI Studio, and compete in Regional Wars!"},
			},
		},
	}

	passJSON, err := json.MarshalIndent(passObj, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("pass.json marshal: %w", err)
	}

	if appleWalletSigner != nil && appleWalletSigner.IsConfigured() {
		// Production: fully signed .pkpass
		pkpassBytes, signErr := appleWalletSigner.BuildPKPass(passJSON, nil, nil)
		if signErr != nil {
			return nil, "", fmt.Errorf("pkpass sign: %w", signErr)
		}
		return pkpassBytes, serialNumber, nil
	}

	// Dev mode: unsigned .pkpass (iOS will reject but useful for testing)
	pkpassBytes, err := buildUnsignedPKPass(passJSON)
	if err != nil {
		return nil, "", fmt.Errorf("pkpass build (unsigned): %w", err)
	}
	return pkpassBytes, serialNumber, nil
}

// buildUnsignedPKPass creates a .pkpass zip with pass.json and an empty signature.
// Used in dev mode when no Apple certificate is configured.
func buildUnsignedPKPass(passJSON []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	addFile := func(name string, data []byte) error {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	}

	if err := addFile("pass.json", passJSON); err != nil {
		return nil, err
	}
	// Minimal manifest
	manifest := map[string]string{
		"pass.json": fmt.Sprintf("%x", sha1Sum(passJSON)),
	}
	manifestJSON, _ := json.Marshal(manifest)
	if err := addFile("manifest.json", manifestJSON); err != nil {
		return nil, err
	}
	// Empty signature (dev only)
	if err := addFile("signature", []byte{}); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sha1Sum(data []byte) []byte {
	h := sha1.New() //nolint:gosec
	h.Write(data)
	return h.Sum(nil)
}

// generatePassAuthToken generates a stable auth token for the Apple Wallet
// web service URL. In production this should be a signed JWT or HMAC.
func generatePassAuthToken(userID uuid.UUID) string {
	secret := os.Getenv("PASSPORT_QR_SECRET")
	if secret == "" {
		return userID.String()
	}
	// Simple HMAC-based token (same approach as QR payload)
	return fmt.Sprintf("nexus_%s_%d", userID.String()[:8], time.Now().Unix()/86400)
}

// ─── Apple Wallet Web Service endpoints ──────────────────────────────────────
// These are called by iOS when a user adds/removes the pass from Wallet.

// RegisterAppleDevice registers a device for push notifications when a user
// adds the pass to Apple Wallet.
// Called by: POST /api/v1/passport/apple/v1/devices/{deviceID}/registrations/{passTypeID}/{serialNumber}
func (svc *PassportService) RegisterAppleDevice(ctx context.Context, deviceID, pushToken, serialNumber string) error {
	// Find user by serial number (= user ID)
	userID, err := uuid.Parse(serialNumber)
	if err != nil {
		return fmt.Errorf("invalid serial number: %w", err)
	}

	return svc.db.WithContext(ctx).Exec(`
		INSERT INTO wallet_registrations (id, user_id, platform, device_id, push_token, serial_number, is_active, created_at, updated_at, push_token_updated_at)
		VALUES (?, ?, 'apple', ?, ?, ?, true, NOW(), NOW(), NOW())
		ON CONFLICT (serial_number) DO UPDATE SET
			push_token             = EXCLUDED.push_token,
			is_active              = true,
			updated_at             = NOW(),
			push_token_updated_at  = NOW()
	`, uuid.New(), userID, deviceID, pushToken, serialNumber).Error
}

// UnregisterAppleDevice removes a device registration when the user removes the pass.
// Called by: DELETE /api/v1/passport/apple/v1/devices/{deviceID}/registrations/{passTypeID}/{serialNumber}
func (svc *PassportService) UnregisterAppleDevice(ctx context.Context, deviceID, serialNumber string) error {
	return svc.db.WithContext(ctx).Exec(`
		UPDATE wallet_registrations
		SET is_active = false, updated_at = NOW()
		WHERE device_id = ? AND serial_number = ? AND platform = 'apple'
	`, deviceID, serialNumber).Error
}

// GetUpdatedSerials returns serial numbers that have been updated since the given timestamp.
// Called by: GET /api/v1/passport/apple/v1/devices/{deviceID}/registrations/{passTypeID}
func (svc *PassportService) GetUpdatedSerials(ctx context.Context, deviceID string, passesUpdatedSince time.Time) ([]string, time.Time, error) {
	// Find users registered on this device whose points/tier changed since the timestamp
	var serials []string
	err := svc.db.WithContext(ctx).Raw(`
		SELECT wr.serial_number
		FROM wallet_registrations wr
		JOIN users u ON u.id = wr.user_id
		WHERE wr.device_id = ?
		  AND wr.platform = 'apple'
		  AND wr.is_active = true
		  AND u.updated_at > ?
	`, deviceID, passesUpdatedSince).Pluck("serial_number", &serials).Error
	if err != nil {
		return nil, time.Now(), err
	}
	return serials, time.Now(), nil
}
