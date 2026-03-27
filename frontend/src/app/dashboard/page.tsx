"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import useSWR from "swr";
import Link from "next/link";
import AppShell from "@/components/layout/AppShell";
import { useStore } from "@/store/useStore";
import api from "@/lib/api";
import { cn, formatPoints, TIER_THRESHOLDS } from "@/lib/utils";
import { Zap, Wand2, Globe, TrendingUp, ChevronRight, Flame } from "lucide-react";

const QUICK_ACTIONS = [
  { href: "/spin",     icon: Zap,        label: "Spin Now",     sub: "Use credits",   color: "bg-nexus-600/20 text-nexus-400" },
  { href: "/studio",   icon: Wand2,      label: "AI Studio",    sub: "17 free tools", color: "bg-purple-600/20 text-purple-400" },
  { href: "/wars",     icon: Globe,      label: "Regional Wars",sub: "Your state rank",color: "bg-green-600/20 text-green-400" },
  { href: "/prizes",   icon: TrendingUp, label: "My Prizes",    sub: "Claim rewards",  color: "bg-gold-500/20 text-gold-400" },
];

const fetcher = (key: string) => {
  if (key === "/user/profile") return api.getProfile();
  if (key === "/user/wallet") return api.getWallet();
  if (key === "/user/bonus-pulse") return api.getBonusPulseAwards();
  return Promise.resolve(null);
};

