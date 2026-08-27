# Mobile Rollout & Rollback Runbook

**Owner:** engineering lead · **Applies to:** Loyalty Nexus iOS + Android
**Last reviewed:** 27 Aug 2026

This closes the last unmet release-readiness gate: *"Define staged rollout,
monitoring thresholds, rollback/hotfix path, and owner."* Everything else in the
audit is code; this is the operational half.

---

## 0. Release identity

| | |
|---|---|
| iOS bundle | `ng.loyaltynexus.loyaltyNexus` |
| Android package | `ng.loyaltynexus.loyalty_nexus` |
| Firebase project | `loyalty-nexus-aee8d` |
| Version source of truth | the **git tag** (`v1.2.0` → `--build-name=1.2.0`) |
| Build number | CI/Codemagic monotonic counter → `--build-number` |

> **Never reuse or lower a build number.** TestFlight rejects duplicates, and Play
> rejects a versionCode ≤ the last uploaded.

---

## 1. Pre-flight — before you tag

Run through this. It is short because CI now enforces most of it.

- [ ] `main` is green on **Mobile — Release Verification** (release APK **and** AAB build, INTERNET permission asserted, not debug-signed)
- [ ] The change has a **rollback story**: is it purely client-side, or does it depend on a backend deploy?
- [ ] If it needs a backend change, **the backend ships first** and is backward-compatible with the *currently installed* app version
- [ ] Crashlytics is receiving events from the previous build (proves telemetry is alive *before* you need it)
- [ ] Release notes drafted (TestFlight "What to Test" + Firebase release notes)

### The upgrade-path check — the one people skip
Install the **previous** production/tester build, then install the new one **over
the top** (do not uninstall). Confirm:
- [ ] it launches
- [ ] the user is **still logged in** (token survives in secure storage)
- [ ] cached data from the old version does not crash the new one
- [ ] any changed `SharedPreferences` / cache key shape is handled, not assumed

> A migration that only works from a clean install is the classic way to brick
> every existing tester at once.

---

## 2. Ship

```bash
git tag v1.0.0
git push origin v1.0.0
```

That fires both Codemagic workflows:

| Platform | Path | Lands in |
|---|---|---|
| iOS | cloud Mac → `.ipa` → App Store Connect | **TestFlight** |
| Android | signed APK + AAB | **Firebase App Distribution** → group `testers` |

**First iOS build only:** Apple runs **Beta App Review** (usually < 24h). Later
builds to *internal* testers skip it. Add internal testers first for the fastest loop.

---

## 3. Staged rollout

Do not hand a build to everyone at once, even in testing.

| Stage | Audience | Soak | Gate to advance |
|---|---|---|---|
| 1 | Internal (you + 2-3) | 2-4 h | launches, login works, no new crash signature |
| 2 | Full tester group | 24 h | crash-free sessions **≥ 99%**, no P1 reports |
| 3 | Play internal → closed track | 48 h | metrics hold at higher volume |
| 4 | Production, **staged %** | — | Play staged rollout 10% → 50% → 100% |

On Play, always use a **staged** production rollout. It is the only mechanism that
lets you halt a bad release without shipping a new binary.

---

## 4. Monitoring thresholds

Watch in **Firebase Crashlytics** (now wired — see `main.dart`) for the first 24h.

| Signal | Green | Investigate | **Halt / roll back** |
|---|---|---|---|
| Crash-free sessions | ≥ 99.5% | 99.0-99.5% | **< 99%** |
| Crash-free users | ≥ 99.0% | 98.0-99.0% | **< 98%** |
| New crash signature in top 5 | none | any | affecting **> 1%** of sessions |
| Login success rate | at baseline | −5% | **−15%** |
| ANR rate (Android) | < 0.3% | 0.3-0.5% | **> 0.5%** |

Also check the **backend** side for the same window: API 5xx rate, OTP delivery
success (Termii), and points-ledger anomalies. A mobile release can surface a
server bug that only the new client path triggers.

---

## 5. Rollback

**The core constraint: you cannot un-ship a mobile binary.** Users who installed
it keep it. Every rollback is really "stop the spread + push a fix forward".

### Android
1. **Play staged rollout** → **Halt rollout** in the console. Stops new installs immediately.
2. Firebase App Distribution → delete the bad release so testers stop installing it.
3. If already at 100%: re-upload the **previous** AAB with a **higher versionCode**
   (Play will not accept the old one as-is), then resume a staged rollout.

