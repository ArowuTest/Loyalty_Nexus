"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import useSWR from "swr";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import toast, { Toaster } from "react-hot-toast";
import Link from "next/link";
import { Zap, Trophy, RotateCcw, Gift, X, Sparkles, Loader2, History, Info } from "lucide-react";
import { cn } from "@/lib/utils";
import DailySpinProgress from "@/components/spin/DailySpinProgress";
import { useStore } from "@/store/useStore";

// ── Fallback segments used only if API is unavailable ──────────────────────
const FALLBACK_SEGMENTS = [
  { label: "₦500 Airtime",  prize_type: "airtime",       base_value: 50000,  probability: 20, color: "#10b981", is_active: true },
  { label: "Try Again",     prize_type: "try_again",      base_value: 0,      probability: 30, color: "#4b5563", is_active: true },
  { label: "100 Points",    prize_type: "pulse_points",   base_value: 100,    probability: 20, color: "#5f72f9", is_active: true },
  { label: "₦1k Data",      prize_type: "data_bundle",    base_value: 100000, probability: 10, color: "#06b6d4", is_active: true },
  { label: "50 Points",     prize_type: "pulse_points",   base_value: 50,     probability: 25, color: "#a78bfa", is_active: true },
  { label: "₦2k Cash",      prize_type: "momo_cash",      base_value: 200000, probability: 5,  color: "#f59e0b", is_active: true },
  { label: "Try Again",     prize_type: "try_again",      base_value: 0,      probability: 25, color: "#374151", is_active: true },
  { label: "₦5k Cash",      prize_type: "momo_cash",      base_value: 500000, probability: 2,  color: "#f43f5e", is_active: true },
];

interface Segment { label: string; prize_type: string; base_value: number; probability: number; color: string; is_active: boolean; }
interface SpinOutcome {
  spin_result?: { id: string; prize_type: string; prize_value: number; slot_index: number; fulfillment_status: string };
  prize_label: string; slot_index: number; message?: string; needs_momo_setup?: boolean;
}
interface SpinHistoryItem {
  id: string; prize_type: string; prize_value: number; fulfillment_status: string; created_at: string;
}
interface Wallet { spin_credits: number; pulse_points: number; }

function fireConfetti() {
  if (typeof window === "undefined") return;
  import("canvas-confetti").then(({ default: confetti }) => {
    confetti({ particleCount: 160, spread: 360, origin: { x: 0.5, y: 0.5 },
      colors: ["#5f72f9","#f9c74f","#10b981","#f43f5e","#06b6d4"], startVelocity: 28, ticks: 60 });
  }).catch(() => {});
}

function prizeLabel(item: SpinHistoryItem): string {
  if (item.prize_type === "try_again") return "Try Again";
  if (item.prize_type === "pulse_points") return `${item.prize_value} Points`;
  if (item.prize_type === "airtime") return `₦${(item.prize_value / 100).toLocaleString()} Airtime`;
  if (item.prize_type === "data_bundle") return `Data Bundle`;
  if (item.prize_type === "momo_cash") return `₦${(item.prize_value / 100).toLocaleString()} Cash`;
  return item.prize_type;
}

function statusBadge(status: string) {
  const map: Record<string, {label:string;cls:string}> = {
    "completed":     { label: "Credited", cls: "bg-green-400/15 text-green-400" },
    "pending":       { label: "Pending",  cls: "bg-yellow-400/15 text-yellow-400" },
    "pending_momo":  { label: "Need MoMo",cls: "bg-orange-400/15 text-orange-400" },
    "failed":        { label: "Failed",   cls: "bg-red-400/15 text-red-400" },
    "n/a":           { label: "No Prize", cls: "bg-white/5 text-white/30" },
  };
  const s = map[status?.toLowerCase()] ?? { label: status, cls: "bg-white/5 text-white/40" };
  return <span className={cn("text-[10px] font-bold px-2 py-0.5 rounded-full", s.cls)}>{s.label}</span>;
}

