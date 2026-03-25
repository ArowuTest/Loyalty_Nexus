package persistence

import (
	"context"
	"database/sql"
	"loyalty-nexus/internal/domain/entities"
	"github.com/google/uuid"
)

type PostgresStudioRepository struct {
	db *sql.DB
}

func NewPostgresStudioRepository(db *sql.DB) *PostgresStudioRepository {
	return &PostgresStudioRepository{db: db}
}

func (r *PostgresStudioRepository) ListActiveTools(ctx context.Context) ([]entities.StudioTool, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, description, category, point_cost, icon_name FROM studio_tools WHERE is_active = true")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []entities.StudioTool
	for rows.Next() {
		var t entities.StudioTool
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.PointCost, &t.Icon); err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return tools, nil
}

func (r *PostgresStudioRepository) FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error) {
	var t entities.StudioTool
	err := r.db.QueryRowContext(ctx, "SELECT id, name, category, point_cost, provider, provider_tool_id FROM studio_tools WHERE id = $1", id).
		Scan(&t.ID, &t.Name, &t.Category, &t.PointCost, &t.Provider, &t.ProviderTool)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PostgresStudioRepository) CreateGenerationTx(ctx context.Context, tx *sql.Tx, gen *entities.AIGeneration) error {
	_, err := tx.ExecContext(ctx, 
		"INSERT INTO ai_generations (id, user_id, tool_id, prompt, status, points_deducted, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		gen.ID, gen.UserID, gen.ToolID, gen.Prompt, gen.Status, gen.PointsDeducted, gen.ExpiresAt)
	return err
}
