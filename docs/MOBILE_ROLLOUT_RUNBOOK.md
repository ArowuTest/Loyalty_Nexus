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

> **Gap, stated honestly:** this app has **no remote feature-flag or
> force-update mechanism** today. Adding one (Firebase Remote Config is the
> cheap option, and `firebase_core` is already present) is the single highest-value
> operational addition before a public launch. Until then, a client bug that
> escapes the tester stages can only be fixed by shipping a new build and waiting
> for review.

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

- **No remote kill switch / feature flags** (see §5) — the biggest operational gap
- **No staged-rollout automation** — Play percentages are moved by hand
- **The iOS build has never run.** Budget one iteration on CocoaPods/signing at the
  first Codemagic Mac build
- **No prior production version exists**, so the upgrade path in §1 cannot be
  exercised until v1.0.1
- **No analytics events are wired** (`firebase_analytics` is installed but has zero
  imports), so funnel regressions will not be visible — only crashes will
