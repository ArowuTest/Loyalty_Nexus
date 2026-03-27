package services

// draw_service_active_draw.go
//
// Adds GetActiveDrawID to DrawService so MTNPushService can look up
// the current active draw without importing gorm directly.

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// GetActiveDrawID returns the UUID of the currently ACTIVE draw.
// Returns an error if no draw is active — callers should treat this as
// non-fatal (draw entries will be created when a draw is opened).
func (svc *DrawService) GetActiveDrawID(ctx context.Context) (uuid.UUID, error) {
	var draws []DrawRecord
	err := svc.db.WithContext(ctx).
		Where("status = 'ACTIVE'").
		Order("draw_time ASC").
		Limit(1).
		Find(&draws).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("GetActiveDrawID: db error: %w", err)
	}
	if len(draws) == 0 {
		return uuid.Nil, fmt.Errorf("no active draw found")
	}
	return draws[0].ID, nil
}
