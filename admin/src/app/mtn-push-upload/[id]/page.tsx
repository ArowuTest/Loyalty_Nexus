"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import adminAPI, { CSVUploadSummary, CSVRowDetail } from "@/lib/api";

// ─── helpers ──────────────────────────────────────────────────────────────────
function fmtDate(iso: string | null | undefined) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("en-NG", {
    day: "2-digit", month: "short", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

const ROW_STATUS: Record<string, { bg: string; text: string; dot: string }> = {
  PROCESSED: { bg: "#dcfce7", text: "#166534", dot: "#16a34a" },
  SKIPPED:   { bg: "#fef9c3", text: "#854d0e", dot: "#ca8a04" },
  FAILED:    { bg: "#fee2e2", text: "#991b1b", dot: "#dc2626" },
};
function RowBadge({ status }: { status: string }) {
  const s = ROW_STATUS[status] ?? { bg: "#f3f4f6", text: "#374151", dot: "#9ca3af" };
  return (
    <span style={{
      display: "inline-flex", alignItems: "center", gap: 5,
      background: s.bg, color: s.text,
      padding: "2px 10px", borderRadius: 12,
      fontSize: 12, fontWeight: 600,
    }}>
      <span style={{ width: 6, height: 6, borderRadius: "50%", background: s.dot, display: "inline-block" }} />
      {status}
    </span>
  );
}

const BATCH_STATUS: Record<string, { bg: string; text: string }> = {
  DONE:    { bg: "#dcfce7", text: "#166534" },
  PARTIAL: { bg: "#fef9c3", text: "#854d0e" },
  FAILED:  { bg: "#fee2e2", text: "#991b1b" },
  PENDING: { bg: "#e0e7ff", text: "#3730a3" },
};

// ─── Summary card ──────────────────────────────────────────────────────────────
function SummaryCard({ label, value, sub }: { label: string; value: string | number; sub?: string }) {
  return (
    <div style={{
      background: "#fff", border: "1px solid #e5e7eb",
      borderRadius: 10, padding: "14px 18px", flex: 1, minWidth: 120,
    }}>
      <div style={{ fontSize: 11, color: "#9ca3af", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em" }}>{label}</div>
      <div style={{ fontSize: 22, fontWeight: 800, color: "#111827", marginTop: 4 }}>{value}</div>
      {sub && <div style={{ fontSize: 11, color: "#9ca3af", marginTop: 2 }}>{sub}</div>}
    </div>
  );
}

// ─── Main page ─────────────────────────────────────────────────────────────────
export default function MTNPushUploadDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;

  const [batch, setBatch]     = useState<CSVUploadSummary | null>(null);
  const [rows, setRows]       = useState<CSVRowDetail[]>([]);
  const [total, setTotal]     = useState(0);
  const [page, setPage]       = useState(0);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter]   = useState<"ALL" | "PROCESSED" | "SKIPPED" | "FAILED">("ALL");
  const PAGE_SIZE = 100;

  const load = async (p = 0) => {
    setLoading(true);
    try {
      const [batchData, rowData] = await Promise.all([
        adminAPI.getMTNPushCSVUpload(id),
        adminAPI.getMTNPushCSVUploadRows(id, PAGE_SIZE, p * PAGE_SIZE),
      ]);
      setBatch(batchData);
      setRows(rowData.rows ?? []);
      setTotal(rowData.total ?? 0);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(0); }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  const handlePageChange = (p: number) => { setPage(p); load(p); };

  const filteredRows = filter === "ALL" ? rows : rows.filter(r => r.status === filter);

  return (
    <AdminShell>
      <div style={{ maxWidth: 1200, margin: "0 auto" }}>

        {/* Back nav */}
        <button
          onClick={() => router.push("/mtn-push-upload")}
          style={{
            background: "none", border: "none", color: "#828cb4",
            fontSize: 13, cursor: "pointer", padding: 0, marginBottom: 16,
            display: "flex", alignItems: "center", gap: 4,
          }}
        >
          ← Back to Upload History
        </button>

        {loading && !batch ? (
          <div style={{ display: "flex", justifyContent: "center", padding: 80 }}>
            <div style={{
              width: 36, height: 36, border: "3px solid #e5e7eb",
              borderTopColor: "#4f46e5", borderRadius: "50%",
              animation: "spin 0.8s linear infinite",
            }} />
          </div>
        ) : batch ? (
          <>
            {/* Batch header */}
            <div style={{ marginBottom: 20 }}>
              <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", flexWrap: "wrap", gap: 12 }}>
                <div>
                  <h1 style={{ fontSize: 20, fontWeight: 800, color: "#e2e8ff", margin: 0 }}>
                    {batch.filename}
                  </h1>
                  <div style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
                    Uploaded by <strong style={{ color: "#a5b4fc" }}>{batch.uploaded_by || "unknown"}</strong>
                    {" · "}{fmtDate(batch.uploaded_at)}
                    {batch.note && <span style={{ marginLeft: 8, color: "#9ca3af" }}>— {batch.note}</span>}
                  </div>
                </div>
                <div>
                  {(() => {
                    const s = BATCH_STATUS[batch.status] ?? { bg: "#f3f4f6", text: "#374151" };
                    return (
                      <span style={{
                        background: s.bg, color: s.text,
                        padding: "4px 14px", borderRadius: 14,
                        fontSize: 13, fontWeight: 700,
                      }}>{batch.status}</span>
                    );
                  })()}
                </div>
              </div>
            </div>

            {/* Summary cards */}
            <div style={{ display: "flex", gap: 12, marginBottom: 24, flexWrap: "wrap" }}>
              <SummaryCard label="Total Rows"      value={batch.total_rows} />
              <SummaryCard label="Processed"       value={batch.processed_rows} sub="full pipeline ran" />
              <SummaryCard label="Skipped"         value={batch.skipped_rows} sub="already processed (idempotent)" />
              <SummaryCard label="Failed"          value={batch.failed_rows} sub="validation or pipeline error" />
              <SummaryCard label="Completed At"    value={fmtDate(batch.completed_at)} />
            </div>

            {/* Row filter tabs */}
            <div style={{ display: "flex", gap: 8, marginBottom: 12 }}>
              {(["ALL", "PROCESSED", "SKIPPED", "FAILED"] as const).map(f => (
                <button
                  key={f}
                  onClick={() => setFilter(f)}
                  style={{
                    padding: "6px 16px", borderRadius: 8, fontSize: 13, fontWeight: 600,
                    border: "1px solid",
                    borderColor: filter === f ? "#4f46e5" : "#d1d5db",
                    background: filter === f ? "#4f46e5" : "#fff",
                    color: filter === f ? "#fff" : "#374151",
                    cursor: "pointer",
                  }}
                >
                  {f === "ALL" ? `All (${total})` : f}
                </button>
              ))}
            </div>

            {/* Row table */}
            <div style={{ background: "#fff", border: "1px solid #e5e7eb", borderRadius: 12, overflow: "hidden" }}>
              {loading ? (
                <div style={{ display: "flex", justifyContent: "center", padding: 40 }}>
                  <div style={{
                    width: 28, height: 28, border: "3px solid #e5e7eb",
                    borderTopColor: "#4f46e5", borderRadius: "50%",
                    animation: "spin 0.8s linear infinite",
                  }} />
                </div>
              ) : filteredRows.length === 0 ? (
                <div style={{ padding: 32, textAlign: "center", color: "#9ca3af", fontSize: 14 }}>
                  No rows match the selected filter.
                </div>
              ) : (
                <div style={{ overflowX: "auto" }}>
                  <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 12 }}>
                    <thead>
                      <tr style={{ background: "#f9fafb", borderBottom: "1px solid #e5e7eb" }}>
                        {["#", "MSISDN", "Date", "Time", "Amount", "Type", "Status", "Spin ✦", "Pulse ✦", "Draw ✦", "Ref / Error"].map(h => (
                          <th key={h} style={{
                            textAlign: "left", padding: "9px 12px",
                            fontSize: 10, fontWeight: 700, color: "#6b7280",
                            textTransform: "uppercase", letterSpacing: "0.05em",
                            whiteSpace: "nowrap",
                          }}>{h}</th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {filteredRows.map((r, i) => (
                        <tr key={r.row_number} style={{
                          borderBottom: "1px solid #f3f4f6",
                          background: i % 2 === 0 ? "#fff" : "#fafafa",
                        }}>
                          <td style={{ padding: "8px 12px", color: "#9ca3af" }}>{r.row_number}</td>
                          <td style={{ padding: "8px 12px", fontWeight: 600, color: "#111827", fontFamily: "monospace" }}>{r.raw_msisdn}</td>
                          <td style={{ padding: "8px 12px", color: "#374151" }}>{r.raw_date}</td>
                          <td style={{ padding: "8px 12px", color: "#374151" }}>{r.raw_time}</td>
                          <td style={{ padding: "8px 12px", color: "#374151", textAlign: "right" }}>
                            ₦{parseFloat(r.raw_amount || "0").toLocaleString("en-NG")}
                          </td>
                          <td style={{ padding: "8px 12px", color: "#374151" }}>
                            <span style={{
                              background: r.recharge_type === "DATA" ? "#dbeafe" : "#f3f4f6",
                              color: r.recharge_type === "DATA" ? "#1e40af" : "#374151",
                              padding: "1px 8px", borderRadius: 8, fontSize: 11, fontWeight: 600,
                            }}>{r.recharge_type || "AIRTIME"}</span>
                          </td>
                          <td style={{ padding: "8px 12px" }}><RowBadge status={r.status} /></td>
                          <td style={{ padding: "8px 12px", textAlign: "right", color: r.spin_credits > 0 ? "#166534" : "#9ca3af", fontWeight: r.spin_credits > 0 ? 700 : 400 }}>
                            {r.spin_credits > 0 ? `+${r.spin_credits}` : "—"}
                          </td>
                          <td style={{ padding: "8px 12px", textAlign: "right", color: r.pulse_points > 0 ? "#166534" : "#9ca3af", fontWeight: r.pulse_points > 0 ? 700 : 400 }}>
                            {r.pulse_points > 0 ? `+${r.pulse_points}` : "—"}
                          </td>
                          <td style={{ padding: "8px 12px", textAlign: "right", color: r.draw_entries > 0 ? "#166534" : "#9ca3af", fontWeight: r.draw_entries > 0 ? 700 : 400 }}>
                            {r.draw_entries > 0 ? `+${r.draw_entries}` : "—"}
                          </td>
                          <td style={{ padding: "8px 12px", maxWidth: 260, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                            {r.status === "PROCESSED" && r.transaction_ref ? (
                              <span style={{ color: "#6b7280", fontFamily: "monospace", fontSize: 11 }}>{r.transaction_ref}</span>
                            ) : r.status === "SKIPPED" && r.skip_reason ? (
                              <span style={{ color: "#854d0e", fontSize: 11 }}>⏭ {r.skip_reason}</span>
                            ) : r.error_msg ? (
                              <span style={{ color: "#991b1b", fontSize: 11 }}>⚠ {r.error_msg}</span>
                            ) : "—"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {/* Pagination */}
              {total > PAGE_SIZE && (
                <div style={{ padding: "12px 20px", borderTop: "1px solid #f3f4f6", display: "flex", gap: 8, justifyContent: "flex-end" }}>
                  <button
                    disabled={page === 0}
                    onClick={() => handlePageChange(page - 1)}
                    style={{
                      padding: "6px 14px", borderRadius: 6, fontSize: 13,
                      border: "1px solid #d1d5db", cursor: page === 0 ? "not-allowed" : "pointer",
                      background: page === 0 ? "#f9fafb" : "#fff", color: "#374151",
                    }}
                  >← Prev</button>
                  <span style={{ padding: "6px 10px", fontSize: 13, color: "#6b7280" }}>
                    Page {page + 1} of {Math.ceil(total / PAGE_SIZE)}
                  </span>
                  <button
                    disabled={(page + 1) * PAGE_SIZE >= total}
                    onClick={() => handlePageChange(page + 1)}
                    style={{
                      padding: "6px 14px", borderRadius: 6, fontSize: 13,
                      border: "1px solid #d1d5db",
                      cursor: (page + 1) * PAGE_SIZE >= total ? "not-allowed" : "pointer",
                      background: (page + 1) * PAGE_SIZE >= total ? "#f9fafb" : "#fff",
                      color: "#374151",
                    }}
                  >Next →</button>
                </div>
              )}
            </div>
          </>
        ) : (
          <div style={{ padding: 40, textAlign: "center", color: "#9ca3af" }}>
            Upload batch not found.
          </div>
        )}
      </div>

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
      `}</style>
    </AdminShell>
  );
}
