package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FraudService detects suspicious activity and records fraud events.
// Implements SEC-005 through SEC-010 from the Master Spec.
type FraudService struct {
	db *gorm.DB
}

func NewFraudService(db *gorm.DB) *FraudService {
	return &FraudService{db: db}
}

type FraudEvent struct {
	ID        uuid.UUID  `gorm:"column:id;primaryKey"   json:"id"`
	UserID    uuid.UUID  `gorm:"column:user_id"         json:"user_id"`
	EventType string     `gorm:"column:event_type"      json:"event_type"`
	Severity  string     `gorm:"column:severity"        json:"severity"` // LOW|MEDIUM|HIGH|CRITICAL
	Details   string     `gorm:"column:details"         json:"details"`
	Resolved  bool       `gorm:"column:resolved"        json:"resolved"`
	CreatedAt time.Time  `gorm:"column:created_at"      json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"      json:"updated_at"`
}

func (FraudEvent) TableName() string { return "fraud_events" }

// ── Detection thresholds ───────────────────────────────────────────────────
const (
	maxRechargePer24h    = 20           // max recharge events per user per 24h
	maxSpinPer24h        = 10           // max spins per user per 24h
	minRechargeKobo      = 10_000       // ₦100 minimum legitimate recharge
	duplicateTxWindow    = 5 * time.Minute
)

// CheckRecharge analyses a recharge webhook for fraud signals.
// Call this inside recharge_service.go before committing points.
func (svc *FraudService) CheckRecharge(ctx context.Context, userID uuid.UUID, amountKobo int64, reference string) error {
	// 1 — duplicate reference within window
	var dupCount int64
	svc.db.WithContext(ctx).Table("transactions").
		Where("reference = ? AND created_at > ?", reference, time.Now().Add(-duplicateTxWindow)).
		Count(&dupCount)
	if dupCount > 0 {
		svc.record(ctx, userID, "DUPLICATE_RECHARGE", "MEDIUM",
			fmt.Sprintf("duplicate reference %s within %v", reference, duplicateTxWindow))
		return fmt.Errorf("duplicate transaction reference")
	}

	// 2 — velocity: too many recharges in 24h
	var txCount int64
	svc.db.WithContext(ctx).Table("transactions").
		Where("user_id = ? AND type = 'CREDIT' AND created_at > ?",
			userID, time.Now().Add(-24*time.Hour)).
		Count(&txCount)
	if txCount >= maxRechargePer24h {
		svc.record(ctx, userID, "RECHARGE_VELOCITY", "HIGH",
			fmt.Sprintf("%d recharges in 24h", txCount))
		return fmt.Errorf("velocity limit exceeded — account flagged for review")
	}

	// 3 — suspiciously small recharge (micro-farming)
	if amountKobo > 0 && amountKobo < minRechargeKobo {
		svc.record(ctx, userID, "MICRO_RECHARGE", "LOW",
			fmt.Sprintf("amount %d kobo below threshold", amountKobo))
		// Don't block, just log
	}
	return nil
}

// CheckSpin validates a spin attempt for abuse.
func (svc *FraudService) CheckSpin(ctx context.Context, userID uuid.UUID) error {
	var spinCount int64
	svc.db.WithContext(ctx).Table("spin_results").
		Where("user_id = ? AND created_at > ?", userID, time.Now().Add(-24*time.Hour)).
		Count(&spinCount)
	if spinCount >= maxSpinPer24h {
		svc.record(ctx, userID, "SPIN_VELOCITY", "MEDIUM",
			fmt.Sprintf("%d spins in 24h", spinCount))
		return fmt.Errorf("spin limit reached for today")
	}
	return nil
}

// ListOpenEvents returns unresolved fraud events for admin review.
func (svc *FraudService) ListOpenEvents(ctx context.Context, limit int) ([]FraudEvent, error) {
	var events []FraudEvent
	err := svc.db.WithContext(ctx).
		Where("resolved = false").
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// ResolveEvent marks a fraud event as resolved.
func (svc *FraudService) ResolveEvent(ctx context.Context, eventID uuid.UUID) error {
	return svc.db.WithContext(ctx).Model(&FraudEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"resolved":   true,
			"updated_at": time.Now(),
		}).Error
}

// SuspendUser sets is_active = false and records a SUSPENSION fraud event.
func (svc *FraudService) SuspendUser(ctx context.Context, userID uuid.UUID, reason string) error {
	if err := svc.db.WithContext(ctx).Table("users").
		Where("id = ?", userID).
		Update("is_active", false).Error; err != nil {
		return err
	}
	svc.record(ctx, userID, "MANUAL_SUSPENSION", "HIGH", reason)
	return nil
}

func (svc *FraudService) record(ctx context.Context, userID uuid.UUID, eventType, severity, details string) {
	now := time.Now()
	ev := FraudEvent{
		ID:        uuid.New(),
		UserID:    userID,
		EventType: eventType,
		Severity:  severity,
		Details:   details,
		Resolved:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	svc.db.WithContext(ctx).Create(&ev)
}
