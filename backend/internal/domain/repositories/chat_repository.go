package repositories

import (
	"context"
	"github.com/google/uuid"
)

type ChatRepository interface {
	GetLastSummaries(ctx context.Context, userID uuid.UUID, limit int) ([]string, error)
}
