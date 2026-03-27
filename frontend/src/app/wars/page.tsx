"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import useSWR from "swr";
import AppShell from "@/components/layout/AppShell";
import { Globe, Trophy, Users, TrendingUp, Clock, AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import api from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────────────────────

interface LeaderboardEntry {
  state: string;
  total_points: number;
  active_members: number;
  rank: number;
  prize_kobo: number;
  period: string;
}

interface WarLeaderboardData {
  leaderboard: LeaderboardEntry[];
  count: number;
  period: string;
}

interface MyRankData {
  ranked: boolean;
  entry?: { state: string; total_points: number; rank: number; prize_kobo: number };
  message?: string;
}

const MEDALS: Record<number, string> = { 1: "🥇", 2: "🥈", 3: "🥉" };

function formatKobo(kobo: number): string {
  const naira = kobo / 100;
  return naira >= 1_000_000
    ? `₦${(naira / 1_000_000).toFixed(1)}M`
    : naira >= 1_000
    ? `₦${(naira / 1_000).toFixed(0)}K`
    : `₦${naira.toLocaleString()}`;
}

function formatPoints(pts: number): string {
  if (pts >= 1_000_000) return `${(pts / 1_000_000).toFixed(1)}M`;
  if (pts >= 1_000) return `${(pts / 1_000).toFixed(1)}K`;
  return String(pts);
}

function daysUntilEnd(period: string): number {
  if (!period) return 0;
  const [year, month] = period.split("-").map(Number);
  const endOfMonth = new Date(year, month, 1); // first of next month = end of this month
  const now = new Date();
  const diff = endOfMonth.getTime() - now.getTime();
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)));
}

// ─── No Active Event State ────────────────────────────────────────────────────

