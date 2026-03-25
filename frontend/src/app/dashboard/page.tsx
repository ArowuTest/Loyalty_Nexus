import { BalanceCard } from "@/components/dashboard/BalanceCard";
import { Trophy, Zap, Sparkles, Smartphone, Apple, Chrome } from "lucide-react";

export default function UserDashboard() {
  return (
    <div className="max-w-screen-xl mx-auto px-6 py-12 space-y-12">
      {/* ... existing header ... */}

      {/* Hero Stats */}
      <div className="grid md:grid-cols-3 gap-6">
        <BalanceCard amount="₦4,250" unit="Airtime" />
        <BalanceCard amount="12.5 GB" unit="Data" />
        
        {/* Passport Status Card */}
        <div className="glass p-6 rounded-3xl flex flex-col justify-between border-brand-gold/30">
          <div className="flex justify-between items-start">
            <div className="w-10 h-10 rounded-xl gold-gradient flex items-center justify-center text-black">
              <Smartphone size={20} />
            </div>
            <div className="bg-green-500/10 text-green-400 text-[10px] font-black px-2 py-1 rounded-full border border-green-500/20">
              PASSPORT ACTIVE
            </div>
          </div>
          
          <div>
            <h3 className="text-xl font-black text-white italic">Digital Passport</h3>
            <p className="text-xs text-slate-500 font-bold uppercase tracking-tighter">Persistent Lock-screen Card</p>
          </div>

          <div className="flex gap-2">
            <button className="flex-grow flex items-center justify-center gap-2 bg-white/5 hover:bg-white/10 border border-white/10 py-2 rounded-xl text-[10px] font-black uppercase transition-all">
              <Apple size={14} /> Apple
            </button>
            <button className="flex-grow flex items-center justify-center gap-2 bg-white/5 hover:bg-white/10 border border-white/10 py-2 rounded-xl text-[10px] font-black uppercase transition-all">
              <Chrome size={14} /> Google
            </button>
          </div>
        </div>
      </div>
      
      {/* ... existing live feed ... */}
    </div>
  );
}
