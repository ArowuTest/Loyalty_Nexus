package repositories

// passport_repository.go — Repository interfaces for the Digital Passport feature.
// Implementations live in infrastructure/persistence/gorm_passport_repository.go.

import (
	"context"
	"time"

	"loyalty-nexus/internal/domain/entities"

	"github.com/google/uuid"
)

// ─── PassportRepository ───────────────────────────────────────────────────────

// PassportRepository handles passport events, badge records, and tier queries.
type PassportRepository interface {
	// LogEvent appends a passport event for the user.
	LogEvent(ctx context.Context, event *entities.PassportEvent) error

	// GetEvents returns the most recent events for a user, ordered newest-first.
	GetEvents(ctx context.Context, userID uuid.UUID, limit int) ([]entities.PassportEvent, error)

	// GetStats returns aggregate passport stats for the admin panel.
	GetStats(ctx context.Context) (*PassportStats, error)
}

// PassportStats is a read-model returned by PassportRepository.GetStats.
type PassportStats struct {
	TotalPassports    int64            `json:"total_passports"`
	AppleDownloads    int64            `json:"apple_downloads"`
	GoogleSaves       int64            `json:"google_saves"`
	QRScansToday      int64            `json:"qr_scans_today"`
	TierBreakdown     []TierCount      `json:"tier_breakdown"`
	TopBadgeEarners   []BadgeEarner    `json:"top_badge_earners"`
}

// TierCount is a single row in the tier breakdown table.
type TierCount struct {
	Tier  string `json:"tier"`
	Count int64  `json:"count"`
}

// BadgeEarner is a single row in the top badge earners table.
type BadgeEarner struct {
	UserID      uuid.UUID `json:"user_id"`
	PhoneNumber string    `json:"phone_number"`
	BadgeCount  int64     `json:"badge_count"`
}

// ─── WalletRegistrationRepository ────────────────────────────────────────────

// WalletRegistrationRepository manages Apple Wallet device registrations.
type WalletRegistrationRepository interface {
	// Upsert inserts or updates a device registration.
	Upsert(ctx context.Context, reg *entities.WalletRegistration) error

	// Deactivate marks a registration as inactive (DELETE callback from iOS).
	Deactivate(ctx context.Context, deviceID, passTypeID, serialNumber string) error

	// GetUpdatedSerials returns serial numbers for passes updated since the given time.
	GetUpdatedSerials(ctx context.Context, deviceID, passTypeID string, since time.Time) ([]string, error)

	// GetActiveByUserID returns all active registrations for a user.
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]entities.WalletRegistration, error)
}

// ─── GoogleWalletObjectRepository ────────────────────────────────────────────

// GoogleWalletObjectRepository persists Google Wallet object IDs.
type GoogleWalletObjectRepository interface {
	// Upsert inserts or updates the Google Wallet object record for a user.
	Upsert(ctx context.Context, obj *entities.GoogleWalletObject) error

	// GetByUserID returns the Google Wallet object for a user, or nil if not found.
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.GoogleWalletObject, error)

	// GetSyncCandidates returns users whose wallet pass needs to be re-synced.
	// Returns users whose last_synced_at is older than syncOlderThan.
	GetSyncCandidates(ctx context.Context, syncOlderThan time.Time, limit int) ([]entities.GoogleWalletObject, error)
}

// ─── GhostNudgeRepository ────────────────────────────────────────────────────

// GhostNudgeRepository manages the ghost nudge log.
type GhostNudgeRepository interface {
	// Log records a nudge sent to a user.
	Log(ctx context.Context, entry *entities.GhostNudgeLog) error

	// WasNudgedSince returns true if the user was nudged after the given time.
	WasNudgedSince(ctx context.Context, userID uuid.UUID, since time.Time) (bool, error)

	// GetRecentLog returns the most recent nudge log entries for the admin panel.
	GetRecentLog(ctx context.Context, limit int) ([]GhostNudgeLogRow, error)
}

// GhostNudgeLogRow is a read-model row for the admin Ghost Nudge log table.
type GhostNudgeLogRow struct {
	UserID      uuid.UUID `json:"user_id"`
	PhoneNumber string    `json:"phone_number"`
	NudgedAt    time.Time `json:"nudged_at"`
	Message     string    `json:"message"`
	Status      string    `json:"status"`
}

// ─── USSDSessionRepository ───────────────────────────────────────────────────

// USSDSessionRepository persists multi-step USSD session state.
type USSDSessionRepository interface {
	// Upsert creates or updates a USSD session.
	Upsert(ctx context.Context, session *entities.USSDSession) error

	// GetBySessionID returns the session for the given Africa's Talking session ID.
	GetBySessionID(ctx context.Context, sessionID string) (*entities.USSDSession, error)

	// DeleteExpired removes sessions past their expiry time.
	DeleteExpired(ctx context.Context) error

	// GetActiveSessions returns active sessions for the admin monitor.
	GetActiveSessions(ctx context.Context, limit int) ([]entities.USSDSession, error)
}
