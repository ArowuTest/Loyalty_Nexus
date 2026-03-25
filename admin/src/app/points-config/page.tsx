"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState, useCallback } from "react";
import adminAPI, { ConfigEntry } from "@/lib/api";

interface Tier { min_recharge: number; points_per_naira_denom: number; label: string; }
interface StreakMilestone { days: number; bonus_points: number; }

// Config keys we manage
const KEYS = {
  BASE_RATE:         "points_naira_per_point",
  SPIN_THRESHOLD:    "spin_credit_threshold_kobo",
  MULTIPLIER:        "points_multiplier",
  MULTIPLIER_START:  "points_multiplier_start",
  MULTIPLIER_END:    "points_multiplier_end",
  MIN_RECHARGE:      "min_qualifying_recharge_kobo",
  STREAK_WINDOW:     "streak_expiry_hours",
  STREAK_FREEZE:     "streak_freeze_days_per_month",
  FIRST_RECHARGE:    "first_recharge_bonus_points",
  REFERRAL_BONUS:    "referral_bonus_points",
  TIERS:             "recharge_tiers_json",
  STREAK_MILESTONES: "streak_milestones_json",
  EXPIRY_DAYS:       "points_expiry_days",
  EXPIRY_WARN_DAYS:  "points_expiry_warn_days",
};

function useConfig() {
  const [configs, setConfigs] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);

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

  return { configs, saving, saved, save, reload: load };
}

