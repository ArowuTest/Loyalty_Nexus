# LOYALTY NEXUS — Agent Handoff & Context Memory
**Version:** Phase 8 in-progress  
**Date written:** March 2026  
**Purpose:** Complete context for any agent picking up this project. Read this ENTIRE document before touching a single file.

---

## 1. WHAT THIS PRODUCT IS

**Loyalty Nexus** is a B2B2C loyalty and churn-prevention platform for African MNOs (primary target: MTN Nigeria). It solves the "multi-SIM churn" problem — Nigerian subscribers carry 2–3 SIMs and switch to whichever is cheapest that day. The platform makes MTN the "sticky" SIM by turning routine ₦1,000 airtime recharges into a gamified, AI-powered experience.

### The Four Pillars
1. **Digital Passport** — Apple Wallet / Google Wallet lock-screen card showing live Pulse Points, streak, next spin progress. The "Ghost Nudge" concept: a 60-minute cron pushes "Streak Expiring!" updates to the wallet card *regardless of which SIM is active*. The card whispers to the user even when they've switched to a competitor.
2. **Spin & Win Engine** — Controlled-probability prize wheel (crypto/rand server-side). VTPass for airtime/data provisioning. MTN MoMo API for cash prizes. Daily liability cap. 12-slot default prize table.
3. **Nexus Studio** — 17 AI tools across 4 categories (Chat/Create/Learn/Build) funded entirely by Pulse Points. The most differentiated feature — does NOT exist in either reference repo. Must be built fresh.
4. **Regional Wars** — 37 Nigerian state teams compete on cumulative recharge volume. Redis sorted sets. 24h cycle. Winning state gets bonus Pulse Points.

### The TWO Non-Negotiable Financial Rules
1. **Pulse Points ≠ Spin Credits.** They are COMPLETELY SEPARATE pools. Pulse Points fund AI Studio. Spin Credits fund the spin wheel. They NEVER convert between pools. This is the core financial safeguard.
2. **Zero Hardcoding.** Every business parameter (point costs, prize values, probability weights, chat limits, spin threshold, streak hours) lives in the `network_configs` PostgreSQL table and is editable via the Admin Dashboard WITHOUT a code deployment.

### Two Operating Modes (Single Codebase)
- **Independent Mode** (`OPERATION_MODE=independent`): Paystack for payments, VTPass for provisioning. No MNO cooperation needed. Launchable in 8 weeks.
- **Integrated Mode** (`OPERATION_MODE=integrated`): Real-time MNO BSS webhooks, deeper personalisation, SaaS licence revenue. Phase 2 (Month 6+).
- Switch via env var only, never a code fork.

---

## 2. REPOSITORY LAYOUT

```
/workspace/
├── loyalty-nexus-prev/          ← THE ACTIVE BUILD TARGET
│   ├── backend/
│   │   ├── cmd/api/main.go      ← HTTP server entry point (net/http, stdlib mux)
│   │   ├── cmd/worker/main.go   ← Background worker entry point
│   │   └── internal/
│   │       ├── application/services/   ← All business logic lives here
│   │       ├── domain/entities/        ← GORM structs
│   │       ├── domain/repositories/    ← Interfaces
│   │       ├── infrastructure/
│   │       │   ├── external/           ← AI/MoMo/VTPass adapter interfaces
│   │       │   └── persistence/        ← GORM PostgreSQL implementations
│   │       └── presentation/http/
│   │           ├── handlers/           ← HTTP handlers (net/http, NOT Gin)
│   │           └── middleware/         ← JWT auth, CORS, rate limiting
│   ├── database/migrations/     ← SQL migration files 001–023
│   ├── admin/                   ← Next.js 14 Admin Cockpit (separate app)
│   ├── frontend/                ← Next.js 14 User PWA (separate app)
│   ├── mobile/                  ← Flutter mobile app
│   ├── .env.example             ← ALL environment variables documented
│   ├── docker-compose.yml       ← Production-hardened compose
│   └── AGENT_CONTEXT.md         ← THIS FILE
│
├── RechargeMax/                 ← REFERENCE REPO 1 (Go/React, same stack)
│   └── backend/internal/        ← Gin-based, adapt to net/http for Nexus
│       ├── application/services/    ← draw_service, spin_service, winner_service etc.
│       └── presentation/handlers/   ← admin handlers to port
│
└── loyalty-saas/                ← REFERENCE REPO 2 (ReBites — Next.js/TypeScript)
    └── lib/
        ├── apple-wallet.ts      ← 688-line .pkpass generation (port to Go)
        └── google-wallet.ts     ← 522-line Google Wallet JWT (port to Go)
```

