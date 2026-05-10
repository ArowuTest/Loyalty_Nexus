package services

// asset_expiry_worker.go — Periodic worker for asset TTL lifecycle management.
//
// Runs two background loops:
//   1. Pre-expiry notifier  — warns users before their assets expire (24h + 6h windows)
//   2. Asset expiry cleaner — marks expired rows, deletes files from R2, sends final nudge
//
// Both windows are admin-configurable via platform_settings:
//   notify_expiry_first_hours   (default 24)
//   notify_expiry_second_hours  (default 6)

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"loyalty-nexus/internal/infrastructure/external"
)

// AssetExpiryWorker manages the full lifecycle of generated assets.
type AssetExpiryWorker struct {
	db          *gorm.DB
	settingsSvc *SettingsService
	notifySvc   *NotificationService
	assetStore  external.AssetStorage // nil-safe — skip R2 deletion if not configured
	stopCh      chan struct{}
}

// NewAssetExpiryWorker creates the worker. Call Start() to begin background loops.
func NewAssetExpiryWorker(
	db *gorm.DB,
	settingsSvc *SettingsService,
	notifySvc *NotificationService,
	assetStore external.AssetStorage,
) *AssetExpiryWorker {
	return &AssetExpiryWorker{
		db:          db,
		settingsSvc: settingsSvc,
		notifySvc:   notifySvc,
		assetStore:  assetStore,
		stopCh:      make(chan struct{}),
	}
}

// Start launches the two background goroutines. Call Stop() to shut them down.
func (w *AssetExpiryWorker) Start() {
	log.Println("[AssetExpiryWorker] starting")
	go w.runPreExpiryNotifier()
	go w.runExpiryCleaner()
}

// Stop signals both goroutines to terminate.
func (w *AssetExpiryWorker) Stop() {
	close(w.stopCh)
}

// ─── Pre-expiry notifier ──────────────────────────────────────────────────────

func (w *AssetExpiryWorker) runPreExpiryNotifier() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.sendPreExpiryNotifications(context.Background())
		}
	}
}

func (w *AssetExpiryWorker) sendPreExpiryNotifications(ctx context.Context) {
	first, second := w.settingsSvc.ExpiryNotifyWindows(ctx)

	// We tag notifications using a JSON column on the generation row so we don't
	// re-notify. We use a lightweight sentinel: store notify timestamps in a
	// separate table (asset_expiry_notifications) keyed by generation_id + window.
	w.notifyWindow(ctx, first, "first")
	w.notifyWindow(ctx, second, "second")
}

func (w *AssetExpiryWorker) notifyWindow(ctx context.Context, before time.Duration, windowName string) {
	// Find generations that:
	//  • have a non-null output_url (asset was actually generated)
	//  • expire within the next (before + 5min) to (before - 5min) window
	//  • have NOT already been notified for this window
	now := time.Now()
	from := now.Add(before - 5*time.Minute)
	to := now.Add(before + 5*time.Minute)

	type row struct {
		ID        uuid.UUID `gorm:"column:id"`
		UserID    uuid.UUID `gorm:"column:user_id"`
		ToolSlug  string    `gorm:"column:tool_slug"`
		ExpiresAt time.Time `gorm:"column:expires_at"`
	}
	var gens []row
	w.db.WithContext(ctx).Raw(`
		SELECT g.id, g.user_id, g.tool_slug, g.expires_at
		FROM ai_generations g
		WHERE g.output_url IS NOT NULL
		  AND g.status = 'completed'
		  AND g.expires_at BETWEEN ? AND ?
		  AND NOT EXISTS (
		    SELECT 1 FROM asset_expiry_notifications n
		    WHERE n.generation_id = g.id AND n.window = ?
		  )
		LIMIT 500
	`, from, to, windowName).Scan(&gens)

	for _, gen := range gens {
		w.recordNotification(ctx, gen.ID, windowName)

		hoursLeft := int(time.Until(gen.ExpiresAt).Hours())
		title := "Your generation expires soon ⏰"
		body := fmt.Sprintf("Your %s result expires in ~%d hours. Download it now to keep it!", gen.ToolSlug, hoursLeft)
		if hoursLeft <= 6 {
			title = "Last chance to download! 🚨"
			body = fmt.Sprintf("Your %s result expires in about %d hours. This is your last reminder!", gen.ToolSlug, hoursLeft)
		}

		if w.notifySvc != nil {
			_ = w.notifySvc.SendToUser(ctx, gen.UserID, title, body, map[string]string{
				"type":          "asset_expiring",
				"generation_id": gen.ID.String(),
				"tool_slug":     gen.ToolSlug,
				"expires_at":    gen.ExpiresAt.Format(time.RFC3339),
			})
		}
		log.Printf("[AssetExpiryWorker] notified user=%s gen=%s window=%s hoursLeft=%d",
			gen.UserID, gen.ID, windowName, hoursLeft)
	}
}

func (w *AssetExpiryWorker) recordNotification(ctx context.Context, genID uuid.UUID, window string) {
	w.db.WithContext(ctx).Exec(
		`INSERT INTO asset_expiry_notifications (generation_id, window, sent_at)
		 VALUES (?, ?, NOW())
		 ON CONFLICT (generation_id, window) DO NOTHING`,
		genID, window,
	)
}

// ─── Asset expiry cleaner ─────────────────────────────────────────────────────

func (w *AssetExpiryWorker) runExpiryCleaner() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.cleanExpiredAssets(context.Background())
		}
	}
}

func (w *AssetExpiryWorker) cleanExpiredAssets(ctx context.Context) {
	type row struct {
		ID        uuid.UUID `gorm:"column:id"`
		UserID    uuid.UUID `gorm:"column:user_id"`
		OutputURL string    `gorm:"column:output_url"`
		ToolSlug  string    `gorm:"column:tool_slug"`
	}
	var gens []row
	w.db.WithContext(ctx).Raw(`
		SELECT id, user_id, output_url, tool_slug
		FROM ai_generations
		WHERE expires_at < NOW()
		  AND expires_at > NOW() - INTERVAL '7 days'
		  AND output_url IS NOT NULL
		  AND output_url != ''
		  AND expired_cleaned_at IS NULL
		LIMIT 200
	`).Scan(&gens)

	for _, gen := range gens {
		// 1. Delete from R2/object storage (best effort)
		if w.assetStore != nil && gen.OutputURL != "" {
			if err := w.assetStore.Delete(ctx, gen.OutputURL); err != nil {
				log.Printf("[AssetExpiryWorker] R2 delete failed gen=%s url=%s: %v", gen.ID, gen.OutputURL, err)
			}
		}

		// 2. Null out the URL + stamp cleaned timestamp
		w.db.WithContext(ctx).Exec(`
			UPDATE ai_generations
			SET output_url = NULL, output_url_2 = NULL,
			    output_text = NULL, expired_cleaned_at = NOW()
			WHERE id = ?
		`, gen.ID)

		log.Printf("[AssetExpiryWorker] cleaned expired asset gen=%s user=%s tool=%s",
			gen.ID, gen.UserID, gen.ToolSlug)
	}
	if len(gens) > 0 {
		log.Printf("[AssetExpiryWorker] cleaned %d expired assets", len(gens))
	}
}
