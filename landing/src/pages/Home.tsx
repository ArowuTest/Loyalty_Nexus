import React, { useState, useEffect, useRef, useCallback } from "react";
import { Link } from "react-router-dom";
import { motion, useScroll, useTransform, AnimatePresence, useInView } from "framer-motion";
import {
  Zap, Sparkles, ArrowRight, Trophy, Users, TrendingUp,
  CheckCircle2, Play, ChevronRight, Star,
  Lock, Phone, Wallet, RotateCcw, Shield
} from "lucide-react";
import NavBar from "@/components/NavBar";
import Footer from "@/components/Footer";
import AuthModal from "@/components/AuthModal";
import { ROUTES, TIER_CONFIG, formatPoints, formatNaira } from "@/lib";
import { AI_TOOLS, SPIN_PRIZES } from "@/data";

// ─── Announcement Banner ──────────────────────────────────────
function AnnouncementBanner({ onLoginClick }: { onLoginClick: () => void }) {
  const [visible, setVisible] = useState(true);
  if (!visible) return null;
  return (
    <motion.div
      initial={{ opacity: 0, y: -40 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -40 }}
      transition={{ type: "spring", stiffness: 300, damping: 28, delay: 0.8 }}
      className="fixed top-16 left-0 right-0 z-40 flex justify-center px-3 sm:px-4 pointer-events-none"
    >
      <div
        className="pointer-events-auto w-full max-w-3xl flex items-center justify-between gap-3 rounded-2xl px-4 py-2.5 border border-primary/25 shadow-lg"
        style={{
          background: "linear-gradient(135deg, oklch(0.14 0.03 47 / 0.90) 0%, oklch(0.12 0.02 240 / 0.90) 100%)",
          backdropFilter: "blur(20px)",
          WebkitBackdropFilter: "blur(20px)",
        }}
      >
        {/* Left — icon + text */}
        <div className="flex items-center gap-2.5 min-w-0">
          <div className="w-7 h-7 rounded-lg bg-gold flex items-center justify-center flex-shrink-0 glow-gold">
            <Zap className="w-3.5 h-3.5 text-black" />
          </div>
          <p className="text-[12px] sm:text-[13px] text-muted-foreground leading-snug min-w-0">
            <button
              onClick={onLoginClick}
              className="font-black text-primary hover:underline underline-offset-2 mr-1"
            >
              Sign in
            </button>
            to see your Pulse Points balance, spin the wheel for recharge rewards, and unlock{" "}
            <span className="font-bold text-foreground">30+ premium AI tools</span>{" "}
            — all earned from your everyday MTN recharges.
          </p>
        </div>

        {/* Right — CTA + close */}
        <div className="flex items-center gap-2 flex-shrink-0">
          <button
            onClick={onLoginClick}
            className="hidden sm:inline-flex items-center gap-1.5 btn-gold rounded-xl h-8 px-4 text-[11px] font-black whitespace-nowrap"
          >
            <Sparkles className="w-3 h-3" />
            Get Started
          </button>
          <button
            onClick={() => setVisible(false)}
            className="w-6 h-6 rounded-lg hover:bg-white/[0.10] flex items-center justify-center transition-colors text-muted-foreground hover:text-foreground"
          >
            <span className="text-xs leading-none">✕</span>
          </button>
        </div>
      </div>
    </motion.div>
  );
}

// ─── Animated counter ─────────────────────────────────────────
function Counter({ to, suffix = "", duration = 1800 }: { to: number; suffix?: string; duration?: number }) {
  const [count, setCount] = useState(0);
  const ref = useRef<HTMLSpanElement>(null);
  const inView = useInView(ref, { once: true });
  useEffect(() => {
    if (!inView) return;
    let start = 0;
    const steps = 60;
    const increment = to / steps;
    const interval = duration / steps;
    const timer = setInterval(() => {
      start += increment;
      if (start >= to) { setCount(to); clearInterval(timer); }
      else setCount(Math.floor(start));
    }, interval);
    return () => clearInterval(timer);
  }, [inView, to, duration]);
  return <span ref={ref}>{count.toLocaleString("en-NG")}{suffix}</span>;
}

// ─── Live activity ticker ──────────────────────────────────────
const ACTIVITY_EVENTS = [
  { emoji: "🎉", text: "Chioma A. won ₦5,000 cash" },
  { emoji: "⚡", text: "Tunde O. earned 2,400 Pulse Points" },
  { emoji: "🎨", text: "Amina K. created an AI photo" },
  { emoji: "🎯", text: "Emeka N. spun the wheel and won 2GB data" },
  { emoji: "💼", text: "Fatima B. generated a business plan" },
  { emoji: "🔊", text: "Biodun S. created a marketing jingle" },
  { emoji: "📸", text: "Kemi A. removed background from 10 photos" },
  { emoji: "🏆", text: "Seun L. reached Gold tier" },
  { emoji: "🎤", text: "Adaeze M. converted voice to business plan" },
  { emoji: "💫", text: "Yusuf K. referred 5 friends, earned 2,500 pts" },
];

