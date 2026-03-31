"use client";
import { useState, useEffect, useCallback } from "react";
import AdminShell from "@/components/layout/AdminShell";
import adminAPI from "@/lib/api";
import Link from "next/link";

// Matches the actual backend GetDashboard response shape:
// spin_stats: { total_spins, spins_today, pending_fulfillments }
// draw_stats: { total_draws, completed_draws, scheduled_draws, total_winners }
interface DashboardData {
  total_users: number;
  active_today: number;
  total_spins: number;
  pending_prizes: number;
  total_points_issued: number;
  spin_stats?: {
    total_spins?: number;
    spins_today?: number;
    pending_fulfillments?: number;
  };
  draw_stats?: {
    total_draws?: number;
    completed_draws?: number;
    scheduled_draws?: number;
    total_winners?: number;
  };
  generated_at?: string;
}

function fmtNaira(kobo: number) {
  if (kobo >= 100_000_00) return `₦${(kobo / 100_000_00).toFixed(1)}M`;
  if (kobo >= 100_000)    return `₦${(kobo / 100_000).toFixed(1)}K`;
  return `₦${(kobo / 100).toLocaleString()}`;
}

export default function Dashboard() {
  const [data, setData]     = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError]   = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const d = await adminAPI.getDashboard() as unknown as DashboardData;
      setData(d);
      setLastRefresh(new Date());
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load dashboard");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const kpis = data ? [
    { label: "Total Users",       value: data.total_users.toLocaleString(),                    icon: "👥", color: "#5f72f9", link: "/users" },
    { label: "Active Today",      value: data.active_today.toLocaleString(),                   icon: "⚡", color: "#f59e0b", link: null },
    { label: "Spins Today",       value: (data.spin_stats?.spins_today ?? 0).toLocaleString(), icon: "🎡", color: "#10b981", link: null },
    { label: "Pending Prizes",    value: data.pending_prizes.toLocaleString(),                 icon: "🏆", color: data.pending_prizes > 0 ? "#ef4444" : "#6b7280", link: "/spin-claims" },
    // draw_stats.scheduled_draws = UPCOMING draws (backend field name)
    { label: "Scheduled Draws",   value: (data.draw_stats?.scheduled_draws ?? 0).toLocaleString(), icon: "🎰", color: "#a78bfa", link: "/draws" },
    { label: "Total Spins",       value: data.total_spins.toLocaleString(),                    icon: "🌀", color: "#5f72f9", link: null },
    { label: "Points Issued",     value: data.total_points_issued.toLocaleString(),            icon: "💎", color: "#10b981", link: null },
    { label: "Pending Fulfilment",value: (data.spin_stats?.pending_fulfillments ?? 0).toLocaleString(), icon: "💰", color: "#f59e0b", link: "/spin-claims" },
  ] : [];

  return (
    <AdminShell>
      <div className="max-w-6xl mx-auto space-y-6 pb-12">
        {/* Header */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: "#e2e8ff" }}>📊 Dashboard</h1>
            {lastRefresh && (
              <p style={{ fontSize: 12, color: "#828cb4", marginTop: 4 }}>
                Last updated: {lastRefresh.toLocaleTimeString("en-NG")}
              </p>
            )}
          </div>
          <button onClick={load}
            style={{ padding: "8px 16px", borderRadius: 8, border: "1px solid rgba(95,114,249,0.3)", color: "#828cb4", fontSize: 13, background: "transparent", cursor: "pointer" }}>
            ↺ Refresh
          </button>
        </div>

        {error && (
          <div style={{ background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.3)", borderRadius: 10, padding: "12px 16px", color: "#fca5a5", fontSize: 13 }}>
            ⚠️ {error}
          </div>
        )}

        {loading ? (
          <div style={{ display: "flex", justifyContent: "center", padding: "60px 0" }}>
            <div style={{ width: 36, height: 36, border: "3px solid #5f72f9", borderTopColor: "transparent", borderRadius: "50%", animation: "spin 0.8s linear infinite" }} />
          </div>
        ) : (
          <>
            {/* KPI Grid */}
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))", gap: 14 }}>
              {kpis.map(k => {
                const inner = (
                  <div key={k.label} className="card" style={{ padding: 20, cursor: k.link ? "pointer" : "default", transition: "border-color 0.2s" }}>
                    <div style={{ fontSize: 26, marginBottom: 8 }}>{k.icon}</div>
                    <div style={{ fontSize: 26, fontWeight: 700, color: k.color }}>{k.value}</div>
                    <div style={{ color: "#828cb4", fontSize: 13, marginTop: 4 }}>{k.label}</div>
                    {k.link && <div style={{ fontSize: 11, color: "#5f72f9", marginTop: 6 }}>View →</div>}
                  </div>
                );
                return k.link ? <Link key={k.label} href={k.link} style={{ textDecoration: "none" }}>{inner}</Link> : inner;
              })}
            </div>

            {/* Quick Links */}
            <div className="card" style={{ padding: 20 }}>
              <h2 style={{ fontSize: 14, fontWeight: 600, color: "#e2e8ff", marginBottom: 14 }}>Quick Actions</h2>
              <div style={{ display: "flex", flexWrap: "wrap", gap: 10 }}>
                {[
                  { href: "/spin-claims",   label: "🏆 Manage Claims",       urgent: (data?.pending_prizes ?? 0) > 0 },
                  { href: "/draws",         label: "🎰 Manage Draws",         urgent: false },
                  { href: "/prizes",        label: "🎁 Prize Pool",           urgent: false },
                  { href: "/spin-config",   label: "⚙️ Spin Config",          urgent: false },
                  { href: "/users",         label: "👥 Users",                urgent: false },
                  { href: "/fraud",         label: "🚨 Fraud Alerts",         urgent: false },
                  { href: "/mtn-push-upload", label: "📤 MTN Push Upload",    urgent: false },
                  { href: "/notifications", label: "📣 Broadcast",            urgent: false },
                ].map(q => (
                  <Link key={q.href} href={q.href}
                    style={{ padding: "8px 14px", borderRadius: 8, border: `1px solid ${q.urgent ? "rgba(239,68,68,0.4)" : "rgba(95,114,249,0.2)"}`, color: q.urgent ? "#fca5a5" : "#c4cde8", fontSize: 13, textDecoration: "none", background: q.urgent ? "rgba(239,68,68,0.08)" : "transparent" }}>
                    {q.label}{q.urgent ? " ⚠️" : ""}
                  </Link>
                ))}
              </div>
            </div>

            {/* Spin & Draw Stats */}
            {(data?.spin_stats || data?.draw_stats) && (
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
                {data?.spin_stats && (
                  <div className="card" style={{ padding: 20 }}>
                    <h2 style={{ fontSize: 14, fontWeight: 600, color: "#e2e8ff", marginBottom: 14 }}>🎡 Spin Statistics</h2>
                    {[
                      ["Spins Today",           (data.spin_stats.spins_today ?? 0).toLocaleString()],
                      ["Total Spins",           (data.spin_stats.total_spins ?? 0).toLocaleString()],
                      ["Pending Fulfilments",   (data.spin_stats.pending_fulfillments ?? 0).toLocaleString()],
                    ].map(([k, v]) => (
                      <div key={k} style={{ display: "flex", justifyContent: "space-between", padding: "6px 0", borderBottom: "1px solid rgba(95,114,249,0.08)" }}>
                        <span style={{ fontSize: 13, color: "#828cb4" }}>{k}</span>
                        <span style={{ fontSize: 13, color: "#e2e8ff", fontWeight: 600 }}>{v}</span>
                      </div>
                    ))}
                  </div>
                )}
                {data?.draw_stats && (
                  <div className="card" style={{ padding: 20 }}>
                    <h2 style={{ fontSize: 14, fontWeight: 600, color: "#e2e8ff", marginBottom: 14 }}>🎰 Draw Statistics</h2>
                    {[
                      ["Total Draws",      (data.draw_stats.total_draws ?? 0).toLocaleString()],
                      ["Scheduled (UPCOMING)", (data.draw_stats.scheduled_draws ?? 0).toLocaleString()],
                      ["Completed",        (data.draw_stats.completed_draws ?? 0).toLocaleString()],
                      ["Total Winners",    (data.draw_stats.total_winners ?? 0).toLocaleString()],
                    ].map(([k, v]) => (
                      <div key={k} style={{ display: "flex", justifyContent: "space-between", padding: "6px 0", borderBottom: "1px solid rgba(95,114,249,0.08)" }}>
                        <span style={{ fontSize: 13, color: "#828cb4" }}>{k}</span>
                        <span style={{ fontSize: 13, color: "#e2e8ff", fontWeight: 600 }}>{v}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </>
        )}
      </div>
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </AdminShell>
  );
}
