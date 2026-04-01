"use client";
import { motion, AnimatePresence } from "framer-motion";
import useSWR from "swr";
import Link from "next/link";
import { useState, useEffect, useCallback } from "react";
import AppShell from "@/components/layout/AppShell";
import { useStore } from "@/store/useStore";
import api from "@/lib/api";
import { cn, formatPoints, TIER_THRESHOLDS } from "@/lib/utils";
import toast, { Toaster } from "react-hot-toast";
import {
  Zap, Wand2, Trophy, ChevronRight, Flame,
  MapPin, Gift, ArrowRight, RotateCcw, Loader2,
  CreditCard, X, Smartphone, Wallet, Sparkles,
  Users, Award, Star, Clock,
} from "lucide-react";

// ─── Design tokens ────────────────────────────────────────────────────────────
const TIER_HEX: Record<string, string> = {
  BRONZE: "#CD7F32", SILVER: "#C0C0C0", GOLD: "#F5A623", PLATINUM: "#E5E4E2",
};
const TIER_EMOJI: Record<string, string> = {
  BRONZE: "🥉", SILVER: "🥈", GOLD: "🥇", PLATINUM: "💎",
};

// ─── Wheel types ──────────────────────────────────────────────────────────────
interface Segment {
  label: string; prize_type: string; base_value: number;
  probability: number; color: string; is_active: boolean;
}
interface SpinOutcome {
  spin_result?: { id: string; prize_type: string; prize_value: number; slot_index: number; fulfillment_status: string };
  prize_label: string; slot_index: number;
}
interface SpinHistoryItem {
  id: string; prize_type: string; prize_value: number;
  fulfillment_status: string; created_at: string;
}
interface StudioTool {
  id: string; name: string; slug: string; icon: string;
  point_cost: number; is_free: boolean;
}

const FALLBACK_SEGMENTS: Segment[] = [
  { label: "₦5,000",   prize_type: "momo_cash",    base_value: 500000, probability: 2,  color: "#F5A623", is_active: true },
  { label: "₦500",     prize_type: "momo_cash",    base_value: 50000,  probability: 8,  color: "#10b981", is_active: true },
  { label: "5GB Data", prize_type: "data_bundle",  base_value: 500,    probability: 10, color: "#06b6d4", is_active: true },
  { label: "₦1,000",   prize_type: "momo_cash",    base_value: 100000, probability: 5,  color: "#8B5CF6", is_active: true },
  { label: "2× Spin",  prize_type: "spin_credit",  base_value: 2,      probability: 10, color: "#F59E0B", is_active: true },
  { label: "500 pts",  prize_type: "pulse_points", base_value: 500,    probability: 20, color: "#5f72f9", is_active: true },
  { label: "₦100 Air", prize_type: "airtime",      base_value: 10000,  probability: 20, color: "#EC4899", is_active: true },
  { label: "1,000 pts",prize_type: "pulse_points", base_value: 1000,   probability: 25, color: "#14b8a6", is_active: true },
];

function fireConfetti() {
  if (typeof window === "undefined") return;
  import("canvas-confetti").then(({ default: confetti }) => {
    confetti({ particleCount: 160, spread: 360, origin: { x: 0.5, y: 0.5 },
      colors: ["#5f72f9","#F5A623","#10b981","#f43f5e","#06b6d4"], startVelocity: 28, ticks: 60 });
  }).catch(() => {});
}

function prizeLabel(item: SpinHistoryItem): string {
  if (item.prize_type === "try_again")    return "Try Again";
  if (item.prize_type === "pulse_points") return `${item.prize_value.toLocaleString()} pts`;
  if (item.prize_type === "airtime")      return `₦${(item.prize_value / 100).toLocaleString()} Airtime`;
  if (item.prize_type === "data_bundle")  return "Data Bundle";
  if (item.prize_type === "momo_cash")    return `₦${(item.prize_value / 100).toLocaleString()} Cash`;
  if (item.prize_type === "spin_credit")  return `${item.prize_value}× Spin`;
  return item.prize_type;
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const m = Math.floor(diff / 60000);
  if (m < 60)  return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24)  return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

// ─── Digital Passport Banner ──────────────────────────────────────────────────
interface BannerConfig {
  banner_enabled: boolean; banner_title: string; banner_subtitle: string;
  banner_cta_ios: string; banner_cta_android: string;
}
const DEFAULT_BANNER: BannerConfig = {
  banner_enabled: true,
  banner_title: "Your Digital Passport is ready",
  banner_subtitle: "Track your Pulse Points and streak right from your lock screen — no app needed.",
  banner_cta_ios: "Add to Apple Wallet",
  banner_cta_android: "Add to Google Wallet",
};

