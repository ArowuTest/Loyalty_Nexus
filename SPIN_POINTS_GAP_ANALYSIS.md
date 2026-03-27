# Spin Wheel & Points — Gap Analysis
## Loyalty Nexus vs RechargeMax Battle-Tested Patterns

_Generated: 2026-03-26_

---

## Summary

The Loyalty Nexus spin/points implementation is architecturally sound and already
mirrors many RechargeMax patterns (GORM transactions, SELECT FOR UPDATE, CSPRNG
prize selection, immutable ledger entries, idempotency guard). The gaps below are
**hardening issues** — none are design flaws, but each is a real production risk.

---

## Gap 1 — Naked goroutines (no panic recovery)

| | RechargeMax | Loyalty Nexus |
|---|---|---|
| Background goroutine wrapper | `safe.Go(fn)` — deferred `recover()` + stack trace log | Bare `go func() { ... }()` |
| Files affected | All services | `spin_service.go`, `recharge_service.go`, `recharge_handler.go`, `prize_fulfillment_service.go`, `ussd_handler.go` |

**Risk:** A panic inside a bare goroutine crashes the entire server process.
RechargeMax's `safe.Go` recovers the panic, logs it with a stack trace, and lets
the server keep running.

**Fix:** Port `internal/pkg/safe/goroutine.go` from RechargeMax and replace all
bare `go func()` calls in service/handler files.

---

## Gap 2 — Wallet mutation uses `Save()` instead of `gorm.Expr` atomic increments

| | RechargeMax | Loyalty Nexus |
|---|---|---|
| Points credit | `Updates(map[string]interface{}{"total_points": gorm.Expr("total_points + ?", pts)})` | `wallet.PulsePoints += pts; dbTx.Save(wallet)` |
| Spin credit deduction | `UpdateColumn("spin_credits", gorm.Expr("spin_credits - 1"))` | `wallet.SpinCredits--; dbTx.Save(wallet)` |

**Risk:** The `Save()` pattern reads the wallet into memory, mutates the Go struct,
then writes the whole row back. If two concurrent requests both read `SpinCredits=1`
before either writes, both will write `SpinCredits=0` — the second deduction is
silently lost. The `SELECT FOR UPDATE` in `GetWalletForUpdate` prevents this in
Postgres, but **SQLite (used in tests) does not honour `FOR UPDATE`**, so the
test environment is not exercising the same code path as production.

`gorm.Expr("spin_credits - 1")` is a database-side atomic decrement that is safe
regardless of whether a row lock is held, and it works identically in both Postgres
and SQLite.

**Fix:** Replace `Save(wallet)` with targeted `Updates(map[string]interface{}{...})`
using `gorm.Expr` for every numeric field that can be mutated concurrently.

---

## Gap 3 — `EXTRACT(EPOCH FROM created_at)` is Postgres-only

| | RechargeMax | Loyalty Nexus |
|---|---|---|
| Date comparison | `created_at >= ?` with a `time.Time` value | `EXTRACT(EPOCH FROM created_at) >= ?` with a Unix int64 |

**Risk:** `EXTRACT(EPOCH FROM ...)` is a Postgres-specific SQL function. It will
fail silently (return 0) on SQLite, meaning `CountByUserAndType`,
`CountByPhoneAndTypeSince`, and `SumAmountByUserSince` always return 0 in tests.
This masks real bugs.

**Fix:** Pass a `time.Time` value directly to GORM's `Where("created_at >= ?", t)`.
GORM handles the dialect-specific formatting automatically for both Postgres and SQLite.

---

## Gap 4 — `DailyLiabilityTotal` uses raw SQL `CURRENT_DATE` (Postgres-only)

| | RechargeMax | Loyalty Nexus |
|---|---|---|
| Daily boundary | `time.Now().UTC().Truncate(24 * time.Hour)` passed as parameter | Raw SQL `CURRENT_DATE` |

**Risk:** `CURRENT_DATE` is a Postgres SQL keyword. In SQLite it works but uses the
server's local timezone, not UTC, causing off-by-one errors around midnight.

**Fix:** Use `time.Now().UTC().Truncate(24 * time.Hour)` as a bound parameter, same
as RechargeMax's `CheckEligibility`.

---

## Gap 5 — `CountUserSpinsToday` uses `CURRENT_DATE` (same issue)

Same as Gap 4 but in `prize_repo_postgres.go`. Fix: pass `time.Now().UTC().Truncate(24 * time.Hour)`.

---

## Gap 6 — No `safe.Go` wrapper for fulfillment goroutines

The `prize_fulfillment_service.go` and `spin_service.go` launch fulfillment in bare
goroutines. A panic in the VTPass/MoMo fulfillment path would crash the server.

---

## Gap 7 — `processAwardTransaction` calls `userRepo.UpdateWallet` which uses `Save`

`UpdateWallet` calls `db.Save(wallet)` — this is the same full-row-overwrite issue
as Gap 2. The wallet was locked with `GetWalletForUpdate`, so it is safe in Postgres
production, but the lock is not honoured in SQLite tests.

**Fix:** Replace `UpdateWallet` with a targeted `Updates` call using `gorm.Expr`
for each changed column.

---

## What is already correct (no change needed)

| Pattern | Status |
|---|---|
| `db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })` for all mutations | ✅ Correct |
| `SELECT FOR UPDATE` via `gorm/clause.Locking{Strength: "UPDATE"}` with SQLite fallback | ✅ Correct |
| CSPRNG prize selection via `crypto/rand` + `math/big` | ✅ Correct |
| Idempotency guard on recharge reference | ✅ Correct |
| Immutable ledger: separate `Transaction` rows for recharge, points award, spin credit | ✅ Correct |
| Daily liability cap check before prize selection | ✅ Correct |
| `RollbackSpin` uses `gorm.Expr("spin_credits + 1")` | ✅ Already atomic |
| Background goroutine uses `context.Background()` not request context | ✅ Correct (fixed in previous session) |

---

## Refactoring Plan

1. **Add `internal/pkg/safe/goroutine.go`** — port directly from RechargeMax
2. **Replace all bare `go func()` calls** with `safe.Go(func() { ... })`
3. **Replace `Save(wallet)` with `gorm.Expr` atomic Updates** in:
   - `recharge_service.go` → `processAwardTransaction`
   - `spin_service.go` → `PlaySpin` (credit deduction + points award)
4. **Fix `EXTRACT(EPOCH ...)` → `time.Time` parameter** in `transaction_repo_postgres.go`
5. **Fix `CURRENT_DATE` → `time.Now().UTC().Truncate(24*time.Hour)`** in:
   - `transaction_repo_postgres.go` → `DailyLiabilityTotal`
   - `prize_repo_postgres.go` → `CountUserSpinsToday`, `GetDailyInventoryUsed`
