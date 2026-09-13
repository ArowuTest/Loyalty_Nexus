package services

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"loyalty-nexus/internal/domain/entities"
)

func testCapacityController(t *testing.T) (*AICapacityController, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewAICapacityController(client), mr
}

func TestCapacityControllerAllowsUnlimitedWithoutRedis(t *testing.T) {
	c := NewAICapacityController(nil)
	release, reason, ok := c.Reserve(context.Background(), entities.AIToolProviderBinding{})
	if !ok || reason != "unlimited" {
		t.Fatalf("got ok=%v reason=%s", ok, reason)
	}
	release()
}

func TestCapacityControllerFailsClosedForGovernedLimitWithoutRedis(t *testing.T) {
	c := NewAICapacityController(nil)
	_, reason, ok := c.Reserve(context.Background(), entities.AIToolProviderBinding{MaxConcurrent: 10})
	if ok {
		t.Fatal("governed capacity must not fail open when Redis is unavailable")
	}
	if reason != "capacity_backend_unavailable" {
		t.Fatalf("unexpected reason %s", reason)
	}
}

func TestCapacityControllerEnforcesConcurrentLimit(t *testing.T) {
	c, _ := testCapacityController(t)
	b := entities.AIToolProviderBinding{
		ID: uuid.New(), MaxConcurrent: 1, TimeoutMS: 5000,
	}
	release1, reason, ok := c.Reserve(context.Background(), b)
	if !ok {
		t.Fatalf("first reserve failed: %s", reason)
	}
	_, reason, ok = c.Reserve(context.Background(), b)
	if ok || reason != "concurrency_exhausted" {
		t.Fatalf("second reserve got ok=%v reason=%s", ok, reason)
	}
	release1()
	release2, reason, ok := c.Reserve(context.Background(), b)
	if !ok {
		t.Fatalf("reserve after release failed: %s", reason)
	}
	release2()
}

func TestCircuitOpensAndAllowsSingleHalfOpenProbe(t *testing.T) {
	c, mr := testCapacityController(t)
	ctx := context.Background()
	b := entities.AIToolProviderBinding{
		ID: uuid.New(), CircuitFailureThreshold: 2, CircuitOpenSeconds: 10,
	}

	halfOpen, reason, ok := c.CircuitPermit(ctx, b)
	if !ok || halfOpen || reason != "circuit_closed" {
		t.Fatalf("initial permit got ok=%v half=%v reason=%s", ok, halfOpen, reason)
	}
	c.CircuitFailure(ctx, b, false)
	_, reason, ok = c.CircuitPermit(ctx, b)
	if !ok || reason != "circuit_closed" {
		t.Fatalf("after one failure got ok=%v reason=%s", ok, reason)
	}
	c.CircuitFailure(ctx, b, false)
	_, reason, ok = c.CircuitPermit(ctx, b)
	if ok || reason != "circuit_open" {
		t.Fatalf("expected open circuit, got ok=%v reason=%s", ok, reason)
	}

	mr.FastForward(11 * time.Second)
	halfOpen, reason, ok = c.CircuitPermit(ctx, b)
	if !ok || !halfOpen || reason != "circuit_half_open_probe" {
		t.Fatalf("expected half-open probe, got ok=%v half=%v reason=%s", ok, halfOpen, reason)
	}
	_, reason, ok = c.CircuitPermit(ctx, b)
	if ok || reason != "circuit_half_open_busy" {
		t.Fatalf("second probe should be blocked, got ok=%v reason=%s", ok, reason)
	}
	c.CircuitSuccess(ctx, b)
	halfOpen, reason, ok = c.CircuitPermit(ctx, b)
	if !ok || halfOpen || reason != "circuit_closed" {
		t.Fatalf("success should close circuit, got ok=%v half=%v reason=%s", ok, halfOpen, reason)
	}
}

func TestPaidBudgetFailsClosedWithoutRedis(t *testing.T) {
	c := NewAICapacityController(nil)
	stage := entities.AIToolStage{ID: uuid.New(), PaidHourlyBudgetMicros: 100}
	_, reason, ok := c.ReservePaidBudget(context.Background(), stage, 10)
	if ok || reason != "budget_backend_unavailable" {
		t.Fatalf("expected fail-closed budget, got ok=%v reason=%s", ok, reason)
	}
}

func TestPaidBudgetReservationAndRefund(t *testing.T) {
	c, _ := testCapacityController(t)
	ctx := context.Background()
	stage := entities.AIToolStage{
		ID: uuid.New(), PaidHourlyBudgetMicros: 100, PaidDailyBudgetMicros: 200,
	}
	first, reason, ok := c.ReservePaidBudget(ctx, stage, 60)
	if !ok {
		t.Fatalf("first budget reserve failed: %s", reason)
	}
	_, reason, ok = c.ReservePaidBudget(ctx, stage, 50)
	if ok || reason != "hourly_paid_budget_exhausted" {
		t.Fatalf("expected hourly rejection, got ok=%v reason=%s", ok, reason)
	}

	c.SettlePaidBudget(ctx, first, 0, false)
	second, reason, ok := c.ReservePaidBudget(ctx, stage, 50)
	if !ok {
		t.Fatalf("reserve after refund failed: %s", reason)
	}
	c.SettlePaidBudget(ctx, second, 40, true)

	_, reason, ok = c.ReservePaidBudget(ctx, stage, 61)
	if ok || reason != "hourly_paid_budget_exhausted" {
		t.Fatalf("expected reconciled hourly rejection, got ok=%v reason=%s", ok, reason)
	}
}
