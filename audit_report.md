# Loyalty Nexus — Pre-Deployment Audit Report
**Date:** 2026-03-25 | **Auditor:** Skywork AI  
**Status:** Pre-production — DO NOT DEPLOY until P0 and P1 items are resolved

---

## SEVERITY LEGEND
- 🔴 **P0 — Blocker**: App will not work correctly in production — fix before deploy
- 🟠 **P1 — Critical**: Core feature broken/incomplete — fix before go-live
- 🟡 **P2 — Important**: Bad UX or missing admin control — fix in first patch
- 🟢 **P3 — Nice-to-have**: Polish/enhancement — can be post-launch

---

# SECTION 1 — AI STUDIO

## 1.1 User-Facing Issues

### 🔴 P0-S1: Generation result never shows in real-time
**Problem:** After `api.generateTool()` is called, the page only shows a toast "Check your gallery when ready" and closes the drawer. There is **no polling** of `GET /studio/generate/{id}` for status updates. The gallery only refreshes via SWR with `refreshInterval: 8000` — users wait 8 seconds and must manually navigate to Gallery tab.  
**Required:** After creating a generation, add a local `Generation` object with `status: "processing"` to the gallery immediately, then poll `GET /studio/generate/{id}` every 2 seconds until `completed` or `failed`, then update in-place.  
**File:** `frontend/src/app/studio/page.tsx` — `handleStart()` function

### 🔴 P0-S2: API route mismatch for generation status
**Problem:** Frontend calls `GET /studio/generate/${id}/status` (with `/status` suffix) but the backend only handles `GET /api/v1/studio/generate/{id}` (no `/status` suffix). Every poll would 404.  
**File:** `frontend/src/lib/api.ts` line 95:
```typescript
// WRONG:
return this.request("GET", `/studio/generate/${id}/status`);
// CORRECT:
return this.request("GET", `/studio/generate/${id}`);
```

### 🔴 P0-S3: Prizes page is 100% hardcoded mock data
**Problem:** `frontend/src/app/prizes/page.tsx` uses `MOCK_PRIZES` array — completely static. No API call to `/spin/history` or a prizes endpoint. Stats (₦3,500 won, 2 pending) are hardcoded strings.  
**Required:** Replace with `api.getSpinHistory()` and map results dynamically. Stats should be computed from real data.

### 🔴 P0-S4: Registration page uses simulated mock flow
**Problem:** `frontend/src/app/register/page.tsx` uses `setTimeout` to simulate OTP send/verify — no real API calls. Uses `useAuth.setSession('mock-jwt-7-days')`.  
**Required:** Replace with real `api.sendOTP()` / `api.verifyOTP()` calls (the hooks already exist in `useStore` and `api.ts`). The working landing page at `/` already does this correctly — the register page is a duplicate with mocked flow.  
**Fix:** Either delete `/register/page.tsx` and redirect to `/`, or wire it with the real API calls.

### 🟠 P1-S5: Subscription page is completely mocked
**Problem:** `DailySubscription.tsx` uses `setTimeout` to simulate subscription — no API call. No connection to `/api/v1/subscriptions` or payment gateway.  
**Required:** Wire to actual subscription endpoint. The admin already manages subscriptions — the user flow must trigger the backend.

### 🟠 P1-S6: Spin wheel displays hardcoded segments (ignores real prize table)
**Problem:** `frontend/src/app/spin/page.tsx` defines `DEFAULT_SEGMENTS` as a hardcoded array of 8 prizes. The API response from `GET /api/v1/spin/wheel` returns the real prize configuration (configured by admin), but this data is never used to render the wheel.  
**Required:** 
```typescript
const { data: wheelConfig } = useSWR("/spin/wheel", () => api.getWheelConfig());
// Use wheelConfig.segments to render the wheel, not DEFAULT_SEGMENTS
```

### 🟠 P1-S7: No frontend page for Daily Draw entries
**Problem:** Users have no UI to see upcoming draws, their draw tickets/entries, or past winners. The backend has `GET /api/v1/draws` and `GET /api/v1/draws/{id}/winners` fully implemented. The admin can create and manage draws. But there is no user-facing draw page at all.  
**Required:** Create `frontend/src/app/draws/page.tsx` showing upcoming draws, the user's ticket count, and recent winners.

