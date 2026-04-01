"use client";
import React, { useState, useRef, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { X, Phone, ArrowRight, Shield, Sparkles, Zap, ChevronDown } from "lucide-react";
import { useStore } from "@/store/useStore";
import { api } from "@/lib/api";

const COUNTRY_CODES = [
  { code: "+234", flag: "🇳🇬", name: "Nigeria" },
  { code: "+233", flag: "🇬🇭", name: "Ghana" },
  { code: "+254", flag: "🇰🇪", name: "Kenya" },
  { code: "+27",  flag: "🇿🇦", name: "South Africa" },
];

type Step = "phone" | "otp" | "success";

interface AuthModalProps {
  open: boolean;
  onClose: () => void;
}

export default function AuthModal({ open, onClose }: AuthModalProps) {
  const router                    = useRouter();
  const { setToken, setUser, setWallet } = useStore();
  const [step, setStep]           = useState<Step>("phone");
  const [phone, setPhone]         = useState("");
  const [cc, setCc]               = useState("+234");
  const [otp, setOtp]             = useState(["", "", "", "", "", ""]);
  const [loading, setLoading]     = useState(false);
  const [error, setError]         = useState<string | null>(null);
  const [resendTimer, setResendTimer] = useState(0);
  const otpRefs                   = useRef<(HTMLInputElement | null)[]>([]);
  const timerRef                  = useRef<NodeJS.Timeout | null>(null);

  // Reset on open
  useEffect(() => {
    if (open) {
      setStep("phone");
      setPhone("");
      setOtp(["", "", "", "", "", ""]);
      setError(null);
      setLoading(false);
      setResendTimer(0);
    }
  }, [open]);

  // Resend countdown
  useEffect(() => {
    if (resendTimer > 0) {
      timerRef.current = setTimeout(() => setResendTimer(t => t - 1), 1000);
    }
    return () => { if (timerRef.current) clearTimeout(timerRef.current); };
  }, [resendTimer]);

  // Close on Escape
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === "Escape") onClose(); };
    if (open) window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);

  const fullPhone = `${cc}${phone.replace(/^0/, "")}`;

  const handleSendOTP = async (e: React.FormEvent) => {
    e.preventDefault();
    if (phone.length < 7) return;
    setLoading(true);
    setError(null);
    try {
      await api.sendOTP(fullPhone);
      setStep("otp");
      setResendTimer(60);
      setTimeout(() => otpRefs.current[0]?.focus(), 150);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to send OTP. Please try again.";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const handleOtpChange = (index: number, value: string) => {
    const digit = value.replace(/\D/g, "").slice(-1);
    const next  = [...otp];
    next[index] = digit;
    setOtp(next);
    if (digit && index < 5) {
      otpRefs.current[index + 1]?.focus();
    }
    // Auto-submit when all 6 digits filled
    if (digit && index === 5 && next.every(d => d)) {
      handleVerifyOTP(next.join(""));
    }
  };

  const handleOtpKeyDown = (index: number, e: React.KeyboardEvent) => {
    if (e.key === "Backspace" && !otp[index] && index > 0) {
      otpRefs.current[index - 1]?.focus();
    }
  };

  const handleOtpPaste = (e: React.ClipboardEvent) => {
    const pasted = e.clipboardData.getData("text").replace(/\D/g, "").slice(0, 6);
    if (pasted.length === 6) {
      const digits = pasted.split("");
      setOtp(digits);
      handleVerifyOTP(pasted);
    }
  };

  const handleVerifyOTP = useCallback(async (code: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.verifyOTP(fullPhone, code) as unknown as { token: string; is_new_user: boolean };
      // Sync the API client token FIRST so subsequent requests are authenticated
      api.setToken(res.token);
      setToken(res.token);
      // Pre-fetch both profile AND wallet in parallel so the dashboard shows
      // real data immediately without a flash-to-zero on first load.
      const [profile, wallet] = await Promise.all([
        api.getProfile() as Promise<{ id: string; phone_number: string; tier: string; streak_count: number; is_active: boolean }>,
        api.getWallet() as Promise<{ pulse_points: number; spin_credits: number; lifetime_points: number }>,
      ]);
      setUser(profile);
      setWallet(wallet);
      setStep("success");
      setTimeout(() => {
        onClose();
        router.push("/dashboard");
      }, 1800);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Invalid OTP. Please try again.";
      setError(msg);
      setOtp(["", "", "", "", "", ""]);
      setTimeout(() => otpRefs.current[0]?.focus(), 50);
    } finally {
      setLoading(false);
    }
  }, [fullPhone, setToken, setUser, setWallet, onClose, router]);

  const handleResend = async () => {
    if (resendTimer > 0) return;
    setLoading(true);
    setError(null);
    try {
      await api.sendOTP(fullPhone);
      setResendTimer(60);
      setOtp(["", "", "", "", "", ""]);
      setTimeout(() => otpRefs.current[0]?.focus(), 50);
    } catch {
      setError("Failed to resend OTP. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-[100] flex items-end sm:items-center justify-center p-0 sm:p-4">
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-black/70 backdrop-blur-sm"
            onClick={onClose}
          />

          {/* Modal */}
          <motion.div
            initial={{ opacity: 0, y: 40, scale: 0.97 }}
            animate={{ opacity: 1, y: 0,  scale: 1 }}
            exit={{ opacity: 0,  y: 40, scale: 0.97 }}
            transition={{ type: "spring", damping: 28, stiffness: 380 }}
            className="relative w-full sm:max-w-md glass-strong rounded-t-3xl sm:rounded-3xl border border-white/[0.10] p-6 sm:p-8 overflow-hidden"
            onClick={e => e.stopPropagation()}
          >
            {/* Gold shimmer top bar */}
            <div className="absolute top-0 left-0 right-0 h-[2px] bg-gradient-to-r from-transparent via-gold-500 to-transparent" />

            {/* Mobile drag handle */}
            <div className="sm:hidden w-10 h-1 rounded-full bg-white/20 mx-auto mb-6" />

            {/* Close button */}
            {step !== "success" && (
              <button
                onClick={onClose}
                className="absolute top-4 right-4 w-8 h-8 rounded-xl glass border border-white/[0.08] flex items-center justify-center text-white/50 hover:text-white transition-all"
                aria-label="Close"
              >
                <X className="w-4 h-4" />
              </button>
            )}

            <AnimatePresence mode="wait">
              {/* PHONE STEP */}
              {step === "phone" && (
                <motion.div
                  key="phone"
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 20 }}
                >
                  <div className="flex items-center gap-3 mb-6">
                    <div className="w-10 h-10 rounded-xl bg-gold-500/15 border border-gold-500/25 flex items-center justify-center">
                      <Zap className="w-5 h-5 text-gold-500" />
                    </div>
                    <div>
                      <h2 className="text-xl font-black text-white leading-none">Welcome to Loyalty Nexus</h2>
                      <p className="text-[12px] text-white/40 mt-0.5">Enter your MTN number to continue</p>
                    </div>
                  </div>

                  {error && (
                    <div className="mb-4 px-4 py-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
                      {error}
                    </div>
                  )}

                  <form onSubmit={handleSendOTP} className="space-y-4">
                    {/* Country code + phone */}
                    <div className="flex gap-2">
                      <div className="relative">
                        <select
                          value={cc}
                          onChange={e => setCc(e.target.value)}
                          className="appearance-none h-12 pl-3 pr-8 rounded-xl text-sm font-bold text-white focus:outline-none border border-white/[0.10] focus:border-gold-500/40 transition-all"
                          style={{ background: "var(--surface-2)" }}
                        >
                          {COUNTRY_CODES.map(c => (
                            <option key={c.code} value={c.code}>
                              {c.flag} {c.code}
                            </option>
                          ))}
                        </select>
                        <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-white/40 pointer-events-none" />
                      </div>
                      <input
                        type="tel"
                        inputMode="numeric"
                        placeholder="080X XXX XXXX"
                        value={phone}
                        onChange={e => setPhone(e.target.value.replace(/\D/g, "").slice(0, 11))}
                        autoFocus
                        className="flex-1 h-12 rounded-xl px-4 text-sm text-white placeholder:text-white/25 focus:outline-none border border-white/[0.10] focus:border-gold-500/40 transition-all font-mono tracking-wider"
                        style={{ background: "var(--surface-2)" }}
                      />
                    </div>

                    <button
                      type="submit"
                      disabled={loading || phone.length < 7}
                      className="btn-gold rounded-xl h-12 w-full text-[14px] font-black glow-gold-sm inline-flex items-center justify-center gap-2 disabled:opacity-60 disabled:cursor-not-allowed"
                    >
                      {loading ? (
                        <>
                          <div className="w-4 h-4 border-2 border-black/30 border-t-black rounded-full animate-spin" />
                          Sending OTP…
                        </>
                      ) : (
                        <>
                          <Phone className="w-4 h-4" />
                          Send OTP
                          <ArrowRight className="w-4 h-4" />
                        </>
                      )}
                    </button>
                  </form>

                  {/* Value prop */}
                  <div className="mt-5 space-y-2.5">
                    <div className="flex items-start gap-3 glass rounded-xl p-3.5 border border-gold-500/15">
                      <Sparkles className="w-4 h-4 text-gold-500 flex-shrink-0 mt-0.5" />
                      <p className="text-[12px] text-white/50 leading-relaxed">
                        <strong className="text-gold-500 font-bold">No subscriptions needed</strong> — access 30+ AI tools completely free using Pulse Points you earn from recharges you already make.
                      </p>
                    </div>
                    <div className="flex items-start gap-3 glass rounded-xl p-3.5 border border-white/[0.07]">
                      <Zap className="w-4 h-4 text-primary flex-shrink-0 mt-0.5" />
                      <p className="text-[12px] text-white/50 leading-relaxed">
                        <strong className="text-white font-bold">Check your free spins</strong> — every qualifying MTN recharge earns a free wheel spin. Sign in to see yours.
                      </p>
                    </div>
                  </div>

                  <div className="mt-4 flex items-center gap-2 text-[11px] text-white/25 justify-center">
                    <Shield className="w-3 h-3" />
                    <span>No passwords. OTP only. Your data is encrypted.</span>
                  </div>
                </motion.div>
              )}

              {/* OTP STEP */}
              {step === "otp" && (
                <motion.div
                  key="otp"
                  initial={{ opacity: 0, x: 20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: -20 }}
                >
                  <h2 className="text-2xl font-black text-white mb-1">Check Your SMS 📱</h2>
                  <p className="text-sm text-white/40 mb-1">We sent a 6-digit code to</p>
                  <p className="text-sm font-bold text-gold-500 mb-6 font-mono">{cc} {phone}</p>

                  {error && (
                    <div className="mb-4 px-4 py-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
                      {error}
                    </div>
                  )}

                  <div
                    className="flex gap-2 justify-between mb-6"
                    onPaste={handleOtpPaste}
                  >
                    {otp.map((digit, i) => (
                      <input
                        key={i}
                        ref={el => { otpRefs.current[i] = el; }}
                        id={`otp-${i}`}
                        type="tel"
                        inputMode="numeric"
                        maxLength={1}
                        value={digit}
                        onChange={e => handleOtpChange(i, e.target.value)}
                        onKeyDown={e => handleOtpKeyDown(i, e)}
                        disabled={loading}
                        className={`w-11 h-14 text-center text-xl font-black rounded-xl border transition-all focus:outline-none font-mono
                          ${digit
                            ? "glass border-gold-500 text-gold-500"
                            : "glass border-white/[0.10] text-white focus:border-gold-500/60"
                          } disabled:opacity-50`}
                      />
                    ))}
                  </div>

                  {loading && (
                    <div className="flex items-center justify-center gap-2 text-sm text-white/50 mb-4">
                      <div className="w-4 h-4 border-2 border-white/20 border-t-gold-500 rounded-full animate-spin" />
                      Verifying…
                    </div>
                  )}

                  <p className="text-center text-xs text-white/40 mb-5">
                    Didn&apos;t receive it?{" "}
                    <button
                      onClick={handleResend}
                      disabled={resendTimer > 0 || loading}
                      className="text-gold-500 hover:underline font-semibold disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      {resendTimer > 0 ? `Resend in ${resendTimer}s` : "Resend OTP"}
                    </button>
                  </p>
                  <button
                    onClick={() => { setStep("phone"); setError(null); }}
                    className="text-xs text-white/30 hover:text-white/60 transition-colors w-full text-center"
                  >
                    ← Change number
                  </button>
                </motion.div>
              )}

              {/* SUCCESS STEP */}
              {step === "success" && (
                <motion.div
                  key="success"
                  initial={{ opacity: 0, scale: 0.9 }}
                  animate={{ opacity: 1, scale: 1 }}
                  className="text-center py-4"
                >
                  <div className="text-6xl mb-4 animate-float-slow">🎉</div>
                  <h2 className="text-2xl font-black text-white mb-2">You&apos;re In!</h2>
                  <p className="text-sm text-white/50 mb-4">
                    +100 Pulse Points added. Your first spin is ready!
                  </p>
                  <div className="inline-flex items-center gap-1.5 text-xs text-gold-500 font-bold animate-pulse">
                    <Zap className="w-3.5 h-3.5" />
                    Loading your dashboard…
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}
