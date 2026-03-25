"use client";
import { useState, useEffect } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { RegionalStat } from "@/lib/api";

export default function RegionalWarsPage() {
  const [lb, setLb] = useState<RegionalStat[]>([]);
  useEffect(() => { adminAPI.getRegionalWars().then(r => setLb(r.leaderboard)).catch(console.error); }, []);

  return (
    <AdminShell>
      <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff", marginBottom: 20 }}>🌍 Regional Wars</h1>
      <div className="card" style={{ overflow: "hidden" }}>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
              {["Rank", "State", "Total Points", "Active Members"].map(h => (
                <th key={h} style={{ padding: "12px 16px", textAlign: "left", color: "#828cb4", fontSize: 12 }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {lb.map((row, i) => (
              <tr key={row.state} style={{ borderBottom: i < lb.length-1 ? "1px solid rgba(95,114,249,0.05)" : "none" }}>
                <td style={{ padding: "10px 16px", color: "#e2e8ff", fontSize: 18 }}>
                  {["🥇","🥈","🥉"][i] || `#${row.rank}`}
                </td>
                <td style={{ padding: "10px 16px", color: "#e2e8ff", fontWeight: 600 }}>{row.state}</td>
                <td style={{ padding: "10px 16px", color: "#f9c74f", fontWeight: 700 }}>{row.total_points.toLocaleString()} pts</td>
                <td style={{ padding: "10px 16px", color: "#828cb4" }}>{row.active_members.toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </AdminShell>
  );
}