### 🟠 P1-S8: No Digital Passport page in frontend
**Problem:** The entire Digital Passport feature (tiers, badges, QR code, streaks, PKPass download) has full backend implementation but **zero frontend UI**. There is no `/passport` page. The nav bar does not even link to it.  
**Required:** Create `frontend/src/app/passport/page.tsx` using:
- `GET /api/v1/passport` — main passport data
- `GET /api/v1/passport/badges` — earned badges
- `GET /api/v1/passport/qr` — QR code display
- `GET /api/v1/passport/share` — shareable card
- `GET /api/v1/passport/pkpass` — Apple Wallet download

### 🟡 P2-S9: Wars page is 100% hardcoded mock data
**Problem:** `frontend/src/app/wars/page.tsx` uses `MOCK_LEADERBOARD` — static data. No call to `GET /api/v1/wars/leaderboard` or `/wars/my-rank`. Prize pool (₦500,000) and timer (14 days) are hardcoded strings.

### 🟡 P2-S10: Dashboard Quick Actions has hardcoded "17 free tools" copy
**Problem:** `frontend/src/app/dashboard/page.tsx` shows `"17 free tools"` as static string. Should be fetched dynamically or at minimum driven by a config value.  
**File:** `QUICK_ACTIONS` array, line: `sub: "17 free tools"`

### 🟡 P2-S11: Studio generate call doesn't pass `tool_slug`
**Problem:** `api.generateTool(tool.id, finalPrompt)` only passes `tool_id`. The backend also accepts `tool_slug` as fallback. Fine for now, but if IDs change between environments (dev vs prod DB), it will break. Should pass both: `{ tool_id: tool.id, tool_slug: tool.slug, prompt }`.

### 🟡 P2-S12: No daily generation limit display to user
**Problem:** Backend enforces `studio_daily_gen_limit` (default 10) from `network_configs`, returns 429 when exceeded. Frontend only shows a generic toast error — user never sees "You've used 8 of 10 daily generations." The `GET /studio/chat/usage` endpoint exists but is unused.  
**Required:** Call `api.getChatUsage()` on load and display a progress indicator in the Studio header.

### 🟢 P3-S13: No "share generation" feature
**Problem:** When a user creates a cool AI photo/video, there's no way to share it. Consider adding a share button that copies the output URL or creates a sharable link.

---

## 1.2 Admin Studio Issues

### 🔴 P0-A1: Admin `GetStudioTools` returns hardcoded tool list, NOT the database
**Problem:** `admin_handler.go` `GetStudioTools()` function has a **hardcoded** slice of 17 tool names pointing to `network_configs` keys — this is completely disconnected from the `studio_tools` table that was seeded with 29 tools (including all Phase 16 tools). The admin sees a stale hard-coded list; the real table state is invisible.  
**Required:** Query `studio_tools` table directly: 
```sql
SELECT id, slug, name, category, provider, point_cost, is_active, description FROM studio_tools ORDER BY category, name
```

### 🔴 P0-A2: Admin `UpdateStudioTool` updates `network_configs`, not `studio_tools`
**Problem:** `PUT /admin/studio-tools/{key}` updates a row in `network_configs` by config key string. But point costs are stored in the `studio_tools` table. Editing from admin does nothing to the actual tool costs the user sees.  
**Required:** Update the handler to `UPDATE studio_tools SET point_cost=$1, is_active=$2 WHERE id=$3`.

### 🟠 P1-A3: Admin studio tools page is read-only view-only table
**Problem:** `admin/src/app/studio-tools/page.tsx` is a plain read-only table. No ability to:
- Toggle a tool on/off (is_active)
- Edit the point cost
- See today's usage count per tool
- Add a new tool or update its description  
**Required:** Add inline edit for `point_cost` and `is_active` toggle, with `PUT /admin/studio-tools/{id}` call.

