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

const TYPE_COLORS: Record<PrizeType, string> = {
  try_again:    "bg-gray-100 border-gray-300",
  pulse_points: "bg-indigo-50 border-indigo-300",
  airtime:      "bg-blue-50 border-blue-300",
  data_bundle:  "bg-cyan-50 border-cyan-300",
  momo_cash:    "bg-green-50 border-green-300",
};

type LocalPrize = Omit<Prize, "id"> & { id?: string; _dirty?: boolean };

function totalWeight(prizes: LocalPrize[]): number {
  return prizes.filter(p => p.is_active).reduce((s, p) => s + p.probability, 0);
}

export default function SpinConfigPage() {
  const [prizes, setPrizes]   = useState<LocalPrize[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving]   = useState(false);
  const [saved, setSaved]     = useState(false);
  const [error, setError]     = useState<string | null>(null);

  // Config keys for spin settings
  const [spinMax, setSpinMax]         = useState("3");
  const [liabilityCap, setLiabCap]    = useState("500000");
  const [savingCfg, setSavingCfg]     = useState(false);
  const [savedCfg, setSavedCfg]       = useState(false);

  const load = useCallback(async () => {
    const [r, cfg] = await Promise.all([adminAPI.getPrizePool(), adminAPI.getConfig()]);
    setPrizes(r.prizes);
    const m: Record<string,string> = {};
    cfg.configs.forEach(c => { m[c.key] = String(c.value); });
    setSpinMax(m["spin_max_per_user_per_day"] ?? "3");
    setLiabCap(String(Number(m["daily_prize_liability_cap_kobo"] ?? "50000000") / 100));
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const update = (i: number, field: keyof LocalPrize, val: unknown) =>
    setPrizes(prev => prev.map((p, j) => j === i ? { ...p, [field]: val, _dirty: true } : p));

  const addSlot = () => {
    if (prizes.length >= 16) { setError("Maximum 16 slots allowed"); return; }
    setPrizes(prev => [...prev, {
      name: "New Prize", prize_type: "try_again", base_value: 0,
      probability: 0, daily_inventory: -1, is_active: true, _dirty: true,
    }]);
  };

  const removeSlot = (i: number) => {
    if (prizes.length <= 8) { setError("Minimum 8 slots required"); return; }
    setPrizes(prev => prev.filter((_, j) => j !== i));
  };

  const validateAndSave = async () => {
    const active = prizes.filter(p => p.is_active);
    const total = active.reduce((s, p) => s + p.probability, 0);
    if (Math.abs(total - 100) > 0.01) {
      setError(`Probability weights must sum to 100% (currently ${total.toFixed(2)}%)`);
      return;
    }
    if (prizes.length < 8 || prizes.length > 16) {
      setError("Wheel must have 8–16 slots");
      return;
    }
    setError(null);
    setSaving(true);
    try {
      // Save each dirty prize via PUT /admin/prizes/:id
      const savePs = prizes.map((p) => {
        if (!p.id || !p._dirty) return Promise.resolve();
        return adminAPI.updatePrize(p.id, {
          name: p.name, prize_type: p.prize_type, base_value: p.base_value,
          probability: p.probability, daily_inventory: p.daily_inventory,
          is_active: p.is_active,
        }).catch((e) => { throw new Error(`Failed to save "${p.name}": ${e.message}`); });
      });
      await Promise.all(savePs);
      // Also sync probability weights via spin config endpoint
      setSaved(true); setTimeout(() => setSaved(false), 2000);
    } catch (e: unknown) { setError((e as Error).message); }
    finally { setSaving(false); }
  };

  const saveSpinConfig = async () => {
    setSavingCfg(true);
    await Promise.all([
      adminAPI.updateConfig("spin_max_per_user_per_day", spinMax),
      adminAPI.updateConfig("daily_prize_liability_cap_kobo", String(Math.round(Number(liabilityCap) * 100))),
    ]);
    setSavingCfg(false); setSavedCfg(true); setTimeout(() => setSavedCfg(false), 2000);
  };

  const tw = totalWeight(prizes);
  const twColor = Math.abs(tw - 100) < 0.01 ? "text-green-600" : "text-red-600";

  return (
    <div className="space-y-8 max-w-5xl">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Spin Wheel Configurator</h1>
          <p className="text-sm text-gray-500 mt-1">Configure prize slots, probabilities, and liability limits. (REQ-5.3.x)</p>
        </div>
        <div className="flex items-center gap-3">
          <span className={`text-sm font-semibold ${twColor}`}>
            Weight total: {tw.toFixed(2)}% {Math.abs(tw - 100) < 0.01 ? "✓" : "(must be 100%)"}
          </span>
          <button onClick={validateAndSave} disabled={saving}
            className={`px-4 py-2 rounded-lg text-sm font-medium ${
              saved ? "bg-green-600 text-white" : "bg-indigo-600 text-white hover:bg-indigo-700"
            } disabled:opacity-50`}>
            {saving ? "Saving…" : saved ? "✓ Saved" : "Save Prize Table"}
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-xl p-4 text-red-700 text-sm flex items-center gap-2">
          ⚠️ {error}
          <button onClick={() => setError(null)} className="ml-auto text-red-400 hover:text-red-600">✕</button>
        </div>
      )}

      {/* ── Spin Controls ── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5">
        <h2 className="text-base font-semibold text-gray-800 mb-4">Spin Limits & Liability (REQ-5.3.5/5.3.6)</h2>
        <div className="grid grid-cols-2 gap-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Max Spins Per User Per Day</label>
            <div className="flex gap-2">
              <input type="number" value={spinMax} onChange={e => setSpinMax(e.target.value)}
                className="flex-1 border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
              <span className="self-center text-xs text-gray-400">spins/day</span>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Daily Prize Liability Cap</label>
            <div className="flex gap-2">
              <span className="self-center text-sm text-gray-500">₦</span>
              <input type="number" value={liabilityCap} onChange={e => setLiabCap(e.target.value)}
                className="flex-1 border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
            </div>
          </div>
        </div>
        <button onClick={saveSpinConfig} disabled={savingCfg}
          className={`mt-4 px-4 py-2 rounded-lg text-sm font-medium ${
            savedCfg ? "bg-green-600 text-white" : "bg-indigo-600 text-white hover:bg-indigo-700"
          } disabled:opacity-50`}>
          {savingCfg ? "Saving…" : savedCfg ? "✓ Saved" : "Save Limits"}
        </button>
      </div>

      {/* ── Prize Slots ── */}
      {loading ? (
        <div className="flex justify-center py-20"><div className="animate-spin h-8 w-8 border-4 border-indigo-600 border-t-transparent rounded-full"/></div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold text-gray-800">
              Prize Slots ({prizes.length}/16)
            </h2>
            <button onClick={addSlot}
              className="px-3 py-1.5 border border-indigo-300 text-indigo-600 rounded-lg text-sm hover:bg-indigo-50">
              + Add Slot
            </button>
          </div>

          {prizes.map((p, i) => (
            <div key={i} className={`rounded-xl border-2 p-4 ${TYPE_COLORS[p.prize_type as PrizeType] ?? "bg-gray-50 border-gray-200"}`}>
              <div className="flex gap-4 items-start flex-wrap">
                {/* Active toggle */}
                <div className="flex items-center gap-1 pt-1">
                  <button onClick={() => update(i, "is_active", !p.is_active)}
                    className={`w-10 h-5 rounded-full transition-colors ${p.is_active ? "bg-indigo-600" : "bg-gray-300"}`}>
                    <span className={`block w-4 h-4 rounded-full bg-white shadow transition-transform mx-0.5 ${p.is_active ? "translate-x-5" : ""}`}/>
                  </button>
                  <span className="text-lg">{TYPE_ICONS[p.prize_type as PrizeType] ?? "🎁"}</span>
                </div>

                {/* Name */}
                <div className="flex-1 min-w-32">
                  <label className="text-xs text-gray-500 mb-1 block">Prize Name</label>
                  <input value={p.name} onChange={e => update(i, "name", e.target.value)}
                    className="w-full border border-white rounded-lg px-2 py-1.5 text-sm bg-white focus:ring-2 focus:ring-indigo-500 outline-none"/>
                </div>

                {/* Type */}
                <div className="w-36">
                  <label className="text-xs text-gray-500 mb-1 block">Type</label>
                  <select value={p.prize_type} onChange={e => update(i, "prize_type", e.target.value)}
                    className="w-full border border-white rounded-lg px-2 py-1.5 text-sm bg-white focus:ring-2 focus:ring-indigo-500 outline-none">
                    {PRIZE_TYPES.map(t => <option key={t} value={t}>{TYPE_ICONS[t]} {t.replace("_"," ")}</option>)}
                  </select>
                </div>

                {/* Value */}
                <div className="w-28">
                  <label className="text-xs text-gray-500 mb-1 block">
                    {p.prize_type === "momo_cash" || p.prize_type === "airtime" ? "Value (₦)" :
                     p.prize_type === "pulse_points" ? "Points" :
                     p.prize_type === "data_bundle" ? "MB" : "—"}
                  </label>
                  <input type="number" value={p.base_value}
                    onChange={e => update(i, "base_value", Number(e.target.value))}
                    disabled={p.prize_type === "try_again"}
                    className="w-full border border-white rounded-lg px-2 py-1.5 text-sm bg-white disabled:bg-gray-50 focus:ring-2 focus:ring-indigo-500 outline-none"/>
                </div>

                {/* Probability */}
                <div className="w-28">
                  <label className="text-xs text-gray-500 mb-1 block">Weight (%)</label>
                  <input type="number" step="0.1" min="0" max="100" value={p.probability}
                    onChange={e => update(i, "probability", parseFloat(e.target.value) || 0)}
                    className="w-full border border-white rounded-lg px-2 py-1.5 text-sm bg-white focus:ring-2 focus:ring-indigo-500 outline-none"/>
                </div>

                {/* Daily Inventory */}
                <div className="w-28">
                  <label className="text-xs text-gray-500 mb-1 block">Daily Cap</label>
                  <input type="number" value={p.daily_inventory === -1 ? "" : p.daily_inventory}
                    placeholder="∞"
                    onChange={e => update(i, "daily_inventory", e.target.value ? Number(e.target.value) : -1)}
                    className="w-full border border-white rounded-lg px-2 py-1.5 text-sm bg-white focus:ring-2 focus:ring-indigo-500 outline-none"/>
                </div>

                {/* Delete */}
                <button onClick={() => removeSlot(i)}
                  className="text-red-400 hover:text-red-600 text-xl pt-5">×</button>
              </div>

              {/* Probability bar */}
              <div className="mt-3">
                <div className="w-full bg-white/60 rounded-full h-1.5">
                  <div className="bg-indigo-500 h-1.5 rounded-full transition-all"
                    style={{ width: `${Math.min(p.probability, 100)}%` }}/>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
