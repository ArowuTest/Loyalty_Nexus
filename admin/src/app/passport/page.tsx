"use client";

import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { ConfigEntry, PassportStats, GhostNudgeLog, USSDSession } from "@/lib/api";


// ─── Config keys (zero-hardcoding — all values live in network_configs) ───────

const PASSPORT_CONFIG_KEYS = {
  NUDGE_INTERVAL:       "ghost_nudge_interval_minutes",
  NUDGE_WARNING_HOURS:  "ghost_nudge_warning_hours",
  NUDGE_MIN_STREAK:     "ghost_nudge_min_streak",
  NUDGE_COOLDOWN:       "ghost_nudge_cooldown_hours",
  NUDGE_BATCH_LIMIT:    "ghost_nudge_batch_limit",
  NUDGE_SMS_ENABLED:    "ghost_nudge_sms_enabled",
  NUDGE_WALLET_ENABLED: "ghost_nudge_wallet_push_enabled",
  USSD_SHORT_CODE:      "ussd_short_code",
  USSD_TIMEOUT:         "ussd_session_timeout_seconds",
  USSD_MAX_DEPTH:       "ussd_max_menu_depth",
  // Dashboard banner
  BANNER_ENABLED:       "passport_banner_enabled",
  BANNER_TITLE:         "passport_banner_title",
  BANNER_SUBTITLE:      "passport_banner_subtitle",
  BANNER_CTA_IOS:       "passport_banner_cta_ios",
  BANNER_CTA_ANDROID:   "passport_banner_cta_android",
  // Wallet card messages
  WALLET_STREAK_EXPIRY_ENABLED:  "wallet_streak_expiry_enabled",
  WALLET_STREAK_EXPIRY_MSG:      "wallet_streak_expiry_message",
  WALLET_SPIN_READY_ENABLED:     "wallet_spin_ready_enabled",
  WALLET_SPIN_READY_MSG:         "wallet_spin_ready_message",
  WALLET_TIER_UPGRADE_ENABLED:   "wallet_tier_upgrade_enabled",
  WALLET_TIER_UPGRADE_MSG:       "wallet_tier_upgrade_message",
  WALLET_PRIZE_WON_ENABLED:      "wallet_prize_won_enabled",
  WALLET_PRIZE_WON_MSG:          "wallet_prize_won_message",
  // Broadcast
  WALLET_BROADCAST_ENABLED:      "wallet_broadcast_enabled",
  WALLET_BROADCAST_LABEL:        "wallet_broadcast_label",
  WALLET_BROADCAST_MSG:          "wallet_broadcast_message",
};

// ─── Config hook (same pattern as points-config/page.tsx) ─────────────────────

function usePassportConfig() {
  const [configs, setConfigs]   = useState<Record<string, string>>({});
  const [saving, setSaving]     = useState<string | null>(null);
  const [saved, setSaved]       = useState<string | null>(null);

  const load = useCallback(async () => {
    const r = await adminAPI.getConfig();
    const m: Record<string, string> = {};
    r.configs.forEach((c: ConfigEntry) => { m[c.key] = String(c.value); });
    setConfigs(m);
  }, []);

  useEffect(() => { load(); }, [load]);

  const save = async (key: string, value: string) => {
    setSaving(key);
    try {
      await adminAPI.updateConfig(key, value);
      setConfigs(prev => ({ ...prev, [key]: value }));
      setSaved(key);
      setTimeout(() => setSaved(null), 2000);
    } finally { setSaving(null); }
  };

  return { configs, saving, saved, save };
}

// ─── Reusable field components (matching points-config style) ─────────────────

