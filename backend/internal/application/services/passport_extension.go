package services

// passport_extension.go — QR generation, Apple Wallet PKPass, Ghost Nudge,
// Passport Events log, ShareableCard (spec §6 "Digital Passport")
//
// Master Spec §6.1:
//   - Every user gets a Digital Passport with: QR code, tier badge, streak flame,
//     lifetime points, earned badges
//   - QR code encodes: user_id + HMAC-SHA256 (using PASSPORT_QR_SECRET env var)
//     so partner merchants can verify without a round-trip to the DB
//   - Apple Wallet PKPass — generic pass structure (no private key required for
//     metadata; signing happens at download time in production)
//
// Ghost Nudge (spec §6.3):
//   - When streak is 1 day away from expiry (last_recharge_at < 23h ago),
//     send SMS nudge: "Your 🔥{n}-day streak expires in 1 hour. Recharge now!"
//   - Cooldown: don't re-nudge the same user within 24h
//
// Passport Events (spec §6.4):
//   - Every tier change, badge earn, streak milestone → logged to passport_events
//     for admin audit and mobile history feed

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// ─── QR Code ─────────────────────────────────────────────────────────────────

// QRPayload is what gets base64-encoded into the QR code value.
type QRPayload struct {
	UserID    string `json:"uid"`
	IssuedAt  int64  `json:"iat"`
	Signature string `json:"sig"`
}

// GenerateQRPayload creates a tamper-proof QR string for the user's passport.
// Format: base64(JSON{uid, iat, sig}) where sig = HMAC-SHA256(uid+":"+iat, secret)
func (svc *PassportService) GenerateQRPayload(userID uuid.UUID) (string, error) {
	secret := os.Getenv("PASSPORT_QR_SECRET")
	if secret == "" {
		secret = "nexus-dev-qr-secret" // fallback for dev only
	}
	now := time.Now().Unix()
	msg := fmt.Sprintf("%s:%d", userID.String(), now)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))

	payload := QRPayload{
		UserID:   userID.String(),
		IssuedAt: now,
		Signature: sig,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("qr marshal: %w", err)
	}
	return base64.URLEncoding.EncodeToString(raw), nil
}

// VerifyQRPayload verifies a QR string from a partner merchant scan.
// Returns the user_id if valid; error if tampered or expired (>5 min).
func (svc *PassportService) VerifyQRPayload(encoded string) (uuid.UUID, error) {
	secret := os.Getenv("PASSPORT_QR_SECRET")
	if secret == "" {
		secret = "nexus-dev-qr-secret"
	}
	raw, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid qr encoding: %w", err)
	}
	var payload QRPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return uuid.Nil, fmt.Errorf("invalid qr payload: %w", err)
	}

	// Expiry: 5 minutes
	if time.Now().Unix()-payload.IssuedAt > 300 {
		return uuid.Nil, fmt.Errorf("qr code expired")
	}

	// Verify HMAC
	msg := fmt.Sprintf("%s:%d", payload.UserID, payload.IssuedAt)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(payload.Signature)) {
		return uuid.Nil, fmt.Errorf("qr signature mismatch")
	}

	uid, err := uuid.Parse(payload.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user_id in qr: %w", err)
	}
	return uid, nil
}

// ─── Apple Wallet PKPass Structure (spec §6.2) ───────────────────────────────

// PKPassGeneric contains the metadata fields for a generic.pass (Apple Wallet).
// Signing is production-only (requires Apple WWDR cert + pass cert + private key).
// This struct is serialised to pass.json; the HTTP handler wraps it in a .pkpass zip.
type PKPassGeneric struct {
	FormatVersion       int            `json:"formatVersion"`
	PassTypeIdentifier  string         `json:"passTypeIdentifier"`
	SerialNumber        string         `json:"serialNumber"`
	TeamIdentifier      string         `json:"teamIdentifier"`
	OrganizationName    string         `json:"organizationName"`
	Description         string         `json:"description"`
	BackgroundColor     string         `json:"backgroundColor"`
	ForegroundColor     string         `json:"foregroundColor"`
	LabelColor          string         `json:"labelColor"`
	LogoText            string         `json:"logoText"`
	Barcode             *PKPassBarcode `json:"barcode,omitempty"`
	Generic             PKPassFields   `json:"generic"`
}

