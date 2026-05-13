"use client";
import React, { useState, useEffect, useCallback, useRef } from "react";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import {
  Zap, ChevronRight, Loader2, AlertCircle,
  Gift, Star, Shield, CheckCircle2, ArrowLeft,
  Wifi, Phone as PhoneIcon, ChevronDown, Info,
  Clock, X,
} from "lucide-react";
import NavBar from "@/components/landing/NavBar";
import AuthModal from "@/components/landing/AuthModal";
import { useStore } from "@/store/useStore";
import api from "@/lib/api";

const API = process.env.NEXT_PUBLIC_API_URL ?? "https://loyalty-nexus-api.onrender.com/api/v1";

// ── Types ─────────────────────────────────────────────────────────────────────

interface Network {
  code: string; name: string; logo: string;
  brand_color: string; is_active: boolean;
  airtime_enabled: boolean; data_enabled: boolean;
}

interface Bundle {
  id: string; name: string; price: number;
  data_size: string; validity: string; network: string;
}

interface RechargeStatusResult {
  status: "PENDING" | "PROCESSING" | "SUCCESS" | "FAILED";
  msisdn: string;
  network: string;
  amount_kobo: number;
  type: string;
  failure_reason?: string;
  points_earned?: number;
  draw_entries?: number;
  spin_eligible?: boolean;
}

interface RechargeSuccess {
  amount: number;           // naira
  phone: string;            // display format 080XXXXXXXX
  network: string;
  type: string;
  points: number;
  drawEntries: number;
  spinEligible: boolean;
  reference: string;
  pending: boolean;         // true while polling
  timedOut?: boolean;       // true after 18-min window
}

// ── Constants ─────────────────────────────────────────────────────────────────

const AIRTIME_PRESETS = [100, 200, 500, 1000, 2000, 5000];
const NETWORK_COLORS: Record<string, string> = {
  MTN: "#FFCC00", GLO: "#00A651", AIRTEL: "#FF0000", "9MOBILE": "#00A859",
};
const PREFIX_MAP: Record<string, string> = {
  "0803":"MTN","0806":"MTN","0703":"MTN","0706":"MTN","0813":"MTN",
  "0816":"MTN","0810":"MTN","0814":"MTN","0903":"MTN","0906":"MTN","0913":"MTN",
  "0805":"GLO","0807":"GLO","0705":"GLO","0815":"GLO","0905":"GLO","0811":"GLO",
  "0802":"AIRTEL","0808":"AIRTEL","0708":"AIRTEL","0812":"AIRTEL",
  "0701":"AIRTEL","0902":"AIRTEL","0907":"AIRTEL",
  "0809":"9MOBILE","0817":"9MOBILE","0818":"9MOBILE","0908":"9MOBILE","0909":"9MOBILE",
};

function detectNetworkFromPrefix(phone: string): string | null {
  const d = phone.replace(/\D/g, "");
  const n = d.startsWith("234") ? "0" + d.slice(3) : d;
  return PREFIX_MAP[n.slice(0, 4)] ?? null;
}

// ── Main Page ─────────────────────────────────────────────────────────────────

