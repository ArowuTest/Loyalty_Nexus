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
}
