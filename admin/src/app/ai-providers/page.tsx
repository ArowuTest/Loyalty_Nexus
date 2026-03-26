"use client";

import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, {
  AIProviderConfig,
  AIProvidersResponse,
  AIProviderMeta,
  AIProviderFormPayload,
  AIProviderTestResult,
} from "@/lib/api";
import {
  Plus, RefreshCw, X, Edit3, Trash2, ChevronDown, ChevronUp,
  CheckCircle2, XCircle, Zap, Play, Power, PowerOff, Key,
  AlertTriangle, Loader2, Info, Database,
} from "lucide-react";

// ── Theme (matches studio-tools palette) ─────────────────────────────────────
const BG        = "#0d0e1a";
const CARD_BG   = "rgba(95,114,249,0.05)";
const PRIMARY   = "#5f72f9";
const SUCCESS   = "#34d399";
const DANGER    = "#f87171";
const WARN      = "#f59e0b";
const TEXT      = "#e2e8ff";
const MUTED     = "#828cb4";
const BORDER    = "rgba(95,114,249,0.12)";
const BORDER_HI = "rgba(95,114,249,0.25)";
const INPUT_BG  = "rgba(255,255,255,0.04)";

// ── Category colour map ───────────────────────────────────────────────────────
const CAT_COLOR: Record<string, string> = {
  text:       "#22d3ee",
  image:      "#a78bfa",
  video:      "#f59e0b",
  tts:        "#34d399",
  transcribe: "#38bdf8",
  translate:  "#fb923c",
  music:      "#e879f9",
  "bg-remove":"#4ade80",
  vision:     "#c084fc",
};
const catColor = (c: string) => CAT_COLOR[c] || PRIMARY;

// ── Priority label ────────────────────────────────────────────────────────────
const priorityLabel = (n: number) =>
  n === 1 ? "Primary" : n === 2 ? "Backup 1" : n === 3 ? "Backup 2" : `Tier ${n}`;

