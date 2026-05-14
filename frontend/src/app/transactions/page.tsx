"use client";

import { useState, useMemo } from "react";
import useSWR from "swr";
import Link from "next/link";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import {
  Smartphone, ArrowLeft, Search, Download, CheckCircle,
  Clock, XCircle, Zap, Trophy, RefreshCw
} from "lucide-react";
import { cn } from "@/lib/utils";
import { motion } from "framer-motion";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Recharge {
  id: string;
  msisdn: string;
  network: string;
  recharge_type: string;
  amount_kobo: number;
  status: string;
  points_earned: number;
  draw_entries: number;
  spin_eligible: boolean;
  payment_reference: string;
  created_at: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

const NETWORK_COLOR: Record<string, string> = {
  MTN:     "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
  AIRTEL:  "bg-red-500/20 text-red-400 border-red-500/30",
  GLO:     "bg-green-500/20 text-green-400 border-green-500/30",
  "9MOBILE":"bg-green-400/20 text-emerald-400 border-emerald-500/30",
};

function networkBadge(net: string) {
  const cls = NETWORK_COLOR[net?.toUpperCase()] ?? "bg-white/10 text-white/50 border-white/10";
  return (
    <span className={cn("text-[10px] font-bold px-2 py-0.5 rounded-full border", cls)}>
      {net}
    </span>
  );
}

function statusIcon(status: string) {
  if (status === "SUCCESS" || status === "success")
    return <CheckCircle size={14} className="text-green-400" />;
  if (status === "PENDING" || status === "pending")
    return <Clock size={14} className="text-amber-400" />;
  return <XCircle size={14} className="text-red-400" />;
}

function formatNaira(kobo: number) {
  return "₦" + (kobo / 100).toLocaleString("en-NG", { minimumFractionDigits: 0, maximumFractionDigits: 0 });
}

function formatDate(iso: string) {
  const d = new Date(iso);
  return d.toLocaleDateString("en-NG", { day: "numeric", month: "short", year: "numeric" }) +
    " · " + d.toLocaleTimeString("en-NG", { hour: "2-digit", minute: "2-digit" });
}

function relativeTime(iso: string) {
  const diff = Date.now() - new Date(iso).getTime();
  const h = Math.floor(diff / 3_600_000);
  if (h < 1) return "Just now";
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function TransactionsPage() {
  const [search, setSearch]   = useState("");
  const [netFilter, setNet]   = useState<string>("All");

  const { data, isLoading, mutate } = useSWR<Recharge[]>(
    "/user/transactions",
    () => api.getTransactions() as unknown as Promise<Recharge[]>,
    { refreshInterval: 30_000 }
  );

  const all = data ?? [];

  // Summary stats
  const stats = useMemo(() => ({
    total:      all.length,
    successful: all.filter(r => r.status === "SUCCESS" || r.status === "success").length,
    totalSpent: all.filter(r => r.status === "SUCCESS" || r.status === "success")
                   .reduce((s, r) => s + r.amount_kobo, 0),
    totalPoints: all.reduce((s, r) => s + (r.points_earned ?? 0), 0),
  }), [all]);

  // Available networks for filter
  const networks = useMemo(() => {
    const nets = [...new Set(all.map(r => r.network).filter(Boolean))];
    return ["All", ...nets];
  }, [all]);

  // Filtered list
  const filtered = useMemo(() => {
    return all
      .filter(r => netFilter === "All" || r.network === netFilter)
      .filter(r => {
        if (!search) return true;
        const q = search.toLowerCase();
        return (
          r.msisdn?.includes(q) ||
          r.network?.toLowerCase().includes(q) ||
          r.recharge_type?.toLowerCase().includes(q) ||
          r.status?.toLowerCase().includes(q) ||
          r.payment_reference?.toLowerCase().includes(q)
        );
      });
  }, [all, search, netFilter]);

  return (
    <AppShell>
      <div className="max-w-5xl mx-auto px-4 md:px-6 py-6 space-y-5">

        {/* Header */}
        <div className="flex items-center gap-3">
          <Link href="/dashboard">
            <button className="w-9 h-9 rounded-xl flex items-center justify-center hover:bg-white/5 transition-colors text-white/40 hover:text-white/80">
              <ArrowLeft size={16} />
            </button>
          </Link>
          <div className="flex-1">
            <h1 className="text-xl font-black text-white">Recharge History</h1>
            <p className="text-[rgb(130,140,180)] text-xs">All your MTN recharges and points earned</p>
          </div>
          <button onClick={() => mutate()} className="w-9 h-9 rounded-xl flex items-center justify-center hover:bg-white/5 transition-colors text-white/40 hover:text-white/80">
            <RefreshCw size={16} />
          </button>
        </div>

        {/* Stats row */}
        {!isLoading && all.length > 0 && (
          <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            className="grid grid-cols-2 sm:grid-cols-4 gap-3"
          >
            {[
              { label: "Total Recharges", value: stats.total.toString(),          icon: Smartphone, color: "text-blue-400" },
              { label: "Successful",      value: stats.successful.toString(),     icon: CheckCircle, color: "text-green-400" },
              { label: "Total Spent",     value: formatNaira(stats.totalSpent),   icon: Zap, color: "text-gold-500" },
              { label: "Points Earned",   value: stats.totalPoints.toLocaleString() + " pts", icon: Trophy, color: "text-purple-400" },
            ].map(({ label, value, icon: Icon, color }) => (
              <div key={label}
                className="rounded-2xl p-4 border"
                style={{ background: "rgba(255,255,255,0.03)", borderColor: "rgba(255,255,255,0.07)" }}
              >
                <div className="flex items-center gap-2 mb-1.5">
                  <Icon size={14} className={color} />
                  <p className="text-[10px] text-white/40 font-semibold uppercase tracking-wider">{label}</p>
                </div>
                <p className="text-lg font-black text-white">{value}</p>
              </div>
            ))}
          </motion.div>
        )}

        {/* Filters bar */}
        <div className="flex gap-2 items-center flex-wrap">
          <div className="relative flex-1 min-w-[160px]">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-white/30" />
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search by number, network, ref…"
              className="w-full bg-white/5 border border-white/10 rounded-xl pl-9 pr-3 py-2.5 text-sm text-white placeholder:text-white/25 focus:outline-none focus:border-gold-500/40"
            />
          </div>
          <div className="flex gap-1 flex-wrap">
            {networks.map(net => (
              <button key={net} onClick={() => setNet(net)}
                className={cn(
                  "px-3 py-2 rounded-xl text-xs font-semibold transition-all border",
                  netFilter === net
                    ? "bg-gold-500/20 text-gold-400 border-gold-500/40"
                    : "bg-white/3 text-white/40 border-white/5 hover:text-white/70"
                )}>
                {net}
              </button>
            ))}
          </div>
        </div>

        {/* Loading skeletons */}
        {isLoading && (
          <div className="space-y-2">
            {[...Array(6)].map((_, i) => (
              <div key={i} className="rounded-2xl p-4 animate-pulse h-16"
                style={{ background: "rgba(255,255,255,0.03)" }} />
            ))}
          </div>
        )}

        {/* Empty state */}
        {!isLoading && filtered.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center space-y-3">
            <Smartphone size={40} className="text-white/10" />
            <div>
              <p className="text-white/40 font-semibold">
                {all.length === 0 ? "No recharges yet" : "No matching recharges"}
              </p>
              {all.length === 0 && (
                <p className="text-white/25 text-sm mt-1">
                  Recharge your MTN number to earn points and win prizes
                </p>
              )}
            </div>
            {all.length === 0 && (
              <Link href="/recharge">
                <button className="bg-gold-500 text-black font-bold px-5 py-2.5 rounded-xl text-sm inline-flex items-center gap-2">
                  <Zap size={14} /> Recharge Now
                </button>
              </Link>
            )}
          </div>
        )}

        {/* Transaction list */}
        {!isLoading && filtered.length > 0 && (
          <div className="space-y-2">
            {filtered.map((tx, i) => (
              <motion.div
                key={tx.id}
                initial={{ opacity: 0, y: 5 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: Math.min(i * 0.03, 0.3) }}
                className="rounded-2xl p-4 border flex items-center gap-3"
                style={{ background: "rgba(255,255,255,0.025)", borderColor: "rgba(255,255,255,0.06)" }}
              >
                {/* Network icon */}
                <div className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0"
                  style={{ background: "rgba(245,166,35,0.10)" }}>
                  <Smartphone size={18} className="text-gold-400" />
                </div>

                {/* Main info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    {networkBadge(tx.network)}
                    <span className="text-[11px] text-white/50 capitalize">
                      {tx.recharge_type === "airtime" ? "Airtime" : tx.recharge_type === "data" ? "Data" : tx.recharge_type}
                    </span>
                    {tx.spin_eligible && (
                      <span className="text-[10px] font-bold px-1.5 py-0.5 rounded-full bg-purple-500/20 text-purple-400 border border-purple-500/30">
                        🎰 Spin earned
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-3 mt-0.5">
                    {statusIcon(tx.status)}
                    <span className="text-[11px] text-white/30">{relativeTime(tx.created_at)}</span>
                    {tx.msisdn && (
                      <span className="text-[11px] text-white/25 font-mono">{tx.msisdn}</span>
                    )}
                  </div>
                </div>

                {/* Amount + points */}
                <div className="text-right shrink-0">
                  <p className="text-white font-black text-sm">{formatNaira(tx.amount_kobo)}</p>
                  {tx.points_earned > 0 && (
                    <p className="text-[11px] font-bold" style={{ color: "var(--gold)" }}>
                      +{tx.points_earned} pts
                    </p>
                  )}
                  {tx.draw_entries > 0 && (
                    <p className="text-[11px] text-purple-400">
                      +{tx.draw_entries} entries
                    </p>
                  )}
                </div>
              </motion.div>
            ))}
          </div>
        )}

        {/* Footer info */}
        {!isLoading && all.length > 0 && (
          <p className="text-center text-white/20 text-xs pb-4">
            Showing {filtered.length} of {all.length} recharge{all.length !== 1 ? "s" : ""}
          </p>
        )}
      </div>
    </AppShell>
  );
}
