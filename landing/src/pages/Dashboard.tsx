import React, { useState } from "react";
import { Link } from "react-router-dom";
import { motion } from "framer-motion";
import {
  Zap, Sparkles, Trophy, Users, TrendingUp, RotateCcw,
  Copy, Share2, CheckCircle2, Clock, ArrowRight, Crown,
  Wallet, Phone, Star, Gift, ChevronRight
} from "lucide-react";
import NavBar from "@/components/NavBar";
import Footer from "@/components/Footer";
import { TIER_CONFIG, formatPoints, formatNaira, ROUTES, type Tier } from "@/lib";
import { MOCK_USER, SPIN_PRIZES, AI_TOOLS, ADMIN_STATS } from "@/data";
import { AnimatePresence } from "framer-motion";

// ─── Mini components ──────────────────────────────────────────
function StatCard({ label, value, icon: Icon, color, sub }: { label: string; value: string; icon: React.ElementType; color: string; sub?: string }) {
  return (
    <div className="glass rounded-2xl p-5 border border-white/[0.07] hover:border-white/[0.14] transition-all duration-250 flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <div className="p-2 rounded-lg" style={{ background: `${color}18` }}>
          <Icon className="w-4 h-4" style={{ color }} />
        </div>
        <span className="text-xs text-muted-foreground font-semibold">{label}</span>
      </div>
      <p className="text-2xl font-black" style={{ color }}>{value}</p>
      {sub && <p className="text-[11px] text-muted-foreground">{sub}</p>}
    </div>
  );
}

