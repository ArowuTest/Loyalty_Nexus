"use client";

import { ReactNode, useEffect, useRef, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { useStore } from "@/store/useStore";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard, Zap, Wand2, Gift, Trophy, Bell,
  User, Settings, LogOut, ChevronDown,
} from "lucide-react";

const NAV_ITEMS = [
  { href: "/dashboard", icon: LayoutDashboard, label: "Home"       },
  { href: "/spin",      icon: Zap,             label: "Spin"       },
  { href: "/studio",    icon: Wand2,           label: "Studio"     },
  { href: "/wars",      icon: Trophy,          label: "Wars"       },
  { href: "/prizes",    icon: Gift,            label: "Prizes"     },
];

const TIER_COLORS: Record<string, string> = {
  BRONZE: "#CD7F32", SILVER: "#C0C0C0", GOLD: "#F5A623", PLATINUM: "#E5E4E2", DIAMOND: "#B9F2FF",
};
const TIER_ICONS: Record<string, string> = {
  BRONZE: "🥉", SILVER: "🥈", GOLD: "🥇", PLATINUM: "💎", DIAMOND: "💠",
};

/** Generates initials from display_name or last 2 digits of phone */
function getInitials(user: { display_name?: string; phone_number?: string } | null): string {
  if (!user) return "?";
  if (user.display_name?.trim()) {
    const parts = user.display_name.trim().split(/\s+/);
    if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
    return parts[0].slice(0, 2).toUpperCase();
  }
  return (user.phone_number ?? "").slice(-2) || "?";
}

