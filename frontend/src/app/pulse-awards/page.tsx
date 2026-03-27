"use client";

import useSWR from "swr";
import { motion } from "framer-motion";
import AppShell from "@/components/layout/AppShell";
import api, { BonusPulseAward } from "@/lib/api";
import { formatPoints } from "@/lib/utils";
import { Gift } from "lucide-react";

const fetcher = () => api.getBonusPulseAwards();

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-NG", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function PulseAwardsPage() {
  const { data, isLoading } = useSWR("/user/bonus-pulse", fetcher);
  const totalBonus = data?.total_bonus || 0;
  const awards: BonusPulseAward[] = data?.awards || [];

  return (
    <AppShell>
      <div className="max-w-2xl mx-auto px-4 py-6 space-y-5">
        {/* Header */}
        <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
          <div className="flex items-center gap-3 mb-1">
            <div className="p-2 rounded-xl bg-nexus-600/20 text-nexus-400">
              <Gift size={20} />
            </div>
            <div>
              <h1 className="text-2xl font-bold font-display text-white">Bonus Awards</h1>
              <p className="text-[rgb(130_140_180)] text-sm">Pulse Points awarded by Nexus campaigns</p>
            </div>
          </div>
        </motion.div>

        {/* Total bonus stat */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.05 }}
          className="relative overflow-hidden rounded-2xl p-5"
          style={{
            background: "linear-gradient(135deg, rgb(74,86,238) 0%, rgb(139,92,246) 60%, rgb(249,199,79,0.3) 100%)",
          }}
        >
          <div className="absolute inset-0 opacity-20 pointer-events-none"
            style={{ backgroundImage: "radial-gradient(circle at 80% 20%, white 0%, transparent 50%)" }}
          />
          <div className="relative">
            <p className="text-white/70 text-xs mb-1 uppercase tracking-widest">Total Bonus Points Received</p>
            <p className="text-4xl font-bold font-display text-white">{formatPoints(totalBonus)}</p>
            <p className="text-white/60 text-xs mt-1">{awards.length} award{awards.length !== 1 ? "s" : ""}</p>
          </div>
        </motion.div>

        {/* Award history list */}
        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map(i => (
              <div key={i} className="nexus-card p-4 animate-pulse">
                <div className="h-4 bg-white/10 rounded w-1/3 mb-2" />
                <div className="h-3 bg-white/5 rounded w-2/3" />
              </div>
            ))}
          </div>
        ) : awards.length === 0 ? (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.2 }}
            className="nexus-card p-8 text-center"
          >
            <Gift size={40} className="mx-auto text-[rgb(130_140_180)] mb-3" />
            <p className="text-white font-semibold mb-1">No bonus awards yet</p>
            <p className="text-[rgb(130_140_180)] text-sm">
              Bonus Pulse Points from campaigns and promotions will appear here.
            </p>
          </motion.div>
        ) : (
          <div className="space-y-3">
            {awards.map((award, i) => (
              <motion.div
                key={award.id}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.1 + i * 0.04 }}
                className="nexus-card p-4 flex items-start justify-between gap-4"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-white font-semibold text-sm truncate">
                      {award.campaign || "Campaign Award"}
                    </span>
                  </div>
                  {award.note && (
                    <p className="text-[rgb(130_140_180)] text-xs truncate">{award.note}</p>
                  )}
                  <p className="text-[rgb(130_140_180)] text-xs mt-1">{formatDate(award.created_at)}</p>
                </div>
                <div className="text-right flex-shrink-0">
                  <p className="text-nexus-400 font-bold text-lg">+{formatPoints(award.points)}</p>
                  <p className="text-[rgb(130_140_180)] text-xs">pts</p>
                </div>
              </motion.div>
            ))}
          </div>
        )}
      </div>
    </AppShell>
  );
}
