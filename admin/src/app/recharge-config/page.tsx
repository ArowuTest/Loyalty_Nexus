"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState, useCallback } from "react";
import adminAPI, { RechargeConfig } from "@/lib/api";

// ─── Types ──────────────────────────────────────────────────────────────────
type Field = keyof RechargeConfig;

interface FieldDef {
  key: Field;
  label: string;
  description: string;
  note: string;
  min: number;
  suffix: string;
}

const FIELDS: FieldDef[] = [
  {
    key: "spin_naira_per_credit",
    label: "Spin Credit Threshold (₦)",
    description: "Minimum cumulative daily recharge (naira) to qualify for the Bronze spin tier and earn 1 Spin Credit.",
    note:
      "This is the primary lever for controlling how quickly users accumulate spin credits. " +
      "Lowering this value makes spin credits easier to earn and will increase spin volume. " +
      "Raising it reduces spin frequency and protects prize liability. Default: ₦1,000.",
    min: 1,
    suffix: "₦ per credit",
  },
  {
    key: "draw_naira_per_entry",
    label: "Draw Entry Threshold (₦)",
    description: "How many naira a user must recharge (per transaction) to earn 1 Draw Entry.",
    note:
      "Draw entries use a flat per-transaction accumulator — every ₦X recharged in a single push " +
      "awards one draw entry. This is separate from the spin credit tier system. Default: ₦200.",
    min: 1,
    suffix: "₦ per entry",
  },
  {
    key: "pulse_naira_per_point",
    label: "Bonus Pulse Rate (₦ per point)",
    description: "How many naira of recharge value equals 1 Bonus Pulse point.",
    note:
      "Bonus Pulse points are the secondary loyalty currency awarded on recharges. " +
      "A lower value means users earn more points per recharge — useful for promotional periods. " +
      "A higher value reduces point inflation and extends the lifetime value of the points economy. Default: ₦250.",
    min: 1,
    suffix: "₦ per point",
  },
  {
    key: "spin_max_per_day",
    label: "Daily Spin Cap (Platinum Tier)",
    description: "Maximum spin credits a user can earn per calendar day (applies as the Platinum tier cap).",
    note:
      "This is a hard daily ceiling on spin credits regardless of how much a user recharges. " +
      "It protects against prize liability from very high-volume rechargers. " +
      "Individual tier caps in the Spin Config page may be lower than this value. Default: 5.",
    min: 1,
    suffix: "spins/day max",
  },
  {
    key: "min_amount_naira",
    label: "MTN Push Minimum Recharge (₦)",
    description: "The minimum recharge amount (in naira) required for an MTN Push transaction to be processed and rewarded.",
    note:
      "Transactions below this threshold are silently skipped by the MTN Push webhook processor and the CSV fallback uploader. " +
      "Set this to match the minimum denomination sold by your MTN distribution partners. " +
      "Setting it to 0 disables the minimum check entirely. Default: ₦50.",
    min: 0,
    suffix: "₦ minimum",
  },
];

// ─── Helpers ─────────────────────────────────────────────────────────────────
function badge(text: string, color: string) {
  return (
    <span style={{
      display: "inline-block", padding: "2px 10px", borderRadius: 99,
      fontSize: 11, fontWeight: 600, background: color + "22", color,
    }}>
      {text}
    </span>
  );
}

