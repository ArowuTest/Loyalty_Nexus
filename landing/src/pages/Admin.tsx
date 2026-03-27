import React, { useState } from "react";
import { Link } from "react-router-dom";
import { motion } from "framer-motion";
import { Zap, Users, Sparkles, TrendingUp, Activity, Settings, Database, Shield, ChevronRight, ArrowUpRight, RefreshCw, CheckCircle2, AlertCircle, Clock } from "lucide-react";
import { formatPoints, formatNaira } from "@/lib";
import { ADMIN_STATS, AI_TOOLS } from "@/data";

const NAV = [
  { icon: Activity,   label: "Overview",    key: "overview" },
  { icon: Users,      label: "Users",       key: "users" },
  { icon: Sparkles,   label: "Generations", key: "generations" },
  { icon: TrendingUp, label: "Revenue",     key: "revenue" },
  { icon: Database,   label: "Providers",   key: "providers" },
  { icon: Settings,   label: "Settings",    key: "settings" },
];

export default function Admin() {
  const [tab, setTab] = useState("overview");
  const s = ADMIN_STATS;

  const topCards = [
    { label: "Total Users",      value: s.total_users.toLocaleString(),              icon: Users,     color: "#F5A623" },
    { label: "Active Today",     value: s.active_today.toLocaleString(),             icon: Activity,  color: "#00D4FF" },
    { label: "Total Generations",value: s.total_generations.toLocaleString(),        icon: Sparkles,  color: "#10B981" },
    { label: "Revenue (month)",  value: formatNaira(s.revenue_month),               icon: TrendingUp,color: "#8B5CF6" },
    { label: "Points Issued",    value: `${(s.points_issued / 1e6).toFixed(1)}M`,  icon: Zap,       color: "#F472B6" },
    { label: "Avg pts/user",     value: s.avg_points_per_user.toLocaleString(),     icon: TrendingUp,color: "#FB923C" },
  ];

  const mockUsers = [
    { name: "Chioma A.",  msisdn: "+234 801 234 5678", tier: "gold",     pts: 3850,  gens: 24, joined: "Mar 2025" },
    { name: "Tunde O.",   msisdn: "+234 803 987 6543", tier: "platinum", pts: 15420, gens: 89, joined: "Jan 2025" },
    { name: "Amina K.",   msisdn: "+234 805 111 2233", tier: "platinum", pts: 8930,  gens: 47, joined: "Feb 2025" },
    { name: "Emeka N.",   msisdn: "+234 706 445 5566", tier: "silver",   pts: 2100,  gens: 11, joined: "Mar 2026" },
    { name: "Fatima B.",  msisdn: "+234 812 778 9900", tier: "gold",     pts: 6750,  gens: 33, joined: "Nov 2025" },
  ];
  const tierColors: Record<string,string> = { bronze:"#CD7F32", silver:"#C0C0C0", gold:"#FFD700", platinum:"#E5E4E2", diamond:"#B9F2FF" };

  return (
    <div className="min-h-screen bg-surface-0 dark flex">
      {/* Sidebar */}
      <div className="glass-strong border-r border-white/[0.07] w-56 shrink-0 flex flex-col fixed top-0 left-0 bottom-0 z-40 hidden lg:flex">
        <div className="p-5 border-b border-white/[0.07]">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-gold flex items-center justify-center">
              <Zap className="w-4 h-4 text-black" />
            </div>
            <div>
              <p className="text-[13px] font-black text-foreground">Loyalty Nexus</p>
              <p className="text-[10px] text-muted-foreground">Admin Panel</p>
            </div>
          </div>
        </div>
        <nav className="flex-1 p-3 flex flex-col gap-0.5">
          {NAV.map(({ icon: Icon, label, key }) => (
            <button
              key={key}
              onClick={() => setTab(key)}
              className={`flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-[13px] font-semibold transition-all duration-150 text-left w-full ${
                tab === key
                  ? "bg-primary/12 text-primary"
                  : "text-muted-foreground hover:text-foreground hover:bg-white/[0.06]"
              }`}
            >
              <Icon className="w-4 h-4 flex-shrink-0" />
              {label}
            </button>
          ))}
        </nav>
        <div className="p-3 border-t border-white/[0.07]">
          <Link to="/" className="flex items-center gap-2 px-3 py-2 rounded-xl text-[13px] text-muted-foreground hover:text-foreground transition-colors">
            <ArrowUpRight className="w-4 h-4" />
            View Site
          </Link>
        </div>
      </div>

      {/* Main */}
      <div className="flex-1 lg:ml-56 min-h-screen">
        {/* Top bar */}
        <div className="glass-strong border-b border-white/[0.07] sticky top-0 z-30 px-6 h-14 flex items-center justify-between">
          <h1 className="text-[15px] font-black text-foreground capitalize">{tab}</h1>
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1.5 text-xs text-chart-3">
              <span className="w-1.5 h-1.5 rounded-full bg-chart-3 inline-block" />
              All systems healthy
            </div>
            <button className="w-8 h-8 rounded-lg hover:bg-white/[0.07] flex items-center justify-center transition-colors">
              <RefreshCw className="w-3.5 h-3.5 text-muted-foreground" />
            </button>
          </div>
        </div>

        <div className="p-6">
          {/* Overview */}
          {tab === "overview" && (
            <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ type: "spring", stiffness: 260, damping: 28 }} className="space-y-6">
              {/* Stat grid */}
              <div className="grid grid-cols-2 lg:grid-cols-3 gap-4">
                {topCards.map(({ label, value, icon: Icon, color }) => (
                  <div key={label} className="glass rounded-2xl p-5 border border-white/[0.07]">
                    <div className="flex items-center gap-2 mb-2">
                      <div className="p-2 rounded-lg" style={{ background: `${color}18` }}>
                        <Icon className="w-4 h-4" style={{ color }} />
                      </div>
                      <span className="text-xs text-muted-foreground font-semibold">{label}</span>
                    </div>
                    <p className="text-2xl font-black" style={{ color }}>{value}</p>
                  </div>
                ))}
              </div>

              {/* Top tools + Provider health side by side */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Top tools */}
                <div className="glass rounded-2xl border border-white/[0.07]">
                  <div className="p-5 border-b border-white/[0.07]">
                    <h3 className="font-black text-sm text-foreground">Top AI Tools Today</h3>
                  </div>
                  <div className="p-4 space-y-3">
                    {s.top_tools.map((slug, i) => {
                      const tool = AI_TOOLS.find(t => t.slug === slug);
                      const pct = [68, 52, 41, 34, 28][i];
                      return (
                        <div key={slug} className="flex items-center gap-3">
                          <span className="text-xl w-8 text-center">{tool?.emoji ?? "🤖"}</span>
                          <div className="flex-1">
                            <div className="flex items-center justify-between mb-1">
                              <span className="text-[12px] font-semibold text-foreground">{tool?.name ?? slug}</span>
                              <span className="text-[11px] text-muted-foreground font-mono">{pct}%</span>
                            </div>
                            <div className="h-1.5 bg-white/[0.07] rounded-full overflow-hidden">
                              <div className="h-full rounded-full bg-gold" style={{ width: `${pct}%` }} />
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>

                {/* Provider health */}
                <div className="glass rounded-2xl border border-white/[0.07]">
                  <div className="p-5 border-b border-white/[0.07]">
                    <h3 className="font-black text-sm text-foreground">Provider Health</h3>
                  </div>
                  <div className="p-4 space-y-3">
                    {Object.entries(s.provider_health).map(([name, info]) => (
                      <div key={name} className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <CheckCircle2 className="w-4 h-4 text-chart-3" />
                          <span className="text-[13px] font-semibold text-foreground capitalize">{name}</span>
                        </div>
                        <div className="flex items-center gap-4 text-[11px] text-muted-foreground">
                          <span className="font-mono">{info.latency_ms}ms</span>
                          <span className="font-mono">{info.calls_today.toLocaleString()} calls</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </motion.div>
          )}

          {/* Users */}
          {tab === "users" && (
            <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ type: "spring", stiffness: 260, damping: 28 }}>
              <div className="glass rounded-2xl border border-white/[0.07] overflow-hidden">
                <div className="p-5 border-b border-white/[0.07] flex items-center justify-between">
                  <h3 className="font-black text-sm text-foreground">Users ({s.total_users.toLocaleString()})</h3>
                  <div className="flex items-center gap-2">
                    <input type="text" placeholder="Search users…" className="glass border border-white/[0.09] rounded-xl h-8 px-3 text-[12px] text-foreground placeholder:text-muted-foreground/40 focus:outline-none focus:border-primary/50 w-48 transition-all" />
                  </div>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-[13px]">
                    <thead>
                      <tr className="border-b border-white/[0.07]">
                        {["Name","MSISDN","Tier","Points","Gens","Joined"].map(h => (
                          <th key={h} className="text-left px-4 py-3 text-[11px] font-black uppercase tracking-wider text-muted-foreground/50">{h}</th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-white/[0.04]">
                      {mockUsers.map((u, i) => (
                        <tr key={i} className="hover:bg-white/[0.03] transition-colors">
                          <td className="px-4 py-3 font-semibold text-foreground">{u.name}</td>
                          <td className="px-4 py-3 text-muted-foreground font-mono text-[12px]">{u.msisdn}</td>
                          <td className="px-4 py-3">
                            <span className="px-2 py-0.5 rounded-full text-[11px] font-bold capitalize" style={{ background: `${tierColors[u.tier]}20`, color: tierColors[u.tier] }}>
                              {u.tier}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-primary font-mono font-bold">{u.pts.toLocaleString()}</td>
                          <td className="px-4 py-3 text-muted-foreground">{u.gens}</td>
                          <td className="px-4 py-3 text-muted-foreground">{u.joined}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </motion.div>
          )}

          {/* Generations */}
          {tab === "generations" && (
            <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ type: "spring", stiffness: 260, damping: 28 }}>
              <div className="glass rounded-2xl border border-white/[0.07] overflow-hidden">
                <div className="p-5 border-b border-white/[0.07]">
                  <h3 className="font-black text-sm text-foreground">AI Generations ({s.total_generations.toLocaleString()})</h3>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-[13px]">
                    <thead>
                      <tr className="border-b border-white/[0.07]">
                        {["ID","User","Tool","Prompt","Points","Status","Date"].map(h => (
                          <th key={h} className="text-left px-4 py-3 text-[11px] font-black uppercase tracking-wider text-muted-foreground/50">{h}</th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-white/[0.04]">
                      {[
                        { id:"g001", user:"Chioma A.", tool:"ai-photo",       prompt:"Lagos skyline at sunset",           pts:10, status:"completed", date:"Mar 26" },
                        { id:"g002", user:"Tunde O.",  tool:"bizplan",        prompt:"Mobile car wash Abuja",             pts:30, status:"completed", date:"Mar 26" },
                        { id:"g003", user:"Amina K.",  tool:"video-cinematic",prompt:"Aerial over Victoria Island",       pts:65, status:"completed", date:"Mar 25" },
                        { id:"g004", user:"Emeka N.",  tool:"narrate",        prompt:"Welcome to Loyalty Nexus",          pts:2,  status:"completed", date:"Mar 25" },
                        { id:"g005", user:"Fatima B.", tool:"study-guide",    prompt:"Machine learning basics",           pts:5,  status:"processing",date:"Mar 27" },
                      ].map((g, i) => {
                        const tool = AI_TOOLS.find(t => t.slug === g.tool);
                        return (
                          <tr key={i} className="hover:bg-white/[0.03] transition-colors">
                            <td className="px-4 py-3 text-muted-foreground font-mono text-[11px]">{g.id}</td>
                            <td className="px-4 py-3 font-semibold text-foreground">{g.user}</td>
                            <td className="px-4 py-3">
                              <div className="flex items-center gap-1.5">
                                <span>{tool?.emoji ?? "🤖"}</span>
                                <span className="text-muted-foreground">{tool?.name ?? g.tool}</span>
                              </div>
                            </td>
                            <td className="px-4 py-3 text-muted-foreground max-w-[200px] truncate">{g.prompt}</td>
                            <td className="px-4 py-3 text-primary font-mono font-bold">{g.pts}</td>
                            <td className="px-4 py-3">
                              <span className={`flex items-center gap-1 text-[11px] font-semibold ${g.status === "completed" ? "text-chart-3" : "text-primary"}`}>
                                {g.status === "completed" ? <CheckCircle2 className="w-3 h-3" /> : <Clock className="w-3 h-3" />}
                                {g.status}
                              </span>
                            </td>
                            <td className="px-4 py-3 text-muted-foreground">{g.date}</td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </div>
            </motion.div>
          )}

          {/* Other tabs placeholder */}
          {!["overview","users","generations"].includes(tab) && (
            <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} className="flex items-center justify-center h-64">
              <div className="text-center">
                <div className="text-5xl mb-4">🔧</div>
                <p className="text-lg font-black text-foreground mb-2 capitalize">{tab}</p>
                <p className="text-sm text-muted-foreground">This section is coming soon.</p>
              </div>
            </motion.div>
          )}
        </div>
      </div>
    </div>
  );
}
