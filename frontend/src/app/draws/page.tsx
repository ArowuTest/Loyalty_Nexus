"use client";

import { useState, useEffect, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import { Ticket, Trophy, Clock, RefreshCw, Gift, Zap, ChevronDown, ChevronUp, Users } from "lucide-react";
import useSWR from "swr";
import { cn } from "@/lib/utils";
import toast, { Toaster } from "react-hot-toast";
import { useStore } from "@/store/useStore";

// ── Types ──────────────────────────────────────────────────────────────────
interface Draw {
  id: string;
  name: string;
  description?: string;
  prize_pool_kobo?: number;
  prize_pool?: number;
  status: string;
  draw_date: string;
  entry_count?: number;
  recurrence?: string;
}

interface DrawWinner {
  id: string;
  user_id?: string;
  phone_number?: string;
  prize_label?: string;
  rank?: number;
  created_at: string;
}

// ── Helpers ────────────────────────────────────────────────────────────────
function maskPhone(phone?: string) {
  if (!phone || phone.length < 8) return "****";
  return phone.slice(0, 4) + "****" + phone.slice(-3);
}

function timeRemaining(dateStr: string) {
  const diff = new Date(dateStr).getTime() - Date.now();
  if (diff <= 0) return "Ended";
  const d = Math.floor(diff / 86400000);
  const h = Math.floor((diff % 86400000) / 3600000);
  const m = Math.floor((diff % 3600000) / 60000);
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function prizePoolLabel(draw: Draw): string {
  const kobo = draw.prize_pool_kobo ?? (draw.prize_pool ?? 0) * 100;
  if (kobo <= 0) return "Prize TBD";
  return `₦${(kobo / 100).toLocaleString()}`;
}

function DrawCard({ draw }: { draw: Draw }) {
  const [expanded, setExpanded] = useState(false);
  const [winners, setWinners] = useState<DrawWinner[]>([]);
  const [loadingWinners, setLoadingWinners] = useState(false);
  const isEnded = draw.status?.toLowerCase() === "completed" || draw.status?.toLowerCase() === "ended";
  const [timeLeft, setTimeLeft] = useState(() => timeRemaining(draw.draw_date));

  useEffect(() => {
    if (isEnded) return;
    const t = setInterval(() => setTimeLeft(timeRemaining(draw.draw_date)), 60000);
    return () => clearInterval(t);
  }, [draw.draw_date, isEnded]);

  const loadWinners = useCallback(async () => {
    if (winners.length > 0) { setExpanded(e => !e); return; }
    setLoadingWinners(true);
    setExpanded(true);
    try {
      const res = await api.getDrawWinners(draw.id) as { winners: DrawWinner[] };
      setWinners(res.winners ?? []);
    } catch {
      toast.error("Failed to load winners");
    } finally {
      setLoadingWinners(false);
    }
  }, [draw.id, winners.length]);

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="nexus-card overflow-hidden"
    >
      {/* Top gradient bar */}
      <div className={cn("h-1 w-full", isEnded ? "bg-white/10" : "bg-gradient-to-r from-nexus-500 via-yellow-400 to-nexus-500")} />

      <div className="p-4 space-y-3">
        {/* Title row */}
        <div className="flex items-start justify-between gap-2">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h3 className="text-white font-bold text-base truncate">{draw.name}</h3>
              <span className={cn(
                "text-[10px] font-bold px-2 py-0.5 rounded-full",
                isEnded ? "bg-white/5 text-white/30" : "bg-nexus-500/20 text-nexus-300 border border-nexus-500/30"
              )}>
                {isEnded ? "Completed" : draw.recurrence?.toUpperCase() ?? "DAILY"}
              </span>
            </div>
            {draw.description && <p className="text-white/40 text-xs mt-0.5 line-clamp-1">{draw.description}</p>}
          </div>
          <div className="text-right flex-shrink-0">
            <p className="text-yellow-400 font-bold text-lg font-display">{prizePoolLabel(draw)}</p>
            <p className="text-white/25 text-[10px]">Prize Pool</p>
          </div>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-3 gap-2 text-center">
          <div className="bg-white/5 rounded-xl p-2">
            <p className="text-white/60 text-xs font-medium">Draw Date</p>
            <p className="text-white text-xs font-bold mt-0.5">
              {new Date(draw.draw_date).toLocaleDateString("en-NG", { month: "short", day: "numeric" })}
            </p>
          </div>
          <div className="bg-white/5 rounded-xl p-2">
            <p className="text-white/60 text-xs font-medium">{isEnded ? "Status" : "Time Left"}</p>
            <p className={cn("text-xs font-bold mt-0.5", isEnded ? "text-white/30" : "text-green-400")}>
              {isEnded ? "Done" : timeLeft}
            </p>
          </div>
          <div className="bg-white/5 rounded-xl p-2">
            <p className="text-white/60 text-xs font-medium">Entries</p>
            <p className="text-white text-xs font-bold mt-0.5">{(draw.entry_count ?? 0).toLocaleString()}</p>
          </div>
        </div>

        {/* How to enter */}
        {!isEnded && (
          <div className="flex items-start gap-2 bg-nexus-500/8 border border-nexus-500/20 rounded-xl p-3">
            <Zap size={14} className="text-nexus-400 mt-0.5 flex-shrink-0" />
            <p className="text-nexus-200/70 text-xs leading-relaxed">
              Every <span className="text-nexus-300 font-semibold">₦200 recharge</span> = 1 draw entry.
              Subscribe to <span className="text-nexus-300 font-semibold">Daily Draw Pass (₦20/day)</span> for guaranteed daily entry.
            </p>
          </div>
        )}

        {/* Winners toggle */}
        {isEnded && (
          <button
            onClick={loadWinners}
            className="w-full flex items-center justify-between px-3 py-2 bg-white/5 rounded-xl text-white/50 text-xs font-medium hover:text-white/70 hover:bg-white/8 transition-colors"
          >
            <span className="flex items-center gap-1.5"><Trophy size={13} className="text-yellow-400" /> View Winners</span>
            {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
          </button>
        )}

        <AnimatePresence>
          {expanded && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              className="overflow-hidden"
            >
              {loadingWinners ? (
                <div className="py-4 text-center text-white/30 text-sm flex items-center justify-center gap-2">
                  <RefreshCw size={14} className="animate-spin" /> Loading winners…
                </div>
              ) : winners.length === 0 ? (
                <p className="text-center text-white/30 text-xs py-3">No winners recorded yet</p>
              ) : (
                <div className="space-y-1.5 pt-1">
                  {winners.slice(0, 5).map((w, i) => (
                    <div key={w.id} className="flex items-center justify-between px-2 py-1.5 bg-white/5 rounded-lg">
                      <div className="flex items-center gap-2">
                        <span className="text-[10px] text-white/30 font-bold w-4">#{w.rank ?? i + 1}</span>
                        <Trophy size={12} className={i === 0 ? "text-yellow-400" : "text-white/20"} />
                        <span className="text-white/60 text-xs font-mono">{maskPhone(w.phone_number)}</span>
                      </div>
                      <span className="text-green-400 text-xs font-semibold">{w.prize_label ?? "Prize"}</span>
                    </div>
                  ))}
                </div>
              )}
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </motion.div>
  );
}
// ── Main Page ──────────────────────────────────────────────────────────────────
export default function DrawsPage() {
  const [draws, setDraws] = useState<Draw[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState(Date.now());

  // Fetch wallet to show user's personal draw entry count; fall back to persisted store value
  const storedWallet = useStore((s) => s.wallet);
  const { data: walletData } = useSWR(
    "/user/wallet",
    () => api.getWallet() as Promise<{ draw_counter: number; spin_credits: number; pulse_points: number }>,
    { refreshInterval: 30000 }
  );
  const effectiveWallet = walletData ?? (storedWallet as typeof walletData | null) ?? null;
  const myDrawEntries = effectiveWallet?.draw_counter ?? 0;

  const fetchDraws = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getDraws() as { draws: Draw[] };
      setDraws(res.draws ?? []);
    } catch (e) {
      setError("Unable to load draws. Please try again.");
    } finally {
      setLoading(false);
      setLastRefresh(Date.now());
    }
  }, []);

  useEffect(() => { fetchDraws(); }, [fetchDraws]);

  // Auto-refresh every 60 seconds
  useEffect(() => {
    const t = setInterval(fetchDraws, 60000);
    return () => clearInterval(t);
  }, [fetchDraws]);

  const upcoming = draws.filter(d => !["completed","ended"].includes(d.status?.toLowerCase()));
  const past = draws.filter(d => ["completed","ended"].includes(d.status?.toLowerCase()));

  return (
    <AppShell>
      <Toaster position="top-center" toastOptions={{
        style: { background: "#1c2038", color: "#fff", border: "1px solid rgba(255,255,255,0.1)" },
      }} />

      <div className="max-w-5xl mx-auto px-4 md:px-6 py-6 pb-28 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold font-display text-white flex items-center gap-2">
              <Ticket className="text-nexus-400" size={22} /> Daily Draws
            </h1>
            <p className="text-white/40 text-sm mt-0.5">Win big with every recharge</p>
          </div>
          <button
            onClick={fetchDraws}
            className="p-2 text-white/30 hover:text-white/60 transition-colors rounded-lg hover:bg-white/5"
          >
            <RefreshCw size={16} className={loading ? "animate-spin" : ""} />
          </button>
        </div>

        {/* My draw entries banner */}
        <div className={cn(
          "nexus-card px-4 py-3 flex items-center justify-between",
          myDrawEntries > 0 ? "border-nexus-500/30" : "border-white/5"
        )}>
          <div className="flex items-center gap-2">
            <Ticket className={myDrawEntries > 0 ? "text-nexus-400" : "text-white/20"} size={18} />
            <div>
              <p className="text-white/60 text-sm">My Draw Entries</p>
              <p className="text-white/30 text-[10px]">Every ₦200 recharge = 1 entry</p>
            </div>
          </div>
          <div className="text-right">
            <p className={cn(
              "text-2xl font-bold font-display",
              myDrawEntries > 0 ? "text-nexus-300" : "text-white/30"
            )}>{effectiveWallet ? myDrawEntries.toLocaleString() : "—"}</p>
            {myDrawEntries === 0 && (
              <p className="text-[10px] text-white/30">Recharge to earn entries</p>
            )}
          </div>
        </div>

        {/* How it works */}
        <div className="nexus-card p-4 space-y-3">
          <p className="text-white/50 text-xs font-semibold uppercase tracking-widest flex items-center gap-1.5">
            <Gift size={12} /> How to Enter
          </p>
          <div className="grid grid-cols-3 gap-3 text-center">
            {[
              { step: "1", text: "Recharge ₦200+", sub: "= 1 draw entry" },
              { step: "2", text: "Subscribe ₦20/day", sub: "guaranteed entry" },
              { step: "3", text: "Win Prizes", sub: "up to ₦50,000" },
            ].map(s => (
              <div key={s.step} className="space-y-1">
                <div className="w-7 h-7 rounded-full bg-nexus-500/20 text-nexus-300 text-sm font-bold mx-auto flex items-center justify-center">{s.step}</div>
                <p className="text-white/60 text-xs font-medium">{s.text}</p>
                <p className="text-white/30 text-[10px]">{s.sub}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Content */}
        {loading ? (
          <div className="space-y-3">
            {[...Array(2)].map((_, i) => <div key={i} className="nexus-card h-36 animate-pulse" />)}
          </div>
        ) : error ? (
          <div className="nexus-card p-8 text-center space-y-3">
            <p className="text-red-400 text-sm">{error}</p>
            <button onClick={fetchDraws} className="text-xs text-nexus-400 hover:text-nexus-300 transition-colors underline">
              Try again
            </button>
          </div>
        ) : (
          <>
            {/* Upcoming/Active draws */}
            {upcoming.length > 0 && (
              <div className="space-y-3">
                <p className="text-white/40 text-xs font-semibold uppercase tracking-widest flex items-center gap-1.5">
                  <Clock size={12} /> Active Draws
                </p>
                {upcoming.map(draw => <DrawCard key={draw.id} draw={draw} />)}
              </div>
            )}

            {upcoming.length === 0 && past.length === 0 && (
              <div className="nexus-card p-10 text-center space-y-3">
                <Ticket size={40} className="text-white/10 mx-auto" />
                <p className="text-white/40 font-medium">No draws available</p>
                <p className="text-white/20 text-sm">Keep recharging — a new draw will be announced soon!</p>
              </div>
            )}

            {/* Past draws */}
            {past.length > 0 && (
              <div className="space-y-3">
                <p className="text-white/40 text-xs font-semibold uppercase tracking-widest flex items-center gap-1.5">
                  <Users size={12} /> Past Draws & Winners
                </p>
                {past.map(draw => <DrawCard key={draw.id} draw={draw} />)}
              </div>
            )}
          </>
        )}

        {/* Last updated */}
        <p className="text-center text-white/15 text-[10px]">
          Last updated: {new Date(lastRefresh).toLocaleTimeString("en-NG")}
        </p>
      </div>
    </AppShell>
  );
}
