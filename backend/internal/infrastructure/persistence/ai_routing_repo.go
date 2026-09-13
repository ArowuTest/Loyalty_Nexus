package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
)

// ErrAttemptNotFinalizable is returned when a finalize targets an attempt that is
// not in STARTED — it was never persisted, or it is already terminal. Surfacing
// it (instead of a silent 0-row update) is what lets the router log a lost
// ledger write rather than leave a phantom STARTED row (review M1/M2).
var ErrAttemptNotFinalizable = errors.New("ai attempt not finalizable: not found or already terminal")

type AIRoutingRepository struct{ db *gorm.DB }

func NewAIRoutingRepository(db *gorm.DB) *AIRoutingRepository {
	return &AIRoutingRepository{db: db}
}

func (r *AIRoutingRepository) ResolveStage(ctx context.Context, toolSlug, stageKey string) (*entities.AIToolStage, error) {
	if stageKey == "" {
		stageKey = "main"
	}
	var stage entities.AIToolStage
	err := r.db.WithContext(ctx).
		Table("ai_tool_stages AS s").
		Select("s.*").
		Joins("JOIN studio_tools t ON t.id = s.tool_id").
		Where("t.slug = ? AND t.is_active = true AND s.stage_key = ? AND s.is_active = true", toolSlug, stageKey).
		First(&stage).Error
	if err != nil {
		return nil, err
	}
	return &stage, nil
}

func (r *AIRoutingRepository) ListCandidates(ctx context.Context, toolSlug, stageKey string) ([]entities.AIRouteCandidate, error) {
	stage, err := r.ResolveStage(ctx, toolSlug, stageKey)
	if err != nil {
		return nil, err
	}

	var bindings []entities.AIToolProviderBinding
	if err := r.db.WithContext(ctx).
		Where("stage_id = ? AND is_active = true", stage.ID).
		Order("priority ASC, created_at ASC").Find(&bindings).Error; err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(bindings))
	for _, b := range bindings {
		ids = append(ids, b.ProviderID)
	}
	var providers []entities.AIProviderConfig
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND is_active = true", ids).Find(&providers).Error; err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]entities.AIProviderConfig, len(providers))
	for _, p := range providers {
		byID[p.ID] = p
	}

	out := make([]entities.AIRouteCandidate, 0, len(bindings))
	for _, b := range bindings {
		p, ok := byID[b.ProviderID]
		if !ok {
			continue
		}
		p.HasKey = p.ResolveKey() != ""
		out = append(out, entities.AIRouteCandidate{Stage: *stage, Binding: b, Provider: p})
	}
	sortRouteCandidates(stage.RoutingPolicy, out)
	return out, nil
}

func costTierRank(tier string) int {
	switch tier {
	case entities.CostTierFree:
		return 0
	case entities.CostTierLowCost:
		return 1
	case entities.CostTierPremium:
		return 2
	default:
		return 3
	}
}

func sortRouteCandidates(policy string, rows []entities.AIRouteCandidate) {
	sort.SliceStable(rows, func(i, j int) bool {
		if policy == entities.RoutingFreeFirst || policy == entities.RoutingFreeOnly {
			ri, rj := costTierRank(rows[i].Binding.CostTier), costTierRank(rows[j].Binding.CostTier)
			if ri != rj {
				return ri < rj
			}
		}
		return rows[i].Binding.Priority < rows[j].Binding.Priority
	})
}
func (r *AIRoutingRepository) ValidateToolRoute(ctx context.Context, toolSlug string) error {
	var tool entities.StudioTool
	if err := r.db.WithContext(ctx).Where("slug = ? AND is_active = true", toolSlug).First(&tool).Error; err != nil {
		return err
	}
	var stages []entities.AIToolStage
	if err := r.db.WithContext(ctx).Where("tool_id = ? AND is_active = true", tool.ID).
		Order("sort_order ASC, stage_key ASC").Find(&stages).Error; err != nil {
		return err
	}
	if len(stages) == 0 {
		return fmt.Errorf("tool %s has no active routing stages", toolSlug)
	}
	for _, stage := range stages {
		candidates, err := r.ListCandidates(ctx, toolSlug, stage.StageKey)
		if err != nil {
			return err
		}
		if stage.IsRequired && len(candidates) == 0 {
			return fmt.Errorf("tool %s required stage %s has no active providers", toolSlug, stage.StageKey)
		}
	}
	return nil
}

func (r *AIRoutingRepository) RecordAttempt(ctx context.Context, attempt *entities.AIGenerationAttempt) error {
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(attempt).Error
}

