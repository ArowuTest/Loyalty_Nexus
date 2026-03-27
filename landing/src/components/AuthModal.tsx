import React, { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Phone, ArrowRight, Zap, Sparkles, Shield } from "lucide-react";

interface AuthModalProps {
  isOpen:    boolean;
  onClose:   () => void;
  onSuccess: () => void;
}

const COUNTRY_CODES = [
  { code: "+234", flag: "🇳🇬", label: "Nigeria" },
  { code: "+233", flag: "🇬🇭", label: "Ghana" },
  { code: "+254", flag: "🇰🇪", label: "Kenya" },
  { code: "+27",  flag: "🇿🇦", label: "S. Africa" },
];

type Step = "phone" | "otp" | "success";

export default function AuthModal({ isOpen, onClose, onSuccess }: AuthModalProps) {
  const [step, setStep] = useState<Step>("phone");
  const [phone, setPhone] = useState("");
  const [otp,   setOtp]   = useState(["","","","","",""]);
  const [cc,    setCc]    = useState("+234");
  const [loading,setLoading] = useState(false);

  const handlePhoneSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (phone.length < 7) return;
    setLoading(true);
    await new Promise(r => setTimeout(r, 900));
    setLoading(false);
    setStep("otp");
  };

  const handleOtpChange = (idx: number, val: string) => {
    if (!/^\d*$/.test(val)) return;
    const next = [...otp];
    next[idx] = val.slice(-1);
    setOtp(next);
    if (val && idx < 5) {
      document.getElementById(`otp-${idx + 1}`)?.focus();
    }
    if (next.every(d => d !== "")) {
      setTimeout(() => { setStep("success"); setTimeout(onSuccess, 900); }, 300);
    }
  };

  const handleOtpKeyDown = (idx: number, e: React.KeyboardEvent) => {
    if (e.key === "Backspace" && !otp[idx] && idx > 0) {
      document.getElementById(`otp-${idx - 1}`)?.focus();
    }
  };

  const reset = () => { setStep("phone"); setPhone(""); setOtp(["","","","","",""]); };
  const close  = () => { onClose(); setTimeout(reset, 400); };

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-[100] flex items-end sm:items-center justify-center p-0 sm:p-6">
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-black/75 backdrop-blur-sm"
            onClick={close}
          />

          {/* Card */}
          <motion.div
            initial={{ opacity: 0, y: 80, scale: 0.96 }}
            animate={{ opacity: 1, y: 0,  scale: 1 }}
            exit={{ opacity: 0, y: 40, scale: 0.97 }}
            transition={{ type: "spring", stiffness: 320, damping: 28 }}
            className="relative glass-strong rounded-t-3xl sm:rounded-3xl w-full max-w-sm p-7 border border-white/[0.10] overflow-hidden"
          >
            {/* Gold top strip */}
            <div className="absolute top-0 left-0 right-0 h-0.5 bg-gold opacity-50" />

            {/* Close */}
            <button onClick={close} className="absolute top-4 right-4 w-8 h-8 rounded-lg hover:bg-white/[0.08] flex items-center justify-center transition-colors">
              <X className="w-4 h-4 text-muted-foreground" />
            </button>

            {/* Logo */}
            <div className="flex items-center gap-2 mb-6">
              <div className="w-9 h-9 rounded-xl bg-gold flex items-center justify-center glow-gold">
                <Zap className="w-5 h-5 text-black" />
              </div>
              <span className="font-black text-base">
                <span className="text-gold">Loyalty</span>
                <span className="text-foreground"> Nexus</span>
              </span>
            </div>

            <AnimatePresence mode="wait">
              {/* PHONE STEP */}
              {step === "phone" && (
                <motion.div key="phone" initial={{ opacity:0,x:20 }} animate={{ opacity:1,x:0 }} exit={{ opacity:0,x:-20 }}>
                  <h2 className="text-2xl font-black text-foreground mb-1">Welcome Back 👋</h2>
                  <p className="text-sm text-muted-foreground mb-6 leading-relaxed">
                    Enter your MTN number to sign in or create your free account. We'll send a 6-digit OTP.
                  </p>
                  <form onSubmit={handlePhoneSubmit} className="space-y-4">
                    {/* Country + phone */}
                    <div className="flex gap-2">
                      <select
                        value={cc}
                        onChange={e => setCc(e.target.value)}
                        className="glass border border-white/[0.09] rounded-xl px-3 h-12 text-sm text-foreground bg-transparent focus:outline-none focus:border-primary/50 transition-all w-24 flex-shrink-0"
                      >
                        {COUNTRY_CODES.map(c => (
                          <option key={c.code} value={c.code} className="bg-surface-0">{c.flag} {c.code}</option>
                        ))}
                      </select>
                      <input
                        type="tel"
                        value={phone}
                        onChange={e => setPhone(e.target.value.replace(/\D/g,""))}
                        placeholder="801 234 5678"
                        required
                        className="flex-1 glass border border-white/[0.09] rounded-xl px-4 h-12 text-sm text-foreground placeholder:text-muted-foreground/40 focus:outline-none focus:border-primary/50 transition-all font-mono tracking-wider"
                      />
                    </div>
                    <button
                      type="submit"
                      disabled={loading || phone.length < 7}
                      className="btn-gold rounded-xl h-12 w-full text-[14px] font-black glow-gold inline-flex items-center justify-center gap-2 disabled:opacity-60 disabled:cursor-not-allowed"
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
                  {/* Bonus */}
                  <div className="mt-5 flex items-start gap-3 glass rounded-xl p-3.5 border border-primary/20">
                    <Sparkles className="w-4 h-4 text-primary flex-shrink-0 mt-0.5" />
                    <p className="text-[12px] text-muted-foreground leading-relaxed">
                      <strong className="text-primary font-bold">New user bonus:</strong>{" "}
                      100 Pulse Points + 1 free spin on your first sign-in!
                    </p>
                  </div>
                  <div className="mt-4 flex items-center gap-2 text-[11px] text-muted-foreground/40 justify-center">
                    <Shield className="w-3 h-3" />
                    <span>No passwords. OTP only. Your data is encrypted.</span>
                  </div>
                </motion.div>
              )}

              {/* OTP STEP */}
              {step === "otp" && (
                <motion.div key="otp" initial={{ opacity:0,x:20 }} animate={{ opacity:1,x:0 }} exit={{ opacity:0,x:-20 }}>
                  <h2 className="text-2xl font-black text-foreground mb-1">Check Your SMS 📱</h2>
                  <p className="text-sm text-muted-foreground mb-1">
                    We sent a 6-digit code to
                  </p>
                  <p className="text-sm font-bold text-primary mb-6 font-mono">{cc} {phone}</p>
                  <div className="flex gap-2 justify-between mb-6">
                    {otp.map((digit, i) => (
                      <input
                        key={i}
                        id={`otp-${i}`}
                        type="tel"
                        inputMode="numeric"
                        maxLength={1}
                        value={digit}
                        onChange={e => handleOtpChange(i, e.target.value)}
                        onKeyDown={e => handleOtpKeyDown(i, e)}
                        className={`w-11 h-14 text-center text-xl font-black rounded-xl border transition-all focus:outline-none font-mono
                          ${digit
                            ? "glass border-primary text-primary"
                            : "glass border-white/[0.10] text-foreground focus:border-primary/60"
                          }`}
                      />
                    ))}
                  </div>
                  <p className="text-center text-xs text-muted-foreground mb-5">
                    Didn't receive it?{" "}
                    <button onClick={() => {}} className="text-primary hover:underline font-semibold">Resend OTP</button>
                    {" "}(60s)
                  </p>
                  <button
                    onClick={() => setStep("phone")}
                    className="text-xs text-muted-foreground hover:text-foreground transition-colors w-full text-center"
                  >
                    ← Change number
                  </button>
                </motion.div>
              )}

              {/* SUCCESS STEP */}
              {step === "success" && (
                <motion.div key="success" initial={{ opacity:0,scale:0.9 }} animate={{ opacity:1,scale:1 }} className="text-center py-4">
                  <div className="text-6xl mb-4 animate-float-slow">🎉</div>
                  <h2 className="text-2xl font-black text-foreground mb-2">You're In!</h2>
                  <p className="text-sm text-muted-foreground mb-4">+100 Pulse Points added. Your first spin is ready!</p>
                  <div className="inline-flex items-center gap-1.5 text-xs text-primary font-bold animate-pulse">
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
