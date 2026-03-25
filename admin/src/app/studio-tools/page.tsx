"use client";
import { useState, useEffect } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { StudioTool } from "@/lib/api";

export default function StudioToolsPage() {
  const [tools, setTools] = useState<StudioTool[]>([]);
  useEffect(() => { adminAPI.getStudioTools().then(r => setTools(r.tools)).catch(console.error); }, []);

  return (
    <AdminShell>
      <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff", marginBottom: 20 }}>🧠 Studio Tools</h1>
      <div className="card" style={{ overflow: "hidden" }}>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
              {["Tool", "Category", "Provider", "Point Cost", "Status"].map(h => (
                <th key={h} style={{ padding: "12px 16px", textAlign: "left", color: "#828cb4", fontSize: 12 }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {tools.map((t, i) => (
              <tr key={t.id} style={{ borderBottom: i < tools.length-1 ? "1px solid rgba(95,114,249,0.05)" : "none" }}>
                <td style={{ padding: "10px 16px", color: "#e2e8ff", fontWeight: 600, fontSize: 13 }}>{t.name}</td>
                <td style={{ padding: "10px 16px", color: "#828cb4", fontSize: 12 }}>{t.category}</td>
                <td style={{ padding: "10px 16px", color: "#5f72f9", fontSize: 12, fontFamily: "monospace" }}>{t.provider}</td>
                <td style={{ padding: "10px 16px", color: t.point_cost === 0 ? "#10b981" : "#f9c74f", fontWeight: 600 }}>
                  {t.point_cost === 0 ? "Free" : `${t.point_cost} pts`}
                </td>
                <td style={{ padding: "10px 16px" }}>
                  <span style={{ color: t.is_active ? "#10b981" : "#f43f5e", fontSize: 12 }}>{t.is_active ? "Active" : "Disabled"}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </AdminShell>
  );
}