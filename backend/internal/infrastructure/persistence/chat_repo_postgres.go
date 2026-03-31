package persistence

import (
"context"
"fmt"
"loyalty-nexus/internal/domain/repositories"
"github.com/google/uuid"
"gorm.io/gorm"
)

type postgresChatRepository struct{ db *gorm.DB }

func NewPostgresChatRepository(db *gorm.DB) repositories.ChatRepository {
return &postgresChatRepository{db: db}
}

// ─── Sessions ─────────────────────────────────────────────────────────────────

func (r *postgresChatRepository) CreateSession(ctx context.Context, userID uuid.UUID, toolSlug string) (*repositories.ChatSession, error) {
if toolSlug == "" {
toolSlug = "general"
}
session := &repositories.ChatSession{
ID:       uuid.New(),
UserID:   userID,
Status:   "active",
ToolSlug: toolSlug,
}
err := r.db.WithContext(ctx).Table("chat_sessions").Create(map[string]interface{}{
"id":        session.ID,
"user_id":   session.UserID,
"status":    session.Status,
"tool_slug": session.ToolSlug,
}).Error
return session, err
}

func (r *postgresChatRepository) GetActiveSession(ctx context.Context, userID uuid.UUID, toolSlug string) (*repositories.ChatSession, error) {
if toolSlug == "" {
toolSlug = "general"
}
var s repositories.ChatSession
err := r.db.WithContext(ctx).Table("chat_sessions").
Where("user_id = ? AND tool_slug = ? AND status = 'active'", userID, toolSlug).
Order("created_at DESC").First(&s).Error
return &s, err
}

func (r *postgresChatRepository) ExpireSession(ctx context.Context, sessionID uuid.UUID) error {
return r.db.WithContext(ctx).Table("chat_sessions").
Where("id = ?", sessionID).
Update("status", "expired").Error
}

func (r *postgresChatRepository) ListStaleActiveSessions(ctx context.Context, idleMinutes int, limit int) ([]repositories.ChatSession, error) {
var sessions []repositories.ChatSession
r.db.WithContext(ctx).Table("chat_sessions").
Where(fmt.Sprintf("status = 'active' AND last_activity_at < NOW() - INTERVAL '%d minutes'", idleMinutes)).
Limit(limit).Find(&sessions)
return sessions, nil
}

// ─── Messages ─────────────────────────────────────────────────────────────────

func (r *postgresChatRepository) AppendMessage(ctx context.Context, sessionID uuid.UUID, role, content string) error {
_ = r.db.WithContext(ctx).Table("chat_sessions").
Where("id = ?", sessionID).
Update("last_activity_at", gorm.Expr("now()")).Error
return r.db.WithContext(ctx).Table("chat_messages").Create(map[string]interface{}{
"id":         uuid.New(),
"session_id": sessionID,
"role":       role,
"content":    content,
}).Error
}

func (r *postgresChatRepository) GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]repositories.ChatMessage, error) {
var msgs []repositories.ChatMessage
r.db.WithContext(ctx).Table("chat_messages").
Where("session_id = ?", sessionID).
Order("created_at ASC").
Find(&msgs)
return msgs, nil
}

func (r *postgresChatRepository) GetRecentMessages(ctx context.Context, sessionID uuid.UUID, limit int) ([]repositories.ChatMessage, error) {
var msgs []repositories.ChatMessage
r.db.WithContext(ctx).Table("chat_messages").
Where("session_id = ?", sessionID).
Order("created_at DESC").
Limit(limit).
Find(&msgs)
for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
msgs[i], msgs[j] = msgs[j], msgs[i]
}
return msgs, nil
}

// ─── Summaries ────────────────────────────────────────────────────────────────

func (r *postgresChatRepository) SaveSummary(ctx context.Context, userID uuid.UUID, toolSlug, summary string) error {
if toolSlug == "" {
toolSlug = "general"
}
return r.db.WithContext(ctx).Table("session_summaries").Create(map[string]interface{}{
"id":        uuid.New(),
"user_id":   userID,
"tool_slug": toolSlug,
"summary":   summary,
}).Error
}

func (r *postgresChatRepository) GetLastSummaries(ctx context.Context, userID uuid.UUID, toolSlug string, n int) ([]string, error) {
if toolSlug == "" {
toolSlug = "general"
}
var summaries []string
r.db.WithContext(ctx).Table("session_summaries").
Where("user_id = ? AND tool_slug = ?", userID, toolSlug).
Order("created_at DESC").
Limit(n).
Pluck("summary", &summaries)
return summaries, nil
}

// ─── Rate limiting ────────────────────────────────────────────────────────────

func (r *postgresChatRepository) CountMessagesToday(ctx context.Context, userID uuid.UUID) (int, error) {
var count int64
r.db.WithContext(ctx).Table("chat_messages").
Joins("JOIN chat_sessions ON chat_sessions.id = chat_messages.session_id").
Where("chat_sessions.user_id = ? AND chat_messages.created_at >= CURRENT_DATE", userID).
Count(&count)
return int(count), nil
}

// ─── Retention ────────────────────────────────────────────────────────────────

func (r *postgresChatRepository) DeleteOldSummarizedMessages(ctx context.Context, olderThanDays int) (int64, error) {
result := r.db.WithContext(ctx).Exec(fmt.Sprintf(`
DELETE FROM chat_messages
WHERE session_id IN (
SELECT id FROM chat_sessions
WHERE status = 'summarized'
  AND last_activity_at < NOW() - INTERVAL '%d days'
)
`, olderThanDays))
return result.RowsAffected, result.Error
}
