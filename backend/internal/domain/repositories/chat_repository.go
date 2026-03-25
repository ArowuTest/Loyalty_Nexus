package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
)

type ChatMessage struct {
	Role    string `db:"role"`    // "user" | "assistant"
	Content string `db:"content"`
}

type ChatSession struct {
	ID             uuid.UUID `db:"id"`
	UserID         uuid.UUID `db:"user_id"`
	Status         string    `db:"status"`
	LastActivityAt interface{} `db:"last_activity_at"`
}

type ChatRepository interface {
	// Sessions
	CreateSession(ctx context.Context, userID uuid.UUID) (*ChatSession, error)
	GetActiveSession(ctx context.Context, userID uuid.UUID) (*ChatSession, error)
	ExpireSession(ctx context.Context, sessionID uuid.UUID) error
	ListStaleActiveSessions(ctx context.Context, idleMinutes int, limit int) ([]ChatSession, error)

	// Messages
	AppendMessage(ctx context.Context, sessionID uuid.UUID, role, content string) error
	GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]ChatMessage, error)

	// Summaries (long-term memory, REQ-4.3.4)
	SaveSummary(ctx context.Context, userID uuid.UUID, summary string) error
	GetLastSummaries(ctx context.Context, userID uuid.UUID, n int) ([]string, error)

	// Rate limiting
	CountMessagesToday(ctx context.Context, userID uuid.UUID) (int, error)
}

// Ensure entities package is used to avoid unused import
var _ = entities.User{}
