# Loyalty Nexus — Full Spec vs Build Audit Report
**Date:** March 2026 | **Sources:** Master Spec v2.1, SRS, Nexus Studio Key Points Discussion

---

## SUMMARY VERDICT

| Module | Status | Notes |
|---|---|---|
| Auth & OTP | ✅ Complete | |
| Recharge (Paystack + VTPass) | ✅ Complete | |
| Points / Two-Pool Ledger | ✅ Complete | |
| Spin & Win | ✅ Complete | |
| Lucky Draw | ✅ Complete | Extra — not in spec, already built |
| Regional Wars | ✅ Complete | |
| Nexus Studio (18 tools) | ✅ Complete | See pricing note below |
| AI Chat (Ask Nexus) | ✅ Complete | |
| Session Memory | ✅ Complete | |
| Passport / Wallet Passes | ⚠️ Partial | APNs live push = log-only stub |
| Badges | ✅ Complete | Built beyond spec |
| Admin Dashboard | ✅ Complete | |
| Notifications (SMS + FCM) | ✅ Complete | |
| USSD | ✅ Complete | |
| Lifecycle Worker (cron) | ✅ Complete | |
| Fraud & Security | ⚠️ Partial | 2 of 5 fraud rules not wired |
| External Adapters | ✅ Production-ready | All real HTTP calls |
| Asset Storage | ✅ Complete | S3 / GCS / local abstracted |
| WebSocket (leaderboard) | ❌ Missing | Spec Phase 3 — not yet built |

---

## SECTION 1 — CONFIRMED ✅ FULLY BUILT

### 1.1 Authentication & Users
- OTP send/verify via Termii (`POST /api/v1/auth/otp/send`, `POST /api/v1/auth/otp/verify`) ✅
- JWT middleware on all protected routes ✅
- User profile, state code, device type ✅
- MoMo number + verification flag ✅
- Subscription tier on user record ✅
- Streak counter + 36-hour configurable expiry ✅
- `is_active` freeze flag ✅
- Dual-mode config flag (standalone / MTN-integrated) ✅

### 1.2 Recharge & Provisioning
- Paystack webhook handler (`POST /api/v1/recharge/paystack-webhook`) ✅
- MNO webhook listener for Phase 2 (`POST /api/v1/recharge/mno-webhook`) ✅
- VTPass adapter: airtime + data top-up + prize fulfilment ✅
- Redis Streams event queue (NATS-compatible) ✅
- 1 Pulse Point per ₦250, 1 Spin Credit per ₦1,000 (admin-configurable) ✅
- Streak update on every qualifying recharge ✅

### 1.3 Points / Two-Pool Ledger
- `wallets` table: `pulse_points`, `spin_credits`, `lifetime_points` ✅
- `transactions` table: correct schema (`type`, `points_delta`, `amount_naira`, `provider`) ✅
- Atomic deduction (PostgreSQL transaction: wallet deduct + tx insert) ✅
- `pending` → `completed` / `failed` lifecycle ✅
- Automatic refund on API failure ✅
- `lifetime_points` never decremented ✅
- `network_configs` table for all admin-configurable parameters ✅
- HTTP 402 on insufficient points ✅

### 1.4 Spin & Win
- 12-slot configurable wheel ✅
- Daily spin limit (admin-configurable `fraud_max_spins_per_day`) ✅
- Daily prize liability cap ✅
- VTPass prize fulfilment (airtime + data) ✅
- MoMo disbursement: full happy path + retry (3× exponential backoff) ✅
- MoMo idempotency (`X-Reference-Id = spin_results.id`) ✅
- "No MoMo account" flow — 48-hour hold + SMS instruction ✅
- MoMo held expiry worker (2-hour cron) ✅
- `spin_results` table with full fulfilment state machine ✅
- Redis race-condition guard ✅

