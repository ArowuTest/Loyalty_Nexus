# ⚡ Loyalty Nexus

> **Nigeria's premium telecom loyalty platform** — recharge airtime, earn Pulse Points, spin to win cash/data prizes, and access 17 free AI tools powered by NotebookLM, Groq and Gemini.

[![Go](https://img.shields.io/badge/Go-1.23-blue)](https://go.dev) [![Next.js](https://img.shields.io/badge/Next.js-15-black)](https://nextjs.org) [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-blue)](https://postgresql.org) [![Redis](https://img.shields.io/badge/Redis-7-red)](https://redis.io) [![License](https://img.shields.io/badge/License-Private-lightgrey)](#)

---

## Table of Contents

- [Architecture](#architecture)
- [Core Features](#core-features)
- [Quick Start](#quick-start)
- [Environment Variables](#environment-variables)
- [API Reference](#api-reference)
- [Database Schema](#database-schema)
- [AI Studio Tools](#ai-studio-tools)
- [Build Phases](#build-phases)

---

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                     LOYALTY NEXUS                         │
│                                                           │
│  ┌───────────┐  ┌───────────┐  ┌───────────────────┐   │
│  │ Next.js   │  │  Flutter  │  │    USSD Gateway    │   │
│  │ Web App   │  │ Mobile    │  │  (Africa's Talking)│   │
│  └─────┬─────┘  └─────┬─────┘  └──────────┬────────┘   │
│        └──────────────┼─────────────────────┘           │
│                       ▼                                   │
│              ┌────────────────┐                          │
│              │  Go REST API   │  :8080                   │
│              │  (Chi Router)  │                          │
│              └───────┬────────┘                          │
│          ┌───────────┼───────────┐                       │
│          ▼           ▼           ▼                        │
│    ┌──────────┐ ┌─────────┐ ┌──────────┐                │
│    │PostgreSQL│ │  Redis  │ │  Queue   │                 │
│    │  (GORM) │ │ (Cache) │ │(Streams) │                 │
│    └──────────┘ └─────────┘ └──────────┘                │
│                                                           │
│  External Services:                                       │
│  • Paystack (payment webhooks)                            │
│  • VTPass (airtime/data provisioning)                    │
│  • MTN MoMo (cash disbursement)                          │
│  • Termii + Africa's Talking (SMS)                       │
│  • Groq → Gemini → DeepSeek (AI chat cascade)            │
│  • NotebookLM CLI (knowledge tools — FREE)               │
└──────────────────────────────────────────────────────────┘
```

### Stack

| Layer        | Technology                                      |
|--------------|-------------------------------------------------|
| API          | Go 1.23, net/http (stdlib router), GORM         |
| Frontend     | Next.js 15, React 19, Tailwind CSS 3, Framer Motion |
| Mobile       | Flutter 3.x (iOS + Android)                     |
| Database     | PostgreSQL 16 with RLS                          |
| Cache/Queue  | Redis 7 (Streams for async jobs)                |
| Infra        | Docker Compose, multi-stage Dockerfiles         |

---

## Core Features

### 💰 Earn — Pulse Points Ledger
- **Tiered earning**: ₦0–199 = 0 pts/spin, ₦200–999 = 2pts + 1 spin/₦200, ₦1000+ = 5pts + 1 spin/₦200 bonus
- **Streak multiplier**: 7-day streak = 1.5×, 30-day = 2×, 90-day = 2.5×
- **Regional war bonus**, referral bonuses, first-recharge reward

### 🎡 Spin & Win
- CSPRNG weighted prize selection (no predictable outcomes)
- Daily spin limits, global liability caps, per-prize inventory caps
- Instant airtime/data via VTPass, cash via MTN MoMo
- MoMo-hold flow: cash prizes held until MoMo wallet linked

### 🧠 Nexus AI Studio (17 Tools — ALL FREE)
| Category | Tools |
|----------|-------|
| Knowledge | PDF Guide, Quiz Generator, Mind Map, Deep Research, Audio Overview |
| Document  | Business Plan, CV Generator, Cover Letter, Slide Deck, Infographic |
| Image     | Image Generator, Background Remover |
| Audio     | Voice Story, Music Jingle, Podcast Episode |
| Language  | Yoruba/Igbo/Hausa Translator, Language Tutor |
| Video     | My Video Story |

Powered by **NotebookLM** (₦0 API cost), **HuggingFace Flux** (free tier), **Groq** (free tier), **Gemini Flash** (free tier).

### 🌍 Regional Wars
- States compete monthly on total Pulse Points earned
- Top 3 states share ₦500k monthly prize pool
- Real-time leaderboard updated every 15 minutes

### 🔐 Security
- OTP authentication (no passwords) — AES-256-GCM encrypted storage
- 30-day JWT sessions with admin role separation
- Row-level locking on wallet updates (atomic operations)
- Fraud detection: velocity limits, device fingerprinting, blacklist
- HMAC-SHA256 webhook signature verification (Paystack, MNO)

---

## Quick Start

### Prerequisites
- Docker + Docker Compose
- Go 1.23+ (for local dev)
- Node.js 22+ (for frontend dev)

### 1. Clone and configure
```bash
git clone https://github.com/ArowuTest/Loyalty_Nexus.git
cd Loyalty_Nexus
cp .env.example .env
# Fill in your API keys in .env
```

### 2. Start all services
```bash
docker compose up -d
```

### 3. Run database migrations
```bash
# Migrations auto-run on container start, or manually:
docker compose exec db psql -U postgres -d loyalty_nexus -f /migrations/001_cockpit_configuration.sql
# ... run 001 through 020 in order
```

### 4. Access
| Service    | URL                          |
|------------|------------------------------|
| API        | http://localhost:8080        |
| Frontend   | http://localhost:3000        |
| PostgreSQL | localhost:5432               |
| Redis      | localhost:6379               |
| pgAdmin    | http://localhost:5050        |

---

## Environment Variables

See `.env.example` for the complete list. Key variables:

```bash
# Database
DATABASE_URL=postgresql://postgres:password@db:5432/loyalty_nexus

# Auth
JWT_SECRET=your-256-bit-secret
OTP_ENCRYPTION_KEY=32-byte-hex-key

# Payments
PAYSTACK_SECRET_KEY=sk_live_xxx
VTPASS_API_KEY=xxx
VTPASS_SECRET_KEY=xxx

# MoMo
MTN_MOMO_SUBSCRIPTION_KEY=xxx
MTN_MOMO_API_USER=xxx
MTN_MOMO_API_KEY=xxx

# AI
GROQ_API_KEY=gsk_xxx
GEMINI_API_KEY=xxx
DEEPSEEK_API_KEY=xxx

# SMS
TERMII_API_KEY=xxx
AFRICAS_TALKING_API_KEY=xxx
```

---

## API Reference

### Authentication
```
POST /api/v1/auth/otp/send    { phone_number, purpose }
POST /api/v1/auth/otp/verify  { phone_number, code, purpose }
```

### User
```
GET  /api/v1/user/profile          # User profile
GET  /api/v1/user/wallet           # Pulse Points + Spin Credits
GET  /api/v1/user/transactions     # Transaction history
POST /api/v1/user/momo/request     # Initiate MoMo link
POST /api/v1/user/momo/verify      # Confirm MoMo link
GET  /api/v1/user/passport         # Apple/Google Wallet URLs
```

### Spin
```
GET  /api/v1/spin/wheel            # Current prize wheel config
POST /api/v1/spin/play             # Play a spin (uses 1 credit)
GET  /api/v1/spin/history          # User's spin history
```

### AI Studio
```
GET  /api/v1/studio/tools                   # All 17 tools
POST /api/v1/studio/chat                    # Chat with Nexus AI
POST /api/v1/studio/generate                # Submit generation job
GET  /api/v1/studio/generate/{id}/status   # Poll status
GET  /api/v1/studio/gallery                 # User's generated assets
```

### Webhooks
```
POST /api/v1/recharge/paystack-webhook      # Paystack charge.success
POST /api/v1/recharge/mno-webhook           # BSS billing event (integrated mode)
POST /api/v1/ussd                           # Africa's Talking USSD
```

### Admin (admin JWT required)
```
GET  /api/v1/admin/dashboard
GET  /api/v1/admin/config
PUT  /api/v1/admin/config/{key}
GET  /api/v1/admin/prize-pool
GET  /api/v1/admin/users
PUT  /api/v1/admin/users/{id}/suspend
GET  /api/v1/admin/fraud-events
GET  /api/v1/admin/regional-wars
```

---

## Database Schema

20 migrations, zero hardcoded config:

| Migration | Description |
|-----------|-------------|
| 001 | network_configs, prize_pool, regional_settings, studio_config |
| 002 | users, transactions, atomic ledger trigger |
| 003 | studio_tools (17 seeded), ai_generations |
| 004 | wallets, wallet_passes (Digital Passport) |
| 005 | regional_settings, regional_stats, tournaments |
| 006 | subscription_plans, user_subscriptions |
| 007 | RLS policies |
| 008 | network_cache (HLR lookup cache) |
| 009 | chat_sessions, chat_messages, session_summaries |
| 010 | regional_wars admin views |
| 011 | auth_otps |
| 012 | momo_links |
| 013 | prize_claims, fulfillment_log |
| 014 | user_spin_credits |
| 015 | msisdn_blacklist, fraud_events |
| 016 | draw_engine, lottery_draws |
| 017 | recharge_tiers, program_bonuses |
| 018 | strategic_monetization (GPU usage, ARPU) |
| 019 | user_profile_expansion (admin roles) |
| 020 | **Spec alignment**: wallets→two-pool, spin_results, fraud_events, multipliers, expiry |

---

## AI Studio Tools

All 17 tools are free. Cost = Pulse Points spent per generation:

| # | Tool | Provider | Cost |
|---|------|----------|------|
| 1 | PDF Study Guide | NotebookLM | 0 pts |
| 2 | Smart Quiz Generator | NotebookLM | 0 pts |
| 3 | Mind Map Builder | NotebookLM | 0 pts |
| 4 | Deep Research Assistant | NotebookLM | 0 pts |
| 5 | Audio Overview | NotebookLM | 0 pts |
| 6 | Business Plan Generator | NotebookLM | 50 pts |
| 7 | CV Generator | NotebookLM | 30 pts |
| 8 | Cover Letter Writer | NotebookLM | 20 pts |
| 9 | Slide Deck Creator | NotebookLM | 40 pts |
| 10 | Infographic Maker | NotebookLM | 30 pts |
| 11 | AI Image Generator | HuggingFace Flux | 10 pts |
| 12 | Background Remover | RemBG (self-hosted) | 5 pts |
| 13 | Voice Story | AssemblyAI + Gemini | 25 pts |
| 14 | Music Jingle | Mubert | 50 pts |
| 15 | Podcast Episode | NotebookLM | 0 pts |
| 16 | Language Translator | Google Translate | 0 pts |
| 17 | My Video Story | FAL.AI | 100 pts |

---

## Build Phases

| Phase | Weeks | Focus |
|-------|-------|-------|
| ✅ 1 | 1-2  | Schema alignment, Go foundation, Docker, all services |
| 🔄 2 | 3-4  | Next.js frontend (complete), all API integrations |
| 3 | 5-6  | Flutter mobile app (complete) |
| 4 | 7-8  | NotebookLM CLI integration, HuggingFace, RemBG |
| 5 | 9-10 | Admin Cockpit frontend |
| 6 | 11-12| Regional Wars real-time leaderboard |
| 7 | 13-14| USSD polish, feature phone UX |
| 8 | 15-16| Digital Passport (Apple/Google Wallet) |
| 9 | 17-18| Load testing, security audit |
| 10| 19-20| Production deployment, monitoring |

---

## License

Private — All rights reserved. © 2026 Loyalty Nexus.
