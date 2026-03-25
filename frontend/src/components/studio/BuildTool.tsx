"use client";

import React, { useState } from 'react';
import { Hammer, Sparkles, Send, Download, ArrowLeft, RefreshCw, AlertCircle, FileText, Presentation } from 'lucide-react';
import Link from 'next/link';

interface BuildToolProps {
  toolId: string;
  toolName: string;
  pointCost: number;
}

export default function BuildTool({ toolId, toolName, pointCost }: BuildToolProps) {
  const [description, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleGenerate = async () => {
    if (!description.trim() || isLoading) return;

    setIsLoading(true);
    setError(null);
    setResult(null);

    try {
      // In production: POST /api/v1/studio/generate/build
      setTimeout(() => {
        setResult(`https://cdn.loyalty-nexus.ai/build/mock-${toolId}.pdf`);
        setIsLoading(false);
      }, 4000);
    } catch (err: any) {
      setError(err.message || 'Build failed. Points refunded.');
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-black text-white max-w-screen-md mx-auto border-x border-white/5 flex flex-col">
      <header className="glass border-b border-brand-gold/20 px-6 py-4 flex items-center gap-4 sticky top-0 z-50">
        <Link href="/studio" className="p-2 -ml-2 text-slate-400 hover:text-brand-gold transition-colors">
          <ArrowLeft size={20} />
        </Link>
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-2xl gold-gradient flex items-center justify-center text-black shadow-lg shadow-yellow-500/20">
            <Hammer size={20} />
          </div>
          <div>
            <h1 className="text-lg font-black tracking-tight italic uppercase">{toolName}</h1>
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest text-brand-gold">Point-to-Computing Engine</p>
          </div>
        </div>
      </header>

      <main className="flex-grow p-6 space-y-8 overflow-y-auto no-scrollbar">
        <div className="glass rounded-3xl p-6 border border-white/5 space-y-2 text-center">
          <h2 className="text-sm font-black text-brand-gold uppercase tracking-wider flex items-center justify-center gap-2">
            <Sparkles size={14} /> Professional Automation
          </h2>
          <p className="text-xs text-slate-400 font-medium leading-relaxed">
            Provide details about your business or project. Our engine will structure and format a professional output for you.
          </p>
        </div>

        {result ? (
          <div className="glass rounded-3xl p-10 border border-brand-gold/30 flex flex-col items-center text-center space-y-6 animate-in zoom-in-95 duration-500">
            <div className="w-20 h-20 rounded-3xl gold-gradient flex items-center justify-center text-black shadow-2xl">
              {toolName.includes('Slide') ? <Presentation size={40} /> : <FileText size={40} />}
            </div>
            <div>
              <h3 className="text-xl font-black text-white tracking-tight italic">Generation Complete</h3>
              <p className="text-sm text-slate-500 font-bold uppercase tracking-widest mt-1">Your document is ready for download</p>
            </div>
            <button className="gold-gradient text-black px-10 py-4 rounded-2xl font-black text-sm uppercase tracking-[0.2em] flex items-center gap-3 shadow-xl hover:scale-105 transition-transform active:scale-95">
              <Download size={20} /> Download File
            </button>
          </div>
        ) : (
          <div className="space-y-6">
            <div className="relative group">
              <textarea
                rows={6}
                placeholder="Describe your project, business idea, or audience..."
                value={description}
                onChange={(e) => setInput(e.target.value)}
                className="w-full bg-white/5 border border-white/10 rounded-3xl py-6 px-8 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/30 focus:bg-white/10 transition-all resize-none"
              />
              <div className="absolute bottom-6 right-8 text-[10px] font-black text-slate-600 uppercase tracking-widest">
                Cost: {pointCost} PTS
              </div>
            </div>

            <button
              onClick={handleGenerate}
              disabled={!description.trim() || isLoading}
              className={`w-full py-5 rounded-2xl font-black text-sm uppercase tracking-[0.2em] transition-all flex items-center justify-center gap-3 shadow-2xl
                ${description.trim() && !isLoading 
                  ? 'gold-gradient text-black shadow-yellow-500/20 active:scale-95' 
                  : 'bg-white/5 text-slate-600 cursor-not-allowed'}
              `}
            >
              {isLoading ? (
                <>
                  <RefreshCw className="w-5 h-5 animate-spin" />
                  Orchestrating AI Pipeline
                </>
              ) : (
                <>
                  Build Document
                  <Send size={18} />
                </>
              )}
            </button>
          </div>
        )}
      </main>

      <footer className="p-6 text-center">
        <p className="text-[9px] font-bold text-slate-700 uppercase tracking-[0.3em]">
          Powered by Nexus Infrastructure
        </p>
      </footer>
    </div>
  );
}