function LiveTicker() {
  const doubled = [...ACTIVITY_EVENTS, ...ACTIVITY_EVENTS];
  return (
    <div className="relative overflow-hidden py-3 border-y border-white/[0.06]" style={{ background: "oklch(0.10 0.01 240)" }}>
      {/* left fade */}
      <div className="absolute left-0 top-0 bottom-0 w-20 z-10 pointer-events-none"
        style={{ background: "linear-gradient(to right, oklch(0.10 0.01 240), transparent)" }} />
      {/* right fade */}
      <div className="absolute right-0 top-0 bottom-0 w-20 z-10 pointer-events-none"
        style={{ background: "linear-gradient(to left, oklch(0.10 0.01 240), transparent)" }} />

      <div className="flex animate-ticker whitespace-nowrap">
        {doubled.map((ev, i) => (
          <div key={i} className="inline-flex items-center gap-2 px-8 text-sm text-muted-foreground">
            <span>{ev.emoji}</span>
            <span>{ev.text}</span>
            <span className="text-primary/30 mx-2">•</span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── Aurora hero background (canvas) ──────────────────────────
function AuroraCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    let raf: number;
    let t = 0;

    const resize = () => {
      canvas.width  = canvas.offsetWidth  * Math.min(window.devicePixelRatio, 2);
      canvas.height = canvas.offsetHeight * Math.min(window.devicePixelRatio, 2);
      ctx.scale(Math.min(window.devicePixelRatio, 2), Math.min(window.devicePixelRatio, 2));
    };
    resize();
    window.addEventListener("resize", resize);

    const orbs = [
      { x: 0.50, y: 0.10, rx: 0.55, ry: 0.42, color: "rgba(245,166,35,",  speed: 0.00018, opacity: 0.22 },
      { x: 0.20, y: 0.55, rx: 0.40, ry: 0.35, color: "rgba(0,212,255,",   speed: 0.00013, opacity: 0.13 },
      { x: 0.80, y: 0.40, rx: 0.38, ry: 0.45, color: "rgba(139,92,246,",  speed: 0.00021, opacity: 0.10 },
      { x: 0.50, y: 0.80, rx: 0.50, ry: 0.30, color: "rgba(245,166,35,",  speed: 0.00015, opacity: 0.12 },
    ];

    const draw = () => {
      t++;
      const W = canvas.offsetWidth;
      const H = canvas.offsetHeight;
      ctx.clearRect(0, 0, W, H);

      orbs.forEach((o, i) => {
        const ox = W * (o.x + Math.sin(t * o.speed * Math.PI * 2 + i) * 0.08);
        const oy = H * (o.y + Math.cos(t * o.speed * Math.PI * 2 + i * 1.3) * 0.06);
        const grd = ctx.createRadialGradient(ox, oy, 0, ox, oy, W * o.rx);
        grd.addColorStop(0, `${o.color}${o.opacity})`);
        grd.addColorStop(1, `${o.color}0)`);
        ctx.fillStyle = grd;
        ctx.fillRect(0, 0, W, H);
      });

      raf = requestAnimationFrame(draw);
    };
    draw();
    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", resize);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className="absolute inset-0 w-full h-full pointer-events-none"
      style={{ mixBlendMode: "screen" }}
    />
  );
}

// ─── Floating AI tool card (hero decoration) ───────────────────
function FloatingToolCard({ emoji, label, pts, delay, className }: {
  emoji: string; label: string; pts: string; delay: number; className?: string;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20, scale: 0.90 }}
      animate={{ opacity: 1, y: 0,  scale: 1 }}
      transition={{ type: "spring", stiffness: 200, damping: 26, delay }}
      className={`glass border-gold-gradient rounded-2xl px-4 py-3 flex items-center gap-3 select-none ${className ?? ""}`}
      style={{ animation: `float-y ${3.5 + delay}s ease-in-out infinite` }}
    >
      <span className="text-2xl">{emoji}</span>
      <div>
        <p className="text-[13px] font-bold text-foreground leading-tight">{label}</p>
        <p className="text-[11px] font-mono text-primary">{pts}</p>
      </div>
    </motion.div>
  );
}

// ─── Feature pill ──────────────────────────────────────────────
function FeaturePill({ icon: Icon, label, color }: { icon: React.ElementType; label: string; color: string }) {
  return (
    <div className="inline-flex items-center gap-2 glass rounded-full px-3.5 py-1.5 border border-white/[0.07]">
      <Icon className={`w-3.5 h-3.5 ${color}`} />
      <span className="text-xs font-semibold text-muted-foreground">{label}</span>
    </div>
  );
}

// ─── Section header ────────────────────────────────────────────
function SectionHeader({ eyebrow, title, sub, light }: {
  eyebrow: string; title: React.ReactNode; sub?: string; light?: boolean;
}) {
  const ref = useRef(null);
  const inView = useInView(ref, { once: true, margin: "-80px" });
  return (
    <div ref={ref} className="text-center max-w-3xl mx-auto mb-14 px-4">
      <motion.p
        initial={{ opacity: 0, y: 10 }}
        animate={inView ? { opacity: 1, y: 0 } : {}}
        transition={{ duration: 0.5 }}
        className="text-xs font-black uppercase tracking-[0.22em] text-primary mb-3"
      >
        {eyebrow}
      </motion.p>
      <motion.h2
        initial={{ opacity: 0, y: 18 }}
        animate={inView ? { opacity: 1, y: 0 } : {}}
        transition={{ duration: 0.55, delay: 0.07 }}
        className="text-4xl sm:text-5xl font-black tracking-tight leading-[1.08] text-foreground"
      >
        {title}
      </motion.h2>
      {sub && (
        <motion.p
          initial={{ opacity: 0, y: 12 }}
          animate={inView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.14 }}
          className="mt-4 text-base text-muted-foreground leading-relaxed"
        >
          {sub}
        </motion.p>
      )}
    </div>
  );
}

