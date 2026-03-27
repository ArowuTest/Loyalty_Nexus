"use client";
import { motion } from "framer-motion";
import useSWR from "swr";
import Link from "next/link";
import AppShell from "@/components/layout/AppShell";
import { useStore } from "@/store/useStore";
import api from "@/lib/api";
import { cn, formatPoints, TIER_THRESHOLDS } from "@/lib/utils";
import {
  Zap, Wand2, Trophy, ChevronRight, Flame, Swords,
  MapPin, Gift, ArrowRight, RotateCcw, Clock, Star,
} from "lucide-react";

const QUICK_ACTIONS = [
  { href: "/spin",   icon: RotateCcw, label: "Spin Wheel",    sub: "Use spin credits",   color: "bg-gold-500/15 text-gold-500",     border: "border-gold-500/20" },
  { href: "/studio", icon: Wand2,     label: "AI Studio",     sub: "30+ AI tools",       color: "bg-nexus-600/15 text-nexus-400",   border: "border-nexus-500/20" },
  { href: "/wars",   icon: Swords,    label: "Regional Wars", sub: "Your state battle",  color: "bg-green-600/15 text-green-400",   border: "border-green-500/20" },
  { href: "/prizes", icon: Trophy,    label: "My Prizes",     sub: "Claim rewards",      color: "bg-purple-600/15 text-purple-400", border: "border-purple-500/20" },
];

const TIER_COLORS: Record<string, string> = {
  BRONZE: "#CD7F32", SILVER: "#C0C0C0", GOLD: "#F5A623", PLATINUM: "#E5E4E2", DIAMOND: "#B9F2FF",
};
const TIER_ICONS: Record<string, string> = {
  BRONZE: "🥉", SILVER: "🥈", GOLD: "🥇", PLATINUM: "💎", DIAMOND: "💠",
};

const fetcher = (key: string) => {
  if (key === "/user/profile")     return api.getProfile();
  if (key === "/user/wallet")      return api.getWallet();
  if (key === "/user/bonus-pulse") return api.getBonusPulseAwards();
  if (key === "/wars/my-rank")     return api.getMyWarRank();
  if (key === "/wars/leaderboard") return api.getWarsLeaderboard(3);
  return Promise.resolve(null);
};

