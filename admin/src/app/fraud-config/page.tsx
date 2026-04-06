"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState, useCallback } from "react";
import adminAPI, { ConfigEntry } from "@/lib/api";

// ─── Config keys — must match backend fraud_service.go reads exactly ──────────
const KEYS = {
  MAX_RECHARGE_24H:    "fraud_max_recharge_per_24h",
  MAX_SPIN_24H:        "fraud_max_spin_per_24h",
  MIN_RECHARGE_KOBO:   "fraud_min_recharge_kobo",
  DUPLICATE_WINDOW:    "fraud_duplicate_tx_window_seconds",
};

// ─── Config hook ──────────────────────────────────────────────────────────────
function useConfig() {
  const [configs, setConfigs] = useState<Record<string, string>>({});
  const [saving,  setSaving]  = useState<string | null>(null);
  const [saved,   setSaved]   = useState<string | null>(null);
  const [error,   setError]   = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await adminAPI.getConfig();
      const m: Record<string, string> = {};
      r.configs.forEach((c: ConfigEntry) => { m[c.key] = String(c.value); });
      setConfigs(m);
    } catch {
      setError("Failed to load fraud configuration");
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const save = async (key: string, value: string) => {
    setSaving(key);
    setError(null);
    try {
      await adminAPI.updateConfig(key, value);
      setConfigs(prev => ({ ...prev, [key]: value }));
      setSaved(key);
      setTimeout(() => setSaved(null), 2500);
    } catch {
      setError(`Failed to save ${key}`);
    } finally {
      setSaving(null);
    }
  };

  return { configs, saving, saved, error, save };
}

// ─── Section Header ───────────────────────────────────────────────────────────
function SectionHeader({ number, title, subtitle, color }: {
  number: string; title: string; subtitle: string; color: string;
}) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 16 }}>
      <div style={{
        width: 28, height: 28, borderRadius: "50%",
        background: `${color}22`, border: `1px solid ${color}44`,
        display: "flex", alignItems: "center", justifyContent: "center",
        color, fontSize: 12, fontWeight: 700, flexShrink: 0,
      }}>{number}</div>
      <div>
        <p style={{ color: "#e2e8f0", fontWeight: 700, fontSize: 15, margin: 0 }}>{title}</p>
        <p style={{ color: "#64748b", fontSize: 12, margin: "2px 0 0" }}>{subtitle}</p>
      </div>
      <div style={{ flex: 1, height: 1, background: "rgba(255,255,255,0.05)", marginLeft: 8 }} />
    </div>
  );
}