function NumberField({
  label, desc, configKey, configs, saving, saved, onSave, suffix = "",
}: {
  label: string; desc: string; configKey: string;
  configs: Record<string, string>; saving: string | null; saved: string | null;
  onSave: (k: string, v: string) => void; suffix?: string;
}) {
  const [val, setVal] = useState(configs[configKey] ?? "");
  useEffect(() => { setVal(configs[configKey] ?? ""); }, [configs, configKey]);

  return (
    <div style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(95,114,249,0.12)", borderRadius: 10, padding: 16 }}>
      <label style={{ display: "block", fontSize: 13, fontWeight: 600, color: "#e2e8ff", marginBottom: 4 }}>{label}</label>
      <p style={{ fontSize: 12, color: "#828cb4", marginBottom: 12, margin: "4px 0 12px" }}>{desc}</p>
      <div style={{ display: "flex", gap: 8 }}>
        <div style={{ position: "relative", flex: 1 }}>
          <input
            type="number"
            value={val}
            onChange={e => setVal(e.target.value)}
            style={{
              width: "100%", background: "#131629", border: "1px solid rgba(95,114,249,0.2)",
              borderRadius: 8, padding: "8px 36px 8px 12px", fontSize: 13, color: "#e2e8ff",
              outline: "none", boxSizing: "border-box",
            }}
          />
          {suffix && (
            <span style={{ position: "absolute", right: 10, top: "50%", transform: "translateY(-50%)", fontSize: 11, color: "#828cb4" }}>
              {suffix}
            </span>
          )}
        </div>
        <button
          disabled={saving === configKey}
          onClick={() => onSave(configKey, val)}
          style={{
            padding: "8px 16px", borderRadius: 8, border: "none", cursor: "pointer",
            fontSize: 13, fontWeight: 500,
            background: saved === configKey ? "#16a34a" : "#5f72f9",
            color: "#fff", opacity: saving === configKey ? 0.5 : 1,
            transition: "background 0.2s",
          }}
        >
          {saving === configKey ? "…" : saved === configKey ? "✓ Saved" : "Save"}
        </button>
      </div>
    </div>
  );
}

function ToggleField({
  label, desc, configKey, configs, saving, saved, onSave,
}: {
  label: string; desc: string; configKey: string;
  configs: Record<string, string>; saving: string | null; saved: string | null;
  onSave: (k: string, v: string) => void;
}) {
  const isOn = configs[configKey] === "true";
  return (
    <div style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(95,114,249,0.12)", borderRadius: 10, padding: 16 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
        <div style={{ flex: 1, marginRight: 16 }}>
          <label style={{ display: "block", fontSize: 13, fontWeight: 600, color: "#e2e8ff", marginBottom: 4 }}>{label}</label>
          <p style={{ fontSize: 12, color: "#828cb4", margin: 0 }}>{desc}</p>
        </div>
        <button
          disabled={saving === configKey}
          onClick={() => onSave(configKey, isOn ? "false" : "true")}
          style={{
            width: 48, height: 26, borderRadius: 13, border: "none", cursor: "pointer",
            background: isOn ? "#5f72f9" : "rgba(255,255,255,0.1)",
            position: "relative", transition: "background 0.2s", flexShrink: 0,
          }}
        >
          <span style={{
            position: "absolute", top: 3, left: isOn ? 25 : 3,
            width: 20, height: 20, borderRadius: "50%", background: "#fff",
            transition: "left 0.2s", display: "block",
          }} />
        </button>
      </div>
      {saved === configKey && (
        <p style={{ fontSize: 11, color: "#22c55e", marginTop: 6, margin: "6px 0 0" }}>✓ Saved</p>
      )}
    </div>
  );
}

