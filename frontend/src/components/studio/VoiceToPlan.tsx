"use client";

import React, { useState } from 'react';
import { Mic, Sparkles, Send, Download, ArrowLeft, RefreshCw, StopCircle } from 'lucide-react';
import Link from 'next/link';

export default function VoiceToBusinessPlan() {
  const [isRecording, setIsRecording] = useState(false);
  const [description, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<string | null>(null);

  const toggleRecording = () => {
    setIsRecording(!isRecording);
    // Simulation: if stopping, fill the text
    if (isRecording) {
      setInput("I want to start a mobile laundry service in Lagos that uses an app for scheduling and provides 24-hour delivery.");
    }
  };

  const handleGenerate = async () => {
    if (!description.trim() || isLoading) return;
    setIsLoading(true);
    setTimeout(() => {
      setResult(`https://cdn.loyalty-nexus.ai/build/voice-plan.pdf`);
      setIsLoading(false);
    }, 5000);
  };

  return (
    <div className="min-h-screen bg-black text-white max-w-screen-md mx-auto border-x border-white/5 flex flex-col">
      <header className="glass border-b border-brand-gold/20 px-6 py-4 flex items-center gap-4 sticky top-0 z-50">
        <Link href="/studio" className="p-2 -ml-2 text-slate-400 hover:text-brand-gold transition-colors">
          <ArrowLeft size={20} />
        </Link>
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-2xl gold-gradient flex items-center justify-center text-black shadow-lg shadow-yellow-500/20">
            <Mic size={20} />
          </div>
          <div>
            <h1 className="text-lg font-black tracking-tight italic uppercase leading-none">Voice to Plan</h1>
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest text-brand-gold mt-1 italic">AssemblyAI + Gemini Pipeline</p>
          </div>
        </div>
      </header>

      <main className="flex-grow p-6 space-y-8 overflow-y-auto no-scrollbar">
        {result ? (
          <div className="glass rounded-3xl p-10 border border-brand-gold/30 flex flex-col items-center text-center space-y-6 animate-in zoom-in-95">
            <div className="w-20 h-20 rounded-3xl gold-gradient flex items-center justify-center text-black shadow-2xl">
              <Sparkles size={40} />
            </div>
            <div>
              <h3 className="text-xl font-black text-white tracking-tight italic">Plan Orchestrated</h3>
              <p className="text-sm text-slate-500 font-bold uppercase tracking-widest mt-1 italic leading-relaxed">
                Your audio was transcribed, structured into a financial model, and formatted.
              </p>
            </div>
            <button className="gold-gradient text-black px-10 py-4 rounded-2xl font-black text-sm uppercase tracking-[0.2em] flex items-center gap-3 shadow-xl hover:scale-105 transition-transform active:scale-95">
              <Download size={20} /> Download PDF
            </button>
          </div>
        ) : (
          <div className="space-y-10">
            {/* Recorder Hub */}
            <div className="flex flex-col items-center space-y-6">
              <div className={`relative p-1 rounded-full transition-all duration-500 ${isRecording ? 'bg-red-500/20' : 'bg-brand-gold/5'}`}>
                <button 
                  onClick={toggleRecording}
                  className={`w-32 h-32 rounded-full flex flex-col items-center justify-center gap-2 transition-all duration-500 border-4
                    ${isRecording 
                      ? 'bg-red-500 border-red-400 scale-110 shadow-2xl shadow-red-500/40' 
                      : 'gold-gradient border-brand-gold/50 shadow-xl shadow-yellow-500/10 hover:scale-105 active:scale-95 text-black'}
                  `}
                >
                  {isRecording ? <StopCircle size={40} className="text-white" /> : <Mic size={40} />}
                  <span className={`text-[10px] font-black uppercase tracking-widest ${isRecording ? 'text-white' : 'text-black opacity-60'}`}>
                    {isRecording ? 'Recording' : 'Tap to Speak'}
                  </span>
                </button>
                {isRecording && (
                  <div className="absolute inset-0 rounded-full border-2 border-red-500 animate-ping opacity-20 scale-125" />
                )}
              </div>
              <p className="text-xs text-slate-500 font-medium text-center max-w-xs leading-relaxed italic">
                Speak naturally about your business idea. We will handle the transcription and structuring.
              </p>
            </div>

            <div className="space-y-4">
              <div className="relative group">
                <textarea
                  rows={4}
                  placeholder="Or type your business idea here... (Transcription appears here after recording)"
                  value={description}
                  onChange={(e) => setInput(e.target.value)}
                  className="w-full bg-white/5 border border-white/10 rounded-3xl py-6 px-8 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/30 focus:bg-white/10 transition-all resize-none shadow-inner"
                />
                <div className="absolute bottom-6 right-8 text-[10px] font-black text-slate-600 uppercase tracking-widest">
                  Cost: 6 PTS
                </div>
              </div>

              <button
                onClick={handleGenerate}
                disabled={!description.trim() || isLoading || isRecording}
                className={`w-full py-5 rounded-2xl font-black text-sm uppercase tracking-[0.2em] transition-all flex items-center justify-center gap-3 shadow-2xl
                  ${description.trim() && !isLoading && !isRecording
                    ? 'gold-gradient text-black shadow-yellow-500/20 active:scale-95' 
                    : 'bg-white/5 text-slate-600 cursor-not-allowed'}
                `}
              >
                {isLoading ? (
                  <>
                    <RefreshCw className="w-5 h-5 animate-spin" />
                    Transcribing & Orchestrating
                  </>
                ) : (
                  <>
                    Build Business Plan
                    <Send size={18} />
                  </>
                )}
              </button>
            </div>
          </div>
        )}
      </main>

      <footer className="p-8 text-center">
        <p className="text-[9px] font-bold text-slate-700 uppercase tracking-[0.3em]">
          Professional Document Automation Engine
        </p>
      </footer>
    </div>
  );
}
