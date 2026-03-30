const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

export interface PassportStats {
  total_passports: number;
  apple_wallet_downloads: number;
  google_wallet_saves: number;
  qr_scans_today: number;
  active_apple_installs: number;
  active_google_installs: number;
  total_active_installs: number;
  removal_rate_pct: number;
  device_breakdown: { device_type: string; count: number }[];
  tier_breakdown: { tier: string; count: number }[];
  top_badge_earners: { user_id: string; phone: string; badge_count: number; tier: string }[];
}

export interface GhostNudgeLog {
  id: string;
  user_id: string;
  phone_number: string;
  nudge_type: string;
  streak_count: number;
  sent_at: string;
  delivered: boolean;
}

export interface USSDSession {
  id: string;
  phone_number: string;
  session_id: string;
  current_menu: string;
  started_at: string;
  last_active: string;
  is_active: boolean;
  step_count: number;
}

class AdminAPI {
  private token: string | null = null;
  setToken(t: string) { this.token = t; typeof window !== "undefined" && localStorage.setItem("nexus_admin_token", t); }
  getToken(): string | null {
    if (this.token) return this.token;
    if (typeof window !== "undefined") this.token = localStorage.getItem("nexus_admin_token");
    return this.token;
  }
  clearToken() { this.token = null; typeof window !== "undefined" && localStorage.removeItem("nexus_admin_token"); }
  async req<T>(method: string, path: string, body?: unknown): Promise<T> {
    const resp = await fetch(`${BASE_URL}${path}`, {
      method,
      headers: { "Content-Type": "application/json", ...(this.getToken() ? { Authorization: `Bearer ${this.getToken()}` } : {}) },
      body: body ? JSON.stringify(body) : undefined,
    });
    if (resp.status === 401) { this.clearToken(); window.location.href = "/login"; throw new Error("Unauthorized"); }
    const data = await resp.json();
    if (!resp.ok) throw new Error((data as {error?: string}).error || "Request failed");
    return data as T;
  }
  getDashboard()   { return this.req<DashboardStats>("GET", "/admin/dashboard"); }
  getConfig()      { return this.req<{ configs: ConfigEntry[] }>("GET", "/admin/config"); }
  updateConfig(key: string, value: string) { return this.req("PUT", `/admin/config/${encodeURIComponent(key)}`, { value }); }
  getPrizePool()   { return this.req<{ prizes: Prize[] }>("GET", "/admin/prize-pool"); }
  getPrizeSummary() { return this.req<PrizeSummary>("GET", "/admin/prizes/summary"); }
  updatePrize(id: string, payload: Partial<Prize>) { return this.req<Prize>("PUT", `/admin/prizes/${id}`, payload); }
  createPrize(payload: Omit<Prize,"id">) { return this.req<Prize>("POST", "/admin/prizes", payload); }
  deletePrize(id: string) { return this.req<void>("DELETE", `/admin/prizes/${id}`); }
  getUsers(page = 1, q = "") { return this.req<{ users: User[]; total: number }>("GET", `/admin/users?page=${page}&limit=50${q ? `&search=${encodeURIComponent(q)}` : ""}`); }
  getUser(id: string) { return this.req<User>("GET", `/admin/users/${id}`); }
  getFraud()       { return this.req<{ events: FraudEvent[] }>("GET", "/admin/fraud"); }
  getFraudEvents() { return this.getFraud(); }
  resolveFraud(id: string) { return this.req("POST", `/admin/fraud/${id}/resolve`, {}); }
  getRegionalStats() { return this.req<{ stats: RegionalStat[] }>("GET", "/admin/regional-stats"); }
  getPointsStats() { return this.req<PointsStats>("GET", "/admin/points/stats"); }
  getPointsHistory(page = 0) { return this.req<{ items: PointsHistoryItem[]; total: number }>("GET", `/admin/points/history?offset=${page * 50}`); }
  getStudioTools() { return this.req<{ tools: StudioTool[] }>("GET", "/admin/studio-tools"); }
  updateStudioTool(id: string, payload: Partial<StudioTool>) { return this.req("PUT", `/admin/studio-tools/${id}`, payload); }
  // New Studio Tool CRUD methods
  createStudioTool(data: Partial<StudioTool>): Promise<StudioTool> { return this.req<StudioTool>("POST", "/admin/studio-tools", data); }
  disableStudioTool(id: string): Promise<void> { return this.req<void>("DELETE", `/admin/studio-tools/${id}`); }
  getStudioToolErrors(id: string): Promise<{ errors: GenerationError[]; count: number }> { return this.req<{ errors: GenerationError[]; count: number }>("GET", `/admin/studio-tools/${id}/errors`); }
  getStudioToolStats(): Promise<{ stats: ToolStat[] }> { return this.req<{ stats: ToolStat[] }>("GET", "/admin/studio-tools/stats"); }
  getStudioGenerations(params?: { status?: string; tool_slug?: string; limit?: number; offset?: number }): Promise<{ items: Generation[]; total: number }> {
    const qs = new URLSearchParams();
    if (params?.status) qs.set("status", params.status);
    if (params?.tool_slug) qs.set("tool_slug", params.tool_slug);
    if (params?.limit !== undefined) qs.set("limit", String(params.limit));
    if (params?.offset !== undefined) qs.set("offset", String(params.offset));
    const q = qs.toString();
    return this.req<{ items: Generation[]; total: number }>("GET", `/admin/studio-generations${q ? `?${q}` : ""}`);
  }
  getBroadcasts() { return this.req<{ broadcasts: Broadcast[] }>("GET", "/admin/notifications/broadcasts"); }
  createBroadcast(payload: BroadcastPayload) { return this.req<{ id: string }>("POST", "/admin/notifications/broadcast", payload); }
  // Draws management
  getDraws()               { return this.req<{ draws: Draw[] }>("GET", "/admin/draws"); }
  createDraw(d: CreateDrawPayload) { return this.req<Draw>("POST", "/admin/draws", d); }
  executeDraw(id: string)  { return this.req("POST", `/admin/draws/${id}/execute`, {}); }
  getDrawWinners(id: string){ return this.req<{ winners: DrawWinner[] }>("GET", `/admin/draws/${id}/winners`); }
  // Notification aliases
  getNotificationHistory() { return this.getBroadcasts(); }
  broadcastNotification(payload: BroadcastPayload) { return this.createBroadcast(payload); }
  // Users
  suspendUser(id: string)   { return this.req("PUT", `/admin/users/${id}/suspend`, { suspended: true }); }
  // Regional Wars
  getRegionalWars() { return this.req<{ leaderboard: RegionalStat[] }>("GET", "/admin/regional-wars"); }
  resolveWar(period: string) { return this.req("POST", "/admin/wars/resolve", { period }); }
  getSecondaryDraws(warId: string) { return this.req<{ draws: WarSecondaryDraw[] }>("GET", `/admin/wars/${warId}/secondary-draws`); }
  runSecondaryDraw(warId: string, payload: { state: string; winner_count: number; prize_per_winner_kobo: number }) {
    return this.req<WarSecondaryDraw>("POST", `/admin/wars/${warId}/secondary-draw`, payload);
  }
  markSecondaryWinnerPaid(winnerId: string, momoNumber: string) {
    return this.req<{ status: string }>("POST", `/admin/wars/secondary-draw/winners/${winnerId}/pay`, { momo_number: momoNumber });
  }
  getHealth() { return this.req<HealthReport>("GET", "/admin/health"); }
  getAIHealth() { return this.req<AIHealthReport>("GET", "/admin/ai-health"); }

