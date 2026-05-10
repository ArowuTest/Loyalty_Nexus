"use client";
import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI from "@/lib/api";

interface Setting {
  key: string;
  value: string;
  label: string;
  description: string;
  category: string;
  updated_at: string;
  updated_by: string;
}

const CATEGORY_LABELS: Record<string, string> = {
  storage: "🗂️ Asset Storage & TTL",
  general: "⚙️ General",
};

const TIER_ORDER = [
  "storage_ttl_platinum_hours",
  "storage_ttl_gold_hours",
  "storage_ttl_silver_hours",
  "storage_ttl_bronze_hours",
  "storage_ttl_free_hours",
  "notify_expiry_first_hours",
  "notify_expiry_second_hours",
];

export default function PlatformSettingsPage() {
  const [settings, setSettings] = useState<Setting[]>([]);
  const [editing, setEditing] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState<Record<string, boolean>>({});
  const [saved, setSaved] = useState<Record<string, boolean>>({});
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchSettings = useCallback(async () => {
    try {
      const data = await adminAPI.req<{ settings: Setting[]; count: number }>("GET", "/admin/settings");
      setSettings(data.settings ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load settings");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchSettings(); }, [fetchSettings]);

  const handleEdit = (key: string, value: string) => {
    setEditing(prev => ({ ...prev, [key]: value }));
  };

  const handleSave = async (key: string) => {
    const value = editing[key];
    if (value === undefined) return;
    // Validate numeric fields
    const num = Number(value);
    if (isNaN(num) || num <= 0) {
      setError(`"${key}": value must be a positive number`);
      return;
    }
    setSaving(prev => ({ ...prev, [key]: true }));
    setError(null);
    try {
      await adminAPI.req<{ ok: boolean }>("PATCH", "/admin/settings", { key, value });
      // Update local state
      setSettings(prev =>
        prev.map(s => (s.key === key ? { ...s, value, updated_at: new Date().toISOString() } : s))
      );
      setEditing(prev => { const n = { ...prev }; delete n[key]; return n; });
      setSaved(prev => ({ ...prev, [key]: true }));
      setTimeout(() => setSaved(prev => { const n = { ...prev }; delete n[key]; return n; }), 2500);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(prev => ({ ...prev, [key]: false }));
    }
  };

  // Group by category
  const byCategory: Record<string, Setting[]> = {};
  for (const s of settings) {
    if (!byCategory[s.category]) byCategory[s.category] = [];
    byCategory[s.category].push(s);
  }
  // Sort each group by tier order
  for (const cat of Object.keys(byCategory)) {
    byCategory[cat].sort((a, b) => {
      const ai = TIER_ORDER.indexOf(a.key);
      const bi = TIER_ORDER.indexOf(b.key);
      if (ai === -1 && bi === -1) return a.key.localeCompare(b.key);
      if (ai === -1) return 1;
      if (bi === -1) return -1;
      return ai - bi;
    });
  }

  const tierColor: Record<string, string> = {
    platinum: "text-violet-400",
    gold:     "text-amber-400",
    silver:   "text-slate-300",
    bronze:   "text-orange-400",
    free:     "text-zinc-400",
  };

  const getTierFromKey = (key: string) =>
    Object.keys(tierColor).find(t => key.includes(t)) ?? "";

  return (
    <AdminShell>
      <div className="max-w-3xl mx-auto py-8 px-4 space-y-8">
        {/* Header */}
        <div>
          <h1 className="text-2xl font-bold text-white">Platform Settings</h1>
          <p className="text-sm text-white/50 mt-1">
            Admin-configurable values — changes take effect within 5 minutes (Redis cache TTL).
            No code deployment needed.
          </p>
        </div>

        {error && (
          <div className="rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-red-300 text-sm">
            {error}
          </div>
        )}

        {loading ? (
          <div className="text-white/40 text-sm animate-pulse">Loading settings…</div>
        ) : (
          Object.entries(byCategory).map(([cat, rows]) => (
            <section key={cat} className="space-y-3">
              <h2 className="text-sm font-semibold text-white/60 uppercase tracking-wider">
                {CATEGORY_LABELS[cat] ?? cat}
              </h2>

              <div className="rounded-2xl border border-white/8 bg-white/[0.02] overflow-hidden divide-y divide-white/[0.06]">
                {rows.map(s => {
                  const isEditing = editing[s.key] !== undefined;
                  const currentVal = isEditing ? editing[s.key] : s.value;
                  const tierKey = getTierFromKey(s.key);
                  const tierClass = tierColor[tierKey] ?? "text-white/70";

                  return (
                    <div key={s.key} className="flex items-center gap-4 px-5 py-4">
                      {/* Label + description */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className={`text-sm font-semibold ${tierClass}`}>
                            {s.label}
                          </span>
                        </div>
                        {s.description && (
                          <p className="text-[11px] text-white/35 mt-0.5 leading-snug">
                            {s.description}
                          </p>
                        )}
                        <p className="text-[10px] text-white/20 mt-1 font-mono">{s.key}</p>
                      </div>

                      {/* Value input */}
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <div className="relative">
                          <input
                            type="number"
                            min="1"
                            value={currentVal}
                            onChange={e => handleEdit(s.key, e.target.value)}
                            className={`w-20 text-right rounded-lg border px-3 py-2 text-sm font-mono
                              bg-white/[0.04] text-white/90 outline-none transition-all
                              ${isEditing
                                ? "border-gold-500/50 ring-1 ring-gold-500/20"
                                : "border-white/10 hover:border-white/25"
                              }`}
                          />
                          <span className="absolute right-2 -bottom-4 text-[10px] text-white/25">
                            {s.key.includes("hours") ? "hours" : ""}
                          </span>
                        </div>

                        {isEditing ? (
                          <button
                            onClick={() => handleSave(s.key)}
                            disabled={saving[s.key]}
                            className="px-3 py-2 rounded-lg bg-gold-500/15 border border-gold-500/30
                                       text-gold-400 text-xs font-semibold
                                       hover:bg-gold-500/25 transition-all disabled:opacity-50"
                          >
                            {saving[s.key] ? "Saving…" : "Save"}
                          </button>
                        ) : saved[s.key] ? (
                          <span className="text-green-400 text-xs font-semibold">✓ Saved</span>
                        ) : (
                          <div className="w-14" /> /* spacer */
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>

              {/* Helpful summary for storage category */}
              {cat === "storage" && (
                <div className="rounded-xl border border-white/6 bg-white/[0.015] px-4 py-3 text-[11px] text-white/35 space-y-1">
                  <p className="font-semibold text-white/45">How this works:</p>
                  <p>• TTL is set when a generation is created and is based on the user&apos;s membership tier at that moment.</p>
                  <p>• Assets are purged from cloud storage after the TTL expires (next hourly cleanup run).</p>
                  <p>• Users get push + SMS notifications at the &quot;First Warning&quot; and &quot;Second Warning&quot; windows before expiry.</p>
                  <p>• Changes take effect for new generations immediately (within 5 min cache refresh). Existing generations are not retroactively changed.</p>
                </div>
              )}
            </section>
          ))
        )}
      </div>
    </AdminShell>
  );
}