// ── Time helper ───────────────────────────────────────────────────────────────
function timeAgo(iso: string | null | undefined) {
  if (!iso) return "—";
  const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (s < 60) return `${s}s ago`;
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

// ── Empty form ────────────────────────────────────────────────────────────────
const EMPTY_FORM: AIProviderFormPayload = {
  name: "", category: "text", template: "openai-compatible",
  env_key: "", api_key: "", model_id: "",
  extra_config: {}, priority: 10, is_primary: false,
  is_active: true, cost_micros: 0, pulse_pts: 0, notes: "",
};

// ── Badge ─────────────────────────────────────────────────────────────────────
function Badge({ label, color }: { label: string; color?: string }) {
  const c = color || PRIMARY;
  return (
    <span style={{
      background: `${c}18`, color: c, border: `1px solid ${c}40`,
      borderRadius: 6, fontSize: 10, fontWeight: 700,
      padding: "2px 7px", letterSpacing: "0.04em",
      textTransform: "uppercase" as const,
    }}>
      {label}
    </span>
  );
}

// ── Status indicator ──────────────────────────────────────────────────────────
function StatusDot({ ok, tested }: { ok: boolean | null; tested: boolean }) {
  if (!tested) return <span style={{ color: MUTED, fontSize: 11 }}>—</span>;
  return ok
    ? <CheckCircle2 size={14} style={{ color: SUCCESS }} />
    : <XCircle size={14} style={{ color: DANGER }} />;
}

// ── Small label ───────────────────────────────────────────────────────────────
function Label({ children }: { children: React.ReactNode }) {
  return (
    <label style={{ fontSize: 11, fontWeight: 600, color: MUTED,
                    letterSpacing: "0.05em", textTransform: "uppercase" as const,
                    display: "block", marginBottom: 4 }}>
      {children}
    </label>
  );
}

// ── Form input ────────────────────────────────────────────────────────────────
function FInput({
  value, onChange, placeholder, type = "text",
}: {
  value: string; onChange: (v: string) => void;
  placeholder?: string; type?: string;
}) {
  return (
    <input
      type={type}
      value={value}
      onChange={e => onChange(e.target.value)}
      placeholder={placeholder}
      style={{
        width: "100%", background: INPUT_BG, border: `1px solid ${BORDER_HI}`,
        borderRadius: 8, padding: "8px 12px", color: TEXT, fontSize: 13,
        outline: "none", boxSizing: "border-box" as const,
      }}
    />
  );
}

function FSelect({
  value, onChange, options,
}: {
  value: string;
  onChange: (v: string) => void;
  options: string[] | { value: string; label: string }[];
}) {
  const normalized = options.map(o =>
    typeof o === "string" ? { value: o, label: o } : o
  );
  return (
    <select
      value={value}
      onChange={e => onChange(e.target.value)}
      style={{
        width: "100%", background: INPUT_BG, border: `1px solid ${BORDER_HI}`,
        borderRadius: 8, padding: "8px 12px", color: TEXT, fontSize: 13,
        outline: "none", cursor: "pointer",
      }}
    >
      {normalized.map(o => (
        <option key={o.value} value={o.value}>{o.label}</option>
      ))}
    </select>
  );
}

// ── Provider Card ──────────────────────────────────────────────────────────────
function ProviderCard({
  provider,
  meta,
  onEdit,
  onDelete,
  onToggleActive,
  onTest,
  testing,
}: {
  provider: AIProviderConfig;
  meta: AIProviderMeta | null;
  onEdit: (p: AIProviderConfig) => void;
  onDelete: (p: AIProviderConfig) => void;
  onToggleActive: (p: AIProviderConfig) => void;
  onTest: (p: AIProviderConfig) => void;
  testing: boolean;
}) {
  const cc = catColor(provider.category);
  const templateDesc = meta?.template_descriptions?.[provider.template] ?? "";

  return (
    <div style={{
      background: CARD_BG,
      border: `1px solid ${provider.is_active ? BORDER_HI : BORDER}`,
      borderLeft: `3px solid ${provider.is_active ? cc : MUTED}`,
      borderRadius: 10,
      padding: "14px 16px",
      display: "flex",
      flexDirection: "column" as const,
      gap: 10,
      opacity: provider.is_active ? 1 : 0.55,
      transition: "opacity .2s",
    }}>
      {/* Header row */}
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 8 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" as const }}>
            <span style={{ fontWeight: 700, color: TEXT, fontSize: 14 }}>{provider.name}</span>
            <Badge label={provider.category} color={cc} />
            <Badge label={priorityLabel(provider.priority)}
                   color={provider.priority === 1 ? SUCCESS : provider.priority <= 2 ? WARN : MUTED} />
            {provider.is_primary && <Badge label="PRIMARY" color={PRIMARY} />}
            {!provider.is_active && <Badge label="DISABLED" color={MUTED} />}
          </div>
          <div style={{ fontSize: 11, color: MUTED, marginTop: 3 }}>
            <code style={{ color: "#a5b4fc" }}>{provider.slug}</code>
            {provider.model_id && <> · model: <code style={{ color: WARN }}>{provider.model_id}</code></>}
          </div>
        </div>

        {/* Action buttons */}
        <div style={{ display: "flex", gap: 4, flexShrink: 0 }}>
          {/* Test */}
          <button
            onClick={() => onTest(provider)}
            disabled={testing}
            title="Ping provider to verify credentials"
            style={{
              background: "rgba(95,114,249,0.12)", border: `1px solid ${BORDER_HI}`,
              borderRadius: 7, padding: "5px 8px", cursor: "pointer", color: PRIMARY,
              display: "flex", alignItems: "center", gap: 4, fontSize: 11,
            }}
          >
            {testing ? <Loader2 size={12} style={{ animation: "spin 1s linear infinite" }} /> : <Play size={12} />}
            Test
          </button>
          {/* Toggle active */}
          <button
            onClick={() => onToggleActive(provider)}
            title={provider.is_active ? "Disable provider" : "Enable provider"}
            style={{
              background: provider.is_active ? "rgba(248,113,113,0.1)" : "rgba(52,211,153,0.1)",
              border: `1px solid ${provider.is_active ? "rgba(248,113,113,0.3)" : "rgba(52,211,153,0.3)"}`,
              borderRadius: 7, padding: "5px 8px", cursor: "pointer",
              color: provider.is_active ? DANGER : SUCCESS,
            }}
          >
            {provider.is_active ? <PowerOff size={13} /> : <Power size={13} />}
          </button>
          {/* Edit */}
          <button
            onClick={() => onEdit(provider)}
            title="Edit"
            style={{
              background: CARD_BG, border: `1px solid ${BORDER_HI}`,
              borderRadius: 7, padding: "5px 8px", cursor: "pointer", color: TEXT,
            }}
          >
            <Edit3 size={13} />
          </button>
          {/* Delete */}
          <button
            onClick={() => onDelete(provider)}
            title="Delete"
            style={{
              background: "rgba(248,113,113,0.1)", border: `1px solid rgba(248,113,113,0.3)`,
              borderRadius: 7, padding: "5px 8px", cursor: "pointer", color: DANGER,
            }}
          >
            <Trash2 size={13} />
          </button>
        </div>
      </div>

      {/* Details row */}
      <div style={{ display: "flex", flexWrap: "wrap" as const, gap: 12, fontSize: 11, color: MUTED }}>
        {/* Template */}
        <span title={templateDesc}>
          <span style={{ color: "#c084fc" }}>template:</span> {provider.template}
        </span>
        {/* Key status */}
        <span>
          <Key size={10} style={{ verticalAlign: "middle", marginRight: 3 }} />
          {provider.has_key
            ? <span style={{ color: SUCCESS }}>Key configured</span>
            : <span style={{ color: DANGER }}>No key</span>}
          {provider.env_key && <> (env: <code>{provider.env_key}</code>)</>}
        </span>
        {/* Cost */}
        {provider.cost_micros > 0 && (
          <span>
            cost: <span style={{ color: WARN }}>${(provider.cost_micros / 1_000_000).toFixed(4)}</span>
          </span>
        )}
        {/* Pulse pts */}
        {provider.pulse_pts > 0 && (
          <span>
            pts: <span style={{ color: "#a78bfa" }}>{provider.pulse_pts}</span>
          </span>
        )}
      </div>

      {/* Notes */}
      {provider.notes && (
        <div style={{ fontSize: 11, color: MUTED, fontStyle: "italic" }}>{provider.notes}</div>
      )}

      {/* Last test result */}
      <div style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 11, color: MUTED }}>
        <StatusDot ok={provider.last_test_ok ?? null} tested={provider.last_tested_at != null} />
        {provider.last_tested_at && (
          <span>
            Tested {timeAgo(provider.last_tested_at)}
            {provider.last_test_msg && (
              <span style={{ color: provider.last_test_ok ? SUCCESS : DANGER, marginLeft: 6 }}>
                — {provider.last_test_msg}
              </span>
            )}
          </span>
        )}
        {!provider.last_tested_at && <span>Never tested</span>}
      </div>
    </div>
  );
}

