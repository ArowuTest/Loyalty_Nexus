package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"loyalty-nexus/internal/domain/entities"
)

type ChatMessage struct {
	Role      string `db:"role"`       // "user" | "assistant"
	Content   string `db:"content"`
	CreatedAt string `db:"created_at"` // ISO timestamp for history restore
}

type ChatSession struct {
	ID             uuid.UUID `db:"id"               gorm:"column:id"`
	UserID         uuid.UUID `db:"user_id"          gorm:"column:user_id"`
	Status         string    `db:"status"           gorm:"column:status"`
	ToolSlug       string    `db:"tool_slug"        gorm:"column:tool_slug"`
	LastActivityAt time.Time `db:"last_activity_at" gorm:"column:last_activity_at"`
}

type ChatRepository interface {
	// Sessions
	CreateSession(ctx context.Context, userID uuid.UUID, toolSlug string) (*ChatSession, error)
	GetActiveSession(ctx context.Context, userID uuid.UUID, toolSlug string) (*ChatSession, error)
	ExpireSession(ctx context.Context, sessionID uuid.UUID) error
	ListStaleActiveSessions(ctx context.Context, idleMinutes int, limit int) ([]ChatSession, error)

	// Messages
	AppendMessage(ctx context.Context, sessionID uuid.UUID, role, content string) error
	GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]ChatMessage, error)
	GetRecentMessages(ctx context.Context, sessionID uuid.UUID, limit int) ([]ChatMessage, error)

	// Summaries (long-term memory, scoped by tool_slug)
	SaveSummary(ctx context.Context, userID uuid.UUID, toolSlug, summary string) error
	GetLastSummaries(ctx context.Context, userID uuid.UUID, toolSlug string, n int) ([]string, error)

	// Rate limiting
	CountMessagesToday(ctx context.Context, userID uuid.UUID) (int, error)

	// Retention
	DeleteOldSummarizedMessages(ctx context.Context, olderThanDays int) (int64, error)
}

// Ensure entities package is used to avoid unused import
var _ = entities.User{}
