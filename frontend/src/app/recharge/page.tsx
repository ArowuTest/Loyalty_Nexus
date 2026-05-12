"use client";
import React, { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import {
  Zap, Smartphone, ChevronRight, Loader2, AlertCircle,
  Gift, Star, TrendingUp, Shield, CheckCircle2, ArrowLeft,
  Wifi, Phone as PhoneIcon,
} from "lucide-react";
import NavBar from "@/components/landing/NavBar";
import AuthModal from "@/components/landing/AuthModal";
import { useStore } from "@/store/useStore";

const API = process.env.NEXT_PUBLIC_API_URL ?? "https://loyalty-nexus-api.onrender.com";

// ── Types ────────────────────────────────────────────────────────────────────

interface Network {
  code: string;
  name: string;
  logo: string;
  brand_color: string;
  is_active: boolean;
  airtime_enabled: boolean;
  data_enabled: boolean;
}

interface Bundle {
  id: string;          // VTPass variation_code e.g. "mtn-10mb-100"
  name: string;
  price: number;       // naira
  data_size: string;
  network: string;
}

// ── Preset amounts ────────────────────────────────────────────────────────────

const AIRTIME_PRESETS = [100, 200, 500, 1000, 2000, 5000];

// ── Network logo fallback ─────────────────────────────────────────────────────

const NETWORK_COLORS: Record<string, string> = {
  MTN:     "#FFCC00",
  GLO:     "#00A651",
  AIRTEL:  "#FF0000",
  "9MOBILE": "#00A859",
};

const NETWORK_EMOJIS: Record<string, string> = {
  MTN:     "🟡",
  GLO:     "🟢",
  AIRTEL:  "🔴",
  "9MOBILE": "🟢",
};

// ── Main Page ─────────────────────────────────────────────────────────────────

export default function RechargePage() {
  const { isAuthenticated, user, _hasHydrated } = useStore();
  const [authOpen, setAuthOpen] = useState(false);

  // ── State ──────────────────────────────────────────────────────────────────
  const [networks, setNetworks]         = useState<Network[]>([]);
  const [selectedNetwork, setNetwork]   = useState<Network | null>(null);
  const [rechargeType, setType]         = useState<"airtime" | "data">("airtime");
  const [phone, setPhone]               = useState("");
  const [amountNaira, setAmount]        = useState<number | "">("");
  const [customAmount, setCustom]       = useState("");
  const [bundles, setBundles]           = useState<Bundle[]>([]);
  const [selectedBundle, setBundle]     = useState<Bundle | null>(null);
  const [loadingBundles, setLoadBundles] = useState(false);
  const [loadingNetworks, setLoadNets]  = useState(true);
  const [submitting, setSubmitting]     = useState(false);
  const [error, setError]               = useState("");
  const [email, setEmail]               = useState("");

  // Pre-fill phone from authenticated user
  useEffect(() => {
    if (_hasHydrated && isAuthenticated && user?.phone_number) {
      const raw = user.phone_number.replace(/^234/, "0");
      setPhone(raw.replace(/\D/g, ""));
    }
  }, [_hasHydrated, isAuthenticated, user?.phone_number]);

  // Pre-fill email from authenticated user
  useEffect(() => {
    if (_hasHydrated && isAuthenticated && user?.email) {
      setEmail(user.email);
    }
  }, [_hasHydrated, isAuthenticated, user?.email]);

  // ── Fetch active networks ──────────────────────────────────────────────────
  useEffect(() => {
    setLoadNets(true);
    fetch(`${API}/api/v1/recharge/networks`)
      .then(r => r.json())
      .then(d => {
        const nets: Network[] = d.networks ?? [];
        setNetworks(nets);
        if (nets.length === 1) setNetwork(nets[0]); // auto-select if only MTN
      })
      .catch(() => setError("Unable to load networks. Please try again."))
      .finally(() => setLoadNets(false));
  }, []);

  // ── Fetch bundles when network + data tab selected ─────────────────────────
  const fetchBundles = useCallback(async (code: string) => {
    setLoadBundles(true);
    setBundles([]);
    setBundle(null);
    try {
      const r = await fetch(`${API}/api/v1/recharge/networks/${code}/bundles`);
      const d = await r.json();
      setBundles(d.bundles ?? []);
    } catch {
      setError("Failed to load data bundles.");
    } finally {
      setLoadBundles(false);
    }
  }, []);

  useEffect(() => {
    if (rechargeType === "data" && selectedNetwork) {
      fetchBundles(selectedNetwork.code);
    }
  }, [rechargeType, selectedNetwork, fetchBundles]);

  // ── Derived state ──────────────────────────────────────────────────────────
  const effectiveAmount: number =
    rechargeType === "data"
      ? (selectedBundle?.price ?? 0)
      : (amountNaira === "" ? (customAmount ? parseFloat(customAmount) : 0) : (amountNaira as number));

  const amountKobo = Math.round(effectiveAmount * 100);
  const isValid =
    !!selectedNetwork &&
    phone.replace(/\D/g, "").length >= 11 &&
    amountKobo >= 10000 &&
    (rechargeType === "airtime" || !!selectedBundle);

  // ── Submit ─────────────────────────────────────────────────────────────────
  const handleSubmit = async () => {
    if (!isValid || submitting) return;
    setError("");
    setSubmitting(true);

    const msisdn = phone.replace(/\D/g, "").startsWith("234")
      ? phone.replace(/\D/g, "")
      : "234" + phone.replace(/\D/g, "").replace(/^0/, "");

    const userID = (isAuthenticated && user?.id) ? user.id : undefined;

    try {
      const res = await fetch(`${API}/api/v1/recharge/initiate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          msisdn,
          network:        selectedNetwork!.code,
          recharge_type:  rechargeType,
          amount_kobo:    amountKobo,
          variation_code: selectedBundle?.id ?? "",
          email:          email || "guest@loyaltynexus.ng",
          user_id:        userID,
        }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Something went wrong");
      // Redirect to Paystack
      window.location.href = data.payment_url;
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to initiate recharge. Please try again.");
    } finally {
      setSubmitting(false);
    }
  };

  // ── Phone formatter ────────────────────────────────────────────────────────
  const handlePhone = (v: string) => {
    const digits = v.replace(/\D/g, "").slice(0, 11);
    setPhone(digits);
  };

  return (
    <div className="min-h-screen bg-[#080808] text-white">
      <NavBar onLoginClick={() => setAuthOpen(true)} />
      <AuthModal open={authOpen} onClose={() => setAuthOpen(false)} />

      {/* ── Hero ── */}
      <div className="relative pt-28 pb-12 px-4 overflow-hidden">
        {/* Background glow */}
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[700px] h-[400px] rounded-full bg-gold-500/5 blur-[120px]" />
        </div>

        <div className="max-w-2xl mx-auto text-center relative z-10">
          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            {/* Badge */}
            <div className="inline-flex items-center gap-2 bg-gold-500/10 border border-gold-500/20 rounded-full px-4 py-1.5 mb-5">
              <Zap className="w-3.5 h-3.5 text-gold-500" />
              <span className="text-[12px] font-bold text-gold-400 uppercase tracking-wider">No login required</span>
            </div>

            <h1 className="text-4xl md:text-5xl font-black mb-3 leading-tight">
              Recharge &{" "}
              <span className="text-gold-500">Earn Double</span>
              <br />
              <span className="text-white/60 text-3xl md:text-4xl">Pulse Points</span>
            </h1>
            <p className="text-white/50 text-lg max-w-lg mx-auto">
              Top up airtime or data for any Nigerian number — instantly.
              Recharge on Loyalty Nexus and earn 2× points when MTN confirms.
            </p>
          </motion.div>
        </div>
      </div>

      {/* ── Double points callout ── */}
      <div className="max-w-2xl mx-auto px-4 mb-6">
        <div className="rounded-2xl border border-gold-500/20 bg-gold-500/5 p-4 flex items-start gap-3">
          <div className="w-10 h-10 rounded-xl bg-gold-500/15 flex items-center justify-center flex-shrink-0 mt-0.5">
            <Gift className="w-5 h-5 text-gold-400" />
          </div>
          <div>
            <p className="text-[14px] font-bold text-gold-400 mb-0.5">🎉 Double Points Offer</p>
            <p className="text-[13px] text-white/60 leading-relaxed">
              When you recharge here, you earn Pulse Points from the platform{" "}
              <strong className="text-white/80">and</strong> again when MTN confirms your recharge.
              That&apos;s 2× the rewards on every single top-up.
            </p>
          </div>
        </div>
      </div>

      {/* ── Main card ── */}
      <div className="max-w-2xl mx-auto px-4 pb-20">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4, delay: 0.1 }}
          className="rounded-2xl border border-white/[0.08] bg-white/[0.03] p-6 md:p-8 space-y-6"
        >

          {/* ── Network selector ── */}
          <div>
            <label className="block text-[13px] font-bold text-white/50 uppercase tracking-wider mb-3">
              Select Network
            </label>
            {loadingNetworks ? (
              <div className="flex items-center gap-3 text-white/40">
                <Loader2 className="w-4 h-4 animate-spin" />
                <span className="text-[13px]">Loading networks…</span>
              </div>
            ) : (
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                {networks.map(net => (
                  <button
                    key={net.code}
                    onClick={() => { setNetwork(net); setBundle(null); setError(""); }}
                    className={`relative p-3 rounded-xl border-2 transition-all flex flex-col items-center gap-1.5 ${
                      selectedNetwork?.code === net.code
                        ? "border-gold-500 bg-gold-500/10"
                        : "border-white/[0.08] hover:border-white/20 bg-white/[0.02]"
                    }`}
                  >
                    <span className="text-2xl">{NETWORK_EMOJIS[net.code] ?? "📱"}</span>
                    <span className="text-[12px] font-bold" style={{ color: NETWORK_COLORS[net.code] ?? "#fff" }}>
                      {net.code}
                    </span>
                    {selectedNetwork?.code === net.code && (
                      <div className="absolute top-1.5 right-1.5 w-4 h-4 rounded-full bg-gold-500 flex items-center justify-center">
                        <CheckCircle2 className="w-2.5 h-2.5 text-black" />
                      </div>
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* ── Phone number ── */}
          <div>
            <label className="block text-[13px] font-bold text-white/50 uppercase tracking-wider mb-2">
              Phone Number
            </label>
            <div className="relative">
              <div className="absolute left-3.5 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                <PhoneIcon className="w-4 h-4 text-white/30" />
                <span className="text-[13px] text-white/30 font-mono">+234</span>
              </div>
              <input
                type="tel"
                value={phone.startsWith("0") ? phone.slice(1) : phone}
                onChange={e => handlePhone(e.target.value.startsWith("0") ? e.target.value : "0" + e.target.value)}
                placeholder="080 0000 0000"
                className="w-full h-12 rounded-xl bg-white/[0.05] border border-white/[0.08] pl-24 pr-4 text-white placeholder:text-white/20 focus:outline-none focus:border-gold-500/50 font-mono text-[15px]"
                maxLength={10}
              />
            </div>
            {isAuthenticated && (
              <p className="text-[11px] text-white/30 mt-1.5">Pre-filled with your registered number</p>
            )}
          </div>

          {/* ── Email (optional) ── */}
          <div>
            <label className="block text-[13px] font-bold text-white/50 uppercase tracking-wider mb-2">
              Email <span className="font-normal normal-case opacity-60">(for receipt)</span>
            </label>
            <input
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              placeholder="you@example.com"
              className="w-full h-12 rounded-xl bg-white/[0.05] border border-white/[0.08] px-4 text-white placeholder:text-white/20 focus:outline-none focus:border-gold-500/50 text-[14px]"
            />
          </div>

          {/* ── Airtime / Data toggle ── */}
          {selectedNetwork && (
            <div>
              <label className="block text-[13px] font-bold text-white/50 uppercase tracking-wider mb-3">
                Recharge Type
              </label>
              <div className="flex rounded-xl overflow-hidden border border-white/[0.08] bg-white/[0.02] p-1 gap-1">
                {([["airtime", PhoneIcon, "Airtime"], ["data", Wifi, "Data"]] as const).map(([type, Icon, label]) => (
                  <button
                    key={type}
                    onClick={() => { setType(type); setBundle(null); setAmount(""); setCustom(""); }}
                    className={`flex-1 flex items-center justify-center gap-2 py-2.5 rounded-lg text-[13px] font-bold transition-all ${
                      rechargeType === type
                        ? "bg-gold-500 text-black"
                        : "text-white/40 hover:text-white"
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                    {label}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* ── Airtime presets ── */}
          <AnimatePresence mode="wait">
            {selectedNetwork && rechargeType === "airtime" && (
              <motion.div
                key="airtime"
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -8 }}
                transition={{ duration: 0.2 }}
              >
                <label className="block text-[13px] font-bold text-white/50 uppercase tracking-wider mb-3">
                  Amount
                </label>
                <div className="grid grid-cols-3 sm:grid-cols-6 gap-2 mb-3">
                  {AIRTIME_PRESETS.map(p => (
                    <button
                      key={p}
                      onClick={() => { setAmount(p); setCustom(""); }}
                      className={`h-11 rounded-xl border text-[13px] font-bold transition-all ${
                        amountNaira === p
                          ? "border-gold-500 bg-gold-500/15 text-gold-400"
                          : "border-white/[0.08] bg-white/[0.02] text-white/60 hover:border-white/20 hover:text-white"
                      }`}
                    >
                      ₦{p >= 1000 ? p / 1000 + "k" : p}
                    </button>
                  ))}
                </div>
                {/* Custom amount */}
                <div className="relative">
                  <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-white/30 font-bold text-[14px]">₦</span>
                  <input
                    type="number"
                    value={customAmount}
                    onChange={e => { setCustom(e.target.value); setAmount(""); }}
                    placeholder="Custom amount"
                    min={100}
                    max={50000}
                    className="w-full h-12 rounded-xl bg-white/[0.05] border border-white/[0.08] pl-8 pr-4 text-white placeholder:text-white/20 focus:outline-none focus:border-gold-500/50 text-[14px]"
                  />
                </div>
              </motion.div>
            )}

            {/* ── Data bundles ── */}
            {selectedNetwork && rechargeType === "data" && (
              <motion.div
                key="data"
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -8 }}
                transition={{ duration: 0.2 }}
              >
                <label className="block text-[13px] font-bold text-white/50 uppercase tracking-wider mb-3">
                  Choose Bundle
                </label>
                {loadingBundles ? (
                  <div className="flex items-center gap-2 text-white/40 py-4">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span className="text-[13px]">Loading {selectedNetwork.code} bundles…</span>
                  </div>
                ) : bundles.length === 0 ? (
                  <p className="text-[13px] text-white/40 py-4">No data bundles available for {selectedNetwork.code}.</p>
                ) : (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-72 overflow-y-auto pr-1 scrollbar-thin">
                    {bundles.map(b => (
                      <button
                        key={b.id}
                        onClick={() => setBundle(b)}
                        className={`text-left p-3 rounded-xl border transition-all ${
                          selectedBundle?.id === b.id
                            ? "border-gold-500 bg-gold-500/10"
                            : "border-white/[0.08] bg-white/[0.02] hover:border-white/20"
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-[13px] font-bold text-white">{b.data_size || b.name.split(" ").slice(0,2).join(" ")}</span>
                          <span className="text-[13px] font-black text-gold-400">₦{b.price.toLocaleString()}</span>
                        </div>
                        <p className="text-[11px] text-white/40 mt-0.5 truncate">{b.name}</p>
                      </button>
                    ))}
                  </div>
                )}
              </motion.div>
            )}
          </AnimatePresence>

          {/* ── Summary ── */}
          {isValid && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              className="rounded-xl bg-white/[0.04] border border-white/[0.07] p-4 space-y-2"
            >
              <div className="flex items-center justify-between text-[13px]">
                <span className="text-white/50">Network</span>
                <span className="font-bold" style={{ color: NETWORK_COLORS[selectedNetwork?.code ?? ""] ?? "#fff" }}>
                  {selectedNetwork?.name}
                </span>
              </div>
              <div className="flex items-center justify-between text-[13px]">
                <span className="text-white/50">Phone</span>
                <span className="font-mono text-white">{phone}</span>
              </div>
              <div className="flex items-center justify-between text-[13px]">
                <span className="text-white/50">Type</span>
                <span className="text-white capitalize">{rechargeType}</span>
              </div>
              <div className="flex items-center justify-between text-[13px] pt-2 border-t border-white/[0.06]">
                <span className="text-white/50">Total</span>
                <span className="text-[16px] font-black text-gold-400">₦{effectiveAmount.toLocaleString()}</span>
              </div>
              <div className="flex items-center gap-2 mt-1">
                <Star className="w-3.5 h-3.5 text-gold-500" />
                <span className="text-[11px] text-gold-400">
                  Earn ~{Math.floor(effectiveAmount / 250)} Pulse Points + double when MTN confirms
                </span>
              </div>
            </motion.div>
          )}

          {/* ── Error ── */}
          {error && (
            <div className="flex items-center gap-2.5 p-3 rounded-xl bg-red-500/10 border border-red-500/20">
              <AlertCircle className="w-4 h-4 text-red-400 flex-shrink-0" />
              <p className="text-[13px] text-red-300">{error}</p>
            </div>
          )}

          {/* ── Submit ── */}
          <button
            onClick={handleSubmit}
            disabled={!isValid || submitting}
            className={`w-full h-14 rounded-xl font-black text-[15px] flex items-center justify-center gap-2.5 transition-all ${
              isValid && !submitting
                ? "bg-gold-500 text-black hover:bg-gold-400 shadow-lg shadow-gold-500/25"
                : "bg-white/[0.05] text-white/20 cursor-not-allowed"
            }`}
          >
            {submitting ? (
              <><Loader2 className="w-5 h-5 animate-spin" />Processing…</>
            ) : (
              <>Proceed to Payment <ChevronRight className="w-5 h-5" /></>
            )}
          </button>

          {/* ── Trust signals ── */}
          <div className="flex items-center justify-center gap-5 text-[11px] text-white/25">
            <span className="flex items-center gap-1"><Shield className="w-3 h-3" /> Secured by Paystack</span>
            <span className="flex items-center gap-1"><Zap className="w-3 h-3" /> Instant delivery</span>
            <span className="flex items-center gap-1"><TrendingUp className="w-3 h-3" /> Double points</span>
          </div>

          {/* ── Not logged in nudge ── */}
          {_hasHydrated && !isAuthenticated && (
            <div className="text-center pt-2 border-t border-white/[0.05]">
              <p className="text-[12px] text-white/30">
                <button onClick={() => setAuthOpen(true)} className="text-gold-400 hover:text-gold-300 font-bold underline">
                  Sign in
                </button>{" "}
                to link points to your account and track your rewards.
              </p>
            </div>
          )}
        </motion.div>

        {/* ── Why recharge here ── */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-8">
          {[
            { icon: <Gift className="w-5 h-5" />, title: "Double Points", body: "Earn from the platform AND from MTN's confirmation. Every recharge rewards you twice." },
            { icon: <Zap className="w-5 h-5" />, title: "Instant Top-Up", body: "Airtime and data delivered in seconds via VTPass — Nigeria's most reliable VTU network." },
            { icon: <Star className="w-5 h-5" />, title: "Spin the Wheel", body: "Recharge ₦1,000+ and earn a spin credit to win cash, data, and exclusive prizes." },
          ].map((f, i) => (
            <motion.div
              key={i}
              initial={{ opacity: 0, y: 16 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.2 + i * 0.08 }}
              className="p-4 rounded-2xl border border-white/[0.06] bg-white/[0.02]"
            >
              <div className="w-9 h-9 rounded-xl bg-gold-500/10 flex items-center justify-center text-gold-400 mb-3">
                {f.icon}
              </div>
              <h3 className="text-[14px] font-bold text-white mb-1">{f.title}</h3>
              <p className="text-[12px] text-white/40 leading-relaxed">{f.body}</p>
            </motion.div>
          ))}
        </div>

        {/* ── Back to home ── */}
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
