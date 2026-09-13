package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loyalty-nexus/internal/domain/entities"
)

type AIModelIntelligenceRepository struct{ db *gorm.DB }

func NewAIModelIntelligenceRepository(db *gorm.DB) *AIModelIntelligenceRepository {
	return &AIModelIntelligenceRepository{db: db}
}

func (r *AIModelIntelligenceRepository) RefreshCatalog(ctx context.Context, source string, models []entities.AIModelCatalog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entities.AIModelCatalog{}).Where("source = ?", source).Update("is_available", false).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		for i := range models {
			models[i].Source = source
			models[i].IsAvailable = true
			models[i].LastSeenAt = now
			if models[i].DiscoveredAt.IsZero() {
				models[i].DiscoveredAt = now
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "source"}, {Name: "model_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"provider", "display_name", "is_free", "is_available", "context_window", "capabilities", "pricing", "metadata", "last_seen_at"}),
			}).Create(&models[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *AIModelIntelligenceRepository) ListAvailable(ctx context.Context, source string, freeOnly bool) ([]entities.AIModelCatalog, error) {
	var rows []entities.AIModelCatalog
	q := r.db.WithContext(ctx).Where("source = ? AND is_available = true", source)
	if freeOnly {
		q = q.Where("is_free = true")
	}
	err := q.Order("provider ASC, display_name ASC").Find(&rows).Error
	return rows, err
}

func (r *AIModelIntelligenceRepository) UpsertScores(ctx context.Context, scores []entities.AIModelScore) error {
	for i := range scores {
		if scores[i].ID == uuid.Nil {
			scores[i].ID = uuid.New()
		}
		if scores[i].EvaluatedAt.IsZero() {
			scores[i].EvaluatedAt = time.Now().UTC()
		}
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "model_catalog_id"}, {Name: "capability"}, {Name: "score_source"}},
			DoUpdates: clause.AssignmentColumns([]string{"score", "metadata", "evaluated_at"}),
		}).Create(&scores[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *AIModelIntelligenceRepository) FindBySourceModelID(ctx context.Context, source, modelID string) (*entities.AIModelCatalog, error) {
	var row entities.AIModelCatalog
	if err := r.db.WithContext(ctx).Where("source = ? AND model_id = ?", source, modelID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AIModelIntelligenceRepository) ReplacePendingRecommendations(ctx context.Context, rows []entities.AIModelRecommendation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entities.AIModelRecommendation{}).Where("status = ?", "PENDING").Update("status", "SUPERSEDED").Error; err != nil {
			return err
		}
		for i := range rows {
			if rows[i].ID == uuid.Nil {
				rows[i].ID = uuid.New()
			}
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}
func (r *AIModelIntelligenceRepository) ListScoutTargets(ctx context.Context) ([]entities.AIModelScoutTarget, error) {
	var rows []entities.AIModelScoutTarget
	err := r.db.WithContext(ctx).Table("ai_tool_stages AS s").
		Select("t.slug AS tool_slug, s.stage_key, s.capability, s.routing_policy").
		Joins("JOIN studio_tools t ON t.id = s.tool_id").
		Where("t.is_active = true AND t.is_internal = false AND s.is_active = true").
		Where("s.capability IN ?", []string{"text.generate", "text.chat", "text.vision", "text.web_search"}).
		Order("t.slug ASC, s.sort_order ASC").Scan(&rows).Error
	return rows, err
}
func (r *AIModelIntelligenceRepository) ListRecommendations(ctx context.Context, status string, limit int) ([]entities.AIModelRecommendation, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []entities.AIModelRecommendation
	q := r.db.WithContext(ctx)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *AIModelIntelligenceRepository) GetRecommendation(ctx context.Context, id uuid.UUID) (*entities.AIModelRecommendation, error) {
	var row entities.AIModelRecommendation
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AIModelIntelligenceRepository) GetCatalogByID(ctx context.Context, id uuid.UUID) (*entities.AIModelCatalog, error) {
	var row entities.AIModelCatalog
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AIModelIntelligenceRepository) SetRecommendationStatus(ctx context.Context, id uuid.UUID, status string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&entities.AIModelRecommendation{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "reviewed_at": now}).Error
}