// ─── Config Field Component ───────────────────────────────────────────────────
function ConfigField({
  def, value, saving, saved, onSave,
}: {
  def: FieldDef;
  value: number;
  saving: Field | null;
  saved: Field | null;
  onSave: (key: Field, val: number) => void;
}) {
  const [localVal, setLocalVal] = useState(String(value));
  useEffect(() => { setLocalVal(String(value)); }, [value]);

  const isSaving = saving === def.key;
  const isSaved  = saved  === def.key;
  const dirty    = Number(localVal) !== value;

  return (
    <div style={{
      background: "#fff", borderRadius: 12, padding: 24, marginBottom: 20,
      border: "1px solid #e5e7eb", boxShadow: "0 1px 3px rgba(0,0,0,0.04)",
    }}>
      {/* Header */}
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 8 }}>
        <div>
          <div style={{ fontWeight: 700, fontSize: 15, color: "#111827", marginBottom: 4 }}>
            {def.label}
          </div>
          <div style={{ fontSize: 13, color: "#6b7280" }}>{def.description}</div>
        </div>
        {isSaved && badge("Saved", "#10b981")}
      </div>

      {/* Explanatory note */}
      <div style={{
        background: "#f0f4ff", border: "1px solid #c7d2fe",
        borderRadius: 8, padding: "10px 14px", margin: "12px 0",
        fontSize: 12, color: "#4338ca", lineHeight: 1.6,
      }}>
        <strong>ℹ️ What this controls:</strong> {def.note}
      </div>

      {/* Input row */}
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginTop: 12 }}>
        <input
          type="number"
          min={def.min}
          value={localVal}
          onChange={e => setLocalVal(e.target.value)}
          style={{
            width: 140, padding: "8px 12px", borderRadius: 8, fontSize: 15,
            border: dirty ? "1.5px solid #5f72f9" : "1.5px solid #e5e7eb",
            outline: "none", fontWeight: 600, color: "#111827",
          }}
        />
        <span style={{ fontSize: 13, color: "#9ca3af" }}>{def.suffix}</span>
        <button
          disabled={!dirty || isSaving}
          onClick={() => onSave(def.key, Number(localVal))}
          style={{
            marginLeft: "auto", padding: "8px 20px", borderRadius: 8,
            background: dirty && !isSaving ? "#5f72f9" : "#e5e7eb",
            color: dirty && !isSaving ? "#fff" : "#9ca3af",
            border: "none", cursor: dirty && !isSaving ? "pointer" : "default",
            fontWeight: 600, fontSize: 13, transition: "all 0.15s",
          }}
        >
          {isSaving ? "Saving…" : "Save"}
        </button>
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────
export default function RechargeConfigPage() {
  const [config, setConfig]   = useState<RechargeConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError]     = useState<string | null>(null);
  const [saving, setSaving]   = useState<Field | null>(null);
  const [saved, setSaved]     = useState<Field | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await adminAPI.getRechargeConfig();
      setConfig(data);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleSave = async (key: Field, val: number) => {
    if (!config) return;
    setSaving(key);
    try {
      const updated = await adminAPI.updateRechargeConfig({ [key]: val });
      setConfig(updated);
      setSaved(key);
      setTimeout(() => setSaved(null), 2500);
    } catch (e) {
      alert((e as Error).message);
    } finally {
      setSaving(null);
    }
  };

  return (
    <AdminShell>
      {/* Page header */}
      <div style={{ marginBottom: 28 }}>
        <h1 style={{ fontSize: 22, fontWeight: 800, color: "#111827", margin: 0 }}>
          ⚡ Recharge Reward Configuration
        </h1>
        <p style={{ fontSize: 14, color: "#6b7280", marginTop: 6, maxWidth: 680 }}>
          These settings control how recharge transactions are translated into rewards — Spin Credits,
          Draw Entries, Bonus Pulse points, and the minimum qualifying amount for MTN Push processing.
          Changes take effect immediately for all new transactions; existing credited rewards are not affected.
        </p>

        {/* Prominent context banner */}
        <div style={{
          marginTop: 16, background: "#fffbeb", border: "1px solid #fcd34d",
          borderRadius: 10, padding: "12px 18px", maxWidth: 680,
          fontSize: 13, color: "#92400e", lineHeight: 1.7,
        }}>
          <strong>⚠️ Important:</strong> These values are separate from the <em>Points Engine</em> config
          (which controls base earning rates and streak bonuses). This page specifically governs the
          recharge-to-reward conversion rates used by the <strong>MTN Push webhook</strong> and the
          <strong> CSV fallback uploader</strong>. If you are unsure which setting to change, consult
          the product team before saving.
        </div>
      </div>

      {/* Loading / error states */}
      {loading && (
        <div style={{ color: "#6b7280", fontSize: 14 }}>Loading configuration…</div>
      )}
      {error && (
        <div style={{
          background: "#fef2f2", border: "1px solid #fca5a5", borderRadius: 8,
          padding: "12px 16px", color: "#b91c1c", fontSize: 13, marginBottom: 20,
        }}>
          Failed to load config: {error}
          <button onClick={load} style={{ marginLeft: 12, color: "#5f72f9", background: "none", border: "none", cursor: "pointer", fontSize: 13 }}>
            Retry
          </button>
        </div>
      )}

      {/* Config fields */}
      {config && !loading && (
        <div style={{ maxWidth: 680 }}>
          {FIELDS.map(def => (
            <ConfigField
              key={def.key}
              def={def}
              value={config[def.key]}
              saving={saving}
              saved={saved}
              onSave={handleSave}
            />
          ))}

          {/* Footer note */}
          <div style={{
            marginTop: 8, padding: "12px 16px", borderRadius: 8,
            background: "#f9fafb", border: "1px solid #e5e7eb",
            fontSize: 12, color: "#9ca3af",
          }}>
            Current values are loaded live from the server. Each field saves independently — you do not
            need to save all fields at once. Changes are logged in the audit trail under{" "}
            <a href="/config" style={{ color: "#5f72f9" }}>⚙️ Config</a>.
          </div>
        </div>
      )}
    </AdminShell>
  );
}
