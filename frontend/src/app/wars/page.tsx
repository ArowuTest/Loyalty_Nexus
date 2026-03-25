"use client";

import { motion } from "framer-motion";
import AppShell from "@/components/layout/AppShell";
import { Globe, Trophy, Users, TrendingUp } from "lucide-react";
import { cn } from "@/lib/utils";

const MOCK_LEADERBOARD = [
  { rank: 1,  state: "Lagos",     points: 48750, members: 12843, change: +2 },
  { rank: 2,  state: "Abuja",     points: 41200, members: 8941,  change: +1 },
  { rank: 3,  state: "Rivers",    points: 38600, members: 9220,  change: -1 },
  { rank: 4,  state: "Kano",      points: 35100, members: 11200, change: 0  },
  { rank: 5,  state: "Oyo",       points: 31500, members: 7800,  change: +3 },
  { rank: 6,  state: "Anambra",   points: 28900, members: 6500,  change: -2 },
  { rank: 7,  state: "Kaduna",    points: 26400, members: 7100,  change: 0  },
  { rank: 8,  state: "Delta",     points: 24100, members: 5800,  change: +1 },
];

const MEDALS: Record<number, string> = { 1: "🥇", 2: "🥈", 3: "🥉" };

export default function RegionalWarsPage() {
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

        {/* Prize pool banner */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          className="rounded-2xl p-5 relative overflow-hidden"
          style={{ background: "linear-gradient(135deg, #10b981 0%, #059669 100%)" }}
        >
          <div className="absolute right-4 top-1/2 -translate-y-1/2 text-6xl opacity-20">🌍</div>
          <p className="text-white/80 text-sm uppercase tracking-wider mb-1">Monthly Prize Pool</p>
          <p className="text-4xl font-bold font-display text-white">₦500,000</p>
          <p className="text-white/70 text-sm mt-2">Top 3 states share the pool • Resets in 14 days</p>
        </motion.div>

        {/* How it works */}
        <div className="grid grid-cols-3 gap-3">
          {[
            { icon: <TrendingUp size={18} />, label: "Earn points", sub: "Recharge more", color: "text-nexus-400" },
            { icon: <Users size={18} />, label: "Team up", sub: "Your state rises", color: "text-green-400" },
            { icon: <Trophy size={18} />, label: "Win prizes", sub: "Monthly rewards", color: "text-gold-400" },
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
            <Trophy size={16} className="text-gold-400" />
            State Leaderboard
          </h2>
          <div className="space-y-2">
            {MOCK_LEADERBOARD.map((row, i) => (
              <motion.div
                key={row.state}
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: i * 0.04 }}
                className={cn(
                  "nexus-card p-3 flex items-center gap-3",
                  row.rank <= 3 && "border-gold-500/30"
                )}
              >
                <div className="w-8 text-center text-lg">
                  {MEDALS[row.rank] || <span className="text-[rgb(130_140_180)] text-sm font-bold">{row.rank}</span>}
                </div>
                <div className="flex-1">
                  <p className="text-white font-semibold text-sm">{row.state}</p>
                  <p className="text-[rgb(130_140_180)] text-xs">{row.members.toLocaleString()} members</p>
                </div>
                <div className="text-right">
                  <p className="text-white font-bold text-sm">{row.points.toLocaleString()} pts</p>
                  <p className={cn("text-xs font-medium",
                    row.change > 0 ? "text-green-400" : row.change < 0 ? "text-red-400" : "text-[rgb(130_140_180)]"
                  )}>
                    {row.change > 0 ? `▲${row.change}` : row.change < 0 ? `▼${Math.abs(row.change)}` : "—"}
                  </p>
                </div>
              </motion.div>
            ))}
          </div>
        </div>
      </div>
    </AppShell>
  );
}
