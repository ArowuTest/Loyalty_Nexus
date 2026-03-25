package repositories

import (
	"context"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
)

type StudioRepository interface {
	ListTools(ctx context.Context) ([]entities.StudioTool, error)
	FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error)
	CreateGeneration(ctx context.Context, gen *entities.AIGeneration) error
	UpdateGeneration(ctx context.Context, gen *entities.AIGeneration) error
	FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error)
	GetUserGallery(ctx context.Context, userID uuid.UUID) ([]entities.AIGeneration, error)
}
