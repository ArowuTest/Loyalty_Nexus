"use client";
import { useState, useEffect } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { ConfigEntry } from "@/lib/api";

// ── Config key metadata: group, label, type hint, and validation ──────────────
type ConfigType = "integer" | "float" | "boolean" | "string" | "json";

interface ConfigMeta {
  group: string;
  label: string;
  type: ConfigType;
  hint?: string;
}

const CONFIG_META: Record<string, ConfigMeta> = {
  // Loyalty Programme
  points_per_naira:              { group: "Loyalty Programme", label: "Points per ₦1 recharge",         type: "float",   hint: "e.g. 0.004 = 1 point per ₦250" },
  spin_credits_per_naira:        { group: "Loyalty Programme", label: "Spin credits per ₦1 recharge",   type: "float",   hint: "e.g. 0.001 = 1 credit per ₦1000" },
  points_multiplier:             { group: "Loyalty Programme", label: "Global points multiplier",        type: "float",   hint: "1.0 = normal, 2.0 = double points event" },
  streak_window_hours:           { group: "Loyalty Programme", label: "Streak window (hours)",           type: "integer", hint: "Hours within which a recharge counts toward streak" },
  streak_bonus_pct:              { group: "Loyalty Programme", label: "Streak bonus (%)",               type: "integer", hint: "Extra % points for maintaining a streak" },
  operation_mode:                { group: "Loyalty Programme", label: "Operation mode",                  type: "string",  hint: "'independent' (Paystack) or 'integrated' (MNO webhook)" },

  // Spin & Win
  daily_spin_limit:              { group: "Spin & Win", label: "Daily spin limit per user",    type: "integer", hint: "Max spins per user per day" },
  daily_prize_liability_cap:     { group: "Spin & Win", label: "Daily prize liability cap (₦)", type: "integer", hint: "Max total prize value paid out per day" },
  spin_draw_window_hours:        { group: "Spin & Win", label: "Draw window (hours)",          type: "integer", hint: "Hours between draw windows" },
  spin_enabled:                  { group: "Spin & Win", label: "Spin & Win enabled",           type: "boolean", hint: "'true' or 'false'" },

  // Fraud Detection
  fraud_max_recharge_per_24h:        { group: "Fraud Detection", label: "Max recharges per 24h",         type: "integer", hint: "Flag user if they exceed this many recharges in 24h" },
  fraud_max_spin_per_24h:            { group: "Fraud Detection", label: "Max spins per 24h",             type: "integer", hint: "Flag user if they exceed this many spins in 24h" },
  fraud_min_recharge_kobo:           { group: "Fraud Detection", label: "Min recharge (kobo)",           type: "integer", hint: "Recharges below this are logged as micro-farming (₦100 = 10000)" },
  fraud_duplicate_tx_window_seconds: { group: "Fraud Detection", label: "Duplicate tx window (seconds)", type: "integer", hint: "Duplicate reference within this window is flagged (300 = 5 min)" },

  // AI Studio
  chat_groq_daily_limit:         { group: "AI Studio", label: "Chat daily limit (Groq)",       type: "integer", hint: "Max AI chat requests per day across all users" },
  chat_gemini_daily_limit:       { group: "AI Studio", label: "Chat daily limit (Gemini)",     type: "integer", hint: "Max AI chat requests per day across all users" },
  studio_session_timeout_mins:   { group: "AI Studio", label: "Studio session timeout (mins)", type: "integer", hint: "Inactivity timeout for AI Studio sessions" },

  // Regional Wars
  wars_enabled:                  { group: "Regional Wars", label: "Regional Wars enabled",    type: "boolean", hint: "'true' or 'false'" },
  wars_season_duration_days:     { group: "Regional Wars", label: "Season duration (days)",   type: "integer", hint: "Length of each Regional Wars season" },
  wars_prize_pool_naira:         { group: "Regional Wars", label: "Prize pool (₦)",           type: "integer", hint: "Total prize pool for the current season" },

  // Notifications
  ghost_nudge_interval_hours:    { group: "Notifications", label: "Ghost nudge interval (hours)", type: "integer", hint: "Hours of inactivity before sending a nudge" },
  streak_alert_threshold:        { group: "Notifications", label: "Streak alert threshold",       type: "integer", hint: "Days before streak expires to send alert" },

  // USSD
  ussd_session_timeout_secs:     { group: "USSD", label: "USSD session timeout (secs)", type: "integer", hint: "Seconds before a USSD session expires" },
  ussd_shortcode:                { group: "USSD", label: "USSD shortcode",              type: "string",  hint: "e.g. *347*100#" },
};