export default function AppShell({ children }: { children: ReactNode }) {
  const router   = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, _hasHydrated, logout, user, wallet } = useStore();
  const tier      = (user?.tier ?? "BRONZE").toUpperCase();
  const tierColor = TIER_COLORS[tier] ?? "#CD7F32";
  const points    = wallet?.pulse_points;

  // Profile dropdown
  const [dropOpen, setDropOpen] = useState(false);
  const dropRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleOutside(e: MouseEvent) {
      if (dropRef.current && !dropRef.current.contains(e.target as Node)) {
        setDropOpen(false);
      }
    }
    document.addEventListener("mousedown", handleOutside);
    return () => document.removeEventListener("mousedown", handleOutside);
  }, []);

  useEffect(() => {
    if (_hasHydrated && !isAuthenticated) router.push("/");
  }, [_hasHydrated, isAuthenticated, router]);

  useEffect(() => {
    const handleSessionExpired = () => { logout(); router.push("/"); };
    window.addEventListener("nexus:session-expired", handleSessionExpired);
    return () => window.removeEventListener("nexus:session-expired", handleSessionExpired);
  }, [logout, router]);

  if (!_hasHydrated) return null;
  if (!isAuthenticated) return null;

  const initials    = getInitials(user);
  const displayName = user?.display_name?.trim() || user?.phone_number || "";
  const email       = user?.email?.trim() || "";

  const handleLogout = () => { setDropOpen(false); logout(); router.push("/"); };

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
        {/* Logo → landing page */}
        <Link href="/" className="flex items-center gap-2.5">
          <div
            className="w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0"
            style={{ background: "rgba(245,166,35,0.12)", border: "1px solid rgba(245,166,35,0.25)" }}
          >
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
                {item.label === "Spin" ? "Wheel Spin" : item.label}
              </Link>
            );
          })}
        </nav>

        {/* Right side */}
        <div className="flex items-center gap-3">
          {/* Tier badge */}
          <span className={cn("tier-badge", `tier-${user?.tier || "BRONZE"}`)}>
            {TIER_ICONS[tier]} {tier}
          </span>

          {/* Notifications */}
          <Link href="/notifications" className="text-white/40 hover:text-white transition-colors">
            <Bell size={18} />
          </Link>

          {/* Profile avatar + dropdown */}
          <div className="relative" ref={dropRef}>
            <button
              onClick={() => setDropOpen((v) => !v)}
              className="flex items-center gap-1.5 px-2 py-1.5 rounded-xl transition-all hover:bg-white/[0.06]"
              style={{ border: "1px solid rgba(255,255,255,0.08)" }}
              aria-label="Open profile menu"
            >
              <div
                className="w-7 h-7 rounded-full flex items-center justify-center text-[11px] font-black flex-shrink-0"
                style={{ background: `${tierColor}22`, border: `1.5px solid ${tierColor}55`, color: tierColor }}
              >
                {initials}
              </div>
              <ChevronDown
                size={13}
                className={cn("text-white/40 transition-transform duration-200", dropOpen && "rotate-180")}
              />
            </button>

            {/* Dropdown */}
            {dropOpen && (
              <div
                className="absolute right-0 mt-2 w-64 rounded-2xl overflow-hidden shadow-2xl z-50"
                style={{
                  background: "rgba(18,20,28,0.98)",
                  border: "1px solid rgba(255,255,255,0.10)",
                  backdropFilter: "blur(24px)",
                }}
              >
                {/* User info */}
                <div className="px-4 py-4 border-b" style={{ borderColor: "rgba(255,255,255,0.07)" }}>
                  <div className="flex items-center gap-3">
                    <div
                      className="w-10 h-10 rounded-full flex items-center justify-center text-[14px] font-black flex-shrink-0"
                      style={{ background: `${tierColor}22`, border: `2px solid ${tierColor}55`, color: tierColor }}
                    >
                      {initials}
                    </div>
                    <div className="min-w-0">
                      <p className="text-white font-bold text-[13px] truncate">{displayName}</p>
                      {email
                        ? <p className="text-white/40 text-[11px] truncate">{email}</p>
                        : <p className="text-white/25 text-[11px] italic">No email set</p>
                      }
                    </div>
                  </div>
                </div>

                {/* Menu items */}
                <div className="py-1.5">
                  <Link
                    href="/profile"
                    onClick={() => setDropOpen(false)}
                    className="flex items-center gap-3 px-4 py-2.5 text-[13px] text-white/70 hover:text-white hover:bg-white/[0.05] transition-all"
                  >
                    <User size={15} className="flex-shrink-0" />
                    Edit Profile
                  </Link>
                  <Link
                    href="/settings"
                    onClick={() => setDropOpen(false)}
                    className="flex items-center gap-3 px-4 py-2.5 text-[13px] text-white/70 hover:text-white hover:bg-white/[0.05] transition-all"
                  >
                    <Settings size={15} className="flex-shrink-0" />
                    Settings
                  </Link>
                  <div className="my-1 mx-3" style={{ borderTop: "1px solid rgba(255,255,255,0.07)" }} />
                  <button
                    onClick={handleLogout}
                    className="w-full flex items-center gap-3 px-4 py-2.5 text-[13px] text-red-400/80 hover:text-red-400 hover:bg-red-500/[0.06] transition-all"
                  >
                    <LogOut size={15} className="flex-shrink-0" />
                    Sign Out
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* ── Mobile top bar ── */}
      <header
        className="md:hidden flex items-center justify-between px-4 py-3 sticky top-0 z-50"
        style={{
          background: "rgba(13,14,20,0.94)",
          backdropFilter: "blur(20px)",
          WebkitBackdropFilter: "blur(20px)",
          borderBottom: "1px solid rgba(255,255,255,0.06)",
        }}
      >
        {/* Logo */}
        <Link href="/" className="flex items-center gap-2">
          <div
            className="w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0"
            style={{ background: "rgba(245,166,35,0.12)", border: "1px solid rgba(245,166,35,0.25)" }}
          >
            <Zap size={14} style={{ color: "var(--gold)" }} />
          </div>
          <span className="font-black text-[14px] text-white tracking-tight">Loyalty Nexus</span>
        </Link>

        {/* Right: points + profile */}
        <div className="flex items-center gap-2">
          {/* Points pill */}
          {points !== undefined && (
            <div
              className="flex items-center gap-1 px-2.5 py-1 rounded-full text-[11px] font-black"
              style={{ background: "rgba(245,166,35,0.12)", border: "1px solid rgba(245,166,35,0.25)", color: "var(--gold)" }}
            >
              <Zap size={11} />
              {points >= 1000 ? `${(points / 1000).toFixed(1)}K` : points.toLocaleString()}
            </div>
          )}

          {/* Profile avatar → profile page */}
          <Link href="/profile">
            <div
              className="w-8 h-8 rounded-full flex items-center justify-center text-[11px] font-black"
              style={{ background: `${tierColor}22`, border: `1.5px solid ${tierColor}55`, color: tierColor }}
            >
              {initials}
            </div>
          </Link>
        </div>
      </header>

      {/* ── Main content ── */}
      {/* Studio manages its own height/overflow; other pages need bottom-nav padding */}
      <main className={cn("flex-1", pathname !== "/studio" && "pb-24 md:pb-8")}>{children}</main>

      {/* ── Mobile bottom nav ── */}
      <nav
        className="md:hidden fixed bottom-0 left-0 right-0 z-50"
        style={{
          background: "rgba(13,14,20,0.96)",
          backdropFilter: "blur(20px)",
          WebkitBackdropFilter: "blur(20px)",
          borderTop: "1px solid rgba(255,255,255,0.06)",
          paddingBottom: "env(safe-area-inset-bottom, 0px)",
        }}
      >
        <div className="flex justify-around py-2">
          {NAV_ITEMS.map((item) => {
            const active = pathname === item.href || pathname.startsWith(item.href + "/");
            return (
              <Link
                key={item.href}
                href={item.href}
                className="flex flex-col items-center gap-0.5 px-2 py-1.5 rounded-xl transition-all min-w-[48px]"
                style={{ color: active ? "var(--gold)" : "rgba(255,255,255,0.35)" }}
              >
                <item.icon size={20} />
                <span className="text-[9px] font-black uppercase tracking-wide leading-none">{item.label}</span>
              </Link>
            );
          })}
          {/* Profile icon — mobile */}
          <Link
            href="/profile"
            className="flex flex-col items-center gap-0.5 px-2 py-1.5 rounded-xl transition-all min-w-[48px]"
            style={{ color: pathname === "/profile" ? "var(--gold)" : "rgba(255,255,255,0.35)" }}
          >
            <User size={20} />
            <span className="text-[9px] font-black uppercase tracking-wide leading-none">Profile</span>
          </Link>
        </div>
      </nav>
    </div>
  );
}
