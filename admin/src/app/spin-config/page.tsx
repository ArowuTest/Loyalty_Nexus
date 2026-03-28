"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState, useCallback } from "react";
import adminAPI, { Prize } from "@/lib/api";

const PRIZE_TYPES = ["try_again","pulse_points","airtime","data_bundle","momo_cash"] as const;
type PrizeType = typeof PRIZE_TYPES[number];

const TYPE_ICONS: Record<PrizeType, string> = {
  try_again:    "🔄",
  pulse_points: "💎",
  airtime:      "📱",
  data_bundle:  "📶",
  momo_cash:    "💵",
};

const TYPE_LABELS: Record<PrizeType, string> = {
  try_again:    "Try Again (no prize)",
  pulse_points: "Pulse Points",
  airtime:      "Airtime",
  data_bundle:  "Data Bundle",
  momo_cash:    "MoMo Cash",
};

const DEFAULT_COLORS: Record<PrizeType, string> = {
  try_again:    "#6b7280",
  pulse_points: "#5f72f9",
  airtime:      "#2196F3",
  data_bundle:  "#9C27B0",
  momo_cash:    "#10b981",
};

type LocalPrize = Prize & { _dirty?: boolean; _isNew?: boolean };

function totalWeight(prizes: LocalPrize[]): number {
  return prizes.filter(p => p.is_active).reduce((s, p) => s + (p.win_probability_weight || 0), 0);
}

function fmt(val: number, type: PrizeType): string {
  if (type === "try_again") return "—";
  if (type === "pulse_points") return `${val} pts`;
  if (type === "airtime" || type === "momo_cash") return `₦${(val / 100).toLocaleString()}`;
  if (type === "data_bundle") return val >= 100000 ? `${val / 100000}GB` : `${val / 100}MB`;
  return String(val);
}

