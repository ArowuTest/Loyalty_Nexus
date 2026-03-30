"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState, useCallback } from "react";
import adminAPI, { SpinClaim, ClaimStatistics } from "@/lib/api";

const STATUS_COLORS: Record<string, string> = {
  PENDING:              "#f59e0b",
  PENDING_ADMIN_REVIEW: "#ef4444",
  APPROVED:             "#10b981",
  REJECTED:             "#6b7280",
  CLAIMED:              "#5f72f9",
  EXPIRED:              "#374151",
};

const PRIZE_TYPE_LABELS: Record<string, string> = {
  try_again:    "No Win",
  pulse_points: "Pulse Points",
  airtime:      "Airtime",
  data_bundle:  "Data Bundle",
  momo_cash:    "Cash Prize",
};

function fmtNaira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString("en-NG", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

function fmtDate(iso: string) {
  return new Date(iso).toLocaleString("en-NG", { dateStyle: "medium", timeStyle: "short" });
}

type ModalState =
  | { type: "approve"; claim: SpinClaim }
  | { type: "reject";  claim: SpinClaim }
  | { type: "detail";  claim: SpinClaim }
  | null;

export default function SpinClaimsPage() {
  const [claims, setClaims]       = useState<SpinClaim[]>([]);
  const [stats, setStats]         = useState<ClaimStatistics | null>(null);
  const [total, setTotal]         = useState(0);
  const [page, setPage]           = useState(1);
  const [statusFilter, setStatus] = useState("");
  const [loading, setLoading]     = useState(true);
  const [error, setError]         = useState<string | null>(null);
  const [modal, setModal]         = useState<ModalState>(null);
  const [actionNote, setActionNote]   = useState("");
  const [actionRef, setActionRef]     = useState("");
  const [actionReason, setActionReason] = useState("");
  const [submitting, setSubmitting]   = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [r, s] = await Promise.all([
        adminAPI.listClaims(statusFilter, page, 50),
        adminAPI.getClaimStatistics(),
      ]);
      setClaims(r.data ?? []);
      setTotal(r.total ?? 0);
      setStats(s);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load claims");
    } finally {
      setLoading(false);
    }
  }, [statusFilter, page]);

  useEffect(() => { load(); }, [load]);

  const handleApprove = async () => {
    if (!modal || modal.type !== "approve") return;
    setSubmitting(true);
    try {
      await adminAPI.approveClaim(modal.claim.id, actionNote, actionRef);
      setModal(null);
      setActionNote(""); setActionRef("");
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Approve failed");
    } finally {
      setSubmitting(false);
    }
  };

  const handleReject = async () => {
    if (!modal || modal.type !== "reject") return;
    if (!actionReason.trim()) { setError("Rejection reason is required"); return; }
    setSubmitting(true);
    try {
      await adminAPI.rejectClaim(modal.claim.id, actionReason, actionNote);
      setModal(null);
      setActionReason(""); setActionNote("");
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Reject failed");
    } finally {
      setSubmitting(false);
    }
  };

  const handleExport = async () => {
    try {
      const csv = await adminAPI.exportClaims(statusFilter);
      const blob = new Blob([csv as unknown as string], { type: "text/csv" });
      const url  = URL.createObjectURL(blob);
      const a    = document.createElement("a");
      a.href     = url;
      a.download = `claims_${statusFilter || "all"}_${new Date().toISOString().slice(0,10)}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Export failed");
    }
  };

  const totalPages = Math.ceil(total / 50);

  return (
    <AdminShell>
      <div className="max-w-6xl mx-auto space-y-6 pb-12">

        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>🏆 Spin Prize Claims</h1>
            <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
              Review, approve, and reject cash prize claims from users.
            </p>
          </div>
          <div style={{ display: "flex", gap: 10 }}>
            <button onClick={load}
              style={{ padding: "8px 16px", borderRadius: 8, border: "1px solid rgba(95,114,249,0.3)", color: "#828cb4", fontSize: 13, background: "transparent", cursor: "pointer" }}>
              ↺ Refresh
            </button>
            <button onClick={handleExport}
              style={{ padding: "8px 16px", borderRadius: 8, border: "1px solid rgba(16,185,129,0.4)", color: "#10b981", fontSize: 13, background: "transparent", cursor: "pointer" }}>
              ↓ Export CSV
            </button>
          </div>
        </div>

        {error && (
          <div style={{ background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.3)", borderRadius: 10, padding: "12px 16px", color: "#fca5a5", fontSize: 13, display: "flex", alignItems: "center", gap: 10 }}>
            ⚠️ {error}
            <button onClick={() => setError(null)} style={{ marginLeft: "auto", background: "none", border: "none", color: "#fca5a5", cursor: "pointer" }}>✕</button>
          </div>
        )}

        {/* Stats Cards */}
        {stats && (
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))", gap: 12 }}>
            {[
              { label: "Total Claims",    value: stats.total_claims,    color: "#e2e8ff" },
              { label: "Pending Review",  value: stats.pending_claims,  color: "#ef4444" },
              { label: "Approved",        value: stats.approved_claims, color: "#10b981" },
              { label: "Rejected",        value: stats.rejected_claims, color: "#6b7280" },
              { label: "Claimed",         value: stats.claimed_claims,  color: "#5f72f9" },
              { label: "Pending Value",   value: `₦${stats.pending_value_ngn.toLocaleString()}`,  color: "#f59e0b" },
              { label: "Approved Value",  value: `₦${stats.approved_value_ngn.toLocaleString()}`, color: "#10b981" },
            ].map(s => (
              <div key={s.label} className="card" style={{ padding: "14px 16px" }}>
                <p style={{ fontSize: 11, color: "#828cb4", marginBottom: 4 }}>{s.label}</p>
                <p style={{ fontSize: 20, fontWeight: 700, color: s.color }}>{s.value}</p>
              </div>
            ))}
          </div>
        )}

        {/* Filters */}
        <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
          <select value={statusFilter} onChange={e => { setStatus(e.target.value); setPage(1); }}
            style={{ background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13 }}>
            <option value="">All Statuses</option>
            <option value="PENDING">Pending</option>
            <option value="PENDING_ADMIN_REVIEW">Pending Admin Review</option>
            <option value="APPROVED">Approved</option>
            <option value="REJECTED">Rejected</option>
            <option value="CLAIMED">Claimed</option>
            <option value="EXPIRED">Expired</option>
          </select>
          <span style={{ color: "#828cb4", fontSize: 13 }}>{total} claim{total !== 1 ? "s" : ""}</span>
        </div>

        {/* Claims Table */}
        {loading ? (
          <div style={{ display: "flex", justifyContent: "center", padding: "60px 0" }}>
            <div style={{ width: 32, height: 32, border: "3px solid #5f72f9", borderTopColor: "transparent", borderRadius: "50%", animation: "spin 0.8s linear infinite" }} />
          </div>
        ) : claims.length === 0 ? (
          <div className="card" style={{ padding: "40px 0", textAlign: "center", color: "#828cb4" }}>
            No claims found{statusFilter ? ` with status "${statusFilter}"` : ""}.
          </div>
        ) : (
          <div className="card" style={{ overflow: "auto" }}>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.15)" }}>
                  {["Date", "User ID", "Prize Type", "Value", "MoMo Number", "Status", "Expires", "Actions"].map(h => (
                    <th key={h} style={{ padding: "10px 14px", textAlign: "left", color: "#828cb4", fontWeight: 600, whiteSpace: "nowrap" }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {claims.map(c => (
                  <tr key={c.id} style={{ borderBottom: "1px solid rgba(95,114,249,0.08)" }}>
                    <td style={{ padding: "10px 14px", color: "#c4cde8", whiteSpace: "nowrap" }}>{fmtDate(c.created_at)}</td>
                    <td style={{ padding: "10px 14px", color: "#828cb4", fontSize: 11, fontFamily: "monospace" }}>{c.user_id.slice(0,8)}…</td>
                    <td style={{ padding: "10px 14px", color: "#e2e8ff" }}>{PRIZE_TYPE_LABELS[c.prize_type] ?? c.prize_type}</td>
                    <td style={{ padding: "10px 14px", color: "#10b981", fontWeight: 600 }}>{fmtNaira(c.prize_value)}</td>
                    <td style={{ padding: "10px 14px", color: "#c4cde8", fontFamily: "monospace" }}>{c.momo_claim_number || c.momo_number || "—"}</td>
                    <td style={{ padding: "10px 14px" }}>
                      <span style={{ background: `${STATUS_COLORS[c.claim_status] ?? "#374151"}22`, color: STATUS_COLORS[c.claim_status] ?? "#828cb4", border: `1px solid ${STATUS_COLORS[c.claim_status] ?? "#374151"}44`, borderRadius: 6, padding: "2px 8px", fontSize: 11, fontWeight: 600 }}>
                        {c.claim_status}
                      </span>
                    </td>
                    <td style={{ padding: "10px 14px", color: "#828cb4", fontSize: 11, whiteSpace: "nowrap" }}>{fmtDate(c.expires_at)}</td>
                    <td style={{ padding: "10px 14px" }}>
                      <div style={{ display: "flex", gap: 6 }}>
                        <button onClick={() => setModal({ type: "detail", claim: c })}
                          style={{ padding: "4px 10px", borderRadius: 6, border: "1px solid rgba(95,114,249,0.3)", color: "#5f72f9", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                          View
                        </button>
                        {c.claim_status === "PENDING_ADMIN_REVIEW" && (
                          <>
                            <button onClick={() => { setModal({ type: "approve", claim: c }); setActionNote(""); setActionRef(""); }}
                              style={{ padding: "4px 10px", borderRadius: 6, border: "1px solid rgba(16,185,129,0.3)", color: "#10b981", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                              ✓ Approve
                            </button>
                            <button onClick={() => { setModal({ type: "reject", claim: c }); setActionReason(""); setActionNote(""); }}
                              style={{ padding: "4px 10px", borderRadius: 6, border: "1px solid rgba(239,68,68,0.3)", color: "#fca5a5", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                              ✕ Reject
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div style={{ display: "flex", gap: 8, justifyContent: "center" }}>
            <button disabled={page === 1} onClick={() => setPage(p => p - 1)}
              style={{ padding: "6px 14px", borderRadius: 7, border: "1px solid rgba(95,114,249,0.3)", color: page === 1 ? "#374151" : "#5f72f9", background: "transparent", cursor: page === 1 ? "default" : "pointer", fontSize: 13 }}>
              ← Prev
            </button>
            <span style={{ padding: "6px 14px", color: "#828cb4", fontSize: 13 }}>Page {page} / {totalPages}</span>
            <button disabled={page === totalPages} onClick={() => setPage(p => p + 1)}
              style={{ padding: "6px 14px", borderRadius: 7, border: "1px solid rgba(95,114,249,0.3)", color: page === totalPages ? "#374151" : "#5f72f9", background: "transparent", cursor: page === totalPages ? "default" : "pointer", fontSize: 13 }}>
              Next →
            </button>
          </div>
        )}
      </div>

      {/* Modals */}
      {modal && (
        <div style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 50 }}>
          <div className="card" style={{ width: "min(520px, 95vw)", padding: 28, maxHeight: "90vh", overflowY: "auto" }}>

            {modal.type === "detail" && (
              <>
                <h2 style={{ fontSize: 16, fontWeight: 700, color: "#e2e8ff", marginBottom: 16 }}>📋 Claim Details</h2>
                {[
                  ["Claim ID",       modal.claim.id],
                  ["User ID",        modal.claim.user_id],
                  ["Prize Type",     PRIZE_TYPE_LABELS[modal.claim.prize_type] ?? modal.claim.prize_type],
                  ["Prize Value",    fmtNaira(modal.claim.prize_value)],
                  ["Claim Status",   modal.claim.claim_status],
                  ["Fulfillment",    modal.claim.fulfillment_status],
                  ["MoMo Number",    modal.claim.momo_claim_number || modal.claim.momo_number || "—"],
                  ["Admin Notes",    modal.claim.admin_notes || "—"],
                  ["Rejection",      modal.claim.rejection_reason || "—"],
                  ["Payment Ref",    modal.claim.payment_reference || "—"],
                  ["Reviewed By",    modal.claim.reviewed_by || "—"],
                  ["Reviewed At",    modal.claim.reviewed_at ? fmtDate(modal.claim.reviewed_at) : "—"],
                  ["Created At",     fmtDate(modal.claim.created_at)],
                  ["Expires At",     fmtDate(modal.claim.expires_at)],
                ].map(([k, v]) => (
                  <div key={k} style={{ display: "flex", gap: 12, marginBottom: 8 }}>
                    <span style={{ width: 120, fontSize: 12, color: "#828cb4", flexShrink: 0 }}>{k}</span>
                    <span style={{ fontSize: 12, color: "#e2e8ff", wordBreak: "break-all" }}>{v}</span>
                  </div>
                ))}
                <button onClick={() => setModal(null)}
                  style={{ marginTop: 16, width: "100%", padding: "10px", borderRadius: 8, background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
                  Close
                </button>
              </>
            )}

            {modal.type === "approve" && (
              <>
                <h2 style={{ fontSize: 16, fontWeight: 700, color: "#10b981", marginBottom: 4 }}>✓ Approve Claim</h2>
                <p style={{ fontSize: 13, color: "#828cb4", marginBottom: 16 }}>
                  Approving {fmtNaira(modal.claim.prize_value)} cash prize for MoMo {modal.claim.momo_claim_number || modal.claim.momo_number || "—"}
                </p>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 6 }}>Payment Reference (optional)</label>
                <input value={actionRef} onChange={e => setActionRef(e.target.value)} placeholder="e.g. MTN_MOMO_REF_12345"
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, marginBottom: 12, boxSizing: "border-box" }} />
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 6 }}>Admin Notes (optional)</label>
                <textarea value={actionNote} onChange={e => setActionNote(e.target.value)} rows={3} placeholder="Internal notes…"
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, marginBottom: 16, resize: "vertical", boxSizing: "border-box" }} />
                <div style={{ display: "flex", gap: 10 }}>
                  <button onClick={() => setModal(null)}
                    style={{ flex: 1, padding: "10px", borderRadius: 8, background: "transparent", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
                    Cancel
                  </button>
                  <button onClick={handleApprove} disabled={submitting}
                    style={{ flex: 1, padding: "10px", borderRadius: 8, background: "#10b981", border: "none", color: "#fff", fontWeight: 600, cursor: "pointer", opacity: submitting ? 0.6 : 1 }}>
                    {submitting ? "Approving…" : "Confirm Approve"}
                  </button>
                </div>
              </>
            )}

            {modal.type === "reject" && (
              <>
                <h2 style={{ fontSize: 16, fontWeight: 700, color: "#ef4444", marginBottom: 4 }}>✕ Reject Claim</h2>
                <p style={{ fontSize: 13, color: "#828cb4", marginBottom: 16 }}>
                  Rejecting {fmtNaira(modal.claim.prize_value)} claim for user {modal.claim.user_id.slice(0,8)}…
                </p>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 6 }}>Rejection Reason <span style={{ color: "#ef4444" }}>*</span></label>
                <input value={actionReason} onChange={e => setActionReason(e.target.value)} placeholder="e.g. Invalid MoMo number provided"
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(239,68,68,0.3)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, marginBottom: 12, boxSizing: "border-box" }} />
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 6 }}>Admin Notes (optional)</label>
                <textarea value={actionNote} onChange={e => setActionNote(e.target.value)} rows={2} placeholder="Internal notes…"
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, marginBottom: 16, resize: "vertical", boxSizing: "border-box" }} />
                <div style={{ display: "flex", gap: 10 }}>
                  <button onClick={() => setModal(null)}
                    style={{ flex: 1, padding: "10px", borderRadius: 8, background: "transparent", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
                    Cancel
                  </button>
                  <button onClick={handleReject} disabled={submitting}
                    style={{ flex: 1, padding: "10px", borderRadius: 8, background: "#ef4444", border: "none", color: "#fff", fontWeight: 600, cursor: "pointer", opacity: submitting ? 0.6 : 1 }}>
                    {submitting ? "Rejecting…" : "Confirm Reject"}
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </AdminShell>
  );
}
