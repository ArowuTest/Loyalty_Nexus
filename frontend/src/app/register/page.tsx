"use client";

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Smartphone, Sparkles, MapPin, ArrowRight, Loader2 } from 'lucide-react';

const NIGERIAN_STATES = [
  'Lagos', 'Abuja (FCT)', 'Kano', 'Rivers', 'Oyo', 'Enugu', 'Kaduna', 'Edo', 'Delta', 'Anambra', 
  'Abia', 'Adamawa', 'Akwa Ibom', 'Bauchi', 'Bayelsa', 'Benue', 'Borno', 'Cross River', 'Ebonyi', 
  'Ekiti', 'Gombe', 'Imo', 'Jigawa', 'Katsina', 'Kebbi', 'Kogi', 'Kwara', 'Nasarawa', 'Niger', 
  'Ogun', 'Ondo', 'Osun', 'Plateau', 'Sokoto', 'Taraba', 'Yobe', 'Zamfara'
];

export default function Register() {
  const router = useRouter();
  const [msisdn, setMsisdn] = useState('');
  const [otp, setOtp] = useState('');
  const [state, setState] = useState('');
  const [step, setStep] = useState(1); // 1: MSISDN, 2: OTP
  const [isLoading, setIsLoading] = useState(false);

  const handleSendOTP = () => {
    if (msisdn.length < 11) return;
    setIsLoading(true);
    // In production: POST /api/v1/auth/otp/send
    setTimeout(() => {
      setStep(2);
      setIsLoading(false);
    }, 1500);
  };

  const handleVerifyOTP = () => {
    setIsLoading(true);
    // In production: POST /api/v1/auth/otp/verify
    setTimeout(() => {
      router.push('/dashboard');
      setIsLoading(false);
    }, 1500);
  };

  return (
    <div className="min-h-screen bg-black text-white flex flex-col items-center justify-center p-6 bg-[url('https://static-s3.skyworkcdn.com/fe/skywork-site-assets/images/skybot/bg-pattern.png')] bg-fixed">
      <div className="w-full max-w-md space-y-10 animate-in fade-in slide-in-from-bottom-4 duration-1000">
        {/* Brand */}
        <div className="text-center space-y-2">
          <h1 className="text-6xl font-black italic tracking-tighter text-white">NEXUS</h1>
          <p className="text-xs font-black text-brand-gold uppercase tracking-[0.3em]">The Private Firm Infrastructure</p>
        </div>

        <div className="glass rounded-[2.5rem] p-10 border border-brand-gold/20 shadow-2xl shadow-brand-gold/10">
          {step === 1 ? (
            <div className="space-y-8">
              <div className="space-y-2 text-center">
                <h2 className="text-2xl font-black text-white italic">Welcome to Pulse</h2>
                <p className="text-sm text-slate-500 font-medium leading-relaxed">
                  Enter your mobile number to begin your <span className="text-brand-gold">Loyalty Nexus</span> journey.
                </p>
              </div>

              <div className="space-y-6">
                <div className="space-y-4">
                  <div className="relative group">
                    <Smartphone className="absolute left-5 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-brand-gold transition-colors" size={20} />
                    <input 
                      type="tel" 
                      placeholder="0803 000 0000"
                      value={msisdn}
                      onChange={(e) => setMsisdn(e.target.value)}
                      className="w-full bg-white/5 border border-white/10 rounded-2xl py-5 pl-14 pr-6 text-lg font-black tracking-widest text-white placeholder:text-slate-700 focus:outline-none focus:border-brand-gold/50 transition-all"
                    />
                  </div>

                  <div className="relative group">
                    <MapPin className="absolute left-5 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-brand-gold transition-colors" size={20} />
                    <select 
                      value={state}
                      onChange={(e) => setState(e.target.value)}
                      className="w-full bg-white/5 border border-white/10 rounded-2xl py-5 pl-14 pr-6 text-sm font-bold text-white focus:outline-none focus:border-brand-gold/50 transition-all appearance-none"
                    >
                      <option value="" disabled className="bg-black">Select your state (REQ-1.5)</option>
                      {NIGERIAN_STATES.map(s => <option key={s} value={s} className="bg-black">{s}</option>)}
                    </select>
                  </div>
                </div>

                <button 
                  onClick={handleSendOTP}
                  disabled={!msisdn || !state || isLoading}
                  className="w-full gold-gradient text-black py-5 rounded-2xl font-black text-sm uppercase tracking-[0.2em] shadow-xl shadow-yellow-500/10 hover:scale-[1.02] active:scale-95 transition-all flex items-center justify-center gap-3 disabled:opacity-50 disabled:grayscale disabled:scale-100"
                >
                  {isLoading ? <Loader2 className="animate-spin" size={20} /> : <>Generate OTP <ArrowRight size={18} /></>}
                </button>
              </div>
            </div>
          ) : (
            <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-500">
              <div className="space-y-2 text-center">
                <h2 className="text-2xl font-black text-white italic">Verify Identity</h2>
                <p className="text-sm text-slate-500 font-medium leading-relaxed">
                  We've sent a 6-digit code to <span className="text-white font-bold">{msisdn}</span>.
                </p>
              </div>

              <div className="space-y-6">
                <input 
                  type="text" 
                  maxLength={6}
                  placeholder="0 0 0 0 0 0"
                  value={otp}
                  onChange={(e) => setOtp(e.target.value)}
                  className="w-full bg-white/5 border border-white/10 rounded-2xl py-5 px-6 text-center text-3xl font-black tracking-[0.5em] text-brand-gold placeholder:text-slate-800 focus:outline-none focus:border-brand-gold/50 transition-all shadow-inner"
                />

                <div className="space-y-4">
                  <button 
                    onClick={handleVerifyOTP}
                    disabled={otp.length < 6 || isLoading}
                    className="w-full gold-gradient text-black py-5 rounded-2xl font-black text-sm uppercase tracking-[0.2em] shadow-xl shadow-yellow-500/10 hover:scale-[1.02] active:scale-95 transition-all flex items-center justify-center gap-3 disabled:opacity-50"
                  >
                    {isLoading ? <Loader2 className="animate-spin" size={20} /> : <>Verify & Continue <Sparkles size={18} /></>}
                  </button>
                  
                  <button 
                    onClick={() => setStep(1)}
                    className="w-full text-slate-500 font-bold text-[10px] uppercase tracking-widest hover:text-white transition-colors"
                  >
                    Change Phone Number
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>

        <p className="text-center text-[10px] font-medium text-slate-600 leading-relaxed uppercase tracking-[0.1em]">
          By continuing, you agree to the <span className="text-slate-400 font-bold underline">Terms of Service</span>. <br/>
          Loyalty Nexus Enterprise Infrastructure V1.0.2
        </p>
      </div>
    </div>
  );
}
