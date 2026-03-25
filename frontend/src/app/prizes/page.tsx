"use client";

import { motion } from "framer-motion";
import AppShell from "@/components/layout/AppShell";
import { Gift, Clock, CheckCircle, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";

const STATUS_STYLES: Record<string, string> = {
  won:      "text-green-400 bg-green-400/10",
  pending:  "text-yellow-400 bg-yellow-400/10",
  failed:   "text-red-400 bg-red-400/10",
  try_again:"text-[rgb(130_140_180)] bg-white/5",
};

const MOCK_PRIZES = [
  { id: "1", label: "₦500 Airtime", type: "airtime", status: "won",      date: "Today, 10:32am", ref: "VT-AA123" },
  { id: "2", label: "100 Points",   type: "points",  status: "won",      date: "Today, 9:15am",  ref: "PTS-001" },
  { id: "3", label: "₦1,000 Data",  type: "data",    status: "pending",  date: "Yesterday",      ref: "VT-BB456" },
  { id: "4", label: "Try Again",    type: "no_prize",status: "try_again",date: "Yesterday",      ref: "—" },
  { id: "5", label: "₦2,000 MoMo", type: "momo",    status: "pending",  date: "2 days ago",     ref: "MM-CC789" },
];

const STATUS_ICON: Record<string, React.ReactNode> = {
  won:      <CheckCircle size={16} className="text-green-400" />,
  pending:  <Clock size={16} className="text-yellow-400" />,
  failed:   <XCircle size={16} className="text-red-400" />,
  try_again:<XCircle size={16} className="text-[rgb(130_140_180)]" />,
};

export default function PrizesPage() {
  return (
    <AppShell>
      <div className="max-w-2xl mx-auto px-4 py-6 space-y-5">
        <div className="flex items-center gap-3">
          <Gift className="text-gold-400" size={24} />
          <div>
            <h1 className="text-2xl font-bold font-display text-white">My Prizes</h1>
            <p className="text-[rgb(130_140_180)] text-sm">Your spin history and reward status</p>
          </div>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-3 gap-3">
          {[
            { label: "Total Won",   value: "₦3,500",  color: "text-green-400" },
            { label: "Pending",     value: "2",        color: "text-yellow-400" },
            { label: "Spins Used",  value: "8",        color: "text-nexus-400" },
          ].map(stat => (
            <div key={stat.label} className="nexus-card p-3 text-center">
              <p className={cn("text-xl font-bold font-display", stat.color)}>{stat.value}</p>
              <p className="text-[rgb(130_140_180)] text-xs mt-0.5">{stat.label}</p>
            </div>
          ))}
        </div>

        {/* Prize list */}
        <div className="space-y-2">
          {MOCK_PRIZES.map((prize, i) => (
            <motion.div
              key={prize.id}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.05 }}
              className="nexus-card p-4 flex items-center gap-3"
            >
              {STATUS_ICON[prize.status]}
              <div className="flex-1">
                <p className="text-white font-medium text-sm">{prize.label}</p>
                <p className="text-[rgb(130_140_180)] text-xs">{prize.date} · Ref: {prize.ref}</p>
              </div>
              <span className={cn("text-xs font-semibold px-2 py-0.5 rounded-full capitalize", STATUS_STYLES[prize.status])}>
                {prize.status.replace("_", " ")}
              </span>
            </motion.div>
          ))}
        </div>
      </div>
    </AppShell>
  );
}
