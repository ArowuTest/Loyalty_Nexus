"use client";
import { ReactNode, useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import adminAPI from "@/lib/api";

const NAV = [
  { href: "/dashboard",     label: "📊 Dashboard" },
  { href: "/config",        label: "⚙️ Config" },
  { href: "/prizes",        label: "🎁 Prizes" },
  { href: "/studio-tools",  label: "🧠 Studio Tools" },
  { href: "/users",         label: "👥 Users" },
  { href: "/fraud",         label: "🛡 Fraud" },
  { href: "/regional-wars", label: "🌍 Regional Wars" },
];

export default function AdminShell({ children }: { children: ReactNode }) {
  const router = useRouter();
  const path = usePathname();
  useEffect(() => {
    if (!adminAPI.getToken()) router.push("/login");
  }, [router]);

  return (
    <div style={{ display: "flex", minHeight: "100vh" }}>
      <aside style={{ width: 220, background: "#1c2038", borderRight: "1px solid rgba(95,114,249,0.15)", padding: "24px 0", flexShrink: 0, display: "flex", flexDirection: "column" }}>
        <div style={{ padding: "0 20px 24px", borderBottom: "1px solid rgba(95,114,249,0.1)" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span style={{ fontSize: 22 }}>⚡</span>
            <div>
              <div style={{ color: "#e2e8ff", fontWeight: 700, fontSize: 14 }}>Nexus Admin</div>
              <div style={{ color: "#828cb4", fontSize: 11 }}>Operations Cockpit</div>
            </div>
          </div>
        </div>
        <nav style={{ padding: "16px 8px", flex: 1 }}>
          {NAV.map(item => (
            <Link key={item.href} href={item.href} style={{
              display: "block", padding: "10px 14px", borderRadius: 8, marginBottom: 2, textDecoration: "none",
              background: path === item.href ? "rgba(95,114,249,0.15)" : "transparent",
              color: path === item.href ? "#5f72f9" : "#828cb4",
              fontWeight: path === item.href ? 600 : 400, fontSize: 13,
              transition: "all 0.15s",
            }}>
              {item.label}
            </Link>
          ))}
        </nav>
        <div style={{ padding: "16px 20px", borderTop: "1px solid rgba(95,114,249,0.1)" }}>
          <button onClick={() => { adminAPI.clearToken(); router.push("/login"); }}
            style={{ color: "#f43f5e", fontSize: 12, background: "none", border: "none", cursor: "pointer" }}>
            Sign out
          </button>
        </div>
      </aside>
      <main style={{ flex: 1, overflow: "auto", padding: 28 }}>{children}</main>
    </div>
  );
}