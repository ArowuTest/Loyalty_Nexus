package entities

// passport.go — Domain entities for the Digital Passport feature.
// All structs carry both `db:` and `gorm:"column:..."` tags, matching the
// project convention established in transaction.go.

import (
	"time"

	"github.com/google/uuid"
)

// ─── PassportEvent ────────────────────────────────────────────────────────────

// PassportEvent records every significant passport action for a user:
// tier changes, badge awards, streak milestones, wallet pass issuances.
type PassportEvent struct {
	ID        uuid.UUID `db:"id"         gorm:"column:id;primaryKey;type:uuid"    json:"id"`
	UserID    uuid.UUID `db:"user_id"    gorm:"column:user_id;index;type:uuid"    json:"user_id"`
	EventType string    `db:"event_type" gorm:"column:event_type;not null"        json:"event_type"`
	Details   string    `db:"details"    gorm:"column:details;type:text"          json:"details"`
	CreatedAt time.Time `db:"created_at" gorm:"column:created_at;autoCreateTime"  json:"created_at"`
}

func (PassportEvent) TableName() string { return "passport_events" }

// ─── WalletRegistration ───────────────────────────────────────────────────────

// WalletRegistration tracks Apple Wallet device registrations for push updates.
// The Apple Wallet web service calls POST /devices/{deviceID}/registrations/{passTypeID}/{serialNumber}
// to register a device, and DELETE to unregister.
type WalletRegistration struct {
	ID                 uuid.UUID  `db:"id"                    gorm:"column:id;primaryKey;type:uuid"    json:"id"`
	UserID             uuid.UUID  `db:"user_id"               gorm:"column:user_id;index;type:uuid"    json:"user_id"`
	Platform           string     `db:"platform"              gorm:"column:platform;not null"          json:"platform"` // "apple" | "google"
	DeviceID           string     `db:"device_id"             gorm:"column:device_id;not null"         json:"device_id"`
	PushToken          string     `db:"push_token"            gorm:"column:push_token;not null"        json:"push_token"`
	SerialNumber       string     `db:"serial_number"         gorm:"column:serial_number;not null"     json:"serial_number"`
	IsActive           bool       `db:"is_active"             gorm:"column:is_active;default:true"     json:"is_active"`
	PushTokenUpdatedAt *time.Time `db:"push_token_updated_at" gorm:"column:push_token_updated_at"      json:"push_token_updated_at,omitempty"`
	CreatedAt          time.Time  `db:"created_at"            gorm:"column:created_at;autoCreateTime"  json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"            gorm:"column:updated_at;autoUpdateTime"  json:"updated_at"`
}

func (WalletRegistration) TableName() string { return "wallet_registrations" }

// ─── GoogleWalletObject ───────────────────────────────────────────────────────

// GoogleWalletObject persists the Google Wallet loyalty object ID for a user
// so we can update the object (points, tier) via the Google Wallet REST API.
type GoogleWalletObject struct {
	ID                uuid.UUID `db:"id"                  gorm:"column:id;primaryKey;type:uuid"       json:"id"`
	UserID            uuid.UUID `db:"user_id"             gorm:"column:user_id;uniqueIndex;type:uuid"  json:"user_id"`
	ObjectID          string    `db:"object_id"           gorm:"column:object_id;not null"            json:"object_id"`
	ClassID           string    `db:"class_id"            gorm:"column:class_id;not null"             json:"class_id"`
	StreakExpiryAlert bool      `db:"streak_expiry_alert" gorm:"column:streak_expiry_alert;default:false" json:"streak_expiry_alert"`
	LastSyncedAt      time.Time `db:"last_synced_at"      gorm:"column:last_synced_at"                json:"last_synced_at"`
	LastSyncStatus    string    `db:"last_sync_status"    gorm:"column:last_sync_status"              json:"last_sync_status"`
	CreatedAt         time.Time `db:"created_at"          gorm:"column:created_at;autoCreateTime"     json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"          gorm:"column:updated_at;autoUpdateTime"     json:"updated_at"`
}

func (GoogleWalletObject) TableName() string { return "google_wallet_objects" }

// ─── GhostNudgeLog ────────────────────────────────────────────────────────────

// GhostNudgeLog records every Ghost Nudge SMS sent to a user.
// Used by the admin panel to audit nudge activity and prevent over-messaging.
type GhostNudgeLog struct {
	ID        uuid.UUID `db:"id"         gorm:"column:id;primaryKey;type:uuid"    json:"id"`
	UserID    uuid.UUID `db:"user_id"    gorm:"column:user_id;index;type:uuid"    json:"user_id"`
	NudgedAt  time.Time `db:"nudged_at"  gorm:"column:nudged_at;not null"         json:"nudged_at"`
	Message   string    `db:"message"    gorm:"column:message;type:text"          json:"message"`
	Status    string    `db:"status"     gorm:"column:status"                     json:"status"` // "sent" | "failed"
	CreatedAt time.Time `db:"created_at" gorm:"column:created_at;autoCreateTime"  json:"created_at"`
}

func (GhostNudgeLog) TableName() string { return "ghost_nudge_log" }

// ─── USSDSession ──────────────────────────────────────────────────────────────

// USSDSession persists multi-step USSD session state between Africa's Talking
// callback requests. Each session is keyed by sessionID (provided by AT).
//
// PendingSpinID: when a user enters the spin confirmation sub-flow, the spin
// job ID is stored here. If the session expires before the user confirms, the
// spin is rolled back by the GhostNudgeWorker / session cleanup job.
type USSDSession struct {
	ID            uuid.UUID  `db:"id"              gorm:"column:id;primaryKey;type:uuid"        json:"id"`
	SessionID     string     `db:"session_id"      gorm:"column:session_id;uniqueIndex;not null" json:"session_id"`
	PhoneNumber   string     `db:"phone_number"    gorm:"column:phone_number;not null"           json:"phone_number"`
	MenuState     string     `db:"menu_state"      gorm:"column:menu_state;not null"             json:"menu_state"`
	InputBuffer   string     `db:"input_buffer"    gorm:"column:input_buffer;type:text"          json:"input_buffer"`
	// PendingSpinID holds the ID of a spin started but not yet confirmed.
	// Non-nil means a rollback is required if the session expires.
	PendingSpinID *uuid.UUID `db:"pending_spin_id" gorm:"column:pending_spin_id;type:uuid"       json:"pending_spin_id,omitempty"`
	ExpiresAt     time.Time  `db:"expires_at"      gorm:"column:expires_at;not null"             json:"expires_at"`
	CreatedAt     time.Time  `db:"created_at"      gorm:"column:created_at;autoCreateTime"       json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"      gorm:"column:updated_at;autoUpdateTime"       json:"updated_at"`
}

func (USSDSession) TableName() string { return "ussd_sessions" }
