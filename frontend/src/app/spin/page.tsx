"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import useSWR from "swr";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import toast, { Toaster } from "react-hot-toast";
import Link from "next/link";
import {
  Zap, Trophy, RotateCcw, Gift, X, Sparkles, Loader2,
  History, Info, CheckCircle, ChevronDown, CreditCard, Smartphone
} from "lucide-react";
import { cn } from "@/lib/utils";
import DailySpinProgress from "@/components/spin/DailySpinProgress";
import { useStore } from "@/store/useStore";

// ── Nigerian banks list ────────────────────────────────────────────────────
const NIGERIAN_BANKS = [
  "Access Bank","Access Bank (Diamond)","Carbon","Citibank Nigeria",
  "Ecobank Nigeria","FCMB","FBNQuest","Fidelity Bank","First Bank of Nigeria",
  "First City Monument Bank","Globus Bank","Guaranty Trust Bank (GTBank)",
  "Heritage Bank","Jaiz Bank","Keystone Bank","Kuda Bank","Lotus Bank",
  "MainStreet Bank","Moniepoint MFB","OPay","Opay Digital Services",
  "Palmpay","Parallex Bank","Polaris Bank","Providus Bank","PremiumTrust Bank",
  "Rand Merchant Bank","Rubies MFB","Signature Bank","Sparkle Microfinance Bank",
  "Standard Chartered Bank","Sterling Bank","SunTrust Bank","Taj Bank",
  "Titan Trust Bank","Union Bank of Nigeria","United Bank For Africa (UBA)",
  "Unity Bank","VFD Microfinance Bank","Wema Bank","Zenith Bank",
];

// ── Fallback segments ──────────────────────────────────────────────────────
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
interface SpinResult { id: string; prize_type: string; prize_value: number; slot_index: number; fulfillment_status: string; claim_status?: string; }
interface SpinOutcome { spin_result?: SpinResult; prize_label: string; slot_index: number; message?: string; needs_momo_setup?: boolean; }
interface SpinHistoryItem { id: string; prize_type: string; prize_value: number; fulfillment_status: string; claim_status?: string; created_at: string; }
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
    "completed":             { label: "Credited",   cls: "bg-green-400/15 text-green-400" },
    "pending":               { label: "Pending",    cls: "bg-yellow-400/15 text-yellow-400" },
    "pending_claim":         { label: "Claim Now",  cls: "bg-nexus-400/20 text-nexus-300" },
    "pending_momo_setup":    { label: "Need MoMo",  cls: "bg-orange-400/15 text-orange-400" },
    "pending_admin_review":  { label: "In Review",  cls: "bg-blue-400/15 text-blue-400" },
    "approved":              { label: "Approved",   cls: "bg-emerald-400/15 text-emerald-400" },
    "failed":                { label: "Failed",     cls: "bg-red-400/15 text-red-400" },
    "na":                    { label: "No Prize",   cls: "bg-white/5 text-white/30" },
  };
  const key = status?.toLowerCase().replace(/ /g, "_");
  const s = map[key] ?? { label: status, cls: "bg-white/5 text-white/40" };
  return <span className={cn("text-[10px] font-bold px-2 py-0.5 rounded-full", s.cls)}>{s.label}</span>;
}

function needsClaim(item: SpinHistoryItem): boolean {
  const s = item.fulfillment_status?.toLowerCase();
  const c = item.claim_status?.toUpperCase();
  return (
    s === "pending_claim" ||
    s === "pending_momo_setup" ||
    c === "PENDING"
  ) && item.prize_type !== "try_again" && item.prize_type !== "pulse_points";
}

// ── Prize Claim Modal ──────────────────────────────────────────────────────
interface PrizeClaimModalProps {
  item: SpinHistoryItem | null;
  onClose: () => void;
  onSuccess: () => void;
}

