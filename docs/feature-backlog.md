# Loyalty Nexus — Feature Backlog
*Living document · Last updated: 2026-03-29*

---

## FEATURE 1 — Public Regional Wars Leaderboard Page

### Overview
A public-facing, real-time leaderboard page that lets anyone (logged in or not) see how Nigerian states are performing in the current month's Regional Wars competition. The goal is **excitement and social proof** — not revealing MTN's commercial recharge volumes.

### Problem It Solves
- Currently the Wars leaderboard is locked behind a dashboard login
- Potential users have no reason to join unless they already know the competition exists
- MTN partners need a shareable, embeddable page to promote the campaign without exposing commercial data

### What To Show (not recharge ₦ amounts)
| Display | Source | Notes |
|---|---|---|
| State name + flag emoji | Static | All 36 states + FCT |
| Rank (1st, 2nd … 37th) | Live API | Ordered by normalised score |
| **Trend indicator** (🔥 rising / ⬇️ falling / ➡️ stable) | Computed | Compare rank vs 24h ago |
| **Activity bar** (relative width, not absolute numbers) | Normalised 0–100 | Max state = 100%, others proportional |
| Prize at stake | Static copy | "₦250,000" for 1st etc. |
| Time remaining in month | Countdown timer | Client-side |
| Top 3 podium callout | API rank | Highlighted with gold/silver/bronze |
| "Your state" highlight | If logged in, highlight user's state | Auth-aware |

### What NOT To Show
- Actual Pulse Points totals
- Recharge naira amounts
- Subscriber counts
- Any individual user data

### API Changes Needed
- `GET /api/v1/wars/public-leaderboard` — returns ranked list of states with normalised scores (0–100), trend direction, and time-remaining. No auth required.
- Backend computes normalised score: `score_i = (raw_points_i / max_points_all_states) * 100`

### Page URL
`/wars` (currently redirects to home for unauthed users — change to show public leaderboard, optionally prompt login to see personal rank)

### Design Direction
- Full-page dark theme matching landing page
- Large animated podium for top 3 states
- Scrollable ranked list for all 37 states
- Live activity ticker (state X just moved up!)
- Countdown to end of month
- CTA: "Recharge now to help [Your State] climb" → opens auth modal

### Priority: HIGH
Excellent MTN demo asset. Shareable link. Drives organic sign-ups.

---

## FEATURE 2 — Loyalty Nexus Community Page

### Overview
A community hub where Loyalty Nexus members can post questions, share wins, give feedback, and engage with product updates — similar in structure to **Perplexity's community forum** (community.perplexity.ai).

### Problem It Solves
- No current channel for users to interact with each other or with the team
- No place to celebrate wins publicly (e.g. "I just won ₦5,000 on the wheel!")
- No lightweight feedback loop between users and the product team
- Reduces support load by enabling peer-to-peer answers

### Inspiration: Perplexity Community
The Perplexity forum (community.perplexity.ai) features:
- **Categorised threads** (How-To, Feature Requests, Bug Reports, Show & Tell)
- **Post cards** with title, reply count, view count, and last-active timestamp
- **Pinned/featured posts** from the team at the top
- **Simple composer** — title + body, optional category tag
- **Upvote system** per post (not per reply)
- Clean, minimal dark UI — no clutter

### Proposed Categories for Loyalty Nexus Community
| Category | Purpose |
|---|---|
| 🏆 **Win of the Week** | Members share their spin wins, AI creations, and prize payouts |
| 💬 **General Chat** | Open conversation about the platform |
| 🤖 **AI Studio Tips** | Share prompts, best practices, tool discoveries |
| ⚔️ **Regional Wars** | State banter, strategy, leaderboard reactions |
| 💡 **Feature Requests** | Users suggest new tools, improvements |
| 🐛 **Help & Support** | Peer-to-peer troubleshooting |
| 📣 **Announcements** | Team-only posts pinned at top (new tools, prize results, updates) |

### MVP Scope (Phase 1 — Frontend-only / Static)
- `/community` page with category cards and a "coming soon" composer
- Pinned team posts displayed as styled cards (hardcoded JSON initially)
- Links to email/WhatsApp for actual support during MVP
- Auth-aware: logged-in users see their name; unauthed users see login prompt to post

### Phase 2 — Full Backend
- Post & reply storage in PostgreSQL
- Auth-gated posting (must be Loyalty Nexus member)
- Upvote / reaction system (🔥 ❤️ 💡)
- Admin moderation panel
- Real-time new-post notifications via existing notification system
- Optional: MTN OTP-verified posting to prevent spam

### Page URL
`/community`

