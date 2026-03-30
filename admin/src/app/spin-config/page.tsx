"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState, useCallback } from "react";
import adminAPI, { Prize, SpinTier } from "@/lib/api";

const PRIZE_TYPES = ["try_again","pulse_points","bonus_points","airtime","data_bundle","momo_cash","studio_credits"] as const;
type PrizeType = typeof PRIZE_TYPES[number];

const TYPE_ICONS: Record<PrizeType, string> = {
  try_again:      "🔄",
  pulse_points:   "💎",
  bonus_points:   "⭐",
  airtime:        "📱",
  data_bundle:    "📶",
  momo_cash:      "💰",
  studio_credits: "🎨",
};

const TYPE_LABELS: Record<PrizeType, string> = {
  try_again:      "Try Again (no prize)",
  pulse_points:   "Pulse Points",
  bonus_points:   "Bonus Points",
  airtime:        "Airtime",
  data_bundle:    "Data Bundle",
  momo_cash:      "Cash Prize",
  studio_credits: "Studio Credits",
};

const DEFAULT_COLORS: Record<PrizeType, string> = {
  try_again:      "#6b7280",
  pulse_points:   "#5f72f9",
  bonus_points:   "#f59e0b",
  airtime:        "#2196F3",
  data_bundle:    "#9C27B0",
  momo_cash:      "#10b981",
  studio_credits: "#8B5CF6",
};

type LocalPrize = Prize & { _dirty?: boolean; _isNew?: boolean };

function totalWeight(prizes: LocalPrize[]): number {
  // weights are NUMERIC(5,2) — each weight IS the percentage (e.g. 25.00 = 25%)
  return prizes.filter(p => p.is_active).reduce((s, p) => s + (p.win_probability_weight || 0), 0);
}

