"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  Shield, Flame, Award, Star, ChevronRight, Download,
  Share2, QrCode, RefreshCw, Wallet, Smartphone,
  Trophy, Zap, Crown, Diamond, CheckCircle2, Clock,
} from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import AppShell from "@/components/layout/AppShell";
import { api, PassportData, WalletPassURLs, BadgeDefinition, PassportEvent } from "@/lib/api";
import { cn } from "@/lib/utils";

// ─── Tier configuration ───────────────────────────────────────────────────────

const TIER_CONFIG: Record<string, {
  label: string;
  icon: React.ElementType;
  gradient: string;
  glow: string;
  ring: string;
  badge: string;
  bgCard: string;
  textAccent: string;
}> = {
  BRONZE: {
    label: "Bronze",
    icon: Shield,
    gradient: "from-amber-700 via-amber-600 to-amber-500",
    glow: "shadow-amber-600/30",
    ring: "ring-amber-500/40",
    badge: "bg-amber-500/20 text-amber-400 border-amber-500/30",
    bgCard: "from-amber-900/40 to-amber-800/20",
    textAccent: "text-amber-400",
  },
  SILVER: {
    label: "Silver",
    icon: Star,
    gradient: "from-slate-500 via-slate-400 to-slate-300",
    glow: "shadow-slate-400/30",
    ring: "ring-slate-400/40",
    badge: "bg-slate-500/20 text-slate-300 border-slate-500/30",
    bgCard: "from-slate-800/40 to-slate-700/20",
    textAccent: "text-slate-300",
  },
  GOLD: {
    label: "Gold",
    icon: Trophy,
    gradient: "from-yellow-600 via-yellow-400 to-yellow-300",
    glow: "shadow-yellow-400/40",
    ring: "ring-yellow-400/50",
    badge: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
    bgCard: "from-yellow-900/40 to-yellow-800/20",
    textAccent: "text-yellow-400",
  },
  PLATINUM: {
    label: "Platinum",
    icon: Diamond,
    gradient: "from-purple-600 via-purple-400 to-indigo-300",
    glow: "shadow-purple-400/40",
    ring: "ring-purple-400/50",
    badge: "bg-purple-500/20 text-purple-300 border-purple-500/30",
    bgCard: "from-purple-900/40 to-indigo-900/20",
    textAccent: "text-purple-300",
  },
};

// ─── Badge display helpers ────────────────────────────────────────────────────

const BADGE_RARITY: Record<string, "common" | "rare" | "epic" | "legendary"> = {
  first_recharge: "common",
  streak_7:       "common",
  streak_30:      "rare",
  streak_90:      "epic",
  spin_first:     "common",
  spin_100:       "rare",
  studio_first:   "common",
  studio_50:      "rare",
  wars_top3:      "epic",
  silver_tier:    "common",
  gold_tier:      "rare",
  platinum_tier:  "legendary",
  big_winner:     "epic",
};

const RARITY_STYLES: Record<string, string> = {
  common:    "border-white/10 bg-white/5",
  rare:      "border-nexus-500/40 bg-nexus-900/30",
  epic:      "border-purple-500/40 bg-purple-900/20",
  legendary: "border-yellow-400/60 bg-yellow-900/20",
};

// ─── Event type display helpers ───────────────────────────────────────────────

const EVENT_META: Record<string, { label: string; icon: string; color: string }> = {
  tier_upgrade:      { label: "Tier Upgrade",      icon: "⬆️",  color: "text-yellow-400" },
  badge_earned:      { label: "Badge Earned",       icon: "🏅",  color: "text-nexus-400"  },
  streak_milestone:  { label: "Streak Milestone",   icon: "🔥",  color: "text-orange-400" },
  qr_scanned:        { label: "QR Scanned",         icon: "📲",  color: "text-green-400"  },
};

function getEventMeta(type: string) {
  return EVENT_META[type] ?? { label: type.replace(/_/g, " "), icon: "📌", color: "text-white/60" };
}