  // ── AI Provider Config (dynamic provider registry) ───────────────────────
  getAIProviders()   { return this.req<AIProvidersResponse>("GET", "/admin/ai-providers"); }
  getAIProviderMeta(){ return this.req<AIProviderMeta>("GET", "/admin/ai-providers/meta"); }
  createAIProvider(data: AIProviderFormPayload) { return this.req<AIProviderConfig>("POST", "/admin/ai-providers", data); }
  updateAIProvider(id: string, data: Partial<AIProviderFormPayload>) { return this.req<AIProviderConfig>("PUT", `/admin/ai-providers/${id}`, data); }
  deleteAIProvider(id: string) { return this.req<{ status: string }>("DELETE", `/admin/ai-providers/${id}`); }
  activateAIProvider(id: string)   { return this.req<{ status: string }>("POST", `/admin/ai-providers/${id}/activate`,   {}); }
  deactivateAIProvider(id: string) { return this.req<{ status: string }>("POST", `/admin/ai-providers/${id}/deactivate`, {}); }
  testAIProvider(id: string)       { return this.req<AIProviderTestResult>("POST", `/admin/ai-providers/${id}/test`,      {}); }

  // ─── MTN Push CSV Upload ──────────────────────────────────────────────────
  async uploadMTNPushCSV(file: File, note?: string): Promise<CSVUploadResult> {
    const form = new FormData();
    form.append("file", file);
    if (note) form.append("note", note);
    const resp = await fetch(`${BASE_URL}/admin/mtn-push/csv-upload`, {
      method: "POST",
      headers: { ...(this.getToken() ? { Authorization: `Bearer ${this.getToken()}` } : {}) },
      body: form,
    });
    if (resp.status === 401) { this.clearToken(); window.location.href = "/login"; throw new Error("Unauthorized"); }
    const data = await resp.json();
    if (!resp.ok) throw new Error((data as { error?: string }).error || "Upload failed");
    return data as CSVUploadResult;
  }
  listMTNPushCSVUploads(limit = 20, offset = 0) {
    return this.req<{ uploads: CSVUploadSummary[]; total: number }>("GET", `/admin/mtn-push/csv-upload?limit=${limit}&offset=${offset}`);
  }
  getMTNPushCSVUpload(id: string) {
    return this.req<CSVUploadSummary>("GET", `/admin/mtn-push/csv-upload/${id}`);
  }
  getMTNPushCSVUploadRows(id: string, limit = 100, offset = 0) {
    return this.req<{ rows: CSVRowDetail[]; total: number }>("GET", `/admin/mtn-push/csv-upload/${id}/rows?limit=${limit}&offset=${offset}`);
  }
  // ─── Bonus Pulse Point Awards ─────────────────────────────────────────────
  // ─── Passport Admin ──────────────────────────────────────────────────────
  getPassportStats() { return this.req<PassportStats>("GET", "/admin/passport/stats"); }
  getPassportNudgeLog(limit = 50) { return this.req<{ logs: GhostNudgeLog[] }>("GET", `/admin/passport/nudge-log?limit=${limit}`); }
  getUSSDSessions(limit = 50) { return this.req<{ sessions: USSDSession[] }>("GET", `/admin/ussd/sessions?limit=${limit}`); }

