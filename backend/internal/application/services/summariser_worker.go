package services

import (
"context"
"fmt"
"log"
"time"

"github.com/google/uuid"
"gorm.io/gorm"
"loyalty-nexus/internal/infrastructure/external"
)

type SummariserWorker struct {
db              *gorm.DB
llmOrchestrator *external.LLMOrchestrator
}

func NewSummariserWorker(db *gorm.DB, llm *external.LLMOrchestrator) *SummariserWorker {
return &SummariserWorker{db: db, llmOrchestrator: llm}
}

func (w *SummariserWorker) Run(ctx context.Context) {
ticker := time.NewTicker(30 * time.Minute)
for {
select {
case <-ctx.Done():
return
case <-ticker.C:
w.ProcessExpiredSessions(ctx)
w.CleanupOldMessages(ctx)
}
}
}

func (w *SummariserWorker) ProcessExpiredSessions(ctx context.Context) {
var sessions []struct {
ID       uuid.UUID
UserID   uuid.UUID
ToolSlug string `gorm:"column:tool_slug"`
}
// REQ-4.3.4: Sessions expire after 30 minutes of inactivity
query := "status = 'active' AND last_activity_at < now() - interval '30 minutes'"
if err := w.db.WithContext(ctx).Table("chat_sessions").
Select("id, user_id, tool_slug").
Where(query).Find(&sessions).Error; err != nil {
return
}
for _, s := range sessions {
transcript, _ := w.getTranscript(ctx, s.ID)
if transcript == "" {
// No messages — just expire the session
w.db.WithContext(ctx).Table("chat_sessions").
Where("id = ?", s.ID).Update("status", "summarized")
continue
}
summary, err := w.llmOrchestrator.Summarize(ctx, transcript)
if err == nil {
w.storeSummary(ctx, s.UserID, s.ID, s.ToolSlug, summary)
log.Printf("[Summariser] Session %s (%s) compressed into memory", s.ID, s.ToolSlug)
}
}
}

func (w *SummariserWorker) getTranscript(ctx context.Context, sessionID uuid.UUID) (string, error) {
var messages []struct {
Role    string
Content string
}
w.db.WithContext(ctx).Table("chat_messages").
Where("session_id = ?", sessionID).
Order("created_at asc").
Find(&messages)
var transcript string
for _, m := range messages {
transcript += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
}
return transcript, nil
}

func (w *SummariserWorker) storeSummary(ctx context.Context, userID, sessionID uuid.UUID, toolSlug, summary string) {
if toolSlug == "" {
toolSlug = "general"
}
if err := w.db.Transaction(func(tx *gorm.DB) error {
tx.Table("session_summaries").Create(map[string]interface{}{
"id":        uuid.New(),
"user_id":   userID,
"tool_slug": toolSlug,
"summary":   summary,
})
tx.Table("chat_sessions").Where("id = ?", sessionID).Update("status", "summarized")
return nil
}); err != nil {
log.Printf("[Summariser] storeSummary transaction error: %v", err)
}
}

// CleanupOldMessages deletes raw chat_messages for sessions that have been
// summarized AND are older than 7 days. Summaries are kept indefinitely.
func (w *SummariserWorker) CleanupOldMessages(ctx context.Context) {
result := w.db.WithContext(ctx).Exec(`
DELETE FROM chat_messages
WHERE session_id IN (
SELECT id FROM chat_sessions
WHERE status = 'summarized'
  AND last_activity_at < NOW() - INTERVAL '7 days'
)
`)
if result.Error != nil {
log.Printf("[Summariser] CleanupOldMessages error: %v", result.Error)
} else if result.RowsAffected > 0 {
log.Printf("[Summariser] Retention cleanup: deleted %d old chat messages", result.RowsAffected)
}
}
