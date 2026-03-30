"use client";
import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { Prize, PrizeSummary } from "@/lib/api";

const PRIZE_TYPES = ["try_again", "pulse_points", "airtime", "data_bundle", "momo_cash"];
const PRIZE_TYPE_LABELS: Record<string, string> = {
  try_again:    "No Win",
  pulse_points: "Pulse Points",
  airtime:      "Airtime",
  data_bundle:  "Data Bundle",
  momo_cash:    "Cash Prize",
};

const EMPTY_FORM: Omit<Prize, "id"> = {
  name: "",
  prize_type: "try_again",
  base_value: 0,
  win_probability_weight: 0,
  daily_inventory_cap: -1,
  is_active: true,
  is_no_win: false,
  no_win_message: "",
  color_scheme: "",
  sort_order: 0,
};

type ModalState =
  | { type: "create" }
  | { type: "edit"; prize: Prize }
  | { type: "delete"; prize: Prize }
  | null;

export default function PrizesPage() {
  const [prizes, setPrizes]     = useState<Prize[]>([]);
  const [summary, setSummary]   = useState<PrizeSummary | null>(null);
  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState<string | null>(null);
  const [modal, setModal]       = useState<ModalState>(null);
  const [form, setForm]         = useState<Omit<Prize, "id">>(EMPTY_FORM);
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [pr, sm] = await Promise.all([
        adminAPI.getPrizePool(),
        adminAPI.getPrizeSummary(),
      ]);
      setPrizes(pr.prizes ?? []);
      setSummary(sm);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load prizes");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const openCreate = () => { setForm(EMPTY_FORM); setModal({ type: "create" }); };
  const openEdit   = (p: Prize) => {
    setForm({
      name: p.name, prize_type: p.prize_type, base_value: p.base_value,
      win_probability_weight: p.win_probability_weight, daily_inventory_cap: p.daily_inventory_cap ?? -1,
      is_active: p.is_active, is_no_win: p.is_no_win ?? false, no_win_message: p.no_win_message ?? "",
      color_scheme: p.color_scheme ?? "", sort_order: p.sort_order ?? 0,
    });
    setModal({ type: "edit", prize: p });
  };

  const handleSave = async () => {
    if (!form.name.trim()) { setError("Prize name is required"); return; }
    if (form.win_probability_weight <= 0) { setError("Probability weight must be > 0"); return; }
    setSubmitting(true);
    setError(null);
    try {
      if (modal?.type === "create") {
        await adminAPI.createPrize(form);
      } else if (modal?.type === "edit") {
        await adminAPI.updatePrize((modal as { type: "edit"; prize: Prize }).prize.id, form);
      }
      setModal(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (modal?.type !== "delete") return;
    setSubmitting(true);
    try {
      await adminAPI.deletePrize(modal.prize.id);
      setModal(null);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Delete failed");
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggle = async (p: Prize) => {
    try {
      await adminAPI.updatePrize(p.id, { is_active: !p.is_active });
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Toggle failed");
    }
  };

  const budgetPct   = summary?.percent_used ?? 0;
  const budgetColor = !summary ? "#828cb4" : summary.is_valid ? "#10b981" : "#ef4444";

  return (
    <AdminShell>
      <div className="max-w-6xl mx-auto space-y-5 pb-12">
        {/* Header */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>🎁 Prize Pool</h1>
            <p style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>
              {prizes.length} prizes configured
            </p>
          </div>
          <div style={{ display: "flex", gap: 10 }}>
            <button onClick={load}
              style={{ padding: "8px 16px", borderRadius: 8, border: "1px solid rgba(95,114,249,0.3)", color: "#828cb4", fontSize: 13, background: "transparent", cursor: "pointer" }}>
              ↺ Refresh
            </button>
            <button onClick={openCreate}
              style={{ padding: "8px 18px", borderRadius: 8, background: "#5f72f9", border: "none", color: "#fff", fontSize: 13, fontWeight: 600, cursor: "pointer" }}>
              + Add Prize
            </button>
          </div>
        </div>

        {error && (
          <div style={{ background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.3)", borderRadius: 10, padding: "12px 16px", color: "#fca5a5", fontSize: 13, display: "flex", gap: 10, alignItems: "center" }}>
            ⚠️ {error}
            <button onClick={() => setError(null)} style={{ marginLeft: "auto", background: "none", border: "none", color: "#fca5a5", cursor: "pointer" }}>✕</button>
          </div>
        )}

        {/* Probability Budget Bar */}
        {summary && (
          <div className="card" style={{ padding: 16 }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 10 }}>
              <span style={{ fontSize: 13, color: "#e2e8ff", fontWeight: 600 }}>Probability Budget</span>
              <span style={{ fontSize: 13, color: budgetColor, fontWeight: 700 }}>
                {budgetPct.toFixed(2)}% / 100%
                {summary.is_valid ? " ✓" : ` — ${summary.remaining_budget.toFixed(2)}% remaining`}
              </span>
            </div>
            <div style={{ height: 8, background: "rgba(255,255,255,0.08)", borderRadius: 4, overflow: "hidden" }}>
              <div style={{ height: "100%", width: `${Math.min(budgetPct, 100)}%`, background: budgetColor, borderRadius: 4, transition: "width 0.3s" }} />
            </div>
            {!summary.is_valid && (
              <p style={{ fontSize: 11, color: "#f59e0b", marginTop: 8 }}>
                ⚠️ Total probability weights must sum to exactly 100%. Current total: {budgetPct.toFixed(2)}%
              </p>
            )}
          </div>
        )}

        {/* Prizes Table */}
        {loading ? (
          <div style={{ display: "flex", justifyContent: "center", padding: "60px 0" }}>
            <div style={{ width: 32, height: 32, border: "3px solid #5f72f9", borderTopColor: "transparent", borderRadius: "50%", animation: "spin 0.8s linear infinite" }} />
          </div>
        ) : (
          <div className="card" style={{ overflow: "auto" }}>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.15)" }}>
                  {["Prize Name", "Type", "Value", "Probability %", "Daily Cap", "Status", "Actions"].map(h => (
                    <th key={h} style={{ padding: "10px 14px", textAlign: "left", color: "#828cb4", fontWeight: 600 }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {prizes.length === 0 ? (
                  <tr><td colSpan={7} style={{ padding: "30px 14px", textAlign: "center", color: "#828cb4" }}>No prizes configured</td></tr>
                ) : prizes.map(p => (
                  <tr key={p.id} style={{ borderBottom: "1px solid rgba(95,114,249,0.07)", opacity: p.is_active ? 1 : 0.5 }}>
                    <td style={{ padding: "10px 14px", color: "#e2e8ff", fontWeight: 600 }}>
                      {p.color_scheme && <span style={{ display: "inline-block", width: 10, height: 10, borderRadius: "50%", background: p.color_scheme, marginRight: 8, verticalAlign: "middle" }} />}
                      {p.name}
                    </td>
                    <td style={{ padding: "10px 14px", color: "#828cb4", fontSize: 12 }}>{PRIZE_TYPE_LABELS[p.prize_type] ?? p.prize_type}</td>
                    <td style={{ padding: "10px 14px", color: "#f9c74f", fontWeight: 600 }}>
                      {p.prize_type === "pulse_points" ? `${p.base_value} pts` : p.base_value > 0 ? `₦${p.base_value.toLocaleString()}` : "—"}
                    </td>
                    <td style={{ padding: "10px 14px", color: "#5f72f9", fontWeight: 600 }}>
                      {Number(p.win_probability_weight).toFixed(2)}%
                    </td>
                    <td style={{ padding: "10px 14px", color: "#828cb4" }}>
                      {p.daily_inventory_cap === -1 || p.daily_inventory_cap == null ? "∞" : p.daily_inventory_cap}
                    </td>
                    <td style={{ padding: "10px 14px" }}>
                      <button onClick={() => handleToggle(p)}
                        style={{ padding: "3px 10px", borderRadius: 6, border: `1px solid ${p.is_active ? "rgba(16,185,129,0.3)" : "rgba(239,68,68,0.3)"}`, color: p.is_active ? "#10b981" : "#fca5a5", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                        {p.is_active ? "● Active" : "● Disabled"}
                      </button>
                    </td>
                    <td style={{ padding: "10px 14px" }}>
                      <div style={{ display: "flex", gap: 6 }}>
                        <button onClick={() => openEdit(p)}
                          style={{ padding: "3px 9px", borderRadius: 6, border: "1px solid rgba(95,114,249,0.3)", color: "#5f72f9", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                          Edit
                        </button>
                        <button onClick={() => setModal({ type: "delete", prize: p })}
                          style={{ padding: "3px 9px", borderRadius: 6, border: "1px solid rgba(239,68,68,0.3)", color: "#fca5a5", background: "transparent", fontSize: 11, cursor: "pointer" }}>
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create / Edit Modal */}
      {(modal?.type === "create" || modal?.type === "edit") && (
        <div style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 50 }}>
          <div className="card" style={{ width: "min(540px, 95vw)", padding: 28, maxHeight: "90vh", overflowY: "auto" }}>
            <h2 style={{ fontSize: 16, fontWeight: 700, color: "#e2e8ff", marginBottom: 20 }}>
              {modal.type === "create" ? "➕ Add Prize" : "✏️ Edit Prize"}
            </h2>

            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, marginBottom: 12 }}>
              <div style={{ gridColumn: "1 / -1" }}>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Prize Name *</label>
                <input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Prize Type *</label>
                <select value={form.prize_type} onChange={e => setForm(f => ({ ...f, prize_type: e.target.value }))}
                  style={{ width: "100%", background: "#1c2038", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13 }}>
                  {PRIZE_TYPES.map(t => <option key={t} value={t}>{PRIZE_TYPE_LABELS[t]}</option>)}
                </select>
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>
                  Base Value {form.prize_type === "pulse_points" ? "(points)" : "(₦ Naira)"}
                </label>
                <input type="number" min="0" value={form.base_value}
                  onChange={e => setForm(f => ({ ...f, base_value: parseFloat(e.target.value) || 0 }))}
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Probability Weight (%) *</label>
                <input type="number" min="0.01" max="100" step="0.01" value={form.win_probability_weight}
                  onChange={e => setForm(f => ({ ...f, win_probability_weight: parseFloat(e.target.value) || 0 }))}
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
                <p style={{ fontSize: 11, color: "#828cb4", marginTop: 4 }}>
                  Budget used: {summary ? `${(summary.percent_used + (form.win_probability_weight - (modal.type === "edit" ? (modal as { type: "edit"; prize: Prize }).prize.win_probability_weight : 0))).toFixed(2)}%` : "—"} / 100%
                </p>
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Daily Inventory Cap (-1 = unlimited)</label>
                <input type="number" min="-1" value={form.daily_inventory_cap ?? -1}
                  onChange={e => setForm(f => ({ ...f, daily_inventory_cap: parseInt(e.target.value) || -1 }))}
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Sort Order</label>
                <input type="number" min="0" value={form.sort_order ?? 0}
                  onChange={e => setForm(f => ({ ...f, sort_order: parseInt(e.target.value) || 0 }))}
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
              <div>
                <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>Wheel Color (hex)</label>
                <input value={form.color_scheme ?? ""} onChange={e => setForm(f => ({ ...f, color_scheme: e.target.value }))}
                  placeholder="#5f72f9"
                  style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
              </div>
              <div style={{ display: "flex", alignItems: "center", gap: 10, paddingTop: 20 }}>
                <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", fontSize: 13, color: "#c4cde8" }}>
                  <input type="checkbox" checked={form.is_active} onChange={e => setForm(f => ({ ...f, is_active: e.target.checked }))} />
                  Active
                </label>
                <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", fontSize: 13, color: "#c4cde8" }}>
                  <input type="checkbox" checked={form.is_no_win ?? false} onChange={e => setForm(f => ({ ...f, is_no_win: e.target.checked }))} />
                  No-Win Slot
                </label>
              </div>
              {form.is_no_win && (
                <div style={{ gridColumn: "1 / -1" }}>
                  <label style={{ fontSize: 12, color: "#828cb4", display: "block", marginBottom: 5 }}>No-Win Message</label>
                  <input value={form.no_win_message ?? ""} onChange={e => setForm(f => ({ ...f, no_win_message: e.target.value }))}
                    placeholder="Better luck next time!"
                    style={{ width: "100%", background: "rgba(255,255,255,0.05)", border: "1px solid rgba(95,114,249,0.2)", borderRadius: 8, padding: "8px 12px", color: "#e2e8ff", fontSize: 13, boxSizing: "border-box" }} />
                </div>
              )}
            </div>

            <div style={{ display: "flex", gap: 10, marginTop: 8 }}>
              <button onClick={() => setModal(null)}
                style={{ flex: 1, padding: "10px", borderRadius: 8, background: "transparent", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
                Cancel
              </button>
              <button onClick={handleSave} disabled={submitting}
                style={{ flex: 1, padding: "10px", borderRadius: 8, background: "#5f72f9", border: "none", color: "#fff", fontWeight: 600, cursor: "pointer", opacity: submitting ? 0.6 : 1 }}>
                {submitting ? "Saving…" : modal.type === "create" ? "Create Prize" : "Save Changes"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirm Modal */}
      {modal?.type === "delete" && (
        <div style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 50 }}>
          <div className="card" style={{ width: "min(400px, 95vw)", padding: 28 }}>
            <h2 style={{ fontSize: 16, fontWeight: 700, color: "#ef4444", marginBottom: 8 }}>🗑 Delete Prize</h2>
            <p style={{ fontSize: 13, color: "#c4cde8", marginBottom: 20 }}>
              Are you sure you want to delete <strong>{modal.prize.name}</strong>? This cannot be undone.
            </p>
            <div style={{ display: "flex", gap: 10 }}>
              <button onClick={() => setModal(null)}
                style={{ flex: 1, padding: "10px", borderRadius: 8, background: "transparent", border: "1px solid rgba(95,114,249,0.2)", color: "#828cb4", cursor: "pointer" }}>
                Cancel
              </button>
              <button onClick={handleDelete} disabled={submitting}
                style={{ flex: 1, padding: "10px", borderRadius: 8, background: "#ef4444", border: "none", color: "#fff", fontWeight: 600, cursor: "pointer", opacity: submitting ? 0.6 : 1 }}>
                {submitting ? "Deleting…" : "Delete Prize"}
              </button>
            </div>
          </div>
        </div>
      )}

      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </AdminShell>
  );
}