// UpdateAttempt finalizes a STARTED attempt. The ledger is append-only — a
// terminal attempt is never rewritten (migration 133 also enforces this with a
// DB trigger) — so the guard is expressed in the WHERE clause, and a 0-row
// result is reported as ErrAttemptNotFinalizable rather than swallowed.
func (r *AIRoutingRepository) UpdateAttempt(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	res := r.db.WithContext(ctx).Table("ai_generation_attempts").
		Where("id = ? AND outcome = 'STARTED'", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAttemptNotFinalizable
	}
	return nil
}

// ReconcileStrandedAttempts closes ledger rows stuck in STARTED for longer than
// olderThan (a process crash mid-call, or a finalize lost beyond the ledger
// timeout). They are marked FAILED/STRANDED so the ledger is terminal and cost
// attribution stops under-counting. STARTED → terminal is the one transition
// the append-only trigger (migration 133) permits, so terminal rows are never
// touched. Returns the number of rows closed.
func (r *AIRoutingRepository) ReconcileStrandedAttempts(ctx context.Context, olderThan time.Duration) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE ai_generation_attempts
		SET outcome = 'FAILED',
		    error_class = 'STRANDED',
		    error_message = 'attempt never finalized - reconciled by lifecycle worker',
		    completed_at = NOW()
		WHERE outcome = 'STARTED'
		  AND started_at < ?`, time.Now().Add(-olderThan))
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (r *AIRoutingRepository) ListStagesForTool(ctx context.Context, toolSlug string) ([]entities.AIToolStage, error) {
	var rows []entities.AIToolStage
	err := r.db.WithContext(ctx).Table("ai_tool_stages AS s").Select("s.*").
		Joins("JOIN studio_tools t ON t.id = s.tool_id").Where("t.slug = ?", toolSlug).
		Order("s.sort_order ASC, s.stage_key ASC").Find(&rows).Error
	return rows, err
}

func (r *AIRoutingRepository) ListBindingsForStage(ctx context.Context, stageID uuid.UUID) ([]entities.AIToolProviderBinding, error) {
	var rows []entities.AIToolProviderBinding
	err := r.db.WithContext(ctx).Where("stage_id = ?", stageID).Order("priority ASC, created_at ASC").Find(&rows).Error
	return rows, err
}

func (r *AIRoutingRepository) CreateBinding(ctx context.Context, b *entities.AIToolProviderBinding) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *AIRoutingRepository) UpdateBinding(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Table("ai_tool_provider_bindings").Where("id = ?", id).Updates(fields).Error
}

func (r *AIRoutingRepository) DeleteBinding(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entities.AIToolProviderBinding{}).Error
}

func (r *AIRoutingRepository) UpdateStage(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Table("ai_tool_stages").Where("id = ?", id).Updates(fields).Error
}

// RecordRoutingChange appends an Admin config change to ai_routing_change_log.
// before/after are JSON-serialized here and cast to jsonb explicitly, so the
// row never depends on how the driver happens to encode an arbitrary Go value,
// and a nil state — a failed post-update re-read, or any caller passing nil for
// the side a create/delete does not have — lands as JSON null rather than SQL
// NULL, which the NOT NULL columns rejected. Every caller discards the returned
// error, so a rejected write used to vanish without trace (review M3); a failed
// audit write is therefore logged HERE, where it cannot be ignored.
func (r *AIRoutingRepository) RecordRoutingChange(ctx context.Context, entityType string, entityID uuid.UUID, action string, beforeState, afterState interface{}, changedBy string) error {
	before, err := json.Marshal(beforeState)
	if err != nil {
		return fmt.Errorf("routing change-log: marshal before_state: %w", err)
	}
	after, err := json.Marshal(afterState)
	if err != nil {
		return fmt.Errorf("routing change-log: marshal after_state: %w", err)
	}
	row := map[string]interface{}{
		"id": uuid.New(), "entity_type": entityType, "entity_id": entityID, "action": action,
		"before_state": gorm.Expr("?::jsonb", string(before)),
		"after_state":  gorm.Expr("?::jsonb", string(after)),
		"changed_by":   changedBy,
	}
	if err := r.db.WithContext(ctx).Table("ai_routing_change_log").Create(row).Error; err != nil {
		log.Printf("[AIRouting] AUDIT WRITE FAILED entity=%s id=%s action=%s by=%s: %v", entityType, entityID, action, changedBy, err)
		return err
	}
	return nil
}
func (r *AIRoutingRepository) GetStage(ctx context.Context, id uuid.UUID) (*entities.AIToolStage, error) {
	var row entities.AIToolStage
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AIRoutingRepository) GetBinding(ctx context.Context, id uuid.UUID) (*entities.AIToolProviderBinding, error) {
	var row entities.AIToolProviderBinding
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