### 🟠 P1-A4: Admin API client missing several studio tool methods
**Problem:** `admin/src/lib/api.ts` has `getStudioTools()` and `updateSubscription()` but is missing:
- `updateStudioTool(id, payload)` — needed for P1-A3
- `getUser(id)` — the backend has `GET /admin/users/{id}` but admin API doesn't expose it
- `adjustPoints(userId, delta, reason)` — backend has `POST /admin/points/adjust` but missing from client
- `getPointsStats()` / `getPointsHistory()` — backend has both, not in admin client  
**File:** `admin/src/lib/api.ts`

### 🟡 P2-A5: Admin points-config page — check what's wired
**Problem:** Need to verify `admin/src/app/points-config/page.tsx` calls real API endpoints and isn't using hardcoded data.

---

# SECTION 2 — SPIN THE WHEEL

## User-Facing

### 🔴 P0-W1: Wheel renders hardcoded segments (see P1-S6 above)
The wheel always shows the same 8 static prizes regardless of what admin configures. This is the most visible branding issue.

### 🟠 P1-W2: No spin credits display before spinning
**Problem:** User doesn't see how many spin credits they have before clicking "Spin." The wallet has `spin_credits` but it's not fetched on the spin page.  
**Required:** `useSWR("/user/wallet", ...)` on the spin page and show credits count + a disabled state when `spin_credits === 0`.

### 🟠 P1-W3: No spin history on the spin page
**Problem:** User can't see their recent spin results. Only the (mocked) prizes page shows results. The backend has `GET /api/v1/spin/history` — use it to show last 5 spins below the wheel.

### 🟡 P2-W4: Spin animation doesn't land on actual winning segment
**Problem:** The wheel animation picks a random rotation, but the outcome is determined server-side. The animation may visually "land" on "Try Again" while the API returns "₦500 Airtime." This is a serious trust issue.  
**Required:** The backend should return the `segment_index` in the spin result so the wheel can animate to the correct segment.

## Admin

### 🟠 P1-W5: Admin prize PUT uses `adminAPI.req?.()` with optional chaining
**Problem:** `spin-config/page.tsx` line: `adminAPI.req?.("PUT", ...)` — `req` is a private method and this will be `undefined` at runtime. Prize saves silently do nothing.  
**Required:** Add a public `updatePrize(id, payload)` method to `AdminAPI` class.

### 🟡 P2-W6: Admin spin config doesn't show live prize pool value
**Problem:** Admin can set the daily liability cap but can't see current total liability (how much has been won today). Should show a live widget.

---

# SECTION 3 — DAILY DRAW

## User-Facing

### 🔴 P0-D1: No user-facing draw page exists
**Problem:** Despite full backend (list draws, get winners, entry management), users cannot see any draws. The subscription page (₦20/day) has no connection to the actual draw system.

### 🟠 P1-D2: Subscription page doesn't call subscription API
See P1-S5 above — it simulates subscription locally with a `setTimeout`.

### 🟠 P1-D3: No draw entry confirmation
**Problem:** When a user subscribes, they should receive a draw entry confirmation with ticket number and draw date. Nothing of this kind exists.

### 🟡 P2-D4: Draw winner announcement not surfaced to users
**Problem:** The backend executes draws and records winners. But there's no notification trigger for "You won the daily draw!" The notification service exists but the draw execution path doesn't call it.

## Admin

### 🟡 P2-D5: Admin draws page is well-built — minor gap
The admin draws page appears complete (create draw, execute, view winners, export). Only gap: no ability to cancel a draw once created.  
**Required:** Add `DELETE /admin/draws/{id}` backend route + admin UI cancel button.

---

# SECTION 4 — DIGITAL PASSPORT

### 🔴 P0-P1: Digital Passport has NO frontend at all
The entire feature is backend-only. Users cannot see their tier, badges, streak, QR code, or download their Apple Wallet pass.  
**Impact:** This is one of the most differentiated features in the spec. It should be in the nav bar alongside Dashboard, Spin, Studio, Wars.