function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins  = Math.floor(diff / 60_000);
  const hours = Math.floor(diff / 3_600_000);
  const days  = Math.floor(diff / 86_400_000);
  if (mins < 1)   return "just now";
  if (mins < 60)  return `${mins}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7)   return `${days}d ago`;
  return new Date(iso).toLocaleDateString("en-GB", { day: "numeric", month: "short" });
}

// ─── Tier progress bar ────────────────────────────────────────────────────────

const TIER_THRESHOLDS: Record<string, number> = {
  BRONZE:   0,
  SILVER:   2000,
  GOLD:     10000,
  PLATINUM: 50000,
};

function getTierProgress(tier: string, lifetimePoints: number, nextTier: string): number {
  const currentMin = TIER_THRESHOLDS[tier] ?? 0;
  const nextMin    = TIER_THRESHOLDS[nextTier] ?? 0;
  if (!nextTier || nextMin === 0) return 100;
  const range = nextMin - currentMin;
  const earned = lifetimePoints - currentMin;
  return Math.min(100, Math.max(0, Math.round((earned / range) * 100)));
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function PassportPage() {
  const [passport, setPassport]       = useState<PassportData | null>(null);
  const [walletURLs, setWalletURLs]   = useState<WalletPassURLs | null>(null);
  const [qrPayload, setQrPayload]     = useState<string | null>(null);
  const [events, setEvents]           = useState<PassportEvent[] | null>(null);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [loading, setLoading]         = useState(true);
  const [qrLoading, setQrLoading]     = useState(false);
  const [walletLoading, setWalletLoading] = useState(false);
  const [error, setError]             = useState<string | null>(null);
  const [showQR, setShowQR]           = useState(false);
  const [copiedShare, setCopiedShare] = useState(false);
  const [activeTab, setActiveTab]     = useState<"overview" | "badges" | "activity">("overview");

  const loadPassport = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.getPassport();
      setPassport(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load passport");
    } finally {
      setLoading(false);
    }
  }, []);

  const loadWalletURLs = useCallback(async () => {
    try {
      setWalletLoading(true);
      const data = await api.getWalletPassURLs();
      setWalletURLs(data);
    } catch {
      // Non-fatal — wallet URLs are optional
    } finally {
      setWalletLoading(false);
    }
  }, []);

  // ── QR: fetch raw payload from backend, render inline SVG client-side ──
  const loadQR = useCallback(async () => {
    try {
      setQrLoading(true);
      const data = await api.getPassportQR();
      setQrPayload(data.qr_payload);
    } catch {
      // Non-fatal — QR is optional
    } finally {
      setQrLoading(false);
    }
  }, []);

  // ── Events: lazy-load when Activity tab is first opened ──────────────────────
  const loadEvents = useCallback(async () => {
    if (events !== null) return; // already loaded
    try {
      setEventsLoading(true);
      const res = await api.getPassportEvents(30);
      setEvents(res.events ?? []);
    } catch {
      setEvents([]);
    } finally {
      setEventsLoading(false);
    }
  }, [events]);

  useEffect(() => {
    loadPassport();
    loadWalletURLs();
  }, [loadPassport, loadWalletURLs]);

  // Load events when Activity tab is activated
  useEffect(() => {
    if (activeTab === "activity") loadEvents();
  }, [activeTab, loadEvents]);

  const handleShowQR = () => {
    setShowQR(true);
    if (!qrPayload) loadQR();
  };

  const handleShare = async () => {
    const text = `I'm a ${passport?.tier} member on Loyalty Nexus with ${passport?.lifetime_points?.toLocaleString()} Pulse Points! 🎯`;
    try {
      if (navigator.share) {
        await navigator.share({ title: "My Loyalty Nexus Passport", text });
      } else {
        await navigator.clipboard.writeText(text);
        setCopiedShare(true);
        setTimeout(() => setCopiedShare(false), 2000);
      }
    } catch {
      // ignore
    }
  };

  const handleAppleWallet = () => {
    window.location.href = api.getApplePKPassURL();
  };

  const handleGoogleWallet = () => {
    if (walletURLs?.google_wallet_url) {
      window.open(walletURLs.google_wallet_url, "_blank");
    }
  };

  if (loading) return <PassportSkeleton />;
  if (error || !passport) return <PassportError error={error} onRetry={loadPassport} />;

  const tierCfg = TIER_CONFIG[passport.tier] ?? TIER_CONFIG.BRONZE;
  const TierIcon = tierCfg.icon;
  const progress = getTierProgress(passport.tier, passport.lifetime_points, passport.next_tier);

  return (
    <AppShell>
      <div className="max-w-5xl mx-auto px-4 md:px-6 py-6 space-y-5">

        {/* ── Page header ─────────────────────────────────────────────────── */}
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          className="flex items-center justify-between"
        >
          <div>
            <h1 className="text-2xl font-display font-bold text-white">Digital Passport</h1>
            <p className="text-[rgb(130_140_180)] text-sm mt-0.5">Your loyalty identity on Loyalty Nexus</p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={handleShare}
              className="p-2 rounded-xl bg-white/5 hover:bg-white/10 text-[rgb(130_140_180)] hover:text-white transition-all"
              title="Share passport"
            >
              {copiedShare ? <CheckCircle2 size={18} className="text-green-400" /> : <Share2 size={18} />}
            </button>
            <button
              onClick={handleShowQR}
              className="p-2 rounded-xl bg-white/5 hover:bg-white/10 text-[rgb(130_140_180)] hover:text-white transition-all"
              title="Show QR code"
            >
              <QrCode size={18} />
            </button>
          </div>
        </motion.div>

        {/* ── Passport card ────────────────────────────────────────────────── */}
        <motion.div
          initial={{ opacity: 0, scale: 0.97 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.4 }}
          className={cn(
            "relative overflow-hidden rounded-2xl p-6 shadow-2xl",
            `shadow-${tierCfg.glow}`,
            `ring-1 ${tierCfg.ring}`
          )}
          style={{
            background: `linear-gradient(135deg, var(--card-bg-from), var(--card-bg-to))`,
          }}
        >
          {/* Gradient background */}
          <div
            className={cn(
              "absolute inset-0 bg-gradient-to-br opacity-20",
              tierCfg.gradient
            )}
          />
          {/* Shimmer effect */}
          <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/5 to-transparent -skew-x-12 animate-[shimmer_3s_ease-in-out_infinite]" />

          <div className="relative z-10">
            {/* Card header */}
            <div className="flex items-start justify-between mb-6">
              <div className="flex items-center gap-3">
                <div className={cn(
                  "p-3 rounded-2xl bg-gradient-to-br",
                  tierCfg.gradient,
                  "shadow-lg"
                )}>
                  <TierIcon size={24} className="text-white" />
                </div>
                <div>
                  <p className="text-white/60 text-xs uppercase tracking-widest font-medium">Loyalty Nexus</p>
                  <p className={cn("text-xl font-display font-bold", tierCfg.textAccent)}>
                    {tierCfg.label} Member
                  </p>
                </div>
              </div>
              <div className="text-right">
                <p className="text-white/40 text-xs">Member since</p>
                <p className="text-white/80 text-sm font-medium">
                  {new Date().getFullYear()}
                </p>
              </div>
            </div>

            {/* Points display */}
            <div className="mb-6">
              <p className="text-white/50 text-xs uppercase tracking-widest mb-1">Lifetime Pulse Points</p>
              <p className="text-4xl font-display font-bold text-white">
                {passport.lifetime_points.toLocaleString()}
                <span className="text-lg font-normal text-white/50 ml-2">pts</span>
              </p>
            </div>

            {/* Stats row */}
            <div className="grid grid-cols-3 gap-3 mb-6">
              <div className="text-center">
                <div className="flex items-center justify-center gap-1 mb-1">
                  <Flame size={14} className="text-orange-400" />
                  <span className="text-orange-400 font-bold text-lg">{passport.streak_count}</span>
                </div>
                <p className="text-white/40 text-xs">Day Streak</p>
              </div>
              <div className="text-center">
                <div className="flex items-center justify-center gap-1 mb-1">
                  <Award size={14} className={tierCfg.textAccent} />
                  <span className={cn("font-bold text-lg", tierCfg.textAccent)}>
                    {passport.badges.length}
                  </span>
                </div>
                <p className="text-white/40 text-xs">Badges</p>
              </div>
              <div className="text-center">
                <div className="flex items-center justify-center gap-1 mb-1">
                  <Zap size={14} className="text-nexus-400" />
                  <span className="text-nexus-400 font-bold text-lg">{passport.tier}</span>
                </div>
                <p className="text-white/40 text-xs">Tier</p>
              </div>
            </div>

            {/* Tier progress */}
            {passport.next_tier && (
              <div>
                <div className="flex justify-between text-xs text-white/50 mb-2">
                  <span>{tierCfg.label}</span>
                  <span>{passport.points_to_next_tier.toLocaleString()} pts to {passport.next_tier}</span>
                </div>
                <div className="h-1.5 bg-white/10 rounded-full overflow-hidden">
                  <motion.div
                    className={cn("h-full rounded-full bg-gradient-to-r", tierCfg.gradient)}
                    initial={{ width: 0 }}
                    animate={{ width: `${progress}%` }}
                    transition={{ duration: 1.2, delay: 0.3, ease: "easeOut" }}
                  />
                </div>
                <p className="text-white/30 text-xs mt-1 text-right">{progress}% to {passport.next_tier}</p>
              </div>
            )}
          </div>
        </motion.div>

        {/* ── Wallet buttons ───────────────────────────────────────────────── */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.15 }}
          className="grid grid-cols-2 gap-3"
        >
          {/* Apple Wallet */}
          <button
            onClick={handleAppleWallet}
            className="flex items-center justify-center gap-2 px-4 py-3 rounded-xl bg-black border border-white/10 hover:border-white/20 text-white font-medium text-sm transition-all active:scale-95"
          >
            <Smartphone size={16} />
            <span>Add to Apple Wallet</span>
          </button>

          {/* Google Wallet */}
          <button
            onClick={handleGoogleWallet}
            disabled={walletLoading || !walletURLs?.google_wallet_url}
            className={cn(
              "flex items-center justify-center gap-2 px-4 py-3 rounded-xl font-medium text-sm transition-all active:scale-95",
              walletURLs?.google_wallet_url
                ? "bg-[#1a73e8] hover:bg-[#1557b0] text-white border border-[#1a73e8]/50"
                : "bg-white/5 text-white/30 border border-white/5 cursor-not-allowed"
            )}
          >
            <Wallet size={16} />
            <span>{walletLoading ? "Loading..." : "Add to Google Wallet"}</span>
          </button>
        </motion.div>

        {/* ── Tabs ─────────────────────────────────────────────────────────── */}
        <div className="flex gap-1 p-1 bg-white/5 rounded-xl">
          <button
            onClick={() => setActiveTab("overview")}
            className={cn(
              "flex-1 py-2 rounded-lg text-sm font-medium transition-all",
              activeTab === "overview"
                ? "bg-nexus-600 text-white shadow-lg shadow-nexus-900/50"
                : "text-[rgb(130_140_180)] hover:text-white"
            )}
          >
            Overview
          </button>
          <button
            onClick={() => setActiveTab("badges")}
            className={cn(
              "flex-1 py-2 rounded-lg text-sm font-medium transition-all",
              activeTab === "badges"
                ? "bg-nexus-600 text-white shadow-lg shadow-nexus-900/50"
                : "text-[rgb(130_140_180)] hover:text-white"
            )}
          >
            Badges ({passport.badges.length})
          </button>
          <button
            onClick={() => setActiveTab("activity")}
            className={cn(
              "flex-1 py-2 rounded-lg text-sm font-medium transition-all",
              activeTab === "activity"
                ? "bg-nexus-600 text-white shadow-lg shadow-nexus-900/50"
                : "text-[rgb(130_140_180)] hover:text-white"
            )}
          >
            Activity
          </button>
        </div>

        {/* ── Tab content ──────────────────────────────────────────────────── */}
        <AnimatePresence mode="wait">
          {activeTab === "overview" && (
            <motion.div
              key="overview"
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.2 }}
              className="space-y-3"
            >
              {/* Streak card */}
              <StreakCard streak={passport.streak_count} />

              {/* Tier journey */}
              <TierJourneyCard
                currentTier={passport.tier}
                lifetimePoints={passport.lifetime_points}
              />

              {/* Recent badges preview */}
              {passport.badges.length > 0 && (
                <div className="nexus-card p-4">
                  <div className="flex items-center justify-between mb-3">
                    <p className="text-white font-semibold text-sm">Recent Badges</p>
                    <button
                      onClick={() => setActiveTab("badges")}
                      className="text-nexus-400 text-xs flex items-center gap-1 hover:text-nexus-300"
                    >
                      View all <ChevronRight size={12} />
                    </button>
                  </div>
                  <div className="flex gap-2 flex-wrap">
                    {passport.badges.slice(0, 5).map((badge) => (
                      <BadgePill key={badge.key} badge={badge} />
                    ))}
                    {passport.badges.length > 5 && (
                      <span className="text-[rgb(130_140_180)] text-xs self-center">
                        +{passport.badges.length - 5} more
                      </span>
                    )}
                  </div>
                </div>
              )}
            </motion.div>
          )}

          {activeTab === "badges" && (
            <motion.div
              key="badges"
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.2 }}
            >
              <BadgesGrid badges={passport.badges} />
            </motion.div>
          )}

          {activeTab === "activity" && (
            <motion.div
              key="activity"
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.2 }}
            >
              <ActivityFeed events={events} loading={eventsLoading} />
            </motion.div>
          )}
        </AnimatePresence>

      </div>

      {/* ── QR Modal ─────────────────────────────────────────────────────── */}
      <AnimatePresence>
        {showQR && (
          <QRModal
            qrPayload={qrPayload}
            loading={qrLoading}
            tier={passport.tier}
            onClose={() => setShowQR(false)}
          />
        )}
      </AnimatePresence>
    </AppShell>
  );
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function StreakCard({ streak }: { streak: number }) {
  const getStreakMessage = () => {
    if (streak === 0) return { msg: "Start your streak today!", color: "text-white/50", icon: "💤" };
    if (streak >= 90) return { msg: "Quarter King! Legendary dedication.", color: "text-purple-400", icon: "👑" };
    if (streak >= 30) return { msg: "Month Master! Incredible consistency.", color: "text-yellow-400", icon: "🏆" };
    if (streak >= 7)  return { msg: "Week Warrior! Keep the fire burning.", color: "text-orange-400", icon: "💪" };
    return { msg: `${7 - streak} more day${7 - streak !== 1 ? "s" : ""} to Week Warrior badge!`, color: "text-nexus-400", icon: "🔥" };
  };
  const { msg, color, icon } = getStreakMessage();

  return (
    <div className="nexus-card p-4 flex items-center gap-4">
      <div className="relative">
        <div className="w-14 h-14 rounded-2xl bg-orange-500/10 border border-orange-500/20 flex items-center justify-center text-2xl">
          {icon}
        </div>
        {streak > 0 && (
          <div className="absolute -top-1 -right-1 w-5 h-5 bg-orange-500 rounded-full flex items-center justify-center">
            <Flame size={10} className="text-white" />
          </div>
        )}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-baseline gap-2">
          <span className="text-3xl font-display font-bold text-white">{streak}</span>
          <span className="text-white/50 text-sm">day streak</span>
        </div>
        <p className={cn("text-sm mt-0.5", color)}>{msg}</p>
      </div>
    </div>
  );
}

