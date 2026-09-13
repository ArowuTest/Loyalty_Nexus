package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"loyalty-nexus/internal/domain/entities"
)

type AICapacityController struct{ rdb *redis.Client }

func NewAICapacityController(rdb *redis.Client) *AICapacityController {
	return &AICapacityController{rdb: rdb}
}

var aiReserveCapacityScript = redis.NewScript(`
local rpm_limit = tonumber(ARGV[1])
local concurrent_limit = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])
local rpm = tonumber(redis.call('GET', KEYS[1]) or '0')
local concurrent = tonumber(redis.call('GET', KEYS[2]) or '0')
if rpm_limit > 0 and rpm >= rpm_limit then return 2 end
if concurrent_limit > 0 and concurrent >= concurrent_limit then return 3 end
if rpm_limit > 0 then
  local n = redis.call('INCR', KEYS[1])
  if n == 1 then redis.call('EXPIRE', KEYS[1], 120) end
end
if concurrent_limit > 0 then
  local n = redis.call('INCR', KEYS[2])
  if n == 1 then redis.call('EXPIRE', KEYS[2], ttl) end
end
return 1
`)
var aiReleaseCapacityScript = redis.NewScript(`
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
if current <= 1 then redis.call('DEL', KEYS[1]); return 0 end
return redis.call('DECR', KEYS[1])
`)

func (c *AICapacityController) Reserve(ctx context.Context, binding entities.AIToolProviderBinding) (func(), string, bool) {
	if binding.MaxConcurrent <= 0 && binding.RequestsPerMinute <= 0 {
		return func() {}, "unlimited", true
	}
	if c == nil || c.rdb == nil {
		return func() {}, "capacity_backend_unavailable", false
	}

	minute := time.Now().UTC().Unix() / 60
	prefix := fmt.Sprintf("nexus:ai:capacity:%s", binding.ID.String())
	rpmKey := fmt.Sprintf("%s:rpm:%d", prefix, minute)
	concurrentKey := prefix + ":concurrent"
	ttl := binding.TimeoutMS/1000 + 60
	if ttl < 180 {
		ttl = 180
	}

	code, err := aiReserveCapacityScript.Run(ctx, c.rdb,
		[]string{rpmKey, concurrentKey}, binding.RequestsPerMinute, binding.MaxConcurrent, ttl).Int()
	if err != nil {
		return func() {}, "capacity_backend_error", false
	}
	switch code {
	case 2:
		return func() {}, "rpm_exhausted", false
	case 3:
		return func() {}, "concurrency_exhausted", false
	case 1:
		released := false
		return func() {
			if released || binding.MaxConcurrent <= 0 {
				return
			}
			released = true
			rctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = aiReleaseCapacityScript.Run(rctx, c.rdb, []string{concurrentKey}).Err()
		}, "reserved", true
	default:
		return func() {}, "capacity_unknown", false
	}
}

var aiCircuitBeforeScript = redis.NewScript(`
local threshold = tonumber(ARGV[1])
local probe_ttl = tonumber(ARGV[2])
if redis.call('EXISTS', KEYS[2]) == 1 then return 2 end
local failures = tonumber(redis.call('GET', KEYS[1]) or '0')
if failures >= threshold then
  local ok = redis.call('SET', KEYS[3], '1', 'NX', 'EX', probe_ttl)
  if ok then return 3 end
  return 4
end
return 1
`)

var aiCircuitFailureScript = redis.NewScript(`
local threshold = tonumber(ARGV[1])
local open_seconds = tonumber(ARGV[2])
local half_open = tonumber(ARGV[3])
local ttl = math.max(open_seconds * 10, 600)
if half_open == 1 then
  redis.call('SET', KEYS[2], '1', 'EX', open_seconds)
  redis.call('SET', KEYS[1], threshold, 'EX', ttl)
  redis.call('DEL', KEYS[3])
  return 1
end
local n = redis.call('INCR', KEYS[1])
redis.call('EXPIRE', KEYS[1], ttl)
if n >= threshold then
  redis.call('SET', KEYS[2], '1', 'EX', open_seconds)
  redis.call('DEL', KEYS[3])
  return 1
end
return 0
`)

var aiCircuitSuccessScript = redis.NewScript(`
redis.call('DEL', KEYS[1])
redis.call('DEL', KEYS[2])
redis.call('DEL', KEYS[3])
return 1
`)

func circuitKeys(bindingID string) []string {
	prefix := "nexus:ai:circuit:" + bindingID
	return []string{prefix + ":failures", prefix + ":open", prefix + ":probe"}
}

func (c *AICapacityController) CircuitPermit(ctx context.Context, binding entities.AIToolProviderBinding) (bool, string, bool) {
	if binding.CircuitFailureThreshold <= 0 || binding.CircuitOpenSeconds <= 0 {
		return false, "circuit_disabled", true
	}
	if c == nil || c.rdb == nil {
		return false, "circuit_backend_unavailable_allow", true
	}
	code, err := aiCircuitBeforeScript.Run(ctx, c.rdb, circuitKeys(binding.ID.String()),
		binding.CircuitFailureThreshold, binding.CircuitOpenSeconds).Int()
	if err != nil {
		return false, "circuit_backend_error_allow", true
	}
	switch code {
	case 1:
		return false, "circuit_closed", true
	case 2:
		return false, "circuit_open", false
	case 3:
		return true, "circuit_half_open_probe", true
	case 4:
		return false, "circuit_half_open_busy", false
	default:
		return false, "circuit_unknown", false
	}
}