### 🟡 P2-P2: PKPass download requires Apple Wallet certificates
**Problem:** `GET /api/v1/passport/pkpass` generates a `.pkpass` file but requires Apple Developer signing certificates (`WWDR certificate`, `pass certificate`, `private key`). These must be configured as env vars before the endpoint works.  
**Required:** Add to env vars reference in `DEPLOYMENT.md` and ensure backend gracefully returns 501 if certificates aren't configured.

### 🟡 P2-P3: QR verification endpoint exists but no scanner UI
**Problem:** `POST /api/v1/passport/qr/verify` can validate a QR — useful for prize redemption kiosks or partner merchants. No frontend scanner exists yet.

---

# SECTION 5 — CROSS-CUTTING ISSUES

### 🔴 P0-X1: Two conflicting auth systems
**Problem:** Two auth flows exist:
1. `frontend/src/app/page.tsx` — uses `api.sendOTP()` / `api.verifyOTP()` → real API → stores in `useStore` (correct ✅)
2. `frontend/src/app/register/page.tsx` — uses `useAuth` hook with `setSession('mock-jwt')` → no real API (broken ❌)
Both pages are accessible. `/register` must be removed or replaced with the real flow.

### 🔴 P0-X2: `useAuth` hook stores token in `localStorage['nexus_token']` but `api.ts` also uses `localStorage['nexus_token']`
**Problem:** Two different systems (`useAuth` in old pages vs `useStore` in new pages) both reading/writing the same key. If both are active, logout in one system won't affect the other.  
**Required:** Consolidate to a single auth system — `useStore` is correct. Delete `useAuth.ts` once all pages are migrated.

### 🟠 P1-X3: No `NEXT_PUBLIC_API_URL` fallback for production
**Problem:** Both `api.ts` files use `process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1"`. In Vercel production, if the env var is not set, it silently falls back to localhost — every API call will fail with no error.  
**Required:** In production build, add a build-time check that `NEXT_PUBLIC_API_URL` is set.

### 🟡 P2-X4: Settings page not wired to API
**Problem:** `frontend/src/app/settings/page.tsx` — need to audit if it actually calls `PATCH /user/profile/state` or just shows UI.

### 🟡 P2-X5: No error boundary / global error handling
**Problem:** If an API call returns 500, only a generic toast message shows. No retry mechanism, no offline detection, no graceful degradation.

---

# PRIORITY BUILD ORDER

## Must fix BEFORE deploying to Render/Vercel:

| # | ID | Fix | Est. Time |
|---|---|---|---|
| 1 | P0-S2 | Fix `/status` URL mismatch in `api.ts` | 5 min |
| 2 | P0-X1 | Remove/fix mock register page | 20 min |
| 3 | P0-S3 | Wire prizes page to real API | 30 min |
| 4 | P0-W1 | Wire spin wheel to real prize config | 45 min |
| 5 | P0-A1 | Fix admin GetStudioTools to query DB | 30 min |
| 6 | P0-A2 | Fix admin UpdateStudioTool to update DB | 15 min |
| 7 | P0-S1 | Add generation polling after submit | 45 min |
| 8 | P0-D1 | Create user-facing draws page | 60 min |
| 9 | P0-P1 | Create Digital Passport page | 90 min |
| 10 | P1-W5 | Fix admin prize save (private `req` call) | 10 min |
| 11 | P1-S6 | Spin wheel uses real segment data | 45 min |
| 12 | P1-W2 | Show spin credits before spinning | 20 min |
| 13 | P1-A3 | Admin studio tools: add toggle + edit | 60 min |

## Fix in first week post-launch:
- P1-S5: Wire subscription page to real API
- P1-S7: Wire draws entry to subscription
- P2-W4: Land spin animation on correct segment
- P2-D4: Notify draw winners
- P2-X4: Audit settings page

---

# SUMMARY COUNTS

| Severity | Count |
|---|---|
| 🔴 P0 Blockers | 9 |
| 🟠 P1 Critical | 9 |
| 🟡 P2 Important | 11 |
| 🟢 P3 Nice-to-have | 1 |
| **Total** | **30** |

