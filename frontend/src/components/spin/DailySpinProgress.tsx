"use client";

/**
 * DailySpinProgress
 *
 * Mirrors the RechargeMax DailySpinProgress component.
 * Shows:
 *  - Current spin tier badge (Bronze / Silver / Gold / Platinum)
 *  - Today's cumulative recharge amount
 *  - Progress bar toward the next tier
 *  - Spins used today vs daily cap
 *  - Upgrade nudge when daily cap is reached or no credits remain
 *  - Full tier overview grid (all 4 tiers + their spin counts)
 *
 * Data comes from GET /api/v1/spin/eligibility which now returns
 * current_tier_name, today_amount_naira, progress_percent, and nudge fields.
 */

import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Zap, ChevronDown, ChevronUp, TrendingUp, Info } from "lucide-react";
import { cn } from "@/lib/utils";
import api from "@/lib/api";

// ─── Tier config ─────────────────────────────────────────────────────────────

interface TierConfig {
  key: string;
  label: string;
  emoji: string;
  color: string;
  minNaira: number;
  spinsPerDay: number;
  description: string;
}

const TIERS: TierConfig[] = [
  {
    key:         "Bronze",
    label:       "Bronze",
    emoji:       "🥉",
    color:       "#CD7F32",
    minNaira:    1000,
    spinsPerDay: 1,
    description: "Recharge ₦1,000+",
  },
  {
    key:         "Silver",
    label:       "Silver",
    emoji:       "🥈",
    color:       "#C0C0C0",
    minNaira:    5000,
    spinsPerDay: 2,
    description: "Recharge ₦5,000+",
  },
  {
    key:         "Gold",
    label:       "Gold",
    emoji:       "🥇",
    color:       "#FFD700",
    minNaira:    10000,
    spinsPerDay: 3,
    description: "Recharge ₦10,000+",
  },
  {
    key:         "Platinum",
    label:       "Platinum",
    emoji:       "💎",
    color:       "#E5E4E2",
    minNaira:    20000,
    spinsPerDay: 5,
    description: "Recharge ₦20,000+",
  },
];

function getTierConfig(tierName: string): TierConfig | null {
  if (!tierName) return null;
  return TIERS.find(t => t.key.toLowerCase() === tierName.toLowerCase()) ?? null;
}

// ─── Types ───────────────────────────────────────────────────────────────────

interface EligibilityData {
  eligible: boolean;
  available_spins: number;
  spins_used_today: number;
  max_spins_today: number;
  spin_credits: number;
  message: string;
  current_tier_name: string;
  today_amount_naira: number;
  progress_percent: number;
  trigger_naira?: number;
  next_tier_name?: string;
  next_tier_min_amount?: number;
  amount_to_next_tier?: number;
  next_tier_spins?: number;
}

// ─── Component ───────────────────────────────────────────────────────────────

interface DailySpinProgressProps {
  /** Called after a successful spin so this component can refresh */
  refreshKey?: number;
  className?: string;
}

