"use client";
import React, { Suspense, useEffect, useState, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import { CheckCircle2, Zap, ArrowRight, Gift, Loader2, AlertCircle, Wifi } from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

type RechargeStatus = "PENDING" | "PROCESSING" | "SUCCESS" | "FAILED" | "unknown";

interface StatusResponse {
  status: RechargeStatus;
  network: string;
  msisdn: string;
  amount_kobo: number;
  type: string;
}

function SuccessContent() {
  const params = useSearchParams();
  const ref = params.get("ref") ?? "";

  const [status, setStatus]       = useState<RechargeStatus>("PENDING");
  const [showToast, setShowToast] = useState(false);
  const [toastMsg, setToastMsg]   = useState("");
  const [details, setDetails]     = useState<StatusResponse | null>(null);
  const [pollCount, setPollCount] = useState(0);
  const MAX_POLLS = 24; // 2 min at 5s intervals

  const pollStatus = useCallback(async () => {
    if (!ref) return;
    try {
      const res = await fetch(`${API_BASE}/api/v1/recharge/status/${encodeURIComponent(ref)}`);
      if (!res.ok) return;
      const data: StatusResponse = await res.json();
      setDetails(data);
      setStatus(data.status);
      if (data.status === "SUCCESS") {
        const naira = Math.round(data.amount_kobo / 100);
        const type  = data.type === "DATA" ? "data bundle" : `₦${naira} airtime`;
        setToastMsg(`✅ ${type} delivered to ${data.msisdn}! Pulse Points credited.`);
        setShowToast(true);
        setTimeout(() => setShowToast(false), 6000);
      } else if (data.status === "FAILED") {
        setToastMsg("⚠️ Recharge could not be completed. Please contact support.");
        setShowToast(true);
      }
    } catch (_) { /* network error — keep polling */ }
  }, [ref]);

  useEffect(() => {
    if (!ref) return;
    // First poll immediately
    pollStatus();
    // Then poll every 5s until SUCCESS/FAILED or max reached
    const interval = setInterval(() => {
      setPollCount(c => {
        if (c >= MAX_POLLS || status === "SUCCESS" || status === "FAILED") {
          clearInterval(interval);
          return c;
        }
        pollStatus();
        return c + 1;
      });
    }, 5000);
    return () => clearInterval(interval);
  }, [ref, pollStatus]); // eslint-disable-line react-hooks/exhaustive-deps

  const isDone       = status === "SUCCESS" || status === "FAILED";
  const isProcessing = status === "PENDING" || status === "PROCESSING";

  return (
    <div className="min-h-screen bg-[#080808] flex items-center justify-center px-4">
      {/* Toast */}
      <AnimatePresence>
        {showToast && (
          <motion.div
            initial={{ opacity: 0, y: -60 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -60 }}
            className="fixed top-4 left-1/2 -translate-x-1/2 z-50 max-w-sm w-full mx-4"
          >
            <div className={`rounded-2xl px-5 py-4 flex items-center gap-3 shadow-2xl border
              ${status === "SUCCESS"
                ? "bg-green-950 border-green-500/40 text-green-300"
                : "bg-red-950 border-red-500/40 text-red-300"}`}
            >
              {status === "SUCCESS"
                ? <CheckCircle2 className="w-5 h-5 text-green-400 flex-shrink-0" />
                : <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0" />
              }
              <p className="text-[13px] font-semibold leading-snug">{toastMsg}</p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <motion.div
        initial={{ opacity: 0, scale: 0.92 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
        className="max-w-md w-full text-center"
      >
        {/* Icon */}
        <div className={`w-20 h-20 rounded-full flex items-center justify-center mx-auto mb-6 transition-all duration-500
          ${status === "SUCCESS" ? "bg-green-500/15 border border-green-500/30"
          : status === "FAILED"  ? "bg-red-500/15 border border-red-500/30"
          : "bg-amber-500/15 border border-amber-500/30"}`}
        >
          {status === "SUCCESS" ? <CheckCircle2 className="w-10 h-10 text-green-400" />
          : status === "FAILED"  ? <AlertCircle  className="w-10 h-10 text-red-400" />
          : <Loader2 className="w-10 h-10 text-amber-400 animate-spin" />}
        </div>

        <h1 className="text-3xl font-black text-white mb-2">
          {status === "SUCCESS" ? "Recharge Complete! 🎉"
           : status === "FAILED" ? "Recharge Failed"
           : "Payment Received!"}
        </h1>

        <p className="text-white/50 mb-6 text-[15px] leading-relaxed">
          {status === "SUCCESS"
            ? `Your ${details?.type === "DATA" ? "data bundle" : "airtime"} has been delivered to ${details?.msisdn}.`
            : status === "FAILED"
            ? "We could not complete your recharge. A refund will be processed within 24 hours."
            : "Your recharge is being processed. This usually takes just a few seconds…"}
        </p>

        {/* Live status indicator */}
        {isProcessing && (
          <div className="rounded-xl bg-amber-500/8 border border-amber-500/20 p-3 mb-4 flex items-center gap-2">
            <Wifi className="w-4 h-4 text-amber-400 animate-pulse flex-shrink-0" />
            <p className="text-[12px] text-amber-300/70">
              Checking recharge status{pollCount > 0 ? ` (${pollCount}/${MAX_POLLS})` : ""}…
            </p>
          </div>
        )}

        {ref && (
          <div className="rounded-xl bg-white/[0.04] border border-white/[0.07] p-4 mb-6">
            <p className="text-[11px] text-white/30 uppercase tracking-wider mb-1">Reference</p>
            <p className="font-mono text-[13px] text-white/70">{ref}</p>
          </div>
        )}

        {/* Double points reminder — only show when not failed */}
        {status !== "FAILED" && (
          <div className="rounded-xl bg-amber-500/8 border border-amber-500/20 p-4 mb-8 flex items-start gap-3 text-left">
            <Gift className="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" />
            <p className="text-[13px] text-amber-300/80 leading-relaxed">
              <strong className="text-amber-400">Double points incoming!</strong>{" "}
              {status === "SUCCESS"
                ? "Pulse Points have been credited to your account!"
                : "You\'ll earn Pulse Points once your recharge is confirmed."}
            </p>
          </div>
        )}

        <div className="flex flex-col sm:flex-row gap-3">
          <Link
            href="/recharge"
            className="flex-1 h-12 rounded-xl border border-white/[0.10] hover:border-white/20 flex items-center justify-center gap-2 text-[13px] font-bold text-white/60 hover:text-white transition-all"
          >
            <Zap className="w-4 h-4" />
            Recharge Again
          </Link>
          <Link
            href="/dashboard"
            className="flex-1 h-12 rounded-xl bg-amber-500 text-black flex items-center justify-center gap-2 text-[13px] font-black hover:bg-amber-400 transition-all"
          >
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
        <Loader2 className="w-8 h-8 text-amber-400 animate-spin" />
      </div>
    }>
      <SuccessContent />
    </Suspense>
  );
}
