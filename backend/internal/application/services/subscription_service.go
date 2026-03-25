package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"github.com/google/uuid"
)

type SubscriptionService struct {
	db *sql.DB
}

func NewSubscriptionService(db *sql.DB) *SubscriptionService {
	return &SubscriptionService{db: db}
}

func (s *SubscriptionService) Subscribe(ctx context.Context, userID uuid.UUID, planID uuid.UUID) error {
	// 1. Initial Billing (Mocking DCB/Paystack call)
	// 2. Create Subscription Record
	query := `
		INSERT INTO user_subscriptions (user_id, plan_id, next_billing_at)
		VALUES ($1, $2, $3)
	`
	_, err := s.db.ExecContext(ctx, query, userID, planID, time.Now().AddDate(0, 0, 1))
	return err
}

func (s *SubscriptionService) ProcessRecurringBilling(ctx context.Context) error {
	// Background job to bill active subscriptions and award draw entries
	// This would be called by a cron or a worker
	return nil
}
