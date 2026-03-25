"use client";

import { BalanceCard } from "@/components/dashboard/BalanceCard";
import { Trophy, Zap, Sparkles, Smartphone, Apple, Chrome, Users, MapPin, TrendingUp } from "lucide-react";
import Link from 'next/link';

  const handleIssueWallet = async (platform: 'apple' | 'google') => {
    try {
      // In production: GET /api/v1/user/wallet/issue?platform=...
      // Redirect to the provided URL
      const mockUrl = platform === 'apple' 
        ? 'https://cdn.loyalty-nexus.ai/passes/mock.pkpass' 
        : 'https://pay.google.com/gp/v/save/mock-jwt';
      window.location.href = mockUrl;
    } catch (error) {
      console.error('Wallet issuance failed:', error);
    }
  };

  return (
    <div className="max-w-screen-xl mx-auto px-6 py-12 space-y-12 bg-black text-white">
      {/* ... header ... */}

      {/* Hero Stats */}
      <div className="grid md:grid-cols-3 gap-6">
        {/* ... */}
        
        {/* Passport Status Card */}
        <div className="glass p-6 rounded-3xl flex flex-col justify-between border border-brand-gold/30">
          {/* ... */}
          
          <div className="flex gap-2">
            <button 
              onClick={() => handleIssueWallet('apple')}
              className="flex-grow flex items-center justify-center gap-2 bg-white/5 hover:bg-white/10 border border-white/10 py-2.5 rounded-xl text-[10px] font-black uppercase transition-all tracking-widest"
            >
              <Apple size={14} /> Apple
            </button>
            <button 
              onClick={() => handleIssueWallet('google')}
              className="flex-grow flex items-center justify-center gap-2 bg-white/5 hover:bg-white/10 border border-white/10 py-2.5 rounded-xl text-[10px] font-black uppercase transition-all tracking-widest"
            >
              <Chrome size={14} /> Google
            </button>
          </div>
        </div>
      </div>
      {/* ... */}
    </div>
  );

      <div className="grid lg:grid-cols-3 gap-8">
        {/* Regional Wars Tournament */}
        <section className="lg:col-span-2 space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-black text-white italic uppercase tracking-tighter flex items-center gap-2">
              <Trophy className="text-brand-gold" size={20} /> Regional Wars
            </h2>
            <div className="bg-brand-gold text-black text-[10px] font-black px-3 py-1 rounded-full animate-pulse uppercase tracking-widest">
              LIVE TOURNAMENT
            </div>
          </div>

          <div className="glass rounded-[2rem] overflow-hidden border border-white/5">
            <div className="bg-white/5 p-6 border-b border-white/5 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="w-12 h-12 rounded-2xl gold-gradient flex items-center justify-center text-black shadow-xl">
                  <MapPin size={24} />
                </div>
                <div>
                  <p className="text-xs text-slate-500 font-bold uppercase tracking-widest leading-none">Your Region</p>
                  <h3 className="text-lg font-black text-white italic">LAGOS</h3>
                </div>
              </div>
              <div className="text-right">
                <p className="text-[10px] font-black text-brand-gold uppercase tracking-widest">Active Bonus</p>
                <div className="flex items-center gap-1.5 text-white justify-end">
                  <Zap size={14} className="fill-brand-gold text-brand-gold" />
                  <span className="text-2xl font-black italic tracking-tighter">2.0X</span>
                </div>
              </div>
            </div>

            <div className="p-2 space-y-1">
              {[
                { rank: 1, name: 'Lagos', amount: '₦12.4M', trend: 'up', bonus: true },
                { rank: 2, name: 'Abuja', amount: '₦8.1M', trend: 'down', bonus: false },
                { rank: 3, name: 'Port Harcourt', amount: '₦5.2M', trend: 'up', bonus: false },
              ].map((region) => (
                <div key={region.name} className={`flex items-center justify-between p-4 rounded-2xl transition-all ${region.bonus ? 'bg-white/5 border border-brand-gold/20' : 'hover:bg-white/5'}`}>
                  <div className="flex items-center gap-4">
                    <span className={`text-xl font-black italic w-6 ${region.rank === 1 ? 'text-brand-gold' : 'text-slate-700'}`}>
                      {region.rank}
                    </span>
                    <div>
                      <h4 className="font-bold text-white uppercase text-sm tracking-tight">{region.name}</h4>
                      <p className="text-[10px] font-bold text-slate-500 uppercase tracking-widest">{region.amount} Recharged</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    {region.trend === 'up' ? <TrendingUp size={16} className="text-green-500" /> : <TrendingUp size={16} className="text-red-500 rotate-180" />}
                    <div className="w-24 bg-white/5 h-1.5 rounded-full overflow-hidden">
                      <div className={`h-full ${region.rank === 1 ? 'gold-gradient' : 'bg-slate-700'}`} style={{ width: `${100 - region.rank * 20}%` }} />
                    </div>
                  </div>
                </div>
              ))}
            </div>
            
            <div className="p-6 bg-brand-gold/5 border-t border-white/5 text-center">
              <p className="text-[10px] font-bold text-slate-400 uppercase tracking-[0.2em]">
                Leading region wins <span className="text-brand-gold">Golden Hour</span> every Friday
              </p>
            </div>
          </div>
        </section>

        {/* Community & Streaks */}
        <section className="space-y-6">
          <h2 className="text-xl font-black text-white italic uppercase tracking-tighter flex items-center gap-2">
            <Zap className="text-brand-gold" size={20} /> Pulse Streaks
          </h2>
          <div className="glass rounded-[2rem] p-8 border border-brand-gold/20 flex flex-col items-center text-center space-y-6">
            <div className="relative">
              <div className="w-32 h-32 rounded-full border-4 border-white/5 flex items-center justify-center relative">
                <span className="text-5xl font-black text-white italic">5</span>
                <div className="absolute inset-0 rounded-full border-4 border-brand-gold border-t-transparent animate-spin duration-[3s]" />
              </div>
              <div className="absolute -bottom-2 left-1/2 -translate-x-1/2 bg-brand-gold text-black text-[10px] font-black px-3 py-1 rounded-full shadow-xl">
                DAY STREAK
              </div>
            </div>
            <div>
              <h3 className="text-lg font-bold text-white uppercase tracking-tight">Keep it going!</h3>
              <p className="text-xs text-slate-500 font-medium leading-relaxed mt-2">
                Recharge within <span className="text-white font-bold">12 hours</span> to keep your streak and earn a Mega Jackpot ticket.
              </p>
            </div>
            <button className="w-full gold-gradient text-black py-4 rounded-2xl font-black text-xs uppercase tracking-[0.2em] shadow-xl hover:scale-105 transition-transform active:scale-95">
              Recharge Now
            </button>
          </div>
        </section>
      </div>

      {/* Live Feed */}
      <section className="glass rounded-3xl overflow-hidden border border-white/5">
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
          <span>💰 ₦10,000 MoMo Cash won in Abuja</span>
        </div>
      </section>
    </div>
  );
}
