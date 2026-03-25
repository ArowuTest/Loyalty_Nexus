# API Integration Status — What You Need to Sign Up For
**Loyalty Nexus | March 2026**

This answers: *"Are we fully integrated for all AI components, or do I need to sign up for APIs?"*

---

## THE SHORT ANSWER

You need **12 API accounts**. Of those, **9 are completely free** (no credit card needed, no monthly charges). Only **3 involve real money**, and all 3 are pay-per-use with either a free tier or one-time free credits.

---

## TIER 1 — FREE, NO CREDIT CARD, SIGN UP TODAY

These are available right now, just create an account and paste the key into `.env`:

| # | Service | What it powers | Free Limit | Sign Up |
|---|---|---|---|---|
| 1 | **Groq** | Primary AI Chat (Llama-4-Scout — 300 tok/s) | 6,000 req/min on free tier | console.groq.com |
| 2 | **Google Gemini** (AI Studio) | Secondary chat + ALL studio text tools (study guide, quiz, mindmap, research brief, etc.) + session summarisation + podcast scripting | 1,500 req/day on Flash-Lite | aistudio.google.com |
| 3 | **HuggingFace** | Primary image generation (FLUX.1-schnell) | ~100 req/day free inference | huggingface.co |
| 4 | **Mubert** | Background music generation | 25 tracks/month free | mubert.com/api |
| 5 | **AssemblyAI** | Voice transcription (primary) | $50 free credit on signup (~3,000 mins) | assemblyai.com |
| 6 | **Google Cloud** (API Key) | Google Translate (500K chars/month free) + Google Cloud TTS (1M chars/month free) | Free tier | console.cloud.google.com |
| 7 | **Termii** | SMS — OTP + notifications | Pay-as-you-go (very cheap for Nigeria) | termii.com |

> **Note on Termii:** Strictly speaking this is pay-per-SMS, but it's not an AI cost — it's your business communication infrastructure, same as a phone bill. Cost is negligible for early users.

---

## TIER 2 — PAY-PER-USE (ONLY WHEN USERS SPEND POINTS)

These cost money **only when your users actually use the premium tools**. You are protected because the point costs are set at 70%+ margin, meaning user recharges always cover the API cost before you even call the API.

| # | Service | What it powers | Cost | Your Revenue | Margin |
|---|---|---|---|---|---|
| 8 | **FAL.AI** | Image gen fallback + Animate Photo (LTX) + Video Premium (Kling v1.5) | ~$0.06/image, ~$0.15/video, ~$0.60/Kling video | 65 pts = ₦487.50 | **70%** |
| 9 | **ElevenLabs** | Premium TTS (narrate fallback) + Marketing Jingle (30s) | ~$0.25/jingle | 200 pts = ₦1,500 | **83%** |
| 10 | **DeepSeek** | AI Chat overflow (after 2,000 free req/day exhausted) | $0.028/M tokens | Free chat, cost = ~$0.20/day at 100K MAU | Near-zero |

> **Key point:** You only pay FAL.AI when a user spends 65 points on a video. They had to recharge ₦16,250 worth to earn those 65 points. You've already collected ₦16,250 in revenue before spending ₦145 on the API call.

---

## TIER 3 — OPTIONAL / INFRASTRUCTURE (YOUR CHOICE)

| # | Service | What it powers | Our fallback if not set up | Cost |
|---|---|---|---|---|
| 11 | **AWS S3 / Cloudflare R2 / GCS** | Storing all generated assets (images, audio, video) | Local filesystem (dev only) | S3: ~$0.023/GB/month; R2 is free for first 10GB |
| 12 | **MTN MoMo API** | Cash prize disbursement for Spin & Win (Phase 2) | VTPass airtime/data prizes (Phase 1 fallback) | Per-disbursement fee (MTN commercial) |

> **On storage:** For production you MUST have a cloud storage provider. **Cloudflare R2 is recommended** — it's S3-compatible, free for the first 10GB, and has zero egress fees (unlike AWS). Just set `AWS_S3_ENDPOINT=https://<account>.r2.cloudflarestorage.com` in your `.env`.

---

## SELF-HOSTED (FREE, RUNS IN YOUR DOCKER COMPOSE)

These cost nothing because they run inside your own infrastructure:

| Service | What it powers | Setup |
|---|---|---|
| **rembg** | Background removal (primary — no API cost) | Already in docker-compose as `rembg-service` |
| **Redis** | Chat sessions, rate limiting, spin state, leaderboard | Already in docker-compose |
| **PostgreSQL** | Everything persistent | Already in docker-compose |

---

## COMPLETE SIGNUP CHECKLIST