export default function DailySpinProgress({ refreshKey = 0, className }: DailySpinProgressProps) {
  const [data, setData]         = useState<EligibilityData | null>(null);
  const [loading, setLoading]   = useState(true);
  const [showTiers, setShowTiers] = useState(false);

  useEffect(() => {
    setLoading(true);
    api.getSpinEligibility()
      .then(d => setData(d))
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [refreshKey]);

  if (loading) {
    return (
      <div className={cn("nexus-card p-4 animate-pulse", className)}>
        <div className="h-3 w-32 bg-white/10 rounded mb-3" />
        <div className="h-2 w-full bg-white/10 rounded" />
      </div>
    );
  }

  if (!data) return null;

  const tierCfg = getTierConfig(data.current_tier_name);
  const tierColor = tierCfg?.color ?? "#5f72f9";
  const tierEmoji = tierCfg?.emoji ?? "⚡";
  const tierLabel = tierCfg?.label ?? (data.current_tier_name || "No Tier");

  const hasActiveTier  = !!tierCfg;
  const capReached     = data.spins_used_today >= data.max_spins_today && data.max_spins_today > 0;
  const noCredits      = data.spin_credits < 1;
  const showNudge      = (capReached || noCredits) && !!data.next_tier_name;

  return (
    <div className={cn("nexus-card overflow-hidden", className)}>
      {/* Tier colour accent bar */}
      <div className="h-1 w-full" style={{ background: `linear-gradient(to right, ${tierColor}88, ${tierColor})` }} />

      <div className="p-4 space-y-3">
        {/* Header row */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-xl">{tierEmoji}</span>
            <div>
              <p className="text-white font-bold text-sm leading-tight">
                {hasActiveTier ? `${tierLabel} Tier` : "No Tier Yet"}
              </p>
              <p className="text-white/40 text-[10px]">
                {hasActiveTier
                  ? `${data.spins_used_today}/${data.max_spins_today} spins used today`
                  : "Recharge ₦1,000+ to unlock spins"}
              </p>
            </div>
          </div>
          {/* Today's recharge badge */}
          <div className="text-right">
            <p className="text-white font-bold text-sm">
              ₦{data.today_amount_naira.toLocaleString("en-NG", { maximumFractionDigits: 0 })}
            </p>
            <p className="text-white/30 text-[10px]">today's recharge</p>
          </div>
        </div>

        {/* Progress bar */}
        <div>
          <div className="flex justify-between text-[10px] text-white/40 mb-1">
            <span>{hasActiveTier ? tierLabel : "₦0"}</span>
            <span>
              {data.next_tier_name
                ? `${data.next_tier_name} (₦${((data.next_tier_min_amount ?? 0) / 100).toLocaleString()})`
                : "Max Tier 🏆"}
            </span>
          </div>
          <div className="h-2 bg-white/10 rounded-full overflow-hidden">
            <motion.div
              className="h-full rounded-full"
              style={{ background: `linear-gradient(to right, ${tierColor}88, ${tierColor})` }}
              initial={{ width: 0 }}
              animate={{ width: `${Math.min(100, data.progress_percent)}%` }}
              transition={{ duration: 0.8, ease: "easeOut" }}
            />
          </div>
        </div>

        {/* Upgrade nudge */}
        <AnimatePresence>
          {showNudge && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              className="flex items-start gap-2 bg-nexus-500/10 border border-nexus-500/20 rounded-xl p-3"
            >
              <TrendingUp size={14} className="text-nexus-400 mt-0.5 flex-shrink-0" />
              <p className="text-nexus-200/80 text-xs leading-relaxed">
                {capReached
                  ? <>Daily cap reached. Recharge <span className="text-nexus-300 font-semibold">₦{((data.amount_to_next_tier ?? 0) / 100).toLocaleString()} more</span> to unlock <span className="text-nexus-300 font-semibold">{data.next_tier_name}</span> tier and get <span className="text-nexus-300 font-semibold">{data.next_tier_spins} spins/day</span>!</>
                  : <>Recharge <span className="text-nexus-300 font-semibold">₦{((data.amount_to_next_tier ?? 0) / 100).toLocaleString()} more</span> to unlock <span className="text-nexus-300 font-semibold">{data.next_tier_name}</span> tier and earn <span className="text-nexus-300 font-semibold">{data.next_tier_spins} spins/day</span>!</>
                }
              </p>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Tier overview toggle */}
        <button
          onClick={() => setShowTiers(v => !v)}
          className="w-full flex items-center justify-between text-white/40 text-xs hover:text-white/60 transition-colors py-1"
        >
          <span className="flex items-center gap-1.5">
            <Info size={12} /> All spin tiers
          </span>
          {showTiers ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
        </button>

        <AnimatePresence>
          {showTiers && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              className="overflow-hidden"
            >
              <div className="grid grid-cols-2 gap-2 pt-1">
                {TIERS.map(tier => {
                  const isCurrent = tier.key.toLowerCase() === (data.current_tier_name ?? "").toLowerCase();
                  return (
                    <div
                      key={tier.key}
                      className={cn(
                        "rounded-xl p-2.5 border transition-all",
                        isCurrent
                          ? "border-opacity-60"
                          : "bg-white/5 border-white/10"
                      )}
                      style={isCurrent ? {
                        background: `${tier.color}15`,
                        borderColor: `${tier.color}50`,
                      } : {}}
                    >
                      <div className="flex items-center gap-1.5 mb-1">
                        <span className="text-sm">{tier.emoji}</span>
                        <span
                          className="text-xs font-bold"
                          style={{ color: isCurrent ? tier.color : undefined }}
                        >
                          {tier.label}
                        </span>
                        {isCurrent && (
                          <span
                            className="text-[9px] font-bold px-1.5 py-0.5 rounded-full ml-auto"
                            style={{ background: `${tier.color}25`, color: tier.color }}
                          >
                            YOU
                          </span>
                        )}
                      </div>
                      <p className="text-white/40 text-[10px]">{tier.description}</p>
                      <div className="flex items-center gap-1 mt-1.5">
                        <Zap size={10} className="text-yellow-400" />
                        <span className="text-white/70 text-[11px] font-semibold">
                          {tier.spinsPerDay} spin{tier.spinsPerDay > 1 ? "s" : ""}/day
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
