# Mobile Release Setup — TestFlight (iOS) + Firebase App Distribution (Android)

**Context:** the team develops on Windows. Apple `.ipa` files can only be built on
macOS, so **Codemagic cloud Mac runners** are the route to TestFlight. Android builds
run in the same pipeline so both platforms share one Flutter pin.

Pipeline config: [`codemagic.yaml`](../codemagic.yaml)

---

## Facts you'll need

| Thing | Value |
|---|---|
| iOS bundle ID | `ng.loyaltynexus.loyaltyNexus` |
| Android package | `ng.loyaltynexus.loyalty_nexus` |
| Firebase project | `loyalty-nexus-aee8d` (number `693548905840`) |
| Firebase Android app ID | `1:693548905840:android:e5dfd61334e514dcdfcbb5` |
| Firebase iOS app ID | `1:693548905840:ios:466e5123ed4fafe2dfcbb5` |
| Flutter version (pin everywhere) | `3.44.3` |
| Min Android | API 24 (Android 7.0) |
| Min iOS | 13.0 |

> ⚠️ The iOS bundle ID and Android package deliberately differ (camelCase vs
> snake_case). Register **`ng.loyaltynexus.loyaltyNexus`** with Apple — not the
> Android one.

---

## Step 1 — Apple: register the app *(you have the Developer account)*

1. **Developer portal** → Certificates, IDs & Profiles → **Identifiers** → **+**
   - Type: App IDs → App
   - Description: `Loyalty Nexus`
   - Bundle ID: **Explicit** → `ng.loyaltynexus.loyaltyNexus`
   - Capabilities: tick **Push Notifications** (the app uses `firebase_messaging`)

2. **App Store Connect** → My Apps → **+** → New App
   - Platform: iOS · Name: `Loyalty Nexus` · Language: English (U.K.)
   - Bundle ID: pick the one you just created
   - SKU: `loyalty-nexus-ios`

3. **App Store Connect API key** (this is what lets Codemagic upload for you)
   - Users and Access → **Integrations** → App Store Connect API → **+**
   - Access: **App Manager**
   - Download the **`.p8` file** (one download only — keep it safe)
   - Note the **Issuer ID** and **Key ID**

---

## Step 2 — Codemagic setup

1. Sign up at [codemagic.io](https://codemagic.io) with GitHub, authorise the
   `ArowuTest/Loyalty_Nexus` repo. *(Free tier: 500 build-min/month.)*

2. **Team → Integrations → App Store Connect → Add key**
   - Name it exactly **`loyalty-nexus-asc`** (matches `codemagic.yaml`)
   - Paste Issuer ID, Key ID, and upload the `.p8`

3. **Environment variables → group `appstore_credentials`**
   - `APP_STORE_APPLE_ID` = the numeric Apple ID from the App Store Connect app page
     (App Information → General Information)

4. **Code signing → Android keystore**
   - Upload your existing release keystore, reference name **`loyalty_nexus_keystore`**
   - *(The same keystore already lives in the GitHub secret `ANDROID_KEYSTORE_BASE64` —
     use that exact file so app signatures stay consistent, or existing testers cannot
     upgrade in place.)*

5. **Environment variables → group `firebase_credentials`**
   - `FIREBASE_SERVICE_ACCOUNT` = contents of a Firebase service-account JSON
     - Firebase console → Project settings → Service accounts → Generate new private key
     - Mark it **secure**

---

## Step 3 — Firebase App Distribution (Android testers)

1. Firebase console → **App Distribution** → get started
2. **Testers & Groups** → create a group named exactly **`testers`**
   (matches `codemagic.yaml`)
3. Add tester emails

---

## Step 4 — Ship a build

Builds trigger on a **version tag**:

```bash
git tag v1.0.0
git push origin v1.0.0
```

That fires both workflows:
- **iOS** → builds `.ipa` → uploads to **TestFlight** → testers get it via the
  TestFlight app
- **Android** → builds signed APK + AAB → **Firebase App Distribution** → testers
  get an email link

You can also run either workflow manually from the Codemagic UI.

---

## Step 5 — First TestFlight run: expect a review pause

The **first** build of a new app goes through Apple's **Beta App Review** (usually
under 24h). Later builds to *internal* testers skip it. Add internal testers under
TestFlight → Internal Testing to iterate fastest.

---

## Optional — iOS simulator in a browser (Appetize)

You already have `mobile-apk.yml` uploading Android builds to **Appetize.io**.
Appetize also hosts **iOS** simulators. To get a shareable iOS simulator link, add a
step to the `ios-testflight` workflow that builds a **simulator** `.app`:

```bash
cd mobile
flutter build ios --simulator --debug
cd build/ios/iphonesimulator
zip -r Runner.zip Runner.app
# then POST Runner.zip to Appetize with platform=ios
```

That gives stakeholders a clickable iOS demo without installing anything — useful
for demos, but **not** a substitute for TestFlight on real devices.

---

## Keeping versions in sync ⚠️

Flutter `3.44.3` is pinned in **four** places. Change them together:

- `codemagic.yaml`
- `.github/workflows/mobile-apk.yml`
- `.github/workflows/ci.yml`
- local dev SDK (`C:\flutter`)

A mismatch is how the `CardTheme` → `CardThemeData` break slipped in: the app
compiled on one version and failed on another.

---

## Known state / caveats

- ✅ Android release manifest fixed — `INTERNET` was previously declared **only** in
  the debug manifest, so a release build would have had no network access at all.
- ✅ iOS store blockers cleared — privacy manifest, usage strings, iOS 13 target.
- ⚠️ **The iOS build has never actually run.** Everything iOS-side is correct by
  inspection but unverified until the first Codemagic Mac build. Expect to iterate
  once on CocoaPods/signing — that is normal for a first iOS build.
- ⚠️ `local_auth` and `permission_handler` are dependencies that **no code imports**.
  Decide whether biometric login was intended, or drop them to slim the build.