func (c *AICapacityController) CircuitSuccess(ctx context.Context, binding entities.AIToolProviderBinding) {
	if binding.CircuitFailureThreshold <= 0 || binding.CircuitOpenSeconds <= 0 || c == nil || c.rdb == nil {
		return
	}
	_ = aiCircuitSuccessScript.Run(ctx, c.rdb, circuitKeys(binding.ID.String())).Err()
}

func (c *AICapacityController) CircuitFailure(ctx context.Context, binding entities.AIToolProviderBinding, halfOpen bool) {
	if binding.CircuitFailureThreshold <= 0 || binding.CircuitOpenSeconds <= 0 || c == nil || c.rdb == nil {
		return
	}
	half := 0
	if halfOpen {
		half = 1
	}
	_ = aiCircuitFailureScript.Run(ctx, c.rdb, circuitKeys(binding.ID.String()),
		binding.CircuitFailureThreshold, binding.CircuitOpenSeconds, half).Err()
}

type AIBudgetReservation struct {
	enabled  bool
	hourKey  string
	dayKey   string
	reserved int64
}

var aiBudgetReserveScript = redis.NewScript(`
local estimate = tonumber(ARGV[1])
local hour_limit = tonumber(ARGV[2])
local day_limit = tonumber(ARGV[3])
local hour_ttl = tonumber(ARGV[4])
local day_ttl = tonumber(ARGV[5])
local hour_used = tonumber(redis.call('GET', KEYS[1]) or '0')
local day_used = tonumber(redis.call('GET', KEYS[2]) or '0')
if hour_limit > 0 and hour_used + estimate > hour_limit then return 2 end
if day_limit > 0 and day_used + estimate > day_limit then return 3 end
if estimate > 0 then
  local h = redis.call('INCRBY', KEYS[1], estimate)
  if h == estimate then redis.call('EXPIRE', KEYS[1], hour_ttl) end
  local d = redis.call('INCRBY', KEYS[2], estimate)
  if d == estimate then redis.call('EXPIRE', KEYS[2], day_ttl) end
end
return 1
`)

var aiBudgetSettleScript = redis.NewScript(`
local delta = tonumber(ARGV[1])
if delta == 0 then return 1 end
local h = tonumber(redis.call('GET', KEYS[1]) or '0')
local d = tonumber(redis.call('GET', KEYS[2]) or '0')
local nh = h + delta
local nd = d + delta
if nh < 0 then nh = 0 end
if nd < 0 then nd = 0 end
redis.call('SET', KEYS[1], nh, 'KEEPTTL')
redis.call('SET', KEYS[2], nd, 'KEEPTTL')
return 1
`)

func (c *AICapacityController) ReservePaidBudget(
	ctx context.Context,
	stage entities.AIToolStage,
	estimateMicros int64,
) (AIBudgetReservation, string, bool) {
	if stage.PaidHourlyBudgetMicros <= 0 && stage.PaidDailyBudgetMicros <= 0 {
		return AIBudgetReservation{}, "budget_unlimited", true
	}
	if estimateMicros <= 0 {
		return AIBudgetReservation{}, "paid_cost_unknown", false
	}
	if c == nil || c.rdb == nil {
		return AIBudgetReservation{}, "budget_backend_unavailable", false
	}

	now := time.Now().UTC()
	prefix := "nexus:ai:budget:" + stage.ID.String()
	hourKey := fmt.Sprintf("%s:hour:%s", prefix, now.Format("2006010215"))
	dayKey := fmt.Sprintf("%s:day:%s", prefix, now.Format("20060102"))
	code, err := aiBudgetReserveScript.Run(ctx, c.rdb, []string{hourKey, dayKey},
		estimateMicros, stage.PaidHourlyBudgetMicros, stage.PaidDailyBudgetMicros,
		7200, 172800).Int()
	if err != nil {
		return AIBudgetReservation{}, "budget_backend_error", false
	}
	switch code {
	case 1:
		return AIBudgetReservation{enabled: true, hourKey: hourKey, dayKey: dayKey, reserved: estimateMicros}, "budget_reserved", true
	case 2:
		return AIBudgetReservation{}, "hourly_paid_budget_exhausted", false
	case 3:
		return AIBudgetReservation{}, "daily_paid_budget_exhausted", false
	default:
		return AIBudgetReservation{}, "budget_unknown", false
	}
}

func (c *AICapacityController) SettlePaidBudget(
	ctx context.Context,
	res AIBudgetReservation,
	actualMicros int64,
	succeeded bool,
) {
	if !res.enabled || c == nil || c.rdb == nil {
		return
	}
	target := int64(0)
	if succeeded {
		target = actualMicros
		if target < 0 {
			target = 0
		}
	}
	delta := target - res.reserved
	_ = aiBudgetSettleScript.Run(ctx, c.rdb, []string{res.hourKey, res.dayKey}, delta).Err()
}

func (c *AICapacityController) CircuitNeutral(ctx context.Context, binding entities.AIToolProviderBinding, halfOpen bool) {
	if !halfOpen || c == nil || c.rdb == nil {
		return
	}
	keys := circuitKeys(binding.ID.String())
	_ = c.rdb.Del(ctx, keys[2]).Err()
}