function NoActiveWar() {
  return (
    <div className="flex flex-col items-center justify-center py-20 px-6 text-center space-y-5">
      <motion.div
        animate={{ scale: [1, 1.05, 1], rotate: [0, -3, 3, 0] }}
        transition={{ duration: 3, repeat: Infinity, ease: "easeInOut" }}
        className="text-7xl"
      >
        🌍
      </motion.div>
      <div className="space-y-2">
        <h2 className="text-xl font-black text-white">No Active War Yet</h2>
        <p className="text-[rgb(130_140_180)] text-sm max-w-xs">
          Regional Wars kick off monthly. Keep recharging and building your points — 
          your state will need you when the battle begins!
        </p>
      </div>
      <div className="nexus-card px-6 py-4 text-center space-y-1 border border-brand-gold/20">
        <p className="text-brand-gold text-xs font-black uppercase tracking-widest">Watch out for the next event</p>
        <p className="text-white/60 text-xs">Admins announce new wars each month</p>
      </div>
      <div className="grid grid-cols-3 gap-3 w-full max-w-sm">
        {[
          { icon: <TrendingUp size={16} />, label: "Earn Points", sub: "Recharge to earn", color: "text-nexus-400" },
          { icon: <Users size={16} />, label: "Set State", sub: "In Settings", color: "text-green-400" },
          { icon: <Trophy size={16} />, label: "Win Prizes", sub: "When war starts", color: "text-brand-gold" },
        ].map(item => (
          <div key={item.label} className="nexus-card p-3 text-center">
            <div className={cn("flex justify-center mb-2", item.color)}>{item.icon}</div>
            <p className="text-white font-medium text-xs">{item.label}</p>
            <p className="text-[rgb(130_140_180)] text-xs">{item.sub}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function RegionalWarsPage() {
  const { data, error, isLoading } = useSWR<WarLeaderboardData>(
    "/wars/leaderboard",
    () => api.getWarsLeaderboard(37) as Promise<WarLeaderboardData>,
    { refreshInterval: 30_000 }
  );

  const { data: myRank } = useSWR<MyRankData>(
    "/wars/my-rank",
    () => api.getMyWarRank() as Promise<MyRankData>,
    { refreshInterval: 30_000 }
  );

  const leaderboard = data?.leaderboard ?? [];
  const period = data?.period ?? "";
  const daysLeft = daysUntilEnd(period);

  // Top-3 prize pool is sum of top 3 entries' prize_kobo
  const totalPrizeKobo = leaderboard.length > 0
    ? leaderboard.slice(0, 3).reduce((sum, e) => sum + (e.prize_kobo || 0), 0)
    : 0;

  const hasActiveWar = leaderboard.length > 0;

  return (
    <AppShell>
      <div className="max-w-2xl mx-auto px-4 py-6 space-y-5">
        {/* Header */}
        <div className="flex items-center gap-3">
          <Globe className="text-green-400" size={24} />
          <div>
            <h1 className="text-2xl font-bold font-display text-white">Regional Wars 🌍</h1>
            <p className="text-[rgb(130_140_180)] text-sm">States battle for monthly prize pools</p>
          </div>
        </div>

        {/* Loading */}
        {isLoading && (
          <div className="space-y-3">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="nexus-card p-4 animate-pulse h-14 rounded-2xl bg-white/5" />
            ))}
          </div>
        )}

        {/* Error */}
        {error && !isLoading && (
          <div className="nexus-card p-4 flex items-center gap-3 border border-red-500/20">
            <AlertCircle size={18} className="text-red-400 shrink-0" />
            <p className="text-red-400 text-sm">Could not load leaderboard. Pull to refresh.</p>
          </div>
        )}

        {/* No active war */}
        {!isLoading && !error && !hasActiveWar && <NoActiveWar />}

        {/* Active war content */}
        {!isLoading && hasActiveWar && (
          <>
            {/* Prize pool banner */}
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              className="rounded-2xl p-5 relative overflow-hidden"
              style={{ background: "linear-gradient(135deg, #10b981 0%, #059669 100%)" }}
            >
              <div className="absolute right-4 top-1/2 -translate-y-1/2 text-6xl opacity-20">🌍</div>
              <p className="text-white/80 text-sm uppercase tracking-wider mb-1">Monthly Prize Pool</p>
              <p className="text-4xl font-bold font-display text-white">
                {totalPrizeKobo > 0 ? formatKobo(totalPrizeKobo) : "TBA"}
              </p>
              <div className="flex items-center gap-2 mt-2">
                <Clock size={13} className="text-white/70" />
                <p className="text-white/70 text-sm">
                  {daysLeft > 0
                    ? `Top 3 states share the pool • Resets in ${daysLeft} day${daysLeft === 1 ? "" : "s"}`
                    : `Period: ${period}`}
                </p>
              </div>
            </motion.div>

            {/* My rank card */}
            {myRank?.ranked && myRank.entry && (
              <motion.div
                initial={{ opacity: 0, y: 5 }}
                animate={{ opacity: 1, y: 0 }}
                className="nexus-card p-4 flex items-center justify-between border border-nexus-500/30"
              >
                <div>
                  <p className="text-xs font-bold text-[rgb(130_140_180)] uppercase tracking-wider mb-0.5">Your State</p>
                  <p className="text-white font-bold">{myRank.entry.state}</p>
                  <p className="text-[rgb(130_140_180)] text-xs">{formatPoints(myRank.entry.total_points)} pts contributed</p>
                </div>
                <div className="text-right">
                  <p className="text-2xl font-black text-brand-gold">#{myRank.entry.rank}</p>
                  <p className="text-xs text-[rgb(130_140_180)]">
                    {myRank.entry.prize_kobo > 0 ? formatKobo(myRank.entry.prize_kobo) + " prize" : "Keep climbing"}
                  </p>
                </div>
              </motion.div>
            )}

            {myRank && !myRank.ranked && (
              <div className="nexus-card p-3 flex items-center gap-2 text-amber-400 border border-amber-500/20">
                <AlertCircle size={14} />
                <p className="text-xs">{myRank.message || "Set your state in Settings to join the war."}</p>
              </div>
            )}

            {/* How it works */}
            <div className="grid grid-cols-3 gap-3">
              {[
                { icon: <TrendingUp size={18} />, label: "Earn points", sub: "Recharge more", color: "text-nexus-400" },
                { icon: <Users size={18} />, label: "Team up", sub: "Your state rises", color: "text-green-400" },
                { icon: <Trophy size={18} />, label: "Win prizes", sub: "Monthly rewards", color: "text-brand-gold" },
              ].map(item => (
                <div key={item.label} className="nexus-card p-3 text-center">
                  <div className={cn("flex justify-center mb-2", item.color)}>{item.icon}</div>
                  <p className="text-white font-medium text-xs">{item.label}</p>
                  <p className="text-[rgb(130_140_180)] text-xs">{item.sub}</p>
                </div>
              ))}
            </div>

            {/* Leaderboard */}
            <div>
              <h2 className="text-white font-semibold mb-3 flex items-center gap-2">
                <Trophy size={16} className="text-brand-gold" />
                State Leaderboard
                <span className="text-xs text-[rgb(130_140_180)] font-normal ml-auto">Period: {period}</span>
              </h2>
              <div className="space-y-2">
                {leaderboard.map((row, i) => (
                  <motion.div
                    key={row.state}
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: i * 0.03 }}
                    className={cn(
                      "nexus-card p-3 flex items-center gap-3",
                      row.rank <= 3 && "border-brand-gold/30"
                    )}
                  >
                    <div className="w-8 text-center text-lg">
                      {MEDALS[row.rank] || <span className="text-sm font-bold text-[rgb(130_140_180)]">#{row.rank}</span>}
                    </div>
                    <div className="flex-1">
                      <p className="text-white font-semibold text-sm">{row.state}</p>
                      <p className="text-[rgb(130_140_180)] text-xs">{row.active_members.toLocaleString()} members</p>
                    </div>
                    <div className="text-right">
                      <p className="text-white font-bold text-sm">{formatPoints(row.total_points)} pts</p>
                      {row.prize_kobo > 0 && (
                        <p className="text-brand-gold text-xs font-semibold">{formatKobo(row.prize_kobo)}</p>
                      )}
                    </div>
                  </motion.div>
                ))}
              </div>
            </div>
          </>
        )}
      </div>
    </AppShell>
  );
}
