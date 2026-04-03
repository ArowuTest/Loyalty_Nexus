"use client";

import { ReactNode, useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { useStore } from "@/store/useStore";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard, Zap, Wand2, Gift, Ticket, Settings, LogOut, Shield, Bell, Trophy
} from "lucide-react";

const NAV_ITEMS = [
  { href: "/dashboard", icon: LayoutDashboard, label: "Home"   },
  { href: "/spin",      icon: Zap,             label: "Wheel Spin"   },
  { href: "/studio",    icon: Wand2,           label: "AI Studio" },
  { href: "/wars",      icon: Trophy,          label: "Wars"   },
  { href: "/prizes",    icon: Gift,            label: "Prizes" },
];

const TIER_COLORS: Record<string, string> = {
  BRONZE: "#CD7F32", SILVER: "#C0C0C0", GOLD: "#F5A623", PLATINUM: "#E5E4E2", DIAMOND: "#B9F2FF",
};
const TIER_ICONS: Record<string, string> = {
  BRONZE: "🥉", SILVER: "🥈", GOLD: "🥇", PLATINUM: "💎", DIAMOND: "💠",
};

export default function AppShell({ children }: { children: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, _hasHydrated, logout, user } = useStore();
  const tier      = (user?.tier ?? "BRONZE").toUpperCase();
  const tierColor = TIER_COLORS[tier] ?? "#CD7F32";
  const tierIcon  = TIER_ICONS[tier]  ?? "🥉";

  useEffect(() => {
    // Only redirect once the Zustand persist store has finished rehydrating
    // from localStorage. Without this guard, the redirect fires before the
    // stored token is read, causing a spurious logout on every hard reload.
    if (_hasHydrated && !isAuthenticated) {
      router.push("/");
    }
  }, [_hasHydrated, isAuthenticated, router]);

  useEffect(() => {
    // Listen for the soft session-expired event dispatched by the API client
    // when a 401 is received. This replaces the hard window.location.href redirect
    // that was causing the dashboard to crash and flicker.
    const handleSessionExpired = () => {
      logout();
      router.push("/");
    };
    window.addEventListener("nexus:session-expired", handleSessionExpired);
    return () => window.removeEventListener("nexus:session-expired", handleSessionExpired);
  }, [logout, router]);

  // Show nothing until hydration is complete to avoid a flash of the landing page
  if (!_hasHydrated) return null;
  if (!isAuthenticated) return null;

  return (
    <div className="min-h-screen flex flex-col" style={{ background: "var(--surface-0)" }}>

      {/* ── Desktop top bar ── */}
      <header
        className="hidden md:flex items-center justify-between px-6 py-3.5 sticky top-0 z-50"
        style={{
          background: "rgba(13,14,20,0.88)",
          backdropFilter: "blur(20px)",
          WebkitBackdropFilter: "blur(20px)",
          borderBottom: "1px solid rgba(255,255,255,0.06)",
        }}
      >
        {/* Logo */}
        <Link href="/" className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0"
            style={{ background: "rgba(245,166,35,0.12)", border: "1px solid rgba(245,166,35,0.25)" }}>
            <Zap size={16} style={{ color: "var(--gold)" }} />
          </div>
          <span className="font-black text-[15px] text-white tracking-tight">Loyalty Nexus</span>
        </Link>

        {/* Nav links */}
        <nav className="flex gap-0.5">
          {NAV_ITEMS.map((item) => {
            const active = pathname === item.href || pathname.startsWith(item.href + "/");
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-2 px-4 py-2 rounded-xl text-[13px] font-black transition-all",
                  active ? "" : "text-white/40 hover:text-white hover:bg-white/[0.05]"
                )}
                style={active ? {
                  background: "rgba(245,166,35,0.10)",
                  border: "1px solid rgba(245,166,35,0.18)",
                  color: "var(--gold)",
                } : {}}
              >
                <item.icon size={15} />
                {item.label}
              </Link>
            );
          })}
        </nav>
        <div className="flex items-center gap-3">
          <span className={cn("tier-badge", `tier-${user?.tier || "BRONZE"}`)}>{user?.tier || "BRONZE"}</span>
          <Link href="/notifications" className="text-[rgb(130_140_180)] hover:text-white transition-colors">
            <Bell size={18} />
          </Link>
          <Link href="/settings" className="text-[rgb(130_140_180)] hover:text-white transition-colors">
            <Settings size={18} />
          </Link>
          <button
            onClick={() => { logout(); router.push("/"); }}
            className="w-8 h-8 rounded-xl flex items-center justify-center text-white/40 hover:text-red-400 transition-colors"
            style={{ border: "1px solid rgba(255,255,255,0.07)" }}
          >
            <LogOut size={15} />
          </button>
        </div>
      </header>

      {/* ── Main content ── */}
      <main className="flex-1 pb-24 md:pb-8">{children}</main>

      {/* ── Mobile bottom nav ── */}
      <nav
        className="md:hidden fixed bottom-0 left-0 right-0 flex justify-around py-2 z-50"
        style={{
          background: "rgba(13,14,20,0.94)",
          backdropFilter: "blur(20px)",
          WebkitBackdropFilter: "blur(20px)",
          borderTop: "1px solid rgba(255,255,255,0.06)",
        }}
      >
        {NAV_ITEMS.map((item) => {
          const active = pathname === item.href || pathname.startsWith(item.href + "/");
          return (
            <Link
              key={item.href}
              href={item.href}
              className="flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl transition-all min-w-[52px]"
              style={{ color: active ? "var(--gold)" : "rgba(255,255,255,0.35)" }}
            >
              <item.icon size={20} />
              <span className="text-[9px] font-black uppercase tracking-wide">{item.label}</span>
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