function TierJourneyCard({ currentTier, lifetimePoints }: { currentTier: string; lifetimePoints: number }) {
  const tiers = ["BRONZE", "SILVER", "GOLD", "PLATINUM"];
  const currentIdx = tiers.indexOf(currentTier);

  return (
    <div className="nexus-card p-4">
      <p className="text-white font-semibold text-sm mb-4">Tier Journey</p>
      <div className="flex items-center gap-0">
        {tiers.map((tier, i) => {
          const cfg = TIER_CONFIG[tier];
          const TIcon = cfg.icon;
          const isReached  = i <= currentIdx;
          const isCurrent  = i === currentIdx;
          const threshold  = TIER_THRESHOLDS[tier];

          return (
            <div key={tier} className="flex items-center flex-1">
              <div className="flex flex-col items-center flex-1">
                <div className={cn(
                  "w-10 h-10 rounded-full flex items-center justify-center border-2 transition-all",
                  isCurrent
                    ? cn("bg-gradient-to-br", cfg.gradient, "border-transparent shadow-lg")
                    : isReached
                    ? "bg-white/10 border-white/20"
                    : "bg-white/5 border-white/10"
                )}>
                  <TIcon size={16} className={isCurrent ? "text-white" : isReached ? "text-white/60" : "text-white/20"} />
                </div>
                <p className={cn(
                  "text-xs mt-1 font-medium",
                  isCurrent ? cfg.textAccent : isReached ? "text-white/50" : "text-white/20"
                )}>
                  {cfg.label}
                </p>
                <p className="text-white/30 text-[10px]">
                  {threshold === 0 ? "Start" : `${(threshold / 1000).toFixed(0)}k`}
                </p>
              </div>
              {i < tiers.length - 1 && (
                <div className={cn(
                  "h-0.5 flex-1 mx-1 rounded-full",
                  i < currentIdx ? "bg-white/30" : "bg-white/10"
                )} />
              )}
            </div>
          );
        })}
      </div>
      <p className="text-center text-white/40 text-xs mt-3">
        {lifetimePoints.toLocaleString()} lifetime points earned
      </p>
    </div>
  );
}