function PassportBanner({ points, streak }: { points: number; streak: number }) {
  const [dismissed, setDismissed] = useState(true);
  const [isIOS, setIsIOS]         = useState(false);
  const [mounted, setMounted]     = useState(false);
  const [cfg, setCfg]             = useState<BannerConfig>(DEFAULT_BANNER);

  useEffect(() => {
    const already = localStorage.getItem("passport_banner_dismissed") === "1";
    setDismissed(already);
    setIsIOS(/iphone|ipad|ipod/i.test(navigator.userAgent));
    setMounted(true);
    fetch("/api/v1/passport/banner-config")
      .then(r => r.ok ? r.json() : null)
      .then(data => {
        if (data) {
          setCfg({
            banner_enabled:    data.banner_enabled !== false,
            banner_title:      data.banner_title      || DEFAULT_BANNER.banner_title,
            banner_subtitle:   data.banner_subtitle   || DEFAULT_BANNER.banner_subtitle,
            banner_cta_ios:    data.banner_cta_ios    || DEFAULT_BANNER.banner_cta_ios,
            banner_cta_android: data.banner_cta_android || DEFAULT_BANNER.banner_cta_android,
          });
          if (data.banner_enabled === false) setDismissed(true);
        }
      })
      .catch(() => {});
  }, []);

  if (!mounted || dismissed || !cfg.banner_enabled) return null;

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0, y: -8, scale: 0.98 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: -8, scale: 0.98 }}
        transition={{ duration: 0.35, ease: "easeOut" }}
        className="relative rounded-2xl overflow-hidden"
        style={{
          background: "linear-gradient(135deg, #0e1a2e 0%, #0d1120 100%)",
          border: "1px solid rgba(99,179,237,0.25)",
          boxShadow: "0 0 32px rgba(99,179,237,0.08), inset 0 1px 0 rgba(255,255,255,0.04)",
        }}
      >
        <div className="absolute top-0 left-0 right-0 h-[2px]"
          style={{ background: "linear-gradient(to right, transparent, rgba(99,179,237,0.6), transparent)" }} />
        <div className="absolute right-0 top-0 w-40 h-40 pointer-events-none"
          style={{ background: "radial-gradient(circle, rgba(99,179,237,0.12) 0%, transparent 70%)", transform: "translate(20%, -30%)" }} />
        <div className="relative p-4 flex items-start gap-3">
          <div className="relative flex-shrink-0 mt-0.5">
            <div className="w-10 h-10 rounded-xl flex items-center justify-center"
              style={{ background: "rgba(99,179,237,0.15)", border: "1px solid rgba(99,179,237,0.3)" }}>
              <CreditCard size={18} className="text-blue-300" />
            </div>
            <span className="absolute inset-0 rounded-xl animate-ping"
              style={{ background: "rgba(99,179,237,0.15)", animationDuration: "2s" }} />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-white font-black text-sm leading-snug">🎫 {cfg.banner_title}</p>
            <p className="text-white/50 text-[12px] mt-0.5 leading-relaxed">
              Track your <strong className="text-blue-300">{formatPoints(points)} pts</strong>
              {streak > 0 && <> and <strong className="text-orange-400">Day {streak} streak</strong></>}{" "}
              {cfg.banner_subtitle}
            </p>
            <div className="flex items-center gap-2 mt-2.5">
              <Link href="/passport">
                <button className="inline-flex items-center gap-1.5 text-[11px] font-black px-3 py-1.5 rounded-lg"
                  style={{ background: "rgba(99,179,237,0.15)", border: "1px solid rgba(99,179,237,0.3)", color: "#93c5fd" }}>
                  {isIOS ? <Smartphone size={12} /> : <Wallet size={12} />}
                  {isIOS ? cfg.banner_cta_ios : cfg.banner_cta_android}
                </button>
              </Link>
              <Link href="/passport">
                <button className="inline-flex items-center gap-1 text-[11px] font-black text-white/40 hover:text-white/70 transition-colors">
                  Learn more <ChevronRight size={11} />
                </button>
              </Link>
            </div>
          </div>
          <button onClick={() => { localStorage.setItem("passport_banner_dismissed", "1"); setDismissed(true); }}
            className="flex-shrink-0 p-1 rounded-lg text-white/25 hover:text-white/60 transition-colors">
            <X size={14} />
          </button>
        </div>
      </motion.div>
    </AnimatePresence>
  );
}