function PrizeClaimModal({ item, onClose, onSuccess }: PrizeClaimModalProps) {
  const [bankAccNum, setBankAccNum] = useState("");
  const [bankAccName, setBankAccName] = useState("");
  const [bankName, setBankName] = useState("");
  const [momoNumber, setMomoNumber] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);

  if (!item) return null;

  const isCash = item.prize_type === "momo_cash";
  const isAirtime = item.prize_type === "airtime";
  const isData = item.prize_type === "data_bundle";
  const isAutoFulfill = isAirtime || isData;

  const valueNaira = item.prize_value ? (item.prize_value / 100).toLocaleString("en-NG", { style: "currency", currency: "NGN" }) : "";

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!item) return;
    if (isCash && (!bankAccNum.trim() || !bankAccName.trim() || !bankName)) {
      toast.error("Please fill in all bank details");
      return;
    }
    setSubmitting(true);
    try {
      const payload: Record<string, string> = {};
      if (isCash) {
        payload.bank_account_number = bankAccNum.trim();
        payload.bank_account_name = bankAccName.trim();
        payload.bank_name = bankName;
      }
      if (momoNumber) payload.momo_number = momoNumber.trim();
      await api.claimPrize(item.id, payload);
      setDone(true);
      toast.success(isCash ? "Bank details submitted! Our team will process your payment within 24h." : "Prize claimed! It will be credited within 10 minutes.");
      setTimeout(() => { onSuccess(); onClose(); }, 2000);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Claim failed. Try again.";
      toast.error(msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4" onClick={onClose}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />
      <motion.div
        initial={{ opacity: 0, y: 40, scale: 0.96 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 40, scale: 0.96 }}
        transition={{ type: "spring", damping: 24, stiffness: 300 }}
        onClick={e => e.stopPropagation()}
        className="relative z-10 w-full max-w-md bg-[#13172b] border border-white/10 rounded-2xl p-6 space-y-5"
      >
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className={cn("w-8 h-8 rounded-full flex items-center justify-center",
              isCash ? "bg-yellow-400/20" : "bg-nexus-500/20")}>
              {isCash ? <CreditCard size={16} className="text-yellow-400" /> : <Smartphone size={16} className="text-nexus-400" />}
            </div>
            <div>
              <p className="text-white font-bold text-sm">Claim Your Prize</p>
              <p className="text-white/40 text-xs">{prizeLabel(item)}</p>
            </div>
          </div>
          <button onClick={onClose} className="text-white/30 hover:text-white/60 transition-colors">
            <X size={18} />
          </button>
        </div>

        {done ? (
          <div className="text-center py-6 space-y-3">
            <CheckCircle className="w-12 h-12 text-green-400 mx-auto" />
            <p className="text-white font-bold">Prize Claimed!</p>
            <p className="text-white/40 text-sm">
              {isCash ? "Bank details received. Processing within 24 hours." : "Being credited to your phone now."}
            </p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Prize value banner */}
            <div className={cn("rounded-xl px-4 py-3 text-center",
              isCash ? "bg-yellow-400/10 border border-yellow-400/20" : "bg-nexus-500/10 border border-nexus-500/20")}>
              <p className="text-xs text-white/40 uppercase tracking-widest mb-0.5">Prize Value</p>
              <p className={cn("text-2xl font-bold font-display", isCash ? "text-yellow-400" : "text-nexus-300")}>
                {valueNaira}
              </p>
            </div>

            {/* AIRTIME / DATA — auto-fulfill, just confirm */}
            {isAutoFulfill && (
              <div className="space-y-3">
                <div className="bg-white/5 rounded-xl p-4 text-sm text-white/60 space-y-1">
                  <p className="text-white/80 font-semibold text-sm">
                    {isAirtime ? "Airtime" : "Data"} will be sent automatically
                  </p>
                  <p>It will be credited to your registered phone number within 5–10 minutes of confirmation.</p>
                </div>
                <button
                  type="submit"
                  disabled={submitting}
                  className="w-full py-3 rounded-xl font-bold text-white bg-nexus-600 hover:bg-nexus-500 disabled:opacity-50 flex items-center justify-center gap-2 transition-colors"
                >
                  {submitting ? <><Loader2 size={16} className="animate-spin" /> Processing…</> : <><CheckCircle size={16} /> Confirm & Receive</>}
                </button>
              </div>
            )}

            {/* CASH — bank details form */}
            {isCash && (
              <div className="space-y-3">
                <p className="text-xs text-white/40">
                  Enter the bank account you want the cash sent to. Our team will verify and process within 24 hours.
                </p>

                {/* Bank selector */}
                <div className="relative">
                  <label className="block text-xs text-white/40 mb-1">Bank Name *</label>
                  <div className="relative">
                    <select
                      value={bankName}
                      onChange={e => setBankName(e.target.value)}
                      required
                      className="w-full appearance-none bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-sm text-white focus:outline-none focus:border-nexus-500 focus:ring-1 focus:ring-nexus-500/40 pr-10"
                    >
                      <option value="" className="bg-[#13172b]">Select bank…</option>
                      {NIGERIAN_BANKS.map(b => (
                        <option key={b} value={b} className="bg-[#13172b]">{b}</option>
                      ))}
                    </select>
                    <ChevronDown size={14} className="absolute right-3 top-1/2 -translate-y-1/2 text-white/30 pointer-events-none" />
                  </div>
                </div>

                {/* Account number */}
                <div>
                  <label className="block text-xs text-white/40 mb-1">Account Number *</label>
                  <input
                    type="text"
                    maxLength={10}
                    value={bankAccNum}
                    onChange={e => setBankAccNum(e.target.value.replace(/\D/g, ""))}
                    placeholder="10-digit NUBAN"
                    required
                    className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-sm text-white placeholder-white/20 focus:outline-none focus:border-nexus-500 focus:ring-1 focus:ring-nexus-500/40"
                  />
                </div>

                {/* Account name */}
                <div>
                  <label className="block text-xs text-white/40 mb-1">Account Name *</label>
                  <input
                    type="text"
                    value={bankAccName}
                    onChange={e => setBankAccName(e.target.value)}
                    placeholder="As it appears on the account"
                    required
                    className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-sm text-white placeholder-white/20 focus:outline-none focus:border-nexus-500 focus:ring-1 focus:ring-nexus-500/40"
                  />
                </div>

                <button
                  type="submit"
                  disabled={submitting || !bankAccNum || !bankAccName || !bankName}
                  className="w-full py-3 rounded-xl font-bold text-white bg-yellow-500 hover:bg-yellow-400 disabled:opacity-40 flex items-center justify-center gap-2 transition-colors"
                >
                  {submitting ? <><Loader2 size={16} className="animate-spin" /> Submitting…</> : <><CreditCard size={16} /> Submit Bank Details</>}
                </button>
                <p className="text-center text-[10px] text-white/25">
                  Payment processed manually within 24 hours of submission.
                </p>
              </div>
            )}
          </form>
        )}
      </motion.div>
    </div>
  );
}

