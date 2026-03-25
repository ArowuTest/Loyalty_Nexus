"use client";
import { useState, useEffect } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI, { DashboardStats } from "@/lib/api";

const STAT_CARDS = [
  { key: "total_users",               label: "Total Users",       icon: "👥", format: (v: number) => v.toLocaleString() },
  { key: "active_today",              label: "Active Today",      icon: "⚡", format: (v: number) => v.toLocaleString() },
  { key: "total_recharge_kobo",       label: "Total Recharge",    icon: "💰", format: (v: number) => `₦${(v/100).toLocaleString()}` },
  { key: "spins_today",               label: "Spins Today",       icon: "🎡", format: (v: number) => v.toLocaleString() },
  { key: "studio_generations_today",  label: "AI Generations",    icon: "🧠", format: (v: number) => v.toLocaleString() },
];

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    adminAPI.getDashboard().then(s => { setStats(s); setLoading(false); }).catch(() => setLoading(false));
  }, []);

  return (
    <AdminShell>
      <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff", marginBottom: 24 }}>📊 Dashboard</h1>
      {loading ? (
        <div style={{ color: "#828cb4" }}>Loading stats…</div>
      ) : (
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))", gap: 16 }}>
          {STAT_CARDS.map(card => {
            const val = stats ? (stats as unknown as Record<string, number>)[card.key] ?? 0 : 0;
            return (
              <div key={card.key} className="card" style={{ padding: 20 }}>
                <div style={{ fontSize: 28, marginBottom: 8 }}>{card.icon}</div>
                <div style={{ fontSize: 26, fontWeight: 700, color: "#e2e8ff" }}>{card.format(val)}</div>
                <div style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>{card.label}</div>
              </div>
            );
          })}
        </div>
      )}
    </AdminShell>
  );
}