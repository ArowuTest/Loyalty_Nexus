# Loyalty Nexus: Exhaustive Codebase & Schema Cross-Reference Report

**Author:** Manus AI
**Date:** March 31, 2026

## Executive Summary

An exhaustive, line-by-line cross-reference analysis was conducted between the Loyalty Nexus Go codebase (handlers, services, repositories, entities) and the 71 database migration files. The objective was to identify mismatches, missing columns, stale references, and active bugs where the application's expectations diverge from the authoritative database schema.

The analysis revealed several **critical mismatches** that will cause runtime panics or SQL errors, particularly in the admin handlers, fraud guard, and lifecycle workers. This report details every discrepancy found and provides a definitive consolidation plan to align the codebase with a clean baseline schema.

## 1. Critical Schema Mismatches & Active Bugs

### 1.1. `ghost_nudge_log` Table Mismatch
**Location:** `backend/internal/presentation/http/handlers/admin_passport_handler.go` (Lines 160-200)
**Issue:** The admin handler executes a raw SQL query expecting several columns that do not exist in the database.
*   **Expected by Go:** `nudge_type`, `streak_count`, `sent_at`, `delivered`
*   **Actual DB Schema (Migrations 025, 036, 060):** `id`, `user_id`, `nudged_at`, `channel`
**Impact:** The `GET /api/v1/admin/ussd/nudges` endpoint will crash with a `column does not exist` SQL error.

### 1.2. `ussd_sessions` Table Mismatch
**Location:** `backend/internal/presentation/http/handlers/admin_passport_handler.go` (Lines 200-241)
**Issue:** The admin handler queries the `ussd_sessions` table expecting legacy columns.
*   **Expected by Go:** `current_menu`, `started_at`, `last_active_at`, `is_active`, `step_count`
*   **Actual DB Schema (Migrations 025, 056, 060):** `session_id`, `phone_number`, `menu_state`, `input_buffer`, `pending_spin_id`, `expires_at`, `created_at`, `updated_at`
**Impact:** The `GET /api/v1/admin/ussd/sessions` endpoint will crash. The Go code is likely ported from an older version or different service and was not updated when the USSD schema was finalized.

### 1.3. `draws` Table: `draw_date` vs `draw_time`
**Location:** 
*   `backend/internal/application/services/lifecycle_worker.go` (Line 275)
*   `backend/internal/presentation/http/handlers/admin_handler.go` (Line 490)
**Issue:** The Go code consistently references `draw_date` in raw SQL queries and JSON payloads.
*   **Expected by Go:** `draw_date`
*   **Actual DB Schema (Migration 024):** `draw_time`
**Impact:** The `LifecycleWorker.RunScheduledDraws` cron job will fail to execute scheduled draws because it queries `WHERE status = 'SCHEDULED' AND draw_date <= ?`. This completely breaks the automated draw engine.

### 1.4. `draws` Table: Missing `draw_code`
**Location:** `backend/internal/application/services/draw_service.go` (Line 33)
**Issue:** The `DrawRecord` entity expects a `draw_code` column with a unique index. It actively generates and inserts this code during `CreateDraw`.
*   **Expected by Go:** `draw_code`
*   **Actual DB Schema:** Migration 016 included `draw_code`, but when the table was recreated/hardened in Migration 021 and 060, `draw_code` was omitted.
**Impact:** Creating a new draw via the admin panel will fail with a `column "draw_code" of relation "draws" does not exist` error.

### 1.5. `msisdn` vs `phone_number` Renaming Inconsistencies
**Location:** `backend/internal/application/services/fraud_guard.go` (Lines 22, 33, 44)
**Issue:** Migration 020 and 060 systematically renamed `msisdn` to `phone_number` across major tables (`users`, `transactions`, `msisdn_blacklist`, `auth_otps`). However, the `FraudGuard` service still uses raw SQL queries referencing `msisdn`.
*   **Expected by Go:** `msisdn` column in `msisdn_blacklist` and `transactions` tables.
*   **Actual DB Schema:** The column is now `phone_number`.
**Impact:** All fraud checks (`IsFraudulent`) will fail, potentially blocking legitimate transactions or allowing fraudulent ones if errors are swallowed.

