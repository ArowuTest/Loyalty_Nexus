"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState, useCallback } from "react";
import adminAPI, { FulfillmentConfig } from "@/lib/api";

// Prize types that support AUTO mode (digital provisioning via VTPass)
const AUTO_PROVISIONABLE = new Set(["airtime", "data_bundle"]);

const PRIZE_ICONS: Record<string, string> = {
  try_again:    "🔄",
  pulse_points: "💎",
  airtime:      "📱",
  data_bundle:  "📶",
  momo_cash:    "💰",
  physical:     "📦",
  goods:        "🛍️",
};

const PRIZE_LABELS: Record<string, string> = {
  try_again:    "Try Again",
  pulse_points: "Pulse Points",
  airtime:      "Airtime",
  data_bundle:  "Data Bundle",
  momo_cash:    "MoMo Cash",
  physical:     "Physical Prize",
  goods:        "Goods",
};

type EditState = {
  fulfillment_mode: "MANUAL" | "AUTO";
  max_retry_attempts: number;
  retry_delay_seconds: number;
  fallback_to_manual: boolean;
};

export default function FulfillmentConfigPage() {
  const [configs, setConfigs]   = useState<FulfillmentConfig[]>([]);
  const [loading, setLoading]   = useState(true);
  const [saving, setSaving]     = useState<string | null>(null); // prize_type being saved
  const [saved, setSaved]       = useState<string | null>(null); // prize_type just saved
  const [error, setError]       = useState<string | null>(null);
  const [edits, setEdits]       = useState<Record<string, EditState>>({});

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const res = await adminAPI.getFulfillmentConfig();
      setConfigs(res.configs ?? []);
      // Seed local edit state from server values
      const map: Record<string, EditState> = {};
      for (const c of res.configs ?? []) {
        map[c.prize_type] = {
          fulfillment_mode:    c.fulfillment_mode,
          max_retry_attempts:  c.max_retry_attempts,
          retry_delay_seconds: c.retry_delay_seconds,
          fallback_to_manual:  c.fallback_to_manual,
        };
      }
      setEdits(map);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Load failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  function updateEdit(prizeType: string, patch: Partial<EditState>) {
    setEdits(prev => ({
      ...prev,
      [prizeType]: { ...prev[prizeType], ...patch },
    }));
  }

  async function save(prizeType: string) {
    const edit = edits[prizeType];
    if (!edit) return;
    setSaving(prizeType);
    setError(null);
    try {
      await adminAPI.updateFulfillmentConfig(prizeType, edit);
      setSaved(prizeType);
      setTimeout(() => setSaved(null), 2000);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(null);
    }
  }

  return (
    <AdminShell>
      <div className="max-w-3xl mx-auto py-8 px-4">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-white">⚙️ Prize Fulfillment Mode</h1>
          <p className="text-gray-400 text-sm mt-1">
            Control whether each prize type is fulfilled automatically at spin time (AUTO)
            or requires the user to click &quot;Claim&quot; from their dashboard (MANUAL).
            <br />
            <span className="text-yellow-400 font-medium">
              AUTO mode is only available for Airtime and Data Bundle prizes.
            </span>{" "}
            All other prize types remain in MANUAL mode.
          </p>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-lg bg-red-500/20 border border-red-500/40 text-red-300 text-sm">
            {error}
          </div>
        )}

        {loading ? (
          <div className="text-gray-400 text-sm py-12 text-center">Loading…</div>
        ) : (
          <div className="space-y-4">
            {configs.map(cfg => {
              const edit    = edits[cfg.prize_type] ?? { fulfillment_mode: "MANUAL", max_retry_attempts: 3, retry_delay_seconds: 30, fallback_to_manual: true };
              const canAuto = AUTO_PROVISIONABLE.has(cfg.prize_type);
              const isSaving = saving === cfg.prize_type;
              const isSaved  = saved  === cfg.prize_type;

              return (
                <div
                  key={cfg.prize_type}
                  className={`rounded-xl border p-5 transition-all ${
                    canAuto
                      ? "bg-gray-800 border-gray-700"
                      : "bg-gray-800/40 border-gray-700/40 opacity-60"
                  }`}
                >
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <span className="text-xl">{PRIZE_ICONS[cfg.prize_type] ?? "🎁"}</span>
                      <span className="font-semibold text-white text-base">
                        {PRIZE_LABELS[cfg.prize_type] ?? cfg.prize_type}
                      </span>
                      {!canAuto && (
                        <span className="text-xs px-2 py-0.5 rounded-full bg-gray-700 text-gray-400">
                          MANUAL only
                        </span>
                      )}
                    </div>

                    {/* Mode toggle */}
                    <div className="flex items-center gap-1 bg-gray-900 rounded-lg p-1">
                      {(["MANUAL", "AUTO"] as const).map(mode => (
                        <button
                          key={mode}
                          disabled={!canAuto || isSaving}
                          onClick={() => canAuto && updateEdit(cfg.prize_type, { fulfillment_mode: mode })}
                          className={`px-3 py-1 rounded-md text-sm font-medium transition-all ${
                            edit.fulfillment_mode === mode
                              ? mode === "AUTO"
                                ? "bg-green-600 text-white"
                                : "bg-blue-600 text-white"
                              : "text-gray-400 hover:text-gray-200"
                          } disabled:opacity-40 disabled:cursor-not-allowed`}
                        >
                          {mode}
                        </button>
                      ))}
                    </div>
                  </div>

                  {/* Advanced options — only show when AUTO is selected and this type supports it */}
                  {canAuto && edit.fulfillment_mode === "AUTO" && (
                    <div className="mt-3 grid grid-cols-3 gap-3 border-t border-gray-700 pt-3">
                      <div>
                        <label className="block text-xs text-gray-400 mb-1">Max retries</label>
                        <input
                          type="number"
                          min={1}
                          max={10}
                          value={edit.max_retry_attempts}
                          onChange={e => updateEdit(cfg.prize_type, { max_retry_attempts: parseInt(e.target.value) || 1 })}
                          className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1 text-white text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-gray-400 mb-1">Retry delay (s)</label>
                        <input
                          type="number"
                          min={5}
                          value={edit.retry_delay_seconds}
                          onChange={e => updateEdit(cfg.prize_type, { retry_delay_seconds: parseInt(e.target.value) || 30 })}
                          className="w-full bg-gray-900 border border-gray-600 rounded px-2 py-1 text-white text-sm"
                        />
                      </div>
                      <div className="flex items-end pb-1">
                        <label className="flex items-center gap-2 cursor-pointer select-none">
                          <input
                            type="checkbox"
                            checked={edit.fallback_to_manual}
                            onChange={e => updateEdit(cfg.prize_type, { fallback_to_manual: e.target.checked })}
                            className="accent-blue-500 w-4 h-4"
                          />
                          <span className="text-xs text-gray-300">Fallback to MANUAL on failure</span>
                        </label>
                      </div>
                    </div>
                  )}

                  {/* Save row */}
                  {canAuto && (
                    <div className="mt-3 flex items-center justify-end gap-3">
                      {isSaved && (
                        <span className="text-green-400 text-xs">✓ Saved</span>
                      )}
                      <button
                        onClick={() => save(cfg.prize_type)}
                        disabled={isSaving}
                        className="px-4 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium transition-colors"
                      >
                        {isSaving ? "Saving…" : "Save"}
                      </button>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        <div className="mt-6 p-4 rounded-xl bg-gray-900 border border-gray-700 text-xs text-gray-400 space-y-1">
          <p><strong className="text-gray-300">MANUAL</strong> — User wins prize → sees it in dashboard → clicks &quot;Claim&quot; → VTPass fires. Default for all prize types.</p>
          <p><strong className="text-gray-300">AUTO</strong> — User wins prize → VTPass fires immediately in background (up to max retries) → if all retries fail and fallback is on, reverts to MANUAL claim.</p>
          <p className="text-yellow-400/80">Changes take effect immediately for new spins. In-flight prizes already set to pending_claim are not affected.</p>
        </div>
      </div>
    </AdminShell>
  );
}