**CRITICAL**: The `loyalty-nexus-prev` backend uses **`net/http` stdlib mux**, NOT Gin. RechargeMax uses Gin. When porting handlers, convert `c *gin.Context` → `(w http.ResponseWriter, r *http.Request)`.

---

## 3. TECH STACK

| Layer | Technology | Notes |
|---|---|---|
| Backend language | Go 1.22+ | Module name: `loyalty-nexus` |
| ORM | GORM v2 | PostgreSQL driver (`gorm.io/driver/postgres`) |
| HTTP router | stdlib `net/http` ServeMux | Pattern: `"METHOD /path"` |
| Database | PostgreSQL 15 | UUID primary keys, ACID ledger |
| Cache / Realtime | Redis 7 | Sessions, leaderboard sorted sets, rate limiting |
| Event broker | NATS | Async recharge event fan-out |
| File storage | AWS S3 | AI-generated assets, 7-day pre-signed URLs |
| Frontend (user) | Next.js 14, React, TypeScript, TailwindCSS | PWA, mobile-first |
| Frontend (admin) | Next.js 14, React, TypeScript, TailwindCSS | `/admin` path, desktop |
| Mobile | Flutter 3.x | iOS + Android, 5 tabs |
| Wallet passes | Go library: `passkit-generator` | Apple .pkpass + Google Wallet JWT |
| Containers | Docker + Docker Compose | Multi-stage Dockerfiles |
| CI | GitHub Actions (`.github/workflows/ci.yml`) | go build + go test |
| Tests | 32 unit tests passing, SQLite in-memory | `CGO_ENABLED=1` required |

---

## 4. ECONOMICS (MUST UNDERSTAND BEFORE TOUCHING LEDGER CODE)

### Points Earning
| Event | Reward |
|---|---|
| Recharge ₦250 | 1 Pulse Point (default, admin-configurable) |
| Recharge ₦1,000 cumulative | 1 Spin Credit (counter resets) |
| Recharge streak milestone | Bonus points (7/14/30 days) |
| Win on spin wheel | Bonus Pulse Points (slots 6-7) |
| Regional Wars winning state | Bonus Pulse Points (cycle end) |

### Spin Wheel (12 slots default, Appendix A of SRS)
| Slot | Prize | Probability |
|---|---|---|
| 1–5 | Try Again | 20%+15%+10%+8.3%+5% = 58.3% |
| 6–7 | Pulse Points (+5, +10) | 8.3% each = 16.6% |
| 8–9 | 10MB/25MB Data | 8.3% each |
| 10 | ₦50 Airtime | 4.2% |
| 11 | ₦100 Airtime | 2.5% |
| 12 | ₦200 Airtime | 1.8% |
- Expected cost per spin: ~₦10.28. Platform earns ₦30/spin. Net margin: **65.7%**

### AI Studio Pricing (Appendix B)
| Tool | Category | Points | Provider |
|---|---|---|---|
| Ask Nexus | Chat | **0 pts** | Groq→Gemini→DeepSeek |
| Translate | Build | 1 pt | Google Translate |
| Quiz / Mind Map | Learn | 2 pts | NotebookLM (async) |
| Narrate Text / Background Remover | Create/Build | 2–3 pts | Google TTS / rembg |
| Study Guide / Deep Research | Learn | 3–5 pts | NotebookLM (async) |
| AI Photo | Create | 10 pts | HuggingFace FLUX.1 → FAL.AI fallback |
| Podcast / Slide Deck / Infographic | Learn/Build | 4 pts | NotebookLM (async) |
| Business Plan | Build | 5–6 pts | NotebookLM + Gemini |
| Animate Photo (Basic) / Video Story | Create | 65 pts | FAL.AI LTX-2-19B |
| Marketing Jingle | Create | 100 pts | Mubert API |
| Animate Photo (Premium) | Create | 250 pts | FAL.AI Kling 2.5 |
| Video + Jingle | Create | 470 pts | ElevenLabs + FAL.AI |

---

## 5. WHAT'S BEEN BUILT (Phase 1–7)

