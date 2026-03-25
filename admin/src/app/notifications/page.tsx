"use client";
import { useState, useEffect, useCallback } from "react";
import adminAPI, { Broadcast, BroadcastPayload } from "@/lib/api";
import AdminShell from "@/components/layout/AdminShell";

const NOTIF_TYPES = ["system","marketing","spin_win","draw_result","subscription_warn","wars_result","studio_ready"];
const TARGETS = [
  { value: "all",                label: "All Users" },
  { value: "active_subscribers", label: "Active Subscribers" },
  { value: "free_tier",          label: "Free Tier Only" },
  { value: "phone_list",         label: "Phone List (CSV)" },
];

export default function NotificationsPage() {
  const [history, setHistory]   = useState<Broadcast[]>([]);
  const [loading, setLoading]   = useState(true);
  const [sending, setSending]   = useState(false);
  const [success, setSuccess]   = useState("");
  const [error, setError]       = useState("");
  const [form, setForm]         = useState<BroadcastPayload>({
    title: "", body: "", type: "system", target: "all", deep_link: "",
  });
  const [phoneList, setPhoneList] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try { setHistory((await adminAPI.getNotificationHistory()).broadcasts ?? []); }
    catch { setHistory([]); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  async function send() {
    if (!form.title.trim() || !form.body.trim()) { setError("Title and body are required"); return; }
    setSending(true); setError(""); setSuccess("");
    try {
      const payload: BroadcastPayload = { ...form };
      if (form.target === "phone_list") {
        payload.phone_list = phoneList.split(/[\n,]+/).map(s => s.trim()).filter(Boolean);
        if (!payload.phone_list.length) { setError("Phone list is empty"); setSending(false); return; }
      }
      await adminAPI.broadcastNotification(payload);
      setSuccess("✅ Broadcast queued successfully");
      setForm({ title: "", body: "", type: "system", target: "all", deep_link: "" });
      setPhoneList("");
      load();
    } catch (e: unknown) {
      setError((e as Error).message);
    } finally { setSending(false); }
  }

  return (
    <AdminShell>
      <div className="space-y-6">
        <h1 className="text-2xl font-bold text-white">Notification Broadcast</h1>

        {/* Compose form */}
        <div className="bg-[#1a1f35] rounded-xl border border-white/10 p-6">
          <h2 className="text-lg font-semibold text-white mb-4">Compose Message</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="text-xs text-gray-400 mb-1 block">Title *</label>
              <input
                className="w-full bg-[#0f1628] border border-white/10 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-[#4A56EE]"
                value={form.title}
                onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
                placeholder="Notification title"
              />
            </div>
            <div>
              <label className="text-xs text-gray-400 mb-1 block">Type</label>
              <select
                className="w-full bg-[#0f1628] border border-white/10 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-[#4A56EE]"
                value={form.type}
                onChange={e => setForm(f => ({ ...f, type: e.target.value }))}
              >
                {NOTIF_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div className="md:col-span-2">
              <label className="text-xs text-gray-400 mb-1 block">Body *</label>
              <textarea
                className="w-full bg-[#0f1628] border border-white/10 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-[#4A56EE] resize-none"
                rows={3}
                value={form.body}
                onChange={e => setForm(f => ({ ...f, body: e.target.value }))}
                placeholder="Notification body text..."
              />
            </div>
            <div>
              <label className="text-xs text-gray-400 mb-1 block">Target Audience</label>
              <select
                className="w-full bg-[#0f1628] border border-white/10 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-[#4A56EE]"
                value={form.target}
                onChange={e => setForm(f => ({ ...f, target: e.target.value as BroadcastPayload["target"] }))}
              >
                {TARGETS.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
              </select>
            </div>
            <div>
              <label className="text-xs text-gray-400 mb-1 block">Deep Link (optional)</label>
              <input
                className="w-full bg-[#0f1628] border border-white/10 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-[#4A56EE]"
                value={form.deep_link ?? ""}
                onChange={e => setForm(f => ({ ...f, deep_link: e.target.value }))}
                placeholder="/spins or /draws/uuid"
              />
            </div>
            {form.target === "phone_list" && (
              <div className="md:col-span-2">
                <label className="text-xs text-gray-400 mb-1 block">Phone Numbers (one per line or comma-separated)</label>
                <textarea
                  className="w-full bg-[#0f1628] border border-white/10 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-[#4A56EE] font-mono resize-none"
                  rows={4}
                  value={phoneList}
                  onChange={e => setPhoneList(e.target.value)}
                  placeholder="2348031234567&#10;2349012345678"
                />
              </div>
            )}
          </div>
          {error   && <p className="mt-3 text-sm text-red-400">{error}</p>}
          {success && <p className="mt-3 text-sm text-green-400">{success}</p>}
          <button
            onClick={send}
            disabled={sending}
            className="mt-4 bg-[#4A56EE] hover:bg-[#3a46de] disabled:opacity-50 text-white font-semibold px-6 py-2 rounded-lg text-sm transition-colors"
          >
            {sending ? "Sending…" : "🚀 Send Broadcast"}
          </button>
        </div>

        {/* Broadcast history */}
        <div className="bg-[#1a1f35] rounded-xl border border-white/10 overflow-hidden">
          <div className="px-6 py-4 border-b border-white/10 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-white">Broadcast History</h2>
            <button onClick={load} className="text-xs text-gray-400 hover:text-white transition-colors">↻ Refresh</button>
          </div>
          {loading ? (
            <div className="p-8 text-center text-gray-400">Loading…</div>
          ) : history.length === 0 ? (
            <div className="p-8 text-center text-gray-400">No broadcasts yet</div>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-white/5">
                <tr>
                  {["Title","Type","Target","Sent","Date"].map(h => (
                    <th key={h} className="px-6 py-3 text-left text-xs font-semibold text-gray-400 uppercase tracking-wider">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {history.map(b => (
                  <tr key={b.id} className="hover:bg-white/5 transition-colors">
                    <td className="px-6 py-3 text-white font-medium">{b.title}</td>
                    <td className="px-6 py-3"><span className="bg-indigo-500/20 text-indigo-300 px-2 py-0.5 rounded text-xs">{b.type}</span></td>
                    <td className="px-6 py-3 text-gray-400">{b.target}</td>
                    <td className="px-6 py-3 text-gray-400">{b.sent_count.toLocaleString()}</td>
                    <td className="px-6 py-3 text-gray-400">{new Date(b.created_at).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </AdminShell>
  );
}
