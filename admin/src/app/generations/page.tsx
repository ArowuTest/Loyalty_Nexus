"use client";

import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { Generation } from "@/lib/api";

// ── helpers ───────────────────────────────────────────────────────────────────
const STATUS_COLORS: Record<string, { bg: string; text: string }> = {
  completed: { bg: "#064e3b", text: "#6ee7b7" },
  pending:   { bg: "#1e3a5f", text: "#93c5fd" },
  processing:{ bg: "#3b2f10", text: "#fcd34d" },
  failed:    { bg: "#4c0519", text: "#fca5a5" },
};
const sc = (s: string) => STATUS_COLORS[s] ?? { bg: "#1c2038", text: "#828cb4" };

function fmt(iso: string) {
  const d = new Date(iso);
  return d.toLocaleString("en-GB", { day:"2-digit", month:"short", year:"numeric",
    hour:"2-digit", minute:"2-digit" });
}

const TOOL_SLUGS = [
  "", "ai-chat","web-search-ai","code-helper","ai-photo","ai-photo-pro","ai-photo-max",
  "ai-photo-dream","bg-remover","photo-editor","animate-photo","video-cinematic",
  "video-premium","video-jingle","video-veo","jingle","bg-music","song-creator",
  "instrumental","narrate","narrate-pro","transcribe","transcribe-african","translate",
  "study-guide","quiz","mindmap","research-brief","bizplan","slide-deck","infographic",
  "podcast","image-analyser","ask-my-photo",
];

