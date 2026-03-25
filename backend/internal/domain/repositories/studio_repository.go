package repositories

import (
	"context"
	"database/sql"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
)

type StudioRepository interface {
	ListActiveTools(ctx context.Context) ([]entities.StudioTool, error)
	FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error)
	CreateGenerationTx(ctx context.Context, tx *sql.Tx, gen *entities.AIGeneration) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, outputURL string, errMsg string) error
	FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error)
	GetUserGallery(ctx context.Context, userID uuid.UUID) ([]entities.AIGeneration, error)
}