### Design Direction
- Match landing page dark aesthetic
- Category grid at top (like Perplexity's sidebar categories as cards on mobile)
- Featured/pinned post section
- Paginated post list with tag, title, reply count, views, timestamp
- "Start a thread" button → auth modal for unauthed, composer modal for authed
- Mobile-first (most Nigerian users will be on phones)

### Priority: MEDIUM
Strong retention and engagement driver. Phase 1 can be shipped as mostly-static in 1–2 days.

---

## Summary Table

| # | Feature | Priority | Est. Effort | Prerequisite |
|---|---|---|---|---|
| 1 | Public Regional Wars Leaderboard | HIGH | 2–3 days (FE + 1 API endpoint) | `GET /wars/public-leaderboard` endpoint |
| 2 | Community Page — Phase 1 (static) | MEDIUM | 1 day | None |
| 2 | Community Page — Phase 2 (full) | MEDIUM | 5–7 days | Auth, DB schema, moderation |
| 3 | Remotion Video Templates | MEDIUM | 3–5 days | Node render-service on Render |
| 4 | Claude/ChatGPT MCP Connector | HIGH | 5–8 days | OAuth 2.1 AS + MCP server + security review |

---

## FEATURE 3 — Remotion Video Templates (cheap, templated video)

### Overview
Programmatic, **templated** video (React/Remotion) as a near-free complement to the AI-generative video tools. Renders deterministic content — slideshows from a user's AI images, personalized **"Recharge Wrapped"**, prize-reveal clips, captioned social clips, lyric/quote videos. ~$0.001/render vs $0.20/sec for FAL avatar.

### Approach
- **Render target: self-hosted `renderMedia()` — NO AWS.** A Docker Node "render-service" runs Remotion + headless Chromium, deployed on **Render** now; the same container is **host-portable to GCP Cloud Run** (plain container, reusing existing GCS creds) when Loyalty Nexus migrates — a config change, not a rewrite. NOT the alpha `@remotion/cloudrun` distributed product. AWS Lambda only if volume ever justifies it.
- **Go backend** reuses the shipped `dispatchAvatar` pattern: new `render` provider category + `remotion` template in `entities/ai_provider.go`; `dispatchRender` in `ai_studio_service.go` calls the render-service via an **env-configured `RENDER_SERVICE_URL`** (never a hardcoded host, so the GCP move is one env var). Reuses `RequestGeneration` (points), `worker.DispatchGeneration`, `uploadOrDataURI` (R2/GCS), and `FailGeneration` (refund).
- **Phase-1 slice:** one composition + one tool — `video-slideshow` (3–5 user images + a music-tool track → montage with transitions/captions → MP4).
- **Owner action:** Remotion Company License (~$100/mo) for commercial use. Cap length/resolution and queue one render at a time on a Standard+ Render instance (Chromium is RAM-heavy).

## FEATURE 4 — Claude / ChatGPT MCP Connector (+ OAuth 2.1)

### Overview
Expose Nexus AI Studio as a remote **MCP server** so a user connects their Loyalty Nexus account inside Claude/ChatGPT and triggers generations that spend **their own Pulse Points**. The assistant is the conversational front-end; LN is the generation backend + wallet.

### Approach
- **OAuth 2.1 authorization layer** (delegated authorization — **does NOT replace OTP; reuses it** to verify identity on a consent page). Spec: OAuth 2.1 + PKCE(S256) + Dynamic Client Registration (RFC 7591 — Claude auto-registers) + Protected Resource Metadata (RFC 9728). Build with a vetted Go library (`ory/fosite`), not hand-rolled. New endpoints: `/.well-known/oauth-protected-resource`, `/.well-known/oauth-authorization-server`, `POST /oauth/register`, `GET /oauth/authorize` (OTP-backed consent), `POST /oauth/token`. New tables: `oauth_clients`, `oauth_auth_codes`, `oauth_tokens` (scopes + `spend_cap_daily` + revoked) — mirrors the admin-refresh-token store (migration 074).
- **MCP server** (Streamable HTTP, 2026 standard) exposing tools — `generate_image`, `generate_video_from_image`, `talking_avatar`, `check_balance` — each validating the token, resolving the user, calling the existing generation pipeline, and enforcing the spend cap.
- **Security (non-negotiable — money path):** per-token daily/absolute spend caps, scopes, rate limiting, an audit row per agent-triggered spend, one-tap revoke, and a **"Connected Apps"** management screen in the LN app.
- **Phase-1 slice:** fosite AS + OTP consent + one tool (`generate_image`) + spend cap + revoke, verified by adding the connector in a real Claude account.
- **Note:** larger, security-critical; recommend after Remotion, as a dedicated reviewed build.

---

*These features are noted for post-launch implementation. Talking Avatar + Voice Cloning (Jul 2026) are BUILT & DEPLOYED (see README / PROJECT_OVERVIEW). Remotion + MCP connector are scoped and approved, not yet built.*
