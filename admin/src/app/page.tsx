"use client";

import React, { useState } from 'react';
import { 
  LayoutDashboard, 
  Settings, 
  Trophy, 
  Wand2, 
  Users, 
  TrendingUp, 
  Zap, 
  ShieldCheck,
  AlertCircle,
  Save,
  Plus
} from 'lucide-react';

export default function AdminPortal() {
  const [activeTab, setActiveCategory] = useState('Overview');

  return (
    <div className="flex h-screen bg-[#050505] text-white overflow-hidden">
      {/* Sidebar */}
      <aside className="w-64 border-r border-white/5 bg-black p-6 flex flex-col justify-between">
        <div className="space-y-8">
          <div>
            <h1 className="text-2xl font-black italic text-brand-gold tracking-tighter">COCKPIT</h1>
            <p className="text-[10px] font-black text-slate-600 uppercase tracking-widest mt-1">Nexus Admin Suite</p>
          </div>

          <nav className="space-y-1">
            {[
              { name: 'Overview', icon: LayoutDashboard },
              { name: 'Program Rules', icon: Settings },
              { name: 'Prize Engine', icon: Trophy },
              { name: 'AI Studio', icon: Wand2 },
              { name: 'Regional Wars', icon: Zap },
              { name: 'Users & Fraud', icon: ShieldCheck },
            ].map((item) => (
              <button
                key={item.name}
                onClick={() => setActiveCategory(item.name)}
                className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl text-xs font-bold transition-all
                  ${activeTab === item.name ? 'bg-brand-gold text-black shadow-lg shadow-yellow-500/10' : 'text-slate-500 hover:text-white hover:bg-white/5'}
                `}
              >
                <item.icon size={16} />
                {item.name}
              </button>
            ))}
          </nav>
        </div>

        <div className="glass p-4 rounded-2xl border border-white/5 space-y-3">
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
            <span className="text-[10px] font-black text-slate-400 uppercase tracking-widest">System Health</span>
          </div>
          <div className="space-y-1">
            <p className="text-[10px] text-slate-600 font-bold">API Lateny: 42ms</p>
            <p className="text-[10px] text-slate-600 font-bold">Redis Ops: 1.2k/s</p>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-grow overflow-y-auto p-10 no-scrollbar">
        {activeTab === 'Overview' && <OverviewTab />}
        {activeTab === 'Program Rules' && <ProgramRulesTab />}
        {activeTab === 'Prize Engine' && <PrizeEngineTab />}
      </main>
    </div>
  );
}

function OverviewTab() {
  return (
    <div className="space-y-10">
      <header>
        <h2 className="text-4xl font-black text-white italic tracking-tighter">System Overview</h2>
        <p className="text-slate-500 font-medium mt-2 uppercase text-xs tracking-widest">Real-time engagement metrics</p>
      </header>

      <div className="grid grid-cols-4 gap-6">
        {[
          { label: 'Active Users', value: '142.5K', trend: '+12%', icon: Users },
          { label: 'Total Revenue', value: '₦8.4M', trend: '+5.2%', icon: TrendingUp },
          { label: 'Studio Renders', value: '4,821', trend: '+24%', icon: Wand2 },
          { label: 'Spin Liability', value: '₦1.2M', trend: '-2%', icon: Trophy },
        ].map((stat) => (
          <div key={stat.label} className="glass p-6 rounded-3xl border border-white/5 space-y-4">
            <div className="flex justify-between items-center text-slate-500">
              <stat.icon size={20} />
              <span className="text-[10px] font-black text-green-400">{stat.trend}</span>
            </div>
            <div>
              <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{stat.label}</p>
              <p className="text-3xl font-black text-white italic mt-1">{stat.value}</p>
            </div>
          </div>
        ))}
      </div>

      <section className="glass rounded-[2.5rem] border border-white/5 overflow-hidden">
        <div className="bg-white/5 p-6 border-b border-white/5 flex items-center justify-between">
          <h3 className="text-sm font-black text-white uppercase tracking-widest flex items-center gap-2">
            <TrendingUp size={16} className="text-brand-gold" /> Recharge Ingestion Velocity
          </h3>
          <div className="flex gap-2">
            <div className="px-3 py-1 rounded-full bg-brand-gold text-black text-[10px] font-black uppercase">Live</div>
          </div>
        </div>
        <div className="h-64 flex items-center justify-center text-slate-700 font-black italic text-sm uppercase tracking-widest">
          Ingestion Chart Placeholder (Recharge_Stream)
        </div>
      </section>
    </div>
  );
}

function ProgramRulesTab() {
  return (
    <div className="space-y-10">
      <header className="flex justify-between items-end">
        <div>
          <h2 className="text-4xl font-black text-white italic tracking-tighter">Program Rules</h2>
          <p className="text-slate-500 font-medium mt-2 uppercase text-xs tracking-widest">Tune the core economic engine</p>
        </div>
        <button className="gold-gradient text-black px-6 py-3 rounded-2xl font-black text-xs uppercase tracking-widest flex items-center gap-2 shadow-xl active:scale-95 transition-all">
          <Save size={16} /> Save All Changes
        </button>
      </header>

      <div className="grid grid-cols-2 gap-8">
        <div className="space-y-6">
          <h3 className="text-xs font-black text-brand-gold uppercase tracking-[0.2em] px-2">Earning Thresholds</h3>
          <div className="glass rounded-3xl border border-white/5 p-8 space-y-8">
            <div className="space-y-3">
              <label className="text-xs font-bold text-slate-400 uppercase tracking-widest">Pulse Point Ratio (Naira)</label>
              <div className="flex items-center gap-4">
                <input type="number" defaultValue={250} className="bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-black text-lg w-32 focus:outline-none focus:border-brand-gold/50" />
                <p className="text-xs text-slate-600 font-medium italic">1 Point = ₦250 Recharge</p>
              </div>
            </div>
            <div className="space-y-3">
              <label className="text-xs font-bold text-slate-400 uppercase tracking-widest">Spin Credit Threshold (Naira)</label>
              <div className="flex items-center gap-4">
                <input type="number" defaultValue={1000} className="bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-black text-lg w-32 focus:outline-none focus:border-brand-gold/50" />
                <p className="text-xs text-slate-600 font-medium italic">₦1,000 Cumulative = 1 Spin</p>
              </div>
            </div>
          </div>
        </div>

        <div className="space-y-6">
          <h3 className="text-xs font-black text-brand-gold uppercase tracking-[0.2em] px-2">Streak Policy</h3>
          <div className="glass rounded-3xl border border-white/5 p-8 space-y-8">
            <div className="space-y-3">
              <label className="text-xs font-bold text-slate-400 uppercase tracking-widest">Streak Window (Hours)</label>
              <div className="flex items-center gap-4">
                <input type="number" defaultValue={36} className="bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-black text-lg w-32 focus:outline-none focus:border-brand-gold/50" />
                <p className="text-xs text-slate-600 font-medium italic">Time allowed between recharges</p>
              </div>
            </div>
            <div className="space-y-3">
              <label className="text-xs font-bold text-slate-400 uppercase tracking-widest">Daily Subscription Fee (Kobo)</label>
              <div className="flex items-center gap-4">
                <input type="number" defaultValue={2000} className="bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white font-black text-lg w-32 focus:outline-none focus:border-brand-gold/50" />
                <p className="text-xs text-slate-600 font-medium italic">2000 Kobo = ₦20 per day</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function PrizeEngineTab() {
  return (
    <div className="space-y-10">
      <header className="flex justify-between items-end">
        <div>
          <h2 className="text-4xl font-black text-white italic tracking-tighter">Prize Engine</h2>
          <p className="text-slate-500 font-medium mt-2 uppercase text-xs tracking-widest">Cryptographic win-probability weights</p>
        </div>
        <button className="bg-white/5 text-white border border-white/10 px-6 py-3 rounded-2xl font-black text-xs uppercase tracking-widest flex items-center gap-2 hover:bg-white/10 transition-all">
          <Plus size={16} /> Add Prize Slot
        </button>
      </header>

      <div className="glass rounded-[2rem] border border-white/5 overflow-hidden shadow-2xl">
        <table className="w-full text-left">
          <thead>
            <tr className="bg-white/5 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] border-b border-white/5">
              <th className="px-8 py-5">Prize Name</th>
              <th className="px-8 py-5">Type</th>
              <th className="px-8 py-5">Value (₦)</th>
              <th className="px-8 py-5">Prob. Weight</th>
              <th className="px-8 py-5 text-right">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            {[
              { name: 'N50,000 Jackpot', type: 'MoMo Cash', value: '50,000', weight: 1, active: true },
              { name: '5GB Data Bundle', type: 'Data', value: '2,500', weight: 50, active: true },
              { name: 'N500 Airtime', type: 'Airtime', value: '500', weight: 200, active: true },
              { name: '100 Pulse Points', type: 'Points', value: '0', weight: 500, active: true },
              { name: 'Try Again', type: 'None', value: '0', weight: 1000, active: true },
            ].map((prize) => (
              <tr key={prize.name} className="hover:bg-white/5 transition-colors group">
                <td className="px-8 py-5 font-bold text-sm text-white italic group-hover:text-brand-gold transition-colors">{prize.name}</td>
                <td className="px-8 py-5 text-xs text-slate-400 font-medium">{prize.type}</td>
                <td className="px-8 py-5 text-sm font-black text-white tracking-tighter">₦{prize.value}</td>
                <td className="px-8 py-5">
                  <div className="flex items-center gap-3">
                    <input type="number" defaultValue={prize.weight} className="bg-white/5 border border-white/10 rounded-lg px-2 py-1 text-white font-bold text-xs w-16 focus:outline-none focus:border-brand-gold/30" />
                    <span className="text-[10px] text-slate-600 font-black">~{((prize.weight/1751)*100).toFixed(1)}%</span>
                  </div>
                </td>
                <td className="px-8 py-5 text-right">
                  <div className={`inline-block w-2 h-2 rounded-full ${prize.active ? 'bg-green-500 shadow-lg shadow-green-500/20' : 'bg-red-500'}`} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="p-6 bg-red-500/5 border border-red-500/10 rounded-3xl flex items-center gap-4">
        <AlertCircle className="text-red-400 shrink-0" size={24} />
        <div>
          <p className="text-sm font-bold text-white uppercase tracking-tight leading-none">Daily Liability Cap Warning</p>
          <p className="text-xs text-slate-500 font-medium mt-1">Current daily awards (₦142,500) represent <span className="text-red-400 font-bold">82%</span> of the admin-configured limit.</p>
        </div>
      </div>
    </div>
  );
}