function BadgePill({ badge }: { badge: BadgeDefinition }) {
  return (
    <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-white/5 border border-white/10">
      <span className="text-sm">{badge.icon}</span>
      <span className="text-white/70 text-xs font-medium">{badge.name}</span>
    </div>
  );
}

function BadgesGrid({ badges }: { badges: BadgeDefinition[] }) {
  if (badges.length === 0) {
    return (
      <div className="nexus-card p-8 text-center">
        <div className="text-4xl mb-3">🏅</div>
        <p className="text-white font-semibold mb-1">No badges yet</p>
        <p className="text-[rgb(130_140_180)] text-sm">
          Recharge daily, spin the wheel, and use AI Studio to earn your first badge.
        </p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-3">
      {badges.map((badge, i) => {
        const rarity = BADGE_RARITY[badge.key] ?? "common";
        return (
          <motion.div
            key={badge.key}
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: i * 0.04 }}
            className={cn(
              "nexus-card p-4 border",
              RARITY_STYLES[rarity]
            )}
          >
            <div className="text-3xl mb-2">{badge.icon}</div>
            <p className="text-white font-semibold text-sm">{badge.name}</p>
            <p className="text-[rgb(130_140_180)] text-xs mt-0.5 leading-relaxed">{badge.description}</p>
            <div className="mt-2">
              <span className={cn(
                "text-[10px] font-medium uppercase tracking-wider px-1.5 py-0.5 rounded",
                rarity === "legendary" ? "bg-yellow-500/20 text-yellow-400" :
                rarity === "epic"      ? "bg-purple-500/20 text-purple-400" :
                rarity === "rare"      ? "bg-nexus-500/20 text-nexus-400" :
                "bg-white/10 text-white/40"
              )}>
                {rarity}
              </span>
            </div>
          </motion.div>
        );
      })}
    </div>
  );
}