### Phase 1 — Backend Foundation ✅
- PostgreSQL migrations 001–023
- Go entities: User, Wallet, Transaction, Prize, AuthOTP, StudioGeneration, ChatSessionSummary
- Services: AuthService, RechargeService (Paystack + VTPass), SpinService (partial), FraudService, HLRService, MoMoService (stub)
- Handlers: auth, spin, user, recharge, ussd
- `docker-compose.yml`, multi-stage `Dockerfile`
- `lifecycle_worker.go` wiring subscription + draw execution

### Phase 2 — Next.js Frontend ✅
- `/frontend` — Next.js 14 PWA, 5 pages: Home, Spin, Studio, Wars, Profile
- Design system, Tailwind, state management, api client

### Phase 3 — Flutter Mobile ✅
- `/mobile` — Flutter app, 5 tabs: Home, Spin, Studio, Wars, Profile
- API client with all service classes
- Firebase push notifications
- Notifications screen

### Phase 4 — Admin Cockpit ✅
- `/admin` — Next.js 14 admin app
- Pages: dashboard, users, fraud, draws, health, notifications, subscriptions, spin-config, points-config, regional-wars, studio-tools, prizes, config

### Phase 5 — AI Studio Services ✅ (STUBS — need real provider wiring)
- `studio_service.go` — orchestration skeleton
- `studio_worker.go` (handler) — 17-tool routing skeleton
- `studio_handler.go` — HTTP handler
- External adapter interfaces: `llm_orchestrator.go`, `image_generator.go`, `knowledge_generator.go`, `document_generator.go`

### Phase 6 — Regional Wars + Draw Engine ✅
- `wars_service.go`, `wars_handler.go`
- `draw_service.go` (159 lines — STUB, needs full RechargeMax port)
- `passport_service.go` (213 lines — STUB, needs ReBites port)
- Migration 021: passport_badges, wars tables
- `lifecycle_worker.go` wired for draws + wars resolution

### Phase 7 — Notifications + FCM + Flutter live API ✅
- `notification_service.go` (260 lines) — FCM push, SMS via Termii
- `notification_handler.go` — list, mark read, push token, preferences
- `summariser_worker.go` — session summarisation skeleton
- Migration 022: push_tokens, notifications, notification_preferences, subscription_events
- Flutter: live API wiring, Firebase push, notifications screen

### Phase 8 — Admin Cockpit Completion (IN PROGRESS 🔄)
- Admin backend routes added to `main.go`: draws, spin prizes, broadcast, subscriptions, health
- `admin_handler.go` extended: GetDraws, CreateDraw, GetDrawWinners, BroadcastNotification, GetBroadcastHistory, GetSubscriptions, UpdateSubscription, GetHealth, UpdatePrizeFull
- User handler: `UpdateProfileState` added
- Migration 023: notification_broadcasts, users.state, draws.recurrence
- **Build currently PASSING** (last verified after Phase 7 commit)

---

## 6. WHAT'S MISSING / PENDING (Phases 8–10)

### Phase 8 Remaining (CURRENT WORK)
- [ ] **PORT** `draw_service.go` from RechargeMax (848 lines → full CreateDraw, GetDraws, UpdateDraw, CSV export, winner lifecycle). Our stub is only 159 lines.
- [ ] **PORT** `spin_service.go` from RechargeMax (1,426 lines → crypto/rand prize selection, VTPass provisioning with retry, CheckEligibility, daily cap, admin CRUD). Our stub is 302 lines.
- [ ] **PORT** `winner_service.go` from RechargeMax (1,515 lines → full MoMo disbursement, cash payout, goods shipping, unclaimed reminders). This is the MoMo fulfillment flow (spec §8) — NEW, doesn't exist in RechargeMax either.
- [ ] Align `network_configs` seed data with spec requirements
- [ ] Fix build errors from Phase 8 admin work
- [ ] **COMMIT Phase 8**

### Phase 9 — Digital Passport (Ghost Nudge) 
- [ ] **PORT** `apple-wallet.ts` (ReBites, 688 lines) → Go `wallet_pass_service.go`
  - Generate `.pkpass` file using `passkit-generator` Go library
  - Fields: "LOYALTY NEXUS" header, Pulse Points primary, Streak + Spin Progress secondary
  - APNs HTTP/2 push for real-time balance updates
- [ ] **PORT** `google-wallet.ts` (ReBites, 522 lines) → Go Google Wallet JWT issuer
  - Google Pay API for real-time pass updates
