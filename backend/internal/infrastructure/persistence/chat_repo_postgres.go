package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type postgresChatRepository struct{ db *gorm.DB }

func NewPostgresChatRepository(db *gorm.DB) repositories.ChatRepository {
	return &postgresChatRepository{db: db}
}

func (r *postgresChatRepository) CreateSession(ctx context.Context, userID uuid.UUID) (*repositories.ChatSession, error) {
	session := &repositories.ChatSession{ID: uuid.New(), UserID: userID, Status: "active"}
	err := r.db.WithContext(ctx).Table("chat_sessions").Create(session).Error
	return session, err
}

func (r *postgresChatRepository) GetActiveSession(ctx context.Context, userID uuid.UUID) (*repositories.ChatSession, error) {
	var s repositories.ChatSession
	err := r.db.WithContext(ctx).Table("chat_sessions").
		Where("user_id = ? AND status = 'active'", userID).
		Order("created_at DESC").First(&s).Error
	return &s, err
}

func (r *postgresChatRepository) ExpireSession(ctx context.Context, sessionID uuid.UUID) error {
	return r.db.WithContext(ctx).Table("chat_sessions").Where("id = ?", sessionID).Update("status", "expired").Error
}

func (r *postgresChatRepository) ListStaleActiveSessions(ctx context.Context, idleMinutes int, limit int) ([]repositories.ChatSession, error) {
	var sessions []repositories.ChatSession
	r.db.WithContext(ctx).Table("chat_sessions").
		Where("status = 'active' AND last_activity_at < NOW() - INTERVAL '? minutes'", idleMinutes).
		Limit(limit).Find(&sessions)
	return sessions, nil
}

func (r *postgresChatRepository) AppendMessage(ctx context.Context, sessionID uuid.UUID, role, content string) error {
	return r.db.WithContext(ctx).Table("chat_messages").Create(map[string]interface{}{
		"id": uuid.New(), "session_id": sessionID, "role": role, "content": content,
	}).Error
}

func (r *postgresChatRepository) GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]repositories.ChatMessage, error) {
	var msgs []repositories.ChatMessage
	r.db.WithContext(ctx).Table("chat_messages").Where("session_id = ?", sessionID).Order("created_at ASC").Find(&msgs)
	return msgs, nil
}

func (r *postgresChatRepository) SaveSummary(ctx context.Context, userID uuid.UUID, summary string) error {
	return r.db.WithContext(ctx).Table("session_summaries").Create(map[string]interface{}{
		"id": uuid.New(), "user_id": userID, "summary": summary,
	}).Error
}

func (r *postgresChatRepository) GetLastSummaries(ctx context.Context, userID uuid.UUID, n int) ([]string, error) {
	var summaries []string
	r.db.WithContext(ctx).Table("session_summaries").
		Where("user_id = ?", userID).
		Order("created_at DESC").Limit(n).Pluck("summary", &summaries)
	return summaries, nil
}

func (r *postgresChatRepository) CountMessagesToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	r.db.WithContext(ctx).Table("chat_messages").
		Joins("JOIN chat_sessions ON chat_sessions.id = chat_messages.session_id").
		Where("chat_sessions.user_id = ? AND chat_messages.created_at >= CURRENT_DATE", userID).
		Count(&count)
	return int(count), nil
}