// ─── Stagger wrapper ───────────────────────────────────────────
function StaggerGrid({ children, className }: { children: React.ReactNode; className?: string }) {
  const ref = useRef(null);
  const inView = useInView(ref, { once: true, margin: "-60px" });
  return (
    <motion.div
      ref={ref}
      initial="hidden"
      animate={inView ? "visible" : "hidden"}
      variants={{ hidden: {}, visible: { transition: { staggerChildren: 0.09 } } }}
      className={className}
    >
      {children}
    </motion.div>
  );
}
const fadeUp = {
  hidden:  { opacity: 0, y: 28 },
  visible: { opacity: 1, y: 0, transition: { type: "spring" as const, stiffness: 260, damping: 28 } },
};

/* ═══════════════════════════════════════════════════════════
   MAIN COMPONENT
═══════════════════════════════════════════════════════════ */
export default function Home() {
  const [authOpen, setAuthOpen] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const heroRef = useRef<HTMLElement>(null);

  const { scrollYProgress } = useScroll({ target: heroRef, offset: ["start start", "end start"] });
  const heroY     = useTransform(scrollYProgress, [0, 1], [0, 80]);
  const heroOpacity = useTransform(scrollYProgress, [0, 0.7], [1, 0]);

  const login = () => setIsLoggedIn(true);

  /* ─ HOW IT WORKS steps ─ */
  const steps = [
    {
      n: "01", icon: "📱", color: "#F5A623",
      title: "Recharge MTN",
      body: "Buy airtime or data as you normally do. Any amount — ₦100 to ₦50,000. Every single naira works for you now.",
      stat: "₦1 = 1 Point, instantly",
    },
    {
      n: "02", icon: "⚡", color: "#FFE066",
      title: "Earn Pulse Points",
      body: "Points hit your wallet instantly after every recharge. First recharge of the day earns 2× bonus points. Refer friends for 500 pts each.",
      stat: "Avg 1,098 pts/user/month",
    },
    {
      n: "03", icon: "🎡", color: "#00D4FF",
      title: "Free Spin Every Recharge",
      body: "Every MTN recharge of ₦1,000 or more earns you a free wheel spin — no extra steps. Win instant cash, data bundles, airtime or bonus Pulse Points. Extra spins available with your points.",
      stat: "₦18M+ prizes distributed",
    },
    {
      n: "04", icon: "🚀", color: "#8B5CF6",
      title: "Unlock AI Studio",
      body: "Spend points to access 30+ AI tools — create stunning photos, generate videos, build business plans, make music, and more.",
      stat: "1.2M+ generations created",
    },
  ];

  /* ─ AI tool categories preview ─ */
  const toolCategories = [
    { key: "chat",   label: "Chat & Search", emoji: "💬", color: "#00D4FF", bg: "rgba(0,212,255,0.08)",  border: "rgba(0,212,255,0.20)", tools: ["Ask Nexus — FREE", "Web Search AI — FREE", "Code Helper — FREE"] },
    { key: "create", label: "Create",        emoji: "🎨", color: "#F5A623", bg: "rgba(245,166,35,0.08)", border: "rgba(245,166,35,0.20)", tools: ["AI Photo (10 pts)", "AI Photo Dream (12 pts)", "BG Remover (3 pts)", "Video Cinematic (65 pts)"] },
    { key: "learn",  label: "Learn",         emoji: "📚", color: "#10B981", bg: "rgba(16,185,129,0.08)", border: "rgba(16,185,129,0.20)", tools: ["Study Guide (5 pts)", "Quiz Maker (3 pts)", "Mind Map (5 pts)", "AI Podcast (50 pts)"] },
    { key: "build",  label: "Build",         emoji: "🏗️", color: "#8B5CF6", bg: "rgba(139,92,246,0.08)", border: "rgba(139,92,246,0.20)", tools: ["Slide Deck (20 pts)", "Business Plan (30 pts)", "Voice to Plan (35 pts)"] },
  ];

  return (
    <div className="min-h-screen bg-surface-0 dark overflow-x-hidden">
      <NavBar onLoginClick={() => setAuthOpen(true)} isLoggedIn={isLoggedIn} />
      {!isLoggedIn && <AnnouncementBanner onLoginClick={() => setAuthOpen(true)} />}
      <AuthModal isOpen={authOpen} onClose={() => setAuthOpen(false)} onSuccess={login} />

      {/* ══════════════════════════════════════════════════════
          HERO
      ══════════════════════════════════════════════════════ */}
      <section ref={heroRef} className="relative min-h-[100dvh] flex items-center justify-center overflow-hidden bg-hero">
        {/* Aurora canvas */}
        <AuroraCanvas />

        {/* Grid overlay */}
        <div
          className="absolute inset-0 pointer-events-none opacity-[0.025]"
          style={{
            backgroundImage: `linear-gradient(oklch(0.94 0.004 240) 1px, transparent 1px), linear-gradient(90deg, oklch(0.94 0.004 240) 1px, transparent 1px)`,
            backgroundSize: "40px 40px",
          }}
        />

        {/* Content */}
        <motion.div
          style={{ y: heroY, opacity: heroOpacity }}
          className="relative z-10 w-full max-w-6xl mx-auto px-4 sm:px-6 pt-24 pb-16 flex flex-col items-center"
        >
          {/* Eyebrow badge */}
          <motion.div
            initial={{ opacity: 0, y: 16, scale: 0.94 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            transition={{ type: "spring", stiffness: 280, damping: 24, delay: 0.05 }}
            className="inline-flex items-center gap-2 glass-gold rounded-full px-5 py-2 mb-8 cursor-default select-none"
          >
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-primary" />
            </span>
            <span className="text-[13px] font-bold text-primary">84,231 Nigerians earning right now</span>
            <span className="hidden sm:inline text-muted-foreground/50 mx-1">·</span>
            <span className="hidden sm:inline text-[12px] font-semibold text-muted-foreground">🇳🇬 Made for Africa</span>
          </motion.div>

          {/* Main headline */}
          <div className="text-center mb-6 px-2">
            <motion.h1
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ type: "spring", stiffness: 220, damping: 28, delay: 0.10 }}
              className="text-[clamp(3rem,10vw,7rem)] font-black tracking-[-0.03em] leading-[1.0] mb-3"
            >
              <span className="text-foreground block">Recharge.</span>
              <span className="shimmer-text block">Earn. Create.</span>
            </motion.h1>

            <motion.p
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ type: "spring", stiffness: 220, damping: 28, delay: 0.18 }}
              className="text-[clamp(1rem,2.5vw,1.25rem)] text-muted-foreground max-w-xl mx-auto leading-relaxed"
            >
              Recharge MTN <span className="font-black text-foreground">₦1,000+</span> and get a{" "}
              <span className="text-primary font-bold">free wheel spin</span> — win cash up to ₦5,000, data & airtime instantly.
              Plus earn <span className="text-primary font-bold">Pulse Points</span> for 30+ AI tools.
            </motion.p>
          </div>

          {/* Feature pills */}
          <motion.div
            initial={{ opacity: 0, y: 14 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.25 }}
            className="flex flex-wrap items-center justify-center gap-2 mb-9"
          >
            <FeaturePill icon={RotateCcw} label="Free spin on ₦1,000+ recharge" color="text-primary" />
            <FeaturePill icon={Trophy}    label="Win up to ₦5,000 instantly"     color="text-chart-2" />
            <FeaturePill icon={Zap}       label="Pulse Points on every recharge" color="text-chart-3" />
            <FeaturePill icon={Sparkles}  label="30+ AI tools unlocked"          color="text-chart-4" />
          </motion.div>

          {/* CTAs */}
          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.30 }}
            className="flex flex-col sm:flex-row gap-3 mb-8"
          >
            <button
              onClick={() => setAuthOpen(true)}
              className="btn-gold rounded-2xl h-14 px-8 text-[15px] font-black glow-gold inline-flex items-center justify-center gap-2 min-w-[220px]"
            >
              <Zap className="w-5 h-5" />
              Start Earning — It's Free
            </button>
            <Link
              to={ROUTES.STUDIO}
              className="inline-flex items-center justify-center gap-2 glass rounded-2xl h-14 px-8 text-[15px] font-semibold border border-white/[0.12] text-foreground hover:border-white/25 transition-all duration-200 min-w-[200px]"
            >
              <Play className="w-4 h-4 fill-current" />
              Try AI Studio Free
            </Link>
          </motion.div>

          {/* Social proof row */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.40 }}
            className="flex items-center gap-4 text-sm text-muted-foreground"
          >
            <div className="flex -space-x-2.5">
              {["C","T","A","E","F","K"].map((l, i) => (
                <div
                  key={i}
                  className="w-8 h-8 rounded-full border-2 flex items-center justify-center text-[11px] font-black text-black"
                  style={{
                    borderColor: "var(--surface-0)",
                    background: ["#F5A623","#FFE066","#00D4FF","#8B5CF6","#10B981","#F472B6"][i],
                  }}
                >
                  {l}
                </div>
              ))}
            </div>
            <div className="hidden sm:flex items-center gap-1">
              {[...Array(5)].map((_,i) => <Star key={i} className="w-3.5 h-3.5 fill-primary text-primary" />)}
            </div>
            <span><strong className="text-foreground">4.9/5</strong> from 12,000+ reviews</span>
          </motion.div>

          {/* Floating tool cards — desktop only */}
          <div className="hidden lg:block">
            <FloatingToolCard emoji="📸" label="AI Photo"       pts="10 pts"  delay={0.6}
              className="absolute left-[3%] top-[32%]" />
            <FloatingToolCard emoji="🤖" label="Ask Nexus"      pts="FREE"    delay={0.75}
              className="absolute left-[1%] top-[56%]" />
            <FloatingToolCard emoji="🎬" label="Video Cinematic" pts="65 pts" delay={0.65}
              className="absolute right-[3%] top-[30%]" />
            <FloatingToolCard emoji="💼" label="Business Plan"  pts="30 pts"  delay={0.80}
              className="absolute right-[1%] top-[55%]" />
          </div>
        </motion.div>

        {/* Bottom gradient fade into next section */}
        <div className="absolute bottom-0 left-0 right-0 h-32 pointer-events-none"
          style={{ background: "linear-gradient(to bottom, transparent, var(--surface-0))" }} />
      </section>

      {/* ══════════════════════════════════════════════════════
          LIVE TICKER
      ══════════════════════════════════════════════════════ */}
      <LiveTicker />

      {/* ══════════════════════════════════════════════════════
          STATS BAR
      ══════════════════════════════════════════════════════ */}
      <section className="py-16 relative">
        <div className="max-w-5xl mx-auto px-4 sm:px-6">
          <StaggerGrid className="grid grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6">
            {[
              { label: "Active Users",      value: 84231,     suffix: "+",  icon: Users,       color: "#F5A623", pre: "" },
              { label: "AI Generations",    value: 1247903,   suffix: "+",  icon: Sparkles,    color: "#00D4FF", pre: "" },
              { label: "Pulse Points Issued", value: 92000000, suffix: "+", icon: Zap,         color: "#10B981", pre: "" },
              { label: "Prize Money Won",   value: 18000000,  suffix: "+",  icon: Trophy,      color: "#8B5CF6", pre: "₦" },
            ].map(({ label, value, suffix, icon: Icon, color, pre }) => (
              <motion.div key={label} variants={fadeUp}>
                <div
                  className="glass rounded-2xl p-5 border border-white/[0.06] flex flex-col gap-2 group hover:border-white/[0.14] transition-all duration-300"
                  style={{ boxShadow: `0 0 0 0 ${color}` }}
                >
                  <div className="flex items-center gap-2">
                    <div className="p-2 rounded-lg" style={{ background: `${color}18` }}>
                      <Icon className="w-4 h-4" style={{ color }} />
                    </div>
                    <span className="text-xs font-semibold text-muted-foreground">{label}</span>
                  </div>
                  <p className="text-2xl sm:text-3xl font-black tracking-tight" style={{ color }}>
                    {pre}<Counter to={value} suffix={suffix} />
                  </p>
                </div>
              </motion.div>
            ))}
          </StaggerGrid>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          HOW IT WORKS
      ══════════════════════════════════════════════════════ */}
      <section className="py-24 relative overflow-hidden">
        {/* Bg accent */}
        <div className="absolute inset-0 pointer-events-none"
          style={{ background: "radial-gradient(ellipse 80% 50% at 50% 50%, rgba(245,166,35,0.04) 0%, transparent 70%)" }} />

        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <SectionHeader
            eyebrow="How It Works"
            title={<>Simple as <span className="text-gold">1 · 2 · 3 · 4</span></>}
            sub="You already recharge MTN. Now every naira works harder. Here's exactly how it flows."
          />

          <StaggerGrid className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
            {steps.map(({ n, icon, color, title, body, stat }) => (
              <motion.div key={n} variants={fadeUp}>
                <div
                  className="relative glass rounded-2xl p-6 h-full flex flex-col gap-4 border border-white/[0.06] hover:border-white/[0.14] transition-all duration-300 group overflow-hidden"
                >
                  {/* Number watermark */}
                  <span
                    className="absolute -right-3 -top-5 text-8xl font-black select-none pointer-events-none transition-opacity duration-300 group-hover:opacity-30"
                    style={{ color, opacity: 0.06 }}
                  >
                    {n}
                  </span>

                  {/* Icon */}
                  <div
                    className="w-14 h-14 rounded-2xl flex items-center justify-center text-2xl flex-shrink-0 transition-transform duration-300 group-hover:scale-110"
                    style={{ background: `${color}18`, border: `1px solid ${color}30` }}
                  >
                    {icon}
                  </div>

                  <div className="flex-1">
                    <div className="text-[11px] font-black uppercase tracking-widest mb-1" style={{ color }}>Step {n}</div>
                    <h3 className="text-lg font-black text-foreground mb-2 leading-tight">{title}</h3>
                    <p className="text-sm text-muted-foreground leading-relaxed">{body}</p>
                  </div>

                  {/* Stat badge */}
                  <div
                    className="inline-flex items-center gap-1.5 text-[11px] font-bold px-3 py-1.5 rounded-full self-start"
                    style={{ background: `${color}15`, color, border: `1px solid ${color}25` }}
                  >
                    <TrendingUp className="w-3 h-3" />
                    {stat}
                  </div>
                </div>
              </motion.div>
            ))}
          </StaggerGrid>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          AI STUDIO SHOWCASE
      ══════════════════════════════════════════════════════ */}
      <section className="py-24 relative overflow-hidden">
        <div className="absolute inset-0 pointer-events-none"
          style={{ background: "radial-gradient(ellipse 80% 60% at 50% 100%, rgba(0,212,255,0.04) 0%, transparent 70%)" }} />

        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <SectionHeader
            eyebrow="AI Studio"
            title={<><span className="text-gold">30+</span> AI Tools — One Platform</>}
            sub="Chat is always free. Spend your Pulse Points to unlock the most powerful AI creation tools in Africa — photos, videos, music, business plans and beyond."
          />

          {/* Category cards */}
          <StaggerGrid className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
            {toolCategories.map((cat) => (
              <motion.div key={cat.key} variants={fadeUp}>
                <Link to={`${ROUTES.STUDIO}?cat=${cat.key}`}>
                  <div
                    className="glass rounded-2xl p-5 border h-full flex flex-col gap-4 transition-all duration-300 hover:scale-[1.02] cursor-pointer group"
                    style={{ borderColor: cat.border, background: cat.bg }}
                  >
                    <div className="flex items-center justify-between">
                      <div
                        className="w-11 h-11 rounded-xl flex items-center justify-center text-xl"
                        style={{ background: `${cat.color}20`, border: `1px solid ${cat.color}30` }}
                      >
                        {cat.emoji}
                      </div>
                      <ChevronRight className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                    </div>
                    <div>
                      <h3 className="font-black text-base text-foreground mb-2" style={{ color: cat.color }}>
                        {cat.label}
                      </h3>
                      <ul className="space-y-1.5">
                        {cat.tools.map((t) => (
                          <li key={t} className="text-xs text-muted-foreground flex items-center gap-1.5">
                            <span className="w-1 h-1 rounded-full flex-shrink-0" style={{ background: cat.color }} />
                            {t}
                          </li>
                        ))}
                      </ul>
                    </div>
                  </div>
                </Link>
              </motion.div>
            ))}
          </StaggerGrid>

          {/* Tool cards grid */}
          <StaggerGrid className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
            {AI_TOOLS.slice(0, 8).map((tool) => (
              <motion.div key={tool.slug} variants={fadeUp}>
                <div
                  className="glass rounded-xl p-4 border border-white/[0.07] hover:border-white/[0.18] transition-all duration-250 cursor-pointer group hover:scale-[1.02]"
                  onClick={() => !tool.is_free && setAuthOpen(true)}
                >
                  {/* badge */}
                  <div className="flex items-start justify-between mb-3">
                    <span className="text-2xl">{tool.emoji}</span>
                    {tool.is_free ? (
                      <span className="text-[10px] font-black px-2 py-0.5 rounded-full text-cyan-grad"
                        style={{ background: "rgba(0,212,255,0.12)", border: "1px solid rgba(0,212,255,0.25)" }}>
                        FREE
                      </span>
                    ) : (
                      <span className="text-[10px] font-bold text-primary font-mono">
                        {tool.point_cost}pts
                      </span>
                    )}
                  </div>
                  <h4 className="text-[13px] font-bold text-foreground leading-snug mb-1">{tool.name}</h4>
                  <p className="text-[11px] text-muted-foreground line-clamp-2 leading-relaxed">{tool.description}</p>
                  {!tool.is_free && (
                    <div className="mt-3 flex items-center gap-1 text-[11px] text-muted-foreground/50">
                      <Lock className="w-3 h-3" />
                      <span>Sign in to unlock</span>
                    </div>
                  )}
                </div>
              </motion.div>
            ))}
          </StaggerGrid>

          <div className="text-center mt-10">
            <Link to={ROUTES.STUDIO}>
              <button className="inline-flex items-center gap-2 glass border border-white/[0.12] rounded-2xl h-12 px-7 text-sm font-semibold text-foreground hover:border-white/25 transition-all duration-200">
                Explore all 30+ tools
                <ArrowRight className="w-4 h-4" />
              </button>
            </Link>
          </div>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          SPIN & WIN — immersive split
      ══════════════════════════════════════════════════════ */}
      <section className="py-24 relative overflow-hidden">
        <div className="absolute inset-0 pointer-events-none"
          style={{ background: "radial-gradient(ellipse 70% 60% at 20% 50%, rgba(245,166,35,0.06) 0%, transparent 65%)" }} />

        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <div className="glass rounded-3xl border border-white/[0.09] overflow-hidden">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-0">

              {/* Left — text */}
              <div className="p-8 sm:p-12 flex flex-col justify-center">
                <div className="inline-block text-[11px] font-black uppercase tracking-[0.22em] text-primary mb-4">
                  Daily Spin & Win
                </div>
                <h2 className="text-4xl sm:text-5xl font-black text-foreground leading-tight mb-4">
                  Recharge <span className="text-gold">₦1,000+</span>.<br />
                  Get a Free Spin.
                </h2>
                <p className="text-base text-muted-foreground leading-relaxed mb-6">
                  Every MTN recharge of ₦1,000 or above automatically earns you a free wheel spin — no hoops, no codes.
                  Win instant cash, data bundles and airtime directly to your line.
                  Stack more spins with your Pulse Points.
                </p>
                <ul className="space-y-2.5 mb-8">
                  {[
                    { icon: "📱", text: "Airtime up to ₦1,000 per spin" },
                    { icon: "📶", text: "Data packs up to 5GB" },
                    { icon: "💵", text: "Cash prizes up to ₦5,000" },
                    { icon: "⚡", text: "Bonus Pulse Points packs" },
                    { icon: "🎟️", text: "Extra spin vouchers" },
                  ].map(({ icon, text }) => (
                    <li key={text} className="flex items-center gap-3 text-sm text-muted-foreground">
                      <span className="text-base w-6 text-center flex-shrink-0">{icon}</span>
                      {text}
                    </li>
                  ))}
                </ul>
                <button
                  onClick={() => setAuthOpen(true)}
                  className="btn-gold rounded-2xl h-13 px-7 text-[15px] font-black glow-gold inline-flex items-center justify-center gap-2 self-start"
                >
                  <RotateCcw className="w-5 h-5" />
                  Claim Your Free Spin
                </button>
              </div>

              {/* Right — interactive wheel */}
              <div
                className="relative flex items-center justify-center p-10"
                style={{ background: "radial-gradient(ellipse at center, rgba(245,166,35,0.07) 0%, transparent 70%)" }}
              >
                <SpinWheelPreview onLoginClick={() => setAuthOpen(true)} />
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          LOYALTY TIERS
      ══════════════════════════════════════════════════════ */}
      <section className="py-24 relative overflow-hidden">
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <SectionHeader
            eyebrow="Loyalty Tiers"
            title={<>Level Up. <span className="text-gold">Unlock More.</span></>}
            sub="The more you recharge, the higher your tier — and the better your multipliers, spin prizes, and AI Studio benefits."
          />

          <StaggerGrid className="grid grid-cols-1 sm:grid-cols-3 lg:grid-cols-5 gap-3">
            {(Object.entries(TIER_CONFIG) as [string, typeof TIER_CONFIG[keyof typeof TIER_CONFIG]][]).map(([key, tier], i) => {
              const perks = [
                "1 spin/day · 5 AI tools/day",
                "2× prizes · 10 AI tools/day",
                "3× prizes · 20 AI tools · Priority",
                "5× prizes · Unlimited AI · Early access",
                "10× prizes · Everything · White-glove",
              ];
              const isTop = key === "diamond";
              return (
                <motion.div key={key} variants={fadeUp}>
                  <div
                    className={`relative glass rounded-2xl p-5 flex flex-col items-center text-center border transition-all duration-300 hover:scale-[1.03] ${
                      isTop ? "border-gold-gradient glow-gold" : "border-white/[0.07] hover:border-white/[0.16]"
                    }`}
                  >
                    {isTop && (
                      <div className="absolute -top-3 left-1/2 -translate-x-1/2 bg-gold rounded-full px-3 py-0.5 text-[10px] font-black text-black uppercase tracking-wider">
                        Best
                      </div>
                    )}
                    <div className="text-4xl mb-3">{tier.icon}</div>
                    <div className="text-base font-black mb-1" style={{ color: tier.color }}>{tier.label}</div>
                    <div className="text-[11px] font-mono text-muted-foreground mb-3">
                      {key === "bronze" ? "0 pts — Start here" : `${formatPoints(tier.minPoints)} pts`}
                    </div>
                    <p className="text-[11px] text-muted-foreground leading-relaxed">{perks[i]}</p>
                  </div>
                </motion.div>
              );
            })}
          </StaggerGrid>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          FINAL CTA — full-bleed gold moment
      ══════════════════════════════════════════════════════ */}
      <section className="py-28 relative overflow-hidden">
        {/* Multi-layer bg */}
        <div className="absolute inset-0 pointer-events-none"
          style={{ background: "radial-gradient(ellipse 100% 80% at 50% 50%, rgba(245,166,35,0.10) 0%, transparent 70%)" }} />
        <div className="absolute inset-0 pointer-events-none opacity-[0.03]"
          style={{
            backgroundImage: `linear-gradient(oklch(0.94 0.004 240) 1px, transparent 1px), linear-gradient(90deg, oklch(0.94 0.004 240) 1px, transparent 1px)`,
            backgroundSize: "40px 40px",
          }}
        />

        <div className="max-w-4xl mx-auto px-4 sm:px-6 text-center relative z-10">
          <motion.div
            initial={{ opacity: 0, y: 32 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ type: "spring", stiffness: 220, damping: 28 }}
          >
            <div className="text-6xl mb-6 animate-float-slow">🇳🇬</div>
            <h2 className="text-[clamp(2.4rem,7vw,5rem)] font-black tracking-tight leading-tight text-foreground mb-5">
              Join <span className="text-gold">84,000+</span> Nigerians<br className="hidden sm:block" />
              Earning Daily
            </h2>
            <p className="text-lg text-muted-foreground max-w-xl mx-auto mb-10 leading-relaxed">
              No sign-up fees. No downloads. No gimmicks.
              Just recharge MTN ₦1,000+ like you always do — and spin the wheel for instant prizes every single time.
            </p>
            <div className="flex flex-col sm:flex-row gap-4 justify-center items-center">
              <button
                onClick={() => setAuthOpen(true)}
                className="btn-gold rounded-2xl h-14 px-10 text-lg font-black glow-gold inline-flex items-center gap-2"
              >
                <Zap className="w-5 h-5" />
                Get Started — 100 Bonus Points on Signup
              </button>
            </div>
            <p className="text-xs text-muted-foreground/40 mt-5">
              MTN Nigeria only · OTP signup in 30 seconds · No credit card
            </p>
          </motion.div>
        </div>
      </section>

      <Footer />
    </div>
  );
}

