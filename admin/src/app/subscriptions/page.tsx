"use client";
import { useState, useEffect, useCallback } from "react";
import adminAPI, { SubscriptionUser, UpdateSubPayload } from "@/lib/api";
import AdminShell from "@/components/layout/AdminShell";

const STATUS_COLORS: Record<string, string> = {
  ACTIVE:    "bg-green-500/20 text-green-300",
  FREE:      "bg-gray-500/20 text-gray-300",
  GRACE:     "bg-yellow-500/20 text-yellow-300",
  SUSPENDED: "bg-red-500/20 text-red-300",
  BANNED:    "bg-red-700/20 text-red-400",
};

const STATUSES = ["ACTIVE","FREE","GRACE","SUSPENDED","BANNED"];

export default function SubscriptionsPage() {
  const [users, setUsers]         = useState<SubscriptionUser[]>([]);
  const [loading, setLoading]     = useState(true);
  const [filter, setFilter]       = useState("");
  const [page, setPage]           = useState(0);
  const [editing, setEditing]     = useState<SubscriptionUser | null>(null);
  const [editForm, setEditForm]   = useState<UpdateSubPayload>({ status: "ACTIVE" });
  const [saving, setSaving]       = useState(false);
  const [msg, setMsg]             = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try { setUsers((await adminAPI.getSubscriptions(page, filter)).users ?? []); }
    catch { setUsers([]); }
    finally { setLoading(false); }
  }, [page, filter]);

  useEffect(() => { load(); }, [load]);

  function openEdit(u: SubscriptionUser) {
    setEditing(u);
    setEditForm({
      status: u.subscription_status,
      expires_at: u.subscription_expires_at ?? "",
      note: "",
    });
    setMsg("");
  }

  async function save() {
    if (!editing) return;
    setSaving(true); setMsg("");
    try {
      await adminAPI.updateSubscription(editing.id, editForm);
      setMsg("✅ Updated");
      load();
      setTimeout(() => { setEditing(null); setMsg(""); }, 1200);
    } catch (e: unknown) {
      setMsg("❌ " + (e as Error).message);
    } finally { setSaving(false); }
  }

  const total = users.length;

  return (
    <AdminShell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-white">Subscription Management</h1>
          <div className="flex gap-2">
            <select
              className="bg-[#1a1f35] border border-white/10 rounded-lg px-3 py-1.5 text-sm text-white"
              value={filter}
              onChange={e => { setFilter(e.target.value); setPage(0); }}
            >
              <option value="">All Statuses</option>
              {STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
            <button onClick={load} className="bg-[#1a1f35] border border-white/10 rounded-lg px-3 py-1.5 text-sm text-gray-400 hover:text-white transition-colors">↻</button>
          </div>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
          {STATUSES.map(s => {
            const cnt = users.filter(u => u.subscription_status === s).length;
            return (
              <div key={s} className="bg-[#1a1f35] rounded-xl border border-white/10 p-4">
                <p className="text-xs text-gray-400">{s}</p>
                <p className="text-2xl font-bold text-white mt-1">{cnt}</p>
              </div>
            );
          })}
        </div>

        {/* Table */}
        <div className="bg-[#1a1f35] rounded-xl border border-white/10 overflow-hidden">
          <div className="px-6 py-3 border-b border-white/10 text-xs text-gray-400">
            Showing {total} users (page {page + 1})
          </div>
          {loading ? (
            <div className="p-10 text-center text-gray-400">Loading…</div>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-white/5">
                <tr>
                  {["Phone","Tier","Status","Expires","Joined","Actions"].map(h => (
                    <th key={h} className="px-5 py-3 text-left text-xs font-semibold text-gray-400 uppercase tracking-wider">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {users.map(u => (
                  <tr key={u.id} className="hover:bg-white/5 transition-colors">
                    <td className="px-5 py-3 text-white font-mono text-xs">{u.phone_number}</td>
                    <td className="px-5 py-3">
                      <span className="bg-indigo-500/20 text-indigo-300 px-2 py-0.5 rounded text-xs">{u.tier}</span>
                    </td>
                    <td className="px-5 py-3">
                      <span className={`px-2 py-0.5 rounded text-xs ${STATUS_COLORS[u.subscription_status] ?? "bg-gray-500/20 text-gray-300"}`}>
                        {u.subscription_status}
                      </span>
                    </td>
                    <td className="px-5 py-3 text-gray-400 text-xs">
                      {u.subscription_expires_at ? new Date(u.subscription_expires_at).toLocaleDateString() : "—"}
                    </td>
                    <td className="px-5 py-3 text-gray-400 text-xs">{new Date(u.created_at).toLocaleDateString()}</td>
                    <td className="px-5 py-3">
                      <button
                        onClick={() => openEdit(u)}
                        className="text-xs text-[#4A56EE] hover:text-indigo-300 transition-colors font-medium"
                      >Edit</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          <div className="px-6 py-3 border-t border-white/10 flex gap-2">
            <button
              disabled={page === 0}
              onClick={() => setPage(p => p - 1)}
              className="px-3 py-1 text-xs rounded bg-white/5 hover:bg-white/10 disabled:opacity-40 text-white transition-colors"
            >← Prev</button>
            <button
              disabled={users.length < 50}
              onClick={() => setPage(p => p + 1)}
              className="px-3 py-1 text-xs rounded bg-white/5 hover:bg-white/10 disabled:opacity-40 text-white transition-colors"
            >Next →</button>
          </div>
        </div>
      </div>

      {/* Edit modal */}
      {editing && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
          <div className="bg-[#1a1f35] rounded-2xl border border-white/10 p-6 w-full max-w-md space-y-4">
            <h3 className="text-lg font-bold text-white">Edit Subscription</h3>
            <p className="text-sm text-gray-400 font-mono">{editing.phone_number}</p>

            <div>
              <label className="text-xs text-gray-400 mb-1 block">Status</label>
              <select
                className="w-full bg-[#0f1628] border border-white/10 rounded-lg px-3 py-2 text-white text-sm"
                value={editForm.status}
                onChange={e => setEditForm(f => ({ ...f, status: e.target.value }))}
              >
                {STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>

            <div>
              <label className="text-xs text-gray-400 mb-1 block">Expires At (ISO datetime)</label>
              <input
                type="datetime-local"
                className="w-full bg-[#0f1628] border border-white/10 rounded-lg px-3 py-2 text-white text-sm"
                value={editForm.expires_at?.slice(0,16) ?? ""}
                onChange={e => setEditForm(f => ({ ...f, expires_at: e.target.value ? e.target.value + ":00Z" : "" }))}
              />
            </div>

            <div>
              <label className="text-xs text-gray-400 mb-1 block">Note (audit log)</label>
              <input
                className="w-full bg-[#0f1628] border border-white/10 rounded-lg px-3 py-2 text-white text-sm"
                value={editForm.note ?? ""}
                onChange={e => setEditForm(f => ({ ...f, note: e.target.value }))}
                placeholder="e.g. Manual upgrade by admin"
              />
            </div>

            {msg && <p className={`text-sm ${msg.startsWith("✅") ? "text-green-400" : "text-red-400"}`}>{msg}</p>}

            <div className="flex gap-3 pt-2">
              <button
                onClick={save}
                disabled={saving}
                className="flex-1 bg-[#4A56EE] hover:bg-[#3a46de] disabled:opacity-50 text-white font-semibold py-2 rounded-lg text-sm transition-colors"
              >{saving ? "Saving…" : "Save"}</button>
              <button
                onClick={() => setEditing(null)}
                className="flex-1 bg-white/5 hover:bg-white/10 text-white py-2 rounded-lg text-sm transition-colors"
              >Cancel</button>
            </div>
          </div>
        </div>
      )}
    </AdminShell>
  );
}
