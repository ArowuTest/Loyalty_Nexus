package services

import (
	"context"
	"log"
	"time"
	"gorm.io/gorm"
)

type LifecycleWorker struct {
	db        *gorm.DB
	notifySvc *NotificationService
}

func NewLifecycleWorker(db *gorm.DB, ns *NotificationService) *LifecycleWorker {
	return &LifecycleWorker{db: db, notifySvc: ns}
}

func (w *LifecycleWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processExpirations(ctx)
			w.sendExpiryNudges(ctx)
		}
	}
}

func (w *LifecycleWorker) processExpirations(ctx context.Context) {
	// SRS Section 4.7: Assets retained for 30 days
	err := w.db.WithContext(ctx).Table("ai_generations").Where("expires_at < now()").Delete(nil).Error
	if err == nil {
		log.Printf("[Lifecycle] Expired AI assets cleaned up from database")
	}
}

func (w *LifecycleWorker) sendExpiryNudges(ctx context.Context) {
	// SRS Section 4.7: Notify via SMS 48 hours before an asset expires
	var pendingNudges []struct {
		ID       string
		MSISDN   string
		ToolName string
	}
	query := `
		SELECT g.id, u.msisdn, t.name as tool_name
		FROM ai_generations g
		JOIN users u ON g.user_id = u.id
		JOIN studio_tools t ON g.tool_id = t.id
		WHERE g.expires_at BETWEEN now() + interval '47 hours' AND now() + interval '48 hours'
		AND g.status = 'completed'
	`
	if err := w.db.WithContext(ctx).Raw(query).Scan(&pendingNudges).Error; err == nil {
		for _, nudge := range pendingNudges {
			msg := "Your Loyalty Nexus creation expires in 48 hours! Download it now from your gallery."
			w.notifySvc.SendSMS(ctx, nudge.MSISDN, msg)
		}
	}
}
