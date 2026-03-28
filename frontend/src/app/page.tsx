"use client";
import React, { useState, useEffect, useRef, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { motion, AnimatePresence, useScroll, useTransform, useInView } from "framer-motion";
import {
  Zap, Sparkles, ArrowRight, RotateCcw, Lock, ChevronRight,
  Trophy, MapPin, Users, Gift, Star, Clock, Swords,
  Brain, Camera, Video, Mic, BookOpen, BarChart2,
} from "lucide-react";
import { useStore } from "@/store/useStore";
import NavBar from "@/components/landing/NavBar";
import Footer from "@/components/landing/Footer";
import AuthModal from "@/components/landing/AuthModal";

// ─── Tier config ──────────────────────────────────────────────
const TIER_CONFIG = {
  bronze:   { label: "Bronze",   color: "#CD7F32", icon: "🥉", minPoints: 0,      spinMultiplier: 1,  pointMultiplier: 1,   perks: ["1× spin per recharge", "Basic AI tools", "Pulse Points earnings"] },
  silver:   { label: "Silver",   color: "#C0C0C0", icon: "🥈", minPoints: 5000,   spinMultiplier: 2,  pointMultiplier: 1.2, perks: ["2× spins per recharge", "All AI tools", "1.2× point multiplier"] },
  gold:     { label: "Gold",     color: "#FFD700", icon: "🥇", minPoints: 15000,  spinMultiplier: 3,  pointMultiplier: 1.5, perks: ["3× spins per recharge", "Priority AI queue", "1.5× point multiplier"] },
  platinum: { label: "Platinum", color: "#E5E4E2", icon: "💎", minPoints: 40000,  spinMultiplier: 5,  pointMultiplier: 2,   perks: ["5× spins per recharge", "Exclusive AI tools", "2× point multiplier"] },
  diamond:  { label: "Diamond",  color: "#B9F2FF", icon: "💠", minPoints: 100000, spinMultiplier: 10, pointMultiplier: 3,   perks: ["10× spins per recharge", "All premium tools", "3× point multiplier"] },
} as const;

// ─── Spin prizes ──────────────────────────────────────────────
const SPIN_PRIZES = [
  { id: "1", label: "₦5,000 Cash",    color: "#F5A623" },
  { id: "2", label: "1GB Data",        color: "#00D4FF" },
  { id: "3", label: "₦1,000 Airtime", color: "#10B981" },
  { id: "4", label: "500 Points",      color: "#8B5CF6" },
  { id: "5", label: "₦2,500 Cash",    color: "#F59E0B" },
  { id: "6", label: "2GB Data",        color: "#06B6D4" },
  { id: "7", label: "Free Spin",       color: "#EC4899" },
  { id: "8", label: "200 Points",      color: "#6366F1" },
];

// ─── Activity ticker events ────────────────────────────────────
const ACTIVITY_EVENTS = [
  { emoji: "🎉", text: "Chioma A. won ₦5,000 cash from the wheel" },
  { emoji: "⚡", text: "Tunde O. earned 2,400 Pulse Points on recharge" },
  { emoji: "🎨", text: "Amina K. created an AI photo in seconds" },
  { emoji: "🎯", text: "Emeka N. spun the wheel and won 2GB data" },
  { emoji: "💼", text: "Fatima B. generated a business plan with AI" },
  { emoji: "🔊", text: "Biodun S. created a marketing jingle" },
  { emoji: "📸", text: "Kemi A. removed background from 10 photos" },
  { emoji: "🏆", text: "Seun L. reached Gold tier" },
  { emoji: "⚔️", text: "Lagos leads Regional Wars with 84K points" },
  { emoji: "🗺️", text: "Abuja climbed to #2 in this month's Wars" },
  { emoji: "💰", text: "Rivers State won ₦150K in last month's Wars" },
  { emoji: "🎤", text: "Adaeze M. converted voice note to business plan" },
  { emoji: "🏅", text: "Kano State entered the top 3 — ₦100K prize pool" },
  { emoji: "🌍", text: "Ogun State is closing in on the top 3 this week" },
];

// ─── AI tool categories ───────────────────────────────────────
const TOOL_CATEGORIES = [
  { key: "chat",   label: "Chat & Search", emoji: "💬", color: "#00D4FF", bg: "rgba(0,212,255,0.08)",  border: "rgba(0,212,255,0.20)", tools: ["Ask Nexus", "Web Search AI", "Code Helper"] },
  { key: "create", label: "Create",        emoji: "🎨", color: "#F5A623", bg: "rgba(245,166,35,0.08)", border: "rgba(245,166,35,0.20)", tools: ["AI Photo Creator", "AI Photo Dream", "BG Remover", "Video Cinematic"] },
  { key: "learn",  label: "Learn",         emoji: "📚", color: "#10B981", bg: "rgba(16,185,129,0.08)", border: "rgba(16,185,129,0.20)", tools: ["Study Guide", "Quiz Maker", "Mind Map", "AI Podcast"] },
  { key: "build",  label: "Build",         emoji: "🏗️", color: "#8B5CF6", bg: "rgba(139,92,246,0.08)", border: "rgba(139,92,246,0.20)", tools: ["Slide Deck", "Business Plan", "Voice to Plan"] },
];

const TOOL_CARDS = [
  { slug: "ask-nexus",        name: "Ask Nexus",         emoji: "🤖", description: "Chat with a powerful AI assistant for any question.", icon: Brain },
  { slug: "ai-photo",         name: "AI Photo Creator",  emoji: "🎨", description: "Generate stunning images from text prompts.", icon: Camera },
  { slug: "video-cinematic",  name: "Video Generator",   emoji: "🎬", description: "Turn your ideas into cinematic short videos.", icon: Video },
  { slug: "voice-to-plan",    name: "Voice to Plan",     emoji: "🎤", description: "Speak your idea, get a full business plan.", icon: Mic },
  { slug: "study-guide",      name: "Study Guide",       emoji: "📚", description: "Turn any topic into a comprehensive study guide.", icon: BookOpen },
  { slug: "business-plan",    name: "Business Plan AI",  emoji: "💼", description: "Generate investor-ready business plans in minutes.", icon: BarChart2 },
  { slug: "marketing-jingle", name: "Marketing Jingle",  emoji: "🎵", description: "Create catchy jingles for your brand or product.", icon: Mic },
  { slug: "bg-remover",       name: "BG Remover",        emoji: "✂️", description: "Remove image backgrounds instantly with AI.", icon: Camera },
];

// ─── Animated counter ─────────────────────────────────────────
function Counter({ to, suffix = "", duration = 1800, prefix = "" }: { to: number; suffix?: string; duration?: number; prefix?: string }) {
  const [count, setCount] = useState(0);
  const ref    = useRef<HTMLSpanElement>(null);
  const inView = useInView(ref, { once: true });
  useEffect(() => {
    if (!inView) return;
    let start    = 0;
    const steps     = 60;
    const increment = to / steps;
    const interval  = duration / steps;
    const timer = setInterval(() => {
      start += increment;
      if (start >= to) { setCount(to); clearInterval(timer); }
      else setCount(Math.floor(start));
    }, interval);
    return () => clearInterval(timer);
  }, [inView, to, duration]);
  return <span ref={ref}>{prefix}{count.toLocaleString("en-NG")}{suffix}</span>;
}

// ─── Aurora canvas background ─────────────────────────────────
function AuroraCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    let raf: number;
    let t = 0;
    const resize = () => { canvas.width = canvas.offsetWidth; canvas.height = canvas.offsetHeight; };
    resize();
    window.addEventListener("resize", resize);
    const draw = () => {
      const { width: w, height: h } = canvas;
      ctx.clearRect(0, 0, w, h);
      const blobs = [
        { x: 0.35 + 0.12 * Math.sin(t * 0.0007), y: 0.38 + 0.10 * Math.cos(t * 0.0009), rx: 0.55 * w, ry: 0.45 * h, color: [245, 166, 35],  a: 0.13 },
        { x: 0.70 + 0.10 * Math.cos(t * 0.0006), y: 0.30 + 0.12 * Math.sin(t * 0.0008), rx: 0.45 * w, ry: 0.40 * h, color: [95, 114, 249],  a: 0.10 },
        { x: 0.20 + 0.08 * Math.sin(t * 0.0010), y: 0.65 + 0.08 * Math.cos(t * 0.0007), rx: 0.40 * w, ry: 0.35 * h, color: [0, 212, 255],   a: 0.08 },
      ];
      blobs.forEach(b => {
        const grd = ctx.createRadialGradient(b.x * w, b.y * h, 0, b.x * w, b.y * h, Math.max(b.rx, b.ry));
        grd.addColorStop(0,   `rgba(${b.color.join(",")},${b.a})`);
        grd.addColorStop(0.5, `rgba(${b.color.join(",")},${b.a * 0.4})`);
        grd.addColorStop(1,   `rgba(${b.color.join(",")},0)`);
        ctx.fillStyle = grd;
        ctx.fillRect(0, 0, w, h);
      });
      t++;
      raf = requestAnimationFrame(draw);
    };
    draw();
    return () => { cancelAnimationFrame(raf); window.removeEventListener("resize", resize); };
  }, []);
  return <canvas ref={canvasRef} className="absolute inset-0 w-full h-full pointer-events-none" />;
}