function fmt(val: number, type: PrizeType): string {
  if (type === "try_again") return "—";
  if (type === "pulse_points") return `${val} pts`;
  if (type === "airtime" || type === "momo_cash") return `₦${val.toLocaleString()}`;
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

  // Spin tiers
  const [tiers, setTiers]         = useState<SpinTier[]>([]);
  const [tierModal, setTierModal] = useState<{ type: "create" } | { type: "edit"; tier: SpinTier } | null>(null);
  const [tierForm, setTierForm]   = useState<Omit<SpinTier, "id">>({ name: "", min_daily_amount: 0, max_daily_amount: 0, spins_per_day: 1, badge_color: "#5f72f9", sort_order: 0 });
  const [tierSaving, setTierSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const [r, cfg, tr] = await Promise.all([adminAPI.getPrizePool(), adminAPI.getConfig(), adminAPI.getSpinTiers()]);
      setPrizes(r.prizes as LocalPrize[]);
      setTiers(tr.tiers ?? []);
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
    if (total > 100.00 + 0.001) { // allow tiny floating-point tolerance
      setError(`Total probability is ${total.toFixed(2)}% — must be ≤ 100.00%. Reduce some weights before saving.`);
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

  const handleSaveTier = async () => {
    if (!tierForm.name.trim()) { setError("Tier name is required"); return; }
    if (tierForm.spins_per_day < 1) { setError("Spins per day must be ≥ 1"); return; }
    setTierSaving(true);
    try {
      if (tierModal?.type === "create") {
        await adminAPI.createSpinTier(tierForm);
      } else if (tierModal?.type === "edit") {
        await adminAPI.updateSpinTier((tierModal as { type: "edit"; tier: SpinTier }).tier.id, tierForm);
      }
      setTierModal(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Save tier failed");
    } finally {
      setTierSaving(false);
    }
  };

  const handleDeleteTier = async (t: SpinTier) => {
    if (!confirm(`Delete tier "${t.name}"?`)) return;
    try {
      await adminAPI.deleteSpinTier(t.id);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Delete tier failed");
    }
  };

  const openCreateTier = () => {
    setTierForm({ name: "", min_daily_amount: 0, max_daily_amount: 0, spins_per_day: 1, badge_color: "#5f72f9", sort_order: 0 });
    setTierModal({ type: "create" });
  };

  const openEditTier = (t: SpinTier) => {
    setTierForm({ name: t.name, min_daily_amount: t.min_daily_amount, max_daily_amount: t.max_daily_amount, spins_per_day: t.spins_per_day, badge_color: t.badge_color ?? "#5f72f9", sort_order: t.sort_order ?? 0 });
    setTierModal({ type: "edit", tier: t });
  };

  const total = totalWeight(prizes);
  const pct = total.toFixed(2); // weight IS the percentage — no conversion needed
  const barColor = total > 100 ? "#ef4444" : Math.abs(total - 100) < 0.01 ? "#10b981" : "#5f72f9";

  return (
    <AdminShell>
      <div className="max-w-4xl mx-auto space-y-6 pb-12">

        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>🎰 Spin Wheel Config</h1>
            <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
              Manage prize slots, probabilities, and payout limits. Probabilities must sum to exactly 100%.
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
            <span style={{ fontSize: 15, fontWeight: 700, color: barColor }}>{pct}% / 100%</span>
          </div>
          <div style={{ height: 8, borderRadius: 4, background: "rgba(255,255,255,0.05)", overflow: "hidden" }}>
            <div style={{ height: "100%", width: `${Math.min(100, total)}%`, background: barColor, borderRadius: 4, transition: "width 0.3s" }} />
          </div>
          {total < 99.99 && (
            <p style={{ marginTop: 8, fontSize: 11, color: "#828cb4" }}>⚠️ {(100 - total).toFixed(2)}% unallocated — add to "Better Luck Next Time" or another slot.</p>
          )}
          {total > 100.001 && (
            <p style={{ marginTop: 8, fontSize: 11, color: "#fca5a5" }}>❌ Over 100% by {(total - 100).toFixed(2)}% — reduce some weights before saving.</p>
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
                      Probability (%)
                    </label>
                    <input type="number" min={0} max={100} step={0.01} value={p.win_probability_weight}
                      onChange={e => update(i, "win_probability_weight", parseFloat(e.target.value) || 0)}
                      style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 7, padding: "7px 10px", color: "#e2e8ff", fontSize: 13 }} />
                    <p style={{ fontSize: 10, color: "#5f72f9", marginTop: 3 }}>
                      {(p.win_probability_weight || 0).toFixed(2)}% chance
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

        {/* Spin Tiers Section */}
        <div className="card" style={{ padding: 20 }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 16 }}>
            <div>
              <h2 style={{ fontSize: 15, fontWeight: 600, color: "#e2e8ff" }}>🏆 Spin Tiers</h2>
              <p style={{ fontSize: 12, color: "#828cb4", marginTop: 3 }}>Configure daily recharge thresholds that determine how many spins a user earns.</p>
            </div>
            <button onClick={openCreateTier}
              style={{ padding: "7px 16px", border: "1px solid rgba(95,114,249,0.4)", borderRadius: 8, color: "#5f72f9", background: "transparent", fontSize: 13, cursor: "pointer" }}>
              + Add Tier
            </button>
          </div>

          {tiers.length === 0 ? (
            <p style={{ color: "#828cb4", fontSize: 13, textAlign: "center", padding: "20px 0" }}>No tiers configured</p>
          ) : (
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.15)" }}>
                  {["Tier", "Min Daily Recharge", "Max Daily Recharge", "Spins/Day", "Actions"].map(h => (
                    <th key={h} style={{ padding: "8px 12px", textAlign: "left", color: "#828cb4", fontWeight: 600 }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {tiers.sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0)).map(t => (
                  <tr key={t.id} style={{ borderBottom: "1px solid rgba(95,114,249,0.07)" }}>
                    <td style={{ padding: "8px 12px" }}>
                      <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                        <span style={{ width: 10, height: 10, borderRadius: "50%", background: t.badge_color ?? "#5f72f9", display: "inline-block" }} />
                        <span style={{ color: t.badge_color ?? "#e2e8ff", fontWeight: 600 }}>{t.name}</span>
                      </span>
                    </td>
                    <td style={{ padding: "8px 12px", color: "#e2e8ff" }}>₦{(t.min_daily_amount / 100).toLocaleString()}</td>
                    <td style={{ padding: "8px 12px", color: "#e2e8ff" }}>{t.max_daily_amount === 0 ? "Unlimited" : `₦${(t.max_daily_amount / 100).toLocaleString()}`}</td>
                    <td style={{ padding: "8px 12px", color: "#10b981", fontWeight: 700 }}>{t.spins_per_day}</td>
                    <td style={{ padding: "8px 12px" }}>
                      <div style={{ display: "flex", gap: 6 }}>
                        <button onClick={() => openEditTier(t)}
                          style={{ padding: "3px 9px", borderRadius: 6, border: "1px solid rgba(95,114,249,0.3)", color: "#5f72f9", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                          Edit
                        </button>
                        <button onClick={() => handleDeleteTier(t)}
                          style={{ padding: "3px 9px", borderRadius: 6, border: "1px solid rgba(239,68,68,0.3)", color: "#fca5a5", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Tier Create/Edit Modal */}
      {tierModal && (
        <div style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 50 }}>
          <div className="card" style={{ width: "min(480px, 95vw)", padding: 28 }}>
            <h2 style={{ fontSize: 16, fontWeight: 700, color: "#e2e8ff", marginBottom: 20 }}>
              {tierModal.type === "create" ? "➕ Add Spin Tier" : "✏️ Edit Spin Tier"}
            </h2>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, marginBottom: 16 }}>
              <div style={{ gridColumn: "1 / -1" }}>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Tier Name *</label>
                <input value={tierForm.name} onChange={e => setTierForm(f => ({ ...f, name: e.target.value }))}
                  placeholder="e.g. Bronze"
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Min Daily Recharge (₦) *</label>
                <input type="number" min="0" value={tierForm.min_daily_amount / 100}
                  onChange={e => setTierForm(f => ({ ...f, min_daily_amount: Math.round(parseFloat(e.target.value) * 100) || 0 }))}
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Max Daily Recharge (₦) — 0 = unlimited</label>
                <input type="number" min="0" value={tierForm.max_daily_amount / 100}
                  onChange={e => setTierForm(f => ({ ...f, max_daily_amount: Math.round(parseFloat(e.target.value) * 100) || 0 }))}
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Spins Per Day *</label>
                <input type="number" min="1" max="20" value={tierForm.spins_per_day}
                  onChange={e => setTierForm(f => ({ ...f, spins_per_day: parseInt(e.target.value) || 1 }))}
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Badge Color</label>
                <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                  <input type="color" value={tierForm.badge_color ?? "#5f72f9"}
                    onChange={e => setTierForm(f => ({ ...f, badge_color: e.target.value }))}
                    style={{ width: 40, height: 36, borderRadius: 6, border: "none", cursor: "pointer", padding: 0 }} />
                  <input value={tierForm.badge_color ?? "#5f72f9"}
                    onChange={e => setTierForm(f => ({ ...f, badge_color: e.target.value }))}
                    style={{ flex: 1, background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13 }} />
                </div>
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Sort Order</label>
                <input type="number" min="0" value={tierForm.sort_order ?? 0}
                  onChange={e => setTierForm(f => ({ ...f, sort_order: parseInt(e.target.value) || 0 }))}
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
            </div>
            <div style={{ display: "flex", gap: 10 }}>
              <button onClick={() => setTierModal(null)}
                style={{ flex: 1, padding: "10px", borderRadius: 8, background: "transparent", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
                Cancel
              </button>
              <button onClick={handleSaveTier} disabled={tierSaving}
                style={{ flex: 1, padding: "10px", borderRadius: 8, background: "#5f72f9", border: "none", color: "#fff", fontWeight: 600, cursor: "pointer", opacity: tierSaving ? 0.6 : 1 }}>
                {tierSaving ? "Saving…" : tierModal.type === "create" ? "Create Tier" : "Save Changes"}
              </button>
            </div>
          </div>
        </div>
      )}

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
      `}</style>
    </AdminShell>
  );
}
