"use client";
import { ReactNode, useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import adminAPI from "@/lib/api";

const NAV = [
  { group: "Operations",   items: [
    { href: "/dashboard",      label: "📊 Dashboard" },
    { href: "/health",         label: "🩺 System Health" },
  ]},
  { group: "Users & Finance", items: [
    { href: "/users",          label: "👥 Users" },
    { href: "/subscriptions",  label: "💳 Subscriptions" },
    { href: "/fraud",          label: "🛡 Fraud Alerts" },
  ]},
  { group: "Rewards Engine", items: [
    { href: "/points-config",  label: "💎 Points Engine" },
    { href: "/spin-config",    label: "🎡 Spin Wheel" },
    { href: "/prizes",         label: "🏆 Prize Pool" },
    { href: "/draws",          label: "🎟 Draws" },
  ]},
  { group: "Content & AI", items: [
    { href: "/studio-tools",   label: "🧠 Studio Tools" },
    { href: "/ai-providers",   label: "🔌 AI Providers" },
    { href: "/ai-health",      label: "⚡ AI Provider Health" },
    { href: "/generations",    label: "🧪 Generations" },
    { href: "/notifications",  label: "📢 Notifications" },
  ]},
  { group: "Platform", items: [
    { href: "/regional-wars",  label: "🌍 Regional Wars" },
    { href: "/passport",       label: "🪪 Passport & USSD" },
    { href: "/config",         label: "⚙️  Config" },
  ]},
];

export default function AdminShell({ children }: { children: ReactNode }) {
  const router  = useRouter();
  const path    = usePathname();
  useEffect(() => {
    if (!adminAPI.getToken()) router.push("/login");
  }, [router]);

  return (
    <div style={{ display: "flex", minHeight: "100vh" }}>
      <aside style={{
        width: 220, background: "#1c2038",
        borderRight: "1px solid rgba(95,114,249,0.15)",
        padding: "24px 0", flexShrink: 0,
        display: "flex", flexDirection: "column",
        overflowY: "auto",
      }}>
        {/* Logo */}
        <div style={{ padding: "0 20px 24px", borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span style={{ fontSize: 22 }}>⚡</span>
            <div>
              <div style={{ color: "#e2e8ff", fontWeight: 700, fontSize: 14 }}>Nexus Admin</div>
              <div style={{ color: "#828cb4", fontSize: 11 }}>Operations Cockpit</div>
            </div>
          </div>
        </div>

        {/* Nav groups */}
        <nav style={{ padding: "12px 8px", flex: 1 }}>
          {NAV.map(group => (
            <div key={group.group} style={{ marginBottom: 12 }}>
              <div style={{ padding: "4px 14px 6px", color: "#4b5563", fontSize: 10, fontWeight: 700, letterSpacing: "0.08em", textTransform: "uppercase" }}>
                {group.group}
              </div>
              {group.items.map(item => (
                <Link key={item.href} href={item.href} style={{
                  display: "block", padding: "9px 14px", borderRadius: 8, marginBottom: 1,
                  textDecoration: "none",
                  background: path === item.href ? "rgba(95,114,249,0.15)" : "transparent",
                  color: path === item.href ? "#5f72f9" : "#828cb4",
                  fontWeight: path === item.href ? 600 : 400, fontSize: 13,
                  transition: "all 0.15s",
                }}>
                  {item.label}
                </Link>
              ))}
            </div>
          ))}
        </nav>

        <div style={{ padding: "16px 20px", borderTop: "1px solid rgba(95,114,249,0.1)" }}>
          <button onClick={() => { adminAPI.clearToken(); router.push("/login"); }}
            style={{ color: "#f43f5e", fontSize: 12, background: "none", border: "none", cursor: "pointer" }}>
            Sign out
          </button>
        </div>
      </aside>

      <main style={{ flex: 1, overflow: "auto", padding: 28, background: "#f8f9fc" }}>
        {children}
      </main>
    </div>
  );
}
