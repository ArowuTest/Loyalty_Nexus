"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState, useCallback } from "react";
import adminAPI, { ConfigEntry } from "@/lib/api";

// ─── Config keys — must match backend service reads exactly ───────────────────
// mtn_push_service.go reads:
//   "pulse_naira_per_point"       → Pulse Points threshold (default 250)
//   "draw_naira_per_entry"        → Draw Entry threshold (default 200)
//   "spin_max_per_day"            → Global daily spin cap (default 5)
//   "mtn_push_min_amount_naira"   → Min qualifying recharge (default 50)
//   "first_recharge_bonus_points" → First recharge bonus (default 0)
const KEYS = {
  PULSE_NAIRA:  "pulse_naira_per_point",
  DRAW_NAIRA:   "draw_naira_per_entry",
  SPIN_MAX:     "spin_max_per_day",
  MIN_RECHARGE: "mtn_push_min_amount_naira",
  FIRST_BONUS:  "first_recharge_bonus_points",
};

function useConfig() {
  const [configs, setConfigs] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await adminAPI.getConfig();
      const m: Record<string, string> = {};
      r.configs.forEach((c: ConfigEntry) => { m[c.key] = String(c.value); });
      setConfigs(m);
    } catch {
      setError("Failed to load configuration");
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

// ─── Config Card ──────────────────────────────────────────────────────────────
function ConfigCard({
  icon, title, description, configKey, configs, saving, saved, onSave,
  suffix = "", prefix = "", min = 0, step = 1, note,
}: {
  icon: string; title: string; description: string; configKey: string;
  configs: Record<string, string>; saving: string | null; saved: string | null;
  onSave: (k: string, v: string) => void;
  suffix?: string; prefix?: string; min?: number; step?: number; note?: string;
}) {
  const raw = configs[configKey] ?? "";
  const [val, setVal] = useState(raw || "0");
  useEffect(() => { if (raw !== "") setVal(raw); }, [raw]);

  const isDirty = val !== (raw || "0");
  const isSaving = saving === configKey;
  const isSaved = saved === configKey;

  return (
    <div style={{
      background: "#1c2038",
      border: `1px solid ${isSaved ? "rgba(34,197,94,0.4)" : "rgba(95,114,249,0.15)"}`,
      borderRadius: 12, padding: "20px", transition: "border-color 0.3s",
    }}>
      <div style={{ display: "flex", alignItems: "flex-start", gap: 12, marginBottom: 16 }}>
        <div style={{
          width: 40, height: 40, borderRadius: 10,
          background: "rgba(95,114,249,0.15)",
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
            background: "rgba(95,114,249,0.1)", color: "#818cf8",
            padding: "8px 12px", borderRadius: 8, fontSize: 13, fontWeight: 600,
            border: "1px solid rgba(95,114,249,0.2)",
          }}>{prefix}</span>
        )}
        <div style={{ flex: 1, position: "relative" }}>
          <input
            type="number" value={val} min={min} step={step}
            onChange={e => setVal(e.target.value)}
            style={{
              width: "100%", background: "#0f1629",
              border: `1px solid ${isDirty ? "rgba(95,114,249,0.5)" : "rgba(255,255,255,0.08)"}`,
              borderRadius: 8, padding: suffix ? "9px 48px 9px 12px" : "9px 12px",
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
            background: isSaved ? "rgba(34,197,94,0.2)" : isDirty ? "#5f72f9" : "rgba(95,114,249,0.15)",
            color: isSaved ? "#4ade80" : isDirty ? "#fff" : "#818cf8",
            transition: "all 0.2s", whiteSpace: "nowrap", opacity: isSaving ? 0.6 : 1,
          }}
        >
          {isSaving ? "Saving…" : isSaved ? "✓ Saved" : "Save"}
        </button>
      </div>
      {note && (
        <p style={{ color: "#64748b", fontSize: 11, marginTop: 8, lineHeight: 1.5 }}>
          {note}
        </p>
      )}
    </div>
  );
}