export default function RechargePage() {
  const { isAuthenticated, user, _hasHydrated } = useStore();
  const [authOpen, setAuthOpen] = useState(false);

  // ── Form state ────────────────────────────────────────────────────────────
  const [phone, setPhone]             = useState("");
  const [networks, setNetworks]       = useState<Network[]>([]);
  const [selectedNetwork, setNetwork] = useState<string>("");
  const [networkDetecting, setDetecting] = useState(false);
  const [networkHint, setNetworkHint] = useState<string>("");
  const [rechargeType, setType]       = useState<"airtime" | "data">("airtime");
  const [amountNaira, setAmount]      = useState<number | "">("");
  const [customAmount, setCustom]     = useState("");
  const [bundles, setBundles]         = useState<Bundle[]>([]);
  const [selectedBundle, setBundle]   = useState<Bundle | null>(null);
  const [loadingBundles, setLoadBundles] = useState(false);
  const [loadingNetworks, setLoadNets]   = useState(true);
  const [submitting, setSubmitting]      = useState(false);
  const [error, setError]                = useState("");
  const [email, setEmail]                = useState("");
  const detectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ── Payment result state (in-page banner) ────────────────────────────────
  const [rechargeSuccess, setRechargeSuccess] = useState<RechargeSuccess | null>(null);
  const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ── Pre-fill from auth user ───────────────────────────────────────────────
  useEffect(() => {
    if (_hasHydrated && isAuthenticated && user?.phone_number) {
      const raw = user.phone_number.replace(/^234/, "0").replace(/\D/g, "");
      setPhone(raw);
    }
    if (_hasHydrated && isAuthenticated && user?.email) setEmail(user.email);
  }, [_hasHydrated, isAuthenticated, user]);

  // ── Payment return: check for ?payment=success&reference=REF ─────────────
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const paymentParam = params.get("payment");
    const reference   = params.get("reference") || params.get("ref");

    if (paymentParam === "failed" && reference) {
      window.history.replaceState({}, "", window.location.pathname);
      setError("Payment was not completed or could not be verified. Please try again.");
      return;
    }

    if (paymentParam !== "success" || !reference) return;

    // Clean URL immediately
    window.history.replaceState({}, "", window.location.pathname);

    // Check if backend already embedded the result in the URL (fast VTPass path)
    const txnStatus   = params.get("txn_status");
    const preAmount   = params.get("amount");
    const prePoints   = params.get("points");
    const preEntries  = params.get("draw_entries");
    const preSpin     = params.get("spin_eligible");
    const preMsisdn   = params.get("msisdn");
    const preNetwork  = params.get("network");

    if (txnStatus === "SUCCESS" && preAmount) {
      // Fast path — show success immediately, no polling needed
      setRechargeSuccess({
        amount:      Number(preAmount),
        phone:       preMsisdn || "",
        network:     preNetwork || "",
        type:        "AIRTIME",
        points:      Number(prePoints || "0"),
        drawEntries: Number(preEntries || "0"),
        spinEligible: preSpin === "true",
        reference,
        pending:     false,
      });
      return;
    }

    // Slow path — VTPass still processing, show pending banner and poll
    setRechargeSuccess({
      amount: 0, phone: "", network: "", type: "AIRTIME",
      points: 0, drawEntries: 0, spinEligible: false,
      reference, pending: true,
    });

    // Adaptive polling: 2s (0-30), 5s (31-60), 10s (61-90), 20s (91-120) = ~18 min
    const pollTransaction = (attempt = 0, maxAttempts = 120) => {
      const delay = attempt === 0 ? 0
        : attempt < 30 ? 2000
        : attempt < 60 ? 5000
        : attempt < 90 ? 10000 : 20000;

      pollTimerRef.current = setTimeout(async () => {
        try {
          const res = await fetch(`${API}/recharge/status/${encodeURIComponent(reference)}`);
          if (!res.ok) throw new Error("not found");
          const data: RechargeStatusResult = await res.json();

          if (data.status === "SUCCESS") {
            setRechargeSuccess({
              amount:      data.amount_kobo / 100,
              phone:       data.msisdn,
              network:     data.network,
              type:        data.type,
              points:      data.points_earned || 0,
              drawEntries: data.draw_entries || 0,
              spinEligible: data.spin_eligible || false,
              reference,
              pending:     false,
            });
            return; // done
          }

          if (data.status === "FAILED") {
            setRechargeSuccess(prev => prev ? {
              ...prev, pending: false,
            } : null);
            setError(data.failure_reason || "Recharge could not be completed. A refund will be issued within 24 hours.");
            setRechargeSuccess(null);
            return;
          }

          // Still PENDING/PROCESSING
          if (attempt < maxAttempts - 1) {
            pollTransaction(attempt + 1, maxAttempts);
          } else {
            // 18-min window exhausted — leave pending banner, show timed-out state
            setRechargeSuccess(prev => prev ? { ...prev, timedOut: true } : null);
          }
        } catch {
          if (attempt < maxAttempts - 1) pollTransaction(attempt + 1, maxAttempts);
        }
      }, delay);
    };

    pollTransaction();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Cleanup polling on unmount
  useEffect(() => () => { if (pollTimerRef.current) clearTimeout(pollTimerRef.current); }, []);

  // ── Fetch networks ────────────────────────────────────────────────────────
  const fetchNetworks = useCallback(() => {
    setLoadNets(true); setError("");
    fetch(`${API}/recharge/networks`)
      .then(r => r.json())
      .then(d => {
        const nets: Network[] = d.networks ?? [];
        setNetworks(nets);
        setNetwork(prev => {
          if (prev) return prev;
          const mtn = nets.find(n => n.code === "MTN" && n.is_active);
          if (mtn) return mtn.code;
          const first = nets.find(n => n.is_active);
          return first ? first.code : prev;
        });
      })
      .catch(() => setError("Unable to load networks. Please try again."))
      .finally(() => setLoadNets(false));
  }, []);

  useEffect(() => {
    fetch(`${API.replace(/\/api\/v1$/, "")}/health`).catch(() => {});
    fetchNetworks();
  }, [fetchNetworks]);

  // ── Network auto-detect ───────────────────────────────────────────────────
  useEffect(() => {
    const digits = phone.replace(/\D/g, "");
    const normalized = digits.startsWith("234") ? "0" + digits.slice(3) : digits;
    if (normalized.length < 11) { setNetworkHint(""); return; }
    if (detectTimerRef.current) clearTimeout(detectTimerRef.current);
    detectTimerRef.current = setTimeout(async () => {
      setDetecting(true);
      let detected: string | null = null; let source = "";
      try {
        const r = await fetch(`${API}/recharge/networks/detect?phone=${normalized}`);
        if (r.ok) { const d = await r.json(); if (d.network) { detected = d.network; source = "Last used"; } }
      } catch { /**/ }
      if (!detected) { detected = detectNetworkFromPrefix(normalized); if (detected) source = "Auto-detected"; }
      if (detected) { if (!selectedNetwork) setNetwork(detected); setNetworkHint(`${source}: ${detected}`); }
      else setNetworkHint("");
      setDetecting(false);
    }, 500);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [phone]);

  // ── Fetch data bundles ────────────────────────────────────────────────────
  const fetchBundles = useCallback(async (code: string) => {
    setLoadBundles(true); setBundles([]); setBundle(null);
    try {
      const r = await fetch(`${API}/recharge/networks/${code}/bundles`);
      const d = await r.json(); setBundles(d.bundles ?? []);
    } catch { setError("Failed to load data bundles."); }
    finally { setLoadBundles(false); }
  }, []);

  useEffect(() => {
    if (rechargeType === "data" && selectedNetwork) fetchBundles(selectedNetwork);
    else { setBundles([]); setBundle(null); }
  }, [rechargeType, selectedNetwork, fetchBundles]);

  // ── Derived ───────────────────────────────────────────────────────────────
  const effectiveAmount: number =
    rechargeType === "data"
      ? (selectedBundle?.price ?? 0)
      : (amountNaira === "" ? (customAmount ? parseFloat(customAmount) : 0) : (amountNaira as number));

  const amountKobo = Math.round(effectiveAmount * 100);
  const digits = phone.replace(/\D/g, "");
  const normalized = digits.startsWith("234") ? "0" + digits.slice(3) : digits;
  const phoneValid = normalized.length === 11;
  const isValid = phoneValid && !!selectedNetwork && amountKobo >= 10000 &&
    (rechargeType === "airtime" || !!selectedBundle);
  const activeNetwork = networks.find(n => n.code === selectedNetwork);

  // ── Submit ────────────────────────────────────────────────────────────────
  const handleSubmit = async () => {
    if (!isValid || submitting) return;
    setError(""); setSubmitting(true);
    const msisdn = normalized.startsWith("234") ? normalized : "234" + normalized.replace(/^0/, "");
    try {
      // Use the api client so the JWT Bearer token is sent automatically when the
      // user is logged in — the backend links the recharge to their account for
      // reward attribution (points, spin credits, draw entries).
      const data = await api.initiateRecharge({
        msisdn, network: selectedNetwork!, recharge_type: rechargeType,
        amount_kobo: amountKobo, variation_code: selectedBundle?.id ?? "",
        email: email || undefined,
      });
      window.location.href = data.payment_url;
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to initiate recharge. Please try again.");
    } finally { setSubmitting(false); }
  };

  const handlePhone = (v: string) => {
    const d = v.replace(/\D/g, "").slice(0, 11);
    setPhone(d);
    if (d.length < 8) { setNetwork(""); setNetworkHint(""); }
  };

  // ── Render ────────────────────────────────────────────────────────────────
  return (
    <div className="min-h-screen bg-[#080808] text-white">
      <NavBar onLoginClick={() => setAuthOpen(true)} />
      <AuthModal open={authOpen} onClose={() => setAuthOpen(false)} />

      {/* Hero */}
      <div className="relative pt-28 pb-10 px-4 overflow-hidden">
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[700px] h-[400px] rounded-full bg-gold-500/5 blur-[120px]" />
        </div>
        <div className="max-w-xl mx-auto text-center relative z-10">
          <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }}>
            <div className="inline-flex items-center gap-2 bg-gold-500/10 border border-gold-500/20 rounded-full px-4 py-1.5 mb-5">
              <Zap className="w-3.5 h-3.5 text-gold-500" />
              <span className="text-[12px] font-bold text-gold-400 uppercase tracking-wider">No login required</span>
            </div>
            <h1 className="text-4xl font-black mb-3 leading-tight">
              Recharge & <span className="text-gold-500">Earn Double</span>
            </h1>
            <p className="text-white/50 text-base max-w-md mx-auto">
              Top up airtime or data — instantly. Earn 2× Pulse Points when MTN confirms.
            </p>
          </motion.div>
        </div>
      </div>

      <div className="max-w-xl mx-auto px-4 pb-20">

        {/* ── In-page result banners ── */}
        <AnimatePresence>
          {rechargeSuccess?.pending && (
            <motion.div key="processing"
              initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }}
              className="mb-5 rounded-xl border border-amber-500/25 bg-amber-500/8 p-4 flex items-start gap-3"
            >
              {rechargeSuccess.timedOut
                ? <Clock className="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" />
                : <Loader2 className="w-5 h-5 text-amber-400 animate-spin flex-shrink-0 mt-0.5" />
              }
              <div>
                <p className="text-[14px] font-bold text-amber-300 mb-0.5">
                  {rechargeSuccess.timedOut ? "Still Processing…" : "Recharge Processing…"}
                </p>
                <p className="text-[12px] text-amber-300/70 leading-relaxed">
                  {rechargeSuccess.timedOut
                    ? `Your recharge (${rechargeSuccess.reference}) is taking longer than usual. The airtime will be delivered — check your phone or history shortly.`
                    : "Your payment was received. Delivering your airtime/data now, usually under 60 seconds…"
                  }
                </p>
              </div>
            </motion.div>
          )}

          {rechargeSuccess && !rechargeSuccess.pending && (
            <motion.div key="success"
              initial={{ opacity: 0, scale: 0.96 }} animate={{ opacity: 1, scale: 1 }} exit={{ opacity: 0 }}
              className="mb-5 rounded-xl border border-green-500/30 bg-green-500/10 p-4 flex items-start gap-3"
            >
              <CheckCircle2 className="w-5 h-5 text-green-400 flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <p className="text-[14px] font-bold text-green-300 mb-0.5">
                  🎉 Recharge Successful!
                </p>
                <p className="text-[13px] text-green-300/80 leading-relaxed">
                  ₦{rechargeSuccess.amount.toLocaleString()} recharged to{" "}
                  <strong>{rechargeSuccess.phone}</strong>.
                  {(rechargeSuccess.points > 0 || rechargeSuccess.drawEntries > 0) && (
                    <> You earned{" "}
                      {rechargeSuccess.points > 0 && <strong>{rechargeSuccess.points} Pulse Point{rechargeSuccess.points !== 1 ? "s" : ""}</strong>}
                      {rechargeSuccess.points > 0 && rechargeSuccess.drawEntries > 0 && " and "}
                      {rechargeSuccess.drawEntries > 0 && <strong>{rechargeSuccess.drawEntries} draw entr{rechargeSuccess.drawEntries !== 1 ? "ies" : "y"}</strong>}
                      !
                    </>
                  )}
                  {rechargeSuccess.spinEligible && " 🎰 Spin wheel unlocked!"}
                </p>
              </div>
              <button onClick={() => setRechargeSuccess(null)} className="text-green-400/50 hover:text-green-300 transition-colors flex-shrink-0">
                <X className="w-4 h-4" />
              </button>
            </motion.div>
          )}
        </AnimatePresence>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.1 }}
          className="rounded-2xl border border-white/[0.08] bg-white/[0.03] p-6 space-y-5"
        >
          {/* ── 1. Phone Number ── */}
          <div>
            <label className="block text-[12px] font-bold text-white/50 uppercase tracking-wider mb-2">Phone Number</label>
            <div className="relative">
              <div className="absolute left-3.5 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                <PhoneIcon className="w-4 h-4 text-white/30" />
                <span className="text-[13px] text-white/30 font-mono">+234</span>
              </div>
              <input type="tel" value={phone.startsWith("0") ? phone.slice(1) : phone}
                onChange={e => handlePhone(e.target.value.startsWith("0") ? e.target.value : "0" + e.target.value)}
                placeholder="080 0000 0000" maxLength={10}
                className="w-full h-12 rounded-xl bg-white/[0.05] border border-white/[0.08] pl-24 pr-4 text-white placeholder:text-white/20 focus:outline-none focus:border-gold-500/50 font-mono text-[15px]"
              />
              {networkDetecting && (
                <div className="absolute right-3.5 top-1/2 -translate-y-1/2">
                  <Loader2 className="w-4 h-4 text-white/30 animate-spin" />
                </div>
              )}
            </div>
            {networkHint && !networkDetecting && (
              <p className="flex items-center gap-1.5 text-[11px] text-gold-400/80 mt-1.5">
                <Info className="w-3 h-3" /> {networkHint} — you can change below
              </p>
            )}
            {isAuthenticated && (
              <p className="text-[11px] text-white/30 mt-1.5">Pre-filled with your registered number</p>
            )}
          </div>

          {/* ── 2. Network Provider ── */}
          <div>
            <label className="block text-[12px] font-bold text-white/50 uppercase tracking-wider mb-2">Network Provider</label>
            {loadingNetworks ? (
              <div className="flex items-center gap-2 text-white/40 h-12">
                <Loader2 className="w-4 h-4 animate-spin" />
                <span className="text-[13px]">Loading networks…</span>
              </div>
            ) : error && networks.length === 0 ? (
              <div className="flex items-start gap-2.5 p-3 rounded-xl bg-red-500/10 border border-red-500/20">
                <AlertCircle className="w-4 h-4 text-red-400 flex-shrink-0 mt-0.5" />
                <div className="flex-1">
                  <p className="text-[13px] text-red-300">{error}</p>
                  <button onClick={fetchNetworks} className="mt-1 text-[12px] font-bold text-red-400 hover:text-red-300 underline underline-offset-2">
                    Tap to retry
                  </button>
                </div>
              </div>
            ) : (
              <div className="relative">
                <select value={selectedNetwork}
                  onChange={e => { setNetwork(e.target.value); setNetworkHint(""); setBundle(null); setError(""); }}
                  className="w-full h-12 rounded-xl bg-white/[0.05] border border-white/[0.08] px-4 pr-10 text-white focus:outline-none focus:border-gold-500/50 text-[14px] appearance-none cursor-pointer"
                  style={{ colorScheme: "dark" }}
                >
                  <option value="" className="bg-[#111]">Select network</option>
                  {networks.map(n => (
                    <option key={n.code} value={n.code} className="bg-[#111]">{n.name}</option>
                  ))}
                </select>
                <div className="absolute right-3.5 top-1/2 -translate-y-1/2 pointer-events-none">
                  {selectedNetwork
                    ? <span className="text-[16px]">{selectedNetwork==="MTN"?"🟡":selectedNetwork==="GLO"?"🟢":selectedNetwork==="AIRTEL"?"🔴":"🟦"}</span>
                    : <ChevronDown className="w-4 h-4 text-white/30" />}
                </div>
              </div>
            )}
          </div>

          {/* ── 3. Recharge Type ── */}
          <div>
            <label className="block text-[12px] font-bold text-white/50 uppercase tracking-wider mb-2">Recharge Type</label>
            <div className="flex rounded-xl overflow-hidden border border-white/[0.08] bg-white/[0.02] p-1 gap-1">
              {(["airtime","data"] as const).map(t => (
                <button key={t}
                  onClick={() => { setType(t); setBundle(null); setAmount(""); setCustom(""); }}
                  className={`flex-1 flex items-center justify-center gap-2 py-2.5 rounded-lg text-[13px] font-bold transition-all ${
                    rechargeType===t ? "bg-gold-500 text-black" : "text-white/40 hover:text-white"
                  }`}
                >
                  {t==="airtime" ? <PhoneIcon className="w-4 h-4" /> : <Wifi className="w-4 h-4" />}
                  {t==="airtime" ? "Airtime" : "Data"}
                </button>
              ))}
            </div>
          </div>

          {/* ── 4a. Airtime amounts ── */}
          <AnimatePresence mode="wait">
            {rechargeType==="airtime" && (
              <motion.div key="airtime" initial={{ opacity:0,y:6 }} animate={{ opacity:1,y:0 }} exit={{ opacity:0 }} transition={{ duration:0.2 }}>
                <label className="block text-[12px] font-bold text-white/50 uppercase tracking-wider mb-3">Select Amount</label>
                <div className="grid grid-cols-3 gap-2 mb-3">
                  {AIRTIME_PRESETS.map(p => (
                    <button key={p} onClick={() => { setAmount(p); setCustom(""); }}
                      className={`h-11 rounded-xl border text-[13px] font-bold transition-all ${
                        amountNaira===p
                          ? "border-gold-500 bg-gold-500/15 text-gold-400"
                          : "border-white/[0.08] bg-white/[0.02] text-white/60 hover:border-white/20 hover:text-white"
                      }`}
                    >
                      ₦{p>=1000 ? p/1000+"k" : p}
                    </button>
                  ))}
                </div>
                <div>
                  <label className="block text-[12px] font-bold text-white/40 uppercase tracking-wider mb-2">Or Enter Custom Amount (₦)</label>
                  <div className="relative">
                    <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-white/30 font-bold text-[14px]">₦</span>
                    <input type="number" value={customAmount}
                      onChange={e => { setCustom(e.target.value); setAmount(""); }}
                      placeholder="Enter amount" min={100} max={50000}
                      className="w-full h-12 rounded-xl bg-white/[0.05] border border-white/[0.08] pl-8 pr-4 text-white placeholder:text-white/20 focus:outline-none focus:border-gold-500/50 text-[14px]"
                    />
                  </div>
                </div>
              </motion.div>
            )}

            {/* ── 4b. Data bundles ── */}
            {rechargeType==="data" && (
              <motion.div key="data" initial={{ opacity:0,y:6 }} animate={{ opacity:1,y:0 }} exit={{ opacity:0 }} transition={{ duration:0.2 }}>
                <label className="block text-[12px] font-bold text-white/50 uppercase tracking-wider mb-3">Choose Data Bundle</label>
                {!selectedNetwork ? (
                  <p className="text-[13px] text-white/30 py-3">← Select a network above to see bundles</p>
                ) : loadingBundles ? (
                  <div className="flex items-center gap-2 text-white/40 py-4">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span className="text-[13px]">Loading bundles…</span>
                  </div>
                ) : bundles.length===0 ? (
                  <p className="text-[13px] text-white/40 py-3">No bundles available for this network right now.</p>
                ) : (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-64 overflow-y-auto pr-1">
                    {bundles.map(b => (
                      <button key={b.id} onClick={() => setBundle(b)}
                        className={`text-left p-3 rounded-xl border transition-all ${
                          selectedBundle?.id===b.id
                            ? "border-gold-500 bg-gold-500/10"
                            : "border-white/[0.08] bg-white/[0.02] hover:border-white/20"
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-[13px] font-bold text-white">{b.data_size || b.name.split(" ").slice(0,2).join(" ")}</span>
                          <span className="text-[13px] font-black text-gold-400">₦{b.price.toLocaleString()}</span>
                        </div>
                        <p className="text-[11px] text-white/40 mt-0.5 truncate">{b.name}</p>
                        {b.validity && <p className="text-[10px] text-white/25 mt-0.5">{b.validity}</p>}
                      </button>
                    ))}
                  </div>
                )}
              </motion.div>
            )}
          </AnimatePresence>

          {/* ── 5. Email (optional) ── */}
          <div>
            <label className="block text-[12px] font-bold text-white/50 uppercase tracking-wider mb-2">
              Email <span className="font-normal normal-case opacity-60">(optional — for receipt)</span>
            </label>
            <input type="email" value={email} onChange={e => setEmail(e.target.value)}
              placeholder="you@example.com"
              className="w-full h-12 rounded-xl bg-white/[0.05] border border-white/[0.08] px-4 text-white placeholder:text-white/20 focus:outline-none focus:border-gold-500/50 text-[14px]"
            />
          </div>

          {/* ── Summary ── */}
          <AnimatePresence>
            {isValid && (
              <motion.div initial={{ opacity:0,height:0 }} animate={{ opacity:1,height:"auto" }} exit={{ opacity:0,height:0 }}
                className="rounded-xl bg-white/[0.04] border border-white/[0.07] p-4 space-y-2"
              >
                {[
                  ["Network", activeNetwork?.name ?? selectedNetwork],
                  ["Phone", normalized],
                  ["Type", rechargeType==="airtime" ? "Airtime" : selectedBundle?.name ?? "Data"],
                ].map(([k,v]) => (
                  <div key={k} className="flex items-center justify-between text-[13px]">
                    <span className="text-white/50">{k}</span>
                    <span className="font-bold text-white" style={k==="Network" ? {color:NETWORK_COLORS[selectedNetwork]??"#fff"} : {}}>{v}</span>
                  </div>
                ))}
                <div className="flex items-center justify-between text-[13px] pt-2 border-t border-white/[0.06]">
                  <span className="text-white/50">Total</span>
                  <span className="text-[16px] font-black text-gold-400">₦{effectiveAmount.toLocaleString()}</span>
                </div>
                <div className="flex items-center gap-2 mt-1">
                  <Star className="w-3.5 h-3.5 text-gold-500" />
                  <span className="text-[11px] text-gold-400">
                    Earn ~{Math.floor(effectiveAmount/250)} Pulse Points + double when MTN confirms
                  </span>
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          {/* ── Error ── */}
          {error && networks.length > 0 && (
            <div className="flex items-center gap-2.5 p-3 rounded-xl bg-red-500/10 border border-red-500/20">
              <AlertCircle className="w-4 h-4 text-red-400 flex-shrink-0" />
              <p className="text-[13px] text-red-300">{error}</p>
            </div>
          )}

          {/* ── Submit ── */}
          <button onClick={handleSubmit} disabled={!isValid || submitting}
            className={`w-full h-14 rounded-xl font-black text-[15px] flex items-center justify-center gap-2.5 transition-all ${
              isValid && !submitting
                ? "bg-gold-500 text-black hover:bg-gold-400 shadow-lg shadow-gold-500/25"
                : "bg-white/[0.05] text-white/20 cursor-not-allowed"
            }`}
          >
            {submitting
              ? <><Loader2 className="w-5 h-5 animate-spin" />Processing…</>
              : <><Shield className="w-4 h-4" />{isValid ? `Pay ₦${effectiveAmount.toLocaleString()} with Paystack` : "Proceed to Payment"}<ChevronRight className="w-5 h-5" /></>
            }
          </button>

          {/* Trust signals */}
          <div className="flex items-center justify-center gap-5 text-[11px] text-white/25">
            <span className="flex items-center gap-1"><Shield className="w-3 h-3" /> Secured by Paystack</span>
            <span className="flex items-center gap-1"><Zap className="w-3 h-3" /> Instant delivery</span>
            <span className="flex items-center gap-1"><Star className="w-3 h-3" /> Double points</span>
          </div>

          {/* Auth nudge */}
          {_hasHydrated && !isAuthenticated && (
            <div className="text-center pt-2 border-t border-white/[0.05]">
              <p className="text-[12px] text-white/30">
                <button onClick={() => setAuthOpen(true)} className="text-gold-400 hover:text-gold-300 font-bold underline">Sign in</button>{" "}
                to link points to your account and track your rewards.
              </p>
            </div>
          )}
        </motion.div>

        {/* Double points callout */}
        <div className="mt-6 rounded-2xl border border-gold-500/20 bg-gold-500/5 p-4 flex items-start gap-3">
          <div className="w-10 h-10 rounded-xl bg-gold-500/15 flex items-center justify-center flex-shrink-0 mt-0.5">
            <Gift className="w-5 h-5 text-gold-400" />
          </div>
          <div>
            <p className="text-[14px] font-bold text-gold-400 mb-0.5">🎉 Double Points Offer</p>
            <p className="text-[13px] text-white/60 leading-relaxed">
              Earn Pulse Points from the platform <strong className="text-white/80">and</strong> again when MTN confirms your recharge. 2× rewards on every top-up.
            </p>
          </div>
        </div>

        <div className="text-center mt-8">
          <Link href="/" className="inline-flex items-center gap-1.5 text-[13px] text-white/30 hover:text-white/60 transition-colors">
            <ArrowLeft className="w-3.5 h-3.5" />
            Back to Home
          </Link>
        </div>
      </div>
    </div>
  );
}
