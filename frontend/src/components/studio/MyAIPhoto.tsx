"use client";

import React, { useState } from 'react';
import { Camera, Sparkles, Send, Download, ArrowLeft, RefreshCw, AlertCircle } from 'lucide-react';
import Link from 'next/link';

export default function MyAIPhoto() {
  const [prompt, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleGenerate = async () => {
    if (!prompt.trim() || isLoading) return;

    setIsLoading(true);
    setError(null);
    setResult(null);

    try {
      // In production: POST /api/v1/studio/generate/image
      // Simulation
      setTimeout(() => {
        // Mock success
        setResult('https://static-s3.skyworkcdn.com/fe/skywork-site-assets/images/skybot/avatar1-new.png'); // Placeholder
        setIsLoading(false);
      }, 3000);
    } catch (err: any) {
      setError(err.message || 'Generation failed. Points refunded.');
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-black text-white max-w-screen-md mx-auto border-x border-white/5 flex flex-col">
      {/* Header */}
      <header className="glass border-b border-brand-gold/20 px-6 py-4 flex items-center gap-4 sticky top-0 z-50">
        <Link href="/studio" className="p-2 -ml-2 text-slate-400 hover:text-brand-gold transition-colors">
          <ArrowLeft size={20} />
        </Link>
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-2xl gold-gradient flex items-center justify-center text-black shadow-lg shadow-yellow-500/20">
            <Camera size={20} />
          </div>
          <div>
            <h1 className="text-lg font-black tracking-tight italic uppercase">My AI Photo</h1>
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest">Powered by Flux-1-Schnell</p>
          </div>
        </div>
      </header>

      <main className="flex-grow p-6 space-y-8 overflow-y-auto no-scrollbar">
        {/* Instruction Card */}
        <div className="glass rounded-3xl p-6 border border-white/5 space-y-2">
          <h2 className="text-sm font-black text-brand-gold uppercase tracking-wider flex items-center gap-2">
            <Sparkles size={14} /> Professional AI Portraits
          </h2>
          <p className="text-xs text-slate-400 font-medium leading-relaxed">
            Describe your desired portrait. Be specific about clothing, background, and expression.
            Each generation costs <span className="text-white font-bold">10 Pulse Points</span>.
          </p>
        </div>

        {/* Result Area */}
        <div className="aspect-square w-full relative rounded-3xl overflow-hidden border border-white/10 glass flex items-center justify-center group">
          {isLoading ? (
            <div className="flex flex-col items-center gap-4">
              <RefreshCw className="w-10 h-10 text-brand-gold animate-spin" />
              <p className="text-xs font-black text-brand-gold uppercase tracking-[0.3em] animate-pulse">Rendering...</p>
            </div>
          ) : result ? (
            <>
              <img src={result} alt="Generated AI Portrait" className="w-full h-full object-cover animate-in fade-in zoom-in-95 duration-700" />
              <div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-4">
                <button className="gold-gradient text-black px-6 py-2.5 rounded-2xl font-black text-xs uppercase flex items-center gap-2 shadow-xl">
                  <Download size={16} /> Download
                </button>
              </div>
            </>
          ) : error ? (
            <div className="flex flex-col items-center gap-2 text-red-400 p-8 text-center">
              <AlertCircle size={32} />
              <p className="text-sm font-bold">{error}</p>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-3 text-slate-600">
              <Camera size={48} strokeWidth={1} />
              <p className="text-xs font-bold uppercase tracking-widest">Your masterpiece appears here</p>
            </div>
          )}
        </div>

        {/* Prompt Input */}
        <div className="space-y-4">
          <div className="relative group">
            <textarea
              rows={3}
              placeholder="e.g., A professional headshot of a confident tech founder in a Lagos rooftop garden, sunset lighting, realistic 8k..."
              value={prompt}
              onChange={(e) => setInput(e.target.value)}
              className="w-full bg-white/5 border border-white/10 rounded-3xl py-5 px-6 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/30 focus:bg-white/10 transition-all resize-none shadow-inner"
            />
            <div className="absolute bottom-4 right-4 text-[10px] font-black text-slate-600 uppercase">
              Cost: 10 PTS
            </div>
          </div>

          <button
            onClick={handleGenerate}
            disabled={!prompt.trim() || isLoading}
            className={`w-full py-4 rounded-2xl font-black text-sm uppercase tracking-[0.2em] transition-all flex items-center justify-center gap-3
              ${prompt.trim() && !isLoading 
                ? 'gold-gradient text-black shadow-xl shadow-yellow-500/20 active:scale-95' 
                : 'bg-white/5 text-slate-600 cursor-not-allowed'}
            `}
          >
            {isLoading ? 'Processing Request' : 'Generate Portrait'}
            {!isLoading && <Send size={18} />}
          </button>
        </div>
      </main>

      <footer className="p-6 pt-0">
        <Link href="/studio/gallery" className="block text-center text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] hover:text-brand-gold transition-colors">
          View My AI Gallery →
        </Link>
      </footer>
    </div>
  );
}
