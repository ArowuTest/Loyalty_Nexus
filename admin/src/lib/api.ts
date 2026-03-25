const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

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
  getStudioTools() { return this.req<{ tools: StudioTool[] }>("GET", "/admin/studio-tools"); }
  getUsers(page = 0) { return this.req<{ users: User[] }>("GET", `/admin/users?offset=${page * 50}`); }
  suspendUser(id: string) { return this.req("PUT", `/admin/users/${id}/suspend`, {}); }
  getFraudEvents() { return this.req<{ events: FraudEvent[] }>("GET", "/admin/fraud-events"); }
  getRegionalWars(){ return this.req<{ leaderboard: RegionalStat[] }>("GET", "/admin/regional-wars"); }
  // Notifications & Broadcasts
  broadcastNotification(payload: BroadcastPayload) { return this.req("POST", "/admin/notifications/broadcast", payload); }
  getNotificationHistory(page = 0) { return this.req<{ broadcasts: Broadcast[] }>("GET", `/admin/notifications/broadcasts?offset=${page * 20}`); }
  // Subscription management
  getSubscriptions(page = 0, status = "") {
    const qs = status ? `&status=${status}` : "";
    return this.req<{ users: SubscriptionUser[] }>("GET", `/admin/subscriptions?offset=${page * 50}${qs}`);
  }
  updateSubscription(userId: string, payload: UpdateSubPayload) { return this.req("PUT", `/admin/users/${userId}/subscription`, payload); }
  // Draws management
  getDraws()               { return this.req<{ draws: Draw[] }>("GET", "/admin/draws"); }
  createDraw(d: CreateDrawPayload) { return this.req<Draw>("POST", "/admin/draws", d); }
  executeDraw(id: string)  { return this.req("POST", `/admin/draws/${id}/execute`, {}); }
  getDrawWinners(id: string){ return this.req<{ winners: DrawWinner[] }>("GET", `/admin/draws/${id}/winners`); }
  // Regional Wars admin
  resolveWar(period: string) { return this.req("POST", "/admin/wars/resolve", { period }); }
  getHealth() { return this.req<HealthReport>("GET", "/admin/health"); }
}
export interface DashboardStats { total_users: number; active_today: number; total_recharge_kobo: number; spins_today: number; studio_generations_today: number; }
export interface ConfigEntry { key: string; value: unknown; description: string; updated_at: string; }
export interface Prize { id: string; name: string; prize_type: string; base_value: number; probability: number; daily_inventory: number; is_active: boolean; }
export interface StudioTool { id: string; name: string; category: string; provider: string; point_cost: number; is_active: boolean; }
export interface User { id: string; phone_number: string; tier: string; streak_count: number; is_active: boolean; created_at: string; }
export interface FraudEvent { id: string; user_id: string; event_type: string; severity: string; resolved: boolean; created_at: string; }
export interface RegionalStat { state: string; total_points: number; active_members: number; rank: number; }
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
export interface SubscriptionUser {
  id: string; phone_number: string; tier: string;
  subscription_status: string; subscription_expires_at: string | null;
  created_at: string;
}
export interface UpdateSubPayload {
  status: string;   // ACTIVE | FREE | GRACE | SUSPENDED
  expires_at?: string; // ISO
  note?: string;
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
