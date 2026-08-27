# Deep Links — SMS and web → app

**Why this matters here:** Loyalty Nexus is a telco loyalty product driven by SMS
campaigns. Before this the manifest had only `MAIN`/`LAUNCHER`, so **nothing
outside the app could open a screen inside it** — a link in a campaign SMS could
only open a browser. That is a functional gap, not polish.

## What is wired in the app

| Platform | Mechanism | Status |
|---|---|---|
| Android | App Links — `https://loyaltynexus.ng/app/...` (`autoVerify="true"`) | ⚠️ needs `assetlinks.json` hosted |
| Android | Custom scheme — `loyaltynexus://...` | ✅ works with no hosting |
| iOS | Universal Links — `applinks:loyaltynexus.ng` entitlement | ⚠️ needs `apple-app-site-association` hosted |
| iOS | Custom scheme — `loyaltynexus://...` | ✅ works with no hosting |
| Routing | `FlutterDeepLinkingEnabled` + GoRouter | ✅ GoRouter resolves the path automatically |

> The **custom scheme works immediately** and needs nothing hosted. Use it for the
> first tester round. The https App/Universal Links are strictly nicer (they open
> the app from a normal web link) but they **fail silently** until the two files
> below are live — no error, the link just opens a browser.

## Two files you must host on `loyaltynexus.ng`

### 1. Android — `https://loyaltynexus.ng/.well-known/assetlinks.json`

```json
[{
  "relation": ["delegate_permission/common.handle_all_urls"],
  "target": {
    "namespace": "android_app",
    "package_name": "ng.loyaltynexus.loyalty_nexus",
    "sha256_cert_fingerprints": ["<SHA-256 OF YOUR RELEASE SIGNING CERT>"]
  }
}]
```

Get the fingerprint from the **same keystore CI signs with**:

```bash
keytool -list -v -keystore loyalty-nexus-release.jks -alias <your-alias> | grep SHA256
```

> ⚠️ It must be the **release** certificate. A debug fingerprint verifies only
> debug builds, and the ones testers install will silently fail to verify.
> If you later enable **Play App Signing**, Play re-signs your upload — take the
> fingerprint from *Play Console → Setup → App integrity* instead, or verification
> breaks the day you ship to Play.

### 2. iOS — `https://loyaltynexus.ng/.well-known/apple-app-site-association`

```json
{
  "applinks": {
    "apps": [],
    "details": [{
      "appID": "<TEAMID>.ng.loyaltynexus.loyaltyNexus",
      "paths": ["/app/*"]
    }]
  }
}
```

Requirements Apple enforces strictly:
- Served over **HTTPS** with **no redirect**
- `Content-Type: application/json`
- **No `.json` extension** on the filename
- `<TEAMID>` is your Apple Developer Team ID (Membership page)

## Serving them from the Next.js frontend

Both belong at `/.well-known/`, which `frontend/public/` serves directly — drop the
files in `frontend/public/.well-known/`. The AASA file has no extension, so add a
header rule in `next.config.js` to force its content type:

```js
async headers() {
  return [{
    source: '/.well-known/apple-app-site-association',
    headers: [{ key: 'Content-Type', value: 'application/json' }],
  }];
}
```

## Link format

```
https://loyaltynexus.ng/app/spin        → /spin
https://loyaltynexus.ng/app/prizes      → /prizes
loyaltynexus://spin                     → /spin   (works with no hosting)
```

The GoRouter `errorBuilder` added earlier catches unknown paths, so a stale or
mistyped campaign link shows a friendly "we could not open that link" screen with a
Home button — not go_router's raw exception page.

## Verifying

**Android** (device attached):
```bash
adb shell am start -W -a android.intent.action.VIEW -d "loyaltynexus://spin"
adb shell am start -W -a android.intent.action.VIEW -d "https://loyaltynexus.ng/app/spin"
```
Check verification status: `adb shell pm get-app-links ng.loyaltynexus.loyalty_nexus`
— you want `verified`, not `legacy_failure`.

**iOS:** paste the link into Notes or iMessage and tap it. Safari's address bar does
*not* trigger Universal Links — a real known gotcha that makes people think it is
broken when it is not.

Apple caches AASA via its CDN, so allow time (or delete and reinstall the app)
after first publishing the file.

## Honest status

The **app side is complete and will ship in the next build**. The **hosting side is
not done** and is not something I can do from here — it needs the two files
deployed to `loyaltynexus.ng`, and the Android one needs a fingerprint that only
comes from your keystore.

Until then: **custom-scheme links work, https links open a browser.** Nothing
breaks either way.