export default function SpinPage() {
  const wheelRef = useRef<HTMLDivElement>(null);
  const [segments, setSegments] = useState<Segment[]>(FALLBACK_SEGMENTS);
  const [loadingSegments, setLoadingSegments] = useState(true);
  const [rotation, setRotation] = useState(0);
  const [spinning, setSpinning] = useState(false);
  const [spun, setSpun] = useState(false);
  const [outcome, setOutcome] = useState<SpinOutcome | null>(null);
  const [showResult, setShowResult] = useState(false);

  // Wallet — shows spin credits; fall back to persisted store value to avoid flash-to-zero
  const storedWallet = useStore((s) => s.wallet);
  const { data: walletData, mutate: mutateWallet } = useSWR<Wallet>(
    "/user/wallet", () => api.getWallet() as Promise<Wallet>, { refreshInterval: 30000 }
  );
  const effectiveWallet = walletData ?? (storedWallet as Wallet | null) ?? null;
  const spinCredits = effectiveWallet?.spin_credits ?? 0;

  // Spin history
  const { data: historyData, mutate: mutateHistory } = useSWR(
    "/spin/history", () => api.getSpinHistory() as Promise<{ history: SpinHistoryItem[] }>,
    { refreshInterval: 15000 }
  );
  const history: SpinHistoryItem[] = (historyData as any)?.history ?? [];

  // Load real wheel config from backend
  // Backend returns: { slots: [{ index, prize_type, label, color, icon_name, is_no_win, no_win_message }], required_credits }
  useEffect(() => {
    api.getWheelConfig()
      .then((res: any) => {
        // Support both response shapes: { slots: [...] } and legacy { prizes: [...] }
        const raw: any[] = res?.slots ?? res?.prizes ?? res?.segments ?? [];
        const mapped = raw
          .map((p: any): Segment => ({
            label:       p.label ?? p.name ?? p.prize_name ?? "Prize",
            prize_type:  (p.prize_type ?? p.type ?? "try_again").toLowerCase(),
            base_value:  Number(p.base_value ?? p.prize_value ?? p.value ?? 0),
            probability: Number(p.probability ?? p.win_probability_weight ?? 0),
            // Backend sends 'color' field; try_again slots get a dark grey
            color:       (p.prize_type === "try_again" || p.is_no_win)
                           ? (p.color ?? "#374151")
                           : (p.color ?? p.color_hex ?? "#5f72f9"),
            is_active:   true,
          }));
        if (mapped.length >= 2) setSegments(mapped);
      })
      .catch(() => {/* use fallback segments */})
      .finally(() => setLoadingSegments(false));
  }, []);

  const segAngle = 360 / segments.length;

  const handleSpin = useCallback(async () => {
    if (spinning || spun) return;
    if (spinCredits < 1) {
      toast.error("No spin credits! Recharge to earn spins. 💫");
      return;
    }
    setSpinning(true);
    setShowResult(false);
    try {
      const res = await api.playSpin() as SpinOutcome;
      // Animate to the correct segment returned by server
      const targetIdx = res.slot_index ?? 0;
      const targetAngle = targetIdx * segAngle + segAngle / 2;
      const extraSpins = 6 + Math.random() * 2;
      const finalRotation = extraSpins * 360 + (360 - targetAngle);
      setRotation(prev => prev + finalRotation);

      setTimeout(() => {
        setSpinning(false);
        setSpun(true);
        setOutcome(res);
        setShowResult(true);
        if (res.spin_result?.prize_type !== "try_again") {
          fireConfetti();
          toast.success(`🎉 ${res.prize_label}`, { duration: 6000 });
        } else {
          toast("Better luck next time! Keep recharging for more spins.", { icon: "🔄" });
        }
        mutateWallet();
        mutateHistory();
      }, 4600);
    } catch (e: unknown) {
      setSpinning(false);
      const msg = e instanceof Error ? e.message : "Spin failed";
      toast.error(msg);
    }
  }, [spinning, spun, spinCredits, segAngle, mutateWallet, mutateHistory]);

  const handleReset = () => {
    setSpun(false);
    setOutcome(null);
    setShowResult(false);
  };

  const isWin = outcome && outcome.spin_result?.prize_type !== "try_again";

  return (
    <AppShell>
      <Toaster position="top-center" toastOptions={{
        style: { background: "#1c2038", color: "#fff", border: "1px solid rgba(255,255,255,0.1)" },
      }} />

      <div className="max-w-5xl mx-auto px-4 md:px-6 py-6 pb-28 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold font-display text-white flex items-center gap-2">
              <Sparkles className="text-yellow-400" size={22} /> Spin & Win
            </h1>
            <p className="text-white/40 text-sm mt-0.5">Earn spins by recharging. Every ₦1,000 = 1 spin.</p>
          </div>
          <Link href="/prizes" className="flex items-center gap-1.5 text-xs text-nexus-300 hover:text-nexus-200 transition-colors">
            <History size={14} /> My Prizes
          </Link>
        </div>

        {/* Spin Credits badge */}
        <div className={cn(
          "nexus-card px-4 py-3 flex items-center justify-between",
          spinCredits > 0 ? "border-nexus-500/30" : "border-white/5"
        )}>
          <div className="flex items-center gap-2">
            <Zap className={spinCredits > 0 ? "text-yellow-400" : "text-white/20"} size={18} />
            <span className="text-white/60 text-sm">Available Spins</span>
          </div>
          <div className="flex items-center gap-2">
            <span className={cn(
              "text-2xl font-bold font-display",
              spinCredits > 0 ? "text-yellow-400" : "text-white/30"
            )}>{effectiveWallet ? spinCredits : "—"}</span>
            {spinCredits === 0 && (
              <span className="text-[10px] text-white/30 border border-white/10 rounded-full px-2 py-0.5">
                Recharge to earn
              </span>
            )}
          </div>
        </div>

        {/* Tier progress — mirrors RechargeMax DailySpinProgress */}
        <DailySpinProgress refreshKey={history.length} />

        {/* Wheel */}
        <div className="nexus-card p-6">
          <div className="relative mx-auto" style={{ width: 280, height: 280 }}>
            {loadingSegments ? (
              <div className="w-full h-full rounded-full flex items-center justify-center border-2 border-nexus-500/20">
                <Loader2 className="text-nexus-400 animate-spin" size={32} />
              </div>
            ) : (
              <>
                {/* Pointer */}
                <div className="absolute -top-3 left-1/2 -translate-x-1/2 z-10">
                  <div className="w-0 h-0 border-l-[10px] border-r-[10px] border-b-[22px] border-l-transparent border-r-transparent border-b-yellow-400 drop-shadow-lg" />
                </div>

                {/* Wheel disc */}
                <div
                  ref={wheelRef}
                  className="w-full h-full rounded-full overflow-hidden relative border-4 border-nexus-500/50"
                  style={{
                    transform: `rotate(${rotation}deg)`,
                    transition: spinning ? "transform 4600ms cubic-bezier(0.17,0.67,0.12,0.99)" : "none",
                    boxShadow: spinning
                      ? "0 0 40px rgba(95,114,249,0.6), 0 0 80px rgba(249,199,79,0.3)"
                      : "0 0 20px rgba(95,114,249,0.25)",
                    background: `conic-gradient(${segments.map((s, i) =>
                      `${s.color} ${i * segAngle}deg ${(i + 1) * segAngle}deg`
                    ).join(", ")})`,
                  }}
                >
                  {segments.map((seg, idx) => {
                    const angle = idx * segAngle + segAngle / 2;
                    const rad = (angle * Math.PI) / 180;
                    const r = 95;
                    return (
                      <div
                        key={idx}
                        className="absolute font-bold text-center pointer-events-none"
                        style={{
                          left: `calc(50% + ${Math.cos(rad) * r}px - 28px)`,
                          top: `calc(50% + ${Math.sin(rad) * r}px - 12px)`,
                          width: 56, transform: `rotate(${angle}deg)`,
                          textShadow: "1px 1px 3px rgba(0,0,0,0.85)",
                          fontSize: 9, color: "#fff", lineHeight: 1.2,
                        }}
                      >
                        {seg.label.split(" ").map((w, i) => <div key={i}>{w}</div>)}
                      </div>
                    );
                  })}
                </div>

                {/* Centre button */}
                <div className="absolute inset-0 flex items-center justify-center">
                  <motion.button
                    onClick={handleSpin}
                    disabled={spinning || spun || spinCredits < 1}
                    className="w-16 h-16 rounded-full font-black text-sm text-nexus-600 bg-white disabled:opacity-50 z-10 flex items-center justify-center"
                    style={{ boxShadow: "0 0 0 4px rgba(95,114,249,0.4), 0 4px 20px rgba(0,0,0,0.4)", border: "3px solid rgba(95,114,249,0.7)" }}
                    whileHover={!spinning && !spun && spinCredits > 0 ? { scale: 1.1 } : {}}
                    whileTap={!spinning && !spun && spinCredits > 0 ? { scale: 0.95 } : {}}
                  >
                    {spinning ? <RotateCcw size={22} className="animate-spin text-nexus-500" /> : "SPIN"}
                  </motion.button>
                </div>
              </>
            )}
          </div>

          {/* Result */}
          <AnimatePresence>
            {showResult && outcome && (
              <motion.div
                initial={{ opacity: 0, y: 16, scale: 0.9 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, scale: 0.9 }}
                transition={{ type: "spring", damping: 22, stiffness: 260 }}
                className={cn(
                  "mt-5 rounded-2xl p-4 text-center space-y-2",
                  isWin
                    ? "bg-gradient-to-br from-nexus-900/60 to-yellow-900/30 border border-yellow-400/30"
                    : "bg-white/5 border border-white/10"
                )}
              >
                {isWin ? (
                  <>
                    <motion.div animate={{ rotate: [0,-10,10,-5,5,0], scale: [1,1.2,1] }} transition={{ duration: 0.6 }}>
                      <Trophy className="w-10 h-10 text-yellow-400 mx-auto" />
                    </motion.div>
                    <p className="text-yellow-300 text-xs font-bold uppercase tracking-wider">🎉 You Won!</p>
                    <p className="text-white text-xl font-bold font-display">{outcome.prize_label}</p>
                    <p className="text-white/40 text-xs">
                      {outcome.spin_result?.prize_type === "momo_cash"
                        ? "Go to My Prizes to submit your bank/MoMo details."
                        : outcome.spin_result?.prize_type === "pulse_points"
                        ? "Points added to your wallet!"
                        : "Being credited to your phone within 5–10 minutes."}
                    </p>
                  </>
                ) : (
                  <>
                    <RotateCcw className="w-9 h-9 text-white/30 mx-auto" />
                    <p className="text-white/50 text-sm font-semibold">Not this time</p>
                    <p className="text-white/30 text-xs">Recharge to earn another spin chance!</p>
                  </>
                )}
                <button onClick={handleReset} className="mt-2 text-xs text-white/40 hover:text-white/70 transition-colors underline">
                  Spin again
                </button>
              </motion.div>
            )}
          </AnimatePresence>

          {/* Spin button (below wheel) */}
          {!spun && (
            <div className="mt-5 space-y-2">
              <motion.button
                onClick={handleSpin}
                disabled={spinning || spinCredits < 1 || loadingSegments}
                className="w-full py-3.5 rounded-2xl font-bold text-white bg-gradient-to-r from-nexus-600 to-nexus-500 disabled:opacity-40 flex items-center justify-center gap-2"
                whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}
              >
                {spinning
                  ? <><Loader2 size={16} className="animate-spin" /> Spinning…</>
                  : spinCredits > 0
                  ? <><Zap size={16} /> Spin Now! ({spinCredits} left)</>
                  : "No Spins — Recharge to Earn"
                }
              </motion.button>
              {spinCredits === 0 && (
                <p className="text-center text-xs text-white/30">
                  Every ₦1,000 recharge = 1 spin credit
                </p>
              )}
            </div>
          )}
        </div>

        {/* Prize list */}
        <div className="nexus-card p-4">
          <p className="text-white/40 text-xs font-semibold uppercase tracking-widest mb-3 flex items-center gap-1.5">
            <Info size={12} /> Possible Prizes
          </p>
          <div className="grid grid-cols-2 gap-2">
            {segments.map((seg, i) => (
              <div key={i} className="flex items-center gap-2 text-xs text-white/60">
                <div className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ backgroundColor: seg.color }} />
                <span className={seg.prize_type === "try_again" ? "text-white/25 italic" : ""}>{seg.label}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Spin History */}
        {history.length > 0 && (
          <div className="space-y-2">
            <p className="text-white/40 text-xs font-semibold uppercase tracking-widest flex items-center gap-1.5">
              <History size={12} /> Recent Spins
            </p>
            <div className="space-y-2">
              {history.slice(0, 5).map((item) => (
                <motion.div
                  key={item.id}
                  initial={{ opacity: 0, x: -8 }}
                  animate={{ opacity: 1, x: 0 }}
                  className="nexus-card px-4 py-2.5 flex items-center justify-between"
                >
                  <div className="flex items-center gap-2">
                    <Gift size={14} className={item.prize_type === "try_again" ? "text-white/20" : "text-nexus-400"} />
                    <span className="text-white/70 text-sm">{prizeLabel(item)}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    {statusBadge(item.fulfillment_status)}
                    <span className="text-white/25 text-[10px]">
                      {new Date(item.created_at).toLocaleDateString("en-NG", { month: "short", day: "numeric" })}
                    </span>
                  </div>
                </motion.div>
              ))}
            </div>
            <Link href="/prizes" className="block text-center text-xs text-nexus-400 hover:text-nexus-300 transition-colors py-2">
              View all spin history →
            </Link>
          </div>
        )}
      </div>
    </AppShell>
  );
}
