package persistence

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
)

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

func (r *AIRoutingRepository) UpdateAttempt(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Table("ai_generation_attempts").Where("id = ?", id).Updates(fields).Error
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

func (r *AIRoutingRepository) RecordRoutingChange(ctx context.Context, entityType string, entityID uuid.UUID, action string, beforeState, afterState interface{}, changedBy string) error {
	row := map[string]interface{}{
		"id": uuid.New(), "entity_type": entityType, "entity_id": entityID, "action": action,
		"before_state": beforeState, "after_state": afterState, "changed_by": changedBy,
	}
	return r.db.WithContext(ctx).Table("ai_routing_change_log").Create(row).Error
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
