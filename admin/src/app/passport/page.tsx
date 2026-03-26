"use client";

import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────────────────────

interface PassportStats {
  total_passports: number;
  apple_wallet_downloads: number;
  google_wallet_saves: number;
  qr_scans_today: number;
  tier_breakdown: { tier: string; count: number }[];
  top_badge_earners: { user_id: string; phone: string; badge_count: number; tier: string }[];
}

interface GhostNudgeLog {
  id: string;
  user_id: string;
  phone_number: string;
  nudge_type: string;
  streak_count: number;
  sent_at: string;
  delivered: boolean;
}

interface USSDSession {
  id: string;
  phone_number: string;
  session_id: string;
  current_menu: string;
  started_at: string;
  last_active: string;
  is_active: boolean;
  step_count: number;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

const TIER_COLORS: Record<string, string> = {
  BRONZE:   "#f59e0b",
  SILVER:   "#94a3b8",
  GOLD:     "#eab308",
  PLATINUM: "#a78bfa",
};

// ─── Main component ───────────────────────────────────────────────────────────

export default function PassportAdminPage() {
  const [stats, setStats]           = useState<PassportStats | null>(null);
  const [nudgeLogs, setNudgeLogs]   = useState<GhostNudgeLog[]>([]);
  const [ussdSessions, setUSSD]     = useState<USSDSession[]>([]);
  const [loading, setLoading]       = useState(true);
  const [activeTab, setActiveTab]   = useState<"overview" | "nudges" | "ussd">("overview");
  const [autoRefresh, setAutoRefresh] = useState(false);

  const load = useCallback(async () => {
    try {
      const [s, n, u] = await Promise.all([
        adminAPI.req<PassportStats>("GET", "/admin/passport/stats"),
        adminAPI.req<{ logs: GhostNudgeLog[] }>("GET", "/admin/passport/nudge-log?limit=50"),
        adminAPI.req<{ sessions: USSDSession[] }>("GET", "/admin/ussd/sessions?limit=50"),
      ]);
      setStats(s);
      setNudgeLogs(n.logs ?? []);
      setUSSD(u.sessions ?? []);
    } catch {
      // Silently handle — data may not be available yet
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (!autoRefresh) return;
    const id = setInterval(load, 15000);
    return () => clearInterval(id);
  }, [autoRefresh, load]);

  const cardStyle: React.CSSProperties = {
    background: "#1c2038",
    border: "1px solid rgba(95,114,249,0.15)",
    borderRadius: 12,
    padding: 20,
  };

  const statCardStyle: React.CSSProperties = {
    ...cardStyle,
    display: "flex",
    flexDirection: "column",
    gap: 4,
  };

  const tabStyle = (active: boolean): React.CSSProperties => ({
    padding: "8px 20px",
    borderRadius: 8,
    border: "none",
    cursor: "pointer",
    fontWeight: active ? 600 : 400,
    fontSize: 13,
    background: active ? "rgba(95,114,249,0.2)" : "transparent",
    color: active ? "#5f72f9" : "#828cb4",
    transition: "all 0.15s",
  });

  return (
    <AdminShell>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <div>
          <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff", margin: 0 }}>
            🪪 Digital Passport &amp; USSD
          </h1>
          <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
            Wallet pass activity, ghost nudge logs, and USSD session monitor
          </p>
        </div>
        <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
          <label style={{ display: "flex", alignItems: "center", gap: 6, color: "#828cb4", fontSize: 13, cursor: "pointer" }}>
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={e => setAutoRefresh(e.target.checked)}
              style={{ accentColor: "#5f72f9" }}
            />
            Auto-refresh (15s)
          </label>
          <button
            onClick={load}
            style={{
              padding: "8px 16px", borderRadius: 8, border: "1px solid rgba(95,114,249,0.3)",
              background: "transparent", color: "#5f72f9", cursor: "pointer", fontSize: 13,
            }}
          >
            ↻ Refresh
          </button>
        </div>
      </div>

      {/* Stat cards */}
      {loading ? (
        <div style={{ color: "#828cb4" }}>Loading passport data…</div>
      ) : (
        <>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))", gap: 14, marginBottom: 24 }}>
            {[
              { label: "Total Passports",      value: stats?.total_passports ?? 0,         icon: "🪪" },
              { label: "Apple Wallet Downloads", value: stats?.apple_wallet_downloads ?? 0, icon: "📱" },
              { label: "Google Wallet Saves",   value: stats?.google_wallet_saves ?? 0,     icon: "💳" },
              { label: "QR Scans Today",        value: stats?.qr_scans_today ?? 0,          icon: "📷" },
            ].map(s => (
              <div key={s.label} style={statCardStyle}>
                <div style={{ fontSize: 26 }}>{s.icon}</div>
                <div style={{ fontSize: 28, fontWeight: 700, color: "#e2e8ff" }}>
                  {s.value.toLocaleString()}
                </div>
                <div style={{ color: "#828cb4", fontSize: 12 }}>{s.label}</div>
              </div>
            ))}
          </div>

          {/* Tier breakdown */}
          {stats?.tier_breakdown && stats.tier_breakdown.length > 0 && (
            <div style={{ ...cardStyle, marginBottom: 24 }}>
              <h3 style={{ color: "#e2e8ff", fontSize: 14, fontWeight: 600, marginBottom: 14 }}>
                Tier Distribution
              </h3>
              <div style={{ display: "flex", gap: 16, flexWrap: "wrap" }}>
                {stats.tier_breakdown.map(t => {
                  const total = stats.tier_breakdown.reduce((a, b) => a + b.count, 0);
                  const pct = total > 0 ? Math.round((t.count / total) * 100) : 0;
                  return (
                    <div key={t.tier} style={{ flex: "1 1 120px" }}>
                      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 6 }}>
                        <span style={{ color: TIER_COLORS[t.tier] ?? "#e2e8ff", fontSize: 13, fontWeight: 600 }}>
                          {t.tier}
                        </span>
                        <span style={{ color: "#828cb4", fontSize: 12 }}>{t.count.toLocaleString()} ({pct}%)</span>
                      </div>
                      <div style={{ height: 6, background: "rgba(255,255,255,0.05)", borderRadius: 3, overflow: "hidden" }}>
                        <div style={{
                          height: "100%",
                          width: `${pct}%`,
                          background: TIER_COLORS[t.tier] ?? "#5f72f9",
                          borderRadius: 3,
                          transition: "width 0.8s ease",
                        }} />
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Tabs */}
          <div style={{ display: "flex", gap: 4, marginBottom: 20, background: "rgba(255,255,255,0.03)", padding: 4, borderRadius: 10, width: "fit-content" }}>
            {(["overview", "nudges", "ussd"] as const).map(tab => (
              <button key={tab} onClick={() => setActiveTab(tab)} style={tabStyle(activeTab === tab)}>
                {tab === "overview" ? "🏆 Top Earners" : tab === "nudges" ? "👻 Ghost Nudges" : "📱 USSD Sessions"}
              </button>
            ))}
          </div>

          {/* Tab: Top Badge Earners */}
          {activeTab === "overview" && (
            <div style={cardStyle}>
              <h3 style={{ color: "#e2e8ff", fontSize: 14, fontWeight: 600, marginBottom: 14 }}>
                Top Badge Earners
              </h3>
              {!stats?.top_badge_earners || stats.top_badge_earners.length === 0 ? (
                <p style={{ color: "#828cb4", fontSize: 13 }}>No badge data yet.</p>
              ) : (
                <table style={{ width: "100%", borderCollapse: "collapse" }}>
                  <thead>
                    <tr>
                      {["Rank", "Phone", "Tier", "Badges"].map(h => (
                        <th key={h} style={{ textAlign: "left", padding: "8px 12px", color: "#4b5563", fontSize: 11, fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.06em", borderBottom: "1px solid rgba(255,255,255,0.05)" }}>
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {stats.top_badge_earners.map((u, i) => (
                      <tr key={u.user_id} style={{ borderBottom: "1px solid rgba(255,255,255,0.04)" }}>
                        <td style={{ padding: "10px 12px", color: "#828cb4", fontSize: 13 }}>#{i + 1}</td>
                        <td style={{ padding: "10px 12px", color: "#e2e8ff", fontSize: 13, fontFamily: "monospace" }}>
                          {u.phone.replace(/(\d{4})(\d+)(\d{4})/, "$1****$3")}
                        </td>
                        <td style={{ padding: "10px 12px" }}>
                          <span style={{
                            padding: "2px 8px", borderRadius: 4, fontSize: 11, fontWeight: 600,
                            color: TIER_COLORS[u.tier] ?? "#e2e8ff",
                            background: `${TIER_COLORS[u.tier] ?? "#5f72f9"}20`,
                          }}>
                            {u.tier}
                          </span>
                        </td>
                        <td style={{ padding: "10px 12px", color: "#5f72f9", fontSize: 13, fontWeight: 600 }}>
                          🏅 {u.badge_count}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          )}

          {/* Tab: Ghost Nudge Log */}
          {activeTab === "nudges" && (
            <div style={cardStyle}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 14 }}>
                <h3 style={{ color: "#e2e8ff", fontSize: 14, fontWeight: 600, margin: 0 }}>
                  Ghost Nudge Log
                </h3>
                <span style={{ color: "#828cb4", fontSize: 12 }}>
                  {nudgeLogs.length} recent nudges
                </span>
              </div>
              {nudgeLogs.length === 0 ? (
                <p style={{ color: "#828cb4", fontSize: 13 }}>No nudges sent yet. The Ghost Nudge worker fires every 5 minutes.</p>
              ) : (
                <table style={{ width: "100%", borderCollapse: "collapse" }}>
                  <thead>
                    <tr>
                      {["Phone", "Type", "Streak", "Sent", "Delivered"].map(h => (
                        <th key={h} style={{ textAlign: "left", padding: "8px 12px", color: "#4b5563", fontSize: 11, fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.06em", borderBottom: "1px solid rgba(255,255,255,0.05)" }}>
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {nudgeLogs.map(log => (
                      <tr key={log.id} style={{ borderBottom: "1px solid rgba(255,255,255,0.04)" }}>
                        <td style={{ padding: "10px 12px", color: "#e2e8ff", fontSize: 13, fontFamily: "monospace" }}>
                          {log.phone_number.replace(/(\d{4})(\d+)(\d{4})/, "$1****$3")}
                        </td>
                        <td style={{ padding: "10px 12px" }}>
                          <span style={{
                            padding: "2px 8px", borderRadius: 4, fontSize: 11, fontWeight: 600,
                            background: log.nudge_type === "streak_at_risk" ? "rgba(239,68,68,0.15)" : "rgba(95,114,249,0.15)",
                            color: log.nudge_type === "streak_at_risk" ? "#ef4444" : "#5f72f9",
                          }}>
                            {log.nudge_type.replace(/_/g, " ")}
                          </span>
                        </td>
                        <td style={{ padding: "10px 12px", color: "#f59e0b", fontSize: 13 }}>
                          🔥 {log.streak_count}
                        </td>
                        <td style={{ padding: "10px 12px", color: "#828cb4", fontSize: 12 }}>
                          {timeAgo(log.sent_at)}
                        </td>
                        <td style={{ padding: "10px 12px" }}>
                          <span style={{ color: log.delivered ? "#22c55e" : "#ef4444", fontSize: 13 }}>
                            {log.delivered ? "✓ Yes" : "✗ No"}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          )}

          {/* Tab: USSD Sessions */}
          {activeTab === "ussd" && (
            <div style={cardStyle}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 14 }}>
                <h3 style={{ color: "#e2e8ff", fontSize: 14, fontWeight: 600, margin: 0 }}>
                  USSD Sessions
                </h3>
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                  <span style={{ color: "#22c55e", fontSize: 12 }}>
                    ● {ussdSessions.filter(s => s.is_active).length} active
                  </span>
                  <span style={{ color: "#828cb4", fontSize: 12 }}>
                    {ussdSessions.length} total shown
                  </span>
                </div>
              </div>
              {ussdSessions.length === 0 ? (
                <p style={{ color: "#828cb4", fontSize: 13 }}>No USSD sessions recorded yet.</p>
              ) : (
                <table style={{ width: "100%", borderCollapse: "collapse" }}>
                  <thead>
                    <tr>
                      {["Phone", "Session ID", "Current Menu", "Steps", "Last Active", "Status"].map(h => (
                        <th key={h} style={{ textAlign: "left", padding: "8px 12px", color: "#4b5563", fontSize: 11, fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.06em", borderBottom: "1px solid rgba(255,255,255,0.05)" }}>
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {ussdSessions.map(session => (
                      <tr key={session.id} style={{ borderBottom: "1px solid rgba(255,255,255,0.04)" }}>
                        <td style={{ padding: "10px 12px", color: "#e2e8ff", fontSize: 13, fontFamily: "monospace" }}>
                          {session.phone_number.replace(/(\d{4})(\d+)(\d{4})/, "$1****$3")}
                        </td>
                        <td style={{ padding: "10px 12px", color: "#828cb4", fontSize: 11, fontFamily: "monospace" }}>
                          {session.session_id.slice(0, 12)}…
                        </td>
                        <td style={{ padding: "10px 12px" }}>
                          <span style={{
                            padding: "2px 8px", borderRadius: 4, fontSize: 11,
                            background: "rgba(95,114,249,0.1)", color: "#9cb7ff",
                          }}>
                            {session.current_menu || "root"}
                          </span>
                        </td>
                        <td style={{ padding: "10px 12px", color: "#828cb4", fontSize: 13 }}>
                          {session.step_count}
                        </td>
                        <td style={{ padding: "10px 12px", color: "#828cb4", fontSize: 12 }}>
                          {timeAgo(session.last_active)}
                        </td>
                        <td style={{ padding: "10px 12px" }}>
                          <span style={{
                            padding: "2px 8px", borderRadius: 4, fontSize: 11, fontWeight: 600,
                            background: session.is_active ? "rgba(34,197,94,0.15)" : "rgba(255,255,255,0.05)",
                            color: session.is_active ? "#22c55e" : "#4b5563",
                          }}>
                            {session.is_active ? "Active" : "Ended"}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          )}
        </>
      )}
    </AdminShell>
  );
}
