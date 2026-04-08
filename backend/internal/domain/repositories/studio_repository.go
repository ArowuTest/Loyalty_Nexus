package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
)

// StudioRepository is the single persistence port for all AI Studio data.
// All implementations MUST use GORM transactions where noted.
type StudioRepository interface {

	// ─── Tool catalogue ───────────────────────────────────────────────────────

	// ListActiveTools returns all tools where is_active = true, ordered by sort_order.
	ListActiveTools(ctx context.Context) ([]entities.StudioTool, error)

	// FindToolByID fetches a single tool; returns ErrNotFound if missing.
	FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error)

	// FindToolBySlug resolves a slug (e.g. "translate") to a StudioTool.
	FindToolBySlug(ctx context.Context, slug string) (*entities.StudioTool, error)

	// FindToolByName is kept for legacy compatibility (prefer FindToolBySlug).
	FindToolByName(ctx context.Context, name string) (*entities.StudioTool, error)

	// UpdateToolCost changes the point cost for a tool — triggers zero-hardcoding rule.
	UpdateToolCost(ctx context.Context, toolID uuid.UUID, newCost int64) error

	// SetToolEnabled activates or deactivates a tool globally.
	SetToolEnabled(ctx context.Context, toolID uuid.UUID, enabled bool) error

	// UpsertTool creates or replaces a tool by slug (used by seed/admin).
	UpsertTool(ctx context.Context, tool *entities.StudioTool) error

	// ─── AI Generation lifecycle ──────────────────────────────────────────────

	// CreateGenerationTx inserts a new job inside an existing DB transaction.
	// The caller MUST start and commit/rollback the transaction.
	CreateGenerationTx(ctx context.Context, dbTx *gorm.DB, gen *entities.AIGeneration) error

	// FindGenerationByID fetches a generation record by primary key.
	FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error)
	// FindGenerationBySlug fetches a website generation by vanity slug.
	FindGenerationBySlug(ctx context.Context, slug string) (*entities.AIGeneration, error)
	// SlugExists returns true if the given vanity slug is already taken.
	SlugExists(ctx context.Context, slug string) (bool, error)
	// SetVanitySlug saves the vanity slug on a generation record.
	SetVanitySlug(ctx context.Context, id uuid.UUID, slug string) error

	// UpdateStatus sets the status, output_url, error_message, and updates updated_at.
	UpdateStatus(ctx context.Context, id uuid.UUID, status, outputURL, errMsg string) error

	// CompleteGeneration persists all result fields in a single UPDATE.
	CompleteGeneration(ctx context.Context, id uuid.UUID, status, outputURL, outputURL2, outputText, provider string, costMicros, durationMs int) error

	// ─── User gallery ─────────────────────────────────────────────────────────

	// GetUserGallery returns the most recent `limit` completed generations for a user.
	GetUserGallery(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.AIGeneration, error)

	// ─── Quota ────────────────────────────────────────────────────────────────

	// CountUserGenerationsToday returns how many jobs a user has started today.
	CountUserGenerationsToday(ctx context.Context, userID uuid.UUID) (int, error)

	// ─── Lifecycle / housekeeping ─────────────────────────────────────────────

	// ListExpiredGenerations returns jobs whose expires_at has passed (for CDN purge).
	ListExpiredGenerations(ctx context.Context, limit int) ([]entities.AIGeneration, error)

	// DeleteGeneration hard-deletes a row (used after CDN asset purge).
	DeleteGeneration(ctx context.Context, id uuid.UUID) error

	// ListPendingGenerations returns jobs stuck in pending/processing state
	// older than staleSeconds seconds (for watchdog/retry logic).
	ListPendingGenerations(ctx context.Context, staleSeconds int, limit int) ([]entities.AIGeneration, error)

	// ─── Admin analytics ──────────────────────────────────────────────────────

	// GetToolErrors returns recent failed generations for a specific tool.
	GetToolErrors(ctx context.Context, toolID uuid.UUID, limit int) ([]entities.AIGeneration, error)

	// GetToolStats returns 30-day aggregated stats per tool.
	GetToolStats(ctx context.Context) ([]ToolStats, error)

	// ListGenerations returns paginated generations with optional filters.
	ListGenerations(ctx context.Context, filter GenerationFilter) ([]entities.AIGeneration, int, error)

	// ─── Session tracking ─────────────────────────────────────────────────────────

	// GetOrCreateActiveSession returns the user's current open session (ended_at IS NULL
	// and last_active_at within last 30 min), or creates a new one.
	GetOrCreateActiveSession(ctx context.Context, userID uuid.UUID) (*entities.StudioSession, error)

	// UpdateSession increments total_pts_used and generation_count, updates last_active_at.
	UpdateSession(ctx context.Context, sessionID uuid.UUID, ptsUsed int64) error

	// GetSessionUsage returns the current open session for a user (nil if none).
	GetSessionUsage(ctx context.Context, userID uuid.UUID) (*entities.StudioSession, error)

	// ─── Dispute flow ─────────────────────────────────────────────────────────────

	// DisputeGeneration marks a generation as disputed and records the refund.
	DisputeGeneration(ctx context.Context, genID uuid.UUID, refundPts int64) error
}

// ToolStats holds 30-day aggregated usage figures for a single studio tool.
type ToolStats struct {
	ToolID         string `json:"tool_id"`
	ToolSlug       string `json:"tool_slug"`
	Total          int    `json:"total"`
	Completed      int    `json:"completed"`
	Failed         int    `json:"failed"`
	PointsConsumed int64  `json:"points_consumed"`
}

// GenerationFilter carries optional predicates for listing ai_generations.
type GenerationFilter struct {
	Status   string
	ToolSlug string
	Limit    int
	Offset   int
}
