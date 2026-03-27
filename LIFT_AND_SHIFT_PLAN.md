# Lift-and-Shift Plan: RechargeMax → Loyalty Nexus
## Spin Wheel · Points · Claim · Fulfillment · Admin

_Last updated: 2026-03-26_

---

## 1. Scope

This document maps every RechargeMax component in the spin/wheel/points/claim/fulfillment/admin
stack to its Loyalty Nexus equivalent, states the current gap, and lists the exact work required.

---

## 2. Component Map

### 2.1 Database / Migrations

| RechargeMax table | LN equivalent | Gap |
|---|---|---|
| `wheel_prizes` | `prize_pool` | Missing: `is_no_win`, `no_win_message`, `color_scheme`, `sort_order`, `minimum_recharge`, `icon_name`, `terms_and_conditions`, `prize_code`, `variation_code` |
| `spin_tiers` | `spin_tiers` (migration 040) | ✅ Created in 040 — but **not seeded** with default 4 tiers |
| `spin_results` | `spin_results` | Missing: `claim_status`, `expires_at`, `momo_claim_number`, `bank_account_number`, `bank_account_name`, `bank_name`, `reviewed_by`, `reviewed_at`, `admin_notes`, `rejection_reason`, `payment_reference`, `claimed_at` (added in 041 — ✅ done) |
| `admin_spin_claims` | Embedded in `spin_results` via claim fields | ✅ Correct approach — no separate table needed |

**Actions:**
- Migration 042: Add missing `prize_pool` fields (`is_no_win`, `no_win_message`, `color_scheme`, `sort_order`, `minimum_recharge`, `icon_name`, `terms_and_conditions`, `prize_code`, `variation_code`)
- Migration 043: Seed default 4 spin tiers (matching RechargeMax defaults)
- Migration 044: Seed default 15 wheel prizes (adapted for LN prize types)

---

### 2.2 Domain Entities

| RechargeMax entity | LN equivalent | Gap |
|---|---|---|
| `WheelPrize` | `PrizePoolEntry` | Missing fields listed above |
| `SpinTierDB` (in utils) | `SpinTier` (entities/spin_tier.go) | ✅ Created — but uses `entities` package; RechargeMax uses `utils` package. Consolidate to `utils.SpinTierDB` pattern |
| `SpinResults` | `SpinResult` | Claim fields added in 041 ✅ |
| `Wallet` (RechargeMax: balance/pending/total_earned) | `Wallet` (LN: pulse_points/spin_credits/lifetime_points) | Different model — LN is correct for its domain, no change needed |

**Actions:**
- Update `PrizePoolEntry` struct with new fields
- Keep `SpinTier` in `entities` package (LN convention) but add `TableName()` method

---

### 2.3 Utils / Calculators

| RechargeMax | LN equivalent | Gap |
|---|---|---|
| `SpinTierCalculatorDB` (utils/spin_tier_calculator.go) | `SpinTierCalculatorDB` (utils/spin_tier_calculator.go) | ✅ Ported — but `ValidateTierConfiguration` is not called on Create/Update in LN's `SpinTiersService` |
| `SpinTierDB.TableName() = "spin_tiers"` | `SpinTier.TableName()` in entities | ✅ Present |

**Actions:**
- Add `ValidateTierConfiguration` call in `SpinTiersService.Create` and `Update`

---

### 2.4 Services

#### SpinService (CheckEligibility + PlaySpin)

| Feature | RechargeMax | LN | Gap |
|---|---|---|---|
| Tier-based daily cap | ✅ DB-driven via `SpinTierCalculatorDB` | ✅ Ported | None |
| Upgrade nudge in eligibility response | ✅ `NextTierName`, `AmountToNextTier` | ✅ Ported | None |
| `minimum_recharge` prize filter | ✅ `selectPrize` filters by `minimum_recharge` | ✅ Ported | None |
| `is_no_win` prize handling | ✅ No win record created for `is_no_win=true` | ❌ **Missing** — LN always creates a `spin_results` row even for try-again |
| `ClaimStatus = PENDING` on win | ✅ Set at spin time | ✅ Ported | None |
| `ExpiresAt = now + 30 days` | ✅ Set at spin time | ✅ Ported | None |
| Spin credit deduction via `gorm.Expr` | ✅ | ✅ Fixed | None |

**Actions:**
- In `PlaySpin`, when `prize.IsNoWin = true`, do NOT create a `spin_results` row; return `SpinOutcome{PrizeType: "try_again"}` without DB write

#### ClaimService (user-facing)

| Feature | RechargeMax | LN | Gap |
|---|---|---|---|
| `GetMyWins` | ✅ `ListWins` | ✅ `ListUserWins` in repo | None |
| MoMo account check on dashboard load | ✅ `VerifyMoMoAccount` | ✅ `CheckMoMoAccount` in claim_service | None |
| `ClaimPrize` for CASH/MoMo | ✅ Collects bank/MoMo details → `PENDING_ADMIN_REVIEW` | ✅ Ported | None |
| `ClaimPrize` for AIRTIME/DATA | ✅ Auto-fulfills via VTPass | ✅ Ported | None |
| `ClaimPrize` for POINTS | ✅ Auto-marks claimed | ✅ Ported | None |
| 30-day expiry guard | ✅ Rejects if `expires_at` passed | ❌ **Missing** in LN `ClaimPrize` |
| Already-claimed guard | ✅ Rejects if `claim_status != PENDING` | ✅ Ported | None |

