"use client";
import { useState, useEffect } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { FraudEvent } from "@/lib/api";

const SEV_COLOR: Record<string, string> = { HIGH: "#f43f5e", MEDIUM: "#f59e0b", LOW: "#10b981" };

export default function FraudPage() {
  const [events, setEvents] = useState<FraudEvent[]>([]);
  useEffect(() => { adminAPI.getFraudEvents().then(r => setEvents(r.events)).catch(console.error); }, []);

  return (
    <AdminShell>
      <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff", marginBottom: 20 }}>🛡 Fraud Events</h1>
      {events.length === 0 ? (
        <div className="card" style={{ padding: 32, textAlign: "center", color: "#828cb4" }}>
          ✅ No unresolved fraud events
        </div>
      ) : (
        <div className="card" style={{ overflow: "hidden" }}>
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr style={{ borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
                {["User", "Event Type", "Severity", "Time", "Status"].map(h => (
                  <th key={h} style={{ padding: "12px 16px", textAlign: "left", color: "#828cb4", fontSize: 12 }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {events.map((e, i) => (
                <tr key={e.id} style={{ borderBottom: i < events.length-1 ? "1px solid rgba(95,114,249,0.05)" : "none" }}>
                  <td style={{ padding: "10px 16px", color: "#e2e8ff", fontFamily: "monospace", fontSize: 12 }}>{e.user_id.slice(0,8)}…</td>
                  <td style={{ padding: "10px 16px", color: "#e2e8ff", fontSize: 13 }}>{e.event_type}</td>
                  <td style={{ padding: "10px 16px" }}>
                    <span style={{ color: SEV_COLOR[e.severity] || "#e2e8ff", fontWeight: 600, fontSize: 12 }}>{e.severity}</span>
                  </td>
                  <td style={{ padding: "10px 16px", color: "#828cb4", fontSize: 12 }}>{new Date(e.created_at).toLocaleString()}</td>
                  <td style={{ padding: "10px 16px" }}>
                    <span style={{ color: e.resolved ? "#10b981" : "#f43f5e", fontSize: 12 }}>{e.resolved ? "Resolved" : "Open"}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </AdminShell>
  );
}