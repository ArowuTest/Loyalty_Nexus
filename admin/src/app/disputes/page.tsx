"use client";

import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { Generation } from "@/lib/api";

// ── helpers ───────────────────────────────────────────────────────────────────
function fmt(iso: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("en-GB", {
    day: "2-digit", month: "short", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

function truncate(s: string, n = 80) {
  return s.length > n ? s.slice(0, n) + "…" : s;
}

const LIMIT = 50;

// ── page ──────────────────────────────────────────────────────────────────────
export default function DisputesPage() {
  const [rows,    setRows]    = useState<Generation[]>([]);
  const [total,   setTotal]   = useState(0);
  const [loading, setLoading] = useState(false);
  const [error,   setError]   = useState("");
  const [page,    setPage]    = useState(0);
  const [toolFilter, setToolFilter] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const res = await adminAPI.getStudioGenerations({
        disputed: true,
        tool_slug: toolFilter || undefined,
        limit: LIMIT,
        offset: page * LIMIT,
      });
      setRows(res.generations ?? []);
      setTotal(res.total ?? 0);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load disputes");
    } finally {
      setLoading(false);
    }
  }, [page, toolFilter]);

  useEffect(() => { load(); }, [load]);
  const applyFilter = (v: string) => { setToolFilter(v); setPage(0); };

  const totalPages = Math.ceil(total / LIMIT);

  // ── styles ──────────────────────────────────────────────────────────────────
  const S: Record<string, React.CSSProperties> = {
    page:    { padding: "32px 24px", maxWidth: 1300 },
    h1:      { color: "#e2e8ff", fontWeight: 700, fontSize: 22, margin: 0 },
    sub:     { color: "#828cb4", fontSize: 13, marginTop: 4 },
    filters: { display: "flex", gap: 12, marginBottom: 20, flexWrap: "wrap", alignItems: "center" },
    badge:   { display: "inline-flex", alignItems: "center", gap: 6,
               padding: "3px 10px", borderRadius: 20, fontSize: 11,
               fontWeight: 600, background: "#3b0a15", color: "#fca5a5", border: "1px solid #7f1d1d" },
    refund:  { display: "inline-flex", alignItems: "center", gap: 4,
               padding: "2px 8px", borderRadius: 12, fontSize: 11,
               fontWeight: 600, background: "#052e16", color: "#86efac", border: "1px solid #14532d" },
    card:    { background: "#1c2038", border: "1px solid rgba(95,114,249,0.1)", borderRadius: 12,
               overflow: "hidden", marginBottom: 1 },
    row:     { display: "grid", gridTemplateColumns: "1fr 120px 110px 90px 80px 80px 80px 140px",
               gap: 12, padding: "12px 16px", alignItems: "center", borderTop: "1px solid rgba(95,114,249,0.07)" },
    hdr:     { display: "grid", gridTemplateColumns: "1fr 120px 110px 90px 80px 80px 80px 140px",
               gap: 12, padding: "10px 16px",
               background: "rgba(95,114,249,0.07)", borderBottom: "1px solid rgba(95,114,249,0.1)" },
    hdrCell: { color: "#5f72f9", fontSize: 10, fontWeight: 700, letterSpacing: "0.06em", textTransform: "uppercase" },
    cell:    { color: "#828cb4", fontSize: 12 },
    id:      { color: "#5f72f9", fontSize: 11, fontFamily: "monospace" },
    prompt:  { color: "#e2e8ff", fontSize: 12 },
    points:  { color: "#fcd34d", fontSize: 12, fontWeight: 600 },
    empty:   { textAlign: "center" as const, padding: "48px 24px", color: "#828cb4" },
    pager:   { display: "flex", gap: 8, justifyContent: "center", marginTop: 20, alignItems: "center" },
    pgBtn:   { padding: "6px 14px", borderRadius: 6, border: "1px solid rgba(95,114,249,0.2)",
               background: "transparent", color: "#828cb4", cursor: "pointer", fontSize: 12 },
    pgBtnA:  { padding: "6px 14px", borderRadius: 6, border: "1px solid rgba(95,114,249,0.4)",
               background: "rgba(95,114,249,0.15)", color: "#5f72f9", cursor: "pointer", fontSize: 12, fontWeight: 600 },
    input:   { padding: "8px 12px", borderRadius: 8, border: "1px solid rgba(95,114,249,0.2)",
               background: "#131627", color: "#e2e8ff", fontSize: 13, outline: "none", minWidth: 180 },
    refresh: { padding: "8px 16px", borderRadius: 8, border: "1px solid rgba(95,114,249,0.3)",
               background: "rgba(95,114,249,0.12)", color: "#5f72f9", cursor: "pointer", fontSize: 13,
               fontWeight: 600 },
  };

  return (
    <AdminShell>
      <div style={S.page}>
        {/* Header */}
        <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", marginBottom: 28 }}>
          <div>
            <h1 style={S.h1}>🛑 Disputes &amp; Refunds</h1>
            <p style={S.sub}>
              User-flagged generations where output quality was disputed. Refunds are auto-processed
              (points restored within the refund window). This view is for audit &amp; monitoring.
            </p>
          </div>
          <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
            <span style={{ color: "#e2e8ff", fontSize: 13 }}>
              <b style={{ color: "#fca5a5" }}>{total}</b> disputed
            </span>
            <button style={S.refresh} onClick={load} disabled={loading}>
              {loading ? "Loading…" : "↻ Refresh"}
            </button>
          </div>
        </div>

        {/* Summary cards */}
        <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 16, marginBottom: 28 }}>
          {[
            { label: "Total Disputes",   value: total,                                        color: "#fca5a5" },
            { label: "Refunds Granted",  value: rows.filter(r => r.refund_granted).length,    color: "#86efac" },
            { label: "Pts Refunded",     value: rows.reduce((a, r) => a + r.refund_pts, 0) + " pts", color: "#fcd34d" },
          ].map(card => (
            <div key={card.label} style={{
              background: "#1c2038", border: "1px solid rgba(95,114,249,0.1)", borderRadius: 12,
              padding: "20px 24px",
            }}>
              <div style={{ color: "#828cb4", fontSize: 12, marginBottom: 6 }}>{card.label}</div>
              <div style={{ color: card.color, fontSize: 22, fontWeight: 700 }}>{card.value}</div>
            </div>
          ))}
        </div>

        {/* Filters */}
        <div style={S.filters}>
          <input
            style={S.input}
            placeholder="Filter by tool slug…"
            value={toolFilter}
            onChange={e => applyFilter(e.target.value)}
          />
          {toolFilter && (
            <button style={{ ...S.pgBtn, color: "#f43f5e", borderColor: "rgba(244,63,94,0.3)" }}
              onClick={() => applyFilter("")}>✕ Clear</button>
          )}
        </div>

        {/* Error */}
        {error && (
          <div style={{ padding: "12px 16px", borderRadius: 8, background: "#3b0a15",
            border: "1px solid #7f1d1d", color: "#fca5a5", marginBottom: 16, fontSize: 13 }}>
            {error}
          </div>
        )}

        {/* Table */}
        <div style={S.card}>
          {/* Header row */}
          <div style={S.hdr}>
            {["Generation ID / Prompt", "Tool", "Status", "Cost (pts)", "Refund Pts", "Refunded?", "Provider", "Disputed At"].map(h => (
              <div key={h} style={S.hdrCell}>{h}</div>
            ))}
          </div>

          {loading ? (
            <div style={S.empty}>Loading disputed generations…</div>
          ) : rows.length === 0 ? (
            <div style={S.empty}>
              <div style={{ fontSize: 32, marginBottom: 8 }}>✅</div>
              <div style={{ fontWeight: 600, color: "#e2e8ff", marginBottom: 4 }}>No disputes found</div>
              <div>No users have flagged any generations{toolFilter ? ` for tool "${toolFilter}"` : ""} yet.</div>
            </div>
          ) : (
            rows.map((g, i) => (
              <div key={g.id} style={{
                ...S.row,
                background: i % 2 === 0 ? "transparent" : "rgba(255,255,255,0.01)",
              }}>
                {/* ID + prompt */}
                <div>
                  <div style={S.id}>{g.id.slice(0, 8)}…</div>
                  <div style={{ ...S.prompt, marginTop: 2 }}>{truncate(g.prompt)}</div>
                </div>

                {/* Tool slug */}
                <div style={{ ...S.cell, fontFamily: "monospace", fontSize: 11, color: "#93c5fd" }}>
                  {g.tool_slug}
                </div>

                {/* Status */}
                <div>
                  <span style={{
                    padding: "2px 8px", borderRadius: 12, fontSize: 11, fontWeight: 600,
                    background: g.status === "completed" ? "#064e3b" : "#3b0a15",
                    color:      g.status === "completed" ? "#6ee7b7" : "#fca5a5",
                  }}>
                    {g.status}
                  </span>
                </div>

                {/* Points deducted */}
                <div style={S.points}>{g.points_deducted} pts</div>

                {/* Refund pts */}
                <div style={{ color: "#86efac", fontSize: 12, fontWeight: 600 }}>
                  {g.refund_pts > 0 ? `+${g.refund_pts} pts` : "—"}
                </div>

                {/* Refund granted */}
                <div>
                  {g.refund_granted ? (
                    <span style={S.refund}>✓ Yes</span>
                  ) : (
                    <span style={{ color: "#828cb4", fontSize: 12 }}>Pending</span>
                  )}
                </div>

                {/* Provider */}
                <div style={{ ...S.cell, fontSize: 11 }}>{g.provider || "—"}</div>

                {/* Disputed at */}
                <div style={{ ...S.cell, fontSize: 11 }}>{fmt(g.disputed_at)}</div>
              </div>
            ))
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div style={S.pager}>
            <button style={S.pgBtn} disabled={page === 0} onClick={() => setPage(p => p - 1)}>← Prev</button>
            {Array.from({ length: Math.min(totalPages, 7) }, (_, i) => i).map(i => (
              <button key={i} style={i === page ? S.pgBtnA : S.pgBtn} onClick={() => setPage(i)}>
                {i + 1}
              </button>
            ))}
            {totalPages > 7 && <span style={{ color: "#828cb4" }}>… {totalPages}</span>}
            <button style={S.pgBtn} disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>Next →</button>
          </div>
        )}

        {/* Info note */}
        <div style={{ marginTop: 24, padding: "14px 18px", borderRadius: 10,
          background: "rgba(95,114,249,0.06)", border: "1px solid rgba(95,114,249,0.15)" }}>
          <div style={{ color: "#93c5fd", fontWeight: 600, fontSize: 13, marginBottom: 4 }}>ℹ️ How disputes work</div>
          <div style={{ color: "#828cb4", fontSize: 12, lineHeight: 1.7 }}>
            Users can flag a generation as disputed within the configured refund window (typically 5 minutes).
            The system automatically calculates the refund percentage, restores PulsePoints to the user&apos;s wallet,
            writes a ledger entry, and marks the generation as disputed. No manual action is required for standard disputes.
            This page is for audit monitoring and fraud review.
          </div>
        </div>
      </div>
    </AdminShell>
  );
}
