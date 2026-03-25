package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type postgresStudioRepository struct{ db *gorm.DB }

func NewPostgresStudioRepository(db *gorm.DB) repositories.StudioRepository {
	return &postgresStudioRepository{db: db}
}

func (r *postgresStudioRepository) ListActiveTools(ctx context.Context) ([]entities.StudioTool, error) {
	var tools []entities.StudioTool
	err := r.db.WithContext(ctx).Table("studio_tools").
		Where("is_active = true").Order("category, name").Find(&tools).Error
	return tools, err
}

func (r *postgresStudioRepository) FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error) {
	var tool entities.StudioTool
	err := r.db.WithContext(ctx).Table("studio_tools").Where("id = ?", id).First(&tool).Error
	return &tool, err
}

func (r *postgresStudioRepository) FindToolByName(ctx context.Context, name string) (*entities.StudioTool, error) {
	var tool entities.StudioTool
	err := r.db.WithContext(ctx).Table("studio_tools").Where("name = ?", name).First(&tool).Error
	return &tool, err
}

func (r *postgresStudioRepository) UpdateToolCost(ctx context.Context, toolID uuid.UUID, cost int64) error {
	return r.db.WithContext(ctx).Table("studio_tools").Where("id = ?", toolID).Update("point_cost", cost).Error
}

func (r *postgresStudioRepository) SetToolEnabled(ctx context.Context, toolID uuid.UUID, enabled bool) error {
	return r.db.WithContext(ctx).Table("studio_tools").Where("id = ?", toolID).Update("is_active", enabled).Error
}

func (r *postgresStudioRepository) CreateGenerationTx(ctx context.Context, dbTx *gorm.DB, gen *entities.AIGeneration) error {
	return dbTx.WithContext(ctx).Create(gen).Error
}

func (r *postgresStudioRepository) FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error) {
	var gen entities.AIGeneration
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&gen).Error
	return &gen, err
}

func (r *postgresStudioRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status, outputURL, errMsg string) error {
	updates := map[string]interface{}{"status": status}
	if outputURL != "" {
		updates["output_url"] = outputURL
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	return r.db.WithContext(ctx).Table("ai_generations").Where("id = ?", id).Updates(updates).Error
}

func (r *postgresStudioRepository) GetUserGallery(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.AIGeneration, error) {
	var gens []entities.AIGeneration
	r.db.WithContext(ctx).
		Where("user_id = ? AND status = 'completed' AND expires_at > NOW()", userID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&gens)
	return gens, nil
}

func (r *postgresStudioRepository) ListExpiredGenerations(ctx context.Context, limit int) ([]entities.AIGeneration, error) {
	var gens []entities.AIGeneration
	r.db.WithContext(ctx).
		Where("expires_at <= NOW()").Order("expires_at ASC").Limit(limit).Find(&gens)
	return gens, nil
}

func (r *postgresStudioRepository) DeleteGeneration(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Table("ai_generations").Where("id = ?", id).Delete(nil).Error
}

func (r *postgresStudioRepository) CountUserGenerationsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	r.db.WithContext(ctx).Table("ai_generations").
		Where("user_id = ? AND created_at >= CURRENT_DATE", userID).Count(&count)
	return int(count), nil
}