export default function DashboardPage() {
  const { setUser, setWallet } = useStore();

  const { data: profile }    = useSWR("/user/profile",     fetcher, { onSuccess: (d: unknown) => setUser(d as Parameters<typeof setUser>[0]) });
  const { data: wallet }     = useSWR("/user/wallet",      fetcher, { onSuccess: (d: unknown) => setWallet(d as Parameters<typeof setWallet>[0]) });
  const { data: bonusData }  = useSWR("/user/bonus-pulse", fetcher);
  const { data: myRankData } = useSWR("/wars/my-rank",     fetcher);
  const { data: lbData }     = useSWR("/wars/leaderboard", fetcher);

  const totalBonus  = (bonusData as { total_bonus?: number } | undefined)?.total_bonus || 0;
  const tierData    = TIER_THRESHOLDS.find(t => t.tier === (profile as { tier?: string } | undefined)?.tier) || TIER_THRESHOLDS[0];
  const nextTier    = TIER_THRESHOLDS[TIER_THRESHOLDS.indexOf(tierData) + 1];
  const progress    = nextTier
    ? Math.min(100, (((wallet as { lifetime_points?: number } | undefined)?.lifetime_points || 0) - tierData.min) / (nextTier.min - tierData.min) * 100)
    : 100;
  const tier        = (profile as { tier?: string } | undefined)?.tier?.toUpperCase() ?? "BRONZE";
  const tierColor   = TIER_COLORS[tier] ?? "#CD7F32";
  const tierIcon    = TIER_ICONS[tier]  ?? "🥉";
  const myRank      = myRankData as { ranked?: boolean; entry?: { state: string; total_points: number; rank: number; prize_kobo: number } } | undefined;
  const leaderboard = (lbData as { leaderboard?: Array<{ state: string; total_points: number; rank: number; prize_kobo: number }> } | undefined)?.leaderboard ?? [];

  return (
    <AppShell>
      <div className="max-w-2xl mx-auto px-4 py-6 space-y-5">

        {/* Welcome header */}
        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
          <div className="flex items-center justify-between mb-1">
            <div>
              <h1 className="text-2xl font-black text-white tracking-tight">Hey! 👋</h1>
              <p className="text-white/40 text-sm mt-0.5">
                {(profile as { phone_number?: string } | undefined)?.phone_number || "Loading…"}
              </p>
            </div>
            <div className="flex items-center gap-2">
              {(profile as { streak_count?: number } | undefined)?.streak_count ? (
                <div className="flex items-center gap-1.5 glass border border-white/[0.08] rounded-xl px-3 py-2">
                  <Flame size={15} className="text-orange-400" />
                  <span className="text-white font-black text-sm">{(profile as { streak_count?: number }).streak_count}d</span>
                  <span className="text-white/40 text-xs">streak</span>
                </div>
              ) : null}
              <div className="flex items-center gap-1.5 glass border border-white/[0.08] rounded-xl px-3 py-2">
                <span className="text-base">{tierIcon}</span>
                <span className="text-[12px] font-black" style={{ color: tierColor }}>{tier}</span>
              </div>
            </div>
          </div>
        </motion.div>

        {/* Points card */}
        <motion.div
          initial={{ opacity: 0, y: 14 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.07 }}
          className="relative rounded-2xl p-6 overflow-hidden"
          style={{
            background: "linear-gradient(135deg, #1a1c2e 0%, #0f1018 100%)",
            border: "1px solid rgba(245,166,35,0.18)",
            boxShadow: "0 0 40px rgba(245,166,35,0.07), inset 0 1px 0 rgba(255,255,255,0.05)",
          }}
        >
          <div className="absolute top-0 left-0 right-0 h-[2px]"
            style={{ background: "linear-gradient(to right, transparent, rgba(245,166,35,0.6), transparent)" }} />
          <div className="absolute top-0 right-0 w-48 h-48 rounded-full pointer-events-none"
            style={{ background: "radial-gradient(circle, rgba(245,166,35,0.08) 0%, transparent 70%)", transform: "translate(20%, -30%)" }} />
          <div className="relative">
            <div className="flex justify-between items-start mb-5">
              <div>
                <p className="text-white/40 text-[11px] font-black uppercase tracking-[0.18em] mb-1.5">Pulse Points</p>
                <p className="text-5xl font-black text-white tracking-tight">
                  {formatPoints((wallet as { pulse_points?: number } | undefined)?.pulse_points || 0)}
                </p>
              </div>
              <div className="flex flex-col items-end gap-1.5">
                <div className="flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-[11px] font-black"
                  style={{ background: `${tierColor}18`, border: `1px solid ${tierColor}30`, color: tierColor }}>
                  {tierIcon} {tier}
                </div>
                <Link href="/spin">
                  <button className="btn-gold rounded-xl h-8 px-4 text-[12px] font-black inline-flex items-center gap-1.5">
                    <Zap className="w-3.5 h-3.5" /> Spin
                  </button>
                </Link>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3 mb-4">
              <div className="rounded-xl px-3 py-2.5" style={{ background: "rgba(255,255,255,0.05)" }}>
                <p className="text-white/40 text-[10px] font-bold uppercase tracking-wider mb-0.5">Spin Credits</p>
                <p className="text-white font-black text-xl">{(wallet as { spin_credits?: number } | undefined)?.spin_credits || 0}</p>
              </div>
              <div className="rounded-xl px-3 py-2.5" style={{ background: "rgba(255,255,255,0.05)" }}>
                <p className="text-white/40 text-[10px] font-bold uppercase tracking-wider mb-0.5">Lifetime Pts</p>
                <p className="text-white font-black text-xl">{formatPoints((wallet as { lifetime_points?: number } | undefined)?.lifetime_points || 0)}</p>
              </div>
            </div>
            {totalBonus > 0 && (
              <div className="rounded-xl px-3 py-2.5 mb-4 flex items-center justify-between"
                style={{ background: "rgba(245,166,35,0.08)", border: "1px solid rgba(245,166,35,0.15)" }}>
                <div>
                  <p className="text-white/40 text-[10px] font-bold uppercase tracking-wider mb-0.5">🎁 Bonus Awards</p>
                  <p className="font-black text-xl" style={{ color: "var(--gold)" }}>{formatPoints(totalBonus)}</p>
                </div>
                <Link href="/pulse-awards" className="text-xs underline underline-offset-2 hover:opacity-80 transition-opacity" style={{ color: "var(--gold)" }}>
                  View history
                </Link>
              </div>
            )}
            {nextTier && (
              <div>
                <div className="flex justify-between text-[11px] text-white/40 mb-1.5">
                  <span className="font-bold">{tierData.label}</span>
                  <span>{formatPoints(nextTier.min - ((wallet as { lifetime_points?: number } | undefined)?.lifetime_points || 0))} to {nextTier.label}</span>
                </div>
                <div className="h-1.5 rounded-full overflow-hidden" style={{ background: "rgba(255,255,255,0.08)" }}>
                  <motion.div
                    className="h-full rounded-full"
                    style={{ background: `linear-gradient(to right, ${tierColor}, ${tierColor}cc)` }}
                    initial={{ width: 0 }} animate={{ width: `${progress}%` }}
                    transition={{ duration: 1.2, delay: 0.4, ease: "easeOut" }}
                  />
                </div>
              </div>
            )}
          </div>
        </motion.div>

        {/* Quick actions */}
        <div className="grid grid-cols-2 gap-3">
          {QUICK_ACTIONS.map((action, i) => (
            <motion.div key={action.href} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.12 + i * 0.05 }}>
              <Link href={action.href}
                className={cn("glass rounded-2xl p-4 flex items-start gap-3 hover:border-white/[0.18] transition-all block border", action.border)}>
                <div className={cn("p-2 rounded-xl flex-shrink-0", action.color)}>
                  <action.icon size={17} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-white font-black text-sm truncate">{action.label}</p>
                  <p className="text-white/40 text-[11px] mt-0.5">{action.sub}</p>
                </div>
                <ChevronRight size={13} className="text-white/25 mt-0.5 flex-shrink-0" />
              </Link>
            </motion.div>
          ))}
        </div>

        {/* Regional Wars widget */}
        <motion.div
          initial={{ opacity: 0, y: 14 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.35 }}
          className="relative rounded-2xl overflow-hidden"
          style={{ background: "linear-gradient(135deg, #0f1a12 0%, #0d0e14 100%)", border: "1px solid rgba(16,185,129,0.18)" }}
        >
          <div className="absolute top-0 left-0 right-0 h-[2px]"
            style={{ background: "linear-gradient(to right, transparent, rgba(16,185,129,0.5), transparent)" }} />
          <div className="p-5">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2.5">
                <div className="w-9 h-9 rounded-xl flex items-center justify-center"
                  style={{ background: "rgba(16,185,129,0.12)", border: "1px solid rgba(16,185,129,0.25)" }}>
                  <Swords size={18} className="text-green-400" />
                </div>
                <div>
                  <h3 className="text-[14px] font-black text-white leading-none">Regional Wars</h3>
                  <p className="text-[11px] text-white/40 mt-0.5">₦500K monthly prize pool</p>
                </div>
              </div>
              <Link href="/wars">
                <button className="text-[11px] font-black text-green-400 flex items-center gap-1 hover:text-green-300 transition-colors">
                  View all <ArrowRight className="w-3 h-3" />
                </button>
              </Link>
            </div>
            {myRank?.ranked && myRank.entry ? (
              <div className="rounded-xl p-3.5 mb-3 flex items-center justify-between"
                style={{ background: "rgba(16,185,129,0.08)", border: "1px solid rgba(16,185,129,0.15)" }}>
                <div className="flex items-center gap-2.5">
                  <MapPin className="w-4 h-4 text-green-400 flex-shrink-0" />
                  <div>
                    <p className="text-[13px] font-black text-white">{myRank.entry.state}</p>
                    <p className="text-[11px] text-white/40">{formatPoints(myRank.entry.total_points)} pts</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-[11px] text-white/40 mb-0.5">Your rank</p>
                  <p className="text-xl font-black text-green-400">#{myRank.entry.rank}</p>
                </div>
              </div>
            ) : (
              <div className="rounded-xl p-3.5 mb-3 flex items-center gap-2.5"
                style={{ background: "rgba(16,185,129,0.05)", border: "1px solid rgba(16,185,129,0.10)" }}>
                <MapPin className="w-4 h-4 text-green-400/50 flex-shrink-0" />
                <p className="text-[12px] text-white/40">Recharge to earn points and join your state&apos;s battle</p>
              </div>
            )}
            {leaderboard.length > 0 && (
              <div className="space-y-2">
                {leaderboard.slice(0, 3).map((entry, i) => {
                  const medals = ["🥇", "🥈", "🥉"];
                  const colors = ["#F5A623", "#C0C0C0", "#CD7F32"];
                  const prize  = entry.prize_kobo > 0 ? `₦${(entry.prize_kobo / 100).toLocaleString("en-NG")}` : null;
                  return (
                    <div key={entry.state} className="flex items-center gap-3 rounded-xl px-3 py-2.5"
                      style={{ background: "rgba(255,255,255,0.03)" }}>
                      <span className="text-base w-5 text-center flex-shrink-0">{medals[i]}</span>
                      <div className="flex-1 min-w-0">
                        <p className="text-[13px] font-black text-white truncate">{entry.state}</p>
                        <p className="text-[10px] text-white/35 font-mono">{formatPoints(entry.total_points)} pts</p>
                      </div>
                      {prize && <span className="text-[11px] font-black flex-shrink-0" style={{ color: colors[i] }}>{prize}</span>}
                    </div>
                  );
                })}
              </div>
            )}
            <div className="mt-3 rounded-xl p-3 flex items-start gap-2.5"
              style={{ background: "rgba(245,166,35,0.05)", border: "1px solid rgba(245,166,35,0.12)" }}>
              <Gift className="w-3.5 h-3.5 flex-shrink-0 mt-0.5" style={{ color: "var(--gold)" }} />
              <p className="text-[11px] text-white/40 leading-relaxed">
                <strong style={{ color: "var(--gold)" }}>Individual draw:</strong> One random member from each top-3 state wins a personal MoMo cash payout at month end.
              </p>
            </div>
          </div>
        </motion.div>

        {/* Coming Soon: Daily & Weekly Draws */}
        <motion.div initial={{ opacity: 0, y: 14 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.42 }}
          className="grid grid-cols-2 gap-3">
          {[
            { icon: Clock, color: "#00D4FF", title: "Daily Draw",     body: "Win prizes daily just for being active." },
            { icon: Star,  color: "#8B5CF6", title: "Weekly Jackpot", body: "Bigger prizes for top rechargees." },
          ].map(({ icon: Icon, color, title, body }) => (
            <div key={title} className="glass rounded-2xl border border-white/[0.07] p-4 opacity-75">
              <div className="flex items-start justify-between mb-2.5">
                <div className="w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0"
                  style={{ background: `${color}12`, border: `1px solid ${color}20`, color }}>
                  <Icon size={15} />
                </div>
                <span className="text-[9px] font-black uppercase tracking-wider px-2 py-0.5 rounded-full"
                  style={{ background: `${color}10`, color, border: `1px solid ${color}20` }}>
                  Soon
                </span>
              </div>
              <h4 className="text-[13px] font-black text-white mb-1">{title}</h4>
              <p className="text-[11px] text-white/35 leading-relaxed">{body}</p>
            </div>
          ))}
        </motion.div>

        {/* Recharge CTA */}
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.5 }}
          className="relative rounded-2xl p-4 flex items-center justify-between overflow-hidden"
          style={{ background: "linear-gradient(135deg, rgba(245,166,35,0.08) 0%, rgba(245,166,35,0.03) 100%)", border: "1px solid rgba(245,166,35,0.18)" }}>
          <div>
            <p className="text-white font-black text-sm">Recharge to earn more ⚡</p>
            <p className="text-white/40 text-[12px] mt-0.5">₦200 = 1 Pulse Point · ₦1,000+ = free spin</p>
          </div>
          <Link href="/recharge">
            <button className="btn-gold rounded-xl h-9 px-4 text-[12px] font-black inline-flex items-center gap-1.5 flex-shrink-0">
              <Zap className="w-3.5 h-3.5" /> Recharge
            </button>
          </Link>
        </motion.div>

      </div>
    </AppShell>
  );
}
