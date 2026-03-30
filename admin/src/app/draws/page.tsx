"use client";
import AdminShell from "@/components/layout/AdminShell";
import { useEffect, useState } from "react";
import adminAPI, {
  Draw, CreateDrawPayload, DrawWinner,
  DrawSchedule, CreateDrawSchedulePayload,
} from "@/lib/api";

const STATUS_COLORS: Record<string, string> = {
  scheduled:  "bg-blue-100 text-blue-800",
  completed:  "bg-green-100 text-green-800",
  cancelled:  "bg-red-100 text-red-800",
};
const DAYS = ["Sun","Mon","Tue","Wed","Thu","Fri","Sat"];

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

const BLANK_DRAW: CreateDrawPayload = { name: "", prize_pool_kobo: 0, draw_date: "", recurrence: "once" };
const BLANK_SCHED: CreateDrawSchedulePayload = {
  draw_name: "", draw_type: "WEEKLY",
  draw_day_of_week: 5, draw_time_wat: "20:00:00",
  window_open_dow: 5, window_open_time: "00:00:00",
  window_close_dow: 5, window_close_time: "19:59:59",
  cutoff_hour_utc: 19, sort_order: 1, is_active: true,
};

type DrawModal = { mode: "create" } | { mode: "edit"; draw: Draw };
type SchedModal = { mode: "create" } | { mode: "edit"; sched: DrawSchedule };