### iOS
1. TestFlight → **expire** the bad build. Testers can no longer install it.
2. App Store (if live) → **Remove from Sale** or halt phased release in App Store
   Connect. There is no "revert to previous binary" — you must submit a fix.

### The real lever: server-side
Most incidents are faster to kill from the backend than from the store.
- Feature-flag or disable the offending endpoint
- Return a maintenance response for the affected route
- If the client is hard-broken, a forced-update prompt is the last resort

### The remote kill switch — BUILT, but you must arm it

`lib/src/core/remote_config/` implements force-update, maintenance mode and
per-feature kill switches over Firebase Remote Config. **It ships permissive**: until
the parameters below exist in the console, the in-app defaults apply and nothing is
ever blocked. Create them *before* the first tester build, so the mechanism is live
when you need it rather than being built under pressure.

**Firebase console → Remote Config → add these parameters:**

| Parameter | Type | Initial value | Effect |
|---|---|---|---|
| `minimum_supported_version` | String | `0.0.0` | Below this → **hard block** + "Update now". `0.0.0` blocks nobody. |
| `latest_version` | String | `0.0.0` | Below this → dismissible banner. Never blocks. |
| `maintenance_mode` | Boolean | `false` | `true` → **hard block** with a message. |
| `maintenance_message` | String | *(see defaults)* | Shown during maintenance. |
| `update_message` | String | *(see defaults)* | Shown on the update screen/banner. |
| `killed_features` | String (JSON) | `[]` | e.g. `["spin"]` disables ONE flow without blocking the app. |
| `android_store_url` | String | Play listing URL | "Update now" target on Android. |
| `ios_store_url` | String | App Store URL | "Update now" target on iOS. **Placeholder until the App Store ID exists — set it after Step 1 of the setup doc.** |

**How to actually use it in an incident**

| Situation | Action | Blast radius |
|---|---|---|
| One feature broken (e.g. spin) | `killed_features` → `["spin"]` | That flow only — everything else keeps working |
| Backend down / migrating | `maintenance_mode` → `true` | Everyone, with an explanation |
| Shipped build is dangerous | `minimum_supported_version` → the FIXED version | Everyone on older builds, pushed to update |
| Nudge onto a new build | `latest_version` → the new version | Banner only, nobody blocked |

Propagation is **≤ 15 minutes** (the fetch interval), and the app re-evaluates on
resume — so a switch flipped while someone has the app backgrounded takes effect
without them relaunching.

> ⚠️ **`minimum_supported_version` is the loaded gun.** Setting it to a version
> nobody has yet locks out **every user**. Verified by test that equal versions do
> NOT count as older, so setting it to the currently-shipped version is safe — but
> setting it *higher* than what is live in the store blocks everyone with no way
> back except shipping. Always confirm the target build is downloadable first.

**Test it before you rely on it.** During Stage 1 of the rollout, flip
`maintenance_mode` on and off with a real device. A kill switch nobody has ever
pulled is not a kill switch.

---

## 6. Hotfix path

1. Branch from the **tag**, not from `main` (main may contain unshipped work):
   `git checkout -b hotfix/v1.0.1 v1.0.0`
2. Smallest possible change. Nothing else rides along.
3. Push the branch → CI runs the same release verification.
4. Tag `v1.0.1` → normal pipeline.
5. Merge the hotfix back into `main`.
6. Restart the rollout at **Stage 1**. A hotfix is not exempt from staging — it is
   written under pressure and is exactly the kind of change that regresses.

---

## 7. Ownership

| Area | Owner |
|---|---|
| Cut the release / tag | engineering lead |
| Crashlytics + threshold watch (24h) | engineering lead |
| Go / no-go to advance a stage | product owner |
| Halt / rollback decision | either, unilaterally — **bias to halting** |
| Backend incident response | backend on-call |

**Anyone may halt a rollout without asking.** Halting is cheap and reversible;
shipping a broken build to everyone is neither.

---

## 8. Known limitations at first release

State these openly rather than discovering them mid-incident.

- ✅ **Remote kill switch now exists** (§5) — but it is INERT until the Remote
  Config parameters are created in the Firebase console, and untested until you
  have pulled it once on a real device
- **No staged-rollout automation** — Play percentages are moved by hand
- **The iOS build has never run.** Budget one iteration on CocoaPods/signing at the
  first Codemagic Mac build
- **No prior production version exists**, so the upgrade path in §1 cannot be
  exercised until v1.0.1
- **No analytics events are wired** (`firebase_analytics` is installed but has zero
  imports), so funnel regressions will not be visible — only crashes will