// ── page ──────────────────────────────────────────────────────────────────────
export default function GenerationsPage() {
  const [rows,    setRows]    = useState<Generation[]>([]);
  const [total,   setTotal]   = useState(0);
  const [loading, setLoading] = useState(false);
  const [error,   setError]   = useState("");

  // filters
  const [status,   setStatus]   = useState("");
  const [toolSlug, setToolSlug] = useState("");
  const [page,     setPage]     = useState(0);
  const limit = 50;

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const res = await adminAPI.getStudioGenerations({
        status:    status    || undefined,
        tool_slug: toolSlug  || undefined,
        limit,
        offset: page * limit,
      });
      // backend returns { generations, total } — api.ts maps to { items, total }
      const data = res as unknown as { generations?: Generation[]; items?: Generation[]; total: number };
      setRows(data.generations ?? data.items ?? []);
      setTotal(data.total ?? 0);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load generations");
    } finally {
      setLoading(false);
    }
  }, [status, toolSlug, page]);

  useEffect(() => { load(); }, [load]);

  // reset page when filters change
  const applyStatus   = (v: string) => { setStatus(v);   setPage(0); };
  const applyToolSlug = (v: string) => { setToolSlug(v); setPage(0); };

  const totalPages = Math.ceil(total / limit);

  // ── styles ──────────────────────────────────────────────────────────────────
  const S = {
    page:   { padding: "32px 24px", maxWidth: 1400 } as React.CSSProperties,
    header: { display:"flex", alignItems:"center", justifyContent:"space-between",
              marginBottom: 28 } as React.CSSProperties,
    h1:     { color:"#e2e8ff", fontWeight:700, fontSize:22, margin:0 } as React.CSSProperties,
    sub:    { color:"#828cb4", fontSize:13, marginTop:4 } as React.CSSProperties,

    filters:{ display:"flex", gap:12, marginBottom:20, flexWrap:"wrap" as const },
    sel:    { background:"#161929", border:"1px solid rgba(95,114,249,0.25)",
              color:"#c5cae9", borderRadius:8, padding:"8px 12px", fontSize:13,
              cursor:"pointer" } as React.CSSProperties,
    btn:    { background:"rgba(95,114,249,0.15)", border:"1px solid rgba(95,114,249,0.3)",
              color:"#a5b4fc", borderRadius:8, padding:"8px 14px", fontSize:13,
              cursor:"pointer" } as React.CSSProperties,
    btnAct: { background:"rgba(95,114,249,0.4)", color:"#e2e8ff" } as React.CSSProperties,

    table:  { width:"100%", borderCollapse:"collapse" as const, fontSize:13 },
    th:     { padding:"10px 12px", textAlign:"left" as const, color:"#5f72f9",
              borderBottom:"1px solid rgba(95,114,249,0.15)",
              fontWeight:600, fontSize:11, textTransform:"uppercase" as const,
              letterSpacing:"0.06em" } as React.CSSProperties,
    td:     { padding:"10px 12px", borderBottom:"1px solid rgba(255,255,255,0.04)",
              color:"#c5cae9", verticalAlign:"top" as const } as React.CSSProperties,
    mono:   { fontFamily:"monospace", fontSize:11 } as React.CSSProperties,
    prompt: { maxWidth:360, overflow:"hidden", textOverflow:"ellipsis",
              whiteSpace:"nowrap" as const } as React.CSSProperties,

    badge:  (s: string): React.CSSProperties => ({
      display:"inline-block", padding:"2px 8px", borderRadius:12, fontSize:11,
      fontWeight:600, background: sc(s).bg, color: sc(s).text,
    }),
    pager:  { display:"flex", gap:8, alignItems:"center", justifyContent:"flex-end",
              marginTop:16 } as React.CSSProperties,
    stat:   { color:"#828cb4", fontSize:13 } as React.CSSProperties,
  };

  return (
    <AdminShell>
      <div style={S.page}>
        {/* ── header ── */}
        <div style={S.header}>
          <div>
            <h1 style={S.h1}>🧪 AI Generations</h1>
            <p style={S.sub}>
              Every studio generation — prompt, output, provider, points deducted.
              {total > 0 && ` (${total.toLocaleString()} total)`}
            </p>
          </div>
          <button style={S.btn} onClick={load} disabled={loading}>
            {loading ? "Loading…" : "↻ Refresh"}
          </button>
        </div>

        {/* ── filters ── */}
        <div style={S.filters}>
          <select style={S.sel} value={status} onChange={e => applyStatus(e.target.value)}>
            <option value="">All Statuses</option>
            <option value="completed">✅ Completed</option>
            <option value="pending">⏳ Pending</option>
            <option value="processing">🔄 Processing</option>
            <option value="failed">❌ Failed</option>
          </select>

          <select style={S.sel} value={toolSlug} onChange={e => applyToolSlug(e.target.value)}>
            {TOOL_SLUGS.map(s => (
              <option key={s} value={s}>{s || "All Tools"}</option>
            ))}
          </select>

          {(status || toolSlug) && (
            <button style={S.btn} onClick={() => { applyStatus(""); applyToolSlug(""); }}>
              ✕ Clear
            </button>
          )}
        </div>

        {/* ── error ── */}
        {error && (
          <div style={{ background:"#4c0519", color:"#fca5a5", padding:"12px 16px",
            borderRadius:8, marginBottom:16, fontSize:13 }}>
            ⚠ {error}
          </div>
        )}

        {/* ── table ── */}
        <div style={{ background:"#161929", borderRadius:12,
          border:"1px solid rgba(95,114,249,0.12)", overflow:"auto" }}>
          <table style={S.table}>
            <thead>
              <tr style={{ background:"rgba(95,114,249,0.05)" }}>
                <th style={S.th}>Tool</th>
                <th style={S.th}>Status</th>
                <th style={S.th}>Provider</th>
                <th style={S.th}>Prompt</th>
                <th style={S.th}>Points</th>
                <th style={S.th}>User</th>
                <th style={S.th}>Created</th>
              </tr>
            </thead>
            <tbody>
              {loading && (
                <tr>
                  <td colSpan={7} style={{ ...S.td, textAlign:"center", padding:40, color:"#828cb4" }}>
                    Loading generations…
                  </td>
                </tr>
              )}
              {!loading && rows.length === 0 && (
                <tr>
                  <td colSpan={7} style={{ ...S.td, textAlign:"center", padding:40, color:"#828cb4" }}>
                    No generations found for the selected filters.
                  </td>
                </tr>
              )}
              {rows.map(g => (
                <tr key={g.id} style={{ transition:"background 0.15s" }}
                  onMouseEnter={e => (e.currentTarget.style.background = "rgba(95,114,249,0.05)")}
                  onMouseLeave={e => (e.currentTarget.style.background = "transparent")}>
                  <td style={S.td}>
                    <span style={{ fontWeight:600, color:"#a5b4fc" }}>{g.tool_slug}</span>
                  </td>
                  <td style={S.td}>
                    <span style={S.badge(g.status)}>{g.status}</span>
                  </td>
                  <td style={{ ...S.td, ...S.mono, color:"#64748b" }}>{g.provider || "—"}</td>
                  <td style={{ ...S.td, ...S.prompt }} title={g.prompt}>{g.prompt || "—"}</td>
                  <td style={{ ...S.td, textAlign:"center" as const }}>
                    {g.points_deducted > 0
                      ? <span style={{ color:"#fcd34d", fontWeight:700 }}>−{g.points_deducted}</span>
                      : <span style={{ color:"#4ade80" }}>Free</span>}
                  </td>
                  <td style={{ ...S.td, ...S.mono, fontSize:11, color:"#64748b" }}>
                    {g.user_id.slice(0, 8)}…
                  </td>
                  <td style={{ ...S.td, color:"#64748b", fontSize:12 }}>{fmt(g.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* ── pagination ── */}
        {totalPages > 1 && (
          <div style={S.pager}>
            <span style={S.stat}>Page {page + 1} / {totalPages}</span>
            <button style={{ ...S.btn, ...(page === 0 ? { opacity:0.4, cursor:"default" } : {}) }}
              onClick={() => setPage(p => Math.max(0, p - 1))} disabled={page === 0}>
              ← Prev
            </button>
            <button style={{ ...S.btn, ...(page >= totalPages - 1 ? { opacity:0.4, cursor:"default" } : {}) }}
              onClick={() => setPage(p => Math.min(totalPages - 1, p + 1))}
              disabled={page >= totalPages - 1}>
              Next →
            </button>
          </div>
        )}
      </div>
    </AdminShell>
  );
}