/* ─── Spin Wheel Preview component (interactive) ─────────────── */
function SpinWheelPreview({ onLoginClick }: { onLoginClick: () => void }) {
  const [rotation, setRotation] = useState(0);
  const [spinning, setSpinning] = useState(false);
  const [result, setResult] = useState<string | null>(null);

  const spin = () => {
    if (spinning) return;
    setResult(null);
    setSpinning(true);
    const extra = 1080 + Math.floor(Math.random() * 360);
    setRotation((r) => r + extra);
    setTimeout(() => {
      setSpinning(false);
      setResult("Login to claim your prize!");
      setTimeout(onLoginClick, 700);
    }, 3000);
  };

  const prizes = SPIN_PRIZES.slice(0, 8);
  const segments = prizes.length;
  const segAngle = 360 / segments;

  return (
    <div className="flex flex-col items-center gap-6 select-none">
      {/* Wheel */}
      <div className="relative w-60 h-60 sm:w-72 sm:h-72">
        {/* Outer ring glow */}
        <div
          className="absolute -inset-2 rounded-full pointer-events-none"
          style={{
            background: "conic-gradient(from 0deg, #F5A623, #FFE066, #00D4FF, #8B5CF6, #F5A623)",
            opacity: 0.20,
            filter: "blur(10px)",
          }}
        />
        {/* Wheel disk */}
        <div
          className="w-full h-full rounded-full overflow-hidden border-4 border-primary/20 relative"
          style={{
            transform: `rotate(${rotation}deg)`,
            transition: spinning ? "transform 3s cubic-bezier(0.17, 0.67, 0.21, 1)" : "none",
            boxShadow: "0 0 40px rgba(245,166,35,0.25), inset 0 0 30px rgba(0,0,0,0.3)",
          }}
        >
          {prizes.map((p, i) => {
            const startAngle = i * segAngle;
            const midAngle = startAngle + segAngle / 2;
            const rad = (midAngle * Math.PI) / 180;
            const r = 37;
            const tx = 50 + r * Math.cos((rad) - Math.PI / 2);
            const ty = 50 + r * Math.sin((rad) - Math.PI / 2);
            return (
              <div
                key={p.id}
                className="absolute inset-0"
                style={{
                  background: `conic-gradient(from ${startAngle}deg at 50% 50%, ${p.color}dd ${startAngle}deg ${startAngle + segAngle}deg, transparent ${startAngle + segAngle}deg)`,
                }}
              >
                {/* Divider line */}
                <div
                  className="absolute top-0 left-1/2 w-px origin-bottom"
                  style={{ height: "50%", background: "rgba(0,0,0,0.25)", transform: `rotate(${startAngle}deg)`, transformOrigin: "50% 100%" }}
                />
                {/* Label */}
                <span
                  className="absolute text-[9px] sm:text-[10px] font-black text-white leading-tight pointer-events-none"
                  style={{
                    left: `${tx}%`,
                    top: `${ty}%`,
                    transform: `translate(-50%,-50%) rotate(${midAngle}deg)`,
                    textShadow: "0 1px 3px rgba(0,0,0,0.8)",
                    maxWidth: 44,
                    textAlign: "center",
                  }}
                >
                  {p.label}
                </span>
              </div>
            );
          })}
          {/* Center hub */}
          <div className="absolute inset-[28%] rounded-full bg-surface-0 border-4 border-primary flex items-center justify-center z-10">
            <Zap className="w-5 h-5 text-primary" />
          </div>
        </div>

        {/* Pointer */}
        <div className="absolute top-1/2 right-0 translate-x-3 -translate-y-1/2 z-20">
          <div
            className="w-5 h-5 glow-gold"
            style={{
              clipPath: "polygon(100% 50%, 0% 0%, 0% 100%)",
              background: "linear-gradient(135deg, #F5A623, #FFE066)",
            }}
          />
        </div>
      </div>

      {/* Spin button */}
      <button
        onClick={spin}
        disabled={spinning}
        className={`rounded-2xl h-12 px-8 text-[14px] font-black inline-flex items-center gap-2 transition-all duration-200 ${
          spinning
            ? "glass border border-white/10 text-muted-foreground cursor-wait"
            : "btn-gold glow-gold"
        }`}
      >
        <RotateCcw className={`w-4 h-4 ${spinning ? "animate-spin" : ""}`} />
        {spinning ? "Spinning…" : "Try a Demo Spin!"}
      </button>

      <AnimatePresence>
        {result && (
          <motion.div
            initial={{ opacity: 0, scale: 0.85, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0 }}
            transition={{ type: "spring", stiffness: 400, damping: 22 }}
            className="glass-gold rounded-2xl px-6 py-3 text-center border border-primary/30"
          >
            <p className="text-sm font-bold text-primary">{result}</p>
          </motion.div>
        )}
      </AnimatePresence>

      <p className="text-[11px] text-muted-foreground/50 text-center max-w-[220px]">
        Sign in for your daily free spin. Extra spins with Pulse Points.
      </p>
    </div>
  );
}
