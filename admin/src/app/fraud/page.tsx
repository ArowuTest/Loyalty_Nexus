"use client";
import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { FraudEvent } from "@/lib/api";

const SEV_COLOR: Record<string, string> = {
  HIGH: "#ef4444", MEDIUM: "#f59e0b", LOW: "#10b981",
};

export default function FraudPage() {
  const [events, setEvents]       = useState<FraudEvent[]>([]);
  const [filter, setFilter]       = useState<"all" | "open" | "resolved">("open");
  const [loading, setLoading]     = useState(true);
  const [error, setError]         = useState<string | null>(null);
  const [resolving, setResolving] = useState<string | null>(null);
  const [noteModal, setNoteModal] = useState<FraudEvent | null>(null);
  const [noteText, setNoteText]   = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const r = await adminAPI.getFraudEvents();
      setEvents(r.events ?? []);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load fraud events");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleResolve = async (ev: FraudEvent, notes = "") => {
    setResolving(ev.id);
    try {
      await adminAPI.resolveFraudEvent(ev.id, notes);
      setEvents(es => es.map(e => e.id === ev.id ? { ...e, resolved: true, notes } : e));
      setNoteModal(null);
      setNoteText("");
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Resolve failed");
    } finally {
      setResolving(null);
    }
  };

  const filtered = events.filter(e => {
    if (filter === "open")     return !e.resolved;
    if (filter === "resolved") return e.resolved;
    return true;
  });

  const openCount     = events.filter(e => !e.resolved).length;
  const resolvedCount = events.filter(e => e.resolved).length;

  return (
    <AdminShell>
      <div className="max-w-5xl mx-auto space-y-5 pb-12">
        {/* Header */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>🛡 Fraud Events</h1>
            <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
              {openCount} open · {resolvedCount} resolved
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

        {/* Filter Tabs */}
        <div style={{ display: "flex", gap: 6 }}>
          {(["open", "resolved", "all"] as const).map(f => (
            <button key={f} onClick={() => setFilter(f)}
              style={{ padding: "7px 16px", borderRadius: 8, border: `1px solid ${filter === f ? "rgba(95,114,249,0.6)" : "rgba(95,114,249,0.15)"}`, color: filter === f ? "#e2e8ff" : "#828cb4", background: filter === f ? "rgba(95,114,249,0.15)" : "transparent", fontSize: 13, cursor: "pointer", textTransform: "capitalize" }}>
              {f} {f === "open" ? `(${openCount})` : f === "resolved" ? `(${resolvedCount})` : `(${events.length})`}
            </button>
          ))}
        </div>

        {loading ? (
          <div style={{ display: "flex", justifyContent: "center", padding: "60px 0" }}>
            <div style={{ width: 32, height: 32, border: "3px solid #5f72f9", borderTopColor: "transparent", borderRadius: "50%", animation: "spin 0.8s linear infinite" }} />
          </div>
        ) : filtered.length === 0 ? (
          <div className="card" style={{ padding: 40, textAlign: "center", color: "#828cb4" }}>
            {filter === "open" ? "✅ No open fraud events" : "No events found"}
          </div>
        ) : (
          <div className="card" style={{ overflow: "auto" }}>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.15)" }}>
                  {["Phone / User", "Event Type", "Severity", "Time", "Status", "Actions"].map(h => (
                    <th key={h} style={{ padding: "10px 14px", textAlign: "left", color: "#828cb4", fontWeight: 600 }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {filtered.map(ev => (
                  <tr key={ev.id} style={{ borderBottom: "1px solid rgba(95,114,249,0.07)", opacity: ev.resolved ? 0.55 : 1 }}>
                    <td style={{ padding: "10px 14px", fontFamily: "monospace", color: "#e2e8ff" }}>
                      {ev.msisdn || ev.user_id.slice(0, 8) + "…"}
                    </td>
                    <td style={{ padding: "10px 14px", color: "#c4cde8" }}>{ev.event_type}</td>
                    <td style={{ padding: "10px 14px" }}>
                      <span style={{ color: SEV_COLOR[ev.severity] || "#e2e8ff", fontWeight: 700, fontSize: 12 }}>
                        {ev.severity}
                      </span>
                    </td>
                    <td style={{ padding: "10px 14px", color: "#828cb4", fontSize: 12, whiteSpace: "nowrap" }}>
                      {new Date(ev.created_at).toLocaleString("en-NG")}
                    </td>
                    <td style={{ padding: "10px 14px" }}>
                      <span style={{ color: ev.resolved ? "#10b981" : "#ef4444", fontSize: 12, fontWeight: 600 }}>
                        {ev.resolved ? "● Resolved" : "● Open"}
                      </span>
                    </td>
                    <td style={{ padding: "10px 14px" }}>
                      {!ev.resolved && (
                        <button
                          onClick={() => { setNoteModal(ev); setNoteText(""); }}
                          disabled={resolving === ev.id}
                          style={{ padding: "4px 10px", borderRadius: 6, border: "1px solid rgba(16,185,129,0.3)", color: "#10b981", background: "transparent", fontSize: 11, cursor: "pointer", opacity: resolving === ev.id ? 0.5 : 1 }}>
                          {resolving === ev.id ? "…" : "✓ Resolve"}
                        </button>
                      )}
                      {ev.resolved && ev.notes && (
                        <span style={{ fontSize: 11, color: "#828cb4", fontStyle: "italic" }}>{ev.notes.slice(0, 40)}{ev.notes.length > 40 ? "…" : ""}</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Resolve Modal */}
      {noteModal && (
        <div style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 50 }}>
          <div className="card" style={{ width: "min(440px, 95vw)", padding: 28 }}>
            <h2 style={{ fontSize: 16, fontWeight: 700, color: "#10b981", marginBottom: 4 }}>✓ Resolve Fraud Event</h2>
            <p style={{ fontSize: 13, color: "#828cb4", marginBottom: 16 }}>
              {noteModal.event_type} — {noteModal.msisdn || noteModal.user_id.slice(0,8) + "…"}
            </p>
            <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 6 }}>Resolution Notes (optional)</label>
            <textarea value={noteText} onChange={e => setNoteText(e.target.value)} rows={3}
              placeholder="e.g. Verified legitimate activity, no action required"
              style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, marginBottom: 16, resize: "vertical", boxSizing: "border-box" }} />
            <div style={{ display: "flex", gap: 10 }}>
              <button onClick={() => setNoteModal(null)}
                style={{ flex: 1, padding: "10px", borderRadius: 8, background: "transparent", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
                Cancel
              </button>
              <button onClick={() => handleResolve(noteModal, noteText)} disabled={!!resolving}
                style={{ flex: 1, padding: "10px", borderRadius: 8, background: "#10b981", border: "none", color: "#fff", fontWeight: 600, cursor: "pointer", opacity: resolving ? 0.6 : 1 }}>
                {resolving ? "Resolving…" : "Mark Resolved"}
              </button>
            </div>
          </div>
        </div>
      )}

      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </AdminShell>
  );
}
