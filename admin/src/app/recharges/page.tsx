"use client";
import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI from "@/lib/api";

interface Recharge {
  id: string;
  msisdn: string;
  network: string;
  recharge_type: "AIRTIME" | "DATA";
  amount_kobo: number;
  data_variation_code: string;
  payment_reference: string;
  vtpass_request_id: string;
  user_id: string | null;
  status: "PENDING" | "PROCESSING" | "SUCCESS" | "FAILED" | "CANCELLED";
  failure_reason: string;
  points_earned: number;
  draw_entries: number;
  spin_eligible: boolean;
  created_at: string;
}

interface RechargesResponse {
  data: Recharge[];
  total: number;
}

const STATUS_COLORS: Record<string, string> = {
  PENDING:    "#f59e0b",
  PROCESSING: "#5f72f9",
  SUCCESS:    "#10b981",
  FAILED:     "#ef4444",
  CANCELLED:  "#6b7280",
};

const NETWORK_COLORS: Record<string, string> = {
  MTN:     "#fbbf24",
  GLO:     "#22c55e",
  AIRTEL:  "#ef4444",
  "9MOBILE": "#5f72f9",
};

function fmtNaira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString("en-NG", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}
function fmtDate(iso: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("en-NG", { dateStyle: "medium", timeStyle: "short" });
}

function StatusBadge({ status }: { status: string }) {
  const color = STATUS_COLORS[status] ?? "#828cb4";
  return (
    <span style={{
      background: `${color}22`, color, border: `1px solid ${color}44`,
      borderRadius: 6, padding: "2px 8px", fontSize: 11, fontWeight: 600, whiteSpace: "nowrap",
    }}>
      {status}
    </span>
  );
}