function TextField({
  label, desc, configKey, configs, saving, saved, onSave,
}: {
  label: string; desc: string; configKey: string;
  configs: Record<string, string>; saving: string | null; saved: string | null;
  onSave: (k: string, v: string) => void;
}) {
  const [val, setVal] = useState(configs[configKey] ?? "");
  useEffect(() => { setVal(configs[configKey] ?? ""); }, [configs, configKey]);

  return (
    <div style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(95,114,249,0.12)", borderRadius: 10, padding: 16 }}>
      <label style={{ display: "block", fontSize: 13, fontWeight: 600, color: "#e2e8ff", marginBottom: 4 }}>{label}</label>
      <p style={{ fontSize: 12, color: "#828cb4", marginBottom: 12, margin: "4px 0 12px" }}>{desc}</p>
      <div style={{ display: "flex", gap: 8 }}>
        <input
          type="text"
          value={val}
          onChange={e => setVal(e.target.value)}
          style={{
            flex: 1, background: "#131629", border: "1px solid rgba(95,114,249,0.2)",
            borderRadius: 8, padding: "8px 12px", fontSize: 13, color: "#e2e8ff",
            outline: "none",
          }}
        />
        <button
          disabled={saving === configKey}
          onClick={() => onSave(configKey, val)}
          style={{
            padding: "8px 16px", borderRadius: 8, border: "none", cursor: "pointer",
            fontSize: 13, fontWeight: 500,
            background: saved === configKey ? "#16a34a" : "#5f72f9",
            color: "#fff", opacity: saving === configKey ? 0.5 : 1,
            transition: "background 0.2s",
          }}
        >
          {saving === configKey ? "…" : saved === configKey ? "✓ Saved" : "Save"}
        </button>
      </div>
    </div>
  );
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
  const [activeTab, setActiveTab]   = useState<"overview" | "nudges" | "ussd" | "config">("overview");
  const [autoRefresh, setAutoRefresh] = useState(false);

  const { configs, saving, saved, save } = usePassportConfig();

  const load = useCallback(async () => {
    try {
      const [s, n, u] = await Promise.all([
        adminAPI.getPassportStats(),
        adminAPI.getPassportNudgeLog(50),
        adminAPI.getUSSDSessions(50),
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

  useEffect(() => { load(); }, [load]);

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

  const sectionHeadStyle: React.CSSProperties = {
    fontSize: 13,
    fontWeight: 700,
    color: "#9ca3af",
    textTransform: "uppercase",
    letterSpacing: "0.08em",
    marginBottom: 12,
    marginTop: 24,
  };

  return (
    <AdminShell>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <div>
          <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff", margin: 0 }}>
            🪪 Digital Passport &amp; USSD
          </h1>
          <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
            Wallet pass activity, ghost nudge logs, USSD session monitor, and configuration
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
              { label: "Total Passports",        value: stats?.total_passports ?? 0,         icon: "🪪" },
              { label: "Apple Wallet Downloads",  value: stats?.apple_wallet_downloads ?? 0,  icon: "📱" },
              { label: "Google Wallet Saves",     value: stats?.google_wallet_saves ?? 0,     icon: "💳" },
              { label: "QR Scans Today",          value: stats?.qr_scans_today ?? 0,          icon: "📷" },
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
            {(["overview", "nudges", "ussd", "config"] as const).map(tab => (
              <button key={tab} onClick={() => setActiveTab(tab)} style={tabStyle(activeTab === tab)}>
                {tab === "overview" ? "🏆 Top Earners"
                  : tab === "nudges" ? "👻 Ghost Nudges"
                  : tab === "ussd"   ? "📱 USSD Sessions"
                  :                    "⚙️ Configuration"}
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
                <p style={{ color: "#828cb4", fontSize: 13 }}>
                  No nudges sent yet. The Ghost Nudge worker fires every{" "}
                  {configs[PASSPORT_CONFIG_KEYS.NUDGE_INTERVAL] ?? "60"} minutes.
                </p>
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

          {/* Tab: Configuration (zero-hardcoding — all values from network_configs) */}
          {activeTab === "config" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 0 }}>
              <div style={{ ...cardStyle, marginBottom: 16 }}>
                <p style={{ color: "#828cb4", fontSize: 13, margin: 0 }}>
                  All values below are stored in the <code style={{ color: "#9cb7ff" }}>network_configs</code> table
                  and take effect on the next Ghost Nudge worker tick — no code deploy required.
                </p>
              </div>

              {/* Ghost Nudge Timing */}
              <p style={sectionHeadStyle}>👻 Ghost Nudge — Timing &amp; Thresholds</p>
              <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))", gap: 12, marginBottom: 8 }}>
                <NumberField
                  label="Cron Interval"
                  desc="How often the Ghost Nudge worker runs. REQ-4.4 mandates 60 min default."
                  configKey={PASSPORT_CONFIG_KEYS.NUDGE_INTERVAL}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                  suffix="min"
                />
                <NumberField
                  label="Warning Window"
                  desc="Hours before streak expiry to trigger a nudge. REQ-4.4 mandates 4h default."
                  configKey={PASSPORT_CONFIG_KEYS.NUDGE_WARNING_HOURS}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                  suffix="hrs"
                />
                <NumberField
                  label="Minimum Streak"
                  desc="Minimum streak length (days) to qualify for a Ghost Nudge. REQ-4.4 mandates 3 days."
                  configKey={PASSPORT_CONFIG_KEYS.NUDGE_MIN_STREAK}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                  suffix="days"
                />
                <NumberField
                  label="Nudge Cooldown"
                  desc="Minimum hours before the same user can be nudged again. Prevents SMS spam."
                  configKey={PASSPORT_CONFIG_KEYS.NUDGE_COOLDOWN}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                  suffix="hrs"
                />
                <NumberField
                  label="Batch Limit"
                  desc="Maximum users to nudge per cron run. Prevents runaway SMS costs."
                  configKey={PASSPORT_CONFIG_KEYS.NUDGE_BATCH_LIMIT}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                  suffix="users"
                />
              </div>

              {/* Ghost Nudge Channels */}
              <p style={sectionHeadStyle}>📡 Ghost Nudge — Channels</p>
              <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))", gap: 12, marginBottom: 8 }}>
                <ToggleField
                  label="SMS Nudges Enabled"
                  desc="Send Termii SMS to users during Ghost Nudge runs. Disable to pause SMS without stopping wallet pushes."
                  configKey={PASSPORT_CONFIG_KEYS.NUDGE_SMS_ENABLED}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <ToggleField
                  label="Wallet Pass Push Enabled"
                  desc="Push updated Apple/Google Wallet passes during Ghost Nudge runs. Disable to pause wallet pushes independently."
                  configKey={PASSPORT_CONFIG_KEYS.NUDGE_WALLET_ENABLED}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
              </div>

              {/* Dashboard Banner */}
              <p style={sectionHeadStyle}>🎫 Dashboard Passport Banner</p>
              <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))", gap: 12, marginBottom: 8 }}>
                <ToggleField
                  label="Banner Enabled"
                  desc="Show the 'Download Your Passport' banner on the user dashboard. Disable to hide it globally."
                  configKey={PASSPORT_CONFIG_KEYS.BANNER_ENABLED}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="Banner Title"
                  desc="Main heading shown on the dashboard passport banner."
                  configKey={PASSPORT_CONFIG_KEYS.BANNER_TITLE}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="Banner Subtitle"
                  desc="Supporting text below the banner title."
                  configKey={PASSPORT_CONFIG_KEYS.BANNER_SUBTITLE}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="iOS CTA Text"
                  desc="Button text shown to iPhone users on the banner."
                  configKey={PASSPORT_CONFIG_KEYS.BANNER_CTA_IOS}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="Android CTA Text"
                  desc="Button text shown to Android users on the banner."
                  configKey={PASSPORT_CONFIG_KEYS.BANNER_CTA_ANDROID}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
              </div>

              {/* Wallet Card Messages */}
              <p style={sectionHeadStyle}>💳 Wallet Card Lock-Screen Messages</p>
              <div style={{ ...cardStyle, marginBottom: 12, padding: "10px 14px" }}>
                <p style={{ color: "#828cb4", fontSize: 12, margin: 0 }}>
                  These messages appear on the user&apos;s lock screen via Apple Wallet / Google Wallet.
                  Changes take effect on the next Ghost Nudge tick — no deploy needed.
                  <strong style={{ color: "#9cb7ff" }}> Broadcast overrides all other messages when enabled.</strong>
                </p>
              </div>
              <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))", gap: 12, marginBottom: 8 }}>
                <ToggleField
                  label="Streak Expiry Push Enabled"
                  desc="Push a wallet card update when a user's streak is about to expire."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_STREAK_EXPIRY_ENABLED}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="Streak Expiry Message"
                  desc="Text shown on the wallet card when streak is expiring."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_STREAK_EXPIRY_MSG}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <ToggleField
                  label="Spin Ready Push Enabled"
                  desc="Push a wallet card update when the user has spin credits available."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_SPIN_READY_ENABLED}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="Spin Ready Message"
                  desc="Text shown on the wallet card when a free spin is available."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_SPIN_READY_MSG}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <ToggleField
                  label="Tier Upgrade Push Enabled"
                  desc="Push a wallet card update when the user is promoted to a new tier."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_TIER_UPGRADE_ENABLED}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="Tier Upgrade Message"
                  desc="Text shown on the wallet card after a tier upgrade."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_TIER_UPGRADE_MSG}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <ToggleField
                  label="Prize Won Push Enabled"
                  desc="Push a wallet card update when the user has an unclaimed prize."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_PRIZE_WON_ENABLED}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="Prize Won Message"
                  desc="Text shown on the wallet card when an unclaimed prize is waiting."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_PRIZE_WON_MSG}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
              </div>

              {/* Broadcast Message */}
              <p style={sectionHeadStyle}>📢 Broadcast Message (overrides all card messages)</p>
              <div style={{ ...cardStyle, marginBottom: 12, padding: "10px 14px", border: "1px solid rgba(234,179,8,0.3)" }}>
                <p style={{ color: "#fbbf24", fontSize: 12, margin: 0 }}>
                  ⚠️ When broadcast is enabled, ALL wallet cards will show this message regardless of user state.
                  Use for promotions, double-points events, or announcements. Disable when done.
                </p>
              </div>
              <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))", gap: 12, marginBottom: 8 }}>
                <ToggleField
                  label="Broadcast Enabled"
                  desc="When ON, the broadcast message overrides all other wallet card messages for every user."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_BROADCAST_ENABLED}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="Broadcast Label"
                  desc="Short label shown above the broadcast message on the wallet card (e.g. '📢 ANNOUNCEMENT')."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_BROADCAST_LABEL}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <TextField
                  label="Broadcast Message"
                  desc="The message shown on every user's wallet card lock screen. Keep it under 40 characters."
                  configKey={PASSPORT_CONFIG_KEYS.WALLET_BROADCAST_MSG}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
              </div>

              {/* USSD Settings */}
              <p style={sectionHeadStyle}>📱 USSD Settings</p>
              <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))", gap: 12, marginBottom: 8 }}>
                <TextField
                  label="USSD Short Code"
                  desc="The USSD short code displayed in SMS nudges and on wallet pass back fields."
                  configKey={PASSPORT_CONFIG_KEYS.USSD_SHORT_CODE}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                />
                <NumberField
                  label="Session Timeout"
                  desc="USSD session inactivity timeout. Africa's Talking drops sessions after 20s."
                  configKey={PASSPORT_CONFIG_KEYS.USSD_TIMEOUT}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                  suffix="sec"
                />
                <NumberField
                  label="Max Menu Depth"
                  desc="Maximum USSD menu levels a user can navigate in a single session."
                  configKey={PASSPORT_CONFIG_KEYS.USSD_MAX_DEPTH}
                  configs={configs} saving={saving} saved={saved} onSave={save}
                  suffix="levels"
                />
              </div>
            </div>
          )}
        </>
      )}
    </AdminShell>
  );
}
