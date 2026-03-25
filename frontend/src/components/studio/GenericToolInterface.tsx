"use client";

import React, { useState } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { Sparkles, Send, Download, ArrowLeft, RefreshCw, AlertCircle, Upload, Image as ImageIcon } from 'lucide-react';
import Link from 'next/link';

export default function GenericToolInterface({ params }: { params: { id: string } }) {
  const searchParams = useSearchParams();
  const toolName = searchParams.get('name') || 'Nexus Tool';
  const pointCost = searchParams.get('cost') || '0';
  
  const [input, setInput] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Determine tool capabilities based on ID
  const isUploadTool = params.id === 'bg-remover' || params.id === 'animate-photo';
  const isGalleryTool = params.id === 'animate-photo' || params.id === 'video-story';

  const handleAction = async () => {
    if (!input.trim() && !file && !isLoading) return;

    setIsLoading(true);
    setError(null);
    setResult(null);

    try {
      // In production: Generic POST /api/v1/studio/generate/[category]
      setTimeout(() => {
        setResult(`https://cdn.loyalty-nexus.ai/generated/mock-${params.id}.webp`);
        setIsLoading(false);
      }, 3500);
    } catch (err: any) {
      setError(err.message || 'Processing failed. Points refunded.');
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
        <div>
          <h1 className="text-lg font-black tracking-tight italic uppercase leading-none">{toolName}</h1>
          <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest text-brand-gold mt-1">
            Studio Engine • {pointCost} PTS
          </p>
        </div>
      </header>

      <main className="flex-grow p-6 space-y-8 overflow-y-auto no-scrollbar">
        {result ? (
          <div className="glass rounded-[2.5rem] p-10 border border-brand-gold/30 flex flex-col items-center text-center space-y-6 animate-in zoom-in-95">
            <div className="w-full aspect-video rounded-3xl bg-white/5 overflow-hidden flex items-center justify-center">
              {params.id.includes('podcast') ? (
                <RefreshCw className="text-brand-gold w-12 h-12 animate-pulse" />
              ) : (
                <ImageIcon className="text-slate-700 w-20 h-20" />
              )}
            </div>
            <div>
              <h3 className="text-xl font-black text-white italic">Generation Complete</h3>
              <p className="text-sm text-slate-500 font-bold uppercase tracking-widest mt-1">Asset added to your gallery</p>
            </div>
            <button className="gold-gradient text-black px-10 py-4 rounded-2xl font-black text-sm uppercase tracking-[0.2em] flex items-center gap-3 shadow-xl">
              <Download size={20} /> Download Result
            </button>
          </div>
        ) : (
          <div className="space-y-8">
            {/* Context Awareness Section */}
            <div className="glass rounded-3xl p-6 border border-white/5 space-y-3">
              <h2 className="text-xs font-black text-brand-gold uppercase tracking-[0.2em] flex items-center gap-2">
                <Sparkles size={14} /> How to use
              </h2>
              <p className="text-xs text-slate-400 font-medium leading-relaxed italic">
                {isUploadTool ? 'Upload a file from your device to begin.' : 'Enter a detailed prompt or topic below.'}
              </p>
            </div>

            {/* Input Hub */}
            <div className="space-y-6">
              {isUploadTool && (
                <div className="w-full h-48 rounded-3xl border-2 border-dashed border-white/10 hover:border-brand-gold/30 transition-all flex flex-col items-center justify-center space-y-3 bg-white/5 cursor-pointer group">
                  <Upload className="text-slate-500 group-hover:text-brand-gold transition-colors" size={32} />
                  <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest group-hover:text-white transition-colors">Tap to upload file</p>
                </div>
              )}

              {isGalleryTool && (
                <div className="flex items-center justify-between px-2">
                  <span className="text-[10px] font-black text-slate-500 uppercase tracking-widest">Or select from gallery</span>
                  <button className="text-[10px] font-black text-brand-gold uppercase tracking-widest hover:underline">Open Gallery</button>
                </div>
              )}

              {!isUploadTool && (
                <textarea
                  rows={5}
                  placeholder="Enter details..."
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  className="w-full bg-white/5 border border-white/10 rounded-3xl py-6 px-8 text-sm text-white placeholder:text-slate-700 focus:outline-none focus:border-brand-gold/30 focus:bg-white/10 transition-all resize-none shadow-inner"
                />
              )}

              <button
                onClick={handleAction}
                disabled={(!input.trim() && !file) || isLoading}
                className={`w-full py-5 rounded-2xl font-black text-sm uppercase tracking-[0.2em] transition-all flex items-center justify-center gap-3
                  ${(input.trim() || file) && !isLoading 
                    ? 'gold-gradient text-black shadow-xl' 
                    : 'bg-white/5 text-slate-600 cursor-not-allowed'}
                `}
              >
                {isLoading ? (
                  <>
                    <RefreshCw className="w-5 h-5 animate-spin" />
                    Processing Pipeline
                  </>
                ) : (
                  <>
                    Process Request
                    <Send size={18} />
                  </>
                )}
              </button>
            </div>
          </div>
        )}
      </main>

      <footer className="p-8 text-center opacity-40">
        <p className="text-[9px] font-bold text-slate-500 uppercase tracking-[0.3em]">
          End-to-End Encryption • Nexus Secure Cloud
        </p>
      </footer>
    </div>
  );
}