  // ─── Admin Auth ─────────────────────────────────────────────────────────
  me() { return this.req<{ admin_id: string; email: string; role: string }>("GET", "/admin/auth/me"); }
  changePassword(oldPassword: string, newPassword: string) {
    return this.req("POST", "/admin/auth/change-password", { old_password: oldPassword, new_password: newPassword });
  }

  awardBonusPulse(payload: { phone_number: string; points: number; campaign?: string; note?: string }) {
    return this.req<BonusPulseAwardResult>("POST", "/admin/bonus-pulse", payload);
  }
  listBonusPulseAwards(params?: { phone?: string; campaign?: string; limit?: number; offset?: number }) {
    const qs = new URLSearchParams();
    if (params?.phone)    qs.set("phone",    params.phone);
    if (params?.campaign) qs.set("campaign", params.campaign);
    if (params?.limit    !== undefined) qs.set("limit",  String(params.limit));
    if (params?.offset   !== undefined) qs.set("offset", String(params.offset));
    const q = qs.toString();
    return this.req<{ records: BonusPulseAwardRecord[]; total: number }>("GET", `/admin/bonus-pulse${q ? `?${q}` : ""}`);
  }

  // ─── Recharge Reward Config ─────────────────────────────────────────────────
  getRechargeConfig() { return this.req<RechargeConfig>("GET", "/admin/recharge/config"); }
  updateRechargeConfig(payload: Partial<RechargeConfigPayload>) {
    return this.req<RechargeConfig>("PUT", "/admin/recharge/config", payload);
  }

