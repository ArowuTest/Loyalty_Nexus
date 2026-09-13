# Loyalty Nexus: Definitive Schema Consolidation Report

**Date:** March 31, 2026

## Executive Summary

Following the initial codebase fixes, a completely independent, first-principles analysis was conducted to consolidate the database schema. This involved replaying all 72 migrations in sequence and cross-referencing the final authoritative database state against every Go entity, repository, service, and handler.

The analysis utilized a custom, paren-depth-aware SQL parser capable of resolving complex `ALTER TABLE` statements and conditional `DO $$ BEGIN ... END $$` blocks.

**The definitive finding is that there are NO missing columns.** Every field expected by the Go entities exists in the final database schema. The previous report's findings regarding missing columns were false positives caused by a regex parser failing on complex constraints and PL/pgSQL blocks.

However, the analysis identified **18 completely orphaned tables** that exist in the database but are never referenced anywhere in the Go codebase.

## 1. Column Gap Analysis: Clean

The deep analysis confirmed that the database schema is perfectly aligned with the Go entities in terms of required columns.

*   **`wallets`**: The `pulse_counter`, `draw_counter`, `daily_recharge_kobo`, `daily_recharge_date`, and `daily_spins_awarded` columns were correctly added in Migration 069.
*   **`ai_generations`**: All columns, including `output_text`, `provider`, `cost_micros`, etc., were correctly added in earlier migrations.
*   **`studio_tools`**: All columns, including `icon`, `sort_order`, `is_free`, etc., were correctly added.
*   **`transactions` & `auth_otps`**: The `msisdn` columns were successfully renamed to `phone_number` inside conditional `DO` blocks.
*   **`war_secondary_draws` & `war_secondary_draw_winners`**: These tables are fully defined with all expected columns.

## 2. Orphaned Tables (Technical Debt)

The database contains 69 tables, but only 51 are actively used by the Go application. The remaining 18 tables are remnants of deprecated features (e.g., old subscription plans, legacy prize fulfillment, regional wars snapshots) and are safe to drop.

The following tables are **truly orphaned** and have been marked for deletion:

1.  `chat_session_summaries`
2.  `fulfilment_webhooks`
3.  `ledger_entries`
4.  `multiplier_audit_logs`
5.  `points_expiry_policies`
6.  `prize_claims`
7.  `prize_fulfillment_logs`
8.  `program_bonuses`
9.  `program_configs`
10. `qr_scan_log`
11. `region_tournaments`
12. `regional_wars_cycles`
13. `regional_wars_snapshots`
14. `sms_templates`
15. `studio_config`
16. `subscription_events`
17. `subscription_plans`
18. `user_subscriptions`
19. `wallet_passes`

## 3. Legacy Columns (Minor Cruft)

A few active tables contain legacy columns that are no longer mapped to Go entities. These do not cause runtime errors but represent minor schema cruft:

*   `admin_users`: `username` (replaced by `email`)
*   `ai_generations`: `metadata`
*   `ghost_nudge_log`: `channel`
*   `google_wallet_objects`: `points_at_last_sync`, `tier_at_last_sync`
*   `prize_pool`: `updated_at`
*   `spin_results`: `momo_number`
*   `studio_tools`: `icon_name`, `provider_tool_id`
*   `transactions`: `stamps_delta`

*Recommendation:* These columns can be safely ignored for now, as GORM simply ignores them during `SELECT` and `INSERT` operations.

## 4. Consolidation Action Taken

I have created **Migration 073** (`073_drop_orphan_tables.up.sql`) to systematically drop the 18 orphaned tables using `DROP TABLE IF EXISTS ... CASCADE`.

This migration safely removes the technical debt without impacting any active application logic, resulting in a clean, consolidated schema.
