const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// ─── Passport Types ───────────────────────────────────────────────────────────

export interface BadgeDefinition {
  key: string;
  name: string;
  description: string;
  icon: string;
}

export interface PassportData {
  user_id: string;
  tier: string;
  streak_count: number;
  lifetime_points: number;
  badges: BadgeDefinition[];
  next_tier: string;
  points_to_next_tier: number;
}

export interface WalletPassURLs {
  apple_pkpass_url: string;
  google_wallet_url: string;
  apple_signed: boolean;
  google_configured: boolean;
}

export interface QRData {
  /** Raw base64url-encoded QR payload returned by the backend.
   *  The frontend renders this into a QR image using the qrcode library. */
  qr_payload: string;
  expires_in: number;
  format: string;
}

export interface PassportEvent {
  id: string;
  user_id: string;
  event_type: "tier_upgrade" | "badge_earned" | "streak_milestone" | "qr_scanned" | string;
  details: Record<string, unknown>;
  created_at: string;
}

// ─── Bonus Pulse Award Types ──────────────────────────────────────
export interface BonusPulseAward {
  id: string;
  points: number;
  campaign: string;
  note: string;
  awarded_by_name: string;
  created_at: string;
}

export interface RegionalStat {
  state: string;
  total_points: number;
  active_members: number;
  rank: number;
}

// ─── API Client ───────────────────────────────────────────────────────────────

class APIClient {
  private token: string | null = null;

  setToken(token: string) {
    this.token = token;
    if (typeof window !== "undefined") {
      localStorage.setItem("nexus_token", token);
    }
  }

  getToken(): string | null {
    if (this.token) return this.token;
    if (typeof window !== "undefined") {
      this.token = localStorage.getItem("nexus_token");
    }
    return this.token;
  }