  // ─── Spin Claims ─────────────────────────────────────────────────────────
  listClaims(status = "", page = 1, limit = 50) {
    const offset = (page - 1) * limit;
    return this.req<{ data: SpinClaim[]; total: number }>("GET", `/admin/spin/claims?status=${status}&limit=${limit}&offset=${offset}`);
  }
  getPendingClaims() {
    return this.req<{ data: SpinClaim[]; total: number }>("GET", "/admin/spin/claims/pending");
  }
  getClaimDetails(id: string) {
    return this.req<SpinClaim>("GET", `/admin/spin/claims/${id}`);
  }
  approveClaim(id: string, adminNotes: string, paymentReference = "") {
    return this.req<SpinClaim>("POST", `/admin/spin/claims/${id}/approve`, { admin_notes: adminNotes, payment_reference: paymentReference });
  }
  rejectClaim(id: string, rejectionReason: string, adminNotes = "") {
    return this.req<SpinClaim>("POST", `/admin/spin/claims/${id}/reject`, { rejection_reason: rejectionReason, admin_notes: adminNotes });
  }
  getClaimStatistics() {
    return this.req<ClaimStatistics>("GET", "/admin/spin/claims/statistics");
  }
  exportClaims(status = "") {
    return this.req<string>("GET", `/admin/spin/claims/export?status=${status}`);
  }

  // ─── Spin Tiers ──────────────────────────────────────────────────────────
  getSpinTiers() {
    return this.req<SpinTier[]>("GET", "/admin/spin/tiers");
  }
  createSpinTier(data: Omit<SpinTier, "id">) {
    return this.req<SpinTier>("POST", "/admin/spin/tiers", data);
  }
  updateSpinTier(id: string, data: Partial<SpinTier>) {
    return this.req<SpinTier>("PUT", `/admin/spin/tiers/${id}`, data);
  }
  deleteSpinTier(id: string) {
    return this.req<void>("DELETE", `/admin/spin/tiers/${id}`);
  }

  // ─── User management extras ───────────────────────────────────────────────
  unsuspendUser(id: string) {
    return this.req<void>("PUT", `/admin/users/${id}/suspend`, { suspended: false });
  }
  adjustPoints(userId: string, delta: number, reason: string) {
    return this.req<void>("POST", "/admin/points/adjust", { user_id: userId, delta, reason });
  }

  // ─── Fraud ────────────────────────────────────────────────────────────────
  resolveFraudEvent(id: string, notes = "") {
    return this.req<void>("POST", `/admin/fraud/${id}/resolve`, { notes });
  }

  // ─── Draws extras ───────────────────────────────────────────────────────
  updateDraw(id: string, data: Partial<CreateDrawPayload>) {
    return this.req<Draw>("PUT", `/admin/draws/${id}`, data);
  }
  exportDrawEntries(id: string) {
    return this.req<string>("GET", `/admin/draws/${id}/export`);
  }