export default function DashboardPage() {
  const { setUser, setWallet, user } = useStore();
  const { data: profile } = useSWR("/user/profile", fetcher, { onSuccess: (d: unknown) => setUser(d as Parameters<typeof setUser>[0]) });
  const { data: wallet } = useSWR("/user/wallet", fetcher, { onSuccess: (d: unknown) => setWallet(d as Parameters<typeof setWallet>[0]) });
  const { data: bonusData } = useSWR("/user/bonus-pulse", fetcher);
  const totalBonus = (bonusData as { total_bonus?: number } | undefined)?.total_bonus || 0;

  const tierData = TIER_THRESHOLDS.find(t => t.tier === (profile as { tier?: string } | undefined)?.tier) || TIER_THRESHOLDS[0];
  const nextTier = TIER_THRESHOLDS[TIER_THRESHOLDS.indexOf(tierData) + 1];
  const progress = nextTier
    ? Math.min(100, (((wallet as { lifetime_points?: number } | undefined)?.lifetime_points || 0) - tierData.min) / (nextTier.min - tierData.min) * 100)
    : 100;

  return (
    <AppShell>
      <div className="max-w-2xl mx-auto px-4 py-6 space-y-5">
        {/* Welcome */}
        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
          <div className="flex items-center justify-between mb-1">
            <div>
              <h1 className="text-2xl font-bold font-display text-white">
                Hey! 👋
              </h1>
              <p className="text-[rgb(130_140_180)] text-sm">
                {(profile as { phone_number?: string } | undefined)?.phone_number || "Loading…"}
              </p>
            </div>
            {(profile as { streak_count?: number } | undefined)?.streak_count ? (
              <div className="flex items-center gap-1.5 nexus-card px-3 py-2">
                <Flame size={16} className="text-orange-400" />
                <span className="text-white font-bold">{(profile as { streak_count?: number }).streak_count}d</span>
                <span className="text-[rgb(130_140_180)] text-xs">streak</span>
              </div>
            ) : null}
          </div>
        </motion.div>

        {/* Wallet card */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="relative overflow-hidden rounded-2xl p-5"
          style={{
            background: "linear-gradient(135deg, rgb(74,86,238) 0%, rgb(139,92,246) 60%, rgb(249,199,79,0.3) 100%)",
          }}
        >
          <div className="absolute inset-0 opacity-20 pointer-events-none"
            style={{ backgroundImage: "radial-gradient(circle at 80% 20%, white 0%, transparent 50%)" }}
          />
          <div className="relative">
            <div className="flex justify-between items-start mb-4">
              <div>
                <p className="text-white/70 text-xs mb-1 uppercase tracking-widest">Pulse Points</p>
                <p className="text-4xl font-bold font-display text-white">
                  {formatPoints((wallet as { pulse_points?: number } | undefined)?.pulse_points || 0)}
                </p>
              </div>
              <div className="text-right">
                <span className={cn("tier-badge", `tier-${(profile as { tier?: string } | undefined)?.tier || "BRONZE"}`)}>
                  {(profile as { tier?: string } | undefined)?.tier || "BRONZE"}
                </span>
              </div>
            </div>
            <div className="flex gap-4 mb-4">
              <div className="bg-white/10 rounded-xl px-3 py-2 flex-1">
                <p className="text-white/70 text-xs">Spin Credits</p>
                <p className="text-white font-bold text-lg">{(wallet as { spin_credits?: number } | undefined)?.spin_credits || 0}</p>
              </div>
              <div className="bg-white/10 rounded-xl px-3 py-2 flex-1">
                <p className="text-white/70 text-xs">Lifetime Pts</p>
                <p className="text-white font-bold text-lg">{formatPoints((wallet as { lifetime_points?: number } | undefined)?.lifetime_points || 0)}</p>
              </div>
            </div>
            {/* Bonus awards row — only shown when the user has received bonus points */}
            {totalBonus > 0 && (
              <div className="bg-white/10 rounded-xl px-3 py-2 mb-4 flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-xs">🎁 Bonus Awards</p>
                  <p className="text-white font-bold text-lg">{formatPoints(totalBonus)}</p>
                </div>
                <Link
                  href="/pulse-awards"
                  className="text-white/60 text-xs underline underline-offset-2 hover:text-white transition-colors"
                >
                  View history
                </Link>
              </div>
            )}
            {/* Tier progress */}
            {nextTier && (
              <div>
                <div className="flex justify-between text-xs text-white/70 mb-1">
                  <span>{tierData.label}</span>
                  <span>{nextTier.label} in {formatPoints(nextTier.min - ((wallet as { lifetime_points?: number } | undefined)?.lifetime_points || 0))}</span>
                </div>
                <div className="h-1.5 bg-white/20 rounded-full overflow-hidden">
                  <motion.div
                    className="h-full bg-white rounded-full"
                    initial={{ width: 0 }}
                    animate={{ width: `${progress}%` }}
                    transition={{ duration: 1, delay: 0.5 }}
                  />
                </div>
              </div>
            )}
          </div>
        </motion.div>

        {/* Quick actions */}
        <div className="grid grid-cols-2 gap-3">
          {QUICK_ACTIONS.map((action, i) => (
            <motion.div
              key={action.href}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.15 + i * 0.05 }}
            >
              <Link href={action.href} className="nexus-card p-4 flex items-start gap-3 hover:border-nexus-500/30 transition-all block">
                <div className={cn("p-2 rounded-xl", action.color)}>
                  <action.icon size={18} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-white font-medium text-sm truncate">{action.label}</p>
                  <p className="text-[rgb(130_140_180)] text-xs">{action.sub}</p>
                </div>
                <ChevronRight size={14} className="text-[rgb(130_140_180)] mt-0.5 flex-shrink-0" />
              </Link>
            </motion.div>
          ))}
        </div>

        {/* Recharge CTA */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.4 }}
          className="nexus-card p-4 flex items-center justify-between"
        >
          <div>
            <p className="text-white font-semibold">Recharge to earn more ⚡</p>
            <p className="text-[rgb(130_140_180)] text-sm">₦200+ gets you 2 Pulse Points & 1 Spin Credit</p>
          </div>
          <Link href="/recharge" className="nexus-btn-primary text-sm px-4 py-2">
            Recharge
          </Link>
        </motion.div>
      </div>
    </AppShell>
  );
}