function getGroup(key: string): string {
  return CONFIG_META[key]?.group ?? "Other";
}

function getLabel(key: string): string {
  return CONFIG_META[key]?.label ?? key;
}

function getType(key: string): ConfigType {
  return CONFIG_META[key]?.type ?? "string";
}

function getHint(key: string): string | undefined {
  return CONFIG_META[key]?.hint;
}

function validateValue(key: string, value: string): string | null {
  const type = getType(key);
  if (type === "integer") {
    if (!/^-?\d+$/.test(value.trim())) return "Must be a whole number (e.g. 20)";
  } else if (type === "float") {
    if (isNaN(parseFloat(value))) return "Must be a decimal number (e.g. 0.004)";
  } else if (type === "boolean") {
    if (value !== "true" && value !== "false") return "Must be 'true' or 'false'";
  } else if (type === "json") {
    try { JSON.parse(value); } catch { return "Must be valid JSON"; }
  }
  return null;
}

const GROUP_ORDER = [
  "Loyalty Programme",
  "Spin & Win",
  "Fraud Detection",
  "AI Studio",
  "Regional Wars",
  "Notifications",
  "USSD",
  "Other",
];

export default function ConfigPage() {
  const [configs, setConfigs]   = useState<ConfigEntry[]>([]);
  const [editing, setEditing]   = useState<{ key: string; value: string } | null>(null);
  const [saving, setSaving]     = useState(false);
  const [saved, setSaved]       = useState(false);
  const [search, setSearch]     = useState("");
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({});

  useEffect(() => {
    adminAPI.getConfig().then(r => {
      setConfigs(r.configs);
      // Default: open all groups
      const groups = Array.from(new Set(r.configs.map((c: ConfigEntry) => getGroup(c.key))));
      const initial: Record<string, boolean> = {};
      groups.forEach((g: string) => { initial[g] = true; });
      setOpenGroups(initial);
    }).catch(console.error);
  }, []);

  const save = async () => {
    if (!editing) return;
    const err = validateValue(editing.key, editing.value);
    if (err) { alert(err); return; }
    setSaving(true);
    await adminAPI.updateConfig(editing.key, editing.value).catch(console.error);
    setConfigs(c => c.map(e => e.key === editing.key ? { ...e, value: editing.value } : e));
    setEditing(null);
    setSaving(false);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  const filtered = configs.filter(c =>
    c.key.toLowerCase().includes(search.toLowerCase()) ||
    String(c.value).toLowerCase().includes(search.toLowerCase()) ||
    (c.description ?? "").toLowerCase().includes(search.toLowerCase()) ||
    getLabel(c.key).toLowerCase().includes(search.toLowerCase())
  );

  // Group and sort
  const grouped: Record<string, ConfigEntry[]> = {};
  filtered.forEach(c => {
    const g = getGroup(c.key);
    if (!grouped[g]) grouped[g] = [];
    grouped[g].push(c);
  });

  const sortedGroups = GROUP_ORDER.filter(g => grouped[g]?.length > 0)
    .concat(Object.keys(grouped).filter(g => !GROUP_ORDER.includes(g)));

  const toggleGroup = (g: string) => setOpenGroups(prev => ({ ...prev, [g]: !prev[g] }));

  const editingValidationError = editing ? validateValue(editing.key, editing.value) : null;

  return (
    <AdminShell>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
        <div>
          <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff", marginBottom: 4 }}>⚙️ Configuration</h1>
          <p style={{ color: "#828cb4", fontSize: 13 }}>
            {configs.length} settings · All changes take effect immediately
          </p>
        </div>
        <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
          {saved && <span style={{ color: "#4ade80", fontSize: 13 }}>✓ Saved</span>}
          <input
            className="input"
            placeholder="Search settings…"
            value={search}
            onChange={e => setSearch(e.target.value)}
            style={{ width: 240 }}
          />
        </div>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
        {sortedGroups.map(group => (
          <div key={group} className="card" style={{ overflow: "hidden" }}>
            {/* Group header */}
            <div
              style={{ padding: "14px 20px", display: "flex", justifyContent: "space-between", alignItems: "center", cursor: "pointer", borderBottom: openGroups[group] ? "1px solid rgba(95,114,249,0.1)" : "none" }}
              onClick={() => toggleGroup(group)}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                <span style={{ color: "#e2e8ff", fontWeight: 600, fontSize: 14 }}>{group}</span>
                <span style={{ background: "rgba(95,114,249,0.15)", color: "#5f72f9", borderRadius: 20, padding: "2px 8px", fontSize: 11, fontWeight: 600 }}>
                  {grouped[group].length}
                </span>
              </div>
              <span style={{ color: "#828cb4", fontSize: 16 }}>{openGroups[group] ? "▲" : "▼"}</span>
            </div>

            {/* Group rows */}
            {openGroups[group] && (
              <table style={{ width: "100%", borderCollapse: "collapse" }}>
                <thead>
                  <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.08)" }}>
                    {["Setting", "Current Value", "Description", ""].map(h => (
                      <th key={h} style={{ padding: "10px 20px", textAlign: "left", color: "#828cb4", fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em" }}>{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {grouped[group].map((c, i) => (
                    <tr key={c.key} style={{ borderBottom: i < grouped[group].length - 1 ? "1px solid rgba(95,114,249,0.05)" : "none" }}>
                      <td style={{ padding: "12px 20px", minWidth: 200 }}>
                        <div style={{ color: "#e2e8ff", fontSize: 13, fontWeight: 500, marginBottom: 2 }}>{getLabel(c.key)}</div>
                        <div style={{ color: "#5f72f9", fontFamily: "monospace", fontSize: 11 }}>{c.key}</div>
                      </td>
                      <td style={{ padding: "12px 20px", minWidth: 140 }}>
                        <span style={{
                          background: "rgba(95,114,249,0.1)",
                          color: "#a5b4fc",
                          fontFamily: "monospace",
                          fontSize: 13,
                          padding: "3px 8px",
                          borderRadius: 6,
                          display: "inline-block",
                          maxWidth: 200,
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                        }}>
                          {String(c.value).slice(0, 50)}{String(c.value).length > 50 ? "…" : ""}
                        </span>
                      </td>
                      <td style={{ padding: "12px 20px", color: "#828cb4", fontSize: 12, maxWidth: 300 }}>
                        {getHint(c.key) ?? c.description ?? "—"}
                      </td>
                      <td style={{ padding: "12px 20px" }}>
                        <button
                          className="btn-outline"
                          style={{ fontSize: 12, padding: "4px 12px" }}
                          onClick={() => setEditing({ key: c.key, value: String(c.value) })}
                        >
                          Edit
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        ))}
      </div>

      {/* Edit modal */}
      {editing && (
        <div
          style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 50 }}
          onClick={() => setEditing(null)}
        >
          <div className="card" style={{ width: 480, padding: 28 }} onClick={e => e.stopPropagation()}>
            <div style={{ marginBottom: 20 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
                <span style={{ background: "rgba(95,114,249,0.15)", color: "#5f72f9", borderRadius: 6, padding: "2px 8px", fontSize: 11, fontWeight: 600 }}>
                  {getGroup(editing.key)}
                </span>
                <span style={{ background: "rgba(255,255,255,0.06)", color: "#828cb4", borderRadius: 6, padding: "2px 8px", fontSize: 11, fontWeight: 600 }}>
                  {getType(editing.key)}
                </span>
              </div>
              <h3 style={{ color: "#e2e8ff", fontWeight: 700, fontSize: 16, marginBottom: 4 }}>{getLabel(editing.key)}</h3>
              <p style={{ color: "#5f72f9", fontFamily: "monospace", fontSize: 12, marginBottom: 8 }}>{editing.key}</p>
              {getHint(editing.key) && (
                <p style={{ color: "#828cb4", fontSize: 12 }}>{getHint(editing.key)}</p>
              )}
            </div>
            <input
              className="input"
              value={editing.value}
              onChange={e => setEditing({ ...editing, value: e.target.value })}
              style={{ marginBottom: 8 }}
              autoFocus
            />
            {editingValidationError && (
              <p style={{ color: "#f87171", fontSize: 12, marginBottom: 12 }}>⚠ {editingValidationError}</p>
            )}
            {!editingValidationError && <div style={{ marginBottom: 12 }} />}
            <div style={{ display: "flex", gap: 10 }}>
              <button className="btn-outline" style={{ flex: 1 }} onClick={() => setEditing(null)}>Cancel</button>
              <button
                className="btn-primary"
                style={{ flex: 1 }}
                onClick={save}
                disabled={saving || !!editingValidationError}
              >
                {saving ? "Saving…" : "Save Changes"}
              </button>
            </div>
          </div>
        </div>
      )}
    </AdminShell>
  );
}
