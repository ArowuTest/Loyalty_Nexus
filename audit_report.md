# Loyalty Nexus Codebase Audit Report

This report details the findings of a comprehensive codebase review, focusing on hardcoded values, mock data, incomplete implementations, and overall production readiness.

## 1. Backend Findings

### 1.1 Incomplete Implementations & Mocks
*   **`subscription_service.go`**: The `Subscribe` method explicitly states `// Initial Billing (Mocking DCB/Paystack call)` and does not integrate with any actual payment gateway. The `ProcessRecurringBilling` method is a completely empty stub that just returns `nil`. This is a major missing feature.
*   **`passport_wallet_service.go`**: 
    *   Contains hardcoded fallback values: `APP_BASE_URL` falls back to `https://app.loyaltynexus.ng`, `USSD_SHORTCODE` falls back to `*384#`.
    *   `passTypeID` defaults to `pass.ng.loyaltynexus.passport` and `teamID` to `XXXXXXXXXX` unless signer config exists.
    *   `generatePassAuthToken` returns either a `userID` or a simple daily token string rather than a proper cryptographic signature (JWT/HMAC), despite comments suggesting it should.
    *   It generates an unsigned dev-mode pkpass.
*   **`user_handler.go`**: The `GetPassportURLs` handler was previously returning a "coming soon" message. (Note: This was partially wired up during the audit, but the underlying service still has the issues mentioned above).

### 1.2 Regional Wars (Backend)
*   **`wars_service.go`**: Winner identification at the state level is implemented. However, there is no logic to select individual cash winners within a winning region. The prize distribution currently only awards bonus pulse points to all active users in the winning states. Actual cash prize disbursement is not implemented.

## 2. Admin Panel Findings

### 2.1 Incomplete UI & Missing Features
*   **`regional-wars/page.tsx`**: The admin page for regional wars is very basic. It only displays the leaderboard. It lacks controls to edit the prize pool, manually resolve a war, view the actual winners, or trigger/manage the prize payouts.
*   **`health/page.tsx`**: The health page was previously using a hardcoded mock fallback if the API call failed. (Note: This was fixed during the audit to show an error state instead, but the backend `GetHealth` endpoint needs to ensure it returns real, dynamic data matching the expected `HealthReport` structure).

## 3. Frontend (User App) Findings

### 3.1 Hardcoded Data
*   **`wars/page.tsx`**: While the leaderboard data is now fetched from the API, the prize pool banner contains hardcoded text: `₦500,000` and `Resets in 14 days`. This should be dynamic based on the active war's configuration.

## Summary of Required Fixes

1.  **Subscriptions**: Implement real payment gateway integration (Paystack/DCB) for subscriptions and complete the recurring billing worker.
2.  **Passport Wallet**: Remove hardcoded team IDs, implement proper JWT/HMAC for pass auth tokens, and ensure production-ready signed pkpass generation.
3.  **Regional Wars**: 
    *   Implement the secondary draw logic to select individual winners from the winning states.
    *   Integrate cash prize disbursement (MoMo/VTPass).
    *   Build out the admin UI for managing wars (resolve, view winners, payout).
    *   Make the frontend wars page fully dynamic (prize pool, countdown).
4.  **Health Endpoint**: Ensure the backend `/admin/health` endpoint returns accurate, real-time metrics.