type PKPassBarcode struct {
	Message         string `json:"message"`
	Format          string `json:"format"` // PKBarcodeFormatQR
	MessageEncoding string `json:"messageEncoding"`
	AltText         string `json:"altText,omitempty"`
}

type PKPassFields struct {
	PrimaryFields   []PKPassField `json:"primaryFields"`
	SecondaryFields []PKPassField `json:"secondaryFields"`
	AuxiliaryFields []PKPassField `json:"auxiliaryFields"`
	BackFields      []PKPassField `json:"backFields"`
}

type PKPassField struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	Value         interface{} `json:"value"`
	TextAlignment string `json:"textAlignment,omitempty"`
}

// BuildPKPass constructs the PKPass metadata for a user's digital passport.
func (svc *PassportService) BuildPKPass(ctx context.Context, userID uuid.UUID) (*PKPassGeneric, error) {
	passport, err := svc.GetPassport(ctx, userID)
	if err != nil {
		return nil, err
	}

	qr, err := svc.GenerateQRPayload(userID)
	if err != nil {
		return nil, fmt.Errorf("qr generate: %w", err)
	}

	teamID := os.Getenv("APPLE_TEAM_ID")
	if teamID == "" {
		teamID = "NEXUS_DEV"
	}
	passTypeID := os.Getenv("APPLE_PASS_TYPE_ID")
	if passTypeID == "" {
		passTypeID = "pass.ng.loyaltynexus.passport"
	}

	// Tier colours
	bgColor := "rgb(30,30,30)"
	switch passport.Tier {
	case "BRONZE":   bgColor = "rgb(140,90,50)"
	case "SILVER":   bgColor = "rgb(120,120,140)"
	case "GOLD":     bgColor = "rgb(180,140,0)"
	case "PLATINUM": bgColor = "rgb(20,100,160)"
	}

	streakText := fmt.Sprintf("🔥 %d day streak", passport.StreakCount)

	return &PKPassGeneric{
		FormatVersion:      1,
		PassTypeIdentifier: passTypeID,
		SerialNumber:       userID.String(),
		TeamIdentifier:     teamID,
		OrganizationName:   "Loyalty Nexus",
		Description:        "Your Loyalty Nexus Digital Passport",
		BackgroundColor:    bgColor,
		ForegroundColor:    "rgb(255,255,255)",
		LabelColor:         "rgb(200,200,200)",
		LogoText:           "NEXUS",
		Barcode: &PKPassBarcode{
			Message:         qr,
			Format:          "PKBarcodeFormatQR",
			MessageEncoding: "iso-8859-1",
			AltText:         passport.Tier + " Member",
		},
		Generic: PKPassFields{
			PrimaryFields: []PKPassField{
				{Key: "tier", Label: "TIER", Value: passport.Tier},
			},
			SecondaryFields: []PKPassField{
				{Key: "points",  Label: "PULSE POINTS",   Value: passport.LifetimePoints},
				{Key: "streak",  Label: "STREAK",         Value: streakText},
			},
			AuxiliaryFields: []PKPassField{
				{Key: "badges",  Label: "BADGES EARNED",  Value: len(passport.Badges)},
				{Key: "next",    Label: "NEXT TIER IN",   Value: fmt.Sprintf("%d pts", passport.PointsToNext)},
			},
			BackFields: []PKPassField{
				{Key: "info",    Label: "About Loyalty Nexus",
					Value: "Earn 1 Pulse Point per ₦200 recharge. Spin the wheel, access AI Studio, and compete in Regional Wars!"},
				{Key: "support", Label: "Support", Value: "support@loyaltynexus.ng"},
			},
		},
	}, nil
}

// ─── Passport Events Log ─────────────────────────────────────────────────────

type PassportEventType string

const (
	PassportEventTierUp    PassportEventType = "tier_upgrade"
	PassportEventBadge     PassportEventType = "badge_earned"
	PassportEventStreak    PassportEventType = "streak_milestone"
	PassportEventQRScanned PassportEventType = "qr_scanned"
)

