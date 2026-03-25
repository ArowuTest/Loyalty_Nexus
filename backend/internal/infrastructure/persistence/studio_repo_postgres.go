package persistence

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresStudioRepository struct {
	db *gorm.DB
}

func NewPostgresStudioRepository(db *gorm.DB) repositories.StudioRepository {
	return &PostgresStudioRepository{db: db}
}

func (r *PostgresStudioRepository) ListActiveTools(ctx context.Context) ([]entities.StudioTool, error) {
	var tools []entities.StudioTool
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&tools).Error
	return tools, err
}

func (r *PostgresStudioRepository) FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error) {
	var tool entities.StudioTool
	if err := r.db.WithContext(ctx).First(&tool, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tool, nil
}

func (r *PostgresStudioRepository) CreateGenerationTx(ctx context.Context, dbtx *gorm.DB, gen *entities.AIGeneration) error {
	return dbtx.WithContext(ctx).Create(gen).Error
}

func (r *PostgresStudioRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, outputURL string, errMsg string) error {
	return r.db.WithContext(ctx).Model(&entities.AIGeneration{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        status,
		"output_url":    outputURL,
		"error_message": errMsg,
		"updated_at":    gorm.Expr("now()"),
	}).Error
}

func (r *PostgresStudioRepository) FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error) {
	var gen entities.AIGeneration
	if err := r.db.WithContext(ctx).First(&gen, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &gen, nil
}

func (r *PostgresStudioRepository) GetUserGallery(ctx context.Context, userID uuid.UUID) ([]entities.AIGeneration, error) {
	var gallery []entities.AIGeneration
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&gallery).Error
	return gallery, err
}