**Actions:**
- Add expiry check in `ClaimService.ClaimPrize`: if `result.ExpiresAt != nil && time.Now().After(*result.ExpiresAt)` → return error "claim window has expired"

#### AdminClaimService

| Feature | RechargeMax | LN | Gap |
|---|---|---|---|
| `ListClaims` with status filter | ✅ | ✅ | None |
| `GetPendingClaims` | ✅ | ❌ **Missing** — no dedicated `GetPendingClaims` method or route |
| `GetClaimDetails` | ✅ | ✅ | None |
| `ApproveClaim` with MoMo disbursement | ✅ | ✅ | None |
| `RejectClaim` with reason | ✅ | ✅ | None |
| `GetStatistics` | ✅ (total/pending/approved/rejected counts + amounts) | ❌ **Missing** |
| `ExportCSV` | ✅ | ❌ **Missing** |

**Actions:**
- Add `GetPendingClaims` method and `GET /api/v1/admin/spin/claims/pending` route
- Add `GetStatistics` method and `GET /api/v1/admin/spin/claims/statistics` route
- Add `ExportCSV` method and `GET /api/v1/admin/spin/claims/export` route

#### SpinTiersService

| Feature | RechargeMax | LN | Gap |
|---|---|---|---|
| `ListAll` | ✅ | ✅ | None |
| `GetByID` | ✅ | ✅ | None |
| `Create` with overlap validation | ✅ | ✅ | None |
| `Update` with overlap validation | ✅ | ✅ | None |
| `Delete` (soft) | ✅ | ✅ | None |

---

### 2.5 Handlers / Routes

#### User-facing spin routes

| Route | RechargeMax | LN | Gap |
|---|---|---|---|
| `GET /spin/eligibility` | ✅ | ✅ | None |
| `POST /spin/play` | ✅ | ✅ | None |
| `GET /spin/wins` | ✅ `GET /winner/my-wins` | ✅ `GET /api/v1/spin/wins` | None |
| `POST /spin/wins/{id}/claim` | ✅ `POST /winner/{id}/claim` | ✅ `POST /api/v1/spin/wins/{id}/claim` | None |
| `GET /spin/wins/{id}/momo-check` | ✅ (inline in claim flow) | ✅ `GET /api/v1/spin/wins/momo-check` | None |

#### Admin spin routes

| Route | RechargeMax | LN | Gap |
|---|---|---|---|
| `GET /admin/prizes` | ✅ | ✅ | None |
| `POST /admin/prizes` | ✅ | ✅ | None |
| `PUT /admin/prizes/{id}` | ✅ | ✅ | None |
| `DELETE /admin/prizes/{id}` | ✅ | ✅ | None |
| `GET /admin/spin/config` | ✅ | ✅ | None |
| `PUT /admin/spin/config` | ✅ | ✅ | None |
| `GET /admin/spin/tiers` | ✅ | ✅ | None |
| `POST /admin/spin/tiers` | ✅ | ✅ | None |
| `PUT /admin/spin/tiers/{id}` | ✅ | ✅ | None |
| `DELETE /admin/spin/tiers/{id}` | ✅ | ✅ | None |
| `GET /admin/spin/claims` | ✅ | ✅ | None |
| `GET /admin/spin/claims/pending` | ✅ | ❌ **Missing route** |
| `GET /admin/spin/claims/{id}` | ✅ | ✅ | None |
| `POST /admin/spin/claims/{id}/approve` | ✅ | ✅ | None |
| `POST /admin/spin/claims/{id}/reject` | ✅ | ✅ | None |
| `GET /admin/spin/claims/statistics` | ✅ | ❌ **Missing route** |
| `GET /admin/spin/claims/export` | ✅ | ❌ **Missing route** |

---

## 3. Execution Order

1. **Migration 042** — Add missing `prize_pool` fields
2. **Migration 043** — Seed default 4 spin tiers
3. **Migration 044** — Seed default 15 wheel prizes (LN-adapted)
4. **Entity update** — `PrizePoolEntry` new fields
5. **Service fix** — `PlaySpin`: skip DB write for `is_no_win` prizes
6. **Service fix** — `ClaimService.ClaimPrize`: add 30-day expiry guard
7. **Service add** — `AdminClaimService.GetPendingClaims`
8. **Service add** — `AdminClaimService.GetStatistics`
9. **Service add** — `AdminClaimService.ExportCSV`
10. **Handler add** — `AdminHandler.GetPendingClaims`, `GetClaimStatistics`, `ExportClaims`
11. **Route add** — Wire new routes in `main.go`
12. **Test fix** — `setupSpinDB` seeds `spin_tiers` + recharge transaction
13. **Tests** — New tests for claim expiry, pending claims, statistics, export

---

## 4. What is already correct (no change needed)

- Tier-based daily cap with upgrade nudge in `CheckEligibility`
- `gorm.Expr` atomic wallet updates
- `safe.Go()` panic-recovering goroutines
- `ClaimStatus = PENDING` + `ExpiresAt` set at spin time
- User-facing claim endpoints (`/spin/wins`, `/spin/wins/{id}/claim`, `/spin/wins/momo-check`)
- Admin CRUD for prizes and spin tiers
- `ApproveClaim` with MoMo disbursement
- `RejectClaim` with reason
- Portable time comparisons (no Postgres-only SQL)
