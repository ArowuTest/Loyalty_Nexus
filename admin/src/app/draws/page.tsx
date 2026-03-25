"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState } from "react";
import adminAPI, { Draw, CreateDrawPayload, DrawWinner } from "@/lib/api";

const STATUS_COLORS: Record<string, string> = {
  scheduled: "bg-blue-100 text-blue-800",
  completed: "bg-green-100 text-green-800",
  cancelled: "bg-red-100 text-red-800",
};

function fmtDate(iso: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("en-NG", {
    day: "2-digit", month: "short", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

function fmtNaira(kobo: number) {
  return "₦" + (kobo / 100).toLocaleString("en-NG");
}

export default function DrawsPage() {
  const [draws, setDraws]       = useState<Draw[]>([]);
  const [loading, setLoading]   = useState(true);
  const [executing, setExec]    = useState<string | null>(null);
  const [showCreate, setCreate] = useState(false);
  const [winners, setWinners]   = useState<{ drawId: string; data: DrawWinner[] } | null>(null);
  const [form, setForm]         = useState<CreateDrawPayload>({
    name: "", prize_pool_kobo: 0, draw_date: "", recurrence: "once",
  });

  const load = () => adminAPI.getDraws().then(r => setDraws(r.draws)).finally(() => setLoading(false));
  useEffect(() => { load(); }, []);

  const doCreate = async () => {
    await adminAPI.createDraw(form);
    setCreate(false);
    load();
  };

  const doExecute = async (id: string) => {
    if (!confirm("Execute this draw now? This action cannot be undone.")) return;
    setExec(id);
    try { await adminAPI.executeDraw(id); load(); }
    catch (e: unknown) { alert((e as Error).message); }
    finally { setExec(null); }
  };

  const showDrawWinners = async (id: string) => {
    const r = await adminAPI.getDrawWinners(id);
    setWinners({ drawId: id, data: r.winners });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Draw Management</h1>
          <p className="text-sm text-gray-500 mt-1">Schedule, execute, and review prize draws</p>
        </div>
        <button onClick={() => setCreate(true)}
          className="bg-indigo-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-indigo-700">
          + Schedule Draw
        </button>
      </div>

      {loading ? (
        <div className="flex justify-center py-20"><div className="animate-spin h-8 w-8 border-4 border-indigo-600 border-t-transparent rounded-full"/></div>
      ) : (
        <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>{["Name","Prize Pool","Draw Date","Recurrence","Status","Entries","Actions"].map(h =>
                <th key={h} className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">{h}</th>)}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {draws.length === 0 && (
                <tr><td colSpan={7} className="text-center py-10 text-gray-400">No draws yet</td></tr>
              )}
              {draws.map(d => (
                <tr key={d.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 font-medium text-gray-900">{d.name}</td>
                  <td className="px-4 py-3 text-indigo-700 font-semibold">{fmtNaira(d.prize_pool_kobo)}</td>
                  <td className="px-4 py-3 text-gray-600">{fmtDate(d.draw_date)}</td>
                  <td className="px-4 py-3 capitalize text-gray-600">{d.recurrence}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-1 rounded-full text-xs font-medium ${STATUS_COLORS[d.status] ?? "bg-gray-100 text-gray-600"}`}>
                      {d.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-600">{(d.entry_count ?? 0).toLocaleString()}</td>
                  <td className="px-4 py-3 flex gap-2">
                    {d.status === "scheduled" && (
                      <button disabled={executing === d.id}
                        onClick={() => doExecute(d.id)}
                        className="px-3 py-1 bg-green-600 text-white rounded text-xs hover:bg-green-700 disabled:opacity-50">
                        {executing === d.id ? "Running…" : "Execute"}
                      </button>
                    )}
                    {d.status === "completed" && (
                      <button onClick={() => showDrawWinners(d.id)}
                        className="px-3 py-1 bg-indigo-600 text-white rounded text-xs hover:bg-indigo-700">
                        Winners
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Draw Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md shadow-xl">
            <h2 className="text-xl font-bold mb-4">Schedule New Draw</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Draw Name</label>
                <input value={form.name} onChange={e => setForm({...form, name: e.target.value})}
                  placeholder="e.g. March Monthly Draw"
                  className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Prize Pool (₦)</label>
                <input type="number" value={form.prize_pool_kobo / 100}
                  onChange={e => setForm({...form, prize_pool_kobo: Math.round(parseFloat(e.target.value || "0") * 100)})}
                  placeholder="e.g. 500000"
                  className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Draw Date &amp; Time</label>
                <input type="datetime-local" value={form.draw_date}
                  onChange={e => setForm({...form, draw_date: e.target.value})}
                  className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Recurrence</label>
                <select value={form.recurrence}
                  onChange={e => setForm({...form, recurrence: e.target.value as "once"|"weekly"|"monthly"})}
                  className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none">
                  <option value="once">One-off</option>
                  <option value="weekly">Weekly</option>
                  <option value="monthly">Monthly</option>
                </select>
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <button onClick={() => setCreate(false)} className="flex-1 border rounded-lg py-2 text-sm text-gray-600 hover:bg-gray-50">
                Cancel
              </button>
              <button onClick={doCreate}
                disabled={!form.name || !form.draw_date}
                className="flex-1 bg-indigo-600 text-white rounded-lg py-2 text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">
                Schedule
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Winners Modal */}
      {winners && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg shadow-xl max-h-[80vh] flex flex-col">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold">Draw Winners</h2>
              <button onClick={() => setWinners(null)} className="text-gray-400 hover:text-gray-600 text-xl">✕</button>
            </div>
            <div className="overflow-y-auto flex-1">
              {winners.data.length === 0 ? (
                <p className="text-center py-8 text-gray-400">No winners recorded</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50">
                    <tr>{["Rank","Phone","Prize","Date"].map(h =>
                      <th key={h} className="text-left px-3 py-2 text-xs font-semibold text-gray-500">{h}</th>)}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {winners.data.map(w => (
                      <tr key={w.id}>
                        <td className="px-3 py-2 font-bold text-indigo-600">#{w.rank}</td>
                        <td className="px-3 py-2 font-mono text-gray-700">{w.phone_number}</td>
                        <td className="px-3 py-2 text-gray-600">{w.prize_label}</td>
                        <td className="px-3 py-2 text-gray-400 text-xs">{fmtDate(w.created_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