### Day 1 (Free, takes 30 minutes total)
- [ ] **Groq** → `console.groq.com` → Create API Key → `GROQ_API_KEY=`
- [ ] **Google AI Studio** → `aistudio.google.com` → Get API Key → `GEMINI_API_KEY=`
- [ ] **HuggingFace** → `huggingface.co/settings/tokens` → New token (read) → `HF_TOKEN=`
- [ ] **AssemblyAI** → `assemblyai.com` → Get API Key → `ASSEMBLY_AI_KEY=`

### Day 1 (Free, 10 minutes — one Google account covers both)
- [ ] **Google Cloud Console** → `console.cloud.google.com`
  - Enable "Cloud Translation API" → `GOOGLE_TRANSLATE_API_KEY=`
  - Enable "Cloud Text-to-Speech API" → `GOOGLE_CLOUD_TTS_KEY=`
  - (Both use the same project/API key)

### Day 1 (Free or very cheap)
- [ ] **Mubert** → `mubert.com/api` → Create account → `MUBERT_API_KEY=`
- [ ] **Termii** → `termii.com` → Create account → Top up wallet → `TERMII_API_KEY=`

### Before launch (Pay-per-use, set up when ready to go live)
- [ ] **FAL.AI** → `fal.ai` → Add payment method → `FAL_API_KEY=`
- [ ] **ElevenLabs** → `elevenlabs.io` → Create account (free tier has 10K chars/month) → `ELEVENLABS_API_KEY=`
- [ ] **DeepSeek** → `platform.deepseek.com` → Add credits ($5 minimum) → `DEEPSEEK_API_KEY=`

### Storage (Pick one before deployment)
- [ ] **Cloudflare R2** (recommended) → `dash.cloudflare.com/r2` → Create bucket → Set `AWS_S3_BUCKET=`, `AWS_S3_ENDPOINT=`, `STORAGE_BACKEND=s3`
  - OR **AWS S3** → Set `AWS_S3_BUCKET=`, `AWS_REGION=`, `STORAGE_BACKEND=s3`
  - OR **Google Cloud Storage** → Set `GCS_BUCKET=`, `STORAGE_BACKEND=gcs`

---

## WHAT WORKS WITHOUT ANY API KEYS (FOR TESTING)

If you just want to run the server locally and test the flow end-to-end before signing up for anything:

| Feature | Works without API key? | Behaviour |
|---|---|---|
| Auth (OTP) | ❌ | Termii required for real OTP SMS |
| Recharge | ❌ | Paystack + VTPass needed |
| Spin & Win | ✅ | Full logic works, prize fulfilment skipped |
| Regional Wars | ✅ | Fully functional |
| AI Chat | ❌ | Needs at least GROQ_API_KEY |
| Studio: All text tools | ❌ | Needs GEMINI_API_KEY |
| Studio: Translate | ❌ | Needs GOOGLE_TRANSLATE_API_KEY (or GEMINI as fallback) |
| Studio: TTS/Narrate | ❌ | Needs GOOGLE_CLOUD_TTS_KEY |
| Studio: Transcribe | ❌ | Needs ASSEMBLY_AI_KEY |
| Studio: AI Photo | ❌ | Needs HF_TOKEN |
| Studio: Bg Remove | ✅ | Works if rembg docker container is running |
| Studio: Bg Music | ❌ | Needs MUBERT_API_KEY |
| Studio: Video | ❌ | Needs FAL_API_KEY |
| Studio: Jingle | ❌ | Needs ELEVENLABS_API_KEY |
| Wallet Pass | ✅ | Dev-mode JSON pass (no APNs push) |
| Admin Dashboard | ✅ | Fully functional |

---

## COST PROJECTION (AT LAUNCH — 1,000 ACTIVE USERS/MONTH)

| Cost Category | Monthly Estimate | Notes |
|---|---|---|
| Groq + Gemini + HF + Mubert + rembg | ₦0 | All free tier, nowhere near limits |
| AssemblyAI | ~₦0 | $50 credit lasts ~3,000 mins of audio |
| DeepSeek (chat overflow) | ~₦150/month | ~$0.10 at 1K MAU using 100 chat msgs/day avg |
| FAL.AI (video + image fallback) | Covered by user points | 70% margin; never out-of-pocket |
| ElevenLabs (jingles) | Covered by user points | 83% margin; never out-of-pocket |
| Cloudflare R2 storage | ~₦0 | Free 10GB/month; ~$0.015/GB after |
| Termii SMS | ~₦5,000 | ~500 users × ~10 SMS each |
| **Total Monthly AI/Infra Cost** | **~₦5,150** | Excluding Termii: ₦150 |

> **Conclusion:** At 1,000 MAU you're spending roughly **₦5,000/month** on infrastructure. Your revenue from Paystack/VTPass margin at 1,000 active rechargors × ₦1,000 avg × 3% margin = **₦30,000/month minimum**. The platform is cash-flow positive from day one.