// ─── Activity Feed ────────────────────────────────────────────────────────────

function ActivityFeed({ events, loading }: { events: PassportEvent[] | null; loading: boolean }) {
  if (loading) {
    return (
      <div className="space-y-3">
        {[...Array(5)].map((_, i) => (
          <div key={i} className="nexus-card p-4 animate-pulse flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-white/5 flex-shrink-0" />
            <div className="flex-1 space-y-2">
              <div className="h-3 bg-white/5 rounded w-32" />
              <div className="h-2.5 bg-white/5 rounded w-20" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (!events || events.length === 0) {
    return (
      <div className="nexus-card p-8 text-center">
        <div className="text-4xl mb-3">📋</div>
        <p className="text-white font-semibold mb-1">No activity yet</p>
        <p className="text-[rgb(130_140_180)] text-sm">
          Your passport events — tier upgrades, badges earned, and QR scans — will appear here.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {events.map((event, i) => {
        const meta = getEventMeta(event.event_type);
        return (
          <motion.div
            key={event.id}
            initial={{ opacity: 0, x: -8 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: i * 0.03 }}
            className="nexus-card p-3.5 flex items-center gap-3"
          >
            <div className="w-10 h-10 rounded-full bg-white/5 border border-white/10 flex items-center justify-center text-xl flex-shrink-0">
              {meta.icon}
            </div>
            <div className="flex-1 min-w-0">
              <p className={cn("text-sm font-medium capitalize", meta.color)}>
                {meta.label}
              </p>
              {event.details && Object.keys(event.details).length > 0 && (
                <p className="text-white/40 text-xs mt-0.5 truncate">
                  {Object.entries(event.details)
                    .filter(([k]) => !["user_id"].includes(k))
                    .map(([k, v]) => `${k.replace(/_/g, " ")}: ${v}`)
                    .join(" · ")}
                </p>
              )}
            </div>
            <div className="flex items-center gap-1 text-white/30 text-xs flex-shrink-0">
              <Clock size={10} />
              <span>{formatRelativeTime(event.created_at)}</span>
            </div>
          </motion.div>
        );
      })}
    </div>
  );
}

// ─── QR Modal ─────────────────────────────────────────────────────────────────

function QRModal({
  qrPayload,
  loading,
  tier,
  onClose,
}: {
  qrPayload: string | null;
  loading: boolean;
  tier: string;
  onClose: () => void;
}) {
  const tierCfg = TIER_CONFIG[tier] ?? TIER_CONFIG.BRONZE;

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <motion.div
        initial={{ scale: 0.9, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        exit={{ scale: 0.9, opacity: 0 }}
        onClick={(e) => e.stopPropagation()}
        className="nexus-card p-6 max-w-xs w-full text-center"
      >
        <p className="text-white font-display font-bold text-lg mb-1">Your Passport QR</p>
        <p className="text-[rgb(130_140_180)] text-sm mb-4">Scan to verify your identity</p>

        <div className={cn(
          "w-48 h-48 mx-auto rounded-2xl flex items-center justify-center overflow-hidden",
          `ring-2 ${tierCfg.ring}`
        )}>
          {loading ? (
            <RefreshCw size={32} className="text-nexus-400 animate-spin" />
          ) : qrPayload ? (
            <QRCodeSVG
              value={qrPayload}
              size={192}
              bgColor="#ffffff"
              fgColor="#0f172a"
              level="M"
              className="rounded-xl"
            />
          ) : (
            <div className="text-center">
              <QrCode size={48} className="text-white/20 mx-auto mb-2" />
              <p className="text-white/40 text-xs">QR unavailable</p>
            </div>
          )}
        </div>

        <p className="text-white/40 text-xs mt-4">
          This QR code is unique to your account and expires in 5 minutes.
        </p>

        <button
          onClick={onClose}
          className="mt-4 w-full nexus-btn-outline py-2 text-sm"
        >
          Close
        </button>
      </motion.div>
    </motion.div>
  );
}

function PassportSkeleton() {
  return (
    <AppShell>
      <div className="max-w-5xl mx-auto px-4 md:px-6 py-6 space-y-5 animate-pulse">
        <div className="h-8 bg-white/5 rounded-xl w-48" />
        <div className="h-56 bg-white/5 rounded-2xl" />
        <div className="grid grid-cols-2 gap-3">
          <div className="h-12 bg-white/5 rounded-xl" />
          <div className="h-12 bg-white/5 rounded-xl" />
        </div>
        <div className="h-10 bg-white/5 rounded-xl" />
        <div className="h-32 bg-white/5 rounded-xl" />
        <div className="h-24 bg-white/5 rounded-xl" />
      </div>
    </AppShell>
  );
}

function PassportError({ error, onRetry }: { error: string | null; onRetry: () => void }) {
  return (
    <AppShell>
      <div className="max-w-5xl mx-auto px-4 md:px-6 py-6">
        <div className="nexus-card p-8 text-center">
          <div className="text-4xl mb-3">⚠️</div>
          <p className="text-white font-semibold mb-1">Could not load passport</p>
          <p className="text-[rgb(130_140_180)] text-sm mb-4">{error ?? "Please try again."}</p>
          <button onClick={onRetry} className="nexus-btn-primary px-6 py-2 text-sm">
            Retry
          </button>
        </div>
      </div>
    </AppShell>
  );
}