- [ ] **60-minute Ghost Nudge cron**: query `streak_expires_at < NOW() + 4h AND streak_count >= 3`, push updated pass
- [ ] Wire wallet-pass endpoints: `GET /api/v1/user/passport/apple`, `GET /api/v1/user/passport/google`
- [ ] New env vars: `APPLE_PASS_TYPE_ID`, `APPLE_TEAM_ID`, `APPLE_PASS_CERT_PEM`, `APPLE_PASS_KEY_PEM`, `APPLE_WWDR_PEM`, `GOOGLE_WALLET_ISSUER_ID`, `GOOGLE_WALLET_CLIENT_EMAIL`, `GOOGLE_WALLET_PRIVATE_KEY`

### Phase 10 — Nexus Studio (Net-New, No Reference Repo)
This is the most important feature. Build from scratch — neither RechargeMax nor ReBites has this.

#### 10a. Nexus Chat ("Ask Nexus") — `chat_service.go`
- **Multi-provider routing**: Groq (Llama 4 Scout, primary) → Gemini 2.5 Flash-Lite (secondary) → DeepSeek V3.2 (overflow)
- **Rate limit**: 20 messages/user/day (stored in Redis, configurable via `network_configs.nexus_chat_daily_limit`)
- **Redis session**: Full conversation history, expires after 30 min inactivity (key: `chat:session:{user_id}`)
- **Session Summarisation Worker** (`summariser_worker.go`): On Redis session expiry, send transcript to Gemini Flash-Lite → extract 3-sentence summary → store in `chat_session_summaries` table
- **Memory reconstruction**: On chat start, fetch last 3 summaries, prepend to system prompt
- **Endpoint**: `POST /api/v1/ai/chat` (0 points, just rate limit check)

#### 10b. Studio Generation — `studio_service.go` (REWRITE)
Standard generation flow (from spec §4.8):
1. Validate JWT, check `pulse_points >= tool_cost`
2. If insufficient: return HTTP 402 with "You need X more points (recharge ₦Y)"
3. Open PostgreSQL transaction: deduct `pulse_points`, insert `ai_generations` row with `status: 'pending'`
4. Dispatch to provider handler
5. For SYNC providers (FAL.AI, HuggingFace, Mubert, ElevenLabs, Google TTS/Translate, rembg, AssemblyAI):
   - Call API → upload to S3 → update `ai_generations.status = 'completed'`, `asset_url = S3_URL`
   - Return `{ asset_url, generation_id }`
6. For ASYNC providers (NotebookLM):
   - Launch goroutine, return `{ generation_id }` immediately
   - Goroutine polls → on completion → upload S3 → update record → send SMS via Termii
7. **Failure/Refund Flow**: Any error → re-credit `pulse_points`, mark `ai_generations.status = 'failed'`

Provider implementations needed:
- `HuggingFaceProvider`: POST to `https://api-inference.huggingface.co/models/black-forest-labs/FLUX.1-schnell` → if rate-limited → fallback to FAL.AI
- `FALAIProvider`: image, animate-basic, animate-premium, video-story
- `ElevenLabsProvider`: marketing-jingle, video+jingle
- `MubertProvider`: background-music  
- `NotebookLMProvider` (async): study-guide, quiz, mind-map, podcast, slide-deck, infographic, research-brief
- `GoogleTTSProvider`: narrate-text
- `GoogleTranslateProvider`: translate
- `AssemblyAIProvider`: voice-to-plan (transcription)
- `rembgProvider`: background-remover (self-hosted Python service)
- `GeminiProvider`: business-plan (+ NotebookLM combined), session summarisation

#### 10c. S3 Asset Storage
- Upload function → pre-signed URL (7-day validity)
- Gallery endpoint: `GET /api/v1/ai/gallery` → paginated list of user's `ai_generations`
- Asset expiry: 30-day retention, cron deletes expired S3 objects + records, SMS reminder 48h before

#### 10d. Admin Studio Configuration (Zero Hardcoding)
- Each tool's cost = `network_configs.ai_{tool_key}_cost_points`
- Tool enabled/disabled = `network_configs.ai_{tool_key}_enabled`
- Monthly spend cap per provider = `network_configs.ai_{provider}_monthly_spend_cap_usd`
- Chat daily limit = `network_configs.nexus_chat_daily_limit` (default: 20)

### Phase 11 — Admin Zero-Hardcoding + Production Polish
- All `network_configs` keys managed through Admin UI
- Spin wheel config API validates probability weights sum to 100%
- Admin studio tools page: enable/disable + reprice each tool live
- USSD studio integration: async NotebookLM → SMS delivery
- MTN webhook listener (Integrated Mode)
- S3 lifecycle rules, CloudFront CDN
- End-to-end tests

