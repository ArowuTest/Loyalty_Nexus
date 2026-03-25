# Loyalty Nexus — Production Deployment Guide

> **Stack:** Go API (Docker) on Render · User Portal (Next.js 15) on Vercel · Admin Cockpit (Next.js 15) on Vercel · PostgreSQL + Redis on Render Free Tier

---

## Table of Contents

1. [Quick Deploy — Render + Vercel](#1-quick-deploy--render--vercel)  
2. [Running DB Migrations on Render](#2-running-db-migrations-on-render)  
3. [Environment Variables Reference](#3-environment-variables-reference)  
4. [Free Tier Limitations](#4-free-tier-limitations)  
5. [Post-Deploy Checklist](#5-post-deploy-checklist)  
6. [Updating Deployments](#6-updating-deployments)  
7. [Troubleshooting](#7-troubleshooting)

---

## 1. Quick Deploy — Render + Vercel

### 1.1 Backend API on Render

The backend ships as a **multi-stage Docker image** (`backend/Dockerfile`). Render builds the `api` stage automatically when you set the Dockerfile path.

#### Step-by-step

1. **Create a free Render account** at [render.com](https://render.com) and connect your GitHub account.

2. **Provision the PostgreSQL database first** (databases must exist before the service starts):
   - Render Dashboard → **New → PostgreSQL**
   - Name: `loyalty-nexus-db`
   - Database Name: `nexus` · User: `nexus`
   - Plan: **Free** (1 GB; expires after 90 days — upgrade before going live)
   - Click **Create Database** and wait ~60 seconds
   - Copy the **Internal Database URL** from the Info tab — you'll need it for the env var

3. **Create the Redis instance:**
   - Render Dashboard → **New → Redis**
   - Name: `loyalty-nexus-redis`
   - Plan: **Free** (25 MB, no persistence)
   - Max Memory Policy: `allkeys-lru`
   - Copy the **Internal Redis URL**

4. **Deploy the API service using Blueprint (recommended):**
   - Render Dashboard → **New → Blueprint**
   - Connect to your GitHub repo (`ArowuTest/Loyalty_Nexus`)
   - Render reads `render.yaml` automatically and provisions all three services (api, worker, redis) plus the database
   - Fill in `sync: false` secrets in the Environment tab (see §3)

   **— OR — Deploy manually:**
   - Render Dashboard → **New → Web Service**
   - Connect GitHub → Select your repository
   - **Name:** `loyalty-nexus-api`
   - **Root Directory:** `backend`
   - **Runtime:** Docker
   - **Dockerfile Path:** `./Dockerfile`
   - **Docker Context:** `.` (repository root relative to rootDir, i.e., `backend/`)
   - **Branch:** `main`
   - **Plan:** Free

5. **Set environment variables** (Environment tab → Add from `render.yaml` values):
   - `DATABASE_URL` — paste the Internal Database URL from step 2
   - `REDIS_URL` — paste the Internal Redis URL from step 3
   - All `sync: false` secrets from §3 (JWT_SECRET, AES_256_KEY, API keys, etc.)
   - `FRONTEND_URL` — fill in *after* Vercel deploy (§1.2)
   - `ADMIN_URL` — fill in *after* Vercel deploy (§1.3)
   - `CORS_ORIGINS` — comma-separated list: `https://<frontend>.vercel.app,https://<admin>.vercel.app`

6. **Health check:**
   - Settings tab → Health & Alerts → Health Check Path: `/api/v1/health`

7. **First deploy** — Render will pull the image, build it, and start the container. Takes ~3–5 min on first build.

8. **Run database migrations** immediately after first deploy (see §2).

9. Note your Render URL, e.g. `https://loyalty-nexus-api.onrender.com`.

10. **Deploy the worker service** (handles draws, summarisation, lifecycle jobs):
    - Render Dashboard → **New → Background Worker**
    - Same Docker settings as above; Docker Command: `/worker`
    - Reuse the same DATABASE_URL, REDIS_URL, and relevant API keys

---

### 1.2 User Portal (Frontend) on Vercel

1. Go to [vercel.com](https://vercel.com) → **Add New → Project**
2. Import from GitHub → select your repository
3. **Root Directory:** `frontend`
4. **Framework Preset:** Next.js (auto-detected)
5. **Environment Variables:**
   | Key | Value |
   |-----|-------|
   | `NEXT_PUBLIC_API_URL` | `https://loyalty-nexus-api.onrender.com/api/v1` |
6. Click **Deploy**
7. Note the Vercel URL (e.g. `https://loyalty-nexus-portal.vercel.app`)
8. Go back to Render → add this URL as `FRONTEND_URL` on the API service

---

### 1.3 Admin Cockpit on Vercel

1. Vercel → **Add New → Project** → same GitHub repo
2. **Root Directory:** `admin`
3. **Framework Preset:** Next.js (auto-detected)
4. **Environment Variables:**
   | Key | Value |
   |-----|-------|
   | `NEXT_PUBLIC_API_URL` | `https://loyalty-nexus-api.onrender.com/api/v1` |
5. Click **Deploy**
6. Note the Vercel URL (e.g. `https://loyalty-nexus-admin.vercel.app`)
7. Go back to Render → add this URL as `ADMIN_URL` on the API service; update `CORS_ORIGINS` to include it

---

## 2. Running DB Migrations on Render

Migrations live in `database/migrations/` (29 sequential SQL files as of March 2026). They **must** be run in order after the first deploy and after any new migration is added.

### Method A — Render Shell Tab (recommended for first run)

1. Render Dashboard → `loyalty-nexus-api` service → **Shell** tab
2. Run all migrations in order:

```bash
# Run all migrations in sorted order, stopping on error
ls /database/migrations/*.sql | sort | xargs -I{} sh -c 'echo "▶ Running {}..." && psql $DATABASE_URL -f {} && echo "✓ Done"'
```

Or file-by-file for more control:

```bash
psql $DATABASE_URL -f /database/migrations/001_cockpit_configuration.sql
psql $DATABASE_URL -f /database/migrations/002_core_ledger.sql
psql $DATABASE_URL -f /database/migrations/003_nexus_studio.sql
psql $DATABASE_URL -f /database/migrations/004_digital_passport.sql
psql $DATABASE_URL -f /database/migrations/005_regional_wars.sql
psql $DATABASE_URL -f /database/migrations/006_daily_subscriptions.sql
psql $DATABASE_URL -f /database/migrations/007_rls_policies.sql
psql $DATABASE_URL -f /database/migrations/008_hlr_cache.sql
psql $DATABASE_URL -f /database/migrations/009_chat_summaries.sql
psql $DATABASE_URL -f /database/migrations/010_regional_wars_admin.sql
psql $DATABASE_URL -f /database/migrations/011_auth_otp.sql
psql $DATABASE_URL -f /database/migrations/012_user_momo.sql
psql $DATABASE_URL -f /database/migrations/013_prize_fulfillment.sql
psql $DATABASE_URL -f /database/migrations/014_user_spin_credits.sql
psql $DATABASE_URL -f /database/migrations/015_fraud_guards.sql
psql $DATABASE_URL -f /database/migrations/016_draw_engine.sql
psql $DATABASE_URL -f /database/migrations/017_tiered_earning_and_bonuses.sql
psql $DATABASE_URL -f /database/migrations/018_strategic_monetization.sql
psql $DATABASE_URL -f /database/migrations/019_user_profile_expansion.sql
psql $DATABASE_URL -f /database/migrations/020_spec_alignment_and_complete_schema.sql
psql $DATABASE_URL -f /database/migrations/021_passport_badges_and_wars.sql
psql $DATABASE_URL -f /database/migrations/022_notifications_and_subscriptions.sql
psql $DATABASE_URL -f /database/migrations/023_admin_phase8.sql
psql $DATABASE_URL -f /database/migrations/024_phase8_draw_spin_winner.sql
psql $DATABASE_URL -f /database/migrations/025_phase9_passport_ussd.sql
psql $DATABASE_URL -f /database/migrations/026_phase10_studio_hardening.sql
psql $DATABASE_URL -f /database/migrations/027_phase11_wars_hardening.sql
psql $DATABASE_URL -f /database/migrations/028_phase12_production_hardening.sql
psql $DATABASE_URL -f /database/migrations/029_phase16_enterprise_studio_tools.sql
```

> **Note:** The `database/migrations/` directory is mounted inside the container at `/database/migrations`. If `psql` is not in the distroless image, connect from your local machine using the **External Database URL** shown in the Render DB Info tab.

### Method B — From Your Local Machine

```bash
# Copy the External Database URL from Render → DB Info tab
export DATABASE_URL="postgres://nexus:<password>@<host>:5432/nexus?sslmode=require"

ls database/migrations/*.sql | sort | xargs -I{} sh -c 'echo "▶ {}" && psql $DATABASE_URL -f {}'
```

### Method C — Render Pre-Deploy Job (CI/CD, production)

Add a `preDeployCommand` in `render.yaml` under the api service to automate migrations on every deploy:

```yaml
preDeployCommand: "ls /database/migrations/*.sql | sort | xargs -I{} psql $DATABASE_URL -f {} || true"
```

> **Caution:** Use `|| true` only when migrations are idempotent (`IF NOT EXISTS`, `CREATE OR REPLACE`, etc.). Review each migration before enabling this.

---

## 3. Environment Variables Reference

> Variables marked **Required** must be set before the service will start correctly.  
> Variables marked `sync: false` in `render.yaml` are secrets that must be entered manually in the Render Dashboard.

### 3.1 Core / Server

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `PORT` | HTTP server port | ✅ | `8080` |
| `ENVIRONMENT` | Runtime environment | ✅ | `production` |
| `OPERATION_MODE` | Business mode | ✅ | `independent` |
| `DATABASE_URL` | PostgreSQL connection string (injected by Render) | ✅ | `postgres://nexus:pw@host:5432/nexus?sslmode=require` |
| `REDIS_URL` | Redis connection string (injected by Render) | ✅ | `redis://red-xxx:6379` |

### 3.2 Security

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `JWT_SECRET` | HS256 signing secret (min 32 chars) | ✅ | `some-long-random-secret-here` |
| `JWT_EXPIRY_HOURS` | User token lifetime | ✅ | `24` |
| `ADMIN_JWT_EXPIRY_HOURS` | Admin token lifetime | ✅ | `8` |
| `AES_256_KEY` | 64 hex chars for OTP encryption | ✅ | `0000...0000` (64 chars) |

### 3.3 AI / LLM Providers

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `GROQ_API_KEY` | Primary chat LLM (Llama-4-Scout) | ✅ | `gsk_...` |
| `GEMINI_API_KEY` | Studio text tools + fallback chat | ✅ | `AIzaSy...` |
| `DEEPSEEK_API_KEY` | Last-resort LLM overflow | ⬜ | `sk-...` |
| `HF_TOKEN` | HuggingFace: FLUX image + MusicGen audio | ✅ | `hf_...` |
| `HF_IMAGE_MODEL` | Image generation model ID | ✅ | `black-forest-labs/FLUX.1-schnell` |
| `FAL_API_KEY` | FAL.AI fallback image + video | ⬜ | `...` |
| `POLLINATIONS_SECRET_KEY` | Unlocks Whisper, TTS, GPT Image, etc. | ✅ | `...` |
| `ASSEMBLY_AI_KEY` | Primary speech-to-text transcription | ⬜ | `...` |

### 3.4 Voice / TTS

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `GOOGLE_CLOUD_TTS_KEY` | Google TTS (1M chars/month free) | ⬜ | `AIzaSy...` |
| `ELEVENLABS_API_KEY` | Premium TTS + jingle generation | ⬜ | `sk_...` |
| `ELEVENLABS_VOICE_ID` | Default ElevenLabs voice | ⬜ | `21m00Tcm4TlvDq8ikWAM` |

### 3.5 Translation

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `GOOGLE_TRANSLATE_API_KEY` | Google Translate v2 (500k chars/month free) | ⬜ | `AIzaSy...` |

### 3.6 Asset Storage

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `STORAGE_BACKEND` | Storage provider: `s3`, `gcs`, or `local` | ✅ | `s3` |
| `STORAGE_CDN_BASE_URL` | CDN prefix for all asset URLs | ⬜ | `https://cdn.loyalty-nexus.ai` |
| `AWS_S3_BUCKET` | S3/R2/MinIO bucket name | ✅ (if s3) | `nexus-assets` |
| `AWS_REGION` | AWS region | ✅ (if s3) | `us-east-1` |
| `AWS_ACCESS_KEY_ID` | AWS / R2 access key | ✅ (if s3) | `AKIA...` |
| `AWS_SECRET_ACCESS_KEY` | AWS / R2 secret | ✅ (if s3) | `...` |
| `AWS_S3_ENDPOINT` | Custom S3 endpoint (R2/MinIO only) | ⬜ | `https://xxx.r2.cloudflarestorage.com` |
| `GCS_BUCKET` | Google Cloud Storage bucket | ✅ (if gcs) | `nexus-assets` |
| `GCS_SERVICE_ACCOUNT_JSON` | GCS service account as raw JSON string | ✅ (if gcs) | `{"type":"service_account",...}` |

### 3.7 Background Removal

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `REMBG_SERVICE_URL` | Self-hosted rembg microservice URL | ⬜ | `https://rembg.your-domain.com` |
| `REMOVEBG_API_KEY` | remove.bg paid fallback | ⬜ | `...` |

### 3.8 SMS / Telco

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `TERMII_API_KEY` | Termii SMS gateway (OTP delivery, NG) | ✅ | `TLxx...` |
| `TERMII_SENDER_ID` | SMS sender name | ✅ | `Nexus` |

### 3.9 Payments

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `PAYSTACK_SECRET_KEY` | Paystack payments | ✅ | `sk_live_...` |
| `PAYSTACK_WEBHOOK_SECRET` | Paystack webhook HMAC secret | ✅ | `...` |
| `VTPASS_API_KEY` | VTPass airtime/data | ✅ | `...` |
| `VTPASS_PUBLIC_KEY` | VTPass public key | ✅ | `...` |
| `VTPASS_SECRET_KEY` | VTPass secret key | ✅ | `...` |
| `VTPASS_BASE_URL` | VTPass API base URL | ✅ | `https://vtpass.com/api` |
| `VTPASS_ENV` | `sandbox` or `production` | ✅ | `production` |
| `MTN_MOMO_BASE_URL` | MTN MoMo API base URL | ✅ | `https://proxy.momodeveloper.mtn.com` |
| `MTN_MOMO_SUBSCRIPTION_KEY` | Ocp-Apim-Subscription-Key | ✅ | `...` |
| `MTN_MOMO_API_USER` | MTN API user UUID | ✅ | `...` |
| `MTN_MOMO_API_KEY` | MTN API key | ✅ | `...` |
| `MTN_MOMO_ENVIRONMENT` | `sandbox` or `production` | ✅ | `production` |
| `MOMO_CALLBACK_URL` | Callback endpoint for MoMo webhooks | ✅ | `https://<api-url>/api/v1/momo/callback` |

### 3.10 CORS / Frontend

| Variable | Description | Required | Example |
|---|---|:---:|---|
| `FRONTEND_URL` | User portal base URL (for email links, CORS) | ✅ | `https://loyalty-nexus.vercel.app` |
| `ADMIN_URL` | Admin cockpit base URL | ✅ | `https://loyalty-nexus-admin.vercel.app` |
| `CORS_ORIGINS` | Comma-separated allowed origins | ✅ | `https://loyalty-nexus.vercel.app,https://loyalty-nexus-admin.vercel.app` |

### 3.11 Fraud Guard Thresholds

| Variable | Description | Required | Default |
|---|---|:---:|---|
| `FRAUD_MAX_RECHARGE_24H` | Max recharge attempts per user per 24h | ✅ | `20` |
| `FRAUD_MAX_SPIN_24H` | Max spin attempts per user per 24h | ✅ | `10` |
| `FRAUD_MIN_RECHARGE_NAIRA` | Minimum recharge amount in Naira | ✅ | `100` |

### 3.12 Feature Flags

| Variable | Description | Required | Default |
|---|---|:---:|---|
| `OPERATION_MODE` | `independent` · `pilot` · `national` | ✅ | `independent` |
| `REGIONAL_WARS_PRIZE_POOL_NAIRA` | Regional wars total prize pool | ✅ | `500000` |
| `MONTHLY_DRAW_PRIZE_NAIRA` | Monthly draw prize pool | ✅ | `1000000` |

### 3.13 Frontend-only (Vercel — set as environment variables in Vercel project settings)

| Variable | App | Description | Required | Example |
|---|---|---|:---:|---|
| `NEXT_PUBLIC_API_URL` | frontend + admin | Backend API base URL | ✅ | `https://loyalty-nexus-api.onrender.com/api/v1` |

> In Vercel, create a project-level secret named `nexus_api_url` (no `NEXT_PUBLIC_` prefix) and reference it as `@nexus_api_url` in `vercel.json`, **or** set `NEXT_PUBLIC_API_URL` directly as a plain environment variable in the Vercel project settings.

---

## 4. Free Tier Limitations

> Read this before inviting real users. Free tiers are for testing only.

| Service | Limitation | Impact | Mitigation |
|---|---|---|---|
| **Render Free Web Service** | Spins down after **15 min inactivity**; cold start ~30 s | First request after idle is slow | Upgrade to **Starter ($7/mo)** — always-on |
| **Render Free PostgreSQL** | **1 GB storage**, **expires after 90 days** | DB deleted automatically | Upgrade to Starter DB ($7/mo) or export data before expiry |
| **Render Free Redis** | **25 MB**, **no persistence** (data lost on restart) | Session data / job queues reset on deploy | Upgrade to Starter Redis ($10/mo) or use Upstash free tier |
| **Render Free Worker** | Same spin-down rules as web service | Scheduled jobs may miss their window | Upgrade or use Render Cron Jobs with a free plan |
| **Vercel Free (Hobby)** | 100 GB bandwidth/month, 12s serverless function timeout, 6000 build minutes/month | Fine for testing; may need upgrade under real load | Upgrade to Vercel Pro ($20/mo) for teams |
| **Vercel Free — Regions** | Only `fra1` is pinned; other regions may activate | Latency variance for Nigerian users | Consider `sin1` or `iad1` for lower latency to NG |

### Recommended Paid Upgrade Path (before real users)

```
Render Starter ($7/mo) — API always on
Render Starter DB ($7/mo) — persistent PostgreSQL
Upstash Redis (free tier, 10k req/day) — replaces Render free Redis
Vercel Hobby (free) — sufficient for early-stage traffic
```

Total minimum cost: **~$14/month** for a production-grade deployment.

---

## 5. Post-Deploy Checklist

Run through these checks after every new deployment:

### Infrastructure

- [ ] **Backend health** — `GET https://<api-url>/api/v1/health` returns `{"status":"ok"}`
- [ ] **Database connected** — no `DB connect` errors in Render logs
- [ ] **Redis connected** — no `redis` dial errors in logs
- [ ] **All 29 migrations run** — check for expected tables in Render DB Shell: `\dt`

### Frontend & Admin

- [ ] **Frontend loads** — navigates to login page without 500 errors
- [ ] **Admin loads** — navigates to login page; protected routes redirect correctly
- [ ] **`NEXT_PUBLIC_API_URL` points to Render URL** — open browser DevTools → Network, check API calls go to the right host

### Auth & Flows

- [ ] **User registration** — `/register` → OTP delivered via Termii → account created
- [ ] **User login** — `/login` → JWT issued → dashboard loads
- [ ] **Admin login** — `/login` on admin app → admin JWT issued → cockpit loads

### AI Studio

- [ ] **AI Photo (free)** — Studio → AI Photo tool → image generated via Pollinations
- [ ] **`POLLINATIONS_SECRET_KEY` set** — check no `401 Unauthorized` in API logs for Studio calls
- [ ] **Chat** — Studio → Chat → Groq response within 5 s

### Payments & Telco

- [ ] **VTPass env** — `VTPASS_ENV` set to `sandbox` for test, `production` for live
- [ ] **MTN MoMo env** — `MTN_MOMO_ENVIRONMENT` matches intended environment
- [ ] **Paystack webhook** — test webhook delivery from Paystack dashboard

### Configuration

- [ ] **`FRONTEND_URL`** set on backend (used in email/OTP links)
- [ ] **`ADMIN_URL`** set on backend
- [ ] **`CORS_ORIGINS`** includes both Vercel domains
- [ ] **`MOMO_CALLBACK_URL`** updated to Render URL
- [ ] **Fraud guard thresholds** reviewed for production values

### Security

- [ ] **`JWT_SECRET`** is a random string of ≥ 32 characters (not the placeholder)
- [ ] **`AES_256_KEY`** is a 64-character hex string (not all zeros)
- [ ] **`.env` file not committed** — `git log --oneline | head` — confirm no secrets in history
- [ ] **`VTPASS_ENV=production`** and **`MTN_MOMO_ENVIRONMENT=production`** when going live with real money

---

## 6. Updating Deployments

### Backend (Render)

Render auto-deploys on push to `main` (when `autoDeploy: true`).  
To deploy manually: Render Dashboard → `loyalty-nexus-api` → **Manual Deploy → Deploy latest commit**.

After deploying a new migration file:
1. Push to `main` → wait for Render build
2. Open Render Shell → run the new migration file only:
   ```bash
   psql $DATABASE_URL -f /database/migrations/029_phase16_enterprise_studio_tools.sql
   ```

### Frontend / Admin (Vercel)

Vercel auto-deploys on push to `main`. Preview deployments are created for all other branches.

To update an environment variable: Vercel Dashboard → Project → Settings → Environment Variables → update → **Redeploy** (required for the change to take effect).

---

## 7. Troubleshooting

| Symptom | Likely Cause | Fix |
|---|---|---|
| API returns 503 on first request | Render free tier cold start | Wait 30s; upgrade to Starter for always-on |
| `DB connect` fatal error on startup | `DATABASE_URL` not set or wrong | Check Render env vars; ensure database is provisioned |
| `relation "users" does not exist` | Migrations not run | Run all migrations (§2) |
| Frontend shows `Network Error` | `NEXT_PUBLIC_API_URL` wrong or missing | Redeploy Vercel after setting correct URL |
| CORS error in browser | `CORS_ORIGINS` missing Vercel domain | Update `CORS_ORIGINS` on Render → redeploy |
| OTP SMS not delivered | `TERMII_API_KEY` not set | Add key in Render env vars |
| AI Studio tools return 401 | `POLLINATIONS_SECRET_KEY` / LLM key missing | Set missing key in Render env vars |
| Worker not processing jobs | Worker service not deployed or crashing | Check Render worker logs; ensure `REDIS_URL` is set |
| Admin pages show blank/403 | Admin JWT not configured or CORS blocked | Verify `ADMIN_URL` and `CORS_ORIGINS` on API |

---

*Generated for Loyalty Nexus · March 2026*