function NumberField({ label, desc, configKey, configs, saving, saved, onSave, divisor = 1, suffix = "" }:
  { label: string; desc: string; configKey: string; configs: Record<string,string>;
    saving: string|null; saved: string|null; onSave: (k:string,v:string)=>void;
    divisor?: number; suffix?: string }) {
  const raw = configs[configKey] ?? "0";
  const [val, setVal] = useState("");
  useEffect(() => { setVal(String(Number(raw) / divisor)); }, [raw, divisor]);

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4">
      <label className="block text-sm font-semibold text-gray-800 mb-1">{label}</label>
      <p className="text-xs text-gray-500 mb-3">{desc}</p>
      <div className="flex gap-2">
        <div className="relative flex-1">
          <input type="number" value={val} onChange={e => setVal(e.target.value)}
            className="w-full border rounded-lg px-3 py-2 text-sm pr-10 focus:ring-2 focus:ring-indigo-500 outline-none"/>
          {suffix && <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-400">{suffix}</span>}
        </div>
        <button disabled={saving === configKey}
          onClick={() => onSave(configKey, String(Math.round(Number(val) * divisor)))}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
            saved === configKey ? "bg-green-600 text-white" : "bg-indigo-600 text-white hover:bg-indigo-700"
          } disabled:opacity-50`}>
          {saving === configKey ? "…" : saved === configKey ? "✓" : "Save"}
        </button>
      </div>
    </div>
  );
}

export default function PointsConfigPage() {
  const { configs, saving, saved, save } = useConfig();

  // Tiers JSON
  const [tiers, setTiersState] = useState<Tier[]>([]);
  const [milestones, setMilestonesState] = useState<StreakMilestone[]>([]);

  useEffect(() => {
    try { setTiersState(JSON.parse(configs[KEYS.TIERS] || "[]")); } catch { setTiersState([]); }
    try { setMilestonesState(JSON.parse(configs[KEYS.STREAK_MILESTONES] || "[]")); } catch { setMilestonesState([]); }
  }, [configs]);

  const saveTiers = () => save(KEYS.TIERS, JSON.stringify(tiers));
  const saveMilestones = () => save(KEYS.STREAK_MILESTONES, JSON.stringify(milestones));

  const addTier = () => setTiersState(prev => [...prev,
    { min_recharge: 0, points_per_naira_denom: 250, label: "" }]);
  const removeTier = (i: number) => setTiersState(prev => prev.filter((_, j) => j !== i));
  const updateTier = (i: number, field: keyof Tier, value: string|number) =>
    setTiersState(prev => prev.map((t, j) => j === i ? { ...t, [field]: value } : t));

  const addMilestone = () => setMilestonesState(prev => [...prev, { days: 7, bonus_points: 10 }]);
  const removeMilestone = (i: number) => setMilestonesState(prev => prev.filter((_, j) => j !== i));
  const updateMilestone = (i: number, field: keyof StreakMilestone, val: number) =>
    setMilestonesState(prev => prev.map((m, j) => j === i ? { ...m, [field]: val } : m));

  return (
    <div className="space-y-8 max-w-4xl">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Points Engine Configuration</h1>
        <p className="text-sm text-gray-500 mt-1">
          All parameters update in real-time — no deployment needed. (REQ-5.2.x Zero Hardcoding)
        </p>
      </div>

      {/* ── Base Earning Rate ── */}
      <section>
        <h2 className="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
          <span className="w-7 h-7 bg-indigo-100 text-indigo-700 rounded-full flex items-center justify-center text-sm font-bold">1</span>
          Base Earning Rate
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <NumberField label="₦ per 1 Pulse Point" desc="Default earning rate. Lower = more generous. (e.g. 250 means ₦250 = 1 pt)"
            configKey={KEYS.BASE_RATE} configs={configs} saving={saving} saved={saved} onSave={save} suffix="₦/pt"/>
          <NumberField label="Spin Credit Threshold" desc="Cumulative recharge amount that awards 1 Spin Credit."
            configKey={KEYS.SPIN_THRESHOLD} configs={configs} saving={saving} saved={saved} onSave={save}
            divisor={100} suffix="₦"/>
          <NumberField label="Minimum Qualifying Recharge" desc="Recharges below this amount earn no points."
            configKey={KEYS.MIN_RECHARGE} configs={configs} saving={saving} saved={saved} onSave={save}
            divisor={100} suffix="₦"/>
        </div>
      </section>

      {/* ── Tiered Earning Rules ── */}
      <section>
        <h2 className="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
          <span className="w-7 h-7 bg-indigo-100 text-indigo-700 rounded-full flex items-center justify-center text-sm font-bold">2</span>
          Tiered Earning Rules (REQ-5.2.3)
        </h2>
        <div className="bg-white rounded-xl border border-gray-200 p-4">
          <p className="text-xs text-gray-500 mb-4">Higher recharges earn at better rates. Label each tier clearly.</p>
          <div className="space-y-2">
            {tiers.map((tier, i) => (
              <div key={i} className="flex gap-2 items-center">
                <input placeholder="Label (e.g. Gold)" value={tier.label}
                  onChange={e => updateTier(i, "label", e.target.value)}
                  className="w-28 border rounded-lg px-2 py-1.5 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                <span className="text-xs text-gray-400">Min ₦</span>
                <input type="number" placeholder="1000" value={tier.min_recharge}
                  onChange={e => updateTier(i, "min_recharge", Number(e.target.value))}
                  className="w-24 border rounded-lg px-2 py-1.5 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                <span className="text-xs text-gray-400">₦ per pt</span>
                <input type="number" placeholder="200" value={tier.points_per_naira_denom}
                  onChange={e => updateTier(i, "points_per_naira_denom", Number(e.target.value))}
                  className="w-20 border rounded-lg px-2 py-1.5 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                <button onClick={() => removeTier(i)} className="text-red-400 hover:text-red-600 text-lg leading-none">×</button>
              </div>
            ))}
          </div>
          <div className="flex gap-3 mt-4">
            <button onClick={addTier}
              className="px-3 py-1.5 border border-indigo-300 text-indigo-600 rounded-lg text-sm hover:bg-indigo-50">
              + Add Tier
            </button>
            <button onClick={saveTiers}
              className={`px-4 py-1.5 rounded-lg text-sm font-medium ${
                saved === KEYS.TIERS ? "bg-green-600 text-white" : "bg-indigo-600 text-white hover:bg-indigo-700"
              }`}>
              {saving === KEYS.TIERS ? "Saving…" : saved === KEYS.TIERS ? "✓ Saved" : "Save Tiers"}
            </button>
          </div>
        </div>
      </section>

      {/* ── Global Multiplier ── */}
      <section>
        <h2 className="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
          <span className="w-7 h-7 bg-yellow-100 text-yellow-700 rounded-full flex items-center justify-center text-sm font-bold">3</span>
          Global & Scheduled Multiplier (REQ-5.2.5/5.2.6)
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <NumberField label="Points Multiplier" desc='1.0 = normal. 2.0 = "Double Points Weekend". Updates immediately.'
            configKey={KEYS.MULTIPLIER} configs={configs} saving={saving} saved={saved} onSave={save} suffix="×"/>
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <label className="block text-sm font-semibold text-gray-800 mb-1">Scheduled Start</label>
            <p className="text-xs text-gray-500 mb-3">Auto-activate multiplier at this time</p>
            <input type="datetime-local" value={configs[KEYS.MULTIPLIER_START] ?? ""}
              onChange={e => save(KEYS.MULTIPLIER_START, e.target.value)}
              className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <label className="block text-sm font-semibold text-gray-800 mb-1">Scheduled End</label>
            <p className="text-xs text-gray-500 mb-3">Auto-deactivate at this time</p>
            <input type="datetime-local" value={configs[KEYS.MULTIPLIER_END] ?? ""}
              onChange={e => save(KEYS.MULTIPLIER_END, e.target.value)}
              className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
          </div>
        </div>
      </section>

      {/* ── Bonus Events ── */}
      <section>
        <h2 className="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
          <span className="w-7 h-7 bg-green-100 text-green-700 rounded-full flex items-center justify-center text-sm font-bold">4</span>
          Bonus Point Events (REQ-5.2.8–5.2.10)
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <NumberField label="First Recharge Bonus" desc="Flat points awarded on a subscriber's very first recharge."
            configKey={KEYS.FIRST_RECHARGE} configs={configs} saving={saving} saved={saved} onSave={save} suffix="pts"/>
          <NumberField label="Referral Bonus" desc="Points awarded to both referrer and new user on first recharge."
            configKey={KEYS.REFERRAL_BONUS} configs={configs} saving={saving} saved={saved} onSave={save} suffix="pts"/>
        </div>
      </section>

      {/* ── Streak Milestones ── */}
      <section>
        <h2 className="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
          <span className="w-7 h-7 bg-orange-100 text-orange-700 rounded-full flex items-center justify-center text-sm font-bold">5</span>
          Streak Configuration (REQ-5.2.9/5.2.12/5.2.13)
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <NumberField label="Streak Expiry Window" desc="Hours a user has to recharge before streak resets."
            configKey={KEYS.STREAK_WINDOW} configs={configs} saving={saving} saved={saved} onSave={save} suffix="hrs"/>
          <NumberField label="Streak Freeze Days" desc="Grace days per month where a missed recharge won't reset streak."
            configKey={KEYS.STREAK_FREEZE} configs={configs} saving={saving} saved={saved} onSave={save} suffix="days/mo"/>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-4">
          <p className="text-sm font-semibold text-gray-800 mb-1">Streak Milestone Bonuses</p>
          <p className="text-xs text-gray-500 mb-4">One-time bonus awarded when user reaches these streak lengths.</p>
          <div className="space-y-2">
            {milestones.map((m, i) => (
              <div key={i} className="flex gap-2 items-center">
                <span className="text-xs text-gray-400">At</span>
                <input type="number" placeholder="7" value={m.days}
                  onChange={e => updateMilestone(i, "days", Number(e.target.value))}
                  className="w-20 border rounded-lg px-2 py-1.5 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                <span className="text-xs text-gray-400">days → +</span>
                <input type="number" placeholder="10" value={m.bonus_points}
                  onChange={e => updateMilestone(i, "bonus_points", Number(e.target.value))}
                  className="w-20 border rounded-lg px-2 py-1.5 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                <span className="text-xs text-gray-400">pts</span>
                <button onClick={() => removeMilestone(i)} className="text-red-400 hover:text-red-600 text-lg">×</button>
              </div>
            ))}
          </div>
          <div className="flex gap-3 mt-4">
            <button onClick={addMilestone} className="px-3 py-1.5 border border-indigo-300 text-indigo-600 rounded-lg text-sm hover:bg-indigo-50">
              + Add Milestone
            </button>
            <button onClick={saveMilestones}
              className={`px-4 py-1.5 rounded-lg text-sm font-medium ${
                saved === KEYS.STREAK_MILESTONES ? "bg-green-600 text-white" : "bg-indigo-600 text-white hover:bg-indigo-700"
              }`}>
              {saving === KEYS.STREAK_MILESTONES ? "Saving…" : saved === KEYS.STREAK_MILESTONES ? "✓ Saved" : "Save Milestones"}
            </button>
          </div>
        </div>
      </section>

      {/* ── Points Expiry ── */}
      <section>
        <h2 className="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
          <span className="w-7 h-7 bg-red-100 text-red-700 rounded-full flex items-center justify-center text-sm font-bold">6</span>
          Points Expiry Policy (REQ-5.2.14/5.2.15)
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <NumberField label="Points Expiry Duration" desc='Days of inactivity before points expire. Set to 0 to disable expiry.'
            configKey={KEYS.EXPIRY_DAYS} configs={configs} saving={saving} saved={saved} onSave={save} suffix="days"/>
          <NumberField label="Expiry Warning Lead Time" desc="Days before expiry to send SMS warning to user."
            configKey={KEYS.EXPIRY_WARN_DAYS} configs={configs} saving={saving} saved={saved} onSave={save} suffix="days"/>
        </div>
      </section>
    </div>
  );
}
