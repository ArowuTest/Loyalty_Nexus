"use client";
import { useState, useEffect } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { ConfigEntry } from "@/lib/api";

export default function ConfigPage() {
  const [configs, setConfigs] = useState<ConfigEntry[]>([]);
  const [editing, setEditing] = useState<{key: string; value: string} | null>(null);
  const [saving, setSaving] = useState(false);
  const [search, setSearch] = useState("");

  useEffect(() => {
    adminAPI.getConfig().then(r => setConfigs(r.configs)).catch(console.error);
  }, []);

  const save = async () => {
    if (!editing) return;
    setSaving(true);
    await adminAPI.updateConfig(editing.key, editing.value).catch(console.error);
    setConfigs(c => c.map(e => e.key === editing.key ? { ...e, value: editing.value } : e));
    setEditing(null);
    setSaving(false);
  };

  const filtered = configs.filter(c => c.key.includes(search) || String(c.value).includes(search));

  return (
    <AdminShell>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 20 }}>
        <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>⚙️ Configuration</h1>
        <input className="input" placeholder="Search configs…" value={search} onChange={e => setSearch(e.target.value)} style={{ width: 240 }} />
      </div>
      <div className="card" style={{ overflow: "hidden" }}>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
              {["Key", "Value", "Description", "Actions"].map(h => (
                <th key={h} style={{ padding: "12px 16px", textAlign: "left", color: "#828cb4", fontSize: 12, fontWeight: 600 }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {filtered.map((c, i) => (
              <tr key={c.key} style={{ borderBottom: i < filtered.length-1 ? "1px solid rgba(95,114,249,0.05)" : "none" }}>
                <td style={{ padding: "10px 16px", color: "#5f72f9", fontFamily: "monospace", fontSize: 13 }}>{c.key}</td>
                <td style={{ padding: "10px 16px", color: "#e2e8ff", fontFamily: "monospace", fontSize: 13, maxWidth: 180 }}>
                  {String(c.value).slice(0, 40)}{String(c.value).length > 40 ? "…" : ""}
                </td>
                <td style={{ padding: "10px 16px", color: "#828cb4", fontSize: 12, maxWidth: 250 }}>{c.description}</td>
                <td style={{ padding: "10px 16px" }}>
                  <button className="btn-outline" style={{ fontSize: 12, padding: "4px 10px" }}
                    onClick={() => setEditing({ key: c.key, value: String(c.value) })}>Edit</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {editing && (
        <div style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.6)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 50 }} onClick={() => setEditing(null)}>
          <div className="card" style={{ width: 440, padding: 24 }} onClick={e => e.stopPropagation()}>
            <h3 style={{ color: "#e2e8ff", fontWeight: 600, marginBottom: 4 }}>{editing.key}</h3>
            <p style={{ color: "#828cb4", fontSize: 12, marginBottom: 16 }}>Edit configuration value</p>
            <input className="input" value={editing.value} onChange={e => setEditing({ ...editing, value: e.target.value })} style={{ marginBottom: 16 }} />
            <div style={{ display: "flex", gap: 8 }}>
              <button className="btn-outline" style={{ flex: 1 }} onClick={() => setEditing(null)}>Cancel</button>
              <button className="btn-primary" style={{ flex: 1 }} onClick={save} disabled={saving}>{saving ? "Saving…" : "Save"}</button>
            </div>
          </div>
        </div>
      )}
    </AdminShell>
  );
}