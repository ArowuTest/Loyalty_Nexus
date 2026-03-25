package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StudioRepository interface {
	ListActiveTools(ctx context.Context) ([]entities.StudioTool, error)
	FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error)
	CreateGenerationTx(ctx context.Context, dbtx *gorm.DB, gen *entities.AIGeneration) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, outputURL string, errMsg string) error
	FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error)
	GetUserGallery(ctx context.Context, userID uuid.UUID) ([]entities.AIGeneration, error)
}