export default function RechargesPage() {
  const [recharges, setRecharges]   = useState<Recharge[]>([]);
  const [total, setTotal]           = useState(0);
  const [page, setPage]             = useState(1);
  const [statusFilter, setStatus]   = useState("");
  const [networkFilter, setNetwork] = useState("");
  const [dateFrom, setDateFrom]     = useState("");
  const [dateTo, setDateTo]         = useState("");
  const [loading, setLoading]       = useState(true);
  const [error, setError]           = useState<string | null>(null);
  const [detail, setDetail]         = useState<Recharge | null>(null);

  const PAGE_SIZE = 50;

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const qs = new URLSearchParams();
      qs.set("page", String(page));
      qs.set("page_size", String(PAGE_SIZE));
      if (statusFilter)  qs.set("status",  statusFilter);
      if (networkFilter) qs.set("network", networkFilter);
      if (dateFrom)      qs.set("date_from", dateFrom);
      if (dateTo)        qs.set("date_to",   dateTo);
      const r = await adminAPI.req<RechargesResponse>("GET", `/admin/recharges?${qs.toString()}`);
      setRecharges(r.data ?? []);
      setTotal(r.total ?? 0);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load recharges");
    } finally {
      setLoading(false);
    }
  }, [statusFilter, networkFilter, dateFrom, dateTo, page]);

  useEffect(() => { load(); }, [load]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  // Derived stat counts from current page (accurate totals come from the API total field)
  const successCount   = recharges.filter(r => r.status === "SUCCESS").length;
  const failedCount    = recharges.filter(r => r.status === "FAILED").length;
  const pendingCount   = recharges.filter(r => r.status === "PENDING" || r.status === "PROCESSING").length;

  return (
    <AdminShell>
      <div className="max-w-7xl mx-auto space-y-6 pb-12">

        {/* Header */}
        <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", flexWrap: "wrap", gap: 12 }}>
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>🧾 Recharge History</h1>
            <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
              VTU recharge transactions — airtime &amp; data.
            </p>
          </div>
          <button onClick={load}
            style={{ padding: "8px 16px", borderRadius: 8, border: "1px solid rgba(95,114,249,0.3)", color: "#828cb4", fontSize: 13, background: "transparent", cursor: "pointer" }}>
            ↺ Refresh
          </button>
        </div>

        {error && (
          <div style={{ background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.3)", borderRadius: 10, padding: "12px 16px", color: "#fca5a5", fontSize: 13, display: "flex", gap: 10, alignItems: "center" }}>
            ⚠️ {error}
            <button onClick={() => setError(null)} style={{ marginLeft: "auto", background: "none", border: "none", color: "#fca5a5", cursor: "pointer" }}>✕</button>
          </div>
        )}

        {/* Stats Cards */}
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(148px, 1fr))", gap: 12 }}>
          {[
            { label: "Total (this page)", value: recharges.length, color: "#e2e8ff" },
            { label: "Successful",        value: successCount,     color: "#10b981" },
            { label: "Failed",            value: failedCount,      color: "#ef4444", highlight: failedCount > 0 },
            { label: "Pending/Processing",value: pendingCount,     color: "#f59e0b" },
          ].map(s => (
            <div key={s.label} className="card" style={{ padding: "14px 16px", border: (s as { highlight?: boolean }).highlight ? "1px solid rgba(239,68,68,0.3)" : undefined }}>
              <p style={{ fontSize: 11, color: "#828cb4", marginBottom: 4 }}>{s.label}</p>
              <p style={{ fontSize: 20, fontWeight: 700, color: s.color }}>{s.value}</p>
            </div>
          ))}
        </div>

        {/* Filters */}
        <div style={{ display: "flex", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
          <select value={statusFilter} onChange={e => { setStatus(e.target.value); setPage(1); }}
            style={{ background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, cursor: "pointer" }}>
            <option value="">All Statuses</option>
            <option value="PENDING">Pending</option>
            <option value="PROCESSING">Processing</option>
            <option value="SUCCESS">Success</option>
            <option value="FAILED">Failed</option>
            <option value="CANCELLED">Cancelled</option>
          </select>
          <select value={networkFilter} onChange={e => { setNetwork(e.target.value); setPage(1); }}
            style={{ background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, cursor: "pointer" }}>
            <option value="">All Networks</option>
            <option value="MTN">MTN</option>
            <option value="GLO">GLO</option>
            <option value="AIRTEL">AIRTEL</option>
            <option value="9MOBILE">9MOBILE</option>
          </select>
          <input type="date" value={dateFrom} onChange={e => { setDateFrom(e.target.value); setPage(1); }}
            style={{ background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13 }} />
          <span style={{ color: "#828cb4", fontSize: 13 }}>→</span>
          <input type="date" value={dateTo} onChange={e => { setDateTo(e.target.value); setPage(1); }}
            style={{ background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13 }} />
          <span style={{ color: "#828cb4", fontSize: 13 }}>
            {loading ? "Loading…" : `${total.toLocaleString()} total`}
          </span>
        </div>

        {/* Table */}
        {loading ? (
          <div style={{ display: "flex", justifyContent: "center", padding: "60px 0" }}>
            <div style={{ width: 32, height: 32, border: "3px solid #5f72f9", borderTopColor: "transparent", borderRadius: "50%", animation: "spin 0.8s linear infinite" }} />
          </div>
        ) : recharges.length === 0 ? (
          <div className="card" style={{ padding: "40px 0", textAlign: "center", color: "#828cb4" }}>
            No recharges found.
          </div>
        ) : (
          <div className="card" style={{ overflow: "auto" }}>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.15)" }}>
                  {["Date", "Phone", "Network", "Type", "Amount", "Status", "VTPass ID", "Points", "Spin?"].map(h => (
                    <th key={h} style={{ padding: "10px 14px", textAlign: "left", color: "#828cb4", fontWeight: 600, whiteSpace: "nowrap" }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {recharges.map(r => (
                  <tr key={r.id}
                    onClick={() => setDetail(r)}
                    style={{ borderBottom: "1px solid rgba(95,114,249,0.08)", cursor: "pointer" }}
                    onMouseEnter={e => (e.currentTarget.style.background = "rgba(95,114,249,0.06)")}
                    onMouseLeave={e => (e.currentTarget.style.background = "transparent")}>
                    <td style={{ padding: "10px 14px", color: "#c4cde8", whiteSpace: "nowrap", fontSize: 12 }}>{fmtDate(r.created_at)}</td>
                    <td style={{ padding: "10px 14px", color: "#e2e8ff", fontFamily: "monospace", fontSize: 12 }}>{r.msisdn}</td>
                    <td style={{ padding: "10px 14px" }}>
                      <span style={{ color: NETWORK_COLORS[r.network] ?? "#828cb4", fontWeight: 700, fontSize: 12 }}>{r.network}</span>
                    </td>
                    <td style={{ padding: "10px 14px", color: "#c4cde8", fontSize: 12 }}>{r.recharge_type}</td>
                    <td style={{ padding: "10px 14px", color: "#10b981", fontWeight: 700 }}>{fmtNaira(r.amount_kobo)}</td>
                    <td style={{ padding: "10px 14px" }}><StatusBadge status={r.status} /></td>
                    <td style={{ padding: "10px 14px", color: "#828cb4", fontFamily: "monospace", fontSize: 11, maxWidth: 160, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {r.vtpass_request_id || "—"}
                    </td>
                    <td style={{ padding: "10px 14px", color: "#5f72f9", fontWeight: 600, fontSize: 12 }}>{r.points_earned.toLocaleString()}</td>
                    <td style={{ padding: "10px 14px", fontSize: 12 }}>
                      {r.spin_eligible ? <span style={{ color: "#10b981" }}>✓</span> : <span style={{ color: "#374151" }}>—</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div style={{ display: "flex", gap: 8, justifyContent: "center", alignItems: "center" }}>
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

      {/* Detail Modal */}
      {detail && (
        <div style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.75)", backdropFilter: "blur(4px)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 50 }}
          onClick={() => setDetail(null)}>
          <div className="card" style={{ width: "min(540px, 95vw)", padding: 28, maxHeight: "90vh", overflowY: "auto" }}
            onClick={e => e.stopPropagation()}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 16 }}>
              <h2 style={{ fontSize: 16, fontWeight: 700, color: "#e2e8ff" }}>🧾 Recharge Detail</h2>
              <StatusBadge status={detail.status} />
            </div>
            {[
              ["ID",               detail.id],
              ["Phone (MSISDN)",   detail.msisdn],
              ["Network",          detail.network],
              ["Type",             detail.recharge_type],
              ["Amount",           fmtNaira(detail.amount_kobo)],
              ["Status",           detail.status],
              ["Failure Reason",   detail.failure_reason || "—"],
              ["Payment Ref",      detail.payment_reference || "—"],
              ["VTPass Request ID",detail.vtpass_request_id || "—"],
              ["Data Code",        detail.data_variation_code || "—"],
              ["User ID",          detail.user_id || "—"],
              ["Points Earned",    detail.points_earned.toLocaleString()],
              ["Draw Entries",     detail.draw_entries.toLocaleString()],
              ["Spin Eligible",    detail.spin_eligible ? "Yes" : "No"],
              ["Created At",       fmtDate(detail.created_at)],
            ].map(([k, v]) => (
              <div key={k} style={{ display: "flex", gap: 12, marginBottom: 7, borderBottom: "1px solid rgba(95,114,249,0.05)", paddingBottom: 7 }}>
                <span style={{ width: 140, fontSize: 12, color: "#828cb4", flexShrink: 0 }}>{k}</span>
                <span style={{ fontSize: 12, color: "#e2e8ff", wordBreak: "break-all" }}>{v}</span>
              </div>
            ))}
            <button onClick={() => setDetail(null)}
              style={{ marginTop: 12, width: "100%", padding: "10px", borderRadius: 8, background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
              Close
            </button>
          </div>
        </div>
      )}

      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </AdminShell>
  );
}