  clearToken() {
    this.token = null;
    if (typeof window !== "undefined") {
      localStorage.removeItem("nexus_token");
    }
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    isPublic = false
  ): Promise<T> {
    const headers: HeadersInit = { "Content-Type": "application/json" };
    const token = this.getToken();
    if (token && !isPublic) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const resp = await fetch(`${BASE_URL}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    // Only treat 401 as a session expiry for authenticated (non-public) requests.
    // Public endpoints like OTP verify return 401 for invalid codes — those should
    // surface the actual error message, not trigger a forced logout.
    if (resp.status === 401 && !isPublic) {
      this.clearToken();
      // Dispatch a soft event so the app can handle session expiry gracefully
      // (e.g., show the login modal) without a hard page reload that causes flicker.
      // Components listening to this event can redirect or show the auth modal.
      if (typeof window !== "undefined") {
        window.dispatchEvent(new CustomEvent("nexus:session-expired"));
      }
      throw new Error("Session expired");
    }

    const data = await resp.json();
    if (!resp.ok) throw new Error(data.error || "Request failed");
    return data as T;
  }

  // ── Auth ──────────────────────────────────────────────────────────────────
  sendOTP(phone: string, purpose = "login") {
    return this.request("POST", "/auth/otp/send", { phone_number: phone, purpose }, true);
  }
  verifyOTP(phone: string, code: string, purpose = "login") {
    return this.request<{ token: string; is_new_user: boolean }>(
      "POST", "/auth/otp/verify", { phone_number: phone, code, purpose }, true
    );
  }

  // ── User ──────────────────────────────────────────────────────────────────
  getProfile() { return this.request("GET", "/user/profile"); }
  updateProfile(data: { display_name?: string | null; email?: string | null }) {
    return this.request<{ display_name?: string; email?: string }>("PATCH", "/user/profile", data);
  }
  getWallet() { return this.request("GET", "/user/wallet"); }
  getTransactions() { return this.request("GET", "/user/transactions"); }
  requestMoMoLink(momoNumber: string) {
    return this.request("POST", "/user/momo/request", { momo_number: momoNumber });
  }
  verifyMoMo(momoNumber: string) {
    return this.request("POST", "/user/momo/verify", { momo_number: momoNumber });
  }
  /** @deprecated Use getPassport() instead */
  getPassportURLs() { return this.request("GET", "/user/passport"); }
  /** Returns the user's bonus Pulse Point awards (total + recent history) */
  getBonusPulseAwards() {
    return this.request<{ total_bonus: number; awards: BonusPulseAward[] }>("GET", "/user/bonus-pulse");
  }

  // ── Passport ──────────────────────────────────────────────────────────────
  /** Returns the full passport profile: tier, streak, badges, lifetime points */
  getPassport() {
    return this.request<PassportData>("GET", "/passport/profile");
  }
  /** Returns the raw QR payload; the frontend renders it into a QR image client-side */
  getPassportQR() {
    return this.request<QRData>("GET", "/passport/qr");
  }
  /** Returns the user's passport event history (tier changes, badge earns, QR scans, etc.) */
  getPassportEvents(limit = 30) {
    return this.request<{ events: PassportEvent[] }>("GET", `/passport/events?limit=${limit}`);
  }
  /** Returns Apple Wallet .pkpass download URL and Google Wallet save URL */
  getWalletPassURLs() {
    return this.request<WalletPassURLs>("GET", "/passport/wallet-urls");
  }
  /** Returns the direct URL to download the Apple .pkpass file (with auth token) */
  getApplePKPassURL(): string {
    const token = this.getToken();
    return `${BASE_URL}/passport/pkpass${token ? `?token=${token}` : ""}`;
  }

  // ── Spin ──────────────────────────────────────────────────────────────────
  getWheelConfig() { return this.request("GET", "/spin/wheel"); }
  playSpin() { return this.request("POST", "/spin/play", {}); }
  getSpinHistory() { return this.request("GET", "/spin/history"); }
  /** Returns eligibility + tier progress data for the DailySpinProgress component.
   *  Includes: current_tier_name, today_amount_naira, progress_percent,
   *  available_spins, spins_used_today, max_spins_today, and upgrade nudge fields. */
  getSpinEligibility() {
    return this.request<{
      eligible: boolean;
      available_spins: number;
      spins_used_today: number;
      max_spins_today: number;
      spin_credits: number;
      message: string;
      current_tier_name: string;
      today_amount_naira: number;
      progress_percent: number;
      trigger_naira?: number;
      next_tier_name?: string;
      next_tier_min_amount?: number;
      amount_to_next_tier?: number;
      next_tier_spins?: number;
    }>("GET", "/spin/eligibility");
  }

  // ── Studio ────────────────────────────────────────────────────────────────
  getStudioTools() { return this.request("GET", "/studio/tools"); }
  sendChat(
    message: string,
    sessionId?: string,
    toolSlug?: string,
    imageURL?: string,
    documentURL?: string,
    fileURL?: string,
    linkURL?: string,
    fileName?: string,
  ) {
    return this.request("POST", "/studio/chat", {
      message,
      session_id:   sessionId,
      tool_slug:    toolSlug,
      ...(imageURL    ? { image_url:    imageURL    } : {}),
      ...(documentURL ? { document_url: documentURL } : {}),
      ...(fileURL     ? { file_url:     fileURL     } : {}),
      ...(linkURL     ? { link_url:     linkURL     } : {}),
      ...(fileName    ? { file_name:    fileName    } : {}),
    });
  }
  generateTool(
    toolId: string,
    payload: {
      prompt: string;
      tool_slug?: string;
      aspect_ratio?: string;
      duration?: number;
      voice_id?: string;
      language?: string;
      vocals?: boolean;
      lyrics?: string;
      style_tags?: string[];
      negative_prompt?: string;
      image_url?: string;
      document_url?: string;  // FEAT-01: pre-uploaded PDF/TXT CDN URL for knowledge tools
      extra_params?: Record<string, unknown>;
    }
  ) {
    return this.request<{ generation_id: string; status: string }>(
      "POST", "/studio/generate", { tool_id: toolId, ...payload }
    );
  }

  /** Convenience: generate by slug (no need to look up UUID first) */
  generateBySlug(
    toolSlug: string,
    payload: { prompt: string; language?: string; image_url?: string; extra_params?: Record<string, unknown> }
  ) {
    return this.request<{ generation_id: string; status: string }>(
      "POST", "/studio/generate", { tool_slug: toolSlug, ...payload }
    );
  }
  getGenerationStatus(id: string) {
    return this.request("GET", `/studio/generate/${id}`);
  }
  buildWebsite(payload: {
    site_type: string;
    fields: Record<string, string>;
    photos: Array<{ base64: string; caption: string }>;
  }) {
    return this.request("POST", "/studio/website", payload);
  }
  getGallery() { return this.request("GET", "/studio/gallery"); }

  /** Upload an audio or image file to cloud storage.
   *  Returns { url: string } — pass the url to generateTool() as prompt (transcribe)
   *  or image_url (image-editor / video-animator).
   */
  async uploadAsset(file: File): Promise<{ url: string; key: string }> {
    const form = new FormData();
    form.append("file", file);
    const token = this.getToken();
    const resp = await fetch(`${BASE_URL}/studio/upload`, {
      method: "POST",
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({})) as { error?: string };
      throw new Error(err.error ?? `Upload failed (${resp.status})`);
    }
    return resp.json() as Promise<{ url: string; key: string }>;
  }

  // ── Draws (user-facing) ───────────────────────────────────────────────────
  getDraws() { return this.request("GET", "/draws"); }
  getDrawWinners(id: string) { return this.request("GET", `/draws/${id}/winners`); }

  // ── Regional Wars ─────────────────────────────────────────────────────────
  getWarsLeaderboard(limit = 37) {
    return this.request<{
      leaderboard: Array<{
        state: string; total_points: number; active_members: number;
        rank: number; prize_kobo: number; period: string;
      }>;
      count: number;
      period: string;
    }>("GET", `/wars/leaderboard?limit=${limit}`);
  }
  getMyWarRank() {
    return this.request<{
      ranked: boolean;
      entry?: { state: string; total_points: number; rank: number; prize_kobo: number };
      message?: string;
    }>("GET", "/wars/my-rank");
  }
  getWarsHistory(limit = 12) {
    return this.request<{
      wars: Array<{
        id: string; period: string; status: string;
        total_prize_kobo: number; starts_at: string; ends_at: string;
      }>;
      count: number;
    }>("GET", `/wars/history?limit=${limit}`);
  }
  getWarWinners(period: string) {
    return this.request("GET", `/wars/${period}/winners`);
  }

  // ── Prizes / Claims ──────────────────────────────────────────────────────
  getMyWins() {
    return this.request<Array<{
      id: string;
      prize_type: string;
      prize_value: number;
      prize_label: string;
      fulfillment_status: string;
      claim_status: string;
      created_at: string;
      expires_at: string;
      needs_momo_setup?: boolean;
    }>>("GET", "/spin/wins");
  }
  claimPrize(id: string, payload: { momo_number?: string; bank_account_number?: string; bank_account_name?: string; bank_name?: string } = {}) {
    return this.request<{ id: string; claim_status: string; fulfillment_status: string }>("POST", `/spin/wins/${id}/claim`, payload);
  }

  // ── Chat history restore (BUG-05 fix) ──────────────────────────────────
  getChatHistory(mode: string) {
    return this.request<{
      session_id: string;
      tool_slug: string;
      messages: { role: string; content: string; created_at: string }[];
    }>("GET", `/studio/chat/history?mode=${mode}`);
  }

  // ── Chat usage quota ──────────────────────────────────────────────────────
  getChatUsage() { return this.request("GET", "/studio/chat/usage"); }

  // ── Notifications ─────────────────────────────────────────────────────────
  getNotifications(cursor?: string) {
    return this.request("GET", "/notifications" + (cursor ? `?cursor=${cursor}` : ""));
  }
  markNotificationRead(id: string) {
    return this.request("POST", `/notifications/${id}/read`, {});
  }
  markAllNotificationsRead() {
    return this.request("POST", "/notifications/read-all", {});
  }
  registerPushToken(token: string, platform: string) {
    return this.request("POST", "/notifications/push-token", { token, platform });
  }

  // ── Dispute & Session ─────────────────────────────────────────────────────
  disputeGeneration(genId: string): Promise<{ message: string; refunded: boolean }> {
    return this.request("POST", `/studio/generate/${genId}/dispute`, {});
  }
  getSessionUsage(): Promise<{
    active: boolean;
    session_id?: string;
    total_pts_used: number;
    generation_count: number;
    started_at?: string;
    last_active_at?: string;
  }> {
    return this.request("GET", "/studio/session");
  }
}

export const api = new APIClient();
export default api;