### 1.5 Regional Wars
- Monthly cycle auto-creation ✅
- Leaderboard from `transactions` (correct column: `points_delta`, type: `recharge`) ✅
- `GetLeaderboard`, `GetMyRank`, `GetHistory`, `GetWinners` endpoints ✅
- `ResolveWar` with prize pool distribution ✅
- Admin: resolve + update prize pool ✅
- State bonus awards via goroutine ✅
- Lifecycle worker: `wars-monthly-resolve` (24-hour cron) ✅

### 1.6 Nexus Studio — 18 Tools
All 18 tools seeded and routed in `ai_studio_service.go`:

| Slug | Points | Provider | Routed |
|---|---|---|---|
| translate | 1 | Google Translate → Gemini fallback | ✅ |
| study-guide | 3 | Gemini Flash | ✅ |
| quiz | 2 | Gemini Flash | ✅ |
| mindmap | 2 | Gemini Flash | ✅ |
| podcast | 4 | Gemini script + Google Cloud TTS | ✅ |
| research-brief | 5 | Gemini Flash | ✅ |
| slide-deck | 4 | Gemini Flash | ✅ |
| infographic | 5 | Gemini Flash | ✅ |
| bizplan | 12 | Gemini Flash | ✅ |
| bg-remover | 3 | rembg self-hosted → FAL BiRefNet → remove.bg | ✅ |
| narrate | 2 | Google Cloud TTS → ElevenLabs → HF Bark | ✅ |
| transcribe | 2 | AssemblyAI → Groq Whisper | ✅ |
| bg-music | 5 | Mubert → ElevenLabs sound-generation | ✅ |
| ai-photo | 10 | HF FLUX.1-schnell → FAL FLUX-dev | ✅ |
| animate-photo | 65 | FAL LTX-Video | ✅ |
| video-premium | 65 | FAL Kling v1.5 | ✅ |
| jingle | 200 | ElevenLabs Music | ✅ |
| video-jingle | 470 | FAL Kling + ElevenLabs (composite) | ✅ |

Studio endpoints:
- `GET /api/v1/studio/tools` ✅
- `POST /api/v1/studio/generate` ✅
- `GET /api/v1/studio/generate/{id}` (async polling) ✅
- `GET /api/v1/studio/gallery` ✅
- Stale-job recovery worker (10-min cron) ✅
- Point refund on failure ✅
- Pre-signed URL generation ✅

### 1.7 AI Chat ("Ask Nexus")
- `POST /api/v1/studio/chat` ✅
- `GET /api/v1/studio/chat/usage` ✅
- **FREE** (0 points) ✅
- Daily limit: 20 messages per user (admin-configurable `chat_daily_message_limit`) ✅
- Provider chain: Groq Llama-4-Scout → Gemini Flash-Lite → DeepSeek V3 ✅
- Daily limits per provider (admin-configurable) ✅
- Groq daily cap: 1,000 req / Gemini daily cap: 1,000 req ✅

### 1.8 Session Memory & Summarisation
- Redis session storage with 30-minute inactivity TTL ✅
- Session summarisation worker (10-min cron) ✅
- Gemini Flash-Lite summarisation with 3-sentence prompt ✅
- `chat_session_summaries` table ✅
- Memory reconstruction: last 3 summaries + last 5 messages → `[NEXUS MEMORY]` block ✅
- `chat_messages` and `chat_sessions` tables ✅

### 1.9 Admin Dashboard (all 7 modules from spec)
- Module 1: Config Panel — live edit `network_configs` ✅
- Module 2: Spin Wheel Configurator (prizes, slots, liability cap) ✅
- Module 3: Points Ledger Audit (searchable transactions) ✅
- Module 4: Spin & Prize Management (metrics, fulfilment status) ✅
- Module 5: AI Studio Usage (breakdown by tool, provider, cost vs. points) ✅
- Module 6: Regional Wars Control (resolve, prize pool, leaderboard) ✅
- Module 7: Fraud Alerts (list, resolve, suspend user) ✅
- Lucky Draw admin (bonus — CRUD + execute + export) ✅