// ─── Spin Wheel Widget ────────────────────────────────────────────────────────
function SpinWheelWidget({ spinCredits }: { spinCredits: number }) {
  const [segments, setSegments]     = useState<Segment[]>(FALLBACK_SEGMENTS);
  const [rotation, setRotation]     = useState(0);
  const [spinning, setSpinning]     = useState(false);
  const [spun, setSpun]             = useState(false);
  const [outcome, setOutcome]       = useState<SpinOutcome | null>(null);
  const [showResult, setShowResult] = useState(false);

  const { mutate: mutateWallet } = useSWR("/user/wallet");

  useEffect(() => {
    api.getWheelConfig()
      .then((res: unknown) => {
        const r = res as Record<string, unknown>;
        const raw = ((r?.prizes ?? r?.segments ?? []) as Record<string, unknown>[]);
        const mapped = raw.filter(p => p.is_active !== false).map(p => ({
          label:       String(p.name ?? p.label ?? p.prize_name ?? "Prize"),
          prize_type:  String((p.prize_type ?? p.type ?? "try_again")).toLowerCase(),
          base_value:  Number(p.base_value ?? p.prize_value ?? p.value ?? 0),
          probability: Number(p.probability ?? 0),
          color:       String(p.prize_type === "try_again" ? "#374151" : (p.color_hex ?? p.color ?? "#5f72f9")),
          is_active:   true,
        }));
        if (mapped.length >= 2) setSegments(mapped);
      })
      .catch(() => {});
  }, []);

  const segAngle = 360 / segments.length;

  const handleSpin = useCallback(async () => {
    if (spinning || spun || spinCredits < 1) return;
    setSpinning(true);
    setShowResult(false);
    try {
      const res = await api.playSpin() as SpinOutcome;
      const targetIdx   = res.slot_index ?? 0;
      const targetAngle = targetIdx * segAngle + segAngle / 2;
      const extraSpins  = 6 + Math.random() * 2;
      const finalRot    = extraSpins * 360 + (360 - targetAngle);
      setRotation(prev => prev + finalRot);
      setTimeout(() => {
        setSpinning(false);
        setSpun(true);
        setOutcome(res);
        setShowResult(true);
        if (res.spin_result?.prize_type !== "try_again") {
          fireConfetti();
          toast.success(`🎉 ${res.prize_label}`, { duration: 6000 });
        } else {
          toast("Better luck next time!", { icon: "🔄" });
        }
        mutateWallet();
      }, 4600);
    } catch (e: unknown) {
      setSpinning(false);
      toast.error(e instanceof Error ? e.message : "Spin failed");
    }
  }, [spinning, spun, spinCredits, segAngle, mutateWallet]);

  const isWin = outcome?.spin_result?.prize_type !== "try_again";

  return (
    <div className="relative rounded-2xl overflow-hidden"
      style={{ background: "linear-gradient(135deg, #0f1018 0%, #141520 100%)", border: "1px solid rgba(245,166,35,0.15)" }}>
      <div className="absolute top-0 left-0 right-0 h-[2px]"
        style={{ background: "linear-gradient(to right, transparent, rgba(245,166,35,0.5), transparent)" }} />
      <div className="p-5 md:p-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-xl flex items-center justify-center"
              style={{ background: "rgba(245,166,35,0.12)", border: "1px solid rgba(245,166,35,0.25)" }}>
              <Sparkles size={17} style={{ color: "var(--gold)" }} />
            </div>
            <div>
              <h3 className="text-[15px] font-black text-white leading-none">Daily Spin &amp; Win</h3>
              <p className="text-[11px] text-white/40 mt-0.5">
                {spinCredits > 0
                  ? `${spinCredits} spin${spinCredits !== 1 ? "s" : ""} available`
                  : "Recharge ₦1,000+ to earn spins"}
              </p>
            </div>
          </div>
          <Link href="/spin">
            <button className="text-[11px] font-black flex items-center gap-1 hover:opacity-80 transition-opacity"
              style={{ color: "var(--gold)" }}>
              Full screen <ArrowRight className="w-3 h-3" />
            </button>
          </Link>
        </div>

        {/* Two-column: wheel + prize pool */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-start">
          {/* Wheel */}
          <div className="flex flex-col items-center gap-3">
            <div className="relative" style={{ width: 220, height: 220 }}>
              {/* Pointer */}
              <div className="absolute top-1/2 right-0 z-20 -translate-y-1/2 translate-x-1"
                style={{ width: 0, height: 0, borderTop: "10px solid transparent", borderBottom: "10px solid transparent", borderRight: "18px solid var(--gold)" }} />
              <motion.div
                className="absolute inset-0 rounded-full overflow-hidden"
                style={{
                  transform: `rotate(${rotation}deg)`,
                  transition: spinning ? "transform 4.5s cubic-bezier(0.17,0.67,0.12,0.99)" : "none",
                  background: `conic-gradient(${segments.map((s, i) => `${s.color} ${i * segAngle}deg ${(i + 1) * segAngle}deg`).join(", ")})`,
                  boxShadow: "0 0 0 4px rgba(245,166,35,0.2), 0 8px 32px rgba(0,0,0,0.5)",
                }}
              >
                {segments.map((seg, idx) => {
                  const angle = idx * segAngle + segAngle / 2;
                  const rad   = (angle * Math.PI) / 180;
                  const r     = 75;
                  return (
                    <div key={idx} className="absolute font-black text-center pointer-events-none"
                      style={{
                        left: `calc(50% + ${Math.cos(rad) * r}px - 22px)`,
                        top:  `calc(50% + ${Math.sin(rad) * r}px - 10px)`,
                        width: 44, transform: `rotate(${angle}deg)`,
                        textShadow: "1px 1px 3px rgba(0,0,0,0.9)",
                        fontSize: 8, color: "#fff", lineHeight: 1.2,
                      }}>
                      {seg.label.split(" ").map((w, i) => <div key={i}>{w}</div>)}
                    </div>
                  );
                })}
              </motion.div>
              {/* Centre button */}
              <div className="absolute inset-0 flex items-center justify-center">
                <motion.button
                  onClick={handleSpin}
                  disabled={spinning || spun || spinCredits < 1}
                  className="w-14 h-14 rounded-full font-black text-xs z-10 flex items-center justify-center disabled:opacity-40"
                  style={{
                    background: "linear-gradient(135deg, var(--gold), #F59E0B)",
                    boxShadow: "0 0 0 4px rgba(245,166,35,0.3), 0 4px 20px rgba(0,0,0,0.5)",
                    color: "#0d0e14",
                  }}
                  whileHover={!spinning && !spun && spinCredits > 0 ? { scale: 1.1 } : {}}
                  whileTap={!spinning && !spun && spinCredits > 0 ? { scale: 0.95 } : {}}
                >
                  {spinning ? <RotateCcw size={18} className="animate-spin" /> : "SPIN"}
                </motion.button>
              </div>
            </div>

            {/* Spin button */}
            {!spun ? (
              <motion.button
                onClick={handleSpin}
                disabled={spinning || spinCredits < 1}
                className="btn-gold rounded-xl h-10 px-6 text-[13px] font-black inline-flex items-center gap-2 disabled:opacity-40"
                whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}
              >
                {spinning
                  ? <><Loader2 size={14} className="animate-spin" /> Spinning…</>
                  : spinCredits < 1
                  ? <><Zap size={14} /> Recharge to Spin</>
                  : <><Sparkles size={14} /> Spin Now!</>}
              </motion.button>
            ) : (
              <button onClick={() => { setSpun(false); setOutcome(null); setShowResult(false); }}
                className="text-[12px] font-black text-white/40 hover:text-white/70 transition-colors underline underline-offset-2">
                Spin again
              </button>
            )}

            {/* Result */}
            <AnimatePresence>
              {showResult && outcome && (
                <motion.div
                  initial={{ opacity: 0, y: 10, scale: 0.95 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.95 }}
                  className={cn("w-full rounded-xl p-3.5 text-center border",
                    isWin ? "border-yellow-400/30" : "border-white/10")}
                  style={{ background: isWin ? "rgba(245,166,35,0.08)" : "rgba(255,255,255,0.04)" }}
                >
                  {isWin ? (
                    <>
                      <p className="text-[11px] font-black uppercase tracking-wider mb-1" style={{ color: "var(--gold)" }}>🎉 You Won!</p>
                      <p className="text-white font-black text-base">{outcome.prize_label}</p>
                      {outcome.spin_result?.prize_type === "momo_cash" && (
                        <Link href="/prizes" className="text-[11px] text-white/50 underline underline-offset-2 mt-1 block hover:text-white/70">
                          Claim in My Prizes →
                        </Link>
                      )}
                    </>
                  ) : (
                    <>
                      <p className="text-white/40 font-black text-sm">Not this time</p>
                      <p className="text-white/25 text-[11px] mt-0.5">Recharge for another spin!</p>
                    </>
                  )}
                </motion.div>
              )}
            </AnimatePresence>
          </div>

          {/* Prize Pool */}
          <div>
            <p className="text-[13px] font-black text-white mb-1">Today&apos;s Prize Pool</p>
            <p className="text-[11px] text-white/40 mb-3">Every spin wins something. Higher tier = bigger prizes.</p>
            <div className="space-y-1.5">
              {segments.filter(s => s.prize_type !== "try_again").map((seg, i) => (
                <div key={i} className="flex items-center justify-between py-1.5 px-2.5 rounded-lg"
                  style={{ background: "rgba(255,255,255,0.03)" }}>
                  <div className="flex items-center gap-2">
                    <div className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ background: seg.color }} />
                    <span className="text-[12px] font-black text-white">{seg.label}</span>
                  </div>
                  <span className="text-[10px] font-black text-white/35 uppercase tracking-wide">
                    {seg.prize_type === "momo_cash"    ? "Cash"    :
                     seg.prize_type === "pulse_points" ? "Points"  :
                     seg.prize_type === "data_bundle"  ? "Data"    :
                     seg.prize_type === "airtime"      ? "Airtime" :
                     seg.prize_type === "spin_credit"  ? "Spin"    : seg.prize_type}
                  </span>
                </div>
              ))}
            </div>
            <div className="mt-3 rounded-xl p-2.5 flex items-start gap-2"
              style={{ background: "rgba(245,166,35,0.05)", border: "1px solid rgba(245,166,35,0.12)" }}>
              <Sparkles size={12} className="flex-shrink-0 mt-0.5" style={{ color: "var(--gold)" }} />
              <p className="text-[10px] text-white/40 leading-relaxed">
                <strong style={{ color: "var(--gold)" }}>Tip:</strong> Reach Gold tier for 3× prize multiplier. Recharge ₦1,000+ to earn free spins.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Loyalty Ladder Widget ────────────────────────────────────────────────────