export default function SpinConfigPage() {
  const [prizes, setPrizes]     = useState<LocalPrize[]>([]);
  const [loading, setLoading]   = useState(true);
  const [saving, setSaving]     = useState(false);
  const [saved, setSaved]       = useState(false);
  const [error, setError]       = useState<string | null>(null);

  // Spin limit config
  const [spinMax, setSpinMax]     = useState("3");
  const [liabCap, setLiabCap]     = useState("500000");
  const [savingCfg, setSavingCfg] = useState(false);
  const [savedCfg, setSavedCfg]   = useState(false);

  const load = useCallback(async () => {
    try {
      const [r, cfg] = await Promise.all([adminAPI.getPrizePool(), adminAPI.getConfig()]);
      setPrizes(r.prizes as LocalPrize[]);
      const m: Record<string, string> = {};
      cfg.configs.forEach(c => { m[c.key] = String(c.value); });
      setSpinMax(m["spin_max_per_user_per_day"] ?? "3");
      setLiabCap(String(Number(m["daily_prize_liability_cap_kobo"] ?? "50000000") / 100));
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const update = (i: number, field: keyof LocalPrize, val: unknown) =>
    setPrizes(prev => prev.map((p, j) => j === i ? { ...p, [field]: val, _dirty: true } : p));

  const addSlot = () => {
    if (prizes.length >= 16) { setError("Maximum 16 slots"); return; }
    const newPrize: LocalPrize = {
      id: "", name: "New Prize",
      prize_type: "try_again",
      base_value: 0, win_probability_weight: 0,
      is_active: true, is_no_win: true,
      color_scheme: DEFAULT_COLORS["try_again"],
      _dirty: true, _isNew: true,
    };
    setPrizes(p => [...p, newPrize]);
  };

  const removeSlot = (i: number) => {
    const p = prizes[i];
    if (p._isNew) { setPrizes(prev => prev.filter((_, j) => j !== i)); return; }
    if (!confirm(`Delete "${p.name}"?`)) return;
    adminAPI.deletePrize(p.id).then(load).catch(e => setError(String(e)));
  };

  const validateAndSave = async () => {
    setError(null);
    const active = prizes.filter(p => p.is_active);
    const total = active.reduce((s, p) => s + (p.win_probability_weight || 0), 0);
    if (total > 10000) {
      setError(`Total probability weight is ${total}/10000 (${(total/100).toFixed(2)}%). Must be ≤ 10000 (100.00%).`);
      return;
    }
    setSaving(true);
    try {
      for (const p of prizes) {
        if (!p._dirty) continue;
        const payload = {
          name: p.name,
          prize_type: p.prize_type,
          base_value: p.base_value,
          win_probability_weight: p.win_probability_weight,
          daily_inventory_cap: p.daily_inventory_cap ?? -1,
          is_active: p.is_active,
          is_no_win: p.is_no_win ?? false,
          no_win_message: p.no_win_message ?? "",
          color_scheme: p.color_scheme ?? DEFAULT_COLORS[p.prize_type as PrizeType] ?? "#888",
          sort_order: p.sort_order ?? 0,
          minimum_recharge: p.minimum_recharge ?? 0,
        };
        if (p._isNew) {
          await adminAPI.createPrize(payload as Omit<Prize, "id">);
        } else {
          await adminAPI.updatePrize(p.id, payload);
        }
      }
      await load();
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  };

  const saveSpinConfig = async () => {
    setSavingCfg(true);
    try {
      await Promise.all([
        adminAPI.updateConfig("spin_max_per_user_per_day", spinMax),
        adminAPI.updateConfig("daily_prize_liability_cap_kobo", String(Number(liabCap) * 100)),
      ]);
      setSavedCfg(true);
      setTimeout(() => setSavedCfg(false), 2500);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Config save failed");
    } finally {
      setSavingCfg(false);
    }
  };

  const total = totalWeight(prizes);
  const pct = (total / 100).toFixed(2);
  const barColor = total > 10000 ? "#ef4444" : total === 10000 ? "#10b981" : "#5f72f9";

  return (
    <AdminShell>
      <div className="max-w-4xl mx-auto space-y-6 pb-12">

        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>🎰 Spin Wheel Config</h1>
            <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
              Manage prize slots, probabilities, and payout limits. Weights must sum to ≤ 10,000 (= 100%).
            </p>
          </div>
          <div style={{ display: "flex", gap: 10 }}>
            <button onClick={load}
              style={{ padding: "8px 16px", borderRadius: 8, border: "1px solid rgba(95,114,249,0.3)", color: "#828cb4", fontSize: 13, background: "transparent", cursor: "pointer" }}>
              ↺ Refresh
            </button>
            <button onClick={validateAndSave} disabled={saving}
              style={{ padding: "8px 18px", borderRadius: 8, background: saved ? "#10b981" : "#5f72f9", color: "#fff", fontWeight: 600, fontSize: 13, border: "none", cursor: "pointer", opacity: saving ? 0.6 : 1 }}>
              {saving ? "Saving…" : saved ? "✓ Saved" : "Save Prize Table"}
            </button>
          </div>
        </div>

        {error && (
          <div style={{ background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.3)", borderRadius: 10, padding: "12px 16px", color: "#fca5a5", fontSize: 13, display: "flex", alignItems: "center", gap: 10 }}>
            ⚠️ {error}
            <button onClick={() => setError(null)} style={{ marginLeft: "auto", background: "none", border: "none", color: "#fca5a5", cursor: "pointer" }}>✕</button>
          </div>
        )}

        {/* Spin Limits */}
        <div className="card" style={{ padding: 20 }}>
          <h2 style={{ fontSize: 15, fontWeight: 600, color: "#e2e8ff", marginBottom: 16 }}>⚙️ Spin Limits</h2>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 20 }}>
            <div>
              <label style={{ display: "block", fontSize: 12, color: "#828cb4", marginBottom: 6 }}>Max Spins / User / Day</label>
              <input type="number" min={1} max={20} value={spinMax}
                onChange={e => setSpinMax(e.target.value)}
                style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 14 }} />
            </div>
            <div>
              <label style={{ display: "block", fontSize: 12, color: "#828cb4", marginBottom: 6 }}>Daily Prize Liability Cap (₦)</label>
              <input type="number" value={liabCap}
                onChange={e => setLiabCap(e.target.value)}
                style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 14 }} />
            </div>
          </div>
          <button onClick={saveSpinConfig} disabled={savingCfg}
            style={{ marginTop: 14, padding: "8px 18px", borderRadius: 8, background: savedCfg ? "#10b981" : "#5f72f9", color: "#fff", fontWeight: 600, fontSize: 13, border: "none", cursor: "pointer", opacity: savingCfg ? 0.6 : 1 }}>
            {savingCfg ? "Saving…" : savedCfg ? "✓ Saved" : "Save Limits"}
          </button>
        </div>

        {/* Probability Meter */}
        <div className="card" style={{ padding: 20 }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 10 }}>
            <span style={{ fontSize: 13, color: "#828cb4" }}>Total Probability Weight</span>
            <span style={{ fontSize: 15, fontWeight: 700, color: barColor }}>{total} / 10,000 ({pct}%)</span>
          </div>
          <div style={{ height: 8, borderRadius: 4, background: "rgba(255,255,255,0.05)", overflow: "hidden" }}>
            <div style={{ height: "100%", width: `${Math.min(100, total / 100)}%`, background: barColor, borderRadius: 4, transition: "width 0.3s" }} />
          </div>
          {total < 10000 && (
            <p style={{ marginTop: 8, fontSize: 11, color: "#828cb4" }}>⚠️ {10000 - total} weight remaining unallocated — add to "Try Again" or another slot.</p>
          )}
          {total > 10000 && (
            <p style={{ marginTop: 8, fontSize: 11, color: "#fca5a5" }}>❌ Over budget by {total - 10000} — reduce some weights before saving.</p>
          )}
        </div>

        {/* Prize Slots */}
        {loading ? (
          <div style={{ display: "flex", justifyContent: "center", padding: "60px 0" }}>
            <div style={{ width: 32, height: 32, border: "3px solid #5f72f9", borderTopColor: "transparent", borderRadius: "50%", animation: "spin 0.8s linear infinite" }} />
          </div>
        ) : (
          <>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
              <h2 style={{ fontSize: 15, fontWeight: 600, color: "#e2e8ff" }}>Prize Slots ({prizes.length}/16)</h2>
              <button onClick={addSlot}
                style={{ padding: "7px 16px", border: "1px solid rgba(95,114,249,0.4)", borderRadius: 8, color: "#5f72f9", background: "transparent", fontSize: 13, cursor: "pointer" }}>
                + Add Slot
              </button>
            </div>

            {prizes.map((p, i) => (
              <div key={p.id || i} className="card" style={{ padding: 16, opacity: p.is_active ? 1 : 0.55 }}>
                <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "flex-start" }}>

                  {/* Active toggle + icon */}
                  <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 4, paddingTop: 4 }}>
                    <button onClick={() => update(i, "is_active", !p.is_active)}
                      style={{ width: 42, height: 22, borderRadius: 11, background: p.is_active ? "#5f72f9" : "#374151", border: "none", cursor: "pointer", position: "relative", transition: "background 0.2s" }}>
                      <span style={{ position: "absolute", top: 2, left: p.is_active ? 22 : 2, width: 18, height: 18, background: "#fff", borderRadius: "50%", transition: "left 0.2s" }} />
                    </button>
                    <span style={{ fontSize: 20 }}>{TYPE_ICONS[p.prize_type as PrizeType] ?? "🎁"}</span>
                  </div>

                  {/* Name */}
                  <div style={{ flex: "1 1 140px" }}>
                    <label style={{ fontSize: 11, color: "#828cb4", display: "block", marginBottom: 4 }}>Prize Name</label>
                    <input value={p.name} onChange={e => update(i, "name", e.target.value)}
                      style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 7, padding: "7px 10px", color: "#e2e8ff", fontSize: 13 }} />
                  </div>

                  {/* Type */}
                  <div style={{ flex: "1 1 130px" }}>
                    <label style={{ fontSize: 11, color: "#828cb4", display: "block", marginBottom: 4 }}>Type</label>
                    <select value={p.prize_type}
                      onChange={e => {
                        const t = e.target.value as PrizeType;
                        update(i, "prize_type", t);
                        update(i, "is_no_win", t === "try_again");
                        update(i, "color_scheme", DEFAULT_COLORS[t] ?? "#888");
                      }}
                      style={{ width: "100%", background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 7, padding: "7px 10px", color: "#e2e8ff", fontSize: 13 }}>
                      {PRIZE_TYPES.map(t => (
                        <option key={t} value={t}>{TYPE_LABELS[t]}</option>
                      ))}
                    </select>
                  </div>

                  {/* Value */}
                  {p.prize_type !== "try_again" && (
                    <div style={{ flex: "1 1 100px" }}>
                      <label style={{ fontSize: 11, color: "#828cb4", display: "block", marginBottom: 4 }}>
                        Value {p.prize_type === "pulse_points" ? "(pts)" : p.prize_type === "data_bundle" ? "(kobo-MB)" : "(kobo)"}
                      </label>
                      <input type="number" min={0} value={p.base_value}
                        onChange={e => update(i, "base_value", Number(e.target.value))}
                        style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 7, padding: "7px 10px", color: "#e2e8ff", fontSize: 13 }} />
                      <p style={{ fontSize: 10, color: "#5f72f9", marginTop: 3 }}>{fmt(p.base_value, p.prize_type as PrizeType)}</p>
                    </div>
                  )}

                  {/* Probability weight */}
                  <div style={{ flex: "1 1 100px" }}>
                    <label style={{ fontSize: 11, color: "#828cb4", display: "block", marginBottom: 4 }}>
                      Weight (of 10,000)
                    </label>
                    <input type="number" min={0} max={10000} value={p.win_probability_weight}
                      onChange={e => update(i, "win_probability_weight", Number(e.target.value))}
                      style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 7, padding: "7px 10px", color: "#e2e8ff", fontSize: 13 }} />
                    <p style={{ fontSize: 10, color: "#5f72f9", marginTop: 3 }}>
                      {((p.win_probability_weight / 100)).toFixed(2)}% chance
                    </p>
                  </div>

                  {/* Daily cap */}
                  <div style={{ flex: "1 1 80px" }}>
                    <label style={{ fontSize: 11, color: "#828cb4", display: "block", marginBottom: 4 }}>Daily Cap (-1 = ∞)</label>
                    <input type="number" min={-1} value={p.daily_inventory_cap ?? -1}
                      onChange={e => update(i, "daily_inventory_cap", Number(e.target.value))}
                      style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 7, padding: "7px 10px", color: "#e2e8ff", fontSize: 13 }} />
                  </div>

                  {/* Color */}
                  <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 4, paddingTop: 2 }}>
                    <label style={{ fontSize: 11, color: "#828cb4" }}>Color</label>
                    <input type="color" value={p.color_scheme || DEFAULT_COLORS[p.prize_type as PrizeType] || "#888"}
                      onChange={e => update(i, "color_scheme", e.target.value)}
                      style={{ width: 36, height: 32, borderRadius: 6, border: "none", background: "none", cursor: "pointer", padding: 0 }} />
                  </div>

                  {/* Delete */}
                  <button onClick={() => removeSlot(i)}
                    style={{ alignSelf: "flex-start", marginTop: 20, background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.2)", borderRadius: 7, color: "#fca5a5", padding: "6px 12px", fontSize: 12, cursor: "pointer" }}>
                    🗑
                  </button>
                </div>

                {/* No-win message */}
                {p.is_no_win && (
                  <div style={{ marginTop: 10, paddingTop: 10, borderTop: "1px solid rgba(95,114,249,0.1)" }}>
                    <label style={{ fontSize: 11, color: "#828cb4", display: "block", marginBottom: 4 }}>No-win message shown to user</label>
                    <input value={p.no_win_message ?? ""}
                      onChange={e => update(i, "no_win_message", e.target.value)}
                      style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.1)", borderRadius: 7, padding: "7px 10px", color: "#e2e8ff", fontSize: 12 }} />
                  </div>
                )}
              </div>
            ))}
          </>
        )}
      </div>

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
      `}</style>
    </AdminShell>
  );
}
