package services

import (
	"context"
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionService struct {
	db *gorm.DB
}

func NewSubscriptionService(db *gorm.DB) *SubscriptionService {
	return &SubscriptionService{db: db}
}

func (s *SubscriptionService) Subscribe(ctx context.Context, userID uuid.UUID, planID uuid.UUID) error {
	// 1. Initial Billing (Mocking DCB/Paystack call)
	// 2. Create Subscription Record
	return s.db.WithContext(ctx).Table("user_subscriptions").Create(map[string]interface{}{
		"user_id":         userID,
		"plan_id":         planID,
		"next_billing_at": time.Now().AddDate(0, 0, 1),
	}).Error
}

func (s *SubscriptionService) ProcessRecurringBilling(ctx context.Context) error {
	return nil
}