// ─── Live activity ticker ──────────────────────────────────────
function LiveTicker() {
  const doubled = [...ACTIVITY_EVENTS, ...ACTIVITY_EVENTS];
  return (
    <div className="relative overflow-hidden py-3 border-y border-white/[0.06]" style={{ background: "#0e0f16" }}>
      <div className="absolute left-0 top-0 bottom-0 w-20 z-10 pointer-events-none"
        style={{ background: "linear-gradient(to right, #0e0f16, transparent)" }} />
      <div className="absolute right-0 top-0 bottom-0 w-20 z-10 pointer-events-none"
        style={{ background: "linear-gradient(to left, #0e0f16, transparent)" }} />
      <div className="flex animate-ticker whitespace-nowrap">
        {doubled.map((ev, i) => (
          <span key={i} className="inline-flex items-center gap-2 mx-6 text-[13px] text-white/50">
            <span>{ev.emoji}</span>
            <span>{ev.text}</span>
            <span className="text-white/20 mx-2">·</span>
          </span>
        ))}
      </div>
    </div>
  );
}

// ─── Section header ────────────────────────────────────────────
function SectionHeader({ eyebrow, title, sub }: { eyebrow: string; title: React.ReactNode; sub?: string }) {
  return (
    <div className="text-center mb-14">
      <div className="eyebrow mb-3">{eyebrow}</div>
      <h2 className="text-4xl sm:text-5xl font-black tracking-[-0.02em] leading-tight text-white mb-4">{title}</h2>
      {sub && <p className="text-base text-white/45 max-w-xl mx-auto leading-relaxed">{sub}</p>}
    </div>
  );
}

// ─── Stagger wrapper ───────────────────────────────────────────
const fadeUp = { hidden: { opacity: 0, y: 24 }, visible: { opacity: 1, y: 0, transition: { duration: 0.5, ease: [0.22, 1, 0.36, 1] as [number, number, number, number] } } };
function StaggerGrid({ children, className }: { children: React.ReactNode; className?: string }) {
  const ref    = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { once: true, margin: "-80px" });
  return (
    <motion.div ref={ref}
      variants={{ hidden: {}, visible: { transition: { staggerChildren: 0.08 } } }}
      initial="hidden" animate={inView ? "visible" : "hidden"}
      className={className}>
      {children}
    </motion.div>
  );
}