function LoyaltyLadder({ currentTier, lifetimePoints }: { currentTier: string; lifetimePoints: number }) {
  const TIERS = [
    { key: "BRONZE",   label: "Bronze",   min: 0,     emoji: "🥉", color: "#CD7F32" },
    { key: "SILVER",   label: "Silver",   min: 500,   emoji: "🥈", color: "#C0C0C0" },
    { key: "GOLD",     label: "Gold",     min: 1500,  emoji: "🥇", color: "#F5A623" },
    { key: "PLATINUM", label: "Platinum", min: 5000,  emoji: "💎", color: "#E5E4E2" },
  ];

  return (
    <div className="relative rounded-2xl overflow-hidden"
      style={{ background: "linear-gradient(135deg, #0f1018 0%, #141520 100%)", border: "1px solid rgba(255,255,255,0.07)" }}>
      <div className="p-5">
        <h3 className="text-[14px] font-black text-white mb-4">Your Loyalty Ladder</h3>
        <div className="space-y-2">
          {TIERS.map((tier, i) => {
            const isCurrent = tier.key === currentTier.toUpperCase();
            const nextTier  = TIERS[i + 1];
            const progress  = isCurrent && nextTier
              ? Math.min(100, ((lifetimePoints - tier.min) / (nextTier.min - tier.min)) * 100)
              : 0;
            return (
              <div key={tier.key}
                className={cn("rounded-xl px-3.5 py-2.5 transition-all border",
                  isCurrent ? "" : "border-transparent")}
                style={isCurrent ? {
                  background: `${tier.color}10`, borderColor: `${tier.color}30`,
                } : { background: "rgba(255,255,255,0.02)" }}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
                    <span className="text-base">{tier.emoji}</span>
                    <div>
                      <p className={cn("text-[13px] font-black", isCurrent ? "text-white" : "text-white/50")}>{tier.label}</p>
                      <p className="text-[10px] text-white/30">
                        {tier.min === 0 ? "Starting tier" : `${tier.min.toLocaleString()} pts`}
                      </p>
                    </div>
                  </div>
                  {isCurrent ? (
                    <span className="text-[9px] font-black uppercase tracking-wider px-2 py-0.5 rounded-full"
                      style={{ background: `${tier.color}20`, color: tier.color, border: `1px solid ${tier.color}30` }}>
                      Current
                    </span>
                  ) : lifetimePoints >= tier.min ? (
                    <span className="text-[11px] text-white/30">✓</span>
                  ) : (
                    <span className="text-[10px] text-white/20">{tier.min.toLocaleString()} pts</span>
                  )}
                </div>
                {isCurrent && nextTier && (
                  <div className="mt-2">
                    <div className="h-1 rounded-full overflow-hidden" style={{ background: "rgba(255,255,255,0.08)" }}>
                      <motion.div className="h-full rounded-full"
                        style={{ background: `linear-gradient(to right, ${tier.color}, ${tier.color}cc)` }}
                        initial={{ width: 0 }} animate={{ width: `${progress}%` }}
                        transition={{ duration: 1.2, delay: 0.3, ease: "easeOut" }} />
                    </div>
                    <p className="text-[10px] text-white/30 mt-1">
                      {(nextTier.min - lifetimePoints).toLocaleString()} pts to {nextTier.label}
                    </p>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ─── Quick AI Tools Widget ────────────────────────────────────────────────────
const TOOL_ICONS: Record<string, string> = {
  "ask-nexus": "💬", "web-search-ai": "🌐", "code-helper": "💻",
  "ai-photo": "📸", "ai-photo-dream": "🎨", "bg-remover": "✂️",
  "business-plan": "📋", "voice-to-plan": "🎙️", "marketing-jingle": "🎵",
};

function QuickAITools() {
  const { data: toolsData } = useSWR("/studio/tools",
    () => api.getStudioTools() as Promise<{ tools: StudioTool[] }>,
    { revalidateOnFocus: false }
  );
  const tools: StudioTool[] = ((toolsData as { tools?: StudioTool[] } | undefined)?.tools ?? [])
    .filter(t => t.is_free || t.point_cost <= 15)
    .slice(0, 6);

  return (
    <div className="relative rounded-2xl overflow-hidden"
      style={{ background: "linear-gradient(135deg, #0f1018 0%, #141520 100%)", border: "1px solid rgba(255,255,255,0.07)" }}>
      <div className="p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-[14px] font-black text-white">Quick AI Tools</h3>
          <Link href="/studio">
            <button className="text-[11px] font-black flex items-center gap-1 hover:opacity-80 transition-opacity"
              style={{ color: "var(--gold)" }}>
              All tools <ArrowRight className="w-3 h-3" />
            </button>
          </Link>
        </div>
        {tools.length > 0 ? (
          <div className="grid grid-cols-2 gap-2">
            {tools.map(tool => (
              <Link key={tool.id} href={`/studio?tool=${tool.slug}`}>
                <div className="rounded-xl p-3 flex items-center gap-2.5 hover:border-white/[0.15] transition-all cursor-pointer border"
                  style={{ background: "rgba(255,255,255,0.03)", borderColor: "rgba(255,255,255,0.06)" }}>
                  <span className="text-xl flex-shrink-0">{TOOL_ICONS[tool.slug] ?? "🤖"}</span>
                  <div className="min-w-0">
                    <p className="text-[12px] font-black text-white truncate leading-tight">{tool.name}</p>
                    <p className="text-[10px] font-black mt-0.5"
                      style={{ color: tool.is_free ? "#10b981" : "var(--gold)" }}>
                      {tool.is_free ? "Free" : `${tool.point_cost} pts`}
                    </p>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-2">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="rounded-xl h-14 animate-pulse"
                style={{ background: "rgba(255,255,255,0.04)" }} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Recent Activity Widget ───────────────────────────────────────────────────
function RecentActivity() {
  const { data: historyData } = useSWR(
    "/spin/history",
    () => api.getSpinHistory() as Promise<{ history: SpinHistoryItem[] }>,
    { refreshInterval: 60000 }
  );
  const history: SpinHistoryItem[] = ((historyData as { history?: SpinHistoryItem[] } | undefined)?.history ?? []).slice(0, 6);

  return (
    <div className="relative rounded-2xl overflow-hidden"
      style={{ background: "linear-gradient(135deg, #0f1018 0%, #141520 100%)", border: "1px solid rgba(255,255,255,0.07)" }}>
      <div className="p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-[14px] font-black text-white">Recent Activity</h3>
          <Link href="/prizes">
            <button className="text-[11px] font-black flex items-center gap-1 hover:opacity-80 transition-opacity"
              style={{ color: "var(--gold)" }}>
              View all <ArrowRight className="w-3 h-3" />
            </button>
          </Link>
        </div>
        {history.length > 0 ? (
          <div className="space-y-1">
            {history.map(item => (
              <div key={item.id} className="flex items-center justify-between py-2 px-2.5 rounded-xl"
                style={{ background: "rgba(255,255,255,0.02)" }}>
                <div className="flex items-center gap-2.5">
                  <div className="w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0"
                    style={{ background: item.prize_type === "try_again" ? "rgba(255,255,255,0.04)" : "rgba(245,166,35,0.10)" }}>
                    <Sparkles size={13} style={{ color: item.prize_type === "try_again" ? "rgba(255,255,255,0.2)" : "var(--gold)" }} />
                  </div>
                  <div>
                    <p className="text-[12px] font-black text-white leading-tight">
                      Daily Spin — {prizeLabel(item)}
                    </p>
                    <p className="text-[10px] text-white/30">{timeAgo(item.created_at)}</p>
                  </div>
                </div>
                <span className={cn("text-[12px] font-black flex-shrink-0",
                  item.prize_type === "try_again" ? "text-white/30" :
                  item.prize_type === "pulse_points" || item.prize_type === "spin_credit" ? "text-blue-400" : "text-green-400")}>
                  {item.prize_type !== "try_again" && `+${prizeLabel(item)}`}
                </span>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-6">
            <Sparkles size={28} className="text-white/10 mx-auto mb-2" />
            <p className="text-[12px] text-white/30">No spins yet — recharge to start winning!</p>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Regional Wars Widget ─────────────────────────────────────────────────────
function RegionalWarsWidget() {
  const { data: myRankData } = useSWR("/wars/my-rank",     () => api.getMyWarRank());
  const { data: lbData }     = useSWR("/wars/leaderboard", () => api.getWarsLeaderboard(3));

  const myRank      = myRankData as { ranked?: boolean; entry?: { state: string; total_points: number; rank: number; prize_kobo: number } } | undefined;
  const leaderboard = (lbData as { leaderboard?: Array<{ state: string; total_points: number; rank: number; prize_kobo: number }> } | undefined)?.leaderboard ?? [];

  return (
    <div className="relative rounded-2xl overflow-hidden"
      style={{ background: "linear-gradient(135deg, #0f1a12 0%, #0d0e14 100%)", border: "1px solid rgba(16,185,129,0.18)" }}>
      <div className="absolute top-0 left-0 right-0 h-[2px]"
        style={{ background: "linear-gradient(to right, transparent, rgba(16,185,129,0.5), transparent)" }} />
      <div className="p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-xl flex items-center justify-center"
              style={{ background: "rgba(16,185,129,0.12)", border: "1px solid rgba(16,185,129,0.25)" }}>
              <Trophy size={17} className="text-green-400" />
            </div>
            <div>
              <h3 className="text-[14px] font-black text-white leading-none">Regional Wars</h3>
              <p className="text-[11px] text-white/40 mt-0.5">₦500K monthly prize pool</p>
            </div>
          </div>
          <Link href="/wars">
            <button className="text-[11px] font-black text-green-400 flex items-center gap-1 hover:text-green-300 transition-colors">
              View all <ArrowRight className="w-3 h-3" />
            </button>
          </Link>
        </div>
        {myRank?.ranked && myRank.entry ? (
          <div className="rounded-xl p-3.5 mb-3 flex items-center justify-between"
            style={{ background: "rgba(16,185,129,0.08)", border: "1px solid rgba(16,185,129,0.15)" }}>
            <div className="flex items-center gap-2.5">
              <MapPin className="w-4 h-4 text-green-400 flex-shrink-0" />
              <div>
                <p className="text-[13px] font-black text-white">{myRank.entry.state}</p>
                <p className="text-[11px] text-white/40">{formatPoints(myRank.entry.total_points)} pts</p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-[10px] text-white/40 mb-0.5">Your rank</p>
              <p className="text-xl font-black text-green-400">#{myRank.entry.rank}</p>
            </div>
          </div>
        ) : (
          <div className="rounded-xl p-3.5 mb-3 flex items-center gap-2.5"
            style={{ background: "rgba(16,185,129,0.05)", border: "1px solid rgba(16,185,129,0.10)" }}>
            <MapPin className="w-4 h-4 text-green-400/50 flex-shrink-0" />
            <p className="text-[12px] text-white/40">Recharge to earn points and join your state&apos;s battle</p>
          </div>
        )}
        {leaderboard.length > 0 && (
          <div className="space-y-1.5">
            {leaderboard.slice(0, 3).map((entry, i) => {
              const medals = ["🥇", "🥈", "🥉"];
              const colors = ["#F5A623", "#C0C0C0", "#CD7F32"];
              const prize  = entry.prize_kobo > 0 ? `₦${(entry.prize_kobo / 100).toLocaleString("en-NG")}` : null;
              return (
                <div key={entry.state} className="flex items-center gap-3 rounded-xl px-3 py-2.5"
                  style={{ background: "rgba(255,255,255,0.03)" }}>
                  <span className="text-base w-5 text-center flex-shrink-0">{medals[i]}</span>
                  <div className="flex-1 min-w-0">
                    <p className="text-[13px] font-black text-white truncate">{entry.state}</p>
                    <p className="text-[10px] text-white/35 font-mono">{formatPoints(entry.total_points)} pts</p>
                  </div>
                  {prize && <span className="text-[11px] font-black flex-shrink-0" style={{ color: colors[i] }}>{prize}</span>}
                </div>
              );
            })}
          </div>
        )}
        <div className="mt-3 rounded-xl p-2.5 flex items-start gap-2"
          style={{ background: "rgba(245,166,35,0.05)", border: "1px solid rgba(245,166,35,0.12)" }}>
          <Gift className="w-3.5 h-3.5 flex-shrink-0 mt-0.5" style={{ color: "var(--gold)" }} />
          <p className="text-[10px] text-white/40 leading-relaxed">
            <strong style={{ color: "var(--gold)" }}>Individual draw:</strong> One random member from each top-3 state wins a personal MoMo cash payout at month end.
          </p>
        </div>
      </div>
    </div>
  );
}

// ─── Main Dashboard Page ──────────────────────────────────────────────────────
export default function DashboardPage() {
  const { setUser, setWallet, wallet: storedWallet } = useStore();
  const fetcher = (key: string) => {
    if (key === "/user/profile")     return api.getProfile();
    if (key === "/user/wallet")      return api.getWallet();
    if (key === "/user/bonus-pulse") return api.getBonusPulseAwards();
    return Promise.resolve(null);
  };
  const { data: profile }   = useSWR("/user/profile",     fetcher, { onSuccess: (d: unknown) => setUser(d as Parameters<typeof setUser>[0]) });
  const { data: wallet }    = useSWR("/user/wallet",      fetcher, { onSuccess: (d: unknown) => setWallet(d as Parameters<typeof setWallet>[0]) });
  const { data: bonusData } = useSWR("/user/bonus-pulse", fetcher);
  const p = profile as { phone_number?: string; tier?: string; streak_count?: number; total_spins?: number; studio_use_count?: number } | undefined;
  // Use SWR data when available, fall back to persisted store value to avoid flash-to-zero
  const w = (wallet ?? storedWallet) as { pulse_points?: number; spin_credits?: number; lifetime_points?: number } | undefined;
  const b = bonusData as { total_bonus?: number } | undefined;

  const tier        = (p?.tier ?? "BRONZE").toUpperCase();
  const tierColor   = TIER_HEX[tier]   ?? "#CD7F32";
  const tierEmoji   = TIER_EMOJI[tier] ?? "🥉";
  const pulsePoints = w?.pulse_points  ?? 0;
  const spinCredits = w?.spin_credits  ?? 0;
  const lifetimePts = w?.lifetime_points ?? 0;
  const streak      = p?.streak_count  ?? 0;
  const totalSpins  = p?.total_spins   ?? 0;
  const studioUses  = p?.studio_use_count ?? 0;
  const totalBonus  = b?.total_bonus   ?? 0;

  const tierData = TIER_THRESHOLDS.find(t => t.tier === tier) || TIER_THRESHOLDS[0];
  const nextTier = TIER_THRESHOLDS[TIER_THRESHOLDS.indexOf(tierData) + 1];
  const progress = nextTier
    ? Math.min(100, (lifetimePts - tierData.min) / (nextTier.min - tierData.min) * 100)
    : 100;

  const phone = p?.phone_number ?? "";
  const displayName = phone.length >= 11
    ? `${phone.slice(0, 4)}****${phone.slice(-4)}`
    : phone || "Loading…";

  const STATS = [
    { icon: RotateCcw, label: "Total Spins",  value: totalSpins.toLocaleString(),  color: "#5f72f9", sub: "All time" },
    { icon: Wand2,     label: "Studio Uses",  value: studioUses.toLocaleString(),  color: "#8B5CF6", sub: "All time" },
    { icon: Award,     label: "Bonus Awards", value: formatPoints(totalBonus),     color: "#F5A623", sub: "Total earned" },
  ];

  return (
    <AppShell>
      <Toaster position="top-center" toastOptions={{
        style: { background: "#1c2038", color: "#fff", border: "1px solid rgba(255,255,255,0.1)" },
      }} />

      <div className="max-w-7xl mx-auto px-4 md:px-6 py-6 space-y-5 pb-24 md:pb-8">

        {/* ── Passport Banner ── */}
        <PassportBanner points={pulsePoints} streak={streak} />

        {/* ── Welcome Hero Card ── */}
        <motion.div
          initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
          className="relative rounded-2xl p-5 md:p-6 overflow-hidden"
          style={{
            background: "linear-gradient(135deg, #1a1c2e 0%, #0f1018 100%)",
            border: "1px solid rgba(245,166,35,0.18)",
            boxShadow: "0 0 40px rgba(245,166,35,0.07), inset 0 1px 0 rgba(255,255,255,0.05)",
          }}
        >
          <div className="absolute top-0 left-0 right-0 h-[2px]"
            style={{ background: "linear-gradient(to right, transparent, rgba(245,166,35,0.6), transparent)" }} />
          <div className="absolute top-0 right-0 w-64 h-64 pointer-events-none"
            style={{ background: "radial-gradient(circle, rgba(245,166,35,0.07) 0%, transparent 70%)", transform: "translate(20%, -30%)" }} />

          <div className="relative">
            {/* Top row */}
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-5">
              {/* Left: avatar + name */}
              <div className="flex items-center gap-3.5">
                <div className="w-12 h-12 rounded-2xl flex items-center justify-center flex-shrink-0 font-black text-lg"
                  style={{ background: `${tierColor}20`, border: `2px solid ${tierColor}40`, color: tierColor }}>
                  {tierEmoji}
                </div>
                <div>
                  <p className="text-white/50 text-[11px] font-black uppercase tracking-wider">Welcome back,</p>
                  <p className="text-white font-black text-lg leading-tight">{displayName}</p>
                  <div className="flex items-center gap-1.5 mt-0.5 flex-wrap">
                    <span className="text-[11px] font-black px-2 py-0.5 rounded-full"
                      style={{ background: `${tierColor}18`, border: `1px solid ${tierColor}30`, color: tierColor }}>
                      {tierEmoji} {tier.charAt(0) + tier.slice(1).toLowerCase()} Tier
                    </span>
                    {streak > 0 && (
                      <span className="flex items-center gap-1 text-[11px] font-black text-orange-400">
                        <Flame size={11} /> {streak}d streak
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Right: key stats */}
              <div className="flex items-center gap-4 md:gap-6">
                <div className="text-center">
                  <p className="font-black text-xl md:text-2xl leading-none" style={{ color: "var(--gold)" }}>
                    {pulsePoints >= 1000 ? `${(pulsePoints / 1000).toFixed(1)}K` : pulsePoints.toLocaleString()}
                  </p>
                  <p className="text-white/40 text-[10px] font-black uppercase tracking-wide mt-0.5">Pulse Points</p>
                </div>
                <div className="w-px h-8 bg-white/10" />
                <div className="text-center">
                  <p className="text-white font-black text-xl md:text-2xl leading-none">{spinCredits}</p>
                  <p className="text-white/40 text-[10px] font-black uppercase tracking-wide mt-0.5">Spin Credits</p>
                </div>
                <div className="w-px h-8 bg-white/10" />
                <div className="text-center">
                  <p className="text-white font-black text-xl md:text-2xl leading-none">
                    {lifetimePts >= 1000 ? `${(lifetimePts / 1000).toFixed(1)}K` : lifetimePts.toLocaleString()}
                  </p>
                  <p className="text-white/40 text-[10px] font-black uppercase tracking-wide mt-0.5">Lifetime Pts</p>
                </div>
              </div>
            </div>

            {/* Progress bar */}
            {nextTier && (
              <div>
                <div className="flex justify-between text-[11px] text-white/40 mb-1.5">
                  <span className="font-black">Progress to {nextTier.label}</span>
                  <span className="font-black">
                    {lifetimePts >= 1000 ? `${(lifetimePts / 1000).toFixed(1)}K` : lifetimePts} / {nextTier.min >= 1000 ? `${(nextTier.min / 1000).toFixed(0)}K` : nextTier.min} pts
                  </span>
                </div>
                <div className="h-2 rounded-full overflow-hidden" style={{ background: "rgba(255,255,255,0.08)" }}>
                  <motion.div className="h-full rounded-full"
                    style={{ background: `linear-gradient(to right, ${tierColor}, ${tierColor}cc)` }}
                    initial={{ width: 0 }} animate={{ width: `${progress}%` }}
                    transition={{ duration: 1.2, delay: 0.3, ease: "easeOut" }} />
                </div>
              </div>
            )}
          </div>
        </motion.div>

        {/* ── Stats Row ── */}
        <motion.div
          initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.08 }}
          className="grid grid-cols-2 md:grid-cols-4 gap-3"
        >
          {STATS.map(({ icon: Icon, label, value, color, sub }, i) => (
            <motion.div key={label}
              initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.08 + i * 0.04 }}
              className="rounded-2xl p-4"
              style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.07)" }}>
              <div className="flex items-center gap-2 mb-2">
                <div className="w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0"
                  style={{ background: `${color}15`, border: `1px solid ${color}25` }}>
                  <Icon size={14} style={{ color }} />
                </div>
                <p className="text-[11px] font-black text-white/50 uppercase tracking-wide">{label}</p>
              </div>
              <p className="text-white font-black text-2xl leading-none">{value}</p>
              <p className="text-white/30 text-[10px] mt-1">{sub}</p>
            </motion.div>
          ))}
        </motion.div>

        {/* ── Spin & Win (full width) ── */}
        <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.16 }}>
          <SpinWheelWidget spinCredits={spinCredits} />
        </motion.div>

        {/* ── Two-column grid ── */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
          {/* Left column */}
          <div className="space-y-5">
            <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.22 }}>
              <LoyaltyLadder currentTier={tier} lifetimePoints={lifetimePts} />
            </motion.div>
            <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.28 }}>
              <RegionalWarsWidget />
            </motion.div>
          </div>

          {/* Right column */}
          <div className="space-y-5">
            <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.24 }}>
              <QuickAITools />
            </motion.div>
            <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.30 }}>
              <RecentActivity />
            </motion.div>
          </div>
        </div>

        {/* ── Coming Soon row ── */}
        <motion.div
          initial={{ opacity: 0, y: 14 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.36 }}
          className="grid grid-cols-1 sm:grid-cols-2 gap-3"
        >
          {[
            { icon: Clock, color: "#00D4FF", title: "Daily Draw",     body: "Win prizes daily just for being active." },
            { icon: Star,  color: "#8B5CF6", title: "Weekly Jackpot", body: "Bigger prizes for top rechargees each week." },
          ].map(({ icon: Icon, color, title, body }) => (
            <div key={title} className="glass rounded-2xl border border-white/[0.07] p-4 opacity-75">
              <div className="flex items-start justify-between mb-2.5">
                <div className="w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0"
                  style={{ background: `${color}12`, border: `1px solid ${color}20`, color }}>
                  <Icon size={15} />
                </div>
                <span className="text-[9px] font-black uppercase tracking-wider px-2 py-0.5 rounded-full"
                  style={{ background: `${color}10`, color, border: `1px solid ${color}20` }}>
                  Soon
                </span>
              </div>
              <h4 className="text-[13px] font-black text-white mb-1">{title}</h4>
              <p className="text-[11px] text-white/35 leading-relaxed">{body}</p>
            </div>
          ))}
        </motion.div>

        {/* ── Recharge CTA ── */}
        <motion.div
          initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.42 }}
          className="relative rounded-2xl p-4 flex items-center justify-between overflow-hidden"
          style={{
            background: "linear-gradient(135deg, rgba(245,166,35,0.08) 0%, rgba(245,166,35,0.03) 100%)",
            border: "1px solid rgba(245,166,35,0.18)",
          }}
        >
          <div>
            <p className="text-white font-black text-sm">Recharge to earn more ⚡</p>
            <p className="text-white/40 text-[12px] mt-0.5">₦200 = 1 Pulse Point · ₦1,000+ = free spin</p>
          </div>
          <Link href="/recharge">
            <button className="btn-gold rounded-xl h-9 px-4 text-[12px] font-black inline-flex items-center gap-1.5 flex-shrink-0">
              <Zap className="w-3.5 h-3.5" /> Recharge
            </button>
          </Link>
        </motion.div>

      </div>
    </AppShell>
  );
}
