package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FraudService detects suspicious activity and records fraud events.
// Implements SEC-005 through SEC-010 from the Master Spec.
// All detection thresholds are read from the network_configs table — zero hardcoding.
type FraudService struct {
	db *gorm.DB
}

func NewFraudService(db *gorm.DB) *FraudService {
	return &FraudService{db: db}
}

type FraudEvent struct {
	ID          uuid.UUID  `gorm:"column:id;primaryKey"   json:"id"`
	UserID      uuid.UUID  `gorm:"column:user_id"         json:"user_id"`
	PhoneNumber string     `gorm:"column:phone_number"    json:"msisdn"`
	RuleName    string     `gorm:"column:rule_name"       json:"event_type"`
	Severity    string     `gorm:"column:severity"        json:"severity"` // LOW|MEDIUM|HIGH|CRITICAL
	Details     string     `gorm:"column:details"         json:"details"`
	Resolved    bool       `gorm:"column:resolved"        json:"resolved"`
	ResolvedBy  *uuid.UUID `gorm:"column:resolved_by"     json:"resolved_by"`
	ResolvedAt  *time.Time `gorm:"column:resolved_at"     json:"resolved_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"      json:"created_at"`
}

func (FraudEvent) TableName() string { return "fraud_events" }

// ── Config helpers ─────────────────────────────────────────────────────────────
// getConfigInt reads an integer value from network_configs, falling back to defaultVal.
func (svc *FraudService) getConfigInt(ctx context.Context, key string, defaultVal int64) int64 {
	var raw string
	err := svc.db.WithContext(ctx).Table("network_configs").
		Where("key = ?", key).
		Pluck("value", &raw).Error
	if err != nil || raw == "" {
		return defaultVal
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return defaultVal
	}
	return v
}

// CheckRecharge analyses a recharge webhook for fraud signals.
// Call this inside recharge_service.go before committing points.
func (svc *FraudService) CheckRecharge(ctx context.Context, userID uuid.UUID, amountKobo int64, reference string) error {
	duplicateTxWindowSecs := svc.getConfigInt(ctx, "fraud_duplicate_tx_window_seconds", 300) // default 5 min
	duplicateTxWindow := time.Duration(duplicateTxWindowSecs) * time.Second

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
	maxRechargePer24h := svc.getConfigInt(ctx, "fraud_max_recharge_per_24h", 20)
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
	minRechargeKobo := svc.getConfigInt(ctx, "fraud_min_recharge_kobo", 10_000) // ₦100
	if amountKobo > 0 && amountKobo < minRechargeKobo {
		svc.record(ctx, userID, "MICRO_RECHARGE", "LOW",
			fmt.Sprintf("amount %d kobo below threshold %d", amountKobo, minRechargeKobo))
		// Don't block, just log
	}
	return nil
}

// CheckSpin validates a spin attempt for abuse.
func (svc *FraudService) CheckSpin(ctx context.Context, userID uuid.UUID) error {
	maxSpinPer24h := svc.getConfigInt(ctx, "fraud_max_spin_per_24h", 10)
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
	now := time.Now()
	return svc.db.WithContext(ctx).Model(&FraudEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"resolved":    true,
			"resolved_at": now,
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

func (svc *FraudService) record(ctx context.Context, userID uuid.UUID, ruleName, severity, details string) {
	now := time.Now()
	var phone string
	svc.db.WithContext(ctx).Table("users").Where("id = ?", userID).Pluck("phone_number", &phone)

	// details is JSONB in DB, so we wrap it in a JSON object
	detailsJSON := fmt.Sprintf(`{"reason": "%s"}`, details)

	ev := FraudEvent{
		ID:          uuid.New(),
		UserID:      userID,
		PhoneNumber: phone,
		RuleName:    ruleName,
		Severity:    severity,
		Details:     detailsJSON,
		Resolved:    false,
		CreatedAt:   now,
	}
	svc.db.WithContext(ctx).Create(&ev)
}
