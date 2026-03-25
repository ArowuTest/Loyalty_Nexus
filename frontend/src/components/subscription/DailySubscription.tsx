"use client";

import React, { useState } from 'react';
import { Gift, ShieldCheck, Zap, ArrowLeft, CheckCircle2, Clock } from 'lucide-react';
import Link from 'next/link';

export default function DailySubscription() {
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubscribe = () => {
    setIsLoading(true);
    // Simulation
    setTimeout(() => {
      setIsSubscribed(true);
      setIsLoading(false);
    }, 2000);
  };

  return (
    <div className="min-h-screen bg-black text-white max-w-screen-md mx-auto border-x border-white/5 flex flex-col">
      {/* Header */}
      <header className="glass border-b border-brand-gold/20 px-6 py-4 flex items-center gap-4 sticky top-0 z-50">
        <Link href="/dashboard" className="p-2 -ml-2 text-slate-400 hover:text-brand-gold transition-colors">
          <ArrowLeft size={20} />
        </Link>
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-2xl gold-gradient flex items-center justify-center text-black shadow-lg shadow-yellow-500/20">
            <Gift size={20} />
          </div>
          <div>
            <h1 className="text-lg font-black tracking-tight italic uppercase">Daily Draw Pass</h1>
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest text-brand-gold">Guaranteed Entry</p>
          </div>
        </div>
      </header>

      <main className="flex-grow p-6 space-y-8 overflow-y-auto no-scrollbar">
        {/* Pricing Card */}
        <div className="glass rounded-[2.5rem] p-10 border border-brand-gold/30 text-center space-y-6 shadow-2xl shadow-brand-gold/5">
          <div className="inline-flex items-center gap-2 bg-brand-gold/10 text-brand-gold text-[10px] font-black px-4 py-1.5 rounded-full border border-brand-gold/20 tracking-[0.2em] uppercase">
            Limited Time Offer
          </div>
          
          <div className="space-y-1">
            <p className="text-sm font-bold text-slate-500 uppercase tracking-widest leading-none">Subscription Fee</p>
            <div className="flex items-baseline justify-center gap-1">
              <span className="text-6xl font-black italic tracking-tighter text-white">₦20</span>
              <span className="text-brand-gold font-black text-lg uppercase italic">/day</span>
            </div>
          </div>

          <div className="py-6 border-y border-white/5 space-y-4">
            <div className="flex items-center gap-3 text-left">
              <div className="w-6 h-6 rounded-full bg-green-500/20 flex items-center justify-center text-green-400">
                <CheckCircle2 size={14} />
              </div>
              <p className="text-sm font-medium text-slate-300">Guaranteed daily entry into the <span className="text-white font-bold italic">₦50,000 Jackpot</span></p>
            </div>
            <div className="flex items-center gap-3 text-left">
              <div className="w-6 h-6 rounded-full bg-green-500/20 flex items-center justify-center text-green-400">
                <CheckCircle2 size={14} />
              </div>
              <p className="text-sm font-medium text-slate-300">Automatic entry — <span className="text-white font-bold italic">No recharge required</span></p>
            </div>
            <div className="flex items-center gap-3 text-left">
              <div className="w-6 h-6 rounded-full bg-green-500/20 flex items-center justify-center text-green-400">
                <CheckCircle2 size={14} />
              </div>
              <p className="text-sm font-medium text-slate-300">Priority reward fulfillment</p>
            </div>
          </div>

          {isSubscribed ? (
            <div className="space-y-4 animate-in fade-in slide-in-from-bottom-2">
              <div className="bg-green-500/10 border border-green-500/20 p-4 rounded-2xl flex items-center gap-3">
                <ShieldCheck className="text-green-400" size={24} />
                <div className="text-left">
                  <p className="text-sm font-bold text-white uppercase tracking-tight leading-none">Subscription Active</p>
                  <p className="text-[10px] text-green-400/70 font-bold uppercase tracking-widest mt-1">Next Billing: Tomorrow, 8:00 AM</p>
                </div>
              </div>
              <button className="w-full text-slate-500 font-black text-[10px] uppercase tracking-[0.2em] hover:text-red-400 transition-colors">
                Cancel Subscription
              </button>
            </div>
          ) : (
            <button 
              onClick={handleSubscribe}
              disabled={isLoading}
              className="w-full gold-gradient text-black py-5 rounded-[1.5rem] font-black text-sm uppercase tracking-[0.2em] shadow-xl hover:scale-105 transition-all active:scale-95 flex items-center justify-center gap-3"
            >
              {isLoading ? 'Processing Account...' : 'Subscribe Now'}
              {!isLoading && <Zap size={18} />}
            </button>
          )}
        </div>

        {/* History / Info */}
        <div className="space-y-4">
          <div className="flex items-center justify-between px-2">
            <h2 className="text-xs font-black text-slate-500 uppercase tracking-widest">Recent Activity</h2>
            <Link href="/history" className="text-[10px] font-black text-brand-gold uppercase tracking-widest hover:underline">View All</Link>
          </div>
          <div className="glass rounded-3xl border border-white/5 divide-y divide-white/5 overflow-hidden">
            {isSubscribed ? (
              <div className="p-5 flex items-center justify-between bg-white/5">
                <div className="flex items-center gap-4">
                  <div className="w-10 h-10 rounded-xl bg-green-500/10 flex items-center justify-center text-green-400">
                    <CheckCircle2 size={20} />
                  </div>
                  <div>
                    <p className="text-sm font-bold text-white uppercase tracking-tight leading-none">Subscribed</p>
                    <p className="text-[10px] text-slate-500 font-bold uppercase tracking-widest mt-1">Daily Draw Pass activated</p>
                  </div>
                </div>
                <p className="text-sm font-black italic text-white">-₦20</p>
              </div>
            ) : (
              <div className="p-10 text-center opacity-30">
                <Clock size={40} className="mx-auto text-slate-500 mb-3" />
                <p className="text-xs font-bold uppercase tracking-widest">No recent transactions</p>
              </div>
            )}
          </div>
        </div>
      </main>

      <footer className="p-8 text-center bg-brand-gold/5 border-t border-white/5">
        <p className="text-[10px] font-medium text-slate-500 leading-relaxed max-w-xs mx-auto">
          Terms applied. Daily fee deducted from your <span className="text-brand-gold font-bold italic">MTN Balance</span> or Linked Card. Cancel anytime from your dashboard.
        </p>
      </footer>
    </div>
  );
}
