"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import adminAPI, { CSVUploadSummary } from "@/lib/api";

// ─── helpers ──────────────────────────────────────────────────────────────────
function fmtDate(iso: string | null | undefined) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("en-NG", {
    day: "2-digit", month: "short", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

const STATUS_STYLE: Record<string, { bg: string; text: string; label: string }> = {
  DONE:    { bg: "#dcfce7", text: "#166534", label: "Done" },
  PARTIAL: { bg: "#fef9c3", text: "#854d0e", label: "Partial" },
  FAILED:  { bg: "#fee2e2", text: "#991b1b", label: "Failed" },
  PENDING: { bg: "#e0e7ff", text: "#3730a3", label: "Pending" },
};
function StatusBadge({ status }: { status: string }) {
  const s = STATUS_STYLE[status] ?? { bg: "#f3f4f6", text: "#374151", label: status };
  return (
    <span style={{
      background: s.bg, color: s.text,
      padding: "2px 10px", borderRadius: 12,
      fontSize: 12, fontWeight: 600,
    }}>{s.label}</span>
  );
}

// ─── CSV template download ─────────────────────────────────────────────────────
function downloadTemplate() {
  const csv = "msisdn,date,time,amount,recharge_type\n08012345678,2024-01-15,14:30,500,AIRTIME\n08098765432,2024-01-15,16:00,1000,DATA\n";
  const blob = new Blob([csv], { type: "text/csv" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url; a.download = "mtn_recharge_template.csv"; a.click();
  URL.revokeObjectURL(url);
}

// ─── Upload form component ─────────────────────────────────────────────────────
function UploadForm({ onDone }: { onDone: () => void }) {
  const [file, setFile]       = useState<File | null>(null);
  const [note, setNote]       = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult]   = useState<{ ok: boolean; msg: string } | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleSubmit = async () => {
    if (!file) return;
    setLoading(true);
    setResult(null);
    try {
      const r = await adminAPI.uploadMTNPushCSV(file, note || undefined);
      setResult({
        ok: r.status !== "FAILED",
        msg: `Upload complete — ${r.processed_rows}/${r.total_rows} processed, ${r.skipped_rows} skipped, ${r.failed_rows} failed.`,
      });
      setFile(null);
      setNote("");
      if (inputRef.current) inputRef.current.value = "";
      onDone();
    } catch (e: unknown) {
      setResult({ ok: false, msg: (e as Error).message });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      background: "#fff", border: "1px solid #e5e7eb",
      borderRadius: 12, padding: 24,
    }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 16 }}>
        <div>
          <h2 style={{ fontSize: 16, fontWeight: 700, color: "#111827", margin: 0 }}>
            Upload Recharge CSV
          </h2>
          <p style={{ fontSize: 13, color: "#6b7280", marginTop: 4 }}>
            Fallback for when the MTN push webhook is unavailable. Each row triggers the full pipeline.
          </p>
        </div>
        <button
          onClick={downloadTemplate}
          style={{
            background: "#f3f4f6", color: "#374151",
            border: "1px solid #d1d5db", borderRadius: 8,
            padding: "6px 14px", fontSize: 13, cursor: "pointer", fontWeight: 500,
          }}
        >
          ↓ Download Template
        </button>
      </div>

      {/* File picker */}
      <div style={{
        border: "2px dashed #d1d5db", borderRadius: 8,
        padding: "24px 16px", textAlign: "center",
        background: file ? "#f0fdf4" : "#fafafa",
        cursor: "pointer", marginBottom: 12,
      }}
        onClick={() => inputRef.current?.click()}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".csv,text/csv"
          style={{ display: "none" }}
          onChange={e => setFile(e.target.files?.[0] ?? null)}
        />
        {file ? (
          <div>
            <div style={{ fontSize: 24, marginBottom: 4 }}>📄</div>
            <div style={{ fontWeight: 600, color: "#166534", fontSize: 14 }}>{file.name}</div>
            <div style={{ color: "#6b7280", fontSize: 12, marginTop: 2 }}>
              {(file.size / 1024).toFixed(1)} KB — click to change
            </div>
          </div>
        ) : (
          <div>
            <div style={{ fontSize: 28, marginBottom: 6 }}>📁</div>
            <div style={{ color: "#6b7280", fontSize: 14 }}>
              Click to select a <strong>.csv</strong> file
            </div>
            <div style={{ color: "#9ca3af", fontSize: 12, marginTop: 4 }}>
              Required columns: <code>msisdn, date, time, amount</code> — optional: <code>recharge_type</code>
            </div>
          </div>
        )}
      </div>

      {/* Note */}
      <input
        type="text"
        placeholder="Optional note (e.g. 'MTN outage 2024-01-15 backfill')"
        value={note}
        onChange={e => setNote(e.target.value)}
        style={{
          width: "100%", padding: "8px 12px",
          border: "1px solid #d1d5db", borderRadius: 8,
          fontSize: 14, outline: "none", marginBottom: 12,
          boxSizing: "border-box",
        }}
      />

      {result && (
        <div style={{
          padding: "10px 14px", borderRadius: 8, marginBottom: 12, fontSize: 13,
          background: result.ok ? "#dcfce7" : "#fee2e2",
          color: result.ok ? "#166534" : "#991b1b",
          border: `1px solid ${result.ok ? "#bbf7d0" : "#fecaca"}`,
        }}>
          {result.ok ? "✅ " : "❌ "}{result.msg}
        </div>
      )}

      <button
        onClick={handleSubmit}
        disabled={!file || loading}
        style={{
          background: (!file || loading) ? "#9ca3af" : "#4f46e5",
          color: "#fff", border: "none", borderRadius: 8,
          padding: "10px 24px", fontSize: 14, fontWeight: 600,
          cursor: (!file || loading) ? "not-allowed" : "pointer",
          width: "100%",
        }}
      >
        {loading ? "Processing…" : "Upload & Process"}
      </button>
    </div>
  );
}

// ─── Main page ─────────────────────────────────────────────────────────────────
export default function MTNPushUploadPage() {
  const router = useRouter();
  const [uploads, setUploads] = useState<CSVUploadSummary[]>([]);
  const [total, setTotal]     = useState(0);
  const [page, setPage]       = useState(0);
  const [loading, setLoading] = useState(true);
  const PAGE_SIZE = 20;

  const load = async (p = page) => {
    setLoading(true);
    try {
      const r = await adminAPI.listMTNPushCSVUploads(PAGE_SIZE, p * PAGE_SIZE);
      setUploads(r.uploads ?? []);
      setTotal(r.total ?? 0);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(0); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handlePageChange = (p: number) => { setPage(p); load(p); };

  return (
    <AdminShell>
      <div style={{ maxWidth: 1100, margin: "0 auto" }}>
        {/* Header */}
        <div style={{ marginBottom: 24 }}>
          <h1 style={{ fontSize: 22, fontWeight: 800, color: "#e2e8ff", margin: 0 }}>
            MTN Push — Manual CSV Upload
          </h1>
          <p style={{ color: "#828cb4", fontSize: 14, marginTop: 4 }}>
            Fallback pipeline for MTN API outages. Upload a CSV to trigger spin credits, pulse points, and draw entries for each recharge row.
          </p>
        </div>

        {/* Upload form */}
        <div style={{ marginBottom: 32 }}>
          <UploadForm onDone={() => load(0)} />
        </div>

        {/* Batch history */}
        <div style={{ background: "#fff", border: "1px solid #e5e7eb", borderRadius: 12, overflow: "hidden" }}>
          <div style={{ padding: "16px 20px", borderBottom: "1px solid #f3f4f6", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
            <div>
              <h2 style={{ fontSize: 15, fontWeight: 700, color: "#111827", margin: 0 }}>Upload History</h2>
              <p style={{ fontSize: 12, color: "#9ca3af", margin: 0 }}>{total} total batches</p>
            </div>
          </div>

          {loading ? (
            <div style={{ display: "flex", justifyContent: "center", padding: 40 }}>
              <div style={{
                width: 32, height: 32, border: "3px solid #e5e7eb",
                borderTopColor: "#4f46e5", borderRadius: "50%",
                animation: "spin 0.8s linear infinite",
              }} />
            </div>
          ) : uploads.length === 0 ? (
            <div style={{ padding: 40, textAlign: "center", color: "#9ca3af", fontSize: 14 }}>
              No uploads yet. Use the form above to upload your first CSV.
            </div>
          ) : (
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ background: "#f9fafb", borderBottom: "1px solid #e5e7eb" }}>
                  {["Filename", "Uploaded By", "Uploaded At", "Rows", "Processed", "Skipped", "Failed", "Status", ""].map(h => (
                    <th key={h} style={{
                      textAlign: "left", padding: "10px 14px",
                      fontSize: 11, fontWeight: 700, color: "#6b7280",
                      textTransform: "uppercase", letterSpacing: "0.05em",
                    }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {uploads.map((u, i) => (
                  <tr key={u.id} style={{
                    borderBottom: "1px solid #f3f4f6",
                    background: i % 2 === 0 ? "#fff" : "#fafafa",
                  }}>
                    <td style={{ padding: "10px 14px", fontWeight: 500, color: "#111827", maxWidth: 200, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {u.filename}
                      {u.note && <div style={{ fontSize: 11, color: "#9ca3af", marginTop: 2 }}>{u.note}</div>}
                    </td>
                    <td style={{ padding: "10px 14px", color: "#374151" }}>{u.uploaded_by || "—"}</td>
                    <td style={{ padding: "10px 14px", color: "#374151" }}>{fmtDate(u.uploaded_at)}</td>
                    <td style={{ padding: "10px 14px", color: "#374151", textAlign: "right" }}>{u.total_rows}</td>
                    <td style={{ padding: "10px 14px", color: "#166534", textAlign: "right", fontWeight: 600 }}>{u.processed_rows}</td>
                    <td style={{ padding: "10px 14px", color: "#854d0e", textAlign: "right" }}>{u.skipped_rows}</td>
                    <td style={{ padding: "10px 14px", color: "#991b1b", textAlign: "right" }}>{u.failed_rows}</td>
                    <td style={{ padding: "10px 14px" }}><StatusBadge status={u.status} /></td>
                    <td style={{ padding: "10px 14px" }}>
                      <button
                        onClick={() => router.push(`/mtn-push-upload/${u.id}`)}
                        style={{
                          background: "#f3f4f6", color: "#374151",
                          border: "1px solid #d1d5db", borderRadius: 6,
                          padding: "4px 12px", fontSize: 12, cursor: "pointer",
                          fontWeight: 500,
                        }}
                      >
                        View Rows →
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
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
      </div>

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
      `}</style>
    </AdminShell>
  );
}