// LogPassportEvent records a passport lifecycle event.
func (svc *PassportService) LogPassportEvent(ctx context.Context, userID uuid.UUID, eventType PassportEventType, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)
	return svc.db.WithContext(ctx).Exec(`
		INSERT INTO passport_events (id, user_id, event_type, details, created_at)
		VALUES (?, ?, ?, ?, NOW())
	`, uuid.New(), userID, string(eventType), string(detailsJSON)).Error
}

// GetPassportEvents returns the event history for a user's passport.
func (svc *PassportService) GetPassportEvents(ctx context.Context, userID uuid.UUID, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var events []map[string]interface{}
	err := svc.db.WithContext(ctx).
		Table("passport_events").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// ─── Ghost Nudge (spec §6.3) ─────────────────────────────────────────────────

// GhostNudgeCandidate is a user whose streak is about to expire.
type GhostNudgeCandidate struct {
	UserID      uuid.UUID `gorm:"column:id"`
	PhoneNumber string    `gorm:"column:phone_number"`
	StreakCount int       `gorm:"column:streak_count"`
}

// GetGhostNudgeCandidates returns users whose last recharge was 23-24h ago
// and who haven't been nudged in the last 24h.
func (svc *PassportService) GetGhostNudgeCandidates(ctx context.Context) ([]GhostNudgeCandidate, error) {
	var candidates []GhostNudgeCandidate
	err := svc.db.WithContext(ctx).Raw(`
		SELECT u.id, u.phone_number, u.streak_count
		FROM users u
		WHERE u.is_active = true
		  AND u.streak_count > 0
		  AND u.last_recharge_at BETWEEN NOW() - INTERVAL '24 hours' AND NOW() - INTERVAL '23 hours'
		  AND u.id NOT IN (
			  SELECT user_id FROM ghost_nudge_log
			  WHERE nudged_at > NOW() - INTERVAL '24 hours'
		  )
		LIMIT 500
	`).Scan(&candidates).Error
	return candidates, err
}

// RecordGhostNudge marks that a user was nudged (prevents re-nudge within 24h).
func (svc *PassportService) RecordGhostNudge(ctx context.Context, userID uuid.UUID) error {
	return svc.db.WithContext(ctx).Exec(`
		INSERT INTO ghost_nudge_log (id, user_id, nudged_at)
		VALUES (?, ?, NOW())
		ON CONFLICT (user_id) DO UPDATE SET nudged_at = NOW()
	`, uuid.New(), userID).Error
}

// ─── Shareable Card (spec §6.5) ──────────────────────────────────────────────

// ShareableCardData is the payload returned to the mobile app for rendering a
// shareable "achievement card" (PNG generation happens on the Flutter client).
type ShareableCardData struct {
	UserID         uuid.UUID         `json:"user_id"`
	DisplayName    string            `json:"display_name"` // last 4 digits of MSISDN
	Tier           string            `json:"tier"`
	StreakCount    int               `json:"streak_count"`
	LifetimePoints int64             `json:"lifetime_points"`
	TopBadges      []BadgeDefinition `json:"top_badges"` // max 3
	ShareURL       string            `json:"share_url"`
	GeneratedAt    time.Time         `json:"generated_at"`
}

// GetShareableCard returns the card data for social sharing.
func (svc *PassportService) GetShareableCard(ctx context.Context, userID uuid.UUID) (*ShareableCardData, error) {
	passport, err := svc.GetPassport(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get user phone number (display last 4 digits only)
	var phone string
	svc.db.WithContext(ctx).Table("users").Where("id = ?", userID).Pluck("phone_number", &phone)
	displayName := "****"
	if len(phone) >= 4 {
		displayName = "****" + phone[len(phone)-4:]
	}

	topBadges := passport.Badges
	if len(topBadges) > 3 {
		topBadges = topBadges[:3]
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "https://app.loyaltynexus.ng"
	}

	return &ShareableCardData{
		UserID:         userID,
		DisplayName:    displayName,
		Tier:           passport.Tier,
		StreakCount:    passport.StreakCount,
		LifetimePoints: passport.LifetimePoints,
		TopBadges:      topBadges,
		ShareURL:       fmt.Sprintf("%s/profile/%s", baseURL, userID.String()),
		GeneratedAt:    time.Now(),
	}, nil
}
