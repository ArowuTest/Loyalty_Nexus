"use client";
import { useState, useEffect } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { Prize } from "@/lib/api";

export default function PrizesPage() {
  const [prizes, setPrizes] = useState<Prize[]>([]);
  useEffect(() => { adminAPI.getPrizePool().then(r => setPrizes(r.prizes)).catch(console.error); }, []);

  return (
    <AdminShell>
      <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff", marginBottom: 20 }}>🎁 Prize Pool</h1>
      <div className="card" style={{ overflow: "hidden" }}>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
              {["Prize", "Type", "Value", "Probability %", "Daily Inventory", "Status"].map(h => (
                <th key={h} style={{ padding: "12px 16px", textAlign: "left", color: "#828cb4", fontSize: 12 }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {prizes.map((p, i) => (
              <tr key={p.id} style={{ borderBottom: i < prizes.length-1 ? "1px solid rgba(95,114,249,0.05)" : "none" }}>
                <td style={{ padding: "10px 16px", color: "#e2e8ff", fontWeight: 600 }}>{p.name}</td>
                <td style={{ padding: "10px 16px", color: "#828cb4", fontSize: 12, textTransform: "uppercase" }}>{p.prize_type}</td>
                <td style={{ padding: "10px 16px", color: "#f9c74f", fontWeight: 600 }}>₦{p.base_value.toLocaleString()}</td>
                <td style={{ padding: "10px 16px", color: "#5f72f9" }}>{((p.win_probability_weight / 100).toFixed(2))}%</td>
                <td style={{ padding: "10px 16px", color: "#828cb4" }}>{p.daily_inventory_cap === -1 ? "∞" : p.daily_inventory_cap}</td>
                <td style={{ padding: "10px 16px" }}>
                  <span style={{ color: p.is_active ? "#10b981" : "#f43f5e", fontSize: 12 }}>{p.is_active ? "Active" : "Disabled"}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </AdminShell>
  );
}