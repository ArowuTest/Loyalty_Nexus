"use client";

import { useState, useRef, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import toast, { Toaster } from "react-hot-toast";
import Link from "next/link";
import { Zap, Trophy, Info } from "lucide-react";

const WHEEL_COLORS = [
  "#5f72f9", "#8b5cf6", "#f9c74f", "#06b6d4",
  "#10b981", "#f43f5e", "#fb923c", "#a78bfa",
];

const DEFAULT_SEGMENTS = [
  { label: "₦500 Airtime", weight: 20 },
  { label: "Try Again", weight: 30 },
  { label: "100 Points", weight: 20 },
  { label: "₦1k Data", weight: 10 },
  { label: "50 Points", weight: 25 },
  { label: "₦2k Cash", weight: 5 },
  { label: "Try Again", weight: 25 },
  { label: "₦5k Cash", weight: 2 },
];

interface SpinOutcome {
  prize_type: string;
  prize_label: string;
  prize_value: number;
  message: string;
  is_win: boolean;
}

export default function SpinPage() {
  const [spinning, setSpinning] = useState(false);
  const [rotation, setRotation] = useState(0);
  const [outcome, setOutcome] = useState<SpinOutcome | null>(null);
  const [spinCount, setSpinCount] = useState(0);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const baseRotation = useRef(0);

  const drawWheel = useCallback((canvas: HTMLCanvasElement, segments: typeof DEFAULT_SEGMENTS, rot: number) => {
    const ctx = canvas.getContext("2d")!;
    const size = canvas.width;
    const cx = size / 2;
    const cy = size / 2;
    const r = cx - 8;
    const segAngle = (2 * Math.PI) / segments.length;

    ctx.clearRect(0, 0, size, size);

    segments.forEach((seg, i) => {
      const start = rot + i * segAngle;
      const end = start + segAngle;

      ctx.beginPath();
      ctx.moveTo(cx, cy);
      ctx.arc(cx, cy, r, start, end);
      ctx.closePath();
      ctx.fillStyle = WHEEL_COLORS[i % WHEEL_COLORS.length];
      ctx.fill();
      ctx.strokeStyle = "rgba(15,17,35,0.6)";
      ctx.lineWidth = 2;
      ctx.stroke();

      // Label
      ctx.save();
      ctx.translate(cx, cy);
      ctx.rotate(start + segAngle / 2);
      ctx.textAlign = "right";
      ctx.fillStyle = "white";
      ctx.font = "bold 11px Inter";
      ctx.shadowColor = "rgba(0,0,0,0.6)";
      ctx.shadowBlur = 4;
      ctx.fillText(seg.label, r - 10, 4);
      ctx.restore();
    });

    // Center circle
    ctx.beginPath();
    ctx.arc(cx, cy, 28, 0, Math.PI * 2);
    ctx.fillStyle = "#0f1123";
    ctx.fill();
    ctx.strokeStyle = "#5f72f9";
    ctx.lineWidth = 3;
    ctx.stroke();
    ctx.fillStyle = "white";
    ctx.font = "bold 13px Inter";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText("⚡", cx, cy);
  }, []);

  const handleSpin = async () => {
    if (spinning) return;
    setSpinning(true);
    setOutcome(null);

    try {
      const result = await api.playSpin() as SpinOutcome;

      // Animate wheel regardless of outcome
      const spins = 6 + Math.random() * 4;
      const targetRotation = baseRotation.current + spins * 360 + Math.random() * 360;
      setRotation(targetRotation);
      baseRotation.current = targetRotation % 360;

      // Reveal outcome after animation
      setTimeout(() => {
        setOutcome(result);
        setSpinCount(c => c + 1);
        setSpinning(false);
      }, 3500);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Spin failed";
      toast.error(msg);
      setSpinning(false);
    }
  };

  return (
    <AppShell>
      <Toaster position="top-center" toastOptions={{ style: { background: "#1c2038", color: "#fff" } }} />

      <div className="max-w-md mx-auto px-4 py-6 space-y-5">
        <div className="text-center">
          <h1 className="text-2xl font-bold font-display text-white flex items-center justify-center gap-2">
            <Zap className="text-nexus-400" size={24} />
            Spin & Win
          </h1>
          <p className="text-[rgb(130_140_180)] text-sm mt-1">Use spin credits to win prizes</p>
        </div>

        {/* Wheel container */}
        <div className="relative flex justify-center">
          {/* Pointer */}
          <div className="absolute top-0 left-1/2 -translate-x-1/2 z-10" style={{ marginTop: "-4px" }}>
            <div className="w-0 h-0 border-l-[12px] border-r-[12px] border-t-[20px] border-transparent border-t-gold-500" />
          </div>

          <motion.div
            animate={{ rotate: rotation }}
            transition={{ duration: 3.2, ease: [0.32, 1.0, 0.75, 1.0] }}
            className="relative"
          >
            {/* Simplified CSS wheel fallback */}
            <div className="w-72 h-72 rounded-full relative overflow-hidden shadow-2xl shadow-nexus-900"
              style={{
                background: "conic-gradient(" +
                  WHEEL_COLORS.map((c, i) => `${c} ${i * (100/8)}% ${(i+1) * (100/8)}%`).join(", ") +
                ")",
              }}
            >
              {DEFAULT_SEGMENTS.map((seg, i) => (
                <div
                  key={i}
                  className="absolute text-white font-bold text-xs"
                  style={{
                    left: "50%",
                    top: "50%",
                    transformOrigin: "0 0",
                    transform: `rotate(${i * (360/8) + (360/8/2)}deg) translateY(-35%) translateX(-50%)`,
                    width: "80px",
                    textAlign: "center",
                    textShadow: "0 1px 3px rgba(0,0,0,0.8)",
                    whiteSpace: "nowrap",
                  }}
                >
                  {seg.label}
                </div>
              ))}
            </div>
            {/* Center pin */}
            <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-14 h-14 rounded-full bg-[rgb(15_17_35)] border-2 border-nexus-500 flex items-center justify-center shadow-lg">
              <span className="text-xl">⚡</span>
            </div>
          </motion.div>
        </div>

        {/* Outcome reveal */}
        <AnimatePresence>
          {outcome && (
            <motion.div
              initial={{ opacity: 0, scale: 0.9, y: 10 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0 }}
              className={`nexus-card p-5 text-center ${outcome.is_win ? "border-gold-500/40" : ""}`}
            >
              <div className="text-4xl mb-2">{outcome.is_win ? "🎉" : "😔"}</div>
              <h3 className="text-xl font-bold text-white mb-1">{outcome.prize_label}</h3>
              <p className="text-[rgb(130_140_180)] text-sm">{outcome.message}</p>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Spin button */}
        <button
          onClick={handleSpin}
          disabled={spinning}
          className="nexus-btn-primary w-full text-lg py-4 flex items-center justify-center gap-2"
        >
          {spinning ? (
            <>
              <motion.div animate={{ rotate: 360 }} transition={{ duration: 1, repeat: Infinity, ease: "linear" }}>
                <Zap size={20} />
              </motion.div>
              Spinning…
            </>
          ) : (
            <>
              <Zap size={20} />
              Spin ({spinCount === 0 ? "Free Daily Spin!" : "Use 1 Credit"})
            </>
          )}
        </button>

        {/* Info */}
        <div className="nexus-card p-4 flex gap-3">
          <Info size={16} className="text-nexus-400 flex-shrink-0 mt-0.5" />
          <div className="text-sm text-[rgb(130_140_180)]">
            Recharge ₦200+ to earn Spin Credits. Tier bonuses give you extra spins daily.
            <Link href="/prizes" className="text-nexus-400 ml-1 hover:underline flex items-center gap-1 mt-1 inline-flex">
              <Trophy size={12} /> View prize history
            </Link>
          </div>
        </div>
      </div>
    </AppShell>
  );
}
