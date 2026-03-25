package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StudioRepository interface {
	// Tools catalogue
	ListActiveTools(ctx context.Context) ([]entities.StudioTool, error)
	FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error)
	FindToolByName(ctx context.Context, name string) (*entities.StudioTool, error)
	UpdateToolCost(ctx context.Context, toolID uuid.UUID, newCost int64) error
	SetToolEnabled(ctx context.Context, toolID uuid.UUID, enabled bool) error

	// AI Generations
	CreateGenerationTx(ctx context.Context, dbTx *gorm.DB, gen *entities.AIGeneration) error
	FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status, outputURL, errMsg string) error
	GetUserGallery(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.AIGeneration, error)
	ListExpiredGenerations(ctx context.Context, limit int) ([]entities.AIGeneration, error)
	DeleteGeneration(ctx context.Context, id uuid.UUID) error
	CountUserGenerationsToday(ctx context.Context, userID uuid.UUID) (int, error)
}
