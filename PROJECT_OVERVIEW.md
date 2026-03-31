# Loyalty Nexus — Comprehensive Project Overview

**Date:** March 31, 2026  
**Author:** Manus AI

## 1. Executive Summary

**Loyalty Nexus** is a B2B2C loyalty and churn-prevention platform designed primarily for African Mobile Network Operators (MNOs), with MTN Nigeria as the primary target. The platform addresses the "multi-SIM churn" problem—where subscribers frequently switch between different network providers based on daily pricing—by transforming routine airtime recharges into a highly gamified, AI-powered loyalty experience.

The platform operates on a strict economic model with two non-convertible currencies:
1.  **Spin Credits:** Earned through cumulative recharges (e.g., ₦1,000) and used exclusively for the Spin & Win wheel.
2.  **Pulse Points:** Earned via recharges, streaks, and wheel prizes, used exclusively to fund the Nexus AI Studio.

A core architectural mandate is **Zero Hardcoding**: all business parameters (point costs, prize weights, thresholds) are stored in the `network_configs` PostgreSQL table and are dynamically adjustable via the Admin Dashboard.

## 2. Core Product Pillars

The platform is built upon four distinct pillars:

### 2.1. Digital Passport & "Ghost Nudge"
A lock-screen card integrated with Apple Wallet and Google Wallet. It displays live Pulse Points, recharge streaks, and spin progress. The critical innovation is the "Ghost Nudge"—a background cron job that pushes notifications (e.g., "Streak Expiring!") directly to the wallet card, reaching the user even if they currently have a competitor's SIM active in their device.

### 2.2. Spin & Win Engine
A gamified prize wheel utilizing a server-side cryptographically secure pseudo-random number generator (CSPRNG) for weighted prize selection. 
*   **Prizes:** Airtime/Data (provisioned via VTPass), Cash (disbursed via MTN MoMo), and bonus Pulse Points.
*   **Controls:** Enforces daily spin limits, global liability caps, and per-prize inventory caps to protect platform margins.

### 2.3. Nexus AI Studio
A suite of 17 free AI tools categorized into Chat, Create, Learn, and Build. This is the platform's primary differentiator.
*   **Tools include:** PDF Study Guides, Business Plan Generators, AI Image Generators, Voice Stories, and Language Translators.
*   **Providers:** Powered by a cascade of AI services including NotebookLM (zero API cost), Groq, Gemini Flash, DeepSeek, HuggingFace Flux, and FAL.AI.
*   **Economy:** Users "pay" for these generations using their earned Pulse Points.

### 2.4. Regional Wars
A monthly competition where the 37 Nigerian states compete based on cumulative recharge volume. Powered by Redis sorted sets for real-time leaderboards, the winning state's participants receive bonus Pulse Points.

## 3. Technical Architecture & Stack

The project is a modern, multi-client SaaS platform deployed via Docker Compose, with a clear separation of concerns.

### 3.1. Technology Stack
*   **Backend API & Workers:** Go 1.23+ utilizing the standard library `net/http` ServeMux (not Gin) and GORM.
*   **Database:** PostgreSQL 16 (with Row-Level Security and UUID primary keys).
*   **Cache & Queue:** Redis 7 (utilizing Redis Streams for asynchronous jobs).
*   **User Frontend:** Next.js 15, React 19, Tailwind CSS 3, Framer Motion (PWA).
*   **Admin Frontend:** Next.js 15, React 19, Tailwind CSS 3 (Separate application).
*   **Mobile App:** Flutter 3.x (iOS and Android).
*   **Infrastructure:** Docker, multi-stage Dockerfiles, AWS S3 for asset storage.

### 3.2. Repository Structure
The repository (`/home/ubuntu/loyalty-nexus-inflight`) is structured as a monorepo containing all platform components:

*   `/backend/`: The Go application.
    *   `/cmd/api/main.go`: The primary HTTP server entrypoint, wiring all routes, middleware, and services.
    *   `/cmd/worker/main.go`: The background worker entrypoint for cron jobs, draws, and the Ghost Nudge system.
    *   `/internal/application/services/`: Core business logic (e.g., `ai_studio_service.go` [2,500+ lines], `spin_service.go`, `draw_service.go`).
    *   `/internal/presentation/http/handlers/`: HTTP transport layer (`admin_handler.go` [2,000+ lines], `ussd_handler.go`).
    *   `/internal/domain/entities/`: GORM database models.
    *   `/internal/infrastructure/persistence/`: PostgreSQL repository implementations.
    *   `/internal/infrastructure/external/`: Adapters for third-party APIs (LLMs, Apple/Google Wallet, VTPass, MoMo).
*   `/frontend/`: The Next.js user-facing web application.
*   `/admin/`: The Next.js administrative cockpit.
*   `/mobile/`: The Flutter mobile application source code.
*   `/database/migrations/`: 73 sequential SQL migration files defining the authoritative database schema.

## 4. Current Project State & Recent Audits

The project is currently transitioning through **Phase 8 (Admin Cockpit Completion)** towards **Phase 9 (Digital Passport)** and **Phase 10 (Nexus Studio Integration)**.

### 4.1. Recent Consolidation Work
A deep, line-by-line schema and codebase audit was recently completed:
*   **Schema Alignment:** The database schema was rigorously verified against the Go entities. Previous false-positive gap reports were resolved by implementing a paren-depth-aware SQL parser that correctly read complex `ALTER TABLE` and `DO $$` blocks.
*   **Orphan Cleanup:** Migration `073_drop_orphan_tables.up.sql` was created and executed to safely drop 18 orphaned tables (remnants of deprecated features), leaving exactly 50 active, fully-utilized tables.
*   **Code Fixes:** Critical mismatches were resolved, including replacing legacy `msisdn` references with `phone_number` in raw SQL, fixing `draw_date` vs `draw_time` mismatches, and updating the `RechargeService` to use the correct daily cumulative tier logic.
*   **Frontend/Backend Alignment:** The Admin Next.js API client was overhauled to perfectly match the Go backend routes, specifically for user suspension, points adjustment, and the 5-lever recharge configuration page.

### 4.2. Build Status
*   The Go backend compiles cleanly (`go build ./...`) and passes `go vet`.
*   Both the Next.js User Frontend and Admin Frontend pass TypeScript type checking and build successfully.

## 5. Conclusion

I have fully read and understood the project documentation, the architectural mandates, the economic rules, and the exact state of the codebase. The Loyalty Nexus platform is a highly complex, well-structured Go/Next.js/Flutter application. The backend is robust, utilizing atomic database operations and strict separation of concerns. The project is currently stable, clean of technical debt regarding its database schema, and ready for the implementation of the Digital Passport and the final wiring of the AI Studio providers.
