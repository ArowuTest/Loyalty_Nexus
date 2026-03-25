"use client";

import { useMemo } from "react";
import useSWR from "swr";
import { motion } from "framer-motion";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import { Gift, Clock, CheckCircle, XCircle, RefreshCw } from "lucide-react";
import { cn } from "@/lib/utils";

// ── Types ──────────────────────────────────────────────────────────────────
interface SpinResult {
  id: string;
  prize_type: string;
  prize_value: number;
  fulfillment_status: string;
  created_at: string;
}

// ── Helpers ────────────────────────────────────────────────────────────────
function prizeLabel(item: SpinResult): string {
  switch (item.prize_type) {
    case "try_again":    return "Try Again";
    case "pulse_points": return `${item.prize_value} Pulse Points`;
    case "airtime":      return `₦${(item.prize_value / 100).toLocaleString()} Airtime`;
    case "data_bundle":  return "Data Bundle";
    case "momo_cash":    return `₦${(item.prize_value / 100).toLocaleString()} MoMo Cash`;
    default:             return item.prize_type;
  }
}

function prizeCategory(item: SpinResult): "won" | "pending" | "try_again" | "failed" {
  const s = item.fulfillment_status?.toLowerCase();
  if (item.prize_type === "try_again" || s === "n/a") return "try_again";
  if (s === "completed") return "won";
  if (s === "failed")    return "failed";
  return "pending";
}

const STATUS_STYLES: Record<string, string> = {
  won:       "text-green-400 bg-green-400/10",
  pending:   "text-yellow-400 bg-yellow-400/10",
  failed:    "text-red-400 bg-red-400/10",
  try_again: "text-white/25 bg-white/5",
};

const STATUS_ICON: Record<string, React.ReactNode> = {
  won:       <CheckCircle size={16} className="text-green-400" />,
  pending:   <Clock size={16} className="text-yellow-400" />,
  failed:    <XCircle size={16} className="text-red-400" />,
  try_again: <XCircle size={16} className="text-white/20" />,
};

const STATUS_LABEL: Record<string, string> = {
  won: "Credited", pending: "Pending", failed: "Failed", try_again: "Try Again",
};

// ── Page ───────────────────────────────────────────────────────────────────
export default function PrizesPage() {
  const { data, isLoading, mutate } = useSWR(
    "/spin/history",
    () => api.getSpinHistory() as Promise<{ history: SpinResult[] }>,
    { refreshInterval: 30000 }
  );
  const history: SpinResult[] = (data as any)?.history ?? [];

  // Computed stats
  const stats = useMemo(() => {
    const won = history.filter(h => h.prize_type !== "try_again" && h.fulfillment_status?.toLowerCase() === "completed");
    const pending = history.filter(h => h.prize_type !== "try_again" && ["pending","pending_momo"].includes(h.fulfillment_status?.toLowerCase()));
    const totalValueKobo = won.reduce((s, h) => s + h.prize_value, 0);
    return {
      totalWon: totalValueKobo >= 100 ? `₦${(totalValueKobo / 100).toLocaleString()}` : `${won.length} prizes`,
      pending:  pending.length,
      spinsUsed: history.length,
    };
  }, [history]);

  return (
    <AppShell>
      <div className="max-w-2xl mx-auto px-4 py-6 space-y-5">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Gift className="text-yellow-400" size={24} />
            <div>
              <h1 className="text-2xl font-bold font-display text-white">My Prizes</h1>
              <p className="text-white/40 text-sm">Your spin history and reward status</p>
            </div>
          </div>
          <button onClick={() => mutate()} className="p-2 text-white/30 hover:text-white/60 transition-colors rounded-lg hover:bg-white/5">
            <RefreshCw size={16} />
          </button>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-3">
          {[
            { label: "Total Won",  value: isLoading ? "…" : stats.totalWon,           color: "text-green-400" },
            { label: "Pending",    value: isLoading ? "…" : String(stats.pending),    color: "text-yellow-400" },
            { label: "Spins Used", value: isLoading ? "…" : String(stats.spinsUsed),  color: "text-nexus-400" },
          ].map(stat => (
            <div key={stat.label} className="nexus-card p-3 text-center">
              <p className={cn("text-xl font-bold font-display", stat.color)}>{stat.value}</p>
              <p className="text-white/30 text-xs mt-0.5">{stat.label}</p>
            </div>
          ))}
        </div>

        {/* List */}
        {isLoading ? (
          <div className="space-y-2">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="nexus-card h-14 animate-pulse" />
            ))}
          </div>
        ) : history.length === 0 ? (
          <div className="nexus-card p-10 text-center space-y-3">
            <Gift size={40} className="text-white/10 mx-auto" />
            <p className="text-white/40 font-medium">No spins yet</p>
            <p className="text-white/20 text-sm">Recharge your phone to earn spin credits and win prizes!</p>
          </div>
        ) : (
          <div className="space-y-2">
            {history.map((item, i) => {
              const cat = prizeCategory(item);
              return (
                <motion.div
                  key={item.id}
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: i * 0.04 }}
                  className="nexus-card px-4 py-3 flex items-center justify-between"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    {STATUS_ICON[cat]}
                    <div className="min-w-0">
                      <p className={cn("font-semibold text-sm truncate", cat === "try_again" ? "text-white/25" : "text-white")}>
                        {prizeLabel(item)}
                      </p>
                      <p className="text-white/25 text-xs">
                        {new Date(item.created_at).toLocaleDateString("en-NG", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}
                      </p>
                    </div>
                  </div>
                  <span className={cn("text-[10px] font-bold px-2.5 py-1 rounded-full flex-shrink-0", STATUS_STYLES[cat])}>
                    {STATUS_LABEL[cat]}
                  </span>
                </motion.div>
              );
            })}
          </div>
        )}
      </div>
    </AppShell>
  );
}
