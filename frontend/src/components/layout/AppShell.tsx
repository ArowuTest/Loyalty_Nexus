"use client";

import { ReactNode, useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { useStore } from "@/store/useStore";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard, Zap, Wand2, Gift, Ticket, Settings, LogOut, Shield, Bell, Globe
} from "lucide-react";

const NAV_ITEMS = [
  { href: "/dashboard",  icon: LayoutDashboard, label: "Home" },
  { href: "/spin",       icon: Zap,             label: "Spin" },
  { href: "/prizes",     icon: Gift,            label: "Prizes" },
  { href: "/studio",     icon: Wand2,           label: "Studio" },
  { href: "/wars",       icon: Globe,           label: "Wars" },
];

export default function AppShell({ children }: { children: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, logout, user } = useStore();

  useEffect(() => {
    if (!isAuthenticated) router.push("/");
  }, [isAuthenticated, router]);

  if (!isAuthenticated) return null;

  return (
    <div className="min-h-screen bg-[rgb(15_17_35)] flex flex-col">
      {/* Top bar (desktop) */}
      <header className="hidden md:flex items-center justify-between px-6 py-4 glass border-b border-nexus-600/10 sticky top-0 z-50">
        <Link href="/dashboard" className="flex items-center gap-2">
          <span className="text-2xl">⚡</span>
          <span className="font-display text-lg font-bold text-white">Loyalty Nexus</span>
        </Link>
        <nav className="flex gap-1">
          {NAV_ITEMS.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition-all",
                pathname === item.href
                  ? "bg-nexus-600/20 text-nexus-400"
                  : "text-[rgb(130_140_180)] hover:text-white hover:bg-white/5"
              )}
            >
              <item.icon size={16} />
              {item.label}
            </Link>
          ))}
          {/* Prizes in desktop nav only (not in mobile bottom bar to save space) */}
          <Link
            href="/prizes"
            className={cn(
              "flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition-all",
              pathname === "/prizes"
                ? "bg-nexus-600/20 text-nexus-400"
                : "text-[rgb(130_140_180)] hover:text-white hover:bg-white/5"
            )}
          >
            <Gift size={16} />
            Prizes
          </Link>
        </nav>
        <div className="flex items-center gap-3">
          <span className={cn("tier-badge", `tier-${user?.tier || "BRONZE"}`)}>{user?.tier || "BRONZE"}</span>
          <Link href="/notifications" className="text-[rgb(130_140_180)] hover:text-white transition-colors" title="Notifications">
                <Bell size={18} />
              </Link>
              <Link href="/settings" className="text-[rgb(130_140_180)] hover:text-white transition-colors">
            <Settings size={18} />
          </Link>
          <button
            onClick={() => { logout(); router.push("/"); }}
            className="text-[rgb(130_140_180)] hover:text-red-400 transition-colors"
          >
            <LogOut size={18} />
          </button>
        </div>
      </header>

      {/* Main content */}
      <main className="flex-1 pb-24 md:pb-8">{children}</main>

      {/* Bottom nav (mobile) — 5 items max for readability */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 glass border-t border-nexus-600/10 flex justify-around py-2 z-50">
        {NAV_ITEMS.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={cn(
              "flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl transition-all",
              pathname === item.href ? "text-nexus-400" : "text-[rgb(130_140_180)]"
            )}
          >
            <item.icon size={20} />
            <span className="text-[10px] font-medium">{item.label}</span>
          </Link>
        ))}
      </nav>
    </div>
  );
}
