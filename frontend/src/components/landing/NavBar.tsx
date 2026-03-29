"use client";
import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { Zap, Sparkles, Menu, X } from "lucide-react";
import { useStore } from "@/store/useStore";

const NAV_LINKS = [
  { label: "Home",      href: "/",          anchor: null,           needsAuth: false },
  { label: "AI Studio", href: "/studio",    anchor: "ai-studio",    needsAuth: false },
  { label: "Dashboard", href: "/dashboard", anchor: null,           needsAuth: true  },
  { label: "Wars",      href: "/wars",      anchor: "regional-wars", needsAuth: false },
];

const TIER_COLORS: Record<string, string> = {
  BRONZE:   "#CD7F32",
  SILVER:   "#C0C0C0",
  GOLD:     "#F5A623",
  PLATINUM: "#E5E4E2",
  DIAMOND:  "#B9F2FF",
};
const TIER_ICONS: Record<string, string> = {
  BRONZE: "🥉", SILVER: "🥈", GOLD: "🥇", PLATINUM: "💎", DIAMOND: "💠",
};

interface NavBarProps {
  onLoginClick: () => void;
}

export default function NavBar({ onLoginClick }: NavBarProps) {
  const [scrolled, setScrolled]   = useState(false);
  const [menuOpen, setMenuOpen]   = useState(false);
  const pathname                  = usePathname();
  const router                    = useRouter();
  const { isAuthenticated, user, _hasHydrated } = useStore();

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 12);
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  // Close mobile menu on route change
  useEffect(() => { setMenuOpen(false); }, [pathname]);

  const tier      = user?.tier?.toUpperCase() ?? "BRONZE";
  const tierColor = TIER_COLORS[tier] ?? "#CD7F32";
  const tierIcon  = TIER_ICONS[tier]  ?? "🥉";
  const initial   = user?.phone_number?.slice(-4) ?? "?";

  const handleAuthNav = (href: string) => {
    if (!_hasHydrated) return;
    if (isAuthenticated) {
      router.push(href);
    } else {
      onLoginClick();
    }
  };

  // Scroll to anchor on home page, or navigate with hash
  const handleAnchorNav = (href: string, anchor: string | null) => {
    if (!anchor) { router.push(href); return; }
    if (pathname === "/") {
      document.getElementById(anchor)?.scrollIntoView({ behavior: "smooth", block: "start" });
    } else {
      router.push(`/#${anchor}`);
    }
  };

  return (
    <motion.header
      initial={{ y: -20, opacity: 0 }}
      animate={{ y: 0,   opacity: 1 }}
      transition={{ duration: 0.4, ease: "easeOut" }}
      className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
        scrolled ? "glass-strong border-b border-white/[0.07]" : "bg-transparent"
      }`}
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="flex items-center justify-between h-16">

          {/* Logo */}
          <Link href="/" className="flex items-center gap-2.5 group flex-shrink-0">
            <div className="w-9 h-9 rounded-xl bg-gold-500 flex items-center justify-center glow-gold-sm flex-shrink-0">
              <Zap className="w-5 h-5 text-black" />
            </div>
            <span className="text-[17px] font-black tracking-[-0.01em]">
              <span className="text-gold">Loyalty</span>
              <span className="text-white"> Nexus</span>
            </span>
          </Link>

          {/* Desktop nav */}
          <nav className="hidden md:flex items-center gap-0.5">
            {NAV_LINKS.map(({ label, href, anchor, needsAuth }) => {
              const active = pathname === href;
              const baseClass = `px-4 py-2 rounded-xl text-[13px] font-semibold transition-all duration-200 ${
                active ? "bg-gold-500/12 text-gold-500" : "text-white/50 hover:text-white hover:bg-white/[0.06]"
              }`;
              if (needsAuth) {
                return (
                  <button key={href} onClick={() => handleAuthNav(href)} className={baseClass}>{label}</button>
                );
              }
              if (anchor) {
                return (
                  <button key={href} onClick={() => handleAnchorNav(href, anchor)} className={baseClass}>{label}</button>
                );
              }
              return (
                <Link key={href} href={href} className={baseClass}>{label}</Link>
              );
            })}
          </nav>

          {/* Right side */}
          <div className="hidden md:flex items-center gap-2.5">
            {_hasHydrated && isAuthenticated ? (
              <Link href="/dashboard">
                <div className="flex items-center gap-2.5 glass border border-white/[0.10] rounded-full pl-1.5 pr-3 py-1.5 hover:border-gold-500/30 transition-all cursor-pointer">
                  <div
                    className="w-7 h-7 rounded-full flex items-center justify-center text-[12px] font-black text-black flex-shrink-0"
                    style={{ background: tierColor }}
                  >
                    {initial}
                  </div>
                  <div className="flex flex-col leading-none">
                    <span className="text-[12px] font-bold text-white">
                      {user?.phone_number?.slice(-8) ?? "My Account"}
                    </span>
                    <span className="text-[10px] font-mono text-gold-500">{tier}</span>
                  </div>
                  <span className="text-base ml-0.5">{tierIcon}</span>
                </div>
              </Link>
            ) : (
              <>
                <button
                  onClick={onLoginClick}
                  className="text-[13px] font-semibold text-white/50 hover:text-white transition-colors px-3 py-2"
                >
                  Sign In
                </button>
                <button
                  onClick={onLoginClick}
                  className="btn-gold rounded-xl h-9 px-5 text-[13px] font-black glow-gold-sm inline-flex items-center gap-1.5"
                >
                  <Sparkles className="w-3.5 h-3.5" />
                  Get Started
                </button>
              </>
            )}
          </div>

          {/* Mobile hamburger */}
          <button
            className="md:hidden w-9 h-9 rounded-xl hover:bg-white/[0.07] flex items-center justify-center transition-colors"
            onClick={() => setMenuOpen(v => !v)}
            aria-label="Toggle menu"
          >
            {menuOpen ? <X className="w-5 h-5 text-white" /> : <Menu className="w-5 h-5 text-white" />}
          </button>
        </div>
      </div>

      {/* Mobile menu */}
      <AnimatePresence>
        {menuOpen && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.22 }}
            className="md:hidden glass-strong border-t border-white/[0.07]"
          >
            <div className="px-4 py-4 flex flex-col gap-1">
              {NAV_LINKS.map(({ label, href, anchor, needsAuth }) => {
                const mobileClass = "px-4 py-3 rounded-xl text-[14px] font-semibold text-white/50 hover:text-white hover:bg-white/[0.06] transition-all text-left";
                if (needsAuth) {
                  return (
                    <button key={href} onClick={() => { setMenuOpen(false); handleAuthNav(href); }} className={mobileClass}>{label}</button>
                  );
                }
                if (anchor) {
                  return (
                    <button key={href} onClick={() => { setMenuOpen(false); handleAnchorNav(href, anchor); }} className={mobileClass}>{label}</button>
                  );
                }
                return (
                  <Link key={href} href={href} className={mobileClass}>{label}</Link>
                );
              })}
              <div className="pt-3 mt-1 border-t border-white/[0.07]">
                {_hasHydrated && isAuthenticated ? (
                  <Link href="/dashboard">
                    <button className="btn-gold rounded-xl h-12 w-full text-[15px] font-black glow-gold-sm inline-flex items-center justify-center gap-2">
                      <Zap className="w-5 h-5" />
                      My Dashboard
                    </button>
                  </Link>
                ) : (
                  <button
                    onClick={() => { setMenuOpen(false); onLoginClick(); }}
                    className="btn-gold rounded-xl h-12 w-full text-[15px] font-black glow-gold-sm inline-flex items-center justify-center gap-2"
                  >
                    <Zap className="w-5 h-5" />
                    Sign In / Register
                  </button>
                )}
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.header>
  );
}
