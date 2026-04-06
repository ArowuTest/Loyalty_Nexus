"use client";
import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { User } from "@/lib/api";

const TIER_COLORS: Record<string, string> = {
  BRONZE: "#f59e0b", SILVER: "#94a3b8", GOLD: "#eab308", PLATINUM: "#a855f7",
};

type ModalState =
  | { type: "adjust"; user: User }
  | { type: "detail"; user: User }
  | null;

export default function UsersPage() {
  const [users, setUsers]       = useState<User[]>([]);
  const [total, setTotal]       = useState(0);
  const [page, setPage]         = useState(1);
  const [search, setSearch]     = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState<string | null>(null);
  const [modal, setModal]       = useState<ModalState>(null);
  const [adjDelta, setAdjDelta] = useState("");
  const [adjReason, setAdjReason] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const r = await adminAPI.getUsers(page, search);
      setUsers(r.users ?? []);
      setTotal(r.total ?? 0);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load users");
    } finally {
      setLoading(false);
    }
  }, [page, search]);

  useEffect(() => { load(); }, [load]);

  const handleSearch = () => { setSearch(searchInput); setPage(1); };

  const handleSuspend = async (u: User) => {
    try {
      await adminAPI.suspendUser(u.id);
      setUsers(us => us.map(x => x.id === u.id ? { ...x, is_active: false } : x));
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Suspend failed");
    }
  };

  const handleUnsuspend = async (u: User) => {
    try {
      await adminAPI.unsuspendUser(u.id);
      setUsers(us => us.map(x => x.id === u.id ? { ...x, is_active: true } : x));
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Unsuspend failed");
    }
  };

  const handleAdjustPoints = async () => {
    if (!modal || modal.type !== "adjust") return;
    const delta = parseInt(adjDelta);
    if (isNaN(delta) || delta === 0) { setError("Enter a non-zero integer delta"); return; }
    if (!adjReason.trim()) { setError("Reason is required"); return; }
    setSubmitting(true);
    try {
      await adminAPI.adjustPoints(modal.user.id, delta, adjReason);
      setModal(null);
      setAdjDelta(""); setAdjReason("");
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Adjust points failed");
    } finally {
      setSubmitting(false);
    }
  };

  const totalPages = Math.ceil(total / 50);

  return (
    <AdminShell>
      <div className="max-w-6xl mx-auto space-y-5 pb-12">
        {/* Header */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>👥 Users</h1>
            <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>{total.toLocaleString()} total users</p>
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

        {/* Search */}
        <div style={{ display: "flex", gap: 8 }}>
          <input value={searchInput} onChange={e => setSearchInput(e.target.value)}
            onKeyDown={e => e.key === "Enter" && handleSearch()}
            placeholder="Search by phone number…"
            style={{ flex: 1, background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13 }} />
          <button onClick={handleSearch}
            style={{ padding: "8px 18px", borderRadius: 8, background: "#5f72f9", border: "none", color: "#fff", fontSize: 13, cursor: "pointer" }}>
            Search
          </button>
          {search && (
            <button onClick={() => { setSearch(""); setSearchInput(""); setPage(1); }}
              style={{ padding: "8px 14px", borderRadius: 8, border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", fontSize: 13, background: "transparent", cursor: "pointer" }}>
              Clear
            </button>
          )}
        </div>

        {/* Table */}
        {loading ? (
          <div style={{ display: "flex", justifyContent: "center", padding: "60px 0" }}>
            <div style={{ width: 32, height: 32, border: "3px solid #5f72f9", borderTopColor: "transparent", borderRadius: "50%", animation: "spin 0.8s linear infinite" }} />
          </div>
        ) : (
          <div className="card" style={{ overflow: "auto" }}>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.15)" }}>
                  {["Phone", "Tier", "Pulse Pts", "Bonus Pts", "Streak", "Status", "Joined", "Actions"].map(h => (
                    <th key={h} style={{ padding: "10px 14px", textAlign: "left", color: "#828cb4", fontWeight: 600 }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {users.length === 0 ? (
                  <tr><td colSpan={8} style={{ padding: "30px 14px", textAlign: "center", color: "#828cb4" }}>No users found</td></tr>
                ) : users.map(u => (
                  <tr key={u.id} style={{ borderBottom: "1px solid rgba(95,114,249,0.07)" }}>
                    <td style={{ padding: "10px 14px", color: "#e2e8ff", fontFamily: "monospace" }}>{u.phone_number}</td>
                    <td style={{ padding: "10px 14px" }}>
                      <span style={{ color: TIER_COLORS[u.tier] || "#e2e8ff", fontWeight: 600, fontSize: 12 }}>{u.tier}</span>
                    </td>
                    <td style={{ padding: "10px 14px", color: "#5f72f9", fontWeight: 600, fontSize: 12 }}>
                      {(u.pulse_points ?? 0).toLocaleString()}
                    </td>
                    <td style={{ padding: "10px 14px", color: "#10b981", fontWeight: 600, fontSize: 12 }}>
                      {(u.bonus_points ?? 0).toLocaleString()}
                    </td>
                    <td style={{ padding: "10px 14px", color: "#828cb4" }}>{u.streak_count}d 🔥</td>
                    <td style={{ padding: "10px 14px" }}>
                      <span style={{ color: u.is_active ? "#10b981" : "#f43f5e", fontSize: 12, fontWeight: 600 }}>
                        {u.is_active ? "● Active" : "● Suspended"}
                      </span>
                    </td>
                    <td style={{ padding: "10px 14px", color: "#828cb4", fontSize: 12 }}>
                      {new Date(u.created_at).toLocaleDateString("en-NG")}
                    </td>
                    <td style={{ padding: "10px 14px" }}>
                      <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                        <button onClick={() => setModal({ type: "detail", user: u })}
                          style={{ padding: "3px 9px", borderRadius: 6, border: "1px solid rgba(95,114,249,0.3)", color: "#5f72f9", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                          Detail
                        </button>
                        <button onClick={() => { setModal({ type: "adjust", user: u }); setAdjDelta(""); setAdjReason(""); }}
                          style={{ padding: "3px 9px", borderRadius: 6, border: "1px solid rgba(16,185,129,0.3)", color: "#10b981", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                          ± Points
                        </button>
                        {u.is_active ? (
                          <button onClick={() => handleSuspend(u)}
                            style={{ padding: "3px 9px", borderRadius: 6, border: "1px solid rgba(239,68,68,0.3)", color: "#fca5a5", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                            Suspend
                          </button>
                        ) : (
                          <button onClick={() => handleUnsuspend(u)}
                            style={{ padding: "3px 9px", borderRadius: 6, border: "1px solid rgba(16,185,129,0.3)", color: "#10b981", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                            Unsuspend
                          </button>
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
          <div className="card" style={{ width: "min(460px, 95vw)", padding: 28 }}>

            {modal.type === "detail" && (
              <>
                <h2 style={{ fontSize: 16, fontWeight: 700, color: "#e2e8ff", marginBottom: 16 }}>👤 User Details</h2>
                {[
                  ["ID",           modal.user.id],
                  ["Phone",        modal.user.phone_number],
                  ["Tier",         modal.user.tier],
                  ["Pulse Points", (modal.user.pulse_points ?? 0).toLocaleString()],
                  ["Bonus Points", (modal.user.bonus_points ?? 0).toLocaleString()],
                  ["Streak",       `${modal.user.streak_count} days`],
                  ["Status",       modal.user.is_active ? "Active" : "Suspended"],
                  ["Joined",       new Date(modal.user.created_at).toLocaleString("en-NG")],
                ].map(([k, v]) => (
                  <div key={k} style={{ display: "flex", gap: 12, marginBottom: 8 }}>
                    <span style={{ width: 80, fontSize: 12, color: "#828cb4", flexShrink: 0 }}>{k}</span>
                    <span style={{ fontSize: 12, color: "#e2e8ff", wordBreak: "break-all" }}>{v}</span>
                  </div>
                ))}
                <button onClick={() => setModal(null)}
                  style={{ marginTop: 16, width: "100%", padding: "10px", borderRadius: 8, background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
                  Close
                </button>
              </>
            )}

            {modal.type === "adjust" && (
              <>
                <h2 style={{ fontSize: 16, fontWeight: 700, color: "#e2e8ff", marginBottom: 4 }}>± Adjust Points</h2>
                <p style={{ fontSize: 13, color: "#828cb4", marginBottom: 16 }}>
                  Adjusting Pulse Points for <strong style={{ color: "#e2e8ff" }}>{modal.user.phone_number}</strong>
                </p>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 6 }}>
                  Delta (positive to add, negative to deduct) <span style={{ color: "#ef4444" }}>*</span>
                </label>
                <input type="number" value={adjDelta} onChange={e => setAdjDelta(e.target.value)}
                  placeholder="e.g. 100 or -50"
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, marginBottom: 12, boxSizing: "border-box" }} />
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 6 }}>
                  Reason <span style={{ color: "#ef4444" }}>*</span>
                </label>
                <input value={adjReason} onChange={e => setAdjReason(e.target.value)}
                  placeholder="e.g. Compensation for failed spin"
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, marginBottom: 16, boxSizing: "border-box" }} />
                <div style={{ display: "flex", gap: 10 }}>
                  <button onClick={() => setModal(null)}
                    style={{ flex: 1, padding: "10px", borderRadius: 8, background: "transparent", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
                    Cancel
                  </button>
                  <button onClick={handleAdjustPoints} disabled={submitting}
                    style={{ flex: 1, padding: "10px", borderRadius: 8, background: "#5f72f9", border: "none", color: "#fff", fontWeight: 600, cursor: "pointer", opacity: submitting ? 0.6 : 1 }}>
                    {submitting ? "Saving…" : "Apply Adjustment"}
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
