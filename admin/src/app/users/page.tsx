"use client";
import { useState, useEffect } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { User } from "@/lib/api";

const TIER_COLORS: Record<string, string> = {
  BRONZE: "#f59e0b", SILVER: "#94a3b8", GOLD: "#eab308", PLATINUM: "#a855f7",
};

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  useEffect(() => { adminAPI.getUsers().then(r => setUsers(r.users)).catch(console.error); }, []);

  return (
    <AdminShell>
      <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff", marginBottom: 20 }}>👥 Users</h1>
      <div className="card" style={{ overflow: "hidden" }}>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
              {["Phone", "Tier", "Streak", "Status", "Joined", "Actions"].map(h => (
                <th key={h} style={{ padding: "12px 16px", textAlign: "left", color: "#828cb4", fontSize: 12 }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {users.map((u, i) => (
              <tr key={u.id} style={{ borderBottom: i < users.length-1 ? "1px solid rgba(95,114,249,0.05)" : "none" }}>
                <td style={{ padding: "10px 16px", color: "#e2e8ff", fontSize: 13, fontFamily: "monospace" }}>{u.phone_number}</td>
                <td style={{ padding: "10px 16px" }}>
                  <span style={{ color: TIER_COLORS[u.tier] || "#e2e8ff", fontWeight: 600, fontSize: 12 }}>{u.tier}</span>
                </td>
                <td style={{ padding: "10px 16px", color: "#828cb4", fontSize: 13 }}>{u.streak_count}d 🔥</td>
                <td style={{ padding: "10px 16px" }}>
                  <span style={{ color: u.is_active ? "#10b981" : "#f43f5e", fontSize: 12 }}>
                    {u.is_active ? "Active" : "Suspended"}
                  </span>
                </td>
                <td style={{ padding: "10px 16px", color: "#828cb4", fontSize: 12 }}>
                  {new Date(u.created_at).toLocaleDateString()}
                </td>
                <td style={{ padding: "10px 16px" }}>
                  {u.is_active && (
                    <button className="btn-outline" style={{ fontSize: 11, padding: "3px 8px", borderColor: "#f43f5e", color: "#f43f5e" }}
                      onClick={() => adminAPI.suspendUser(u.id).then(() => setUsers(us => us.map(x => x.id === u.id ? { ...x, is_active: false } : x)))}>
                      Suspend
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </AdminShell>
  );
}