// ── Main Page ──────────────────────────────────────────────────────────────
export default function SpinPage() {
  const wheelRef = useRef<HTMLDivElement>(null);
  const [segments, setSegments] = useState<Segment[]>(FALLBACK_SEGMENTS);
  const [loadingSegments, setLoadingSegments] = useState(true);
  const [rotation, setRotation] = useState(0);
  const [spinning, setSpinning] = useState(false);
  const [spun, setSpun] = useState(false);
  const [outcome, setOutcome] = useState<SpinOutcome | null>(null);
  const [showResult, setShowResult] = useState(false);
  const [claimItem, setClaimItem] = useState<SpinHistoryItem | null>(null);

  // Wallet
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

  // Load wheel config
  useEffect(() => {
    api.getWheelConfig()
      .then((res: any) => {
        const raw: any[] = res?.slots ?? res?.prizes ?? res?.segments ?? [];
        const mapped = raw.map((p: any): Segment => ({
          label:       p.label ?? p.name ?? p.prize_name ?? "Prize",
          prize_type:  (p.prize_type ?? p.type ?? "try_again").toLowerCase(),
          base_value:  Number(p.base_value ?? p.prize_value ?? p.value ?? 0),
          probability: Number(p.probability ?? p.win_probability_weight ?? 0),
          color:       (p.prize_type === "try_again" || p.is_no_win)
                         ? (p.color ?? "#374151")
                         : (p.color ?? p.color_hex ?? "#5f72f9"),
          is_active:   true,
        }));
        if (mapped.length >= 2) setSegments(mapped);
      })
      .catch(() => {})
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

  const handleReset = () => { setSpun(false); setOutcome(null); setShowResult(false); };

  const isWin = outcome && outcome.spin_result?.prize_type !== "try_again";
  const prizeNeedsClaim = outcome?.spin_result && needsClaim({
    id: outcome.spin_result.id,
    prize_type: outcome.spin_result.prize_type,
    prize_value: outcome.spin_result.prize_value,
    fulfillment_status: outcome.spin_result.fulfillment_status,
    claim_status: outcome.spin_result.claim_status,
    created_at: "",
  });

  return (
    <AppShell>
      <Toaster position="top-center" toastOptions={{
        style: { background: "#1c2038", color: "#fff", border: "1px solid rgba(255,255,255,0.1)" },
      }} />

      {/* Prize Claim Modal */}
      <AnimatePresence>
        {claimItem && (
          <PrizeClaimModal
            item={claimItem}
            onClose={() => setClaimItem(null)}
            onSuccess={() => { mutateHistory(); mutateWallet(); }}
          />
        )}
      </AnimatePresence>

      <div className="max-w-5xl mx-auto px-3 md:px-6 py-4 md:py-6 pb-28 space-y-4 md:space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl md:text-2xl font-bold font-display text-white flex items-center gap-2">
              <Sparkles className="text-yellow-400" size={20} /> Spin & Win
            </h1>
            <p className="text-white/40 text-xs md:text-sm mt-0.5">Earn spins by recharging. Every ₦1,000 = 1 spin.</p>
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
            <span className={cn("text-2xl font-bold font-display", spinCredits > 0 ? "text-yellow-400" : "text-white/30")}>
              {effectiveWallet ? spinCredits : "—"}
            </span>
            {spinCredits === 0 && (
              <span className="text-[10px] text-white/30 border border-white/10 rounded-full px-2 py-0.5">
                Recharge to earn
              </span>
            )}
          </div>
        </div>

        {/* Tier progress */}
        <DailySpinProgress refreshKey={history.length} />

        {/* Wheel */}
        <div className="nexus-card p-6">
          <div className="relative mx-auto" style={{ width: 'min(280px, calc(100vw - 64px))', height: 'min(280px, calc(100vw - 64px))' }}>
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

                    {/* CTA varies by prize type */}
                    {prizeNeedsClaim && outcome.spin_result ? (
                      <button
                        onClick={() => setClaimItem({
                          id: outcome.spin_result!.id,
                          prize_type: outcome.spin_result!.prize_type,
                          prize_value: outcome.spin_result!.prize_value,
                          fulfillment_status: outcome.spin_result!.fulfillment_status,
                          claim_status: outcome.spin_result!.claim_status,
                          created_at: "",
                        })}
                        className="mt-2 w-full py-2.5 rounded-xl font-bold text-sm bg-yellow-500 hover:bg-yellow-400 text-white transition-colors flex items-center justify-center gap-1.5"
                      >
                        <Gift size={14} /> Claim Prize Now
                      </button>
                    ) : (
                      <p className="text-white/40 text-xs">
                        {outcome.spin_result?.prize_type === "pulse_points"
                          ? "Points added to your wallet instantly!"
                          : "Being credited to your phone within 5–10 minutes."}
                      </p>
                    )}
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
                <p className="text-center text-xs text-white/30">Every ₦1,000 recharge = 1 spin credit</p>
              )}
            </div>
          )}
        </div>

        {/* Prize list */}
        <div className="nexus-card p-4">
          <p className="text-white/40 text-xs font-semibold uppercase tracking-widest mb-3 flex items-center gap-1.5">
            <Info size={12} /> Possible Prizes
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            {segments.map((seg, i) => (
              <div key={i} className="flex items-center gap-2 text-xs text-white/60">
                <div className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ backgroundColor: seg.color }} />
                <span className={seg.prize_type === "try_again" ? "text-white/25 italic" : ""}>{seg.label}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Spin History — with Claim Now buttons */}
        {history.length > 0 && (
          <div className="space-y-2">
            <p className="text-white/40 text-xs font-semibold uppercase tracking-widest flex items-center gap-1.5">
              <History size={12} /> Recent Spins
            </p>
            <div className="space-y-2">
              {history.slice(0, 8).map((item) => (
                <motion.div
                  key={item.id}
                  initial={{ opacity: 0, x: -8 }}
                  animate={{ opacity: 1, x: 0 }}
                  className="nexus-card px-4 py-2.5 flex items-center justify-between gap-2"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Gift size={14} className={item.prize_type === "try_again" ? "text-white/20 flex-shrink-0" : "text-nexus-400 flex-shrink-0"} />
                    <div className="min-w-0">
                      <span className="text-white/70 text-sm truncate block">{prizeLabel(item)}</span>
                      <span className="text-white/25 text-[10px]">
                        {new Date(item.created_at).toLocaleDateString("en-NG", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    {needsClaim(item) ? (
                      <button
                        onClick={() => setClaimItem(item)}
                        className="text-[11px] font-bold px-3 py-1 rounded-full bg-nexus-500/20 text-nexus-300 hover:bg-nexus-500/30 border border-nexus-500/30 transition-colors whitespace-nowrap"
                      >
                        Claim →
                      </button>
                    ) : (
                      statusBadge(item.fulfillment_status)
                    )}
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