### 1.10 Notifications
- SMS via Termii (OTP, streak expiry, studio completion, MoMo confirmations) ✅
- FCM push notifications (registered via `POST /api/v1/notifications/push-token`) ✅
- In-app notification inbox + read/unread ✅
- Preferences management ✅
- Ghost Nudge (streak expiry) — 15-min cron ✅
- Studio asset expiry SMS (48-hour warning) ✅

### 1.11 USSD
- `POST /api/v1/ussd` handler (HMAC-verified) ✅
- Main menu: balance, streak, spin, AI Studio sub-menu ✅
- AI Studio USSD sub-menu: Study Guide (3pts), Quiz (2pts), Mind Map (2pts) ✅
- Spin via USSD ✅
- Daily-limit error handling ✅
- Result delivery: text→SMS, image→SMS link ✅

### 1.12 Lifecycle Worker (all cron jobs)
| Job | Frequency | Status |
|---|---|---|
| ghost-nudge | 15 min | ✅ |
| asset-expiry | 1 hour | ✅ |
| points-expiry | 24 hours | ✅ |
| otp-cleanup | 30 min | ✅ |
| fulfill-retry | 5 min | ✅ |
| sub-lifecycle | 6 hours | ✅ |
| scheduled-draws | 1 hour | ✅ |
| monthly-spin-grant | 24 hours | ✅ |
| session-summarise | 10 min | ✅ |
| momo-held-recovery | 1 hour | ✅ |
| momo-held-expiry | 2 hours | ✅ |
| studio-stale-recovery | 10 min | ✅ |
| wars-monthly-resolve | 24 hours | ✅ |

### 1.13 External Adapters (all production HTTP)
| Adapter | Implementation | Status |
|---|---|---|
| Termii (SMS) | Real HTTP POST | ✅ |
| VTPass | Real HTTP POST | ✅ |
| Paystack | Real webhook verify + HMAC | ✅ |
| MTN MoMo | Real API + poll | ✅ |
| Groq | Real API (Llama-4-Scout) | ✅ |
| Gemini | Real API (Flash-Lite) | ✅ |
| DeepSeek | Real API (V3 chat) | ✅ |
| HuggingFace FLUX | Real API (binary PNG) | ✅ |
| FAL.AI | Real API (image + video) | ✅ |
| ElevenLabs TTS + Music | Real API | ✅ |
| Google Cloud TTS | Real API | ✅ |
| Google Translate | Real API | ✅ |
| AssemblyAI | Real submit→poll | ✅ |
| Mubert | Real API | ✅ |
| rembg (self-hosted) | Real HTTP POST | ✅ |
| remove.bg | Real API (fallback) | ✅ |
| S3 / GCS / Local Storage | Real provider-agnostic | ✅ |
| Apple Wallet (pass JSON) | Real pass generation | ✅ |
| Google Wallet (JWT) | Real JWT signing | ✅ |

---

## SECTION 2 — PARTIALLY BUILT ⚠️

### 2.1 APNs Live Push (Ghost Nudge delivery)
**What spec says:** Updated `.pkpass` pushed to device via APNs (HTTP/2 to `api.push.apple.com`) when streak is within 4 hours of expiry.

**What's built:**
- Ghost Nudge cron fires correctly every 15 minutes ✅
- Queries users with streak expiring within warning window ✅
- SMS fallback fires via Termii ✅
- `WalletPassportAdapter.PushUpdate()` is a **log statement only** — no real APNs HTTP/2 call ⚠️

**What's missing:** The actual `POST https://api.push.apple.com/3/device/{token}` HTTP/2 call with JWT auth from your `.p8` key. The pass generation itself (Apple JSON structure) IS implemented.

**Impact:** Ghost Nudge works via SMS. Lock-screen wallet card nudge does not work until APNs push is wired.

