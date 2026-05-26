"use client";

import { useState, useMemo } from "react";
import useSWR from "swr";
import Link from "next/link";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import {
  Smartphone, ArrowLeft, Search, CheckCircle,
  Clock, XCircle, Zap, Trophy, RefreshCw, Database
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

interface UserRecharge {
  id: string;
  msisdn: string;
  network: string;
  recharge_type: "AIRTIME" | "DATA";
  amount_kobo: number;
  status: "PENDING" | "PROCESSING" | "SUCCESS" | "FAILED";
  points_earned: number;
  spin_eligible: boolean;
  draw_entries: number;
  created_at: string;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

const NETWORK_COLOR: Record<string, string> = {
  MTN:       "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
  AIRTEL:    "bg-red-500/20 text-red-400 border-red-500/30",
  GLO:       "bg-green-500/20 text-green-400 border-green-500/30",
  "9MOBILE": "bg-green-400/20 text-emerald-400 border-emerald-500/30",
};

function networkBadge(net: string) {
  const cls = NETWORK_COLOR[net?.toUpperCase()] ?? "bg-white/10 text-white/50 border-white/10";
  return (
    <span className={cn("text-[10px] font-bold px-2 py-0.5 rounded-full border", cls)}>
      {net}
    </span>
  );
}

function statusBadge(status: string) {
  const s = (status ?? "").toUpperCase();
  const map: Record<string, string> = {
    SUCCESS:    "bg-green-500/15 text-green-400 border-green-500/30",
    PENDING:    "bg-amber-500/15 text-amber-400 border-amber-500/30",
    PROCESSING: "bg-blue-500/15 text-blue-400 border-blue-500/30",
    FAILED:     "bg-red-500/15 text-red-400 border-red-500/30",
  };
  const cls = map[s] ?? "bg-white/10 text-white/40 border-white/10";
  return <span className={cn("text-[10px] font-bold px-2 py-0.5 rounded-full border", cls)}>{s}</span>;
}

function statusIcon(status: string) {
  const s = (status ?? "").toUpperCase();
  if (s === "SUCCESS") return <CheckCircle size={14} className="text-green-400" />;
  if (s === "PENDING") return <Clock size={14} className="text-amber-400" />;
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
  if (h < 1)  return "Just now";
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

// ─── Stat cards ───────────────────────────────────────────────────────────────

function StatCards({ items }: {
  items: { label: string; value: string; icon: React.ElementType; color: string }[]
}) {
  return (
    <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }}
      className="grid grid-cols-2 sm:grid-cols-4 gap-3">
      {items.map(({ label, value, icon: Icon, color }) => (
        <div key={label} className="rounded-2xl p-4 border"
          style={{ background: "rgba(255,255,255,0.03)", borderColor: "rgba(255,255,255,0.07)" }}>
          <div className="flex items-center gap-2 mb-1.5">
            <Icon size={14} className={color} />
            <p className="text-[10px] text-white/40 font-semibold uppercase tracking-wider">{label}</p>
          </div>
          <p className="text-lg font-black text-white">{value}</p>
        </div>
      ))}
    </motion.div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function TransactionsPage() {
  const [tab,       setTab]    = useState<"transactions" | "recharges">("transactions");
  const [search,    setSearch] = useState("");
  const [netFilter, setNet]    = useState<string>("All");

  // Transactions tab data
  const { data: txData, isLoading: txLoading, mutate: txMutate } = useSWR<Recharge[]>(
    "/user/transactions",
    () => api.getTransactions() as Promise<Recharge[]>,
    { refreshInterval: 30_000 }
  );

  // Recharges tab data (lazy-fetched)
  const { data: rcData, isLoading: rcLoading, mutate: rcMutate } = useSWR<UserRecharge[]>(
    tab === "recharges" ? "/user/recharges" : null,
    () => api.getUserRecharges() as Promise<UserRecharge[]>,
    { refreshInterval: 30_000 }
  );

  const all   = txData ?? [];
  const allRc = rcData ?? [];

  const txStats = useMemo(() => ({
    total:       all.length,
    successful:  all.filter(r => (r.status ?? "").toUpperCase() === "SUCCESS").length,
    totalSpent:  all.filter(r => (r.status ?? "").toUpperCase() === "SUCCESS").reduce((s, r) => s + r.amount_kobo, 0),
    totalPoints: all.reduce((s, r) => s + (r.points_earned ?? 0), 0),
  }), [all]);

  const rcStats = useMemo(() => ({
    total:       allRc.length,
    successful:  allRc.filter(r => r.status === "SUCCESS").length,
    totalSpent:  allRc.filter(r => r.status === "SUCCESS").reduce((s, r) => s + r.amount_kobo, 0),
    totalPoints: allRc.reduce((s, r) => s + (r.points_earned ?? 0), 0),
  }), [allRc]);

  const networks = useMemo(() =>
    ["All", ...[...new Set(all.map(r => r.network).filter(Boolean))]],
    [all]
  );

  const filtered = useMemo(() =>
    all
      .filter(r => netFilter === "All" || r.network === netFilter)
      .filter(r => {
        if (!search) return true;
        const q = search.toLowerCase();
        return r.msisdn?.includes(q) || r.network?.toLowerCase().includes(q) ||
          r.recharge_type?.toLowerCase().includes(q) || r.status?.toLowerCase().includes(q) ||
          r.payment_reference?.toLowerCase().includes(q);
      }),
    [all, search, netFilter]
  );

  const filteredRc = useMemo(() =>
    allRc.filter(r => {
      if (!search) return true;
      const q = search.toLowerCase();
      return r.msisdn?.includes(q) || r.network?.toLowerCase().includes(q) ||
        r.recharge_type?.toLowerCase().includes(q) || r.status?.toLowerCase().includes(q);
    }),
    [allRc, search]
  );

  const isLoading     = tab === "transactions" ? txLoading : rcLoading;
  const handleRefresh = () => (tab === "transactions" ? txMutate() : rcMutate());

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
            <h1 className="text-xl font-black text-white">History</h1>
            <p className="text-[rgb(130,140,180)] text-xs">Your transactions and recharges</p>
          </div>
          <button onClick={handleRefresh}
            className="w-9 h-9 rounded-xl flex items-center justify-center hover:bg-white/5 transition-colors text-white/40 hover:text-white/80">
            <RefreshCw size={16} className={isLoading ? "animate-spin" : ""} />
          </button>
        </div>

        {/* Tab switcher */}
        <div className="flex gap-1 p-1 rounded-2xl border"
          style={{ background: "rgba(255,255,255,0.025)", borderColor: "rgba(255,255,255,0.07)" }}>
          {([
            { key: "transactions" as const, label: "Transactions", icon: Smartphone },
            { key: "recharges"    as const, label: "Recharges",    icon: Database   },
          ]).map(({ key, label, icon: Icon }) => (
            <button key={key}
              onClick={() => { setTab(key); setSearch(""); setNet("All"); }}
              className={cn(
                "flex-1 flex items-center justify-center gap-2 py-2.5 rounded-xl text-sm font-semibold transition-all",
                tab === key
                  ? "bg-gold-500/20 text-gold-400 border border-gold-500/30"
                  : "text-white/40 hover:text-white/70"
              )}>
              <Icon size={14} />
              {label}
            </button>
          ))}
        </div>

        {/* ═══════════ TRANSACTIONS TAB ═══════════ */}
        {tab === "transactions" && (
          <>
            {!txLoading && all.length > 0 && (
              <StatCards items={[
                { label: "Total",         value: txStats.total.toString(),                      icon: Smartphone,  color: "text-blue-400"   },
                { label: "Successful",    value: txStats.successful.toString(),                 icon: CheckCircle, color: "text-green-400"  },
                { label: "Total Spent",   value: formatNaira(txStats.totalSpent),               icon: Zap,         color: "text-gold-500"   },
                { label: "Points Earned", value: txStats.totalPoints.toLocaleString() + " pts", icon: Trophy,      color: "text-purple-400" },
              ]} />
            )}

            <div className="flex gap-2 items-center flex-wrap">
              <div className="relative flex-1 min-w-[160px]">
                <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-white/30" />
                <input value={search} onChange={e => setSearch(e.target.value)}
                  placeholder="Search by number, network, ref..."
                  className="w-full bg-white/5 border border-white/10 rounded-xl pl-9 pr-3 py-2.5 text-sm text-white placeholder:text-white/25 focus:outline-none focus:border-gold-500/40" />
              </div>
              <div className="flex gap-1 flex-wrap">
                {networks.map(net => (
                  <button key={net} onClick={() => setNet(net)}
                    className={cn(
                      "px-3 py-2 rounded-xl text-xs font-semibold transition-all border",
                      netFilter === net
                        ? "bg-gold-500/20 text-gold-400 border-gold-500/40"
                        : "bg-white/3 text-white/40 border-white/5 hover:text-white/70"
                    )}>{net}</button>
                ))}
              </div>
            </div>

            {txLoading && (
              <div className="space-y-2">
                {[...Array(6)].map((_, i) => (
                  <div key={i} className="rounded-2xl p-4 animate-pulse h-16"
                    style={{ background: "rgba(255,255,255,0.03)" }} />
                ))}
              </div>
            )}

            {!txLoading && filtered.length === 0 && (
              <div className="flex flex-col items-center justify-center py-20 text-center space-y-3">
                <Smartphone size={40} className="text-white/10" />
                <div>
                  <p className="text-white/40 font-semibold">
                    {all.length === 0 ? "No transactions yet" : "No matching transactions"}
                  </p>
                  {all.length === 0 && (
                    <p className="text-white/25 text-sm mt-1">Recharge to earn points and win prizes</p>
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

            {!txLoading && filtered.length > 0 && (
              <div className="space-y-2">
                {filtered.map((tx, i) => (
                  <motion.div key={tx.id}
                    initial={{ opacity: 0, y: 5 }} animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: Math.min(i * 0.03, 0.3) }}
                    className="rounded-2xl p-4 border flex items-center gap-3"
                    style={{ background: "rgba(255,255,255,0.025)", borderColor: "rgba(255,255,255,0.06)" }}>
                    <div className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0"
                      style={{ background: "rgba(245,166,35,0.10)" }}>
                      <Smartphone size={18} className="text-gold-400" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        {networkBadge(tx.network)}
                        <span className="text-[11px] text-white/50 capitalize">
                          {tx.recharge_type === "airtime" ? "Airtime" : tx.recharge_type === "data" ? "Data" : tx.recharge_type}
                        </span>
                        {tx.spin_eligible && (
                          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded-full bg-purple-500/20 text-purple-400 border border-purple-500/30">
                            Spin earned
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-3 mt-0.5">
                        {statusIcon(tx.status)}
                        <span className="text-[11px] text-white/30">{relativeTime(tx.created_at)}</span>
                        {tx.msisdn && <span className="text-[11px] text-white/25 font-mono">{tx.msisdn}</span>}
                      </div>
                    </div>
                    <div className="text-right shrink-0">
                      <p className="text-white font-black text-sm">{formatNaira(tx.amount_kobo)}</p>
                      {tx.points_earned > 0 && (
                        <p className="text-[11px] font-bold" style={{ color: "var(--gold)" }}>
                          +{tx.points_earned} pts
                        </p>
                      )}
                      {tx.draw_entries > 0 && (
                        <p className="text-[11px] text-purple-400">+{tx.draw_entries} entries</p>
                      )}
                    </div>
                  </motion.div>
                ))}
              </div>
            )}

            {!txLoading && all.length > 0 && (
              <p className="text-center text-white/20 text-xs pb-4">
                Showing {filtered.length} of {all.length} transaction{all.length !== 1 ? "s" : ""}
              </p>
            )}
          </>
        )}

        {/* ═══════════ RECHARGES TAB ═══════════ */}
        {tab === "recharges" && (
          <>
            {!rcLoading && allRc.length > 0 && (
              <StatCards items={[
                { label: "Total Recharges", value: rcStats.total.toString(),                      icon: Database,    color: "text-blue-400"   },
                { label: "Successful",      value: rcStats.successful.toString(),                 icon: CheckCircle, color: "text-green-400"  },
                { label: "Total Spent",     value: formatNaira(rcStats.totalSpent),               icon: Zap,         color: "text-gold-500"   },
                { label: "Points Earned",   value: rcStats.totalPoints.toLocaleString() + " pts", icon: Trophy,      color: "text-purple-400" },
              ]} />
            )}

            <div className="relative">
              <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-white/30" />
              <input value={search} onChange={e => setSearch(e.target.value)}
                placeholder="Search by number, network, type..."
                className="w-full bg-white/5 border border-white/10 rounded-xl pl-9 pr-3 py-2.5 text-sm text-white placeholder:text-white/25 focus:outline-none focus:border-gold-500/40" />
            </div>

            {rcLoading && (
              <div className="space-y-2">
                {[...Array(6)].map((_, i) => (
                  <div key={i} className="rounded-2xl p-4 animate-pulse h-20"
                    style={{ background: "rgba(255,255,255,0.03)" }} />
                ))}
              </div>
            )}

            {!rcLoading && filteredRc.length === 0 && (
              <div className="flex flex-col items-center justify-center py-20 text-center space-y-3">
                <Database size={40} className="text-white/10" />
                <div>
                  <p className="text-white/40 font-semibold">
                    {allRc.length === 0 ? "No recharges yet" : "No matching recharges"}
                  </p>
                  {allRc.length === 0 && (
                    <p className="text-white/25 text-sm mt-1">Recharge your number to earn points and win prizes</p>
                  )}
                </div>
                {allRc.length === 0 && (
                  <Link href="/recharge">
                    <button className="bg-gold-500 text-black font-bold px-5 py-2.5 rounded-xl text-sm inline-flex items-center gap-2">
                      <Zap size={14} /> Recharge Now
                    </button>
                  </Link>
                )}
              </div>
            )}

            {!rcLoading && filteredRc.length > 0 && (
              <div className="space-y-2">
                {filteredRc.map((rc, i) => (
                  <motion.div key={rc.id}
                    initial={{ opacity: 0, y: 5 }} animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: Math.min(i * 0.03, 0.3) }}
                    className="rounded-2xl p-4 border"
                    style={{ background: "rgba(255,255,255,0.025)", borderColor: "rgba(255,255,255,0.06)" }}>
                    {/* Row 1: badges + amount */}
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex items-center gap-2 flex-wrap">
                        {networkBadge(rc.network)}
                        <span className={cn(
                          "text-[10px] font-bold px-2 py-0.5 rounded-full border",
                          rc.recharge_type === "AIRTIME"
                            ? "bg-amber-500/15 text-amber-400 border-amber-500/30"
                            : "bg-cyan-500/15 text-cyan-400 border-cyan-500/30"
                        )}>
                          {rc.recharge_type}
                        </span>
                        {statusBadge(rc.status)}
                        {rc.spin_eligible && (
                          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded-full bg-purple-500/20 text-purple-400 border border-purple-500/30">
                            Spin Eligible
                          </span>
                        )}
                      </div>
                      <p className="text-white font-black text-sm shrink-0">{formatNaira(rc.amount_kobo)}</p>
                    </div>
                    {/* Row 2: date + points */}
                    <div className="flex items-center justify-between mt-2">
                      <div className="flex items-center gap-2">
                        <span className="text-[11px] text-white/30">{formatDate(rc.created_at)}</span>
                        {rc.msisdn && (
                          <span className="text-[11px] text-white/25 font-mono">{rc.msisdn}</span>
                        )}
                      </div>
                      <div className="flex items-center gap-3">
                        {rc.points_earned > 0 && (
                          <span className="text-[11px] font-bold" style={{ color: "var(--gold)" }}>
                            +{rc.points_earned} pts
                          </span>
                        )}
                        {rc.draw_entries > 0 && (
                          <span className="text-[11px] text-purple-400">+{rc.draw_entries} entries</span>
                        )}
                      </div>
                    </div>
                  </motion.div>
                ))}
              </div>
            )}

            {!rcLoading && allRc.length > 0 && (
              <p className="text-center text-white/20 text-xs pb-4">
                Showing {filteredRc.length} of {allRc.length} recharge{allRc.length !== 1 ? "s" : ""}
              </p>
            )}
          </>
        )}

      </div>
    </AppShell>
  );
}
