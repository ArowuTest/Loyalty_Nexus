"use client";
import React, { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { motion } from "framer-motion";
import { CheckCircle2, Zap, ArrowRight, Gift } from "lucide-react";

function SuccessContent() {
  const params = useSearchParams();
  const ref = params.get("ref") ?? "";

  return (
    <div className="min-h-screen bg-[#080808] flex items-center justify-center px-4">
      <motion.div
        initial={{ opacity: 0, scale: 0.92 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
        className="max-w-md w-full text-center"
      >
        {/* Icon */}
        <div className="w-20 h-20 rounded-full bg-green-500/15 border border-green-500/30 flex items-center justify-center mx-auto mb-6">
          <CheckCircle2 className="w-10 h-10 text-green-400" />
        </div>

        <h1 className="text-3xl font-black text-white mb-2">Payment Received!</h1>
        <p className="text-white/50 mb-6 text-[15px] leading-relaxed">
          Your recharge is being processed. You&apos;ll receive your airtime/data within
          seconds, and your Pulse Points will be credited automatically.
        </p>

        {ref && (
          <div className="rounded-xl bg-white/[0.04] border border-white/[0.07] p-4 mb-6">
            <p className="text-[11px] text-white/30 uppercase tracking-wider mb-1">Reference</p>
            <p className="font-mono text-[13px] text-white/70">{ref}</p>
          </div>
        )}

        {/* Double points reminder */}
        <div className="rounded-xl bg-gold-500/8 border border-gold-500/20 p-4 mb-8 flex items-start gap-3 text-left">
          <Gift className="w-5 h-5 text-gold-400 flex-shrink-0 mt-0.5" />
          <p className="text-[13px] text-gold-300/80 leading-relaxed">
            <strong className="text-gold-400">Double points incoming!</strong> You&apos;ve already earned
            Pulse Points from this recharge. You&apos;ll earn more when MTN confirms delivery.
          </p>
        </div>

        <div className="flex flex-col sm:flex-row gap-3">
          <Link href="/recharge" className="flex-1 h-12 rounded-xl border border-white/[0.10] hover:border-white/20 flex items-center justify-center gap-2 text-[13px] font-bold text-white/60 hover:text-white transition-all">
            <Zap className="w-4 h-4" />
            Recharge Again
          </Link>
          <Link href="/dashboard" className="flex-1 h-12 rounded-xl bg-gold-500 text-black flex items-center justify-center gap-2 text-[13px] font-black hover:bg-gold-400 transition-all">
            View My Points
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </motion.div>
    </div>
  );
}

export default function RechargeSuccess() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-[#080808] flex items-center justify-center">
        <div className="w-8 h-8 rounded-full border-2 border-gold-500/30 border-t-gold-500 animate-spin" />
      </div>
    }>
      <SuccessContent />
    </Suspense>
  );
}
