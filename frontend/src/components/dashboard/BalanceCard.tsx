import React from 'react';

export const BalanceCard = ({ amount, unit }: { amount: string, unit: string }) => (
  <div className="glass p-6 rounded-3xl space-y-2">
    <p className="text-xs font-bold tracking-widest text-slate-500 uppercase">{unit} Balance</p>
    <h3 className="text-4xl font-black text-white">{amount}</h3>
    <div className="flex items-center gap-2 text-xs text-green-400">
      <span className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
      Real-time Status
    </div>
  </div>
);