  // ─── Draw Schedule (window rules) ───────────────────────────────────────
  getDrawSchedules() {
    return this.req<{ schedules: DrawSchedule[] }>("GET", "/admin/draw/schedule");
  }
  createDrawSchedule(data: CreateDrawSchedulePayload) {
    return this.req<DrawSchedule>("POST", "/admin/draw/schedule", data);
  }
  updateDrawSchedule(id: string, data: Partial<CreateDrawSchedulePayload>) {
    return this.req<DrawSchedule>("PUT", `/admin/draw/schedule/${id}`, data);
  }
  deleteDrawSchedule(id: string) {
    return this.req<void>("DELETE", `/admin/draw/schedule/${id}`);
  }
  previewDrawWindow() {
    return this.req<{ qualifying_draws: { draw_id: string; draw_type: string; draw_name: string }[] }>("GET", "/admin/draw/schedule/preview");
  }
}
export interface DashboardStats {
  total_users: number;
  active_today: number;
  total_recharge_kobo: number;
  spins_today: number;
  studio_generations_today: number;
  pending_claims?: number;
  draws_active?: number;
  mtn_pushes_today?: number;
}
export interface ConfigEntry { key: string; value: unknown; description: string; updated_at: string; }
export interface Prize {
  id: string;
  name: string;
  prize_code?: string;
  prize_type: string;
  base_value: number;
  win_probability_weight: number;
  daily_inventory_cap?: number;
  is_active: boolean;
  is_no_win?: boolean;
  no_win_message?: string;
  color_scheme?: string;
  sort_order?: number;
  minimum_recharge?: number;
  icon_name?: string;
}
export interface StudioTool {
  id: string; name: string; slug?: string; category: string; provider: string;
  point_cost: number; is_active: boolean; description?: string; icon?: string;
  provider_tool?: string; sort_order?: number; generated_today?: number; success_rate?: number;
  entry_point_cost: number;    // min wallet balance to open the tool (0 = no gate)
  refund_window_mins: number;  // minutes user can dispute after generation (0 = no refunds)
  refund_pct: number;          // % of pts returned on approved dispute (0-100)
  is_free: boolean;            // true = bypass all point checks (e.g. chat)
}
export interface User {
  id: string;
  phone_number: string;
  tier: string;
  state?: string;
  streak_count: number;
  is_active: boolean;
  created_at: string;
  last_recharge_at?: string | null;
  pulse_points?: number;
  spin_credits?: number;
}
export interface FraudEvent {
  id: string;
  user_id: string;
  msisdn?: string;
  event_type: string;
  severity: string;
  resolved: boolean;
  notes?: string;
  created_at: string;
}

export interface SpinClaim {
  id: string;
  user_id: string;
  prize_type: string;
  prize_value: number;       // in kobo
  claim_status: string;      // PENDING | PENDING_ADMIN_REVIEW | APPROVED | REJECTED | CLAIMED | EXPIRED
  fulfillment_status: string;
  momo_number?: string;
  momo_claim_number?: string;
  bank_account_number?: string;
  bank_account_name?: string;
  bank_name?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  rejection_reason?: string;
  admin_notes?: string;
  payment_reference?: string;
  expires_at: string;
  created_at: string;
  claimed_at?: string;
  fulfilled_at?: string;
}

export interface ClaimStatistics {
  total_claims: number;
  pending_claims: number;
  approved_claims: number;
  rejected_claims: number;
  claimed_claims: number;
  expired_claims: number;
  total_value_ngn: number;
  approved_value_ngn: number;
  pending_value_ngn: number;
}

export interface PrizeSummaryItem {
  prize_id: string;
  name: string;
  prize_type: string;
  weight: number;
  percent: number;
}
export interface PrizeSummary {
  items: PrizeSummaryItem[];
  total_weight: number;
  remaining_budget: number;
  percent_used: number;
  is_valid: boolean;
}

export interface SpinTier {
  id: string;
  tier_name: string;
  tier_display_name?: string;
  min_daily_amount: number;  // in kobo
  max_daily_amount: number;  // in kobo (0 = unlimited)
  spins_per_day: number;
  tier_color?: string;
  sort_order?: number;
}
export interface RegionalStat { state: string; total_points: number; active_members: number; rank: number; }

export interface WarSecondaryDrawWinner {
  id: string;
  secondary_draw_id: string;
  war_id: string;
  state: string;
  user_id: string;
  phone_number: string;
  position: number;
  prize_kobo: number;
  momo_number?: string;
  payment_status: "PENDING_PAYMENT" | "PAID" | "FAILED";
  paid_at?: string;
  notes?: string;
}

