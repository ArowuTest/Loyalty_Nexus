import { BalanceCard } from "@/components/dashboard/BalanceCard";
import { Trophy, Zap, Sparkles } from "lucide-react";

export default function UserDashboard() {
  return (
    <div className="max-w-screen-xl mx-auto px-6 py-12 space-y-12">
      {/* Header */}
      <header className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-black italic text-brand-gold">NEXUS</h1>
          <p className="text-slate-500 font-medium">Lagos, Nigeria</p>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right">
            <p className="text-xs text-slate-500 font-bold">PLATINUM TIER</p>
            <p className="text-sm font-bold text-white">Chukwudi O.</p>
          </div>
          <div className="w-12 h-12 rounded-full gold-gradient shadow-lg shadow-yellow-500/20" />
        </div>
      </header>

      {/* Hero Stats */}
      <div className="grid md:grid-cols-3 gap-6">
        <BalanceCard amount="₦4,250" unit="Airtime" />
        <BalanceCard amount="12.5 GB" unit="Data" />
        <div className="glass p-6 rounded-3xl flex flex-col justify-center items-center text-center space-y-2 border-brand-gold/30">
          <div className="w-12 h-12 rounded-2xl gold-gradient flex items-center justify-center mb-2">
            <Sparkles className="text-black w-6 h-6" />
          </div>
          <h3 className="text-xl font-bold text-white">Daily Spin Ready</h3>
          <button className="text-xs font-black text-brand-gold uppercase tracking-tighter hover:underline">
            Play Now & Win →
          </button>
        </div>
      </div>

      {/* Live Feed */}
      <section className="glass rounded-3xl overflow-hidden">
        <div className="bg-white/5 px-6 py-3 border-b border-white/10 flex items-center justify-between">
          <div className="flex items-center gap-2 text-xs font-bold text-slate-400">
            <div className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
            LIVE WINNERS
          </div>
        </div>
        <div className="px-6 py-4 h-12 flex items-center gap-8 whitespace-nowrap overflow-hidden italic font-bold text-sm text-slate-300">
          <span>🏆 User 234803*** just won 5GB Data</span>
          <span className="opacity-30">/</span>
          <span>💎 Aisha from Kano earned 500 Studio Credits</span>
          <span className="opacity-30">/</span>
          <span>💰 N10,000 MoMo Cash won in Abuja</span>
        </div>
      </section>
    </div>
  );
}
