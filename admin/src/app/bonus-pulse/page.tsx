"use client";

import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { BonusPulseAwardRecord, BonusPulseAwardResult } from "@/lib/api";

// ─── helpers ──────────────────────────────────────────────────────────────────
function fmt(iso: string) {
  return new Date(iso).toLocaleString("en-NG", {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

function pts(n: number) {
  return n.toLocaleString();
}

// ─── Award Form ───────────────────────────────────────────────────────────────
function AwardForm({ onSuccess }: { onSuccess: (r: BonusPulseAwardResult) => void }) {
  const [phone, setPhone]       = useState("");
  const [points, setPoints]     = useState("");
  const [campaign, setCampaign] = useState("");
  const [note, setNote]         = useState("");
  const [loading, setLoading]   = useState(false);
  const [error, setError]       = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const pts = parseInt(points, 10);
    if (!phone.trim() || isNaN(pts) || pts <= 0) {
      setError("Phone number and a positive point value are required.");
      return;
    }
    setLoading(true);
    try {
      const result = await adminAPI.awardBonusPulse({
        phone_number: phone.trim(),
        points: pts,
        campaign: campaign.trim() || undefined,
        note: note.trim() || undefined,
      });
      onSuccess(result);
      setPhone(""); setPoints(""); setCampaign(""); setNote("");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Award failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ background: "#fff", borderRadius: 12, padding: 28, boxShadow: "0 1px 4px rgba(0,0,0,.08)", marginBottom: 32 }}>
      <h2 style={{ margin: "0 0 20px", fontSize: 18, fontWeight: 700, color: "#111" }}>Award Bonus Pulse Points</h2>

      {error && (
        <div style={{ background: "#fef2f2", border: "1px solid #fca5a5", borderRadius: 8, padding: "10px 14px", marginBottom: 16, color: "#b91c1c", fontSize: 14 }}>
          {error}
        </div>
      )}

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16, marginBottom: 16 }}>
        <label style={labelStyle}>
          MSISDN (phone number) <span style={{ color: "#ef4444" }}>*</span>
          <input
            value={phone}
            onChange={e => setPhone(e.target.value)}
            placeholder="08012345678"
            required
            style={inputStyle}
          />
        </label>
        <label style={labelStyle}>
          Pulse Points to award <span style={{ color: "#ef4444" }}>*</span>
          <input
            type="number"
            min={1}
            value={points}
            onChange={e => setPoints(e.target.value)}
            placeholder="e.g. 500"
            required
            style={inputStyle}
          />
        </label>
        <label style={labelStyle}>
          Campaign name <span style={{ color: "#9ca3af", fontWeight: 400 }}>(optional)</span>
          <input
            value={campaign}
            onChange={e => setCampaign(e.target.value)}
            placeholder="e.g. Ramadan 2025"
            style={inputStyle}
          />
        </label>
        <label style={labelStyle}>
          Note <span style={{ color: "#9ca3af", fontWeight: 400 }}>(optional)</span>
          <input
            value={note}
            onChange={e => setNote(e.target.value)}
            placeholder="e.g. VIP incentive"
            style={inputStyle}
          />
        </label>
      </div>

      <button
        type="submit"
        disabled={loading}
        style={{
          background: loading ? "#6b7280" : "#7c3aed",
          color: "#fff",
          border: "none",
          borderRadius: 8,
          padding: "10px 24px",
          fontWeight: 600,
          fontSize: 14,
          cursor: loading ? "not-allowed" : "pointer",
        }}
      >
        {loading ? "Awarding…" : "Award Points"}
      </button>
    </form>
  );
}