export interface WarSecondaryDraw {
  id: string;
  war_id: string;
  state: string;
  winner_count: number;
  prize_per_winner_kobo: number;
  total_pool_kobo: number;
  participant_count: number;
  status: "PENDING" | "COMPLETED" | "CANCELLED";
  executed_at?: string;
  created_at: string;
  winners?: WarSecondaryDrawWinner[];
}
export const adminAPI = new AdminAPI();
export default adminAPI;
export interface BroadcastPayload {
  title: string;
  body: string;
  type: string;
  target: "all" | "active_subscribers" | "free_tier" | "phone_list";
  phone_list?: string[];
  deep_link?: string;
}
export interface Broadcast {
  id: string; title: string; body: string; type: string;
  target: string; sent_count: number; created_at: string;
}
export interface Draw {
  id: string; name: string; prize_pool_kobo: number;
  status: string; draw_date: string; entry_count: number;
  recurrence: string; created_at: string;
}
export interface CreateDrawPayload {
  name: string; prize_pool_kobo: number;
  draw_date: string; recurrence: "once" | "weekly" | "monthly";
}
export interface DrawWinner {
  id: string; user_id: string; phone_number: string;
  prize_label: string; rank: number; created_at: string;
}
export interface DrawSchedule {
  id: string;
  draw_name: string;
  draw_type: string;          // DAILY | WEEKLY
  draw_day_of_week: number;   // 0=Sun … 6=Sat
  draw_time_wat: string;      // "HH:MM:SS"
  window_open_dow: number;
  window_open_time: string;
  window_close_dow: number;
  window_close_time: string;
  cutoff_hour_utc: number;
  is_active: boolean;
  sort_order: number;
}
export interface CreateDrawSchedulePayload {
  draw_name: string;
  draw_type: string;
  draw_day_of_week: number;
  draw_time_wat: string;
  window_open_dow: number;
  window_open_time: string;
  window_close_dow: number;
  window_close_time: string;
  cutoff_hour_utc: number;
  sort_order: number;
  is_active: boolean;
}

// REQ-5.8.3 — System health endpoint
export interface ServiceHealth {
  name: string; status: "up"|"degraded"|"down";
  latency_ms: number; uptime_pct: number;
  last_checked: string; note?: string;
}
export interface HealthReport {
  overall: "healthy"|"degraded"|"outage";
  services: ServiceHealth[];
  webhook_success_rate_24h: number;
  paystack_success_rate_24h: number;
  api_p99_ms: number;
  db_pool_used: number; db_pool_max: number;
  redis_hit_rate: number;
  checked_at: string;
}

// ─── AI Health types ──────────────────────────────────────────────────────────
export interface AIProviderStatus {
  name: string;
  status: "ok" | "error" | "limit_reached" | "unknown";
  requests_today: number;
  last_used_at: string | null;
  last_error: string | null;
}
export interface ProviderSwitchEvent {
  from: string;
  to: string;
  reason: string;
  ts: number;
}
export interface StudioToolHealth {
  slug: string;
  requests_today: number;
  last_provider: string;
  last_used_at: string | null;
  error_count_today?: number;
  last_error_at?: string | null;
}
export interface AIHealthReport {
  active_chat_provider: string;
  providers: AIProviderStatus[];
  recent_switches: ProviderSwitchEvent[];
  studio_tools: StudioToolHealth[];
  checked_at: string;
}
export interface PointsStats {
  total_points_issued: number;
  total_points_spent: number;
  points_in_circulation: number;
  active_wallets: number;
}
export interface PointsHistoryItem {
  id: string;
  user_id: string;
  phone_number: string;
  type: string;
  points_delta: number;
  created_at: string;
}

