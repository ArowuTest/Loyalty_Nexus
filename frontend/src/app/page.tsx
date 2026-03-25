"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import api from "@/lib/api";
import { useStore } from "@/store/useStore";
import toast, { Toaster } from "react-hot-toast";

type Step = "phone" | "otp";

const FEATURES = [
  { icon: "⚡", label: "Earn Pulse Points", sub: "Every ₦200 recharge" },
  { icon: "🎡", label: "Spin & Win", sub: "Daily prizes up to ₦50k" },
  { icon: "🧠", label: "Nexus AI Studio", sub: "17 free AI tools" },
  { icon: "🌍", label: "Regional Wars", sub: "Battle for your state" },
];

export default function LandingPage() {
  const router = useRouter();
  const { setToken, setUser } = useStore();
  const [step, setStep] = useState<Step>("phone");
  const [phone, setPhone] = useState("");
  const [otp, setOtp] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSendOTP = async () => {
    if (phone.replace(/\D/g, "").length < 11) {
      toast.error("Enter a valid 11-digit phone number");
      return;
    }
    setLoading(true);
    try {
      await api.sendOTP(phone.replace(/\D/g, ""));
      setStep("otp");
      toast.success("OTP sent! Check your SMS.");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Failed to send OTP");
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyOTP = async () => {
    if (otp.length < 4) { toast.error("Enter the 4-digit OTP"); return; }
    setLoading(true);
    try {
      const result = await api.verifyOTP(phone.replace(/\D/g, ""), otp) as { token: string; is_new_user: boolean };
      api.setToken(result.token);
      setToken(result.token);
      toast.success(result.is_new_user ? "Welcome to Loyalty Nexus! 🎉" : "Welcome back! 👋");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Invalid OTP");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[rgb(15_17_35)] flex flex-col items-center justify-center px-4 py-12">
      <Toaster position="top-center" toastOptions={{ style: { background: "#1c2038", color: "#fff" } }} />

      {/* Hero */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="text-center mb-10"
      >
        <div className="text-5xl mb-4">⚡</div>
        <h1 className="font-display text-4xl font-bold text-gradient mb-2">Loyalty Nexus</h1>
        <p className="text-[rgb(130_140_180)] text-lg">Recharge. Earn. Spin. Win.</p>
      </motion.div>

      {/* Feature pills */}
      <div className="flex flex-wrap gap-2 justify-center mb-10">
        {FEATURES.map((f) => (
          <div key={f.label} className="nexus-card flex items-center gap-2 px-3 py-2 text-sm">
            <span>{f.icon}</span>
            <div>
              <div className="text-white font-medium text-xs">{f.label}</div>
              <div className="text-[rgb(130_140_180)] text-xs">{f.sub}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Login card */}
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        className="nexus-card w-full max-w-sm p-6"
      >
        <h2 className="text-xl font-semibold text-white mb-1">
          {step === "phone" ? "Get started" : "Enter OTP"}
        </h2>
        <p className="text-[rgb(130_140_180)] text-sm mb-6">
          {step === "phone"
            ? "Enter your MTN number to continue"
            : `Enter the 4-digit code sent to ${phone.slice(0, 7)}****`}
        </p>

        <AnimatePresence mode="wait">
          {step === "phone" ? (
            <motion.div key="phone" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
              <input
                type="tel"
                placeholder="080X XXX XXXX"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                className="nexus-input mb-4"
                onKeyDown={(e) => e.key === "Enter" && handleSendOTP()}
                maxLength={14}
              />
              <button onClick={handleSendOTP} disabled={loading} className="nexus-btn-primary w-full">
                {loading ? "Sending…" : "Send OTP →"}
              </button>
            </motion.div>
          ) : (
            <motion.div key="otp" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
              <input
                type="number"
                placeholder="Enter 4-digit OTP"
                value={otp}
                onChange={(e) => setOtp(e.target.value.slice(0, 6))}
                className="nexus-input mb-4 text-center text-2xl tracking-widest"
                onKeyDown={(e) => e.key === "Enter" && handleVerifyOTP()}
              />
              <button onClick={handleVerifyOTP} disabled={loading} className="nexus-btn-primary w-full mb-3">
                {loading ? "Verifying…" : "Verify & Enter →"}
              </button>
              <button
                onClick={() => { setStep("phone"); setOtp(""); }}
                className="nexus-btn-outline w-full text-sm"
              >
                ← Change number
              </button>
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>

      <p className="text-[rgb(130_140_180)] text-xs mt-6 text-center max-w-xs">
        By continuing you agree to our Terms of Service. Loyalty Nexus is a licensed
        telecommunications loyalty program.
      </p>
    </div>
  );
}
