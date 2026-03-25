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

func (r *PostgresStudioRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, outputURL string, errMsg string) error {
	_, err := r.db.ExecContext(ctx, 
		"UPDATE ai_generations SET status = $1, output_url = $2, error_message = $3, updated_at = now() WHERE id = $4",
		status, outputURL, errMsg, id)
	return err
}

func (r *PostgresStudioRepository) FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error) {
	var g entities.AIGeneration
	err := r.db.QueryRowContext(ctx, "SELECT id, user_id, tool_id, points_deducted FROM ai_generations WHERE id = $1", id).
		Scan(&g.ID, &g.UserID, &g.ToolID, &g.PointsDeducted)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *PostgresStudioRepository) GetUserGallery(ctx context.Context, userID uuid.UUID) ([]entities.AIGeneration, error) {
	rows, err := r.db.QueryContext(ctx, 
		"SELECT id, tool_id, prompt, status, output_url, created_at FROM ai_generations WHERE user_id = $1 ORDER BY created_at DESC", 
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gallery []entities.AIGeneration
	for rows.Next() {
		var g entities.AIGeneration
		if err := rows.Scan(&g.ID, &g.ToolID, &g.Prompt, &g.Status, &g.OutputURL, &g.CreatedAt); err != nil {
			return nil, err
		}
		gallery = append(gallery, g)
	}
	return gallery, nil
}