// ─── Demo spin wheel ───────────────────────────────────────────
function DemoSpinWheel({ onLoginClick }: { onLoginClick: () => void }) {
  const [rotation, setRotation] = useState(0);
  const [spinning, setSpinning] = useState(false);
  const [result, setResult]     = useState<string | null>(null);
  const spin = () => {
    if (spinning) return;
    setResult(null);
    setSpinning(true);
    const extra = 1080 + Math.floor(Math.random() * 360);
    setRotation(r => r + extra);
    setTimeout(() => {
      setSpinning(false);
      setResult("Sign in to claim your prize!");
      setTimeout(onLoginClick, 700);
    }, 3000);
  };
  const segments = SPIN_PRIZES.length;
  const segAngle = 360 / segments;
  return (
    <div className="flex flex-col items-center gap-6 select-none">
      <div className="relative w-60 h-60 sm:w-72 sm:h-72">
        <div className="absolute -inset-2 rounded-full pointer-events-none"
          style={{ background: "conic-gradient(from 0deg, #F5A623, #FFE066, #00D4FF, #8B5CF6, #F5A623)", opacity: 0.20, filter: "blur(10px)" }} />
        <div className="w-full h-full rounded-full overflow-hidden border-4 border-white/10 relative"
          style={{
            transform: `rotate(${rotation}deg)`,
            transition: spinning ? "transform 3s cubic-bezier(0.17, 0.67, 0.21, 1)" : "none",
            boxShadow: "0 0 40px rgba(245,166,35,0.25), inset 0 0 30px rgba(0,0,0,0.3)",
          }}>
          {SPIN_PRIZES.map((p, i) => {
            const startAngle = i * segAngle;
            const midAngle   = startAngle + segAngle / 2;
            const rad        = (midAngle * Math.PI) / 180;
            const r          = 37;
            const tx         = 50 + r * Math.cos(rad - Math.PI / 2);
            const ty         = 50 + r * Math.sin(rad - Math.PI / 2);
            return (
              <div key={p.id} className="absolute inset-0"
                style={{ background: `conic-gradient(from ${startAngle}deg at 50% 50%, ${p.color}dd ${startAngle}deg ${startAngle + segAngle}deg, transparent ${startAngle + segAngle}deg)` }}>
                <div className="absolute top-0 left-1/2 w-px origin-bottom"
                  style={{ height: "50%", background: "rgba(0,0,0,0.25)", transform: `rotate(${startAngle}deg)`, transformOrigin: "50% 100%" }} />
                <span className="absolute text-[9px] sm:text-[10px] font-black text-white leading-tight pointer-events-none"
                  style={{ left: `${tx}%`, top: `${ty}%`, transform: `translate(-50%,-50%) rotate(${midAngle}deg)`, textShadow: "0 1px 3px rgba(0,0,0,0.8)", maxWidth: 44, textAlign: "center" }}>
                  {p.label}
                </span>
              </div>
            );
          })}
          <div className="absolute inset-[28%] rounded-full flex items-center justify-center z-10"
            style={{ background: "#0d0e14", border: "4px solid rgba(245,166,35,0.4)" }}>
            <Zap className="w-5 h-5 text-gold-500" />
          </div>
        </div>
        <div className="absolute top-1/2 right-0 translate-x-3 -translate-y-1/2 z-20">
          <div className="w-5 h-5" style={{ clipPath: "polygon(100% 50%, 0% 0%, 0% 100%)", background: "linear-gradient(135deg, #F5A623, #FFE066)", filter: "drop-shadow(0 0 6px rgba(245,166,35,0.6))" }} />
        </div>
      </div>
      <button onClick={spin} disabled={spinning}
        className={`rounded-2xl h-12 px-8 text-[14px] font-black inline-flex items-center gap-2 transition-all duration-200 ${spinning ? "glass border border-white/10 text-white/40 cursor-wait" : "btn-gold"}`}>
        <RotateCcw className={`w-4 h-4 ${spinning ? "animate-spin" : ""}`} />
        {spinning ? "Spinning…" : "Try a Demo Spin!"}
      </button>
      <AnimatePresence>
        {result && (
          <motion.div initial={{ opacity: 0, scale: 0.85, y: 10 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0 }}
            transition={{ type: "spring", stiffness: 400, damping: 22 }}
            className="rounded-2xl px-6 py-3 text-center border"
            style={{ background: "rgba(245,166,35,0.08)", borderColor: "rgba(245,166,35,0.3)" }}>
            <p className="text-sm font-bold text-gold-500">{result}</p>
          </motion.div>
        )}
      </AnimatePresence>
      <p className="text-[11px] text-white/30 text-center max-w-[220px]">
        Sign in for your daily free spin. Extra spins with Pulse Points.
      </p>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────
export default function HomePage() {
  const [authOpen, setAuthOpen]           = useState(false);
  const [bannerVisible, setBannerVisible] = useState(true);
  const { isAuthenticated, _hasHydrated } = useStore();
  const router                            = useRouter();
  const heroRef                           = useRef<HTMLElement>(null);
  const { scrollY }                       = useScroll();
  const heroY                             = useTransform(scrollY, [0, 600], [0, -80]);
  const heroOpacity                       = useTransform(scrollY, [0, 400], [1, 0.3]);

  // Redirect authenticated users to dashboard
  useEffect(() => {
    if (_hasHydrated && isAuthenticated) router.replace("/dashboard");
  }, [_hasHydrated, isAuthenticated, router]);

  const openAuth = useCallback(() => setAuthOpen(true), []);

  if (_hasHydrated && isAuthenticated) return null;

  return (
    <div className="min-h-screen overflow-x-hidden" style={{ background: "#0d0e14" }}>
      <NavBar onLoginClick={openAuth} />
      <AuthModal open={authOpen} onClose={() => setAuthOpen(false)} />

      {/* ── Announcement banner ── */}
      <AnimatePresence>
        {bannerVisible && (
          <motion.div
            initial={{ opacity: 0, y: -40 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -40 }}
            transition={{ type: "spring", stiffness: 300, damping: 28, delay: 0.8 }}
            className="fixed top-16 left-0 right-0 z-40 flex justify-center px-3 sm:px-4 pointer-events-none"
          >
            <div className="pointer-events-auto w-full max-w-3xl flex items-center justify-between gap-3 rounded-2xl px-4 py-2.5 border border-gold-500/20 shadow-lg"
              style={{ background: "rgba(17,18,25,0.92)", backdropFilter: "blur(20px)" }}>
              <div className="flex items-center gap-2.5 min-w-0">
                <div className="w-7 h-7 rounded-lg bg-gold-500 flex items-center justify-center flex-shrink-0"
                  style={{ boxShadow: "0 0 12px rgba(245,166,35,0.5)" }}>
                  <Zap className="w-3.5 h-3.5 text-black" />
                </div>
                <p className="text-[12px] sm:text-[13px] text-white/50 leading-snug min-w-0">
                  <button onClick={openAuth} className="font-black text-gold-500 hover:underline underline-offset-2 mr-1">Sign in</button>
                  to see your Pulse Points, spin the wheel, and unlock{" "}
                  <span className="font-bold text-white">30+ premium AI tools</span> — all from your MTN recharges.
                </p>
              </div>
              <div className="flex items-center gap-2 flex-shrink-0">
                <button onClick={openAuth}
                  className="hidden sm:inline-flex items-center gap-1.5 rounded-xl h-8 px-4 text-[11px] font-black whitespace-nowrap btn-gold">
                  <Sparkles className="w-3 h-3" />
                  Get Started
                </button>
                <button onClick={() => setBannerVisible(false)} className="text-white/25 hover:text-white/60 transition-colors text-lg leading-none">×</button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* ══════════════════════════════════════════════════════
          HERO
      ══════════════════════════════════════════════════════ */}
      <section ref={heroRef} className="relative min-h-[100dvh] flex items-center justify-center overflow-hidden">
        <AuroraCanvas />
        <div className="absolute inset-0 pointer-events-none opacity-[0.025]"
          style={{ backgroundImage: "linear-gradient(rgba(240,242,255,1) 1px, transparent 1px), linear-gradient(90deg, rgba(240,242,255,1) 1px, transparent 1px)", backgroundSize: "40px 40px" }} />
        <motion.div style={{ y: heroY, opacity: heroOpacity }}
          className="relative z-10 w-full max-w-6xl mx-auto px-4 sm:px-6 pt-36 pb-16 flex flex-col items-center">
          {/* Live badge */}
          <motion.div initial={{ opacity: 0, y: 16, scale: 0.94 }} animate={{ opacity: 1, y: 0, scale: 1 }}
            transition={{ type: "spring", stiffness: 280, damping: 24, delay: 0.05 }}
            className="inline-flex items-center gap-2 rounded-full px-5 py-2 mb-8 cursor-default select-none border"
            style={{ background: "rgba(245,166,35,0.08)", borderColor: "rgba(245,166,35,0.25)" }}>
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-nexus-500 opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-nexus-500" />
            </span>
            <span className="text-[13px] font-bold text-nexus-400">84,231 Nigerians earning right now</span>
            <span className="hidden sm:inline text-white/20 mx-1">·</span>
            <span className="hidden sm:inline text-[12px] font-semibold text-white/40">🇳🇬 Made for Africa</span>
          </motion.div>

          {/* Headline */}
          <div className="text-center mb-6 px-2">
            <motion.h1 initial={{ opacity: 0, y: 30 }} animate={{ opacity: 1, y: 0 }}
              transition={{ type: "spring", stiffness: 220, damping: 28, delay: 0.10 }}
              className="text-[clamp(3rem,10vw,7rem)] font-black tracking-[-0.03em] leading-[1.0] mb-3">
              <span className="text-white block">Recharge.</span>
              <span className="shimmer-text block">Earn. Create.</span>
            </motion.h1>
            <motion.p initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }}
              transition={{ type: "spring", stiffness: 220, damping: 28, delay: 0.18 }}
              className="text-[clamp(1rem,2.5vw,1.25rem)] text-white/45 max-w-xl mx-auto leading-relaxed">
              Recharge MTN <span className="font-black text-white">₦1,000+</span> and get a{" "}
              <span className="text-nexus-400 font-bold">free wheel spin</span> — win cash up to ₦5,000, data &amp; airtime instantly.
              Plus earn <span className="text-nexus-400 font-bold">Pulse Points</span> for 30+ AI tools.
            </motion.p>
          </div>

          {/* Feature pills */}
          <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.28 }}
            className="flex flex-wrap items-center justify-center gap-2 mb-10">
            {[
              { icon: "↺", text: "Free spin on ₦1,000+ recharge" },
              { icon: "🏆", text: "Win up to ₦5,000 instantly" },
              { icon: "⚡", text: "Pulse Points on every recharge" },
              { icon: "✦", text: "30+ AI tools unlocked" },
            ].map(({ icon, text }) => (
              <div key={text} className="glass border border-white/[0.10] rounded-full px-4 py-1.5 text-[13px] font-semibold text-white/60 flex items-center gap-1.5">
                <span>{icon}</span>{text}
              </div>
            ))}
          </motion.div>

          {/* CTA buttons */}
          <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.35 }}
            className="flex flex-col sm:flex-row items-center gap-3 mb-8">
            <button onClick={openAuth}
              className="btn-gold rounded-2xl h-14 px-8 text-[15px] font-black inline-flex items-center gap-2 w-full sm:w-auto justify-center"
              style={{ boxShadow: "0 0 24px rgba(245,166,35,0.4)" }}>
              <Zap className="w-5 h-5" />
              Start Earning — It's Free
              <ArrowRight className="w-4 h-4" />
            </button>
            <button onClick={openAuth}
              className="glass border border-white/[0.12] rounded-2xl h-14 px-8 text-[15px] font-semibold text-white hover:border-white/25 transition-all inline-flex items-center gap-2 w-full sm:w-auto justify-center">
              <Sparkles className="w-4 h-4 text-gold-500" />
              Try AI Studio Free
            </button>
          </motion.div>

          {/* Social proof — avatar stack + star rating */}
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.44 }}
            className="flex items-center gap-3 mb-0">
            <div className="flex -space-x-2">
              {["#F5A623","#FFE066","#00D4FF","#8B5CF6","#10B981","#F472B6"].map((bg, i) => (
                <div key={i} className="w-8 h-8 rounded-full border-2 flex items-center justify-center text-[10px] font-black text-black"
                  style={{ borderColor: "#0d0e14", background: bg }}>
                  {"CTAEFK"[i]}
                </div>
              ))}
            </div>
            <div className="flex items-center gap-1">
              {[...Array(5)].map((_,i) => <Star key={i} className="w-3.5 h-3.5 fill-gold-500 text-gold-500" />)}
            </div>
            <span className="text-[13px] text-white/50"><strong className="text-white">4.9/5</strong> · 12,000+ users</span>
          </motion.div>
        </motion.div>
      </section>

      {/* Live ticker */}
      <LiveTicker />

      {/* ══════════════════════════════════════════════════════
          HOW IT WORKS
      ══════════════════════════════════════════════════════ */}
      <section className="py-24 relative">
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <SectionHeader
            eyebrow="How It Works"
            title={<>Four steps to <span className="text-gold-500">everything</span></>}
            sub="From your first recharge to winning cash and creating with AI — here's the full journey."
          />
          <StaggerGrid className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
            {[
              { n: "01", icon: "📱", color: "#00D4FF", title: "Recharge MTN",       body: "Recharge ₦1,000 or more on any MTN line. Your recharge is automatically detected — no codes, no hassle.", stat: "₦200 = 1 Pulse Point" },
              { n: "02", icon: "⚡", color: "#F5A623", title: "Earn Pulse Points",   body: "Every naira you recharge earns Pulse Points. The more you recharge, the more you earn. Higher tiers multiply your earnings.", stat: "Up to 3× multiplier" },
              { n: "03", icon: "🎰", color: "#10B981", title: "Spin & Win",          body: "Each qualifying recharge earns a free wheel spin. Win instant cash, data bundles, airtime or bonus Pulse Points.", stat: "₦18M+ prizes distributed" },
              { n: "04", icon: "🚀", color: "#8B5CF6", title: "Unlock AI Studio",    body: "Spend points to access 30+ AI tools — create photos, generate videos, build business plans, make music, and more.", stat: "1.2M+ generations created" },
            ].map((step) => (
              <motion.div key={step.n} variants={fadeUp}>
                <div className="glass rounded-2xl border border-white/[0.08] p-6 h-full flex flex-col hover:border-white/[0.15] transition-all duration-300">
                  <div className="flex items-start justify-between mb-5">
                    <div className="w-12 h-12 rounded-2xl flex items-center justify-center text-2xl flex-shrink-0"
                      style={{ background: `${step.color}15`, border: `1px solid ${step.color}30` }}>
                      {step.icon}
                    </div>
                    <span className="text-[11px] font-black text-white/15 font-mono">{step.n}</span>
                  </div>
                  <h3 className="text-lg font-black text-white mb-2">{step.title}</h3>
                  <p className="text-[13px] text-white/45 leading-relaxed flex-1">{step.body}</p>
                  <div className="mt-4 pt-4 border-t border-white/[0.07]">
                    <span className="text-[12px] font-bold" style={{ color: step.color }}>{step.stat}</span>
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
          style={{ background: "radial-gradient(ellipse 80% 50% at 80% 50%, rgba(95,114,249,0.06) 0%, transparent 65%)" }} />
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <SectionHeader
            eyebrow="AI Studio"
            title={<>30+ AI tools, <span className="text-nexus-400">all yours</span></>}
            sub="Earn Pulse Points from your recharges and spend them on world-class AI tools. Tool pricing is set by the admin — always transparent."
          />
          {/* Category cards */}
          <StaggerGrid className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-8">
            {TOOL_CATEGORIES.map((cat) => (
              <motion.div key={cat.key} variants={fadeUp}>
                <Link href="/studio">
                  <div className="glass rounded-2xl p-4 hover:border-white/[0.18] transition-all duration-250 cursor-pointer group h-full border"
                    style={{ borderColor: cat.border }}>
                    <div className="flex items-center justify-between mb-3">
                      <div className="w-9 h-9 rounded-xl flex items-center justify-center text-xl"
                        style={{ background: cat.bg, border: `1px solid ${cat.border}` }}>
                        {cat.emoji}
                      </div>
                      <ChevronRight className="w-4 h-4 text-white/30 group-hover:text-white transition-colors" />
                    </div>
                    <h3 className="font-black text-sm mb-2" style={{ color: cat.color }}>{cat.label}</h3>
                    <ul className="space-y-1">
                      {cat.tools.map(t => (
                        <li key={t} className="text-[11px] text-white/40 flex items-center gap-1.5">
                          <span className="w-1 h-1 rounded-full flex-shrink-0" style={{ background: cat.color }} />
                          {t}
                        </li>
                      ))}
                    </ul>
                  </div>
                </Link>
              </motion.div>
            ))}
          </StaggerGrid>
          {/* Tool cards */}
          <StaggerGrid className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
            {TOOL_CARDS.map((tool) => (
              <motion.div key={tool.slug} variants={fadeUp}>
                <div className="glass rounded-2xl border border-white/[0.07] p-4 hover:border-white/[0.18] transition-all duration-250 cursor-pointer group hover:scale-[1.02]"
                  onClick={openAuth}>
                  <div className="flex items-start justify-between mb-3">
                    <span className="text-2xl">{tool.emoji}</span>
                    <div className="flex items-center gap-1 text-[10px] text-white/30">
                      <Lock className="w-3 h-3" />
                      <span>pts</span>
                    </div>
                  </div>
                  <h4 className="text-[13px] font-bold text-white leading-snug mb-1">{tool.name}</h4>
                  <p className="text-[11px] text-white/40 line-clamp-2 leading-relaxed">{tool.description}</p>
                  <div className="mt-3 flex items-center gap-1 text-[11px] text-white/30">
                    <Lock className="w-3 h-3" />
                    <span>Sign in to unlock</span>
                  </div>
                </div>
              </motion.div>
            ))}
          </StaggerGrid>
          <div className="text-center mt-10">
            <Link href="/studio">
              <button className="inline-flex items-center gap-2 glass border border-white/[0.12] rounded-2xl h-12 px-7 text-sm font-semibold text-white hover:border-white/25 transition-all duration-200">
                Explore all 30+ tools
                <ArrowRight className="w-4 h-4" />
              </button>
            </Link>
          </div>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          SPIN & WIN
      ══════════════════════════════════════════════════════ */}
      <section className="py-24 relative overflow-hidden">
        <div className="absolute inset-0 pointer-events-none"
          style={{ background: "radial-gradient(ellipse 70% 60% at 20% 50%, rgba(245,166,35,0.06) 0%, transparent 65%)" }} />
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <div className="glass rounded-3xl border border-white/[0.09] overflow-hidden">
            <div className="grid grid-cols-1 lg:grid-cols-2">
              {/* Left — text */}
              <div className="p-8 sm:p-12 flex flex-col justify-center">
                <div className="eyebrow mb-4">Daily Spin &amp; Win</div>
                <h2 className="text-4xl sm:text-5xl font-black text-white leading-tight mb-4">
                  Recharge <span className="text-gold-500">₦1,000+</span>.<br />
                  Get a Free Spin.
                </h2>
                <p className="text-base text-white/45 leading-relaxed mb-6">
                  Every MTN recharge of ₦1,000 or above automatically earns you a free wheel spin — no hoops, no codes.
                  Win instant cash, data bundles and airtime directly to your line.
                  Stack more spins with your Pulse Points.
                </p>
                <ul className="space-y-2.5 mb-8">
                  {[
                    { icon: "💰", text: "Cash prizes up to ₦5,000 per spin" },
                    { icon: "📶", text: "Data bundles — 1GB, 2GB, 5GB" },
                    { icon: "📱", text: "Airtime credited instantly" },
                    { icon: "⚡", text: "Bonus Pulse Points on every spin" },
                  ].map(({ icon, text }) => (
                    <li key={text} className="flex items-center gap-3 text-[14px] text-white/60">
                      <span className="text-base">{icon}</span>
                      <span>{text}</span>
                    </li>
                  ))}
                </ul>
                <button onClick={openAuth}
                  className="btn-gold rounded-2xl h-12 px-7 text-[14px] font-black inline-flex items-center gap-2 w-fit">
                  <Zap className="w-4 h-4" />
                  Claim Your Free Spin
                </button>
              </div>
              {/* Right — demo wheel */}
              <div className="p-8 sm:p-12 flex items-center justify-center border-t lg:border-t-0 lg:border-l border-white/[0.07]"
                style={{ background: "rgba(245,166,35,0.02)" }}>
                <DemoSpinWheel onLoginClick={openAuth} />
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          REGIONAL WARS  (replaces Referral)
      ══════════════════════════════════════════════════════ */}
      <section className="py-24 relative overflow-hidden">
        <div className="absolute inset-0 pointer-events-none"
          style={{ background: "radial-gradient(ellipse 80% 60% at 50% 50%, rgba(245,166,35,0.04) 0%, rgba(0,212,255,0.03) 60%, transparent 100%)" }} />
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <SectionHeader
            eyebrow="⚔️ Regional Wars"
            title={<>Your state vs <span className="text-gold-500">the nation</span></>}
            sub="Every month, all 37 Nigerian states battle for a ₦500,000 prize pool. The more your state's members recharge, the higher you climb."
          />

          {/* Prize pool breakdown */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-12">
            {[
              { rank: "🥇 1st Place", prize: "₦250,000", share: "50%", color: "#F5A623", glow: "rgba(245,166,35,0.20)" },
              { rank: "🥈 2nd Place", prize: "₦150,000", share: "30%", color: "#C0C0C0", glow: "rgba(192,192,192,0.15)" },
              { rank: "🥉 3rd Place", prize: "₦100,000", share: "20%", color: "#CD7F32", glow: "rgba(205,127,50,0.15)" },
            ].map((p) => (
              <div key={p.rank} className="glass rounded-2xl border border-white/[0.08] p-6 text-center relative overflow-hidden"
                style={{ boxShadow: `0 0 30px ${p.glow}` }}>
                <div className="text-3xl mb-3">{p.rank.split(" ")[0]}</div>
                <div className="text-[11px] font-black uppercase tracking-[0.15em] mb-2" style={{ color: p.color }}>
                  {p.rank.split(" ").slice(1).join(" ")}
                </div>
                <div className="text-3xl font-black text-white mb-1">{p.prize}</div>
                <div className="text-[12px] text-white/40">{p.share} of prize pool</div>
              </div>
            ))}
          </div>

          {/* Individual draw highlight */}
          <div className="glass rounded-3xl p-8 mb-12 relative overflow-hidden border"
            style={{ background: "rgba(245,166,35,0.03)", borderColor: "rgba(245,166,35,0.15)" }}>
            <div className="absolute top-0 left-0 right-0 h-[1px]"
              style={{ background: "linear-gradient(to right, transparent, rgba(245,166,35,0.5), transparent)" }} />
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 items-center">
              <div>
                <div className="eyebrow mb-3">🎲 Individual Cash Draw</div>
                <h3 className="text-2xl sm:text-3xl font-black text-white mb-4">
                  Even if your state wins,<br />
                  <span className="text-gold-500">you could win personally</span>
                </h3>
                <p className="text-[15px] text-white/50 leading-relaxed mb-5">
                  Within each top-3 winning state, one lucky participant is randomly selected for a{" "}
                  <strong className="text-white">direct MoMo cash payout</strong>. You don&apos;t need to be the top earner —
                  just be active in your state&apos;s recharges during the month.
                </p>
                <div className="flex flex-wrap gap-3">
                  {[
                    { icon: "🎯", text: "Random draw from active members" },
                    { icon: "💸", text: "Paid directly via MoMo" },
                    { icon: "📅", text: "Drawn at month end" },
                  ].map(({ icon, text }) => (
                    <div key={text} className="flex items-center gap-2 glass rounded-xl px-3 py-2 text-[12px] text-white/60 border border-white/[0.08]">
                      <span>{icon}</span>{text}
                    </div>
                  ))}
                </div>
              </div>
              {/* Live leaderboard preview */}
              <div className="grid grid-cols-3 gap-3">
                {[
                  { state: "Lagos",  pts: "84,231", rank: 1, color: "#F5A623" },
                  { state: "Abuja",  pts: "71,840", rank: 2, color: "#C0C0C0" },
                  { state: "Rivers", pts: "63,120", rank: 3, color: "#CD7F32" },
                ].map((s) => (
                  <div key={s.state} className="glass rounded-2xl border border-white/[0.08] p-4 text-center">
                    <div className="text-2xl mb-2">{s.rank === 1 ? "🥇" : s.rank === 2 ? "🥈" : "🥉"}</div>
                    <div className="text-[13px] font-black text-white mb-1">{s.state}</div>
                    <div className="text-[11px] font-mono font-bold mb-2" style={{ color: s.color }}>{s.pts}</div>
                    <div className="text-[10px] text-white/30">points</div>
                    <div className="mt-2 text-[10px] font-bold text-gold-500">Draw eligible ✓</div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* How Wars work */}
          <StaggerGrid className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {[
              { icon: <MapPin className="w-5 h-5" />,  color: "#00D4FF", title: "Your State Competes",  body: "You're automatically enrolled in your state's team based on your MTN number registration." },
              { icon: <Users className="w-5 h-5" />,   color: "#F5A623", title: "Recharge Together",    body: "Every recharge from every member in your state adds to the collective points total." },
              { icon: <Trophy className="w-5 h-5" />,  color: "#10B981", title: "Top 3 States Win",     body: "At month end, the top 3 states by total Pulse Points share the ₦500K prize pool." },
              { icon: <Gift className="w-5 h-5" />,    color: "#8B5CF6", title: "Individual Draw",      body: "One random member from each winning state receives a personal MoMo cash payout." },
            ].map((item, i) => (
              <motion.div key={i} variants={fadeUp}>
                <div className="glass rounded-2xl border border-white/[0.08] p-5 h-full">
                  <div className="w-10 h-10 rounded-xl flex items-center justify-center mb-4"
                    style={{ background: `${item.color}15`, border: `1px solid ${item.color}25`, color: item.color }}>
                    {item.icon}
                  </div>
                  <h4 className="text-[14px] font-black text-white mb-2">{item.title}</h4>
                  <p className="text-[12px] text-white/45 leading-relaxed">{item.body}</p>
                </div>
              </motion.div>
            ))}
          </StaggerGrid>

          <div className="text-center mt-10">
            <button onClick={openAuth}
              className="btn-gold rounded-2xl h-12 px-8 text-[14px] font-black inline-flex items-center gap-2">
              <Swords className="w-4 h-4" />
              Join Your State&apos;s Battle
              <ArrowRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          COMING SOON — Daily & Weekly Draws
      ══════════════════════════════════════════════════════ */}
      <section className="py-16 relative">
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <div className="text-center mb-10">
            <div className="eyebrow-cyan mb-3">Coming Soon</div>
            <h2 className="text-3xl sm:text-4xl font-black text-white mb-3">More ways to win</h2>
            <p className="text-white/40 max-w-md mx-auto text-sm">
              We&apos;re building even more prize opportunities. Stay active to be first in line.
            </p>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-5 max-w-2xl mx-auto">
            {[
              {
                icon: <Clock className="w-6 h-6" />,
                color: "#00D4FF",
                title: "Daily Draw",
                body: "A daily prize draw for all active users. Recharge any amount to earn an entry. Winners announced every midnight.",
                badge: "🎲 Daily",
              },
              {
                icon: <Star className="w-6 h-6" />,
                color: "#8B5CF6",
                title: "Weekly Jackpot",
                body: "A bigger weekly prize pool for the most active rechargees of the week. Top earners get bonus entries.",
                badge: "🏆 Weekly",
              },
            ].map((item) => (
              <div key={item.title} className="glass rounded-2xl border border-white/[0.07] p-6 relative overflow-hidden opacity-80">
                <div className="absolute top-3 right-3">
                  <span className="text-[10px] font-black px-2.5 py-1 rounded-full text-white/50 border border-white/[0.10]"
                    style={{ background: `${item.color}10` }}>
                    Coming Soon
                  </span>
                </div>
                <div className="w-12 h-12 rounded-2xl flex items-center justify-center mb-4"
                  style={{ background: `${item.color}12`, border: `1px solid ${item.color}20`, color: item.color }}>
                  {item.icon}
                </div>
                <div className="text-[11px] font-bold mb-2" style={{ color: item.color }}>{item.badge}</div>
                <h3 className="text-lg font-black text-white mb-2">{item.title}</h3>
                <p className="text-[13px] text-white/40 leading-relaxed">{item.body}</p>
                <div className="mt-4 flex items-center gap-2 text-[12px] text-white/25">
                  <Clock className="w-3.5 h-3.5" />
                  <span>Launching soon — stay active to qualify</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          LOYALTY TIERS
      ══════════════════════════════════════════════════════ */}
      <section className="py-24 relative overflow-hidden">
        <div className="absolute inset-0 pointer-events-none"
          style={{ background: "radial-gradient(ellipse 60% 50% at 50% 100%, rgba(245,166,35,0.06) 0%, transparent 70%)" }} />
        <div className="max-w-6xl mx-auto px-4 sm:px-6">
          <SectionHeader
            eyebrow="Loyalty Tiers"
            title={<>The more you recharge,<br /><span className="text-gold-500">the more you earn</span></>}
            sub="Five tiers, each unlocking higher spin multipliers, point bonuses, and exclusive AI tools."
          />
          <StaggerGrid className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
            {(Object.entries(TIER_CONFIG) as [string, typeof TIER_CONFIG[keyof typeof TIER_CONFIG]][]).map(([key, tier], i) => (
              <motion.div key={key} variants={fadeUp}>
                <div className="glass rounded-2xl border p-5 text-center h-full flex flex-col hover:scale-[1.03] transition-transform duration-200"
                  style={{ borderColor: `${tier.color}25` }}>
                  <div className="text-3xl mb-3">{tier.icon}</div>
                  <div className="text-[13px] font-black mb-1" style={{ color: tier.color }}>{tier.label}</div>
                  <div className="text-[11px] text-white/30 mb-4 font-mono">
                    {tier.minPoints === 0 ? "Start here" : `${tier.minPoints.toLocaleString()}+ pts`}
                  </div>
                  <ul className="space-y-1.5 text-left flex-1">
                    {tier.perks.map(perk => (
                      <li key={perk} className="text-[11px] text-white/50 flex items-start gap-1.5">
                        <span className="mt-0.5 flex-shrink-0" style={{ color: tier.color }}>✓</span>
                        {perk}
                      </li>
                    ))}
                  </ul>
                  {i === 0 && (
                    <button onClick={openAuth}
                      className="mt-4 text-[11px] font-black px-3 py-1.5 rounded-lg transition-all"
                      style={{ background: `${tier.color}18`, color: tier.color, border: `1px solid ${tier.color}30` }}>
                      Start Here →
                    </button>
                  )}
                </div>
              </motion.div>
            ))}
          </StaggerGrid>
        </div>
      </section>

      {/* ══════════════════════════════════════════════════════
          FINAL CTA
      ══════════════════════════════════════════════════════ */}
      <section className="py-24 relative overflow-hidden">
        <div className="absolute inset-0"
          style={{ background: "linear-gradient(135deg, rgba(245,166,35,0.07) 0%, rgba(95,114,249,0.05) 50%, rgba(0,212,255,0.04) 100%)" }} />
        <div className="absolute inset-0 pointer-events-none"
          style={{ backgroundImage: "linear-gradient(rgba(255,255,255,0.015) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.015) 1px, transparent 1px)", backgroundSize: "40px 40px" }} />
        <div className="max-w-3xl mx-auto px-4 sm:px-6 text-center relative z-10">
          <motion.div initial={{ opacity: 0, y: 30 }} whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }} transition={{ duration: 0.6 }}>
            <div className="text-6xl mb-6" style={{ animation: "breathe 3s ease-in-out infinite" }}>⚡</div>
            <h2 className="text-4xl sm:text-6xl font-black text-white tracking-[-0.02em] mb-5">
              Your MTN recharge<br />
              <span className="shimmer-text">just got smarter</span>
            </h2>
            <p className="text-lg text-white/45 mb-10 leading-relaxed max-w-xl mx-auto">
              Join 84,000+ Nigerians already earning cash, data, and AI credits from their everyday recharges.
              No subscription. No hidden fees. Just rewards.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
              <button onClick={openAuth}
                className="btn-gold rounded-2xl h-14 px-10 text-[16px] font-black inline-flex items-center gap-2 w-full sm:w-auto justify-center"
                style={{ boxShadow: "0 0 32px rgba(245,166,35,0.4)" }}>
                <Zap className="w-5 h-5" />
                Start Earning Now — It&apos;s Free
              </button>
              <Link href="/wars">
                <button className="glass border border-white/[0.12] rounded-2xl h-14 px-8 text-[15px] font-semibold text-white hover:border-gold-500/30 transition-all inline-flex items-center gap-2 w-full sm:w-auto justify-center">
                  <Swords className="w-4 h-4 text-gold-500" />
                  View Regional Wars
                </button>
              </Link>
            </div>
            <p className="mt-6 text-[12px] text-white/25">
              MTN Nigeria subscribers only · OTP login · No password required
            </p>
          </motion.div>
        </div>
      </section>

      <Footer />
    </div>
  );
}