---

## 7. KEY DESIGN DECISIONS (DO NOT CHANGE WITHOUT JUSTIFICATION)

1. **`net/http` stdlib, not Gin** — All handlers use `(w http.ResponseWriter, r *http.Request)`. Route registration: `mux.Handle("POST /path", middleware(http.HandlerFunc(handler.Method)))`.
2. **GORM v2 with PostgreSQL** — All entities have `TableName()` methods and explicit GORM tags.
3. **SQLite for unit tests** — Tests use `gorm.io/driver/sqlite` with `CGO_ENABLED=1`. Test DB setup creates tables inline in test files, not migrations.
4. **UUID primary keys everywhere** — Using `github.com/google/uuid`.
5. **Immutable transaction ledger** — `transactions` table: INSERT only, never UPDATE or DELETE. This is a financial audit requirement.
6. **Atomic point operations** — All point deductions/awards use PostgreSQL transactions with row-level locking (`SELECT FOR UPDATE`). No application-level locking.
7. **Crypto/rand for spin outcomes** — Never `math/rand`. The frontend receives only the result; it never determines outcome.
8. **Two-pool separation enforced at DB level** — `wallets.pulse_points` and `wallets.spin_credits` are separate columns. No conversion function exists or should ever exist.
9. **config via `network_configs` table** — `db_config.go` (`ConfigManager`) reads from this table. Pattern: `cfg.GetString("key", "default")`. Values stored as JSON-serializable strings.
10. **Operation mode via env var** — `OPERATION_MODE=independent|integrated`. All mode-specific code must check this flag, never hardcode assumptions.

---

## 8. CRITICAL FILES TO UNDERSTAND

### Backend
| File | Purpose | Status |
|---|---|---|
| `cmd/api/main.go` | Wires ALL services and routes | Complete, grows each phase |
| `cmd/worker/main.go` | Wires lifecycle workers | Complete |
| `application/services/auth_service.go` | OTP/JWT auth | ✅ Complete |
| `application/services/recharge_service.go` | Paystack + VTPass + points award | ✅ 334 lines |
| `application/services/spin_service.go` | Spin engine | ⚠️ 302 lines — needs RechargeMax port |
| `application/services/draw_service.go` | Draw management | ⚠️ 159 lines — needs RechargeMax port |
| `application/services/passport_service.go` | Digital Passport generation | ⚠️ 213 lines — needs ReBites port |
| `application/services/studio_service.go` | AI Studio orchestration | ⚠️ 157 lines — stub, needs full build |
| `application/services/notification_service.go` | FCM + SMS + APNs | ✅ 260 lines |
| `application/services/wars_service.go` | Regional Wars | ✅ 198 lines |
| `application/services/fraud_service.go` | Fraud detection | ✅ 137 lines |
| `application/services/lifecycle_worker.go` | Subscription + draw + wars crons | ✅ 349 lines |
| `application/services/summariser_worker.go` | Chat session summarisation | ⚠️ 85 lines — stub |
| `infrastructure/external/llm_orchestrator.go` | LLM provider interface | ⚠️ Stub |
| `infrastructure/external/image_generator.go` | Image provider interface | ⚠️ Stub |
| `infrastructure/config/db_config.go` | ConfigManager (network_configs) | ✅ |
| `presentation/http/middleware/auth.go` | JWT middleware | ✅ |

### Reference Sources
| File | Use For |
|---|---|
| `RechargeMax/backend/internal/application/services/draw_service.go` | Port draw engine (848 lines) |
| `RechargeMax/backend/internal/application/services/spin_service.go` | Port spin engine (1,426 lines) |
| `RechargeMax/backend/internal/application/services/winner_service.go` | Port winner/fulfillment (1,515 lines) |
| `RechargeMax/backend/internal/application/services/subscription_service.go` | Subscription billing patterns |
| `RechargeMax/backend/internal/presentation/handlers/admin_spin_handler.go` | Admin prize CRUD patterns |
| `RechargeMax/backend/internal/presentation/handlers/admin_points_handler.go` | Admin points adjust/history |
| `RechargeMax/backend/internal/presentation/handlers/draw_handler.go` | Draw handler patterns |
| `loyalty-saas/lib/apple-wallet.ts` | Apple .pkpass generation logic (port to Go) |
| `loyalty-saas/lib/google-wallet.ts` | Google Wallet JWT logic (port to Go) |

---

