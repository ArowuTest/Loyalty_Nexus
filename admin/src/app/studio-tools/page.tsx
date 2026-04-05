"use client";

import { useState, useEffect, useCallback, Fragment } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, {
  StudioTool, GenerationError, ToolStat, Generation,
} from "@/lib/api";
import {
  Plus, RefreshCw, X, Edit3, Trash2, AlertTriangle,
  ChevronRight, CheckCircle2, XCircle, BarChart3,
  Zap, Activity, Package, Eye, Loader2,
} from "lucide-react";

// ─── Theme tokens ─────────────────────────────────────────────────────────────
const BG        = "#0d0e1a";
const CARD_BG   = "rgba(95,114,249,0.05)";
const PRIMARY   = "#5f72f9";
const TEXT      = "#e2e8ff";
const MUTED     = "#828cb4";
const BORDER    = "rgba(95,114,249,0.12)";
const BORDER_HI = "rgba(95,114,249,0.25)";
const INPUT_BG  = "rgba(255,255,255,0.04)";

// ─── Helpers ─────────────────────────────────────────────────────────────────
function slugify(str: string) {
  return str.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}

function timeAgo(isoOrNull: string | null | undefined): string {
  if (!isoOrNull) return "—";
  const diff = Math.floor((Date.now() - new Date(isoOrNull).getTime()) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

const CATEGORIES = ["All", "Chat", "Create", "Learn", "Build"] as const;
type Category = (typeof CATEGORIES)[number];

const CAT_COLORS: Record<string, string> = {
  Chat: "#22d3ee", Create: "#a78bfa", Learn: "#34d399", Build: "#f59e0b",
};

// ─── Sub-components ───────────────────────────────────────────────────────────

function Badge({ label, color }: { label: string; color?: string }) {
  const c = color || CAT_COLORS[label] || PRIMARY;
  return (
    <span style={{
      background: `${c}18`, color: c, border: `1px solid ${c}40`,
      borderRadius: 6, fontSize: 10, fontWeight: 700, padding: "2px 7px",
      letterSpacing: "0.04em", textTransform: "uppercase" as const,
    }}>
      {label}
    </span>
  );
}

function StatusPill({ active }: { active: boolean }) {
  return (
    <span style={{
      display: "inline-flex", alignItems: "center", gap: 5,
      background: active ? "rgba(52,211,153,0.12)" : "rgba(239,68,68,0.1)",
      color: active ? "#34d399" : "#f87171",
      border: `1px solid ${active ? "rgba(52,211,153,0.3)" : "rgba(239,68,68,0.25)"}`,
      borderRadius: 20, fontSize: 11, fontWeight: 600, padding: "3px 9px",
    }}>
      <span style={{
        width: 6, height: 6, borderRadius: "50%",
        background: active ? "#34d399" : "#f87171",
        display: "inline-block",
      }} />
      {active ? "Active" : "Disabled"}
    </span>
  );
}

// ─── Shared input style ───────────────────────────────────────────────────────
const inp = (extra?: React.CSSProperties): React.CSSProperties => ({
  background: INPUT_BG, border: `1px solid ${BORDER_HI}`, borderRadius: 8,
  color: TEXT, fontSize: 13, padding: "7px 10px", width: "100%", outline: "none",
  ...extra,
});

// ─── Stat card ────────────────────────────────────────────────────────────────
function StatCard({ icon, label, value, color }: {
  icon: React.ReactNode; label: string; value: number | string; color: string;
}) {
  return (
    <div style={{
      background: CARD_BG, border: `1px solid ${BORDER}`, borderRadius: 14,
      padding: "16px 20px", display: "flex", alignItems: "center", gap: 14,
    }}>
      <div style={{
        width: 40, height: 40, borderRadius: 10,
        background: `${color}18`, display: "flex", alignItems: "center", justifyContent: "center",
        color,
      }}>
        {icon}
      </div>
      <div>
        <div style={{ color: MUTED, fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em" }}>{label}</div>
        <div style={{ color: TEXT, fontSize: 22, fontWeight: 700, lineHeight: 1.2 }}>{value}</div>
      </div>
    </div>
  );
}

// ─── Create Tool Modal ────────────────────────────────────────────────────────
const UI_TEMPLATES = [
  { value: "knowledge-doc",    label: "📄 Knowledge Doc" },
  { value: "music-composer",   label: "🎵 Music Composer" },
  { value: "image-creator",    label: "🖼️ Image Creator" },
  { value: "image-editor",     label: "✏️ Image Editor" },
  { value: "image-compose",    label: "🎨 Image Compose" },
  { value: "video-creator",    label: "🎬 Video Creator" },
  { value: "video-animator",   label: "🎞️ Video Animator" },
  { value: "video-editor",     label: "🎥 Video Editor" },
  { value: "video-extender",   label: "⏩ Video Extender" },
  { value: "video-multi-scene",label: "🎭 Video Multi-Scene" },
  { value: "voice-studio",     label: "🎙️ Voice Studio" },
  { value: "transcribe",       label: "📝 Transcribe" },
  { value: "vision-ask",       label: "👁️ Vision Ask" },
  { value: "chat",             label: "💬 Chat" },
] as const;

interface CreateForm {
  name: string; slug: string; description: string; category: string;
  point_cost: string; provider: string; provider_tool: string;
  icon: string; sort_order: string;
  entry_point_cost: string; refund_window_mins: string; refund_pct: string; is_free: boolean;
  ui_template: string;
}

const EMPTY_FORM: CreateForm = {
  name: "", slug: "", description: "", category: "Chat",
  point_cost: "10", provider: "", provider_tool: "", icon: "✨", sort_order: "100",
  entry_point_cost: "0", refund_window_mins: "5", refund_pct: "100", is_free: false,
  ui_template: "knowledge-doc",
};

function CreateModal({ onClose, onCreated }: {
  onClose: () => void;
  onCreated: (t: StudioTool) => void;
}) {
  const [form, setForm] = useState<CreateForm>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const set = (k: keyof CreateForm, v: string) => {
    setForm(prev => {
      const next = { ...prev, [k]: v };
      if (k === "name") next.slug = slugify(v);
      return next;
    });
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const cost = parseInt(form.point_cost, 10);
    if (isNaN(cost) || cost < 0) { setErr("Point cost must be ≥ 0"); return; }
    setSaving(true); setErr(null);
    try {
      const tool = await adminAPI.createStudioTool({
        name: form.name, slug: form.slug, description: form.description,
        category: form.category, point_cost: cost, provider: form.provider,
        provider_tool: form.provider_tool, icon: form.icon,
        sort_order: parseInt(form.sort_order, 10) || 100,
        is_active: true,
        entry_point_cost: parseInt(form.entry_point_cost, 10) || 0,
        refund_window_mins: parseInt(form.refund_window_mins, 10) || 0,
        refund_pct: Math.min(100, Math.max(0, parseInt(form.refund_pct, 10) || 100)),
        is_free: form.is_free,
        ui_template: form.ui_template || "knowledge-doc",
      });
      onCreated(tool);
      onClose();
    } catch (e: unknown) {
      setErr((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div style={{
      position: "fixed", inset: 0, zIndex: 1000,
      background: "rgba(0,0,0,0.7)", backdropFilter: "blur(6px)",
      display: "flex", alignItems: "center", justifyContent: "center", padding: 20,
    }} onClick={onClose}>
      <div style={{
        background: "#131525", border: `1px solid ${BORDER_HI}`, borderRadius: 18,
        width: "100%", maxWidth: 540, maxHeight: "90vh", overflowY: "auto",
        padding: 28,
      }} onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
          <h2 style={{ color: TEXT, fontSize: 18, fontWeight: 700, margin: 0 }}>
            ✨ New Studio Tool
          </h2>
          <button onClick={onClose} style={{ background: "none", border: "none", cursor: "pointer", color: MUTED }}>
            <X size={18} />
          </button>
        </div>

        <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          {/* Row: name + icon */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 80px", gap: 10 }}>
            <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
              <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>TOOL NAME *</span>
              <input required style={inp()} value={form.name}
                onChange={e => set("name", e.target.value)} placeholder="e.g. Essay Writer" />
            </label>
            <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
              <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>ICON</span>
              <input style={inp({ textAlign: "center", fontSize: 20 })} value={form.icon}
                onChange={e => set("icon", e.target.value)} placeholder="✨" />
            </label>
          </div>

          {/* Slug */}
          <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
            <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>SLUG *</span>
            <input required style={inp({ fontFamily: "monospace", color: PRIMARY })} value={form.slug}
              onChange={e => set("slug", e.target.value)} placeholder="auto-generated" />
          </label>

          {/* Description */}
          <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
            <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>DESCRIPTION</span>
            <textarea rows={2} style={{ ...inp(), resize: "vertical" as const }} value={form.description}
              onChange={e => set("description", e.target.value)} placeholder="What does this tool do?" />
          </label>

          {/* Row: category + point_cost */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
            <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
              <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>CATEGORY *</span>
              <select required style={inp()} value={form.category}
                onChange={e => set("category", e.target.value)}>
                {["Chat", "Create", "Learn", "Build"].map(c => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
            </label>
            <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
              <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>POINT COST</span>
              <input type="number" min={0} style={inp()} value={form.point_cost}
                onChange={e => set("point_cost", e.target.value)} />
            </label>
          </div>

          {/* UI Template */}
          <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
            <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>UI TEMPLATE *</span>
            <select required style={inp()} value={form.ui_template}
              onChange={e => set("ui_template", e.target.value)}>
              {UI_TEMPLATES.map(t => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
            <span style={{ color: MUTED, fontSize: 10 }}>Controls which input form users see for this tool.</span>
          </label>

          {/* Row: provider + provider_tool */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
            <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
              <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>PROVIDER</span>
              <input style={inp({ fontFamily: "monospace" })} value={form.provider}
                onChange={e => set("provider", e.target.value)} placeholder="GROQ" />
            </label>
            <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
              <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>PROVIDER TOOL</span>
              <input style={inp({ fontFamily: "monospace" })} value={form.provider_tool}
                onChange={e => set("provider_tool", e.target.value)} placeholder="llama-4-scout" />
            </label>
          </div>

          {/* Sort order */}
          <label style={{ display: "flex", flexDirection: "column", gap: 5 }}>
            <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>SORT ORDER</span>
            <input type="number" min={0} style={inp()} value={form.sort_order}
              onChange={e => set("sort_order", e.target.value)} />
          </label>

          {/* Session token fields */}
          <div style={{ borderTop: `1px solid ${BORDER}`, paddingTop: 14 }}>
            <p style={{ color: MUTED, fontSize: 11, fontWeight: 700, marginBottom: 10, textTransform: "uppercase", letterSpacing: 1 }}>
              Session Token Settings
            </p>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 10, marginBottom: 10 }}>
              <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>ENTRY THRESHOLD (pts)</span>
                <input type="number" min={0} style={inp()} value={form.entry_point_cost}
                  onChange={e => set("entry_point_cost", e.target.value)} />
                <span style={{ color: MUTED, fontSize: 10 }}>Min balance to open. 0 = no gate.</span>
              </label>
              <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>REFUND WINDOW (mins)</span>
                <input type="number" min={0} style={inp()} value={form.refund_window_mins}
                  onChange={e => set("refund_window_mins", e.target.value)} />
                <span style={{ color: MUTED, fontSize: 10 }}>0 = no disputes allowed.</span>
              </label>
              <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                <span style={{ color: MUTED, fontSize: 11, fontWeight: 600 }}>REFUND % ON DISPUTE</span>
                <input type="number" min={0} max={100} style={inp()} value={form.refund_pct}
                  onChange={e => set("refund_pct", e.target.value)} />
                <span style={{ color: MUTED, fontSize: 10 }}>% of pts returned (0-100).</span>
              </label>
            </div>
            <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer" }}>
              <input type="checkbox" checked={form.is_free}
                onChange={e => setForm(prev => ({ ...prev, is_free: e.target.checked }))}
                style={{ width: 15, height: 15, accentColor: PRIMARY }} />
              <span style={{ color: TEXT, fontSize: 13 }}>Free Tool (bypass all point checks)</span>
            </label>
          </div>

          {err && (
            <div style={{
              background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.3)",
              borderRadius: 8, padding: "10px 14px", color: "#f87171", fontSize: 13,
            }}>{err}</div>
          )}

          {/* Actions */}
          <div style={{ display: "flex", gap: 10, justifyContent: "flex-end", marginTop: 4 }}>
            <button type="button" onClick={onClose} style={{
              background: "rgba(255,255,255,0.05)", border: `1px solid ${BORDER}`,
              color: MUTED, borderRadius: 10, padding: "9px 18px", cursor: "pointer", fontSize: 13,
            }}>Cancel</button>
            <button type="submit" disabled={saving} style={{
              background: PRIMARY, border: "none", color: "#fff",
              borderRadius: 10, padding: "9px 22px", cursor: saving ? "not-allowed" : "pointer",
              fontSize: 13, fontWeight: 600, opacity: saving ? 0.7 : 1,
              display: "flex", alignItems: "center", gap: 6,
            }}>
              {saving ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}
              {saving ? "Creating…" : "Create Tool"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Errors Side Panel ────────────────────────────────────────────────────────
function ErrorsPanel({ tool, onClose }: { tool: StudioTool; onClose: () => void }) {
  const [errors, setErrors] = useState<GenerationError[]>([]);
  const [count, setCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    adminAPI.getStudioToolErrors(tool.id)
      .then(r => { setErrors(r.errors ?? []); setCount(r.count ?? 0); })
      .catch(e => setErr((e as Error).message))
      .finally(() => setLoading(false));
  }, [tool.id]);

  return (
    <div style={{
      position: "fixed", inset: 0, zIndex: 1000,
      background: "rgba(0,0,0,0.65)", backdropFilter: "blur(4px)",
      display: "flex", justifyContent: "flex-end",
    }} onClick={onClose}>
      <div style={{
        background: "#131525", borderLeft: `1px solid ${BORDER_HI}`,
        width: "100%", maxWidth: 500, height: "100%",
        display: "flex", flexDirection: "column", overflowY: "hidden",
      }} onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div style={{
          padding: "20px 22px", borderBottom: `1px solid ${BORDER}`,
          display: "flex", justifyContent: "space-between", alignItems: "center",
        }}>
          <div>
            <h3 style={{ color: TEXT, fontSize: 16, fontWeight: 700, margin: 0 }}>
              {tool.icon || "🔧"} {tool.name} — Errors
            </h3>
            <p style={{ color: MUTED, fontSize: 12, margin: "4px 0 0" }}>
              {count} total failures recorded
            </p>
          </div>
          <button onClick={onClose} style={{ background: "none", border: "none", cursor: "pointer", color: MUTED }}>
            <X size={18} />
          </button>
        </div>

        {/* Body */}
        <div style={{ flex: 1, overflowY: "auto", padding: "14px 16px", display: "flex", flexDirection: "column", gap: 10 }}>
          {loading && (
            <div style={{ textAlign: "center", padding: 40, color: MUTED }}>
              <Loader2 size={24} style={{ margin: "0 auto 8px" }} />
              <p>Loading errors…</p>
            </div>
          )}
          {err && (
            <div style={{ background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.3)", borderRadius: 10, padding: "14px", color: "#f87171" }}>
              {err}
            </div>
          )}
          {!loading && !err && errors.length === 0 && (
            <div style={{ textAlign: "center", padding: 50, color: MUTED }}>
              <CheckCircle2 size={32} style={{ color: "#34d399", marginBottom: 10 }} />
              <p>No errors recorded for this tool 🎉</p>
            </div>
          )}
          {errors.map(e => (
            <div key={e.id} style={{
              background: "rgba(239,68,68,0.05)", border: "1px solid rgba(239,68,68,0.15)",
              borderRadius: 10, padding: "12px 14px",
            }}>
              <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 6 }}>
                <span style={{
                  background: "rgba(239,68,68,0.12)", color: "#f87171",
                  fontSize: 10, fontWeight: 700, padding: "2px 7px", borderRadius: 5,
                }}>
                  {e.provider || "Unknown"}
                </span>
                <span style={{ color: MUTED, fontSize: 11 }}>{timeAgo(e.created_at)}</span>
              </div>
              <p style={{ color: "#f87171", fontSize: 12, fontFamily: "monospace", margin: "0 0 6px", lineHeight: 1.5 }}>
                {e.error_message}
              </p>
              <p style={{
                color: MUTED, fontSize: 11, margin: 0,
                overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
              }}>
                Prompt: <em>{e.prompt?.slice(0, 80)}{(e.prompt?.length ?? 0) > 80 ? "…" : ""}</em>
              </p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── Stats Panel ──────────────────────────────────────────────────────────────
function StatsPanel({ onClose }: { onClose: () => void }) {
  const [stats, setStats] = useState<ToolStat[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    adminAPI.getStudioToolStats()
      .then(r => setStats(r.stats ?? []))
      .catch(e => setErr((e as Error).message))
      .finally(() => setLoading(false));
  }, []);

  const total30 = stats.reduce((s, t) => s + t.total, 0);
  const totalPts = stats.reduce((s, t) => s + t.points_consumed, 0);
  const avgSuccess = stats.length
    ? Math.round(stats.reduce((s, t) => s + (t.total > 0 ? (t.completed / t.total) * 100 : 0), 0) / stats.length)
    : 0;

  return (
    <div style={{
      position: "fixed", inset: 0, zIndex: 1000,
      background: "rgba(0,0,0,0.7)", backdropFilter: "blur(6px)",
      display: "flex", alignItems: "center", justifyContent: "center", padding: 20,
    }} onClick={onClose}>
      <div style={{
        background: "#131525", border: `1px solid ${BORDER_HI}`,
        borderRadius: 18, width: "100%", maxWidth: 680, maxHeight: "80vh",
        overflowY: "auto", padding: 28,
      }} onClick={e => e.stopPropagation()}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 22 }}>
          <h2 style={{ color: TEXT, fontSize: 18, fontWeight: 700, margin: 0 }}>📊 30-Day Tool Stats</h2>
          <button onClick={onClose} style={{ background: "none", border: "none", cursor: "pointer", color: MUTED }}>
            <X size={18} />
          </button>
        </div>

        {loading && <div style={{ textAlign: "center", padding: 40, color: MUTED }}><Loader2 size={24} /></div>}
        {err && <div style={{ color: "#f87171", padding: 20 }}>{err}</div>}

        {!loading && !err && (
          <>
            {/* Summary cards */}
            <div style={{ display: "grid", gridTemplateColumns: "repeat(3,1fr)", gap: 12, marginBottom: 22 }}>
              <div style={{ background: CARD_BG, border: `1px solid ${BORDER}`, borderRadius: 12, padding: "14px 16px" }}>
                <div style={{ color: MUTED, fontSize: 10, fontWeight: 700, textTransform: "uppercase" }}>Total Generations</div>
                <div style={{ color: TEXT, fontSize: 24, fontWeight: 700 }}>{total30.toLocaleString()}</div>
              </div>
              <div style={{ background: CARD_BG, border: `1px solid ${BORDER}`, borderRadius: 12, padding: "14px 16px" }}>
                <div style={{ color: MUTED, fontSize: 10, fontWeight: 700, textTransform: "uppercase" }}>Avg Success Rate</div>
                <div style={{ color: "#34d399", fontSize: 24, fontWeight: 700 }}>{avgSuccess}%</div>
              </div>
              <div style={{ background: CARD_BG, border: `1px solid ${BORDER}`, borderRadius: 12, padding: "14px 16px" }}>
                <div style={{ color: MUTED, fontSize: 10, fontWeight: 700, textTransform: "uppercase" }}>Points Consumed</div>
                <div style={{ color: PRIMARY, fontSize: 24, fontWeight: 700 }}>{totalPts.toLocaleString()}</div>
              </div>
            </div>

            {/* Per-tool table */}
            <div style={{ background: CARD_BG, border: `1px solid ${BORDER}`, borderRadius: 12, overflow: "hidden" }}>
              <table style={{ width: "100%", borderCollapse: "collapse" }}>
                <thead>
                  <tr style={{ borderBottom: `1px solid ${BORDER}` }}>
                    {["Tool", "Total", "Completed", "Failed", "Success %", "Points Used"].map(h => (
                      <th key={h} style={{ padding: "10px 14px", textAlign: "left", color: MUTED, fontSize: 10, fontWeight: 700, textTransform: "uppercase" }}>{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {stats.map((s, i) => {
                    const pct = s.total > 0 ? Math.round((s.completed / s.total) * 100) : 0;
                    return (
                      <tr key={s.tool_id} style={{ borderBottom: i < stats.length - 1 ? `1px solid ${BORDER}` : "none" }}>
                        <td style={{ padding: "10px 14px", color: TEXT, fontFamily: "monospace", fontSize: 12 }}>{s.tool_slug}</td>
                        <td style={{ padding: "10px 14px", color: TEXT, fontSize: 13, fontWeight: 600 }}>{s.total.toLocaleString()}</td>
                        <td style={{ padding: "10px 14px", color: "#34d399", fontSize: 13 }}>{s.completed.toLocaleString()}</td>
                        <td style={{ padding: "10px 14px", color: "#f87171", fontSize: 13 }}>{s.failed.toLocaleString()}</td>
                        <td style={{ padding: "10px 14px" }}>
                          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                            <div style={{
                              height: 4, width: 50, borderRadius: 99,
                              background: "rgba(255,255,255,0.08)", overflow: "hidden",
                            }}>
                              <div style={{ height: "100%", width: `${pct}%`, background: pct > 90 ? "#34d399" : pct > 70 ? "#f59e0b" : "#f87171", borderRadius: 99 }} />
                            </div>
                            <span style={{ color: pct > 90 ? "#34d399" : pct > 70 ? "#f59e0b" : "#f87171", fontSize: 12, fontWeight: 600 }}>{pct}%</span>
                          </div>
                        </td>
                        <td style={{ padding: "10px 14px", color: PRIMARY, fontSize: 13 }}>{s.points_consumed.toLocaleString()}</td>
                      </tr>
                    );
                  })}
                  {stats.length === 0 && (
                    <tr>
                      <td colSpan={6} style={{ textAlign: "center", padding: 30, color: MUTED }}>No stats available</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

// ─── Inline Edit State ────────────────────────────────────────────────────────
interface EditState {
  point_cost: string;
  is_active: boolean;
  provider: string;
  description: string;
  icon: string;
  entry_point_cost: string;
  refund_window_mins: string;
  refund_pct: string;
  is_free: boolean;
  ui_template: string;
}

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function StudioToolsPage() {
  const [tools, setTools]           = useState<StudioTool[]>([]);
  const [loading, setLoading]       = useState(true);
  const [error, setError]           = useState<string | null>(null);
  const [activeTab, setActiveTab]   = useState<Category>("All");

  // Inline edit
  const [editId, setEditId]         = useState<string | null>(null);
  const [editState, setEditState]   = useState<EditState | null>(null);
  const [saving, setSaving]         = useState(false);
  const [savedId, setSavedId]       = useState<string | null>(null);

  // Modals
  const [showCreate, setShowCreate]   = useState(false);
  const [showStats, setShowStats]     = useState(false);
  const [errPanelTool, setErrPanelTool] = useState<StudioTool | null>(null);

  // Disable confirm
  const [disableTarget, setDisableTarget] = useState<StudioTool | null>(null);
  const [disabling, setDisabling]         = useState(false);

  const load = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const r = await adminAPI.getStudioTools();
      setTools(r.tools ?? []);
    } catch (e: unknown) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  // ── Computed stats ─────────────────────────────────────────────────────────
  const totalTools    = tools.length;
  const activeTools   = tools.filter(t => t.is_active).length;
  const disabledTools = tools.filter(t => !t.is_active).length;
  const categories    = [...new Set(tools.map(t => t.category))].length;

  const filtered = activeTab === "All" ? tools : tools.filter(t => t.category === activeTab);

  // ── Inline edit helpers ────────────────────────────────────────────────────
  const startEdit = (t: StudioTool) => {
    setEditId(t.id);
    setEditState({
      point_cost: String(t.point_cost),
      is_active: t.is_active,
      provider: t.provider || "",
      description: t.description || "",
      icon: t.icon || "",
      entry_point_cost: String(t.entry_point_cost ?? 0),
      refund_window_mins: String(t.refund_window_mins ?? 0),
      refund_pct: String(t.refund_pct ?? 100),
      is_free: t.is_free ?? false,
      ui_template: t.ui_template || "knowledge-doc",
    });
    setError(null);
  };

  const cancelEdit = () => { setEditId(null); setEditState(null); };

  const saveEdit = async (tool: StudioTool) => {
    if (!editState) return;
    const cost = parseInt(editState.point_cost, 10);
    if (isNaN(cost) || cost < 0) { setError("Point cost must be ≥ 0"); return; }
    setSaving(true);
    try {
      await adminAPI.updateStudioTool(tool.id, {
        point_cost: cost,
        is_active: editState.is_active,
        provider: editState.provider,
        description: editState.description,
        icon: editState.icon,
        entry_point_cost: parseInt(editState.entry_point_cost, 10) || 0,
        refund_window_mins: parseInt(editState.refund_window_mins, 10) || 0,
        refund_pct: Math.min(100, Math.max(0, parseInt(editState.refund_pct, 10) || 100)),
        is_free: editState.is_free,
        ui_template: editState.ui_template || "knowledge-doc",
      });
      setSavedId(tool.id);
      setEditId(null); setEditState(null);
      setTimeout(() => setSavedId(null), 2000);
      await load();
    } catch (e: unknown) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  // ── Disable helper ─────────────────────────────────────────────────────────
  const confirmDisable = async () => {
    if (!disableTarget) return;
    setDisabling(true);
    try {
      await adminAPI.disableStudioTool(disableTarget.id);
      setDisableTarget(null);
      await load();
    } catch (e: unknown) {
      setError((e as Error).message);
    } finally {
      setDisabling(false);
    }
  };

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <AdminShell>
      <div style={{ background: BG, minHeight: "100vh", padding: "28px 24px", fontFamily: "system-ui, sans-serif" }}>

        {/* ── Header ── */}
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 24, flexWrap: "wrap", gap: 12 }}>
          <div>
            <h1 style={{ color: TEXT, fontSize: 26, fontWeight: 800, margin: 0 }}>🧠 AI Studio Tools</h1>
            <p style={{ color: MUTED, fontSize: 13, margin: "4px 0 0" }}>
              {totalTools} tools registered · Manage, monitor and configure all AI studio tools
            </p>
          </div>
          <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
            <button onClick={() => setShowStats(true)} style={{
              background: "rgba(95,114,249,0.12)", border: `1px solid ${BORDER_HI}`,
              color: PRIMARY, borderRadius: 10, padding: "8px 14px",
              cursor: "pointer", fontSize: 13, fontWeight: 600,
              display: "flex", alignItems: "center", gap: 6,
            }}>
              <BarChart3 size={14} /> 30-Day Stats
            </button>
            <button onClick={load} disabled={loading} style={{
              background: "rgba(255,255,255,0.05)", border: `1px solid ${BORDER}`,
              color: MUTED, borderRadius: 10, padding: "8px 14px",
              cursor: loading ? "not-allowed" : "pointer", fontSize: 13,
              display: "flex", alignItems: "center", gap: 6,
            }}>
              <RefreshCw size={13} style={{ animation: loading ? "spin 1s linear infinite" : "none" }} />
              Refresh
            </button>
            <button onClick={() => setShowCreate(true)} style={{
              background: PRIMARY, border: "none", color: "#fff",
              borderRadius: 10, padding: "8px 18px", cursor: "pointer",
              fontSize: 13, fontWeight: 700,
              display: "flex", alignItems: "center", gap: 6,
            }}>
              <Plus size={14} /> Add Tool
            </button>
          </div>
        </div>

        {/* ── Stats Bar ── */}
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit,minmax(160px,1fr))", gap: 12, marginBottom: 24 }}>
          <StatCard icon={<Package size={18} />}      label="Total Tools"     value={totalTools}    color={PRIMARY} />
          <StatCard icon={<CheckCircle2 size={18} />} label="Active Tools"    value={activeTools}   color="#34d399" />
          <StatCard icon={<XCircle size={18} />}      label="Disabled Tools"  value={disabledTools} color="#f87171" />
          <StatCard icon={<Activity size={18} />}     label="Categories"      value={categories}    color="#f59e0b" />
        </div>

        {/* ── Error Banner ── */}
        {error && (
          <div style={{
            background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.3)",
            borderRadius: 10, padding: "12px 16px", color: "#f87171",
            display: "flex", alignItems: "center", gap: 10, marginBottom: 16,
          }}>
            <AlertTriangle size={15} />
            <span style={{ flex: 1, fontSize: 13 }}>{error}</span>
            <button onClick={() => setError(null)} style={{ background: "none", border: "none", color: "#f87171", cursor: "pointer" }}><X size={14} /></button>
          </div>
        )}

        {/* ── Category Tabs ── */}
        <div style={{
          display: "flex", gap: 6, marginBottom: 18,
          background: CARD_BG, border: `1px solid ${BORDER}`,
          borderRadius: 12, padding: 5, width: "fit-content",
        }}>
          {CATEGORIES.map(cat => {
            const active = activeTab === cat;
            const count  = cat === "All" ? tools.length : tools.filter(t => t.category === cat).length;
            return (
              <button key={cat} onClick={() => setActiveTab(cat)} style={{
                background: active ? PRIMARY : "transparent",
                border: "none", color: active ? "#fff" : MUTED,
                borderRadius: 8, padding: "6px 14px", cursor: "pointer",
                fontSize: 12, fontWeight: active ? 700 : 500,
                transition: "all 0.15s",
                display: "flex", alignItems: "center", gap: 6,
              }}>
                {cat}
                <span style={{
                  background: active ? "rgba(255,255,255,0.25)" : "rgba(255,255,255,0.06)",
                  borderRadius: 99, fontSize: 10, fontWeight: 700,
                  padding: "1px 6px", color: active ? "#fff" : MUTED,
                }}>{count}</span>
              </button>
            );
          })}
        </div>

        {/* ── Tools Table ── */}
        {loading ? (
          <div style={{ textAlign: "center", padding: 80, color: MUTED }}>
            <Loader2 size={32} style={{ margin: "0 auto 12px", animation: "spin 1s linear infinite" }} />
            <p>Loading tools…</p>
          </div>
        ) : (
          <div style={{
            background: CARD_BG, border: `1px solid ${BORDER}`,
            borderRadius: 16, overflow: "hidden",
          }}>
            <table style={{ width: "100%", borderCollapse: "collapse" }}>
              <thead>
                <tr style={{ borderBottom: `1px solid ${BORDER}` }}>
                  {["Tool", "Category", "UI Template", "Provider / Model", "Point Cost", "Entry Gate", "Status", "Gen Today", "Success", "Actions"].map(h => (
                    <th key={h} style={{
                      padding: "12px 16px", textAlign: "left",
                      color: MUTED, fontSize: 10, fontWeight: 700,
                      textTransform: "uppercase", letterSpacing: "0.06em",
                      whiteSpace: "nowrap",
                    }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={8} style={{ textAlign: "center", padding: 48, color: MUTED, fontSize: 14 }}>
                      No tools in this category
                    </td>
                  </tr>
                )}
                {filtered.map((t, i) => {
                  const isEditing = editId === t.id;
                  const wasSaved  = savedId === t.id;
                  const isLast    = i === filtered.length - 1;
                  const successPct = t.success_rate != null ? `${Math.round(t.success_rate)}%` : "—";

                  return (
                    <Fragment key={t.id}>
                      <tr style={{
                        borderBottom: isEditing || !isLast ? `1px solid ${BORDER}` : "none",
                        background: wasSaved ? "rgba(52,211,153,0.05)" : isEditing ? "rgba(95,114,249,0.06)" : "transparent",
                        transition: "background 0.3s",
                      }}>
                        {/* Tool name + slug */}
                        <td style={{ padding: "13px 16px", minWidth: 160 }}>
                          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                            <span style={{ fontSize: 18 }}>{t.icon || "🔧"}</span>
                            <div>
                              <div style={{ color: TEXT, fontWeight: 600, fontSize: 13 }}>{t.name}</div>
                              <div style={{ color: "#4d567a", fontSize: 11, fontFamily: "monospace" }}>
                                {t.slug ?? t.id?.slice(0, 8)}
                              </div>
                            </div>
                          </div>
                        </td>

                        {/* Category */}
                        <td style={{ padding: "13px 16px" }}>
                          <Badge label={t.category} />
                        </td>

                        {/* UI Template */}
                        <td style={{ padding: "13px 16px" }}>
                          <span style={{
                            fontSize: 11, fontFamily: "monospace",
                            color: "#a78bfa",
                            background: "rgba(167,139,250,0.1)",
                            border: "1px solid rgba(167,139,250,0.25)",
                            borderRadius: 6, padding: "2px 7px",
                          }}>
                            {t.ui_template || "knowledge-doc"}
                          </span>
                        </td>

                        {/* Provider / Model */}
                        <td style={{ padding: "13px 16px" }}>
                          {isEditing && editState ? (
                            <input
                              style={inp({ width: 120, padding: "5px 8px" })}
                              value={editState.provider}
                              onChange={e => setEditState(prev => prev ? { ...prev, provider: e.target.value } : prev)}
                            />
                          ) : (
                            <span style={{ color: PRIMARY, fontSize: 12, fontFamily: "monospace" }}>{t.provider || "—"}</span>
                          )}
                        </td>

                        {/* Point Cost */}
                        <td style={{ padding: "13px 16px" }}>
                          {isEditing && editState ? (
                            <input
                              type="number" min={0} style={inp({ width: 70, padding: "5px 8px" })}
                              value={editState.point_cost}
                              onChange={e => setEditState(prev => prev ? { ...prev, point_cost: e.target.value } : prev)}
                            />
                          ) : (
                            <div>
                              <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
                                <Zap size={11} style={{ color: "#f59e0b" }} />
                                <span style={{ color: TEXT, fontWeight: 600, fontSize: 13 }}>{t.point_cost} pts/gen</span>
                              </div>
                              {(t.entry_point_cost ?? 0) > 0 && (
                                <div style={{ color: MUTED, fontSize: 10, marginTop: 2 }}>Entry: {t.entry_point_cost} pts</div>
                              )}
                            </div>
                          )}
                        </td>

                        {/* Entry Gate */}
                        <td style={{ padding: "13px 16px" }}>
                          {isEditing && editState ? (
                            <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                              <input
                                type="number" min={0} placeholder="Entry pts"
                                style={inp({ width: 70, padding: "5px 8px" })}
                                value={editState.entry_point_cost}
                                onChange={e => setEditState(prev => prev ? { ...prev, entry_point_cost: e.target.value } : prev)}
                              />
                              <label style={{ display: "flex", alignItems: "center", gap: 4, cursor: "pointer" }}>
                                <input type="checkbox" checked={editState.is_free}
                                  onChange={e => setEditState(prev => prev ? { ...prev, is_free: e.target.checked } : prev)}
                                  style={{ accentColor: PRIMARY }} />
                                <span style={{ color: MUTED, fontSize: 10 }}>Free</span>
                              </label>
                            </div>
                          ) : (
                            <span style={{
                              fontSize: 11, fontWeight: 600,
                              color: t.is_free ? "#34d399" : (t.entry_point_cost ?? 0) > 0 ? "#f59e0b" : MUTED,
                              background: t.is_free ? "rgba(52,211,153,0.1)" : (t.entry_point_cost ?? 0) > 0 ? "rgba(245,158,11,0.1)" : "rgba(255,255,255,0.04)",
                              border: `1px solid ${t.is_free ? "rgba(52,211,153,0.25)" : (t.entry_point_cost ?? 0) > 0 ? "rgba(245,158,11,0.25)" : "rgba(255,255,255,0.08)"}`,
                              borderRadius: 20, padding: "3px 10px",
                            }}>
                              {t.is_free ? "FREE" : (t.entry_point_cost ?? 0) > 0 ? `${t.entry_point_cost} pts` : "None"}
                            </span>
                          )}
                        </td>

                        {/* Status */}
                        <td style={{ padding: "13px 16px" }}>
                          {isEditing && editState ? (
                            <button
                              onClick={() => setEditState(prev => prev ? { ...prev, is_active: !prev.is_active } : prev)}
                              style={{
                                background: editState.is_active ? "rgba(52,211,153,0.12)" : "rgba(239,68,68,0.1)",
                                border: `1px solid ${editState.is_active ? "rgba(52,211,153,0.3)" : "rgba(239,68,68,0.25)"}`,
                                color: editState.is_active ? "#34d399" : "#f87171",
                                borderRadius: 20, fontSize: 11, fontWeight: 600,
                                padding: "3px 10px", cursor: "pointer",
                                display: "flex", alignItems: "center", gap: 5,
                              }}
                            >
                              <span style={{
                                width: 7, height: 7, borderRadius: "50%",
                                background: editState.is_active ? "#34d399" : "#f87171",
                                display: "inline-block",
                              }} />
                              {editState.is_active ? "Active" : "Disabled"}
                            </button>
                          ) : (
                            <StatusPill active={t.is_active} />
                          )}
                        </td>

                        {/* Generated today */}
                        <td style={{ padding: "13px 16px", color: TEXT, fontSize: 13, fontWeight: 600 }}>
                          {t.generated_today?.toLocaleString() ?? "—"}
                        </td>

                        {/* Success rate */}
                        <td style={{ padding: "13px 16px" }}>
                          <span style={{
                            color: t.success_rate == null ? MUTED
                              : t.success_rate >= 90 ? "#34d399"
                              : t.success_rate >= 70 ? "#f59e0b"
                              : "#f87171",
                            fontSize: 13, fontWeight: 600,
                          }}>
                            {successPct}
                          </span>
                        </td>

                        {/* Actions */}
                        <td style={{ padding: "13px 16px" }}>
                          <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                            {isEditing ? (
                              <>
                                <button
                                  onClick={() => saveEdit(t)}
                                  disabled={saving}
                                  style={{
                                    background: "#34d399", border: "none", color: "#0d0e1a",
                                    borderRadius: 7, padding: "5px 12px", cursor: saving ? "not-allowed" : "pointer",
                                    fontSize: 11, fontWeight: 700, opacity: saving ? 0.7 : 1,
                                  }}
                                >
                                  {saving ? "Saving…" : "Save"}
                                </button>
                                <button onClick={cancelEdit} style={{
                                  background: "rgba(255,255,255,0.05)", border: `1px solid ${BORDER}`,
                                  color: MUTED, borderRadius: 7, padding: "5px 10px",
                                  cursor: "pointer", fontSize: 11,
                                }}>Cancel</button>
                              </>
                            ) : (
                              <>
                                <button
                                  onClick={() => startEdit(t)}
                                  title="Edit"
                                  style={{
                                    background: "rgba(95,114,249,0.1)", border: `1px solid ${BORDER_HI}`,
                                    color: PRIMARY, borderRadius: 7, padding: "5px 8px",
                                    cursor: "pointer", display: "flex", alignItems: "center",
                                  }}
                                >
                                  <Edit3 size={13} />
                                </button>
                                <button
                                  onClick={() => setErrPanelTool(t)}
                                  title="View Errors"
                                  style={{
                                    background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.2)",
                                    color: "#f87171", borderRadius: 7, padding: "5px 8px",
                                    cursor: "pointer", display: "flex", alignItems: "center",
                                  }}
                                >
                                  <Eye size={13} />
                                </button>
                                {t.is_active && (
                                  <button
                                    onClick={() => setDisableTarget(t)}
                                    title="Disable tool"
                                    style={{
                                      background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.2)",
                                      color: "#f87171", borderRadius: 7, padding: "5px 8px",
                                      cursor: "pointer", display: "flex", alignItems: "center",
                                    }}
                                  >
                                    <Trash2 size={13} />
                                  </button>
                                )}
                              </>
                            )}
                          </div>
                        </td>
                      </tr>

                      {/* Inline edit expanded row for description + ui_template */}
                      {isEditing && editState && (
                        <tr style={{ borderBottom: !isLast ? `1px solid ${BORDER}` : "none", background: "rgba(95,114,249,0.03)" }}>
                          <td colSpan={10} style={{ padding: "12px 16px 16px" }}>
                            <div style={{ display: "grid", gridTemplateColumns: "1fr 200px 80px", gap: 10, maxWidth: 760 }}>
                              <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                                <span style={{ color: MUTED, fontSize: 10, fontWeight: 700, textTransform: "uppercase" }}>Description</span>
                                <textarea rows={2} style={{ ...inp(), resize: "vertical" as const }}
                                  value={editState.description}
                                  onChange={e => setEditState(prev => prev ? { ...prev, description: e.target.value } : prev)}
                                  placeholder="Tool description…"
                                />
                              </label>
                              <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                                <span style={{ color: MUTED, fontSize: 10, fontWeight: 700, textTransform: "uppercase" }}>UI Template</span>
                                <select style={inp()}
                                  value={editState.ui_template}
                                  onChange={e => setEditState(prev => prev ? { ...prev, ui_template: e.target.value } : prev)}
                                >
                                  {UI_TEMPLATES.map(tmpl => (
                                    <option key={tmpl.value} value={tmpl.value}>{tmpl.label}</option>
                                  ))}
                                </select>
                                <span style={{ color: MUTED, fontSize: 10 }}>Input form shown to users</span>
                              </label>
                              <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                                <span style={{ color: MUTED, fontSize: 10, fontWeight: 700, textTransform: "uppercase" }}>Icon</span>
                                <input style={inp({ textAlign: "center", fontSize: 20, padding: "12px 8px" })}
                                  value={editState.icon}
                                  onChange={e => setEditState(prev => prev ? { ...prev, icon: e.target.value } : prev)}
                                  placeholder="✨"
                                />
                              </label>
                            </div>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {/* ── Disable Confirm Modal ── */}
        {disableTarget && (
          <div style={{
            position: "fixed", inset: 0, zIndex: 1000,
            background: "rgba(0,0,0,0.7)", backdropFilter: "blur(4px)",
            display: "flex", alignItems: "center", justifyContent: "center",
          }} onClick={() => setDisableTarget(null)}>
            <div style={{
              background: "#131525", border: "1px solid rgba(239,68,68,0.3)",
              borderRadius: 16, padding: 28, maxWidth: 400, width: "100%",
            }} onClick={e => e.stopPropagation()}>
              <div style={{ display: "flex", gap: 14, marginBottom: 18 }}>
                <div style={{
                  width: 42, height: 42, borderRadius: 10,
                  background: "rgba(239,68,68,0.12)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  color: "#f87171", flexShrink: 0,
                }}>
                  <AlertTriangle size={20} />
                </div>
                <div>
                  <h3 style={{ color: TEXT, fontSize: 16, fontWeight: 700, margin: "0 0 6px" }}>Disable Tool?</h3>
                  <p style={{ color: MUTED, fontSize: 13, margin: 0, lineHeight: 1.5 }}>
                    <strong style={{ color: TEXT }}>{disableTarget.name}</strong> will be marked as disabled.
                    Users won't be able to access it until re-enabled.
                  </p>
                </div>
              </div>
              <div style={{ display: "flex", gap: 10, justifyContent: "flex-end" }}>
                <button onClick={() => setDisableTarget(null)} style={{
                  background: "rgba(255,255,255,0.05)", border: `1px solid ${BORDER}`,
                  color: MUTED, borderRadius: 9, padding: "8px 16px", cursor: "pointer", fontSize: 13,
                }}>Cancel</button>
                <button onClick={confirmDisable} disabled={disabling} style={{
                  background: "#ef4444", border: "none", color: "#fff",
                  borderRadius: 9, padding: "8px 20px", cursor: disabling ? "not-allowed" : "pointer",
                  fontSize: 13, fontWeight: 600, opacity: disabling ? 0.7 : 1,
                  display: "flex", alignItems: "center", gap: 6,
                }}>
                  {disabling ? <Loader2 size={13} /> : <Trash2 size={13} />}
                  {disabling ? "Disabling…" : "Disable"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* ── Modals ── */}
        {showCreate && (
          <CreateModal
            onClose={() => setShowCreate(false)}
            onCreated={tool => {
              setTools(prev => [...prev, tool]);
            }}
          />
        )}
        {showStats && <StatsPanel onClose={() => setShowStats(false)} />}
        {errPanelTool && <ErrorsPanel tool={errPanelTool} onClose={() => setErrPanelTool(null)} />}

      </div>

      {/* ── Spin animation ── */}
      <style>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        .animate-spin { animation: spin 1s linear infinite; }
        select option { background: #1a1b2e; color: #e2e8ff; }
      `}</style>
    </AdminShell>
  );
}
