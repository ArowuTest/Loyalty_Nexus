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
  qr_data_url: string;
  qr_payload: string;
}

// ─── Bonus Pulse Award Types ─────────────────────────────────────────────────
export interface BonusPulseAward {
  id: string;
  points: number;
  campaign: string;
  note: string;
  created_at: string;
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

    if (resp.status === 401) {
      this.clearToken();
      window.location.href = "/";
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
  /** Returns a QR code data URL and the raw payload for display */
  getPassportQR() {
    return this.request<QRData>("GET", "/passport/qr");
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

  // ── Studio ────────────────────────────────────────────────────────────────
  getStudioTools() { return this.request("GET", "/studio/tools"); }
  sendChat(message: string, sessionId?: string, toolSlug?: string) {
    return this.request("POST", "/studio/chat", {
      message,
      session_id: sessionId,
      tool_slug:  toolSlug,
    });
  }
  generateTool(
    toolId: string,
    payload: {
      prompt: string;
      aspect_ratio?: string;
      duration?: number;
      voice_id?: string;
      language?: string;
      vocals?: boolean;
      lyrics?: string;
      style_tags?: string[];
      negative_prompt?: string;
      image_url?: string;
      extra_params?: Record<string, unknown>;
    }
  ) {
    return this.request<{ generation_id: string; status: string }>(
      "POST", "/studio/generate", { tool_id: toolId, ...payload }
    );
  }
  getGenerationStatus(id: string) {
    return this.request("GET", `/studio/generate/${id}`);
  }
  getGallery() { return this.request("GET", "/studio/gallery"); }

  // ── Draws (user-facing) ───────────────────────────────────────────────────
  getDraws() { return this.request("GET", "/draws"); }
  getDrawWinners(id: string) { return this.request("GET", `/draws/${id}/winners`); }

  // ── Chat usage quota ──────────────────────────────────────────────────────
  getChatUsage() { return this.request("GET", "/studio/chat/usage"); }

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
