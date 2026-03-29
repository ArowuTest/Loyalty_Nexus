import React, { useState, useEffect } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { motion, AnimatePresence } from "framer-motion";
import { Menu, X, Zap, Sparkles } from "lucide-react";
import { ROUTES, TIER_CONFIG, formatPoints, type Tier } from "@/lib";
import { MOCK_USER } from "@/data";

interface NavBarProps {
  onLoginClick?: () => void;
  isLoggedIn?: boolean;
}

export default function NavBar({ onLoginClick, isLoggedIn = false }: NavBarProps) {
  const [scrolled, setScrolled] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    const fn = () => setScrolled(window.scrollY > 24);
    window.addEventListener("scroll", fn, { passive: true });
    return () => window.removeEventListener("scroll", fn);
  }, []);
  useEffect(() => setMenuOpen(false), [location]);

  // Scroll to a section on the home page; if not on home page, navigate there first
  const scrollToSection = (sectionId: string) => {
    setMenuOpen(false);
    if (location.pathname !== "/") {
      navigate("/");
      // Use a small delay to allow navigation before scrolling
      setTimeout(() => {
        document.getElementById(sectionId)?.scrollIntoView({ behavior: "smooth", block: "start" });
      }, 150);
    } else {
      document.getElementById(sectionId)?.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  };

  const handleDashboard = () => {
    setMenuOpen(false);
    if (!isLoggedIn && onLoginClick) {
      onLoginClick();
    } else {
      navigate(ROUTES.DASHBOARD);
    }
  };

  const tier = MOCK_USER.tier as Tier;
  const tc   = TIER_CONFIG[tier];

  // Nav items: Home, AI Studio (anchor), Wars (anchor), Dashboard (auth-guarded)
  const navItems = [
    { label: "Home",       type: "link",    to: ROUTES.HOME },
    { label: "AI Studio",  type: "anchor",  section: "ai-studio" },
    { label: "Wars",       type: "anchor",  section: "regional-wars" },
    { label: "Dashboard",  type: "action",  action: handleDashboard },
  ];

  return (
    <motion.header
      initial={{ y: -64, opacity: 0 }}
      animate={{ y: 0,   opacity: 1 }}
      transition={{ type: "spring", stiffness: 280, damping: 28 }}
      className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
        scrolled ? "glass-strong border-b border-white/[0.07]" : "bg-transparent"
      }`}
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="flex items-center justify-between h-16">

          {/* Logo */}
          <Link to={ROUTES.HOME} className="flex items-center gap-2.5 group flex-shrink-0">
            <div className="w-9 h-9 rounded-xl bg-gold flex items-center justify-center glow-gold flex-shrink-0">
              <Zap className="w-5 h-5 text-black" />
            </div>
            <span className="text-[17px] font-black tracking-[-0.01em]">
              <span className="text-gold">Loyalty</span>
              <span className="text-foreground"> Nexus</span>
            </span>
          </Link>

          {/* Desktop nav */}
          <nav className="hidden md:flex items-center gap-0.5">
            {navItems.map((item) => {
              const isActive =
                item.type === "link" && location.pathname === item.to;
              const isActiveAnchor =
                item.type === "action" && location.pathname === ROUTES.DASHBOARD;

              const base =
                "px-4 py-2 rounded-xl text-[13px] font-semibold transition-all duration-200 cursor-pointer";
              const activeClass = "bg-primary/12 text-primary";
              const inactiveClass =
                "text-muted-foreground hover:text-foreground hover:bg-white/[0.06]";

              if (item.type === "link") {
                return (
                  <Link
                    key={item.label}
                    to={item.to!}
                    className={`${base} ${isActive ? activeClass : inactiveClass}`}
                  >
                    {item.label}
                  </Link>
                );
              }
              if (item.type === "anchor") {
                return (
                  <button
                    key={item.label}
                    onClick={() => scrollToSection(item.section!)}
                    className={`${base} ${inactiveClass}`}
                  >
                    {item.label}
                  </button>
                );
              }
              // action (Dashboard)
              return (
                <button
                  key={item.label}
                  onClick={item.action}
                  className={`${base} ${
                    isActiveAnchor ? activeClass : inactiveClass
                  }`}
                >
                  {item.label}
                </button>
              );
            })}
          </nav>

          {/* Right side */}
          <div className="hidden md:flex items-center gap-2.5">
            {isLoggedIn ? (
              <Link to={ROUTES.DASHBOARD}>
                <div className="flex items-center gap-2.5 glass border border-white/[0.10] rounded-full pl-1.5 pr-3 py-1.5 hover:border-primary/30 transition-all cursor-pointer">
                  <div
                    className="w-7 h-7 rounded-full flex items-center justify-center text-[12px] font-black text-black flex-shrink-0"
                    style={{ background: tc.color }}
                  >
                    {MOCK_USER.display_name[0]}
                  </div>
                  <div className="flex flex-col leading-none">
                    <span className="text-[12px] font-bold text-foreground">{MOCK_USER.display_name}</span>
                    <span className="text-[10px] font-mono text-primary">{formatPoints(MOCK_USER.pulse_points)} pts</span>
                  </div>
                  <span className="text-base ml-0.5">{tc.icon}</span>
                </div>
              </Link>
            ) : (
              <>
                <button
                  onClick={onLoginClick}
                  className="text-[13px] font-semibold text-muted-foreground hover:text-foreground transition-colors px-3 py-2"
                >
                  Sign In
                </button>
                <button
                  onClick={onLoginClick}
                  className="btn-gold rounded-xl h-9 px-5 text-[13px] font-black glow-gold inline-flex items-center gap-1.5"
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
          >
            {menuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
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
              {navItems.map((item) => {
                if (item.type === "link") {
                  return (
                    <Link
                      key={item.label}
                      to={item.to!}
                      className="px-4 py-3 rounded-xl text-[14px] font-semibold text-muted-foreground hover:text-foreground hover:bg-white/[0.06] transition-all"
                    >
                      {item.label}
                    </Link>
                  );
                }
                if (item.type === "anchor") {
                  return (
                    <button
                      key={item.label}
                      onClick={() => scrollToSection(item.section!)}
                      className="px-4 py-3 rounded-xl text-[14px] font-semibold text-muted-foreground hover:text-foreground hover:bg-white/[0.06] transition-all text-left"
                    >
                      {item.label}
                    </button>
                  );
                }
                return (
                  <button
                    key={item.label}
                    onClick={item.action}
                    className="px-4 py-3 rounded-xl text-[14px] font-semibold text-muted-foreground hover:text-foreground hover:bg-white/[0.06] transition-all text-left"
                  >
                    {item.label}
                  </button>
                );
              })}
              <div className="pt-3 mt-1 border-t border-white/[0.07]">
                <button
                  onClick={isLoggedIn ? () => navigate(ROUTES.DASHBOARD) : onLoginClick}
                  className="btn-gold rounded-xl h-12 w-full text-[15px] font-black glow-gold inline-flex items-center justify-center gap-2"
                >
                  <Zap className="w-5 h-5" />
                  {isLoggedIn ? "My Dashboard" : "Sign In / Register"}
                </button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.header>
  );
}