### 1.6. Transaction Types Drift
**Location:** `backend/internal/presentation/http/handlers/admin_handler.go` (Lines 105, 907, 935)
**Issue:** The admin dashboard analytics queries use hardcoded transaction types that do not match the domain entities.
*   **Used in Admin SQL:** `'recharge_reward'`
*   **Defined in `entities.TransactionType`:** `TxTypeRecharge` (`"recharge"`), `TxTypePointsAward` (`"points_award"`)
**Impact:** Admin dashboard statistics (Total Points Issued, etc.) will return 0 or incorrect values because `'recharge_reward'` does not exist in the ledger.

### 1.7. Wallet Counters and Tier Logic (Migration 069)
**Location:** `backend/internal/application/services/recharge_service.go`
**Issue:** Migration 069 fundamentally changed how spin credits are awarded, moving from a simple `spin_draw_counter` to a daily cumulative tier-based system (`daily_recharge_kobo`, `daily_spins_awarded`).
*   **Go Code Implementation:** `recharge_service.go` still uses the old logic, directly incrementing `recharge_counter` and calculating `spinCreditsEarned := int(newCounter / spinTriggerKobo)`.
**Impact:** The application logic is out of sync with the business rules defined in the latest migrations. The DB schema has the new columns, but the Go code is not utilizing them, leading to incorrect reward distribution.

## 2. Minor Discrepancies & Technical Debt

*   **`studio_tools` Active Flag:** Migration 060 defines `is_enabled`, but Migration 067 and the Go entity use `is_active`. This needs to be standardized to `is_active` across all environments to prevent GORM mapping issues.
*   **Referral System:** Migration 068 drops the referral system (`referral_code`, `referred_by`, `total_referrals` from `users`). The Go `User` entity still contains these fields. While GORM might ignore missing columns on SELECT if not explicitly requested, any INSERT/UPDATE including these fields will fail.
*   **`draw_entries` Generated Column:** The `DrawEntry` struct has a note about `PhoneNumber` being a Postgres `GENERATED ALWAYS AS (msisdn) STORED` alias. Since `msisdn` was renamed to `phone_number` globally, this alias logic is likely broken or redundant.

## 3. Definitive Consolidation Plan

To achieve a stable, production-ready state, the following consolidation steps must be executed:

### Phase 1: Go Codebase Alignment (Immediate Fixes)
1.  **Update `admin_passport_handler.go`:**
    *   Rewrite the `GetUSSDNudges` SQL query to select `id`, `user_id`, `nudged_at`, and `channel`. Remove references to `streak_count`, `nudge_type`, etc.
    *   Rewrite the `GetUSSDSessions` SQL query to match the actual `ussd_sessions` schema (`session_id`, `phone_number`, `menu_state`, `expires_at`, `updated_at`).
2.  **Update `fraud_guard.go`:**
    *   Change all raw SQL queries to use `phone_number` instead of `msisdn` for both `msisdn_blacklist` and `transactions` tables.
3.  **Update `lifecycle_worker.go` & `admin_handler.go`:**
    *   Change `draw_date` to `draw_time` in all raw SQL queries and struct tags.
4.  **Update `admin_handler.go` Analytics:**
    *   Replace `'recharge_reward'` with `'points_award'` and `'bonus'` in the transaction type `IN` clauses to accurately reflect the ledger.

### Phase 2: Schema & Entity Synchronization
1.  **Fix `draws` Table:**
    *   Create a new migration (e.g., `072_restore_draw_code.up.sql`) to add `draw_code TEXT UNIQUE` back to the `draws` table, as the application relies on it for idempotency and referencing.
2.  **Clean up `User` Entity:**
    *   Remove `ReferralCode`, `ReferredBy`, and `TotalReferrals` from the `entities.User` struct to align with Migration 068.
3.  **Refactor `RechargeService`:**
    *   Rewrite the `processAwardTransaction` method to implement the daily cumulative tier logic introduced in Migration 069. It must update `daily_recharge_kobo` and `daily_spins_awarded` instead of the deprecated `recharge_counter`.

### Phase 3: Baseline Schema Generation
Instead of running 71 sequential migrations which have conflicting `CREATE`, `DROP`, and `ALTER` statements (e.g., creating `msisdn`, renaming to `phone_number`, dropping tables, recreating them), a single **V2 Baseline Schema** (`001_baseline_v2.up.sql`) should be generated. This baseline will represent the exact final state of the database after migration 071, providing a clean slate for production deployment and eliminating the historical cruft.

## Conclusion
The codebase contains several critical SQL mismatches due to rapid schema evolution across 71 migrations where the Go handlers (especially admin and workers) were not updated to reflect dropped or renamed columns. Implementing the Phase 1 and Phase 2 fixes will resolve all identified runtime panics and restore full system functionality.
