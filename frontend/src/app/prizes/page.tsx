"use client";

import { useState } from "react";
import useSWR from "swr";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import { Gift, Trophy, Phone, Wifi, Coins, CheckCircle, Clock, AlertCircle, ChevronRight, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { motion, AnimatePresence } from "framer-motion";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Win {
  id: string;
  prize_type: string;
  prize_value: number;
  prize_label: string;
  fulfillment_status: string;
  claim_status: string;
  created_at: string;
  expires_at: string;
  needs_momo_setup?: boolean;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const PRIZE_ICON: Record<string, React.ReactNode> = {
  airtime:      <Phone  size={20} className="text-blue-400" />,
  data_bundle:  <Wifi   size={20} className="text-purple-400" />,
  pulse_points: <Coins  size={20} className="text-brand-gold" />,
  momo_cash:    <Trophy size={20} className="text-green-400" />,
  try_again:    <X      size={20} className="text-white/20" />,
};

const PRIZE_BG: Record<string, string> = {
  airtime:      "bg-blue-500/10 border-blue-500/20",
  data_bundle:  "bg-purple-500/10 border-purple-500/20",
  pulse_points: "bg-brand-gold/10 border-brand-gold/20",
  momo_cash:    "bg-green-500/10 border-green-500/20",
  try_again:    "bg-white/3 border-white/5",
};

function claimBadge(status: string) {
  const map: Record<string, { label: string; cls: string }> = {
    PENDING:        { label: "Claim Now",     cls: "bg-brand-gold/20 text-brand-gold border-brand-gold/30" },
    PENDING_ADMIN_REVIEW: { label: "Under Review", cls: "bg-amber-500/20 text-amber-400 border-amber-500/30" },
    APPROVED:       { label: "Approved",      cls: "bg-green-500/20 text-green-400 border-green-500/30" },
    CLAIMED:        { label: "Claimed ✓",     cls: "bg-white/5 text-white/40 border-white/10" },
    REJECTED:       { label: "Rejected",      cls: "bg-red-500/20 text-red-400 border-red-500/30" },
    EXPIRED:        { label: "Expired",       cls: "bg-white/5 text-white/20 border-white/5" },
  };
  const s = map[status] ?? { label: status, cls: "bg-white/5 text-white/30 border-white/10" };
  return <span className={cn("text-[10px] font-bold px-2 py-0.5 rounded-full border", s.cls)}>{s.label}</span>;
}

function fulfillBadge(status: string) {
  const map: Record<string, string> = {
    completed:         "text-green-400",
    pending:           "text-amber-400",
    pending_claim:     "text-brand-gold",
    pending_momo_setup:"text-orange-400",
    processing:        "text-blue-400",
    failed:            "text-red-400",
    na:                "text-white/20",
  };
  return <span className={cn("text-[10px]", map[status] ?? "text-white/30")}>
    {status === "completed" ? "✓ Credited" :
     status === "pending" ? "⏳ Processing" :
     status === "pending_claim" ? "⚡ Awaiting claim" :
     status === "pending_momo_setup" ? "📱 Need MoMo" :
     status === "processing" ? "⚙️ In progress" :
     status === "failed" ? "✗ Failed" :
     status === "na" ? "" : status}
  </span>;
}

function timeAgo(iso: string) {
  const d = Date.now() - new Date(iso).getTime();
  const h = Math.floor(d / 3_600_000);
  if (h < 1) return "just now";
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

function expiresIn(iso: string) {
  const diff = new Date(iso).getTime() - Date.now();
  if (diff <= 0) return "Expired";
  const h = Math.floor(diff / 3_600_000);
  if (h < 1) return "< 1h left";
  if (h < 24) return `${h}h left`;
  return `${Math.floor(h / 24)}d left`;
}

// ─── Claim Modal ──────────────────────────────────────────────────────────────

function ClaimModal({ win, onClose, onSuccess }: {
  win: Win;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const needsMomo = win.prize_type === "momo_cash";
  const [momoNumber, setMomoNumber] = useState("");
  const [claiming, setClaiming]     = useState(false);
  const [claimErr, setClaimErr]     = useState<string | null>(null);

  const handleClaim = async () => {
    setClaiming(true);
    setClaimErr(null);
    try {
      await api.claimPrize(win.id, needsMomo ? { momo_number: momoNumber } : {});
      onSuccess();
      onClose();
    } catch (e: unknown) {
      setClaimErr(e instanceof Error ? e.message : "Claim failed. Please try again.");
    } finally {
      setClaiming(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4"
      style={{ background: "rgba(0,0,0,0.8)" }}
      onClick={e => e.target === e.currentTarget && onClose()}>
      <motion.div
        initial={{ opacity: 0, y: 40 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: 40 }}
        className={cn("nexus-card w-full max-w-sm p-6 space-y-4 border", PRIZE_BG[win.prize_type] ?? "border-white/10")}
      >
        {/* Icon + title */}
        <div className="flex items-center gap-3">
          <div className="w-12 h-12 rounded-2xl flex items-center justify-center bg-white/5 text-2xl">
            {win.prize_type === "airtime" ? "📱" :
             win.prize_type === "data_bundle" ? "📦" :
             win.prize_type === "momo_cash" ? "💵" : "🎁"}
          </div>
          <div>
            <h3 className="text-white font-bold">{win.prize_label}</h3>
            <p className="text-[rgb(130_140_180)] text-xs">
              {win.prize_type === "momo_cash" ? `Cash prize — ₦${(win.prize_value / 100).toLocaleString()}` :
               win.prize_type === "airtime" ? "Airtime credited to your number" :
               "Data bundle provisioned to your number"}
            </p>
          </div>
          <button onClick={onClose} className="ml-auto text-white/30 hover:text-white/60">
            <X size={18} />
          </button>
        </div>

        {/* MTN MoMo number for cash prizes (MoMo is the only disbursement channel — MTN-exclusive platform) */}
        {needsMomo && (
          <div>
            <label className="text-xs font-medium text-[rgb(130_140_180)] block mb-2">
              MTN MoMo Number (to receive ₦{(win.prize_value / 100).toLocaleString()})
            </label>
            <input
              type="tel"
              value={momoNumber}
              onChange={e => setMomoNumber(e.target.value)}
              placeholder="0801 234 5678"
              className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white text-sm
                         placeholder:text-white/20 focus:outline-none focus:border-brand-gold/40"
            />
            <p className="text-xs text-white/30 mt-1.5">
              Cash prizes are reviewed by admin within 24h before disbursement.
            </p>
          </div>
        )}

        {/* Airtime / data info */}
        {!needsMomo && (
          <div className="bg-white/3 rounded-xl p-3 text-xs text-[rgb(130_140_180)]">
            <p>✅ Your {win.prize_type === "airtime" ? "airtime" : "data"} will be provisioned to your registered number within a few minutes of claiming.</p>
          </div>
        )}

        {claimErr && (
          <div className="flex items-center gap-2 text-red-400 text-xs bg-red-500/10 rounded-xl p-3">
            <AlertCircle size={14} />
            {claimErr}
          </div>
        )}

        <button
          onClick={handleClaim}
          disabled={claiming || (needsMomo && momoNumber.length < 10)}
          className="w-full py-3.5 rounded-2xl font-bold text-sm bg-brand-gold text-black
                     hover:bg-brand-gold/90 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
        >
          {claiming ? "Processing…" : needsMomo ? "Submit Claim" : "Claim Now"}
        </button>
      </motion.div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

const TABS = ["All", "Pending", "Claimed"] as const;
type Tab = typeof TABS[number];

export default function PrizesPage() {
  const [tab, setTab]         = useState<Tab>("All");
  const [claimWin, setClaimWin] = useState<Win | null>(null);

  const { data: wins, mutate, isLoading } = useSWR<Win[]>(
    "/spin/wins",
    () => api.getMyWins() as Promise<Win[]>,
    { refreshInterval: 15_000 }
  );

  const all = wins ?? [];

  const filtered = all.filter(w => {
    if (tab === "Pending") return w.claim_status === "PENDING" || w.claim_status === "PENDING_ADMIN_REVIEW";
    if (tab === "Claimed") return w.claim_status === "CLAIMED" || w.claim_status === "APPROVED";
    return true;
  }).filter(w => w.prize_type !== "try_again");

  const pendingCount = all.filter(w =>
    w.claim_status === "PENDING" && w.prize_type !== "try_again"
  ).length;

  return (
    <AppShell>
      <div className="max-w-5xl mx-auto px-4 md:px-6 py-6 space-y-5">
        {/* Header */}
        <div className="flex items-center gap-3">
          <Gift size={22} className="text-brand-gold" />
          <div>
            <h1 className="text-xl font-black text-white">My Prizes</h1>
            <p className="text-[rgb(130_140_180)] text-xs">
              {pendingCount > 0 ? `${pendingCount} prize${pendingCount > 1 ? "s" : ""} waiting to be claimed` : "All your spin wins"}
            </p>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 bg-white/3 rounded-2xl p-1">
          {TABS.map(t => (
            <button key={t} onClick={() => setTab(t)}
              className={cn(
                "flex-1 py-2 rounded-xl text-sm font-semibold transition-all",
                tab === t ? "bg-nexus-600 text-white" : "text-[rgb(130_140_180)] hover:text-white"
              )}>
              {t}
              {t === "Pending" && pendingCount > 0 && (
                <span className="ml-1.5 bg-brand-gold text-black text-[10px] font-black px-1.5 rounded-full">
                  {pendingCount}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Loading */}
        {isLoading && (
          <div className="space-y-3">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="nexus-card p-4 animate-pulse h-20 rounded-2xl" />
            ))}
          </div>
        )}

        {/* Empty */}
        {!isLoading && filtered.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center space-y-3">
            <Gift size={40} className="text-white/10" />
            <p className="text-white/40 font-semibold">
              {tab === "Pending" ? "No prizes waiting to be claimed" :
               tab === "Claimed" ? "No claimed prizes yet" :
               "No prizes yet — go spin the wheel!"}
            </p>
          </div>
        )}

        {/* Prize List */}
        {!isLoading && filtered.length > 0 && (
          <AnimatePresence initial={false}>
            <div className="space-y-3">
              {filtered.map((win, i) => {
                const canClaim = win.claim_status === "PENDING" &&
                  (win.prize_type === "airtime" || win.prize_type === "data_bundle" || win.prize_type === "momo_cash");
                const expiry = expiresIn(win.expires_at);
                const isExpiring = !expiry.includes("d") && expiry !== "Expired";

                return (
                  <motion.div
                    key={win.id}
                    initial={{ opacity: 0, y: 6 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.04 }}
                    className={cn(
                      "nexus-card p-4 flex items-center gap-3 border",
                      PRIZE_BG[win.prize_type] ?? "border-white/10",
                      canClaim && "cursor-pointer hover:border-brand-gold/30 transition-colors"
                    )}
                    onClick={() => canClaim && setClaimWin(win)}
                  >
                    {/* Icon */}
                    <div className={cn("w-11 h-11 rounded-2xl flex items-center justify-center shrink-0", "bg-white/5")}>
                      {PRIZE_ICON[win.prize_type] ?? <Gift size={20} />}
                    </div>

                    {/* Info */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <p className="text-white font-semibold text-sm leading-tight">{win.prize_label}</p>
                        {claimBadge(win.claim_status)}
                      </div>
                      <div className="flex items-center gap-3 mt-0.5">
                        {fulfillBadge(win.fulfillment_status)}
                        <span className="text-[10px] text-white/30">{timeAgo(win.created_at)}</span>
                        {win.claim_status === "PENDING" && (
                          <span className={cn("text-[10px] flex items-center gap-0.5", isExpiring ? "text-amber-400" : "text-white/30")}>
                            <Clock size={10} />
                            {expiry}
                          </span>
                        )}
                      </div>
                    </div>

                    {/* CTA */}
                    {canClaim && (
                      <div className="flex items-center gap-1.5 text-brand-gold">
                        <span className="text-xs font-bold">Claim</span>
                        <ChevronRight size={14} />
                      </div>
                    )}
                    {win.claim_status === "CLAIMED" && (
                      <CheckCircle size={18} className="text-green-400 shrink-0" />
                    )}
                    {win.claim_status === "PENDING_ADMIN_REVIEW" && (
                      <Clock size={18} className="text-amber-400 shrink-0" />
                    )}
                  </motion.div>
                );
              })}
            </div>
          </AnimatePresence>
        )}

        {/* Info card */}
        <div className="nexus-card p-4 space-y-1.5 border border-white/5">
          <p className="text-white/40 text-xs font-semibold uppercase tracking-wider">How Claiming Works</p>
          <div className="space-y-1">
            {[
              { icon: "📱", text: "Airtime & Data: Auto-credited to your number within minutes of claiming" },
              { icon: "💵", text: "MoMo Cash: Submit your MoMo number → reviewed by admin → paid within 24h" },
              { icon: "💎", text: "Pulse Points: Instantly credited to your wallet at spin time — nothing to claim" },
              { icon: "⏰", text: "Claims expire in 7 days — claim promptly!" },
            ].map(({ icon, text }) => (
              <p key={text} className="text-white/30 text-xs flex items-start gap-2">
                <span className="shrink-0">{icon}</span>
                {text}
              </p>
            ))}
          </div>
        </div>
      </div>

      {/* Claim Modal */}
      <AnimatePresence>
        {claimWin && (
          <ClaimModal
            win={claimWin}
            onClose={() => setClaimWin(null)}
            onSuccess={() => mutate()}
          />
        )}
      </AnimatePresence>
    </AppShell>
  );
}