// ─── New Studio Tool types ─────────────────────────────────────────────────────
export interface GenerationError {
  id: string;
  user_id: string;
  prompt: string;
  error_message: string;
  provider: string;
  created_at: string;
}
export interface ToolStat {
  tool_id: string;
  tool_slug: string;
  total: number;
  completed: number;
  failed: number;
  points_consumed: number;
}
export interface Generation {
  id: string;
  user_id: string;
  tool_slug: string;
  status: string;
  provider: string;
  prompt: string;
  points_deducted: number;
  created_at: string;
}

// ── AI Provider Config types ──────────────────────────────────────────────────
export interface AIProviderConfig {
  id: string;
  name: string;
  slug: string;
  category: string;
  template: string;
  env_key: string;
  model_id: string;
  extra_config: Record<string, unknown>;
  priority: number;
  is_primary: boolean;
  is_active: boolean;
  cost_micros: number;
  pulse_pts: number;
  notes: string;
  has_key: boolean;
  last_tested_at: string | null;
  last_test_ok: boolean | null;
  last_test_msg: string;
  created_at: string;
  updated_at: string;
}

export interface AIProvidersResponse {
  providers: AIProviderConfig[];
  grouped: Record<string, AIProviderConfig[]>;
  total: number;
}

export interface AIProviderMeta {
  categories: string[];
  templates: string[];
  template_descriptions: Record<string, string>;
}

export interface AIProviderFormPayload {
  name: string;
  slug?: string;
  category: string;
  template: string;
  env_key?: string;
  api_key?: string;
  model_id?: string;
  extra_config?: Record<string, unknown>;
  priority?: number;
  is_primary?: boolean;
  is_active?: boolean;
  cost_micros?: number;
  pulse_pts?: number;
  notes?: string;
}

export interface AIProviderTestResult {
  status: "ok" | "failed";
  message: string;
  last_tested_at: string;
}

// ─── Bonus Pulse Point Awards types ─────────────────────────────────────────────
export interface BonusPulseAwardResult {
  award_id: string;
  transaction_id: string;
  user_id: string;
  phone_number: string;
  points_awarded: number;
  new_balance: number;
  campaign: string;
  awarded_at: string;
}
export interface BonusPulseAwardRecord {
  id: string;
  user_id: string;
  phone_number: string;
  points: number;
  campaign: string;
  note: string;
  awarded_by: string;
  awarded_by_name: string;
  transaction_id: string;
  created_at: string;
}

// ─── MTN Push CSV Upload types ──────────────────────────────────────────────────
export interface CSVUploadResult {
  upload_id: string;
  total_rows: number;
  processed_rows: number;
  skipped_rows: number;
  failed_rows: number;
  status: string; // DONE | PARTIAL | FAILED
}
export interface CSVUploadSummary {
  id: string;
  uploaded_by: string;
  filename: string;
  uploaded_at: string;
  total_rows: number;
  processed_rows: number;
  skipped_rows: number;
  failed_rows: number;
  status: string; // DONE | PARTIAL | FAILED
  note?: string;
  completed_at?: string | null;
}
export interface CSVRowDetail {
  row_number: number;
  raw_msisdn: string;
  raw_date: string;
  raw_time: string;
  raw_amount: string;
  recharge_type: string;
  status: string; // PROCESSED | SKIPPED | FAILED
  skip_reason?: string;
  error_msg?: string;
  transaction_ref?: string;
  spin_credits: number;
  pulse_points: number;
  draw_entries: number;
  processed_at?: string | null;
}

// ─── Recharge Reward Config types ──────────────────────────────────────────────
export interface RechargeConfig {
  spin_naira_per_credit: number;   // ₦ minimum daily recharge for Bronze spin tier
  draw_naira_per_entry: number;    // ₦ per Draw Entry (flat per-transaction)
  pulse_naira_per_point: number;   // ₦ per Pulse Point
  spin_max_per_day: number;        // max spin credits per calendar day
  min_amount_naira: number;        // minimum qualifying recharge amount
}
export interface RechargeConfigPayload {
  spin_naira_per_credit?: number;
  draw_naira_per_entry?: number;
  pulse_naira_per_point?: number;
  spin_max_per_day?: number;
  min_amount_naira?: number;
}