// ─── Config Card ──────────────────────────────────────────────────────────────
function ConfigCard({
  icon, title, description, configKey, configs, saving, saved, onSave,
  suffix = "", prefix = "", min = 0, step = 1, note, accentColor = "#5f72f9",
}: {
  icon: string; title: string; description: string; configKey: string;
  configs: Record<string, string>; saving: string | null; saved: string | null;
  onSave: (k: string, v: string) => void;
  suffix?: string; prefix?: string; min?: number; step?: number;
  note?: string; accentColor?: string;
}) {
  const raw = configs[configKey] ?? "";
  const [val, setVal] = useState(raw || "0");
  useEffect(() => { if (raw !== "") setVal(raw); }, [raw]);

  const isDirty  = val !== (raw || "0");
  const isSaving = saving === configKey;
  const isSaved  = saved  === configKey;

  return (
    <div style={{
      background: "#1c2038",
      border: `1px solid ${isSaved ? "rgba(34,197,94,0.4)" : "rgba(95,114,249,0.15)"}`,
      borderRadius: 12, padding: "20px", transition: "border-color 0.3s",
    }}>
      <div style={{ display: "flex", alignItems: "flex-start", gap: 12, marginBottom: 16 }}>
        <div style={{
          width: 40, height: 40, borderRadius: 10,
          background: `${accentColor}22`,
          display: "flex", alignItems: "center", justifyContent: "center",
          fontSize: 18, flexShrink: 0,
        }}>{icon}</div>
        <div style={{ flex: 1 }}>
          <p style={{ color: "#e2e8f0", fontWeight: 600, fontSize: 14, margin: 0 }}>{title}</p>
          <p style={{ color: "#94a3b8", fontSize: 12, margin: "4px 0 0", lineHeight: 1.5 }}>{description}</p>
        </div>
      </div>
      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
        {prefix && (
          <span style={{
            background: `${accentColor}18`, color: accentColor,
            padding: "8px 12px", borderRadius: 8, fontSize: 13, fontWeight: 600,
            border: `1px solid ${accentColor}33`,
          }}>{prefix}</span>
        )}
        <div style={{ flex: 1, position: "relative" }}>
          <input
            type="number" value={val} min={min} step={step}
            onChange={e => setVal(e.target.value)}
            style={{
              width: "100%", background: "#0f1629",
              border: `1px solid ${isDirty ? `${accentColor}80` : "rgba(255,255,255,0.08)"}`,
              borderRadius: 8, padding: suffix ? "9px 52px 9px 12px" : "9px 12px",
              color: "#e2e8f0", fontSize: 15, fontWeight: 600,
              outline: "none", boxSizing: "border-box", transition: "border-color 0.2s",
            }}
          />
          {suffix && (
            <span style={{
              position: "absolute", right: 12, top: "50%", transform: "translateY(-50%)",
              color: "#64748b", fontSize: 12, pointerEvents: "none",
            }}>{suffix}</span>
          )}
        </div>
        <button
          disabled={isSaving}
          onClick={() => onSave(configKey, val)}
          style={{
            padding: "9px 18px", borderRadius: 8, fontSize: 13, fontWeight: 600,
            border: "none", cursor: isSaving ? "not-allowed" : "pointer",
            background: isSaved
              ? "rgba(34,197,94,0.2)"
              : isDirty ? accentColor : `${accentColor}22`,
            color: isSaved ? "#4ade80" : isDirty ? "#fff" : accentColor,
            transition: "all 0.2s", whiteSpace: "nowrap", opacity: isSaving ? 0.6 : 1,
          }}
        >
          {isSaving ? "Saving…" : isSaved ? "✓ Saved" : "Save"}
        </button>
      </div>
      {note && (
        <p style={{ color: "#64748b", fontSize: 11, marginTop: 8, lineHeight: 1.5 }}>
          💡 {note}
        </p>
      )}
    </div>
  );
}