**Effort to fix:** ~1 day. Requires `APPLE_PASS_P8_KEY`, `APPLE_PASS_KEY_ID`, `APPLE_TEAM_ID` env vars + HTTP/2 client.

---

### 2.2 Fraud Rules — 2 of 5 Not Wired

The spec defines 5 fraud rules. Here's the status:

| Rule | Spec Requirement | Status |
|---|---|---|
| FR-02 | >3 spins/day → block + flag | ✅ Wired in `spin_service.go` |
| FR-03 | Same IP >10 recharges/hour → rate limit + CAPTCHA | ⚠️ Not implemented — `fraud_service.go` checks 24h velocity, not per-IP per-hour |
| FR-04 | MoMo same number >5 disbursements/24h → hold | ✅ Wired in `winner_service.go` |
| FR-05 | AI generation >50/day → throttle to 10/hour | ⚠️ Daily generation limit exists (10/day default) but 50→10/hour throttle not implemented |
| FR-06 | Points balance change >500 in 1 min → freeze + alert | ⚠️ Not implemented |

**Impact:** IP-based farm detection and per-minute points velocity detection are absent. Low priority for launch but important pre-scale.

---

## SECTION 3 — NOT YET BUILT ❌

### 3.1 WebSocket Real-Time Leaderboard
**What spec says:** Phase 3 (Weeks 17–18) — real-time leaderboard updates via WebSocket so users see live rank changes during Regional Wars.

**What's built:** REST polling endpoint (`GET /api/v1/wars/leaderboard`) which works fine.

**Impact:** Users can manually refresh to see updated rankings. Not a blocker for launch.

**Effort:** ~2 days. `gorilla/websocket` or `nhooyr.io/websocket` + Redis pub/sub for leaderboard events.

**Note:** This is explicitly a Phase 3 item in the spec. Not a gap in current build phase.

---

## SECTION 4 — PRICING DISCREPANCY (NEEDS YOUR DECISION)

The **Nexus Studio Key Points Discussion** document contains two conflicting pricing tables. Both appear in the same document with no resolution noted:

| Tool | Main Catalogue Price | Tier-2 Corrected Price | Current Build |
|---|---|---|---|
| AI Photo | 10 pts | 20 pts | **10 pts** |
| Animate Photo | 65 pts | 40 pts | **65 pts** |
| Video Premium | 65 pts | (not in corrected table) | **65 pts** |
| Infographic | 10 pts | 5 pts | **5 pts** (build uses key-points doc) |
| Jingle | 200 pts | 120 pts | **200 pts** |
| Video Jingle | 470 pts | 280 pts | **470 pts** |

**The "Tier-2 Corrected" table** applies the financially validated formula: `₦7.50 × pts ≥ API cost + 40% margin`. That produces the lower prices.

**The "Main Catalogue" prices** appear to be the original spec before the financial validation pass.

➡️ **You need to decide which pricing to use.** This is a one-line SQL update per tool in `network_configs` or a single migration. The code itself is agnostic — prices are read from `studio_tools.point_cost` in the database.

---

## SECTION 5 — ITEMS IN BUILD BEYOND THE SPEC (BONUS FEATURES)

These are fully implemented but were **not required by any spec document**:

| Feature | Details |
|---|---|
| Lucky Draw | Full CRUD, weighted random winner selection, export, audit trail |
| Passport Badges | 12 badge types, earned on milestones (streaks, spins, Studio usage) |
| Passport QR code | QR generation + verification endpoint |
| Passport share card | Shareable social card generation |
| AI Studio gallery | User history of all past generations |
| Notification inbox | In-app notification centre with read/unread state |
| Notification preferences | Granular user-controlled notification settings |
| Lucky Draw admin | Full admin CRUD for managing draws and exporting entries |

---

## SECTION 6 — API KEYS YOU NEED TO SIGN UP FOR

See next section (Question 2 answer).
