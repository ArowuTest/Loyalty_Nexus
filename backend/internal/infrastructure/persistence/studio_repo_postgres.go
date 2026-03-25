package persistence

// studio_repo_postgres.go — GORM implementation of repositories.StudioRepository
// All writes that mutate wallet/points go through the service layer's DB transactions;
// this repo only owns studio_tools and ai_generations tables.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
)

type postgresStudioRepository struct{ db *gorm.DB }

func NewPostgresStudioRepository(db *gorm.DB) repositories.StudioRepository {
	return &postgresStudioRepository{db: db}
}

// ─── Tool catalogue ───────────────────────────────────────────────────────────

func (r *postgresStudioRepository) ListActiveTools(ctx context.Context) ([]entities.StudioTool, error) {
	var tools []entities.StudioTool
	err := r.db.WithContext(ctx).
		Where("is_active = true").
		Order("sort_order ASC, name ASC").
		Find(&tools).Error
	return tools, err
}

func (r *postgresStudioRepository) FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error) {
	var tool entities.StudioTool
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&tool).Error
	if err != nil {
		return nil, err
	}
	return &tool, nil
}

func (r *postgresStudioRepository) FindToolBySlug(ctx context.Context, slug string) (*entities.StudioTool, error) {
	var tool entities.StudioTool
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&tool).Error
	if err != nil {
		return nil, err
	}
	return &tool, nil
}

func (r *postgresStudioRepository) FindToolByName(ctx context.Context, name string) (*entities.StudioTool, error) {
	var tool entities.StudioTool
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tool).Error
	if err != nil {
		return nil, err
	}
	return &tool, nil
}

func (r *postgresStudioRepository) UpdateToolCost(ctx context.Context, toolID uuid.UUID, newCost int64) error {
	return r.db.WithContext(ctx).
		Model(&entities.StudioTool{}).
		Where("id = ?", toolID).
		Update("point_cost", newCost).Error
}

func (r *postgresStudioRepository) SetToolEnabled(ctx context.Context, toolID uuid.UUID, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&entities.StudioTool{}).
		Where("id = ?", toolID).
		Update("is_active", enabled).Error
}

func (r *postgresStudioRepository) UpsertTool(ctx context.Context, tool *entities.StudioTool) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "description", "category", "point_cost", "provider", "provider_tool", "icon", "sort_order", "updated_at"}),
		}).
		Create(tool).Error
}

// ─── AI Generation lifecycle ──────────────────────────────────────────────────

// CreateGenerationTx inserts a new AIGeneration inside an existing GORM transaction.
// Callers MUST pass the *gorm.DB from the enclosing db.Transaction() callback.
func (r *postgresStudioRepository) CreateGenerationTx(ctx context.Context, dbTx *gorm.DB, gen *entities.AIGeneration) error {
	return dbTx.WithContext(ctx).Create(gen).Error
}

func (r *postgresStudioRepository) FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error) {
	var gen entities.AIGeneration
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&gen).Error
	if err != nil {
		return nil, err
	}
	return &gen, nil
}

// UpdateStatus is a minimal update used during the transition from pending→failed
// or pending→completed when only these three fields need persisting.
func (r *postgresStudioRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status, outputURL, errMsg string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if outputURL != "" {
		updates["output_url"] = outputURL
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	return r.db.WithContext(ctx).
		Model(&entities.AIGeneration{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// CompleteGeneration persists ALL result fields in a single UPDATE — used by
// AIStudioOrchestrator after a successful provider call.
func (r *postgresStudioRepository) CompleteGeneration(
	ctx context.Context,
	id uuid.UUID,
	status, outputURL, outputText, provider string,
	costMicros, durationMs int,
) error {
	return r.db.WithContext(ctx).
		Model(&entities.AIGeneration{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      status,
			"output_url":  outputURL,
			"output_text": outputText,
			"provider":    provider,
			"cost_micros": costMicros,
			"duration_ms": durationMs,
			"updated_at":  time.Now(),
		}).Error
}

// ─── User gallery ─────────────────────────────────────────────────────────────

func (r *postgresStudioRepository) GetUserGallery(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.AIGeneration, error) {
	var gens []entities.AIGeneration
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = 'completed' AND expires_at > NOW()", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&gens).Error
	return gens, err
}

// ─── Quota ────────────────────────────────────────────────────────────────────

func (r *postgresStudioRepository) CountUserGenerationsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.AIGeneration{}).
		Where("user_id = ? AND created_at >= CURRENT_DATE", userID).
		Count(&count).Error
	return int(count), err
}

// ─── Lifecycle / housekeeping ─────────────────────────────────────────────────

func (r *postgresStudioRepository) ListExpiredGenerations(ctx context.Context, limit int) ([]entities.AIGeneration, error) {
	var gens []entities.AIGeneration
	err := r.db.WithContext(ctx).
		Where("expires_at <= NOW() AND status = 'completed'").
		Order("expires_at ASC").
		Limit(limit).
		Find(&gens).Error
	return gens, err
}

func (r *postgresStudioRepository) DeleteGeneration(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&entities.AIGeneration{}).Error
}

func (r *postgresStudioRepository) ListPendingGenerations(ctx context.Context, staleSeconds int, limit int) ([]entities.AIGeneration, error) {
	var gens []entities.AIGeneration
	threshold := time.Now().Add(-time.Duration(staleSeconds) * time.Second)
	err := r.db.WithContext(ctx).
		Where("status IN ('pending','processing') AND created_at <= ?", threshold).
		Order("created_at ASC").
		Limit(limit).
		Find(&gens).Error
	return gens, err
}