## 9. DATABASE MIGRATIONS (Current State)

| File | Contents |
|---|---|
| 001 | network_configs seed (admin-configurable parameters) |
| 002 | Core ledger: users, wallets, transactions, spin_results |
| 003 | Nexus Studio: ai_generations, chat_sessions |
| 004 | Digital Passport: wallet_passes table |
| 005 | Regional Wars: wars_snapshots, teams |
| 006 | Daily subscriptions |
| 007 | RLS policies |
| 008 | HLR cache |
| 009 | Chat summaries: chat_session_summaries |
| 010 | Regional Wars admin |
| 011 | Auth OTP |
| 012 | User MoMo fields |
| 013 | Prize fulfillment |
| 014 | User spin credits |
| 015 | Fraud guards, fraud_events table |
| 016 | Draw engine: draws, draw_entries, draw_winners |
| 017 | Tiered earning and bonuses |
| 018 | Strategic monetization |
| 019 | User profile expansion |
| 020 | Spec alignment — comprehensive schema |
| 021 | Passport badges, wars cycle tables |
| 022 | Notifications: push_tokens, notifications, notification_preferences |
| 023 | Admin Phase 8: notification_broadcasts, users.state, draws.recurrence, prize cols |

**Next migration to write: 024** (for Phase 9 Digital Passport passkit tables, or Phase 10 Studio enhancements)

---

## 10. ENVIRONMENT VARIABLES (All in `.env.example`)

```
# Core
DATABASE_URL, REDIS_URL, JWT_SECRET, AES_256_KEY, PORT

# SMS/OTP
TERMII_API_KEY, TERMII_SENDER_ID

# Payments & Provisioning  
PAYSTACK_SECRET_KEY, PAYSTACK_WEBHOOK_SECRET
VTPASS_API_KEY, VTPASS_PUBLIC_KEY, VTPASS_SECRET_KEY, VTPASS_BASE_URL
MOMO_SUBSCRIPTION_KEY, MOMO_API_USER, MOMO_API_KEY, MOMO_ENVIRONMENT

# AI Providers
GROQ_API_KEY                    (Nexus Chat — primary)
GEMINI_API_KEY                  (Nexus Chat — secondary + session summarisation)
DEEPSEEK_API_KEY                (Nexus Chat — overflow)
HF_TOKEN                        (AI Photo — HuggingFace FLUX.1-schnell)
FAL_AI_KEY                      (AI Photo fallback, Animate, Video)
ELEVENLABS_API_KEY              (Marketing Jingle, Video+Jingle)
MUBERT_API_KEY                  (Background Music)
ASSEMBLY_AI_KEY                 (Voice transcription)
REMOVEBG_API_KEY                (Background remover — optional)

# Digital Passport (Phase 9 — NOT YET IN .env.example)
APPLE_PASS_TYPE_ID              (e.g., pass.com.loyalty-nexus.passport)
APPLE_TEAM_ID
APPLE_PASS_CERT_PEM             (base64 or path)
APPLE_PASS_KEY_PEM
APPLE_WWDR_PEM
GOOGLE_WALLET_ISSUER_ID
GOOGLE_WALLET_CLIENT_EMAIL
GOOGLE_WALLET_PRIVATE_KEY

# AWS S3 (Phase 10)
AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION, S3_BUCKET_NAME

# Platform config
OPERATION_MODE=independent|integrated
CORS_ORIGINS
```

---

## 11. API ROUTES (Current State)

### Public
```
POST /api/v1/auth/request-otp
POST /api/v1/auth/verify-otp
POST /api/v1/webhooks/paystack
POST /api/v1/webhooks/bss
POST /api/v1/ussd
```

### Authenticated User (`/api/v1/user/*`, requires Bearer JWT)
```
GET  /api/v1/user/profile
GET  /api/v1/user/wallet
GET  /api/v1/user/transactions
POST /api/v1/user/momo/request
POST /api/v1/user/momo/verify
GET  /api/v1/user/passport          ← returns Apple/Google wallet URLs
POST /api/v1/user/profile/state     ← set Nigerian state for Regional Wars

POST /api/v1/spin/play
GET  /api/v1/spin/prizes
GET  /api/v1/spin/status

GET  /api/v1/draws
GET  /api/v1/draws/{id}/winners

GET  /api/v1/wars/leaderboard
GET  /api/v1/wars/my-stats

POST /api/v1/ai/generate            ← Studio generation
GET  /api/v1/ai/gallery             ← User's gallery
GET  /api/v1/ai/status/{id}         ← Poll async generation
POST /api/v1/ai/chat                ← Ask Nexus (0 pts, 20msg/day limit)

GET  /api/v1/notifications
POST /api/v1/notifications/{id}/read
POST /api/v1/notifications/read-all
POST /api/v1/notifications/push-token
GET  /api/v1/notifications/preferences
PUT  /api/v1/notifications/preferences
```

