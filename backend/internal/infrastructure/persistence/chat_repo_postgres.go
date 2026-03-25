package persistence

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
)

type PostgresChatRepository struct {
	db *sql.DB
}

func NewPostgresChatRepository(db *sql.DB) *PostgresChatRepository {
	return &PostgresChatRepository{db: db}
}

func (r *PostgresChatRepository) GetLastSummaries(ctx context.Context, userID uuid.UUID, limit int) ([]string, error) {
	query := `
		SELECT summary FROM session_summaries 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}
