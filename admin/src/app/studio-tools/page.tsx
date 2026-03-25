"use client";
import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { StudioTool } from "@/lib/api";

type EditingTool = { point_cost: string; is_active: boolean } | null;

export default function StudioToolsPage() {
  const [tools, setTools]     = useState<StudioTool[]>([]);
  const [loading, setLoading] = useState(true);
  const [editId, setEditId]   = useState<string | null>(null);
  const [editing, setEditing] = useState<EditingTool>(null);
  const [saving, setSaving]   = useState(false);
  const [saved, setSaved]     = useState<string | null>(null);
  const [error, setError]     = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await adminAPI.getStudioTools();
      setTools(r.tools ?? []);
    } catch (e: unknown) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const startEdit = (t: StudioTool) => {
    setEditId(t.id);
    setEditing({ point_cost: String(t.point_cost), is_active: t.is_active });
    setError(null);
  };

  const cancelEdit = () => { setEditId(null); setEditing(null); };

  const saveEdit = async (tool: StudioTool) => {
    if (!editing) return;
    const cost = parseInt(editing.point_cost, 10);
    if (isNaN(cost) || cost < 0) { setError("Point cost must be 0 or a positive integer"); return; }
    setSaving(true);
    try {
      await adminAPI.updateStudioTool(tool.id, { point_cost: cost, is_active: editing.is_active });
      setSaved(tool.id);
      setEditId(null);
      setEditing(null);
      setTimeout(() => setSaved(null), 2000);
      // Optimistically update local state
      setTools(prev => prev.map(t => t.id === tool.id
        ? { ...t, point_cost: cost, is_active: editing.is_active }
        : t
      ));
    } catch (e: unknown) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const categories = [...new Set(tools.map(t => t.category))].sort();

  const inputStyle: React.CSSProperties = {
    background: "rgba(95,114,249,0.08)", border: "1px solid rgba(95,114,249,0.35)",
    borderRadius: 6, color: "#e2e8ff", padding: "4px 8px", fontSize: 13, width: 80,
    outline: "none",
  };
  const btnStyle = (variant: "save" | "cancel" | "edit"): React.CSSProperties => ({
    padding: "3px 10px", borderRadius: 6, fontSize: 11, fontWeight: 600, cursor: "pointer",
    border: "none",
    background: variant === "save" ? "#5f72f9" : variant === "edit" ? "rgba(95,114,249,0.15)" : "rgba(255,255,255,0.06)",
    color: variant === "save" ? "#fff" : variant === "edit" ? "#5f72f9" : "#828cb4",
  });

  return (
    <AdminShell>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 20 }}>
        <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>🧠 Studio Tools</h1>
        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <span style={{ color: "#828cb4", fontSize: 12 }}>{tools.length} tools</span>
          <button onClick={load} style={{ ...btnStyle("edit"), padding: "5px 12px" }}>↻ Refresh</button>
        </div>
      </div>

      {error && (
        <div style={{ background: "rgba(244,63,94,0.1)", border: "1px solid rgba(244,63,94,0.3)", borderRadius: 8, padding: "8px 12px", color: "#f43f5e", fontSize: 12, marginBottom: 12 }}>
          {error}
        </div>
      )}

      {loading ? (
        <div style={{ color: "#828cb4", padding: 40, textAlign: "center" }}>Loading tools from database…</div>
      ) : tools.length === 0 ? (
        <div style={{ color: "#828cb4", padding: 40, textAlign: "center" }}>No tools found. Run migrations to seed the studio_tools table.</div>
      ) : (
        categories.map(cat => (
          <div key={cat} style={{ marginBottom: 24 }}>
            <h2 style={{ fontSize: 13, fontWeight: 700, color: "#5f72f9", textTransform: "uppercase", letterSpacing: "0.08em", marginBottom: 8 }}>
              {cat}
            </h2>
            <div className="card" style={{ overflow: "hidden" }}>
              <table style={{ width: "100%", borderCollapse: "collapse" }}>
                <thead>
                  <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
                    {["Tool", "Provider", "Point Cost", "Status", "Usage Today", "Actions"].map(h => (
                      <th key={h} style={{ padding: "10px 14px", textAlign: "left", color: "#828cb4", fontSize: 11, fontWeight: 600, textTransform: "uppercase" }}>{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {tools.filter(t => t.category === cat).map((t, i, arr) => {
                    const isEditing = editId === t.id;
                    return (
                      <tr key={t.id} style={{ borderBottom: i < arr.length - 1 ? "1px solid rgba(95,114,249,0.05)" : "none" }}>
                        <td style={{ padding: "10px 14px" }}>
                          <div style={{ color: "#e2e8ff", fontWeight: 600, fontSize: 13 }}>{t.name}</div>
                          <div style={{ color: "#4d567a", fontSize: 11, fontFamily: "monospace" }}>{(t as any).slug ?? t.id?.slice(0, 8)}</div>
                        </td>
                        <td style={{ padding: "10px 14px", color: "#5f72f9", fontSize: 12, fontFamily: "monospace" }}>{t.provider}</td>
                        <td style={{ padding: "10px 14px" }}>
                          {isEditing && editing ? (
                            <input
                              type="number" min={0} style={inputStyle}
                              value={editing.point_cost}
                              onChange={e => setEditing(prev => prev ? { ...prev, point_cost: e.target.value } : prev)}
                            />
                          ) : (
                            <span style={{ color: t.point_cost === 0 ? "#10b981" : "#f9c74f", fontWeight: 700 }}>
                              {t.point_cost === 0 ? "Free" : `${t.point_cost} pts`}
                            </span>
                          )}
                        </td>
                        <td style={{ padding: "10px 14px" }}>
                          {isEditing && editing ? (
                            <label style={{ display: "flex", alignItems: "center", gap: 6, cursor: "pointer" }}>
                              <input
                                type="checkbox" checked={editing.is_active}
                                onChange={e => setEditing(prev => prev ? { ...prev, is_active: e.target.checked } : prev)}
                                style={{ accentColor: "#5f72f9" }}
                              />
                              <span style={{ color: "#c8cef5", fontSize: 12 }}>{editing.is_active ? "Active" : "Disabled"}</span>
                            </label>
                          ) : (
                            <span style={{
                              color: t.is_active ? "#10b981" : "#f43f5e",
                              background: t.is_active ? "rgba(16,185,129,0.1)" : "rgba(244,63,94,0.1)",
                              borderRadius: 4, padding: "2px 7px", fontSize: 11, fontWeight: 600,
                            }}>
                              {t.is_active ? "● Active" : "○ Disabled"}
                            </span>
                          )}
                        </td>
                        <td style={{ padding: "10px 14px", color: "#828cb4", fontSize: 12 }}>
                          {(t as any).usage_count ?? 0}
                        </td>
                        <td style={{ padding: "10px 14px" }}>
                          {saved === t.id ? (
                            <span style={{ color: "#10b981", fontSize: 12 }}>✓ Saved</span>
                          ) : isEditing ? (
                            <div style={{ display: "flex", gap: 6 }}>
                              <button onClick={() => saveEdit(t)} disabled={saving} style={btnStyle("save")}>
                                {saving ? "…" : "Save"}
                              </button>
                              <button onClick={cancelEdit} style={btnStyle("cancel")}>Cancel</button>
                            </div>
                          ) : (
                            <button onClick={() => startEdit(t)} style={btnStyle("edit")}>Edit</button>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        ))
      )}
    </AdminShell>
  );
}