### Admin (`/api/v1/admin/*`, requires admin JWT)
```
# Users
GET  /api/v1/admin/users
GET  /api/v1/admin/users/{id}
PUT  /api/v1/admin/users/{id}/freeze
PUT  /api/v1/admin/users/{id}/subscription

# Points
GET  /api/v1/admin/points/users
GET  /api/v1/admin/points/history
POST /api/v1/admin/points/adjust
GET  /api/v1/admin/points/statistics

# Spin / Prizes
GET  /api/v1/admin/prizes
POST /api/v1/admin/prizes
PUT  /api/v1/admin/prizes/{id}
DELETE /api/v1/admin/prizes/{id}
GET  /api/v1/admin/spin/config
PUT  /api/v1/admin/spin/config

# Draws
GET  /api/v1/admin/draws
POST /api/v1/admin/draws
PUT  /api/v1/admin/draws/{id}
POST /api/v1/admin/draws/{id}/execute
GET  /api/v1/admin/draws/{id}/winners

# Subscriptions
GET  /api/v1/admin/subscriptions

# Notifications
POST /api/v1/admin/notifications/broadcast
GET  /api/v1/admin/notifications/broadcasts

# Regional Wars
GET  /api/v1/admin/wars/leaderboard
POST /api/v1/admin/wars/cycle/reset

# Fraud
GET  /api/v1/admin/fraud/events
POST /api/v1/admin/fraud/events/{id}/resolve

# Config (network_configs table)
GET  /api/v1/admin/config
PUT  /api/v1/admin/config/{key}

# Health
GET  /api/v1/admin/health
```

---

## 12. UNIT TESTS (32 passing as of Phase 7)

```
backend/internal/application/services/
├── auth_service_test.go        (SendOTP, VerifyOTP)
├── spin_service_test.go        (WithCredits, NoCredits, DailyLimit)
├── fraud_service_test.go       (velocity checks, wallet freeze)
├── wars_service_test.go        (leaderboard, monthly resolve)
└── draw_service_test.go        (create draw, execute, winners)
```

**Test constraints:**
- Use `gorm.io/driver/sqlite` (in-memory). `CGO_ENABLED=1` required.
- No PostgreSQL-specific SQL (no `EXTRACT`, no `gen_random_uuid()` in test setup).
- Cross-DB compatible queries only in service code.
- `CREATE TABLE` statements in test setup must exactly match GORM column tags in entities.

**Run tests:**
```bash
export PATH=$PATH:/usr/local/go/bin
cd /workspace/loyalty-nexus-prev/backend
CGO_ENABLED=1 go test ./... 2>&1
```

**Build check:**
```bash
cd /workspace/loyalty-nexus-prev/backend
go build ./... 2>&1
```

---

## 13. REFERENCE REPO USAGE GUIDE

### From RechargeMax → Loyalty Nexus Port Checklist

| RechargeMax File | Port To | Adaptation Needed |
|---|---|---|
| `services/draw_service.go` (848L) | `services/draw_service.go` | Change module path `rechargemax` → `loyalty-nexus`, add `recurrence` + `next_draw_at` fields |
| `services/spin_service.go` (1426L) | `services/spin_service.go` | Add `momo_cash` hold flow (spec §8), keep credit-based model, remove subscription-based spin earning |
| `services/winner_service.go` (1515L) | New `services/winner_service.go` | Adapt MoMo disbursement to use `momo_service.go`, add Nexus-specific fields |
| `services/fraud_detection_service.go` (116L) | Merge into `services/fraud_service.go` | Add IP rate limit + AI generation velocity check (spec §11) |
| `handlers/admin_spin_handler.go` | Merge into `handlers/admin_handler.go` | Convert Gin → net/http |
| `handlers/admin_points_handler.go` | Merge into `handlers/admin_handler.go` | Convert Gin → net/http |
| `handlers/draw_handler.go` | Expand `handlers/draw_handler.go` | Convert Gin → net/http |
| `database/19_draws.sql` | Reference for draw table structure | Already implemented in migration 016 |
| `database/46_wheel_prizes.sql` | Reference for prizes table | Already in migration 002 |

