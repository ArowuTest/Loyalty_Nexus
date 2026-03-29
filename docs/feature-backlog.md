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

---

*These features are noted for post-launch implementation. Current sprint focus: E2E testing and production data seeding.*