// ── Provider Form (Create / Edit) ─────────────────────────────────────────────
function ProviderForm({
  initial,
  meta,
  onSave,
  onCancel,
  saving,
}: {
  initial: AIProviderFormPayload & { id?: string };
  meta: AIProviderMeta | null;
  onSave: (data: AIProviderFormPayload & { id?: string }) => void;
  onCancel: () => void;
  saving: boolean;
}) {
  const [form, setForm] = useState({ ...EMPTY_FORM, ...initial });
  const set = (k: keyof AIProviderFormPayload) => (v: unknown) =>
    setForm(f => ({ ...f, [k]: v }));

  const templateDesc = meta?.template_descriptions?.[form.template ?? ""] ?? "";

  return (
    <div style={{
      position: "fixed" as const, inset: 0,
      background: "rgba(0,0,0,0.75)", zIndex: 1000,
      display: "flex", alignItems: "center", justifyContent: "center",
      padding: 16,
    }}>
      <div style={{
        background: "#131425", border: `1px solid ${BORDER_HI}`,
        borderRadius: 14, padding: 24, width: "100%", maxWidth: 620,
        maxHeight: "90vh", overflowY: "auto" as const,
      }}>
        <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 20 }}>
          <h2 style={{ color: TEXT, margin: 0, fontSize: 17 }}>
            {initial.id ? "Edit Provider" : "Add Provider"}
          </h2>
          <button onClick={onCancel} style={{ background: "none", border: "none", cursor: "pointer", color: MUTED }}>
            <X size={18} />
          </button>
        </div>

        <div style={{ display: "grid", gap: 14 }}>
          {/* Name */}
          <div>
            <Label>Name *</Label>
            <FInput value={form.name} onChange={set("name")} placeholder="e.g. Pollinations FLUX" />
          </div>

          {/* Category + Template */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
            <div>
              <Label>Category *</Label>
              <FSelect
                value={form.category}
                onChange={set("category")}
                options={meta?.categories ?? ["text","image","video","tts","transcribe","translate","music","bg-remove","vision"]}
              />
            </div>
            <div>
              <Label>Template *</Label>
              <FSelect
                value={form.template}
                onChange={set("template")}
                options={meta?.templates ?? []}
              />
            </div>
          </div>
          {templateDesc && (
            <div style={{ fontSize: 11, color: MUTED, marginTop: -8 }}>
              <Info size={10} style={{ verticalAlign: "middle", marginRight: 3 }} />
              {templateDesc}
            </div>
          )}

          {/* Model ID */}
          <div>
            <Label>Model / Endpoint ID</Label>
            <FInput value={form.model_id ?? ""} onChange={set("model_id")}
                    placeholder="e.g. gemini-2.5-flash" />
          </div>

          {/* Env key + API key */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
            <div>
              <Label>Env Var Name</Label>
              <FInput value={form.env_key ?? ""} onChange={set("env_key")}
                      placeholder="e.g. FAL_API_KEY" />
              <div style={{ fontSize: 10, color: MUTED, marginTop: 3 }}>
                Name of env var already set on the server
              </div>
            </div>
            <div>
              <Label>API Key (optional override)</Label>
              <FInput value={form.api_key ?? ""} onChange={set("api_key")}
                      type="password" placeholder="sk_... (stored encrypted)" />
              <div style={{ fontSize: 10, color: MUTED, marginTop: 3 }}>
                Stored encrypted. Leave blank to use env var.
              </div>
            </div>
          </div>

          {/* Priority + is_primary */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
            <div>
              <Label>Priority (1=Primary, higher=Backup)</Label>
              <FInput value={String(form.priority ?? 10)} onChange={v => set("priority")(parseInt(v) || 10)} />
            </div>
            <div style={{ display: "flex", flexDirection: "column" as const, justifyContent: "flex-end", gap: 8 }}>
              <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", color: TEXT, fontSize: 13 }}>
                <input type="checkbox" checked={!!form.is_primary}
                       onChange={e => set("is_primary")(e.target.checked)} />
                Mark as primary provider for category
              </label>
              <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", color: TEXT, fontSize: 13 }}>
                <input type="checkbox" checked={!!form.is_active}
                       onChange={e => set("is_active")(e.target.checked)} />
                Active (enable in dispatch chain)
              </label>
            </div>
          </div>

          {/* Cost + Pulse pts */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
            <div>
              <Label>Cost per call (microdollars)</Label>
              <FInput value={String(form.cost_micros ?? 0)} onChange={v => set("cost_micros")(parseInt(v) || 0)} />
              <div style={{ fontSize: 10, color: MUTED, marginTop: 3 }}>
                1 USD = 1,000,000 µ$. e.g. $0.025 → 25000
              </div>
            </div>
            <div>
              <Label>Pulse Points charged per call</Label>
              <FInput value={String(form.pulse_pts ?? 0)} onChange={v => set("pulse_pts")(parseInt(v) || 0)} />
            </div>
          </div>

          {/* Notes */}
          <div>
            <Label>Notes</Label>
            <FInput value={form.notes ?? ""} onChange={set("notes")} placeholder="Admin notes / description" />
          </div>
        </div>

        {/* Actions */}
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 10, marginTop: 20 }}>
          <button onClick={onCancel} style={{
            background: "none", border: `1px solid ${BORDER_HI}`,
            borderRadius: 8, padding: "8px 16px", cursor: "pointer", color: MUTED, fontSize: 13,
          }}>
            Cancel
          </button>
          <button
            onClick={() => onSave({ ...form, id: initial.id })}
            disabled={saving || !form.name || !form.category || !form.template}
            style={{
              background: PRIMARY, border: "none", borderRadius: 8,
              padding: "8px 18px", cursor: "pointer", color: "#fff", fontSize: 13,
              fontWeight: 600, opacity: saving ? 0.6 : 1,
              display: "flex", alignItems: "center", gap: 6,
            }}
          >
            {saving && <Loader2 size={13} style={{ animation: "spin 1s linear infinite" }} />}
            {initial.id ? "Save Changes" : "Add Provider"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Delete confirm modal ───────────────────────────────────────────────────────
function DeleteConfirm({
  provider,
  onConfirm,
  onCancel,
  deleting,
}: {
  provider: AIProviderConfig;
  onConfirm: () => void;
  onCancel: () => void;
  deleting: boolean;
}) {
  return (
    <div style={{
      position: "fixed" as const, inset: 0,
      background: "rgba(0,0,0,0.75)", zIndex: 1001,
      display: "flex", alignItems: "center", justifyContent: "center",
    }}>
      <div style={{
        background: "#131425", border: `1px solid rgba(248,113,113,0.4)`,
        borderRadius: 14, padding: 24, maxWidth: 380,
      }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 12 }}>
          <AlertTriangle size={20} style={{ color: DANGER }} />
          <h3 style={{ color: TEXT, margin: 0, fontSize: 15 }}>Delete Provider</h3>
        </div>
        <p style={{ color: MUTED, fontSize: 13, marginBottom: 20 }}>
          Remove <strong style={{ color: TEXT }}>{provider.name}</strong>?
          This cannot be undone. If this is the only provider for its category,
          the dispatch chain will use hardcoded fallbacks.
        </p>
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 10 }}>
          <button onClick={onCancel} style={{
            background: "none", border: `1px solid ${BORDER_HI}`,
            borderRadius: 8, padding: "7px 14px", cursor: "pointer", color: MUTED, fontSize: 13,
          }}>
            Cancel
          </button>
          <button onClick={onConfirm} disabled={deleting} style={{
            background: DANGER, border: "none", borderRadius: 8,
            padding: "7px 14px", cursor: "pointer", color: "#fff", fontSize: 13,
            opacity: deleting ? 0.6 : 1,
          }}>
            {deleting ? "Deleting…" : "Delete"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Main Page ─────────────────────────────────────────────────────────────────
export default function AIProvidersPage() {
  const [data, setData]         = useState<AIProvidersResponse | null>(null);
  const [meta, setMeta]         = useState<AIProviderMeta | null>(null);
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState<string | null>(null);
  const [filterCat, setFilterCat] = useState<string>("all");
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  // Modal state
  const [showForm, setShowForm] = useState(false);
  const [editTarget, setEditTarget] = useState<(AIProviderConfig & AIProviderFormPayload) | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AIProviderConfig | null>(null);
  const [saving, setSaving]     = useState(false);
  const [deleting, setDeleting] = useState(false);

  // Per-provider testing state
  const [testingIds, setTestingIds] = useState<Set<string>>(new Set());
  const [testResults, setTestResults] = useState<Record<string, AIProviderTestResult>>({});

  // Toast
  const [toast, setToast] = useState<{ msg: string; ok: boolean } | null>(null);
  const showToast = (msg: string, ok = true) => {
    setToast({ msg, ok });
    setTimeout(() => setToast(null), 3500);
  };

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [d, m] = await Promise.all([adminAPI.getAIProviders(), adminAPI.getAIProviderMeta()]);
      setData(d);
      setMeta(m);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Failed to load";
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  // ── CRUD handlers ─────────────────────────────────────────────────────────
  const handleSave = async (form: AIProviderFormPayload & { id?: string }) => {
    setSaving(true);
    try {
      if (form.id) {
        await adminAPI.updateAIProvider(form.id, form);
        showToast("Provider updated");
      } else {
        await adminAPI.createAIProvider(form);
        showToast("Provider added");
      }
      setShowForm(false);
      setEditTarget(null);
      load();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Save failed";
      showToast(msg, false);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await adminAPI.deleteAIProvider(deleteTarget.id);
      showToast("Provider deleted");
      setDeleteTarget(null);
      load();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Delete failed";
      showToast(msg, false);
    } finally {
      setDeleting(false);
    }
  };

  const handleToggleActive = async (p: AIProviderConfig) => {
    try {
      if (p.is_active) {
        await adminAPI.deactivateAIProvider(p.id);
        showToast(`${p.name} disabled`);
      } else {
        await adminAPI.activateAIProvider(p.id);
        showToast(`${p.name} enabled`);
      }
      load();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Toggle failed";
      showToast(msg, false);
    }
  };

  const handleTest = async (p: AIProviderConfig) => {
    setTestingIds(s => new Set([...s, p.id]));
    try {
      const res = await adminAPI.testAIProvider(p.id);
      setTestResults(r => ({ ...r, [p.id]: res }));
      showToast(`${p.name}: ${res.message}`, res.status === "ok");
      load(); // refresh last_tested_at in card
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Test failed";
      showToast(msg, false);
    } finally {
      setTestingIds(s => { const n = new Set(s); n.delete(p.id); return n; });
    }
  };

  // ── Derived display list ──────────────────────────────────────────────────
  const categories = meta?.categories ?? Object.keys(data?.grouped ?? {});
  const displayCategories = filterCat === "all" ? categories : [filterCat];

  const allCategories = ["all", ...categories];

  const totalActive = data?.providers.filter(p => p.is_active).length ?? 0;
  const totalMissingKey = data?.providers.filter(p => !p.has_key).length ?? 0;

  return (
    <AdminShell>
      {/* Spinner overlay on initial load */}
      {loading && !data && (
        <div style={{ display: "flex", justifyContent: "center", padding: 60 }}>
          <Loader2 size={28} style={{ color: PRIMARY, animation: "spin 1s linear infinite" }} />
        </div>
      )}

      {error && (
        <div style={{
          background: "rgba(248,113,113,0.1)", border: `1px solid rgba(248,113,113,0.3)`,
          borderRadius: 10, padding: 14, color: DANGER, marginBottom: 16,
          display: "flex", alignItems: "center", gap: 8,
        }}>
          <AlertTriangle size={16} />
          {error}
        </div>
      )}

      {data && (
        <>
          {/* ── Top bar ────────────────────────────────────────────────────── */}
          <div style={{
            display: "flex", alignItems: "center", justifyContent: "space-between",
            flexWrap: "wrap" as const, gap: 12, marginBottom: 20,
          }}>
            {/* Stats pills */}
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" as const }}>
              <div style={{
                background: CARD_BG, border: `1px solid ${BORDER_HI}`,
                borderRadius: 8, padding: "6px 14px", display: "flex", alignItems: "center", gap: 6,
              }}>
                <Database size={13} style={{ color: PRIMARY }} />
                <span style={{ color: TEXT, fontSize: 12, fontWeight: 600 }}>{data.total}</span>
                <span style={{ color: MUTED, fontSize: 12 }}>providers</span>
              </div>
              <div style={{
                background: "rgba(52,211,153,0.05)", border: `1px solid rgba(52,211,153,0.2)`,
                borderRadius: 8, padding: "6px 14px", display: "flex", alignItems: "center", gap: 6,
              }}>
                <Zap size={13} style={{ color: SUCCESS }} />
                <span style={{ color: SUCCESS, fontSize: 12, fontWeight: 600 }}>{totalActive}</span>
                <span style={{ color: MUTED, fontSize: 12 }}>active</span>
              </div>
              {totalMissingKey > 0 && (
                <div style={{
                  background: "rgba(245,158,11,0.05)", border: `1px solid rgba(245,158,11,0.2)`,
                  borderRadius: 8, padding: "6px 14px", display: "flex", alignItems: "center", gap: 6,
                }}>
                  <Key size={13} style={{ color: WARN }} />
                  <span style={{ color: WARN, fontSize: 12, fontWeight: 600 }}>{totalMissingKey}</span>
                  <span style={{ color: MUTED, fontSize: 12 }}>missing keys</span>
                </div>
              )}
            </div>

            {/* Actions */}
            <div style={{ display: "flex", gap: 8 }}>
              <button
                onClick={load}
                disabled={loading}
                style={{
                  background: CARD_BG, border: `1px solid ${BORDER_HI}`,
                  borderRadius: 8, padding: "7px 12px", cursor: "pointer",
                  color: MUTED, display: "flex", alignItems: "center", gap: 5,
                }}
              >
                <RefreshCw size={13} style={loading ? { animation: "spin 1s linear infinite" } : {}} />
                Refresh
              </button>
              <button
                onClick={() => { setEditTarget(null); setShowForm(true); }}
                style={{
                  background: PRIMARY, border: "none", borderRadius: 8,
                  padding: "7px 14px", cursor: "pointer", color: "#fff",
                  display: "flex", alignItems: "center", gap: 5, fontWeight: 600, fontSize: 13,
                }}
              >
                <Plus size={13} />
                Add Provider
              </button>
            </div>
          </div>

          {/* ── Category filter tabs ───────────────────────────────────────── */}
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap" as const, marginBottom: 20 }}>
            {allCategories.map(cat => {
              const active = filterCat === cat;
              const cc = catColor(cat);
              return (
                <button
                  key={cat}
                  onClick={() => setFilterCat(cat)}
                  style={{
                    background: active ? `${cc}20` : CARD_BG,
                    border: `1px solid ${active ? `${cc}60` : BORDER}`,
                    borderRadius: 20, padding: "4px 14px", cursor: "pointer",
                    color: active ? cc : MUTED, fontSize: 12, fontWeight: active ? 700 : 400,
                    textTransform: "capitalize" as const,
                    transition: "all .15s",
                  }}
                >
                  {cat === "all" ? "All Categories" : cat}
                  {cat !== "all" && data.grouped[cat] && (
                    <span style={{ marginLeft: 5, opacity: 0.7 }}>
                      ({data.grouped[cat].length})
                    </span>
                  )}
                </button>
              );
            })}
          </div>

          {/* ── Provider groups ─────────────────────────────────────────────── */}
          <div style={{ display: "flex", flexDirection: "column" as const, gap: 24 }}>
            {displayCategories.map(cat => {
              const providers = data.grouped[cat] ?? [];
              if (providers.length === 0) return null;
              const isCollapsed = collapsed[cat];
              const cc = catColor(cat);
              const activeCount = providers.filter(p => p.is_active).length;

              return (
                <div key={cat}>
                  {/* Group header */}
                  <button
                    onClick={() => setCollapsed(c => ({ ...c, [cat]: !c[cat] }))}
                    style={{
                      width: "100%", background: "none", border: "none",
                      cursor: "pointer", display: "flex", alignItems: "center",
                      justifyContent: "space-between", padding: "0 0 10px 0",
                      borderBottom: `1px solid ${BORDER}`, marginBottom: 12,
                    }}
                  >
                    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                      <span style={{
                        width: 10, height: 10, borderRadius: "50%",
                        background: cc, display: "inline-block",
                      }} />
                      <span style={{ color: TEXT, fontWeight: 700, fontSize: 14, textTransform: "capitalize" as const }}>
                        {cat}
                      </span>
                      <span style={{ color: MUTED, fontSize: 12 }}>
                        {activeCount}/{providers.length} active
                      </span>
                    </div>
                    {isCollapsed ? <ChevronDown size={14} style={{ color: MUTED }} /> : <ChevronUp size={14} style={{ color: MUTED }} />}
                  </button>

                  {/* Cards */}
                  {!isCollapsed && (
                    <div style={{ display: "flex", flexDirection: "column" as const, gap: 8 }}>
                      {[...providers].sort((a, b) => a.priority - b.priority).map(p => (
                        <ProviderCard
                          key={p.id}
                          provider={{
                            ...p,
                            last_test_ok: testResults[p.id]?.status === "ok"
                              ? true
                              : testResults[p.id]?.status === "failed"
                                ? false
                                : p.last_test_ok ?? null,
                            last_test_msg: testResults[p.id]?.message ?? p.last_test_msg,
                            last_tested_at: testResults[p.id]?.last_tested_at ?? p.last_tested_at,
                          }}
                          meta={meta}
                          onEdit={p2 => {
                            setEditTarget({ ...p2, api_key: "" });
                            setShowForm(true);
                          }}
                          onDelete={setDeleteTarget}
                          onToggleActive={handleToggleActive}
                          onTest={handleTest}
                          testing={testingIds.has(p.id)}
                        />
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          {/* ── Empty state ────────────────────────────────────────────────── */}
          {data.total === 0 && (
            <div style={{ textAlign: "center" as const, padding: 60, color: MUTED }}>
              <Database size={32} style={{ marginBottom: 12, opacity: 0.4 }} />
              <div style={{ fontSize: 14 }}>No providers configured yet.</div>
              <div style={{ fontSize: 12, marginTop: 6 }}>
                Click <strong>Add Provider</strong> to register your first AI provider.
              </div>
            </div>
          )}
        </>
      )}

      {/* ── Modals ──────────────────────────────────────────────────────────── */}
      {showForm && (
        <ProviderForm
          initial={editTarget ?? EMPTY_FORM}
          meta={meta}
          onSave={handleSave}
          onCancel={() => { setShowForm(false); setEditTarget(null); }}
          saving={saving}
        />
      )}

      {deleteTarget && (
        <DeleteConfirm
          provider={deleteTarget}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
          deleting={deleting}
        />
      )}

      {/* ── Toast ─────────────────────────────────────────────────────────── */}
      {toast && (
        <div style={{
          position: "fixed" as const, bottom: 24, right: 24, zIndex: 2000,
          background: toast.ok ? "rgba(52,211,153,0.15)" : "rgba(248,113,113,0.15)",
          border: `1px solid ${toast.ok ? "rgba(52,211,153,0.4)" : "rgba(248,113,113,0.4)"}`,
          borderRadius: 10, padding: "10px 18px", color: toast.ok ? SUCCESS : DANGER,
          fontSize: 13, maxWidth: 320, boxShadow: "0 4px 20px rgba(0,0,0,0.4)",
        }}>
          {toast.msg}
        </div>
      )}

      {/* ── Global spin keyframe ─────────────────────────────────────────── */}
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </AdminShell>
  );
}