### From ReBites (loyalty-saas) → Loyalty Nexus Port Checklist

| ReBites File | Port To | Notes |
|---|---|---|
| `lib/apple-wallet.ts` (688L) | Go `wallet_pass_service.go` | Use `github.com/walletpass/pass-kit` or `github.com/aolexe/passkit` Go library. Port pass.json structure, manifest, signing. |
| `lib/google-wallet.ts` (522L) | Go `wallet_pass_service.go` | Port JWT generation using `github.com/golang-jwt/jwt`. Google Wallet class/object creation. |

### From OpenLoyalty (PHP) — LOGIC ONLY, NO CODE
- Extract tier thresholds (Bronze/Silver/Gold/Platinum) as `network_configs` values
- Extract points decay/expiry mathematics
- **DO NOT copy PHP code** — implement the math in Go fresh

---

## 14. FLUTTER MOBILE APP (Current State)

**Path**: `/workspace/loyalty-nexus-prev/mobile/lib/src/`

**5 Tabs** (per spec §6.1): Home, Spin, Studio, Wars, Profile

**Key files:**
- `core/api/api_client.dart` — All API calls (AuthApi, UserApi, SpinApi, StudioApi, DrawsApi, WarsApi, NotificationsApi)
- `core/router/app_router.dart` — GoRouter routes
- `core/shell/main_shell.dart` — Bottom nav (5 tabs)
- `features/profile/presentation/profile_screen.dart` — MoMo linking, streak, passport download, state selection, notification prefs, sign-out
- `features/notifications/presentation/notifications_screen.dart` — Push notifications list

**Firebase Push**: Configured via `push_notification_service.dart`. FCM token registered to `/api/v1/notifications/push-token` on login.

---

## 15. ADMIN COCKPIT (Current State)

**Path**: `/workspace/loyalty-nexus-prev/admin/src/app/`

**Pages built** (Next.js 14 App Router):
- `dashboard/` — KPI overview
- `users/` — User management, freeze/unfreeze
- `fraud/` — Fraud alerts queue
- `draws/` — Draw scheduling, execution, winners
- `health/` — System health dashboard (REQ-5.8.3)
- `notifications/` — Broadcast composer + history
- `subscriptions/` — Subscription management
- `spin-config/` — Spin wheel visual configurator (prize table editor)
- `points-config/` — Tiered earning rates, multipliers, bonuses
- `regional-wars/` — Leaderboard + cycle control
- `studio-tools/` — AI tool enable/disable + repricing
- `prizes/` — Prize pool overview
- `config/` — network_configs live editor

**API client**: `admin/src/lib/api.ts` — `AdminAPI` class with all methods.
**Shell**: `admin/src/components/layout/AdminShell.tsx` — Sidebar nav with all sections.

---

## 16. CURRENT BUILD STATUS

**Last clean build**: After Phase 7 commit `b9f6b8d`  
**Phase 8 in-progress** — latest changes NOT yet committed:
- `admin_handler.go` — extended with 8 new admin methods
- `user_handler.go` — `UpdateProfileState` added
- `user_repo_postgres.go` — `UpdateState` implementation added
- `user_repository.go` — `UpdateState` added to interface
- `cmd/api/main.go` — New admin routes wired + `POST /api/v1/user/profile/state`
- `database/migrations/023_admin_phase8.sql` — New migration

**Potential build issue**: `UpdateProfileState` in `user_handler.go` was recently fixed. Run `go build ./...` before anything else.

---

## 17. GITHUB REPOSITORY

**URL**: `https://github.com/ArowuTest/Loyalty_Nexus`  
**Branch**: `main`  
**PAT**: Set by developer — use `git push origin main` with stored credentials  
**Commit convention**: `feat: Phase N — description` or `fix: description`

---

## 18. PHASE BUILD SEQUENCE (REMAINING)

```
Phase 8  (CURRENT)  Port draw + spin from RechargeMax. Fix build. Commit.
Phase 9  (NEXT)     Digital Passport: Apple Wallet + Google Wallet + Ghost Nudge cron.
Phase 10 (AFTER)    Nexus Studio: Chat (Groq/Gemini/DeepSeek) + All 17 tools + S3.
Phase 11 (POLISH)   Admin zero-hardcoding API, USSD studio, MTN webhook, E2E tests.
```

---

*This document was last updated during Phase 8 build. Always update the "Current Build Status" section when resuming work.*