// ─── Impact Simulator ─────────────────────────────────────────────────────────
function ImpactSimulator({ configs }: { configs: Record<string, string> }) {
  const maxRecharge24h  = Number(configs[KEYS.MAX_RECHARGE_24H]  || 20);
  const maxSpin24h      = Number(configs[KEYS.MAX_SPIN_24H]      || 10);
  const minKobo         = Number(configs[KEYS.MIN_RECHARGE_KOBO] || 10000);
  const dupWindow       = Number(configs[KEYS.DUPLICATE_WINDOW]  || 300);

  const minNaira        = minKobo / 100;
  const dupWindowMins   = Math.round(dupWindow / 60);

  const scenarios = [
    {
      label: "Normal User",
      icon: "✅",
      color: "#10b981",
      recharges: 3,
      spins: 3,
      rechargeAmt: 500,
      hasDup: false,
    },
    {
      label: "Power User",
      icon: "⚠️",
      color: "#f59e0b",
      recharges: 12,
      spins: 8,
      rechargeAmt: 1000,
      hasDup: false,
    },
    {
      label: "Suspected Farmer",
      icon: "🚨",
      color: "#ef4444",
      recharges: maxRecharge24h + 5,
      spins: maxSpin24h + 3,
      rechargeAmt: minNaira - 10,
      hasDup: true,
    },
  ];

  return (
    <div style={{
      background: "linear-gradient(135deg, rgba(239,68,68,0.06) 0%, rgba(95,114,249,0.06) 100%)",
      border: "1px solid rgba(239,68,68,0.15)", borderRadius: 12, padding: "20px",
    }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 20 }}>
        <span style={{ fontSize: 16 }}>🔬</span>
        <p style={{ color: "#f87171", fontWeight: 700, fontSize: 13, margin: 0, letterSpacing: "0.05em" }}>
          IMPACT SIMULATOR — Current Thresholds
        </p>
      </div>

      {/* Threshold Summary */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 10, marginBottom: 20 }}>
        {[
          { label: "Max Recharges / 24h", value: `${maxRecharge24h}`, sub: "before flag", color: "#f87171" },
          { label: "Max Spins / 24h",     value: `${maxSpin24h}`,     sub: "before flag", color: "#fb923c" },
          { label: "Min Recharge",        value: `₦${minNaira}`,      sub: "micro-farm threshold", color: "#fbbf24" },
          { label: "Duplicate Window",    value: `${dupWindowMins} min`, sub: "same ref = fraud", color: "#a78bfa" },
        ].map(s => (
          <div key={s.label} style={{
            background: "rgba(15,22,41,0.6)", borderRadius: 8, padding: "12px 14px",
            border: "1px solid rgba(255,255,255,0.05)",
          }}>
            <p style={{ color: "#64748b", fontSize: 10, margin: "0 0 4px", textTransform: "uppercase", letterSpacing: "0.05em" }}>{s.label}</p>
            <p style={{ color: s.color, fontSize: 18, fontWeight: 700, margin: "0 0 2px" }}>{s.value}</p>
            <p style={{ color: "#475569", fontSize: 11, margin: 0 }}>{s.sub}</p>
          </div>
        ))}
      </div>

      {/* Scenario Cards */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 12 }}>
        {scenarios.map(s => {
          const rechargeFlag = s.recharges > maxRecharge24h;
          const spinFlag     = s.spins > maxSpin24h;
          const microFlag    = s.rechargeAmt < minNaira;
          const dupFlag      = s.hasDup;
          const flagCount    = [rechargeFlag, spinFlag, microFlag, dupFlag].filter(Boolean).length;

          return (
            <div key={s.label} style={{
              background: "rgba(15,22,41,0.7)",
              border: `1px solid ${flagCount > 0 ? `${s.color}44` : "rgba(255,255,255,0.06)"}`,
              borderRadius: 10, padding: "16px",
            }}>
              <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 12 }}>
                <span style={{ fontSize: 18 }}>{s.icon}</span>
                <div>
                  <p style={{ color: s.color, fontWeight: 700, fontSize: 13, margin: 0 }}>{s.label}</p>
                  <p style={{ color: "#64748b", fontSize: 11, margin: 0 }}>
                    {flagCount === 0 ? "No flags triggered" : `${flagCount} flag${flagCount > 1 ? "s" : ""} triggered`}
                  </p>
                </div>
              </div>
              <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                {[
                  { label: `${s.recharges} recharges/24h`, flagged: rechargeFlag, limit: `limit: ${maxRecharge24h}` },
                  { label: `${s.spins} spins/24h`,         flagged: spinFlag,     limit: `limit: ${maxSpin24h}` },
                  { label: `₦${s.rechargeAmt} recharge`,   flagged: microFlag,    limit: `min: ₦${minNaira}` },
                  { label: "Duplicate reference",           flagged: dupFlag,      limit: `window: ${dupWindowMins}m` },
                ].map(row => (
                  <div key={row.label} style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <span style={{ color: row.flagged ? "#f87171" : "#64748b", fontSize: 12 }}>
                      {row.flagged ? "⚑ " : "  "}{row.label}
                    </span>
                    <span style={{
                      fontSize: 10, padding: "2px 6px", borderRadius: 4,
                      background: row.flagged ? "rgba(239,68,68,0.15)" : "rgba(255,255,255,0.04)",
                      color: row.flagged ? "#f87171" : "#475569",
                    }}>{row.limit}</span>
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function FraudConfigPage() {
  const { configs, saving, saved, error, save } = useConfig();

  return (
    <AdminShell>
      <div style={{ maxWidth: 900, margin: "0 auto", padding: "32px 24px" }}>

        {/* Page Header */}
        <div style={{ marginBottom: 32 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 8 }}>
            <div style={{
              width: 44, height: 44, borderRadius: 12,
              background: "linear-gradient(135deg, #ef4444, #f97316)",
              display: "flex", alignItems: "center", justifyContent: "center", fontSize: 22,
            }}>🛡</div>
            <div>
              <h1 style={{ color: "#e2e8f0", fontSize: 22, fontWeight: 700, margin: 0 }}>Fraud Detection Config</h1>
              <p style={{ color: "#64748b", fontSize: 13, margin: "2px 0 0" }}>
                Configure thresholds that trigger automatic fraud flags — changes take effect immediately
              </p>
            </div>
            <div style={{ marginLeft: "auto" }}>
              <a href="/fraud" style={{
                display: "inline-flex", alignItems: "center", gap: 6,
                padding: "8px 16px", borderRadius: 8, textDecoration: "none",
                border: "1px solid rgba(239,68,68,0.3)", color: "#f87171",
                fontSize: 13, background: "rgba(239,68,68,0.08)",
              }}>
                🛡 View Fraud Alerts →
              </a>
            </div>
          </div>
          {error && (
            <div style={{
              background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.3)",
              borderRadius: 8, padding: "10px 14px", marginTop: 12, color: "#f87171", fontSize: 13,
            }}>⚠ {error}</div>
          )}
        </div>

        {/* Impact Simulator */}
        <div style={{ marginBottom: 32 }}>
          <ImpactSimulator configs={configs} />
        </div>

        {/* Section 1: Volume Limits */}
        <SectionHeader
          number="1" color="#ef4444"
          title="Volume Limits"
          subtitle="Flag users who exceed normal transaction frequency — catches reward farming bots"
        />
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16, marginBottom: 32 }}>
          <ConfigCard
            icon="🔄" title="Max Recharges per 24h"
            description="If a user submits more recharges than this in any 24-hour window, a HIGH severity fraud event is raised."
            configKey={KEYS.MAX_RECHARGE_24H}
            configs={configs} saving={saving} saved={saved} onSave={save}
            suffix="recharges" min={1} accentColor="#ef4444"
            note="Typical legitimate users: 1–5/day. Set too low and you'll flag power users. Recommended: 15–25."
          />
          <ConfigCard
            icon="🎡" title="Max Spins per 24h"
            description="If a user triggers more spins than this in any 24-hour window, a MEDIUM severity fraud event is raised."
            configKey={KEYS.MAX_SPIN_24H}
            configs={configs} saving={saving} saved={saved} onSave={save}
            suffix="spins" min={1} accentColor="#f97316"
            note="Daily spin limit (from Spin Config) already caps spins. This threshold catches API-level abuse. Recommended: daily limit + 2."
          />
        </div>

        {/* Section 2: Micro-Farming */}
        <SectionHeader
          number="2" color="#f59e0b"
          title="Micro-Farming Detection"
          subtitle="Identify users making abnormally small recharges to farm points cheaply"
        />
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16, marginBottom: 32 }}>
          <ConfigCard
            icon="💰" title="Minimum Recharge Amount"
            description="Recharges below this amount are logged as potential micro-farming. Value is in kobo (₦1 = 100 kobo)."
            configKey={KEYS.MIN_RECHARGE_KOBO}
            configs={configs} saving={saving} saved={saved} onSave={save}
            suffix="kobo" min={100} step={100} accentColor="#f59e0b"
            note={`Current value: ₦${Number(configs[KEYS.MIN_RECHARGE_KOBO] || 10000) / 100}. Enter 10000 for ₦100 minimum. Recommended: ₦50–₦200 (5000–20000 kobo).`}
          />
          <div style={{
            background: "#1c2038",
            border: "1px solid rgba(95,114,249,0.15)",
            borderRadius: 12, padding: "20px",
            display: "flex", flexDirection: "column", justifyContent: "center",
          }}>
            <div style={{ display: "flex", gap: 10, alignItems: "flex-start", marginBottom: 12 }}>
              <div style={{ width: 40, height: 40, borderRadius: 10, background: "rgba(245,158,11,0.15)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 18, flexShrink: 0 }}>📊</div>
              <div>
                <p style={{ color: "#e2e8f0", fontWeight: 600, fontSize: 14, margin: 0 }}>Kobo ↔ Naira Converter</p>
                <p style={{ color: "#94a3b8", fontSize: 12, margin: "4px 0 0" }}>Quick reference for setting the minimum</p>
              </div>
            </div>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 6 }}>
              {[
                ["₦50",  "5,000 kobo"],
                ["₦100", "10,000 kobo"],
                ["₦200", "20,000 kobo"],
                ["₦500", "50,000 kobo"],
              ].map(([naira, kobo]) => (
                <div key={naira} style={{ display: "flex", justifyContent: "space-between", background: "rgba(15,22,41,0.5)", borderRadius: 6, padding: "6px 10px" }}>
                  <span style={{ color: "#f59e0b", fontWeight: 600, fontSize: 12 }}>{naira}</span>
                  <span style={{ color: "#64748b", fontSize: 12 }}>{kobo}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Section 3: Duplicate Detection */}
        <SectionHeader
          number="3" color="#a78bfa"
          title="Duplicate Transaction Detection"
          subtitle="Prevent replay attacks and accidental double-processing of the same transaction"
        />
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16, marginBottom: 32 }}>
          <ConfigCard
            icon="🔁" title="Duplicate Reference Window"
            description="If the same transaction reference is seen again within this window, it is flagged as a duplicate and rejected. Value is in seconds."
            configKey={KEYS.DUPLICATE_WINDOW}
            configs={configs} saving={saving} saved={saved} onSave={save}
            suffix="seconds" min={60} step={60} accentColor="#a78bfa"
            note={`Current window: ${Math.round(Number(configs[KEYS.DUPLICATE_WINDOW] || 300) / 60)} minutes. Recommended: 300s (5 min) to 900s (15 min). Too short risks missing retries; too long blocks legitimate reprocessing.`}
          />
          <div style={{
            background: "#1c2038",
            border: "1px solid rgba(95,114,249,0.15)",
            borderRadius: 12, padding: "20px",
          }}>
            <div style={{ display: "flex", gap: 10, alignItems: "flex-start", marginBottom: 16 }}>
              <div style={{ width: 40, height: 40, borderRadius: 10, background: "rgba(167,139,250,0.15)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 18, flexShrink: 0 }}>⏱</div>
              <div>
                <p style={{ color: "#e2e8f0", fontWeight: 600, fontSize: 14, margin: 0 }}>Seconds ↔ Minutes</p>
                <p style={{ color: "#94a3b8", fontSize: 12, margin: "4px 0 0" }}>Quick reference for setting the window</p>
              </div>
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              {[
                ["1 minute",  "60 seconds",  "Too short — may miss webhook retries"],
                ["5 minutes", "300 seconds", "Recommended — covers most retry windows"],
                ["10 minutes","600 seconds", "Conservative — good for slow networks"],
                ["15 minutes","900 seconds", "Maximum — use only if retries are slow"],
              ].map(([label, secs, note]) => (
                <div key={secs} style={{ display: "flex", gap: 8, alignItems: "center", background: "rgba(15,22,41,0.5)", borderRadius: 6, padding: "7px 10px" }}>
                  <span style={{ color: "#a78bfa", fontWeight: 600, fontSize: 12, minWidth: 70 }}>{label}</span>
                  <span style={{ color: "#64748b", fontSize: 12, minWidth: 80 }}>{secs}</span>
                  <span style={{ color: "#475569", fontSize: 11 }}>{note}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Footer Note */}
        <div style={{
          background: "rgba(95,114,249,0.06)",
          border: "1px solid rgba(95,114,249,0.15)",
          borderRadius: 10, padding: "14px 18px",
          display: "flex", gap: 10, alignItems: "flex-start",
        }}>
          <span style={{ fontSize: 16, flexShrink: 0 }}>ℹ️</span>
          <div>
            <p style={{ color: "#818cf8", fontWeight: 600, fontSize: 13, margin: "0 0 4px" }}>How Fraud Events Work</p>
            <p style={{ color: "#64748b", fontSize: 12, margin: 0, lineHeight: 1.6 }}>
              When a threshold is exceeded, a fraud event is logged in the{" "}
              <a href="/fraud" style={{ color: "#818cf8", textDecoration: "underline" }}>Fraud Alerts</a>{" "}
              page with severity HIGH, MEDIUM, or LOW. The user&apos;s transaction is still processed — fraud events are
              investigative flags, not automatic blocks. Admins can review and resolve each event with notes.
              To auto-block users, contact engineering to enable the account suspension flow.
            </p>
          </div>
        </div>

      </div>
    </AdminShell>
  );
}