export default function DrawsPage() {
  const [tab, setTab]           = useState<"draws"|"schedules">("draws");
  const [draws, setDraws]       = useState<Draw[]>([]);
  const [schedules, setScheds]  = useState<DrawSchedule[]>([]);
  const [loading, setLoading]   = useState(true);
  const [executing, setExec]    = useState<string | null>(null);
  const [drawModal, setDrawModal] = useState<DrawModal | null>(null);
  const [schedModal, setSchedModal] = useState<SchedModal | null>(null);
  const [drawForm, setDrawForm] = useState<CreateDrawPayload>(BLANK_DRAW);
  const [schedForm, setSchedForm] = useState<CreateDrawSchedulePayload>(BLANK_SCHED);
  const [winners, setWinners]   = useState<{ drawId: string; data: DrawWinner[] } | null>(null);
  const [saving, setSaving]     = useState(false);
  const [err, setErr]           = useState("");

  const loadDraws    = () => adminAPI.getDraws().then(r => setDraws(r.draws));
  const loadSchedules = () => adminAPI.getDrawSchedules().then(r => setScheds(r.schedules));

  const load = async () => {
    setLoading(true);
    await Promise.all([loadDraws(), loadSchedules()]);
    setLoading(false);
  };
  useEffect(() => { load(); }, []);

  /* ── Draw CRUD ── */
  const openCreate = () => { setDrawForm(BLANK_DRAW); setDrawModal({ mode: "create" }); setErr(""); };
  const openEdit   = (d: Draw) => {
    setDrawForm({ name: d.name, prize_pool_kobo: d.prize_pool_kobo, draw_date: d.draw_date.slice(0,16), recurrence: d.recurrence as "once"|"weekly"|"monthly" });
    setDrawModal({ mode: "edit", draw: d });
    setErr("");
  };
  const saveDraw = async () => {
    setSaving(true); setErr("");
    try {
      if (drawModal?.mode === "create") await adminAPI.createDraw(drawForm);
      else if (drawModal?.mode === "edit") await adminAPI.updateDraw(drawModal.draw.id, drawForm);
      setDrawModal(null); await loadDraws();
    } catch (e: unknown) { setErr((e as Error).message); }
    finally { setSaving(false); }
  };

  const doExecute = async (id: string) => {
    if (!confirm("Execute this draw now? This action cannot be undone.")) return;
    setExec(id);
    try { await adminAPI.executeDraw(id); await loadDraws(); }
    catch (e: unknown) { alert((e as Error).message); }
    finally { setExec(null); }
  };

  const showWinners = async (id: string) => {
    const r = await adminAPI.getDrawWinners(id);
    setWinners({ drawId: id, data: r.winners });
  };

  const doExport = async (id: string) => {
    try {
      const csv = await adminAPI.exportDrawEntries(id);
      const blob = new Blob([csv as unknown as string], { type: "text/csv" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a"); a.href = url; a.download = `draw_entries_${id}.csv`; a.click();
    } catch (e: unknown) { alert((e as Error).message); }
  };

  /* ── Schedule CRUD ── */
  const openSchedCreate = () => { setSchedForm(BLANK_SCHED); setSchedModal({ mode: "create" }); setErr(""); };
  const openSchedEdit   = (s: DrawSchedule) => {
    setSchedForm({
      draw_name: s.draw_name, draw_type: s.draw_type,
      draw_day_of_week: s.draw_day_of_week, draw_time_wat: s.draw_time_wat,
      window_open_dow: s.window_open_dow, window_open_time: s.window_open_time,
      window_close_dow: s.window_close_dow, window_close_time: s.window_close_time,
      cutoff_hour_utc: s.cutoff_hour_utc, sort_order: s.sort_order, is_active: s.is_active,
    });
    setSchedModal({ mode: "edit", sched: s }); setErr("");
  };
  const saveSched = async () => {
    setSaving(true); setErr("");
    try {
      if (schedModal?.mode === "create") await adminAPI.createDrawSchedule(schedForm);
      else if (schedModal?.mode === "edit") await adminAPI.updateDrawSchedule(schedModal.sched.id, schedForm);
      setSchedModal(null); await loadSchedules();
    } catch (e: unknown) { setErr((e as Error).message); }
    finally { setSaving(false); }
  };
  const deleteSched = async (id: string) => {
    if (!confirm("Delete this draw schedule rule?")) return;
    try { await adminAPI.deleteDrawSchedule(id); await loadSchedules(); }
    catch (e: unknown) { alert((e as Error).message); }
  };

  return (
    <AdminShell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Draw Management</h1>
            <p className="text-sm text-gray-500 mt-1">Schedule, execute, and configure prize draws</p>
          </div>
          <div className="flex gap-2">
            {tab === "draws" && (
              <button onClick={openCreate}
                className="bg-indigo-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-indigo-700">
                + Schedule Draw
              </button>
            )}
            {tab === "schedules" && (
              <button onClick={openSchedCreate}
                className="bg-indigo-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-indigo-700">
                + Add Window Rule
              </button>
            )}
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 bg-gray-100 p-1 rounded-lg w-fit">
          {(["draws","schedules"] as const).map(t => (
            <button key={t} onClick={() => setTab(t)}
              className={`px-4 py-1.5 rounded-md text-sm font-medium transition-colors ${tab === t ? "bg-white shadow text-gray-900" : "text-gray-500 hover:text-gray-700"}`}>
              {t === "draws" ? "Draws" : "Window Rules"}
            </button>
          ))}
        </div>

        {loading ? (
          <div className="flex justify-center py-20">
            <div className="animate-spin h-8 w-8 border-4 border-indigo-600 border-t-transparent rounded-full"/>
          </div>
        ) : tab === "draws" ? (
          /* ── Draws Table ── */
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
                    <td className="px-4 py-3">
                      <div className="flex gap-1 flex-wrap">
                        {d.status === "scheduled" && (
                          <>
                            <button onClick={() => openEdit(d)}
                              className="px-2 py-1 bg-gray-100 text-gray-700 rounded text-xs hover:bg-gray-200">
                              Edit
                            </button>
                            <button disabled={executing === d.id} onClick={() => doExecute(d.id)}
                              className="px-2 py-1 bg-green-600 text-white rounded text-xs hover:bg-green-700 disabled:opacity-50">
                              {executing === d.id ? "Running…" : "Execute"}
                            </button>
                          </>
                        )}
                        {d.status === "completed" && (
                          <>
                            <button onClick={() => showWinners(d.id)}
                              className="px-2 py-1 bg-indigo-600 text-white rounded text-xs hover:bg-indigo-700">
                              Winners
                            </button>
                            <button onClick={() => doExport(d.id)}
                              className="px-2 py-1 bg-gray-100 text-gray-700 rounded text-xs hover:bg-gray-200">
                              Export
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
        ) : (
          /* ── Draw Schedule Table ── */
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <div className="px-4 py-3 border-b border-gray-100 bg-gray-50">
              <p className="text-xs text-gray-500">
                Window rules define which active draws a recharge qualifies for based on the day and time it occurs.
                Each rule maps a draw to an open/close window (WAT) and a draw execution time.
              </p>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>{["Draw Name","Type","Draw Day/Time (WAT)","Window Open","Window Close","Active","Actions"].map(h =>
                  <th key={h} className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wide">{h}</th>)}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {schedules.length === 0 && (
                  <tr><td colSpan={7} className="text-center py-10 text-gray-400">No window rules configured</td></tr>
                )}
                {schedules.map(s => (
                  <tr key={s.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 font-medium text-gray-900">{s.draw_name}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${s.draw_type === "WEEKLY" ? "bg-purple-100 text-purple-800" : "bg-blue-100 text-blue-800"}`}>
                        {s.draw_type}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-gray-600">{DAYS[s.draw_day_of_week]} {s.draw_time_wat.slice(0,5)}</td>
                    <td className="px-4 py-3 text-gray-600">{DAYS[s.window_open_dow]} {s.window_open_time.slice(0,5)}</td>
                    <td className="px-4 py-3 text-gray-600">{DAYS[s.window_close_dow]} {s.window_close_time.slice(0,5)}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${s.is_active ? "bg-green-100 text-green-800" : "bg-gray-100 text-gray-500"}`}>
                        {s.is_active ? "Active" : "Inactive"}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-1">
                        <button onClick={() => openSchedEdit(s)}
                          className="px-2 py-1 bg-gray-100 text-gray-700 rounded text-xs hover:bg-gray-200">Edit</button>
                        <button onClick={() => deleteSched(s.id)}
                          className="px-2 py-1 bg-red-50 text-red-600 rounded text-xs hover:bg-red-100">Delete</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Draw Create/Edit Modal */}
        {drawModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-white rounded-2xl p-6 w-full max-w-md shadow-xl">
              <h2 className="text-xl font-bold mb-4">
                {drawModal.mode === "create" ? "Schedule New Draw" : "Edit Draw"}
              </h2>
              {err && <p className="text-red-600 text-sm mb-3 bg-red-50 rounded p-2">{err}</p>}
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Draw Name</label>
                  <input value={drawForm.name} onChange={e => setDrawForm({...drawForm, name: e.target.value})}
                    placeholder="e.g. March Monthly Draw"
                    className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Prize Pool (₦)</label>
                  <input type="number" value={drawForm.prize_pool_kobo / 100}
                    onChange={e => setDrawForm({...drawForm, prize_pool_kobo: Math.round(parseFloat(e.target.value || "0") * 100)})}
                    placeholder="e.g. 500000"
                    className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Draw Date &amp; Time</label>
                  <input type="datetime-local" value={drawForm.draw_date}
                    onChange={e => setDrawForm({...drawForm, draw_date: e.target.value})}
                    className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Recurrence</label>
                  <select value={drawForm.recurrence}
                    onChange={e => setDrawForm({...drawForm, recurrence: e.target.value as "once"|"weekly"|"monthly"})}
                    className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none">
                    <option value="once">One-off</option>
                    <option value="weekly">Weekly</option>
                    <option value="monthly">Monthly</option>
                  </select>
                </div>
              </div>
              <div className="flex gap-3 mt-6">
                <button onClick={() => setDrawModal(null)} className="flex-1 border rounded-lg py-2 text-sm text-gray-600 hover:bg-gray-50">
                  Cancel
                </button>
                <button onClick={saveDraw} disabled={saving || !drawForm.name || !drawForm.draw_date}
                  className="flex-1 bg-indigo-600 text-white rounded-lg py-2 text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">
                  {saving ? "Saving…" : drawModal.mode === "create" ? "Schedule" : "Save Changes"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Draw Schedule Create/Edit Modal */}
        {schedModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-y-auto py-8">
            <div className="bg-white rounded-2xl p-6 w-full max-w-lg shadow-xl">
              <h2 className="text-xl font-bold mb-4">
                {schedModal.mode === "create" ? "Add Window Rule" : "Edit Window Rule"}
              </h2>
              {err && <p className="text-red-600 text-sm mb-3 bg-red-50 rounded p-2">{err}</p>}
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Draw Name</label>
                    <input value={schedForm.draw_name} onChange={e => setSchedForm({...schedForm, draw_name: e.target.value})}
                      placeholder="e.g. Weekly Friday Draw"
                      className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Draw Type</label>
                    <select value={schedForm.draw_type} onChange={e => setSchedForm({...schedForm, draw_type: e.target.value})}
                      className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none">
                      <option value="DAILY">Daily</option>
                      <option value="WEEKLY">Weekly</option>
                    </select>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Draw Day (WAT)</label>
                    <select value={schedForm.draw_day_of_week} onChange={e => setSchedForm({...schedForm, draw_day_of_week: +e.target.value})}
                      className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none">
                      {DAYS.map((d,i) => <option key={i} value={i}>{d}</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Draw Time (WAT)</label>
                    <input type="time" value={schedForm.draw_time_wat.slice(0,5)}
                      onChange={e => setSchedForm({...schedForm, draw_time_wat: e.target.value + ":00"})}
                      className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                  </div>
                </div>
                <div className="bg-gray-50 rounded-lg p-3 space-y-3">
                  <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Entry Window</p>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">Opens (Day)</label>
                      <select value={schedForm.window_open_dow} onChange={e => setSchedForm({...schedForm, window_open_dow: +e.target.value})}
                        className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none">
                        {DAYS.map((d,i) => <option key={i} value={i}>{d}</option>)}
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">Opens (Time WAT)</label>
                      <input type="time" value={schedForm.window_open_time.slice(0,5)}
                        onChange={e => setSchedForm({...schedForm, window_open_time: e.target.value + ":00"})}
                        className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">Closes (Day)</label>
                      <select value={schedForm.window_close_dow} onChange={e => setSchedForm({...schedForm, window_close_dow: +e.target.value})}
                        className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none">
                        {DAYS.map((d,i) => <option key={i} value={i}>{d}</option>)}
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">Closes (Time WAT)</label>
                      <input type="time" value={schedForm.window_close_time.slice(0,5)}
                        onChange={e => setSchedForm({...schedForm, window_close_time: e.target.value + ":00"})}
                        className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                    </div>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Cutoff Hour (UTC)</label>
                    <input type="number" min={0} max={23} value={schedForm.cutoff_hour_utc}
                      onChange={e => setSchedForm({...schedForm, cutoff_hour_utc: +e.target.value})}
                      className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Sort Order</label>
                    <input type="number" min={1} value={schedForm.sort_order}
                      onChange={e => setSchedForm({...schedForm, sort_order: +e.target.value})}
                      className="w-full border rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 outline-none"/>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <input type="checkbox" id="sched_active" checked={schedForm.is_active}
                    onChange={e => setSchedForm({...schedForm, is_active: e.target.checked})}
                    className="h-4 w-4 rounded border-gray-300 text-indigo-600"/>
                  <label htmlFor="sched_active" className="text-sm text-gray-700">Active (recharges will qualify for this draw)</label>
                </div>
              </div>
              <div className="flex gap-3 mt-6">
                <button onClick={() => setSchedModal(null)} className="flex-1 border rounded-lg py-2 text-sm text-gray-600 hover:bg-gray-50">
                  Cancel
                </button>
                <button onClick={saveSched} disabled={saving || !schedForm.draw_name}
                  className="flex-1 bg-indigo-600 text-white rounded-lg py-2 text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">
                  {saving ? "Saving…" : schedModal.mode === "create" ? "Create Rule" : "Save Changes"}
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
    </AdminShell>
  );
}