// ─── Success Banner ───────────────────────────────────────────────────────────
function SuccessBanner({ result, onDismiss }: { result: BonusPulseAwardResult; onDismiss: () => void }) {
  return (
    <div style={{ background: "#f0fdf4", border: "1px solid #86efac", borderRadius: 10, padding: "14px 18px", marginBottom: 24, display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
      <div>
        <p style={{ margin: "0 0 4px", fontWeight: 700, color: "#166534", fontSize: 15 }}>
          ✓ {pts(result.points_awarded)} Pulse Points awarded to {result.phone_number}
        </p>
        <p style={{ margin: 0, color: "#15803d", fontSize: 13 }}>
          New balance: <strong>{pts(result.new_balance)}</strong> pts
          {result.campaign ? ` · Campaign: ${result.campaign}` : ""}
          {" · "}Award ID: <code style={{ fontSize: 12 }}>{result.award_id.slice(0, 8)}</code>
        </p>
      </div>
      <button onClick={onDismiss} style={{ background: "none", border: "none", cursor: "pointer", color: "#166534", fontSize: 18, lineHeight: 1 }}>×</button>
    </div>
  );
}

// ─── Audit Log Table ──────────────────────────────────────────────────────────
function AuditLog({ refresh }: { refresh: number }) {
  const [records, setRecords]   = useState<BonusPulseAwardRecord[]>([]);
  const [total, setTotal]       = useState(0);
  const [page, setPage]         = useState(0);
  const [filterPhone, setFilterPhone]       = useState("");
  const [filterCampaign, setFilterCampaign] = useState("");
  const [loading, setLoading]   = useState(false);
  const LIMIT = 20;

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await adminAPI.listBonusPulseAwards({
        phone:    filterPhone.trim()    || undefined,
        campaign: filterCampaign.trim() || undefined,
        limit:    LIMIT,
        offset:   page * LIMIT,
      });
      setRecords(res.records ?? []);
      setTotal(res.total ?? 0);
    } catch {
      // silently keep previous data on error
    } finally {
      setLoading(false);
    }
  }, [filterPhone, filterCampaign, page]);

  useEffect(() => { load(); }, [load, refresh]);

  const pages = Math.max(1, Math.ceil(total / LIMIT));

  return (
    <div style={{ background: "#fff", borderRadius: 12, padding: 24, boxShadow: "0 1px 4px rgba(0,0,0,.08)" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16, flexWrap: "wrap", gap: 12 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 700, color: "#111" }}>
          Award Audit Log
          <span style={{ marginLeft: 10, fontSize: 14, fontWeight: 400, color: "#6b7280" }}>({total.toLocaleString()} total)</span>
        </h2>
        <div style={{ display: "flex", gap: 10 }}>
          <input
            value={filterPhone}
            onChange={e => { setFilterPhone(e.target.value); setPage(0); }}
            placeholder="Filter by phone…"
            style={{ ...inputStyle, width: 180, marginBottom: 0 }}
          />
          <input
            value={filterCampaign}
            onChange={e => { setFilterCampaign(e.target.value); setPage(0); }}
            placeholder="Filter by campaign…"
            style={{ ...inputStyle, width: 200, marginBottom: 0 }}
          />
        </div>
      </div>

      {loading ? (
        <p style={{ color: "#6b7280", textAlign: "center", padding: 32 }}>Loading…</p>
      ) : records.length === 0 ? (
        <p style={{ color: "#6b7280", textAlign: "center", padding: 32 }}>No awards found.</p>
      ) : (
        <div style={{ overflowX: "auto" }}>
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ background: "#f9fafb", borderBottom: "1px solid #e5e7eb" }}>
                {["Date / Time", "MSISDN", "Points", "Campaign", "Note", "Awarded By", "Tx Ref"].map(h => (
                  <th key={h} style={{ padding: "10px 12px", textAlign: "left", fontWeight: 600, color: "#374151", whiteSpace: "nowrap" }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {records.map((r, i) => (
                <tr key={r.id} style={{ borderBottom: "1px solid #f3f4f6", background: i % 2 === 0 ? "#fff" : "#fafafa" }}>
                  <td style={tdStyle}>{fmt(r.created_at)}</td>
                  <td style={{ ...tdStyle, fontFamily: "monospace" }}>{r.phone_number}</td>
                  <td style={{ ...tdStyle, fontWeight: 700, color: "#7c3aed" }}>{pts(r.points)}</td>
                  <td style={tdStyle}>{r.campaign || <span style={{ color: "#9ca3af" }}>—</span>}</td>
                  <td style={{ ...tdStyle, maxWidth: 200, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                    {r.note || <span style={{ color: "#9ca3af" }}>—</span>}
                  </td>
                  <td style={tdStyle}>{r.awarded_by_name || r.awarded_by.slice(0, 8)}</td>
                  <td style={{ ...tdStyle, fontFamily: "monospace", fontSize: 11, color: "#6b7280" }}>
                    {r.transaction_id.slice(0, 8)}…
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {pages > 1 && (
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8, marginTop: 16 }}>
          <button onClick={() => setPage(p => Math.max(0, p - 1))} disabled={page === 0} style={pageBtnStyle}>← Prev</button>
          <span style={{ lineHeight: "32px", fontSize: 13, color: "#374151" }}>Page {page + 1} / {pages}</span>
          <button onClick={() => setPage(p => Math.min(pages - 1, p + 1))} disabled={page >= pages - 1} style={pageBtnStyle}>Next →</button>
        </div>
      )}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────
export default function BonusPulsePage() {
  const [lastResult, setLastResult] = useState<BonusPulseAwardResult | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  function handleSuccess(r: BonusPulseAwardResult) {
    setLastResult(r);
    setRefreshKey(k => k + 1); // trigger audit log reload
  }

  return (
    <AdminShell>
      <div style={{ maxWidth: 1100, margin: "0 auto", padding: "24px 16px" }}>
        <div style={{ marginBottom: 24 }}>
          <h1 style={{ margin: "0 0 4px", fontSize: 24, fontWeight: 800, color: "#111" }}>Bonus Pulse Points</h1>
          <p style={{ margin: 0, color: "#6b7280", fontSize: 14 }}>
            Award bonus Pulse Points to individual users as part of campaigns or incentive programmes.
            Every award is recorded in the audit log below.
          </p>
        </div>

        {lastResult && (
          <SuccessBanner result={lastResult} onDismiss={() => setLastResult(null)} />
        )}

        <AwardForm onSuccess={handleSuccess} />
        <AuditLog refresh={refreshKey} />
      </div>
    </AdminShell>
  );
}

// ─── Shared micro-styles ──────────────────────────────────────────────────────
const labelStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 6,
  fontSize: 13,
  fontWeight: 600,
  color: "#374151",
};

const inputStyle: React.CSSProperties = {
  border: "1px solid #d1d5db",
  borderRadius: 8,
  padding: "8px 12px",
  fontSize: 14,
  color: "#111",
  outline: "none",
  width: "100%",
  boxSizing: "border-box",
  marginBottom: 0,
};

const tdStyle: React.CSSProperties = {
  padding: "10px 12px",
  color: "#374151",
  verticalAlign: "middle",
};

const pageBtnStyle: React.CSSProperties = {
  background: "#f3f4f6",
  border: "1px solid #e5e7eb",
  borderRadius: 6,
  padding: "6px 14px",
  fontSize: 13,
  cursor: "pointer",
  color: "#374151",
};