// ─── SpinWheel full interactive ───────────────────────────────
function SpinSection() {
  const [rotation, setRotation] = useState(0);
  const [spinning, setSpinning] = useState(false);
  const [lastWin,  setLastWin]  = useState<string | null>(null);
  const [spinsLeft, setSpinsLeft] = useState(MOCK_USER.spins_today);

  const spin = () => {
    if (spinning || spinsLeft < 1) return;
    setLastWin(null);
    setSpinning(true);
    const extra = 1440 + Math.floor(Math.random() * 360);
    setRotation(r => r + extra);
    setTimeout(() => {
      setSpinning(false);
      setSpinsLeft(s => s - 1);
      // Determine winner by final angle
      const normalized = ((rotation + extra) % 360);
      const segAngle = 360 / SPIN_PRIZES.length;
      const idx = Math.floor(((360 - (normalized % 360)) / segAngle)) % SPIN_PRIZES.length;
      setLastWin(SPIN_PRIZES[idx]?.label ?? SPIN_PRIZES[0].label);
    }, 3400);
  };

  const prizes = SPIN_PRIZES.slice(0, 8);
  const segAngle = 360 / prizes.length;

  return (
    <div className="glass rounded-3xl border border-white/[0.09] overflow-hidden">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-0">
        {/* Left wheel */}
        <div
          className="relative flex flex-col items-center justify-center gap-6 p-8 sm:p-10"
          style={{ background: "radial-gradient(ellipse at center, rgba(245,166,35,0.07) 0%, transparent 70%)" }}
        >
          {/* Spins badge */}
          <div className="flex items-center gap-2 glass-gold rounded-full px-4 py-1.5 border border-primary/25">
            <RotateCcw className="w-3.5 h-3.5 text-primary" />
            <span className="text-[12px] font-black text-primary">{spinsLeft} free spin{spinsLeft !== 1 ? "s" : ""} remaining</span>
          </div>

          {/* Wheel */}
          <div className="relative w-56 h-56 sm:w-64 sm:h-64">
            <div
              className="absolute -inset-2 rounded-full pointer-events-none opacity-20"
              style={{ background: "conic-gradient(from 0deg, #F5A623, #FFE066, #00D4FF, #8B5CF6, #F5A623)", filter: "blur(12px)" }}
            />
            <div
              className="w-full h-full rounded-full overflow-hidden border-4 border-primary/20 relative"
              style={{
                transform: `rotate(${rotation}deg)`,
                transition: spinning ? "transform 3.4s cubic-bezier(0.17, 0.67, 0.21, 1)" : "none",
                boxShadow: "0 0 40px rgba(245,166,35,0.2)",
              }}
            >
              {prizes.map((p, i) => {
                const start = i * segAngle;
                const mid   = start + segAngle / 2;
                const rad   = (mid * Math.PI) / 180;
                const tx    = 50 + 37 * Math.cos(rad - Math.PI / 2);
                const ty    = 50 + 37 * Math.sin(rad - Math.PI / 2);
                return (
                  <div key={p.id} className="absolute inset-0" style={{
                    background: `conic-gradient(from ${start}deg at 50% 50%, ${p.color}dd ${start}deg ${start + segAngle}deg, transparent ${start + segAngle}deg)`,
                  }}>
                    <span className="absolute text-[9px] font-black text-white pointer-events-none"
                      style={{ left:`${tx}%`, top:`${ty}%`, transform:`translate(-50%,-50%) rotate(${mid}deg)`, textShadow:"0 1px 3px rgba(0,0,0,0.8)", maxWidth:40, textAlign:"center" }}>
                      {p.label}
                    </span>
                  </div>
                );
              })}
              <div className="absolute inset-[28%] rounded-full bg-surface-0 border-4 border-primary flex items-center justify-center z-10">
                <Zap className="w-5 h-5 text-primary" />
              </div>
            </div>
            {/* Pointer */}
            <div className="absolute top-1/2 right-0 translate-x-3 -translate-y-1/2 z-20">
              <div className="w-5 h-5 glow-gold" style={{ clipPath:"polygon(100% 50%, 0% 0%, 0% 100%)", background:"linear-gradient(135deg, #F5A623, #FFE066)" }} />
            </div>
          </div>

          <button
            onClick={spin}
            disabled={spinning || spinsLeft < 1}
            className={`rounded-2xl h-12 px-8 text-[14px] font-black inline-flex items-center gap-2 transition-all duration-200 ${
              spinsLeft < 1 ? "glass border border-white/10 text-muted-foreground cursor-not-allowed opacity-50"
              : spinning    ? "glass border border-white/10 text-muted-foreground cursor-wait"
              : "btn-gold glow-gold"
            }`}
          >
            <RotateCcw className={`w-4 h-4 ${spinning ? "animate-spin" : ""}`} />
            {spinsLeft < 1 ? "No spins left today" : spinning ? "Spinning…" : "Spin Now!"}
          </button>

          <AnimatePresence>
            {lastWin && (
              <motion.div
                initial={{ opacity:0, scale:0.85, y:10 }}
                animate={{ opacity:1, scale:1, y:0 }}
                exit={{ opacity:0 }}
                className="glass-gold rounded-2xl px-6 py-3 text-center border border-primary/30"
              >
                <p className="text-2xl font-black text-primary">🎉 {lastWin}!</p>
                <p className="text-[11px] text-muted-foreground mt-1">Added to your wallet</p>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Right — prize table */}
        <div className="p-7 flex flex-col justify-between">
          <div>
            <h3 className="text-xl font-black text-foreground mb-1">Today's Prize Pool</h3>
            <p className="text-sm text-muted-foreground mb-5">Every spin wins something. Higher tier = bigger prizes.</p>
            <div className="space-y-2.5">
              {prizes.map(p => (
                <div key={p.id} className="flex items-center gap-3">
                  <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ background: p.color }} />
                  <span className="text-[13px] font-semibold text-foreground flex-1">{p.label}</span>
                  <span className="text-[11px] text-muted-foreground capitalize font-mono">{p.type}</span>
                </div>
              ))}
            </div>
          </div>
          <div className="mt-6 glass rounded-xl p-4 border border-primary/15">
            <p className="text-[12px] text-muted-foreground leading-relaxed">
              <span className="text-primary font-bold">Tip:</span> Reach Gold tier for 3× prize multiplier.
              Refer friends to earn bonus spins instantly.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Transactions ─────────────────────────────────────────────
const TRANSACTIONS = [
  { type:"earn",  label:"MTN Recharge — ₦1,000",   pts:"+1,000", time:"2h ago",   icon:"📱" },
  { type:"earn",  label:"Daily Spin — Won ₦500",   pts:"+250",   time:"2h ago",   icon:"🎯" },
  { type:"earn",  label:"Referral — Emeka joined", pts:"+500",   time:"1d ago",   icon:"👥" },
  { type:"spend", label:"AI Photo x2",             pts:"-20",    time:"1d ago",   icon:"📸" },
  { type:"earn",  label:"MTN Recharge — ₦500",    pts:"+500",   time:"3d ago",   icon:"📱" },
  { type:"spend", label:"Business Plan AI",        pts:"-30",    time:"4d ago",   icon:"💼" },
];

/* ═══════════════════════════════════════════════════════════
   DASHBOARD
══════════════════════════════════════════════════════════ */
export default function Dashboard() {
  const [authOpen]   = useState(false);
  const [copied, setCopied] = useState(false);
  const user  = MOCK_USER;
  const tier  = user.tier as Tier;
  const tc    = TIER_CONFIG[tier];
  const tiers = Object.entries(TIER_CONFIG) as [Tier, typeof TIER_CONFIG[Tier]][];
  const tierKeys = tiers.map(([k]) => k);
  const tierIdx  = tierKeys.indexOf(tier);
  const nextTier = tiers[tierIdx + 1];
  const progressPct = nextTier
    ? Math.min(100, Math.round((user.pulse_points / nextTier[1].minPoints) * 100))
    : 100;

  const copyRef = () => {
    navigator.clipboard.writeText(user.referral_code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="min-h-screen bg-surface-0 dark">
      <NavBar isLoggedIn={true} />

      <div className="pt-20 max-w-6xl mx-auto px-4 sm:px-6 pb-24 space-y-8">

        {/* ── Welcome banner ── */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ type: "spring", stiffness: 260, damping: 28 }}
          className="glass-gold rounded-3xl p-6 sm:p-8 border border-primary/20 relative overflow-hidden"
          style={{ background: "radial-gradient(ellipse 80% 80% at 0% 50%, rgba(245,166,35,0.10) 0%, transparent 70%)" }}
        >
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-5">
            <div className="flex items-center gap-4">
              <div
                className="w-14 h-14 rounded-2xl flex items-center justify-center text-2xl font-black text-black flex-shrink-0 glow-gold"
                style={{ background: `linear-gradient(135deg, ${tc.color}, #F5A623)` }}
              >
                {user.display_name[0]}
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Welcome back,</p>
                <h2 className="text-2xl font-black text-foreground">{user.display_name}</h2>
                <div className="flex items-center gap-2 mt-0.5">
                  <span className="text-base">{tc.icon}</span>
                  <span className="text-[12px] font-bold" style={{ color: tc.color }}>{tc.label} Tier</span>
                </div>
              </div>
            </div>
            <div className="flex flex-wrap gap-3">
              <div className="glass rounded-xl px-4 py-3 border border-white/[0.09] text-center">
                <p className="text-2xl font-black text-primary">{formatPoints(user.pulse_points)}</p>
                <p className="text-[11px] text-muted-foreground">Pulse Points</p>
              </div>
              <div className="glass rounded-xl px-4 py-3 border border-white/[0.09] text-center">
                <p className="text-2xl font-black text-chart-3">{user.spins_today}</p>
                <p className="text-[11px] text-muted-foreground">Spins Today</p>
              </div>
              <div className="glass rounded-xl px-4 py-3 border border-white/[0.09] text-center">
                <p className="text-2xl font-black" style={{ color: "#00D4FF" }}>{formatNaira(user.total_earned)}</p>
                <p className="text-[11px] text-muted-foreground">Total Earned</p>
              </div>
            </div>
          </div>

          {/* Tier progress */}
          {nextTier && (
            <div className="mt-5 pt-5 border-t border-white/[0.08]">
              <div className="flex items-center justify-between mb-2">
                <p className="text-[12px] text-muted-foreground">
                  Progress to <span className="font-bold" style={{ color: nextTier[1].color }}>{nextTier[1].label}</span>
                </p>
                <p className="text-[12px] font-mono text-muted-foreground">{formatPoints(user.pulse_points)} / {formatPoints(nextTier[1].minPoints)}</p>
              </div>
              <div className="h-2 bg-white/[0.06] rounded-full overflow-hidden">
                <motion.div
                  initial={{ width: 0 }}
                  animate={{ width: `${progressPct}%` }}
                  transition={{ duration: 1, delay: 0.3, ease: "easeOut" }}
                  className="h-full rounded-full"
                  style={{ background: `linear-gradient(to right, ${tc.color}, ${nextTier[1].color})` }}
                />
              </div>
            </div>
          )}
        </motion.div>

        {/* ── Stats row ── */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
          <StatCard label="This Month"    value="₦1,450"  icon={Wallet}     color="#F5A623"   sub="Cash prizes won" />
          <StatCard label="AI Creations"  value="24"      icon={Sparkles}   color="#00D4FF"   sub="This month" />
          <StatCard label="Referrals"     value="7"       icon={Users}      color="#10B981"   sub="+3,500 bonus pts" />
          <StatCard label="Total Recharge" value="₦12,400" icon={Phone}     color="#8B5CF6"  sub="Since joined" />
        </div>

        {/* ── Spin & Win ── */}
        <div>
          <div className="flex items-center gap-2 mb-4">
            <div className="w-6 h-6 rounded-lg bg-gold flex items-center justify-center">
              <RotateCcw className="w-3.5 h-3.5 text-black" />
            </div>
            <h2 className="text-lg font-black text-foreground">Daily Spin & Win</h2>
          </div>
          <SpinSection />
        </div>

        {/* ── Tier ladder + AI tools ── */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">

          {/* Tier ladder */}
          <div className="glass rounded-2xl border border-white/[0.07]">
            <div className="p-5 border-b border-white/[0.07]">
              <h3 className="font-black text-sm text-foreground">Your Loyalty Ladder</h3>
            </div>
            <div className="p-4 space-y-2">
              {tiers.map(([key, t]) => {
                const isActive = key === tier;
                const isUnlocked = tierKeys.indexOf(key) <= tierIdx;
                return (
                  <div
                    key={key}
                    className={`flex items-center gap-3 rounded-xl px-4 py-3 transition-all ${
                      isActive ? "border" : "opacity-60 hover:opacity-80"
                    }`}
                    style={isActive ? {
                      background: `${t.color}12`,
                      borderColor: `${t.color}30`,
                    } : {}}
                  >
                    <span className="text-xl w-6 text-center">{t.icon}</span>
                    <div className="flex-1">
                      <p className="text-[13px] font-bold" style={{ color: isUnlocked ? t.color : undefined }}>{t.label}</p>
                      <p className="text-[10px] text-muted-foreground font-mono">
                        {key === "bronze" ? "Starting tier" : `${formatPoints(t.minPoints)} pts`}
                      </p>
                    </div>
                    {isActive && (
                      <span className="text-[10px] font-black px-2 py-0.5 rounded-full text-black" style={{ background: t.color }}>CURRENT</span>
                    )}
                    {isUnlocked && !isActive && (
                      <CheckCircle2 className="w-4 h-4 flex-shrink-0" style={{ color: t.color }} />
                    )}
                  </div>
                );
              })}
            </div>
          </div>

          {/* Quick AI tools */}
          <div className="glass rounded-2xl border border-white/[0.07]">
            <div className="p-5 border-b border-white/[0.07] flex items-center justify-between">
              <h3 className="font-black text-sm text-foreground">Quick AI Tools</h3>
              <Link to={ROUTES.STUDIO} className="text-[12px] text-primary hover:underline flex items-center gap-1">
                All tools <ChevronRight className="w-3.5 h-3.5" />
              </Link>
            </div>
            <div className="p-4 grid grid-cols-2 gap-2.5">
              {AI_TOOLS.slice(0, 6).map(tool => (
                <Link to={ROUTES.STUDIO} key={tool.slug}>
                  <div className="glass rounded-xl p-3.5 border border-white/[0.07] hover:border-white/[0.18] cursor-pointer transition-all hover:scale-[1.02]">
                    <span className="text-xl block mb-2">{tool.emoji}</span>
                    <p className="text-[12px] font-bold text-foreground leading-tight mb-1">{tool.name}</p>
                    <p className="text-[10px] font-mono" style={{ color: tool.is_free ? "#00D4FF" : "#F5A623" }}>
                      {tool.is_free ? "FREE" : `${tool.point_cost} pts`}
                    </p>
                  </div>
                </Link>
              ))}
            </div>
          </div>
        </div>

        {/* ── Referral + Transactions ── */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Referral */}
          <div className="glass rounded-2xl border border-white/[0.07] overflow-hidden">
            <div
              className="p-6 relative"
              style={{ background: "radial-gradient(ellipse 80% 80% at 0% 0%, rgba(16,185,129,0.08) 0%, transparent 70%)" }}
            >
              <div className="flex items-center gap-2 mb-3">
                <Users className="w-5 h-5 text-chart-3" />
                <h3 className="font-black text-sm text-foreground">Refer & Earn</h3>
              </div>
              <p className="text-sm text-muted-foreground mb-4 leading-relaxed">
                Earn <span className="text-chart-3 font-bold">500 Pulse Points</span> for every friend you refer.
                They get 100 bonus points when they sign up.
              </p>
              <div className="flex items-center gap-2 mb-5">
                <div className="flex-1 glass rounded-xl px-4 py-2.5 border border-white/[0.09] font-mono text-[13px] font-bold text-primary">
                  {user.referral_code}
                </div>
                <button
                  onClick={copyRef}
                  className={`h-10 px-4 rounded-xl text-[12px] font-bold transition-all duration-200 flex items-center gap-1.5 ${
                    copied ? "bg-chart-3 text-black" : "btn-gold text-black"
                  }`}
                >
                  {copied ? <><CheckCircle2 className="w-3.5 h-3.5" /> Copied!</> : <><Copy className="w-3.5 h-3.5" /> Copy</>}
                </button>
              </div>
              <div className="grid grid-cols-3 gap-3 text-center">
                {[
                  { label:"Friends Referred", val:"7",      color:"#F5A623" },
                  { label:"Points Earned",    val:"3,500",  color:"#10B981" },
                  { label:"Friends Earning",  val:"5",      color:"#00D4FF" },
                ].map(({ label, val, color }) => (
                  <div key={label} className="glass rounded-xl p-3 border border-white/[0.07]">
                    <p className="text-xl font-black" style={{ color }}>{val}</p>
                    <p className="text-[10px] text-muted-foreground mt-0.5 leading-tight">{label}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Transactions */}
          <div className="glass rounded-2xl border border-white/[0.07]">
            <div className="p-5 border-b border-white/[0.07]">
              <h3 className="font-black text-sm text-foreground">Recent Activity</h3>
            </div>
            <div className="divide-y divide-white/[0.05]">
              {TRANSACTIONS.map((tx, i) => (
                <div key={i} className="flex items-center gap-3 px-5 py-3">
                  <span className="text-lg w-8 text-center">{tx.icon}</span>
                  <div className="flex-1 min-w-0">
                    <p className="text-[13px] font-semibold text-foreground truncate">{tx.label}</p>
                    <p className="text-[11px] text-muted-foreground">{tx.time}</p>
                  </div>
                  <span
                    className={`text-[13px] font-black font-mono ${tx.type === "earn" ? "text-chart-3" : "text-destructive"}`}
                  >
                    {tx.pts} pts
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      <Footer />
    </div>
  );
}