// ─── Live Preview ─────────────────────────────────────────────────────────────
function LivePreview({ configs }: { configs: Record<string, string> }) {
  const pulseNaira  = Number(configs[KEYS.PULSE_NAIRA]  || 250);
  const drawNaira   = Number(configs[KEYS.DRAW_NAIRA]   || 200);
  const spinMax     = Number(configs[KEYS.SPIN_MAX]     || 5);
  const minRecharge = Number(configs[KEYS.MIN_RECHARGE] || 50);
  const firstBonus  = Number(configs[KEYS.FIRST_BONUS]  || 0);

  const example = 450;
  const pulseEarned    = Math.floor(example / pulseNaira);
  const pulseRemainder = example % pulseNaira;
  const drawEarned     = Math.floor(example / drawNaira);
  const drawRemainder  = example % drawNaira;

  return (
    <div style={{
      background: "linear-gradient(135deg, rgba(95,114,249,0.08) 0%, rgba(139,92,246,0.08) 100%)",
      border: "1px solid rgba(95,114,249,0.2)", borderRadius: 12, padding: "20px",
    }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 16 }}>
        <span style={{ fontSize: 16 }}>🧮</span>
        <p style={{ color: "#818cf8", fontWeight: 700, fontSize: 13, margin: 0, letterSpacing: "0.05em" }}>
          LIVE PREVIEW — Example: ₦{example} Recharge
        </p>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 12 }}>
        {[
          { label: "Pulse Points", value: `+${pulseEarned} pt${pulseEarned !== 1 ? "s" : ""}`, sub: `₦${pulseRemainder} carries forward`, color: "#818cf8" },
          { label: "Draw Entries", value: `+${drawEarned} entr${drawEarned !== 1 ? "ies" : "y"}`, sub: `₦${drawRemainder} carries forward`, color: "#34d399" },
          { label: "Daily Spin Cap", value: `${spinMax} spins/day`, sub: "Tier-based eligibility", color: "#f59e0b" },
          { label: "Min Recharge", value: `₦${minRecharge}`, sub: firstBonus > 0 ? `1st recharge: +${firstBonus} pts` : "No 1st-recharge bonus", color: "#94a3b8" },
        ].map(s => (
          <div key={s.label} style={{
            background: "rgba(15,22,41,0.6)", borderRadius: 8, padding: "12px 14px",
            border: "1px solid rgba(255,255,255,0.05)",
          }}>
            <p style={{ color: "#64748b", fontSize: 11, margin: "0 0 4px", textTransform: "uppercase", letterSpacing: "0.05em" }}>{s.label}</p>
            <p style={{ color: s.color, fontSize: 17, fontWeight: 700, margin: "0 0 2px" }}>{s.value}</p>
            <p style={{ color: "#475569", fontSize: 11, margin: 0 }}>{s.sub}</p>
          </div>
        ))}
      </div>
    </div>
  );
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

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function PointsConfigPage() {
  const { configs, saving, saved, error, save } = useConfig();

  return (
    <AdminShell>
      <div style={{ maxWidth: 900, margin: "0 auto", padding: "32px 24px" }}>

        {/* Page Header */}
        <div style={{ marginBottom: 32 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 8 }}>
            <div style={{
              width: 44, height: 44, borderRadius: 12,
              background: "linear-gradient(135deg, #5f72f9, #8b5cf6)",
              display: "flex", alignItems: "center", justifyContent: "center", fontSize: 22,
            }}>💎</div>
            <div>
              <h1 style={{ color: "#e2e8f0", fontSize: 22, fontWeight: 700, margin: 0 }}>Points Engine</h1>
              <p style={{ color: "#64748b", fontSize: 13, margin: "2px 0 0" }}>
                Configure how subscribers earn Pulse Points, Draw Entries, and Spin Credits
              </p>
            </div>
          </div>
          {error && (
            <div style={{
              background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.3)",
              borderRadius: 8, padding: "10px 14px", marginTop: 12, color: "#f87171", fontSize: 13,
            }}>⚠ {error}</div>
          )}
        </div>

        {/* Live Preview */}
        <div style={{ marginBottom: 32 }}>
          <LivePreview configs={configs} />
        </div>

        {/* Section 1: Pulse Points */}
        <SectionHeader number="1" title="Pulse Points" subtitle="AI Studio currency — earned on every recharge via accumulator" color="#818cf8" />
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16, marginBottom: 32 }}>
          <ConfigCard
            icon="⚡" title="Pulse Point Threshold"
            description="Naira required to earn 1 Pulse Point. Remainder carries forward across recharges."
            configKey={KEYS.PULSE_NAIRA} configs={configs} saving={saving} saved={saved} onSave={save}
            prefix="₦" suffix="= 1 pt" min={1}
            note="e.g. ₦250 threshold: recharge ₦200 → 0 pts (₦200 carried), recharge ₦100 → 1 pt (₦50 carried)"
          />
          <ConfigCard
            icon="🎁" title="First Recharge Bonus"
            description="Flat Pulse Points awarded on a subscriber's very first qualifying recharge. Set to 0 to disable."
            configKey={KEYS.FIRST_BONUS} configs={configs} saving={saving} saved={saved} onSave={save}
            suffix="pts" min={0}
          />
        </div>

        {/* Section 2: Draw Entries */}
        <SectionHeader number="2" title="Draw Entries (Lottery Points)" subtitle="Daily lottery currency — earned per recharge, resets after each draw" color="#34d399" />
        <div style={{ marginBottom: 32 }}>
          <ConfigCard
            icon="🎟" title="Draw Entry Threshold"
            description="Naira required to earn 1 Draw Entry. Remainder carries forward within the day. Resets after each draw."
            configKey={KEYS.DRAW_NAIRA} configs={configs} saving={saving} saved={saved} onSave={save}
            prefix="₦" suffix="= 1 entry" min={1}
            note="e.g. ₦200 threshold: recharge ₦350 → 1 entry (₦150 carried), recharge ₦100 → 1 more entry (₦50 carried)"
          />
        </div>

        {/* Section 3: Spin Credits */}
        <SectionHeader number="3" title="Spin Credits" subtitle="Wheel spin tokens — tier-based on cumulative daily recharge (configure tiers in Spin Wheel page)" color="#f59e0b" />
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16, marginBottom: 32 }}>
          <ConfigCard
            icon="🎡" title="Global Daily Spin Cap"
            description="Maximum spins any user can earn in a single day, regardless of tier."
            configKey={KEYS.SPIN_MAX} configs={configs} saving={saving} saved={saved} onSave={save}
            suffix="spins/day" min={1}
            note="Spin tiers (Bronze/Silver/Gold/Platinum) and their per-tier caps are configured on the Spin Wheel page."
          />
          <ConfigCard
            icon="🔒" title="Minimum Qualifying Recharge"
            description="Minimum recharge amount to qualify for any rewards (Pulse Points, Draw Entries, Spin Credits)."
            configKey={KEYS.MIN_RECHARGE} configs={configs} saving={saving} saved={saved} onSave={save}
            prefix="₦" suffix="min" min={0}
          />
        </div>

        {/* Info Banner */}
        <div style={{
          background: "rgba(95,114,249,0.06)", border: "1px solid rgba(95,114,249,0.15)",
          borderRadius: 12, padding: "16px 20px", display: "flex", gap: 12, alignItems: "flex-start",
        }}>
          <span style={{ fontSize: 18, flexShrink: 0, marginTop: 1 }}>ℹ️</span>
          <div>
            <p style={{ color: "#818cf8", fontWeight: 600, fontSize: 13, margin: "0 0 6px" }}>
              How the Accumulator Works
            </p>
            <p style={{ color: "#64748b", fontSize: 12, margin: 0, lineHeight: 1.7 }}>
              Each user has a persistent counter per currency. When a recharge arrives, the amount is added to their counter.
              Every time the counter reaches the threshold, one unit is awarded and the threshold is subtracted from the counter.
              The remainder stays in the counter for the next recharge — no naira is ever wasted.
              Spin credits use a different model: the user&apos;s cumulative daily recharge total determines their tier,
              and the tier&apos;s daily cap is what they can earn that day.
            </p>
          </div>
        </div>

      </div>
    </AdminShell>
  );
}
