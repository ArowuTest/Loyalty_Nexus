# Schema Alignment Fix Summary
**Date:** 2026-03-31  
**Status:** All fixes applied — `go build ./...` and `go vet ./...` pass with zero errors or warnings.

---

## Overview

A full cross-reference of all 71 migration files against every Go entity, repository, service, and handler was performed. Nine distinct categories of schema mismatch were identified and fixed. A new migration (072) was also written to close a schema gap on fresh deployments.

---

## Fixes Applied

### 1. `fraud_guard.go` — `msisdn` → `phone_number` column rename

**File:** `internal/application/services/fraud_guard.go`

| Before (broken) | After (fixed) |
|---|---|
| `WHERE msisdn = ?` on `msisdn_blacklist` | `WHERE phone_number = ?` |
| `WHERE msisdn = ?` on `transactions` | `WHERE phone_number = ?` |
| `AND is_active = true` on `msisdn_blacklist` | Removed — column does not exist (migration 060) |
| `type = 'visit'` on daily cap query | `type = 'recharge'` — correct `TxTypeRecharge` constant |

**Root cause:** Migration 020 globally renamed `msisdn` → `phone_number`. Migration 060 recreated `msisdn_blacklist` without an `is_active` column (every row is an active block). The transaction type `visit` never existed.

---

### 2. `lifecycle_worker.go` — `draw_date` → `draw_time`, `IN_PROGRESS` → `ACTIVE`

**File:** `internal/application/services/lifecycle_worker.go`

| Before (broken) | After (fixed) |
|---|---|
| `draw_date <= ?` in `RunScheduledDraws` | `draw_time <= ?` |
| `ORDER BY draw_date ASC` | `ORDER BY draw_time ASC` |
| `status = 'SCHEDULED'` filter | `status = 'UPCOMING'` |
| Transition to `IN_PROGRESS` | Transition to `ACTIVE` |
| Revert to `SCHEDULED` on failure | Revert to `UPCOMING` on failure |

**Root cause:** The `draws` table has always used `draw_time` (migration 024 `ADD COLUMN draw_time`). `draw_date` never existed. The CHECK constraint (migration 016) only permits `UPCOMING | ACTIVE | COMPLETED | CANCELLED` — `IN_PROGRESS` and `SCHEDULED` are not valid values.

---

### 3. `admin_handler.go` — `draw_date` JSON field → `draw_time`

**File:** `internal/presentation/http/handlers/admin_handler.go` — `CreateDraw` handler

| Before (broken) | After (fixed) |
|---|---|
| `DrawDate time.Time \`json:"draw_date"\`` | `DrawTime time.Time \`json:"draw_time"\`` |
| Single field, no fallback | Accepts `draw_time` (primary) with `draw_date` as deprecated alias for backwards compatibility |

---

### 4. `admin_handler.go` — `recharge_reward` → correct transaction types

**File:** `internal/presentation/http/handlers/admin_handler.go` — `GetDashboardStats`, `GetPointsStats`, `GetPointsHistory`

| Before (broken) | After (fixed) |
|---|---|
| `type IN ('recharge_reward','prize_award','bonus')` | `type IN ('points_award','prize_award','bonus','spin_credit_award','draw_entry_award')` |
| `type IN ('recharge_reward','bonus','prize_award')` | Same as above |
| `type IN ('recharge_reward','bonus','prize_award','studio_spend','admin_adjust','spin_play')` | `type IN ('recharge','points_award','bonus','prize_award','studio_spend','studio_refund','admin_adjust','spin_play','spin_credit_award','draw_entry_award')` |

**Root cause:** `recharge_reward` is not a valid transaction type. The correct types are defined in `entities.TransactionType` constants: `TxTypePointsAward = "points_award"`, `TxTypeRecharge = "recharge"`, etc.

---

### 5. `draw_service.go` — `SCHEDULED` → `UPCOMING` throughout

**File:** `internal/application/services/draw_service.go`

All 8 occurrences of `SCHEDULED` replaced with `UPCOMING` to match the DB CHECK constraint. Affected locations: `DrawRecord` struct comment, `CreateDraw`, `UpdateDraw` allowed-status map, `ExecuteDraw` fetch query, recurring draw creation, `ListUpcomingDraws`, and `GetStats`.

---

### 6. `draw_window_service.go` — `SCHEDULED` → `UPCOMING`

**File:** `internal/application/services/draw_window_service.go`

`findNextActiveDraw` query and its doc comment updated from `SCHEDULED` to `UPCOMING`.

---

### 7. `recharge_service.go` — Implement migration 069 daily cumulative tier spin logic

**File:** `internal/application/services/recharge_service.go`

**Before:** Spin credits were calculated using the old `recharge_counter` accumulator (₦1,000 per spin, simple modulo). This ignored the `daily_recharge_kobo`, `daily_recharge_date`, `daily_spins_awarded`, and `draw_counter` columns added by migration 069, and the `spin_tiers` table seeded by migration 067.

**After:** Full implementation of the migration 069 algorithm:

1. Reset `daily_recharge_kobo` and `daily_spins_awarded` if `daily_recharge_date` is not today (WAT).
2. Add `amountKobo` to `daily_recharge_kobo`.
3. Query `spin_tiers` for the tier matching the new cumulative total (with hardcoded fallback).
4. Award `tier.SpinsPerDay - daily_spins_awarded` incremental spins (never negative).
5. Draw entries continue to use `draw_counter` with the ₦200-per-entry accumulator (unchanged).
6. New helper: `calculateDailySpinCredits()` — pure, testable, WAT-timezone-aware.

**Wallet update map** now writes: `draw_counter`, `daily_recharge_kobo`, `daily_recharge_date`, `daily_spins_awarded` (and conditionally `spin_credits`). The stale `recharge_counter` field is no longer written.

---

### 8. Migration 072 — Restore `draw_code` column on fresh deployments

**File:** `database/migrations/072_restore_draw_code.up.sql` (new)  
**File:** `database/migrations/072_restore_draw_code.down.sql` (new)

**Problem:** Migration 016 created `draws` with `draw_code TEXT UNIQUE NOT NULL`. Migration 060 (`ensure_critical_tables`) re-created `draws` without `draw_code` using `CREATE TABLE IF NOT EXISTS` — meaning on a **fresh** database (where migration 060 runs first), the column is missing, causing every `DrawService.CreateDraw()` call to fail with a column-not-found error.

**Fix:** Migration 072 adds `draw_code` if absent, back-fills existing rows with `DRAW-LEGACY-{id_prefix}`, enforces `NOT NULL`, and adds a unique index. Fully idempotent.

---

### 9. User entity — Referral fields already clean

**File:** `internal/domain/entities/user.go`

Confirmed: migration 068 physically dropped `referral_code`, `referred_by`, and `total_referrals`. The entity struct correctly has no referral fields. No change needed.

Subscription columns (`subscription_tier`, `subscription_status`, `subscription_expires_at`) were **not** dropped (migration 053 only added deprecation comments). The entity correctly retains them with `json:"-"` to suppress API exposure.

---

## Verification

```
go build ./...   → 0 errors
go vet ./...     → 0 warnings
```

All remaining `msisdn` references in the codebase are either:
- The table name `"msisdn_blacklist"` (correct)
- MoMo API URL path segments (external API — not a DB column)
- CSV column header parsing in `mtn_push_csv_service.go` (external format)
- Paystack webhook payload field parsing in `recharge_handler.go` (external format)

All remaining `draw_date` references are the backwards-compatibility JSON alias in the `CreateDraw` request struct.
