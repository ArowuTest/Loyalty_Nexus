"use client";

import React, { useEffect, useState } from 'react';
import {
  LayoutGrid, Download, ArrowLeft,
  Image as ImageIcon, Music, FileText, Film,
  Calendar, RefreshCw,
} from 'lucide-react';
import Link from 'next/link';
import api from '@/lib/api';

// Mirrors entities.AIGeneration JSON fields returned by GET /api/v1/studio/gallery
interface Generation {
  id:           string;
  tool_slug:    string;
  prompt:       string;
  status:       string;
  output_url?:  string;
  output_text?: string;
  created_at:   string;
}

// Derive display type from tool_slug
function mediaType(slug: string): 'image' | 'audio' | 'video' | 'text' {
  if (/photo|image|bg-remover/.test(slug))    return 'image';
  if (/audio|music|jingle|narrate|podcast|tts|bg-music|song|instrumental/.test(slug)) return 'audio';
  if (/video|animate|cinematic/.test(slug))   return 'video';
  return 'text';
}

function toolLabel(slug: string): string {
  return slug
    .split('-')
    .map(w => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

export default function Gallery() {
  const [items,   setItems]   = useState<Generation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error,   setError]   = useState('');

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await api.getGallery() as { items: Generation[]; count: number };
      setItems(res.items ?? []);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load gallery');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  return (
    <div className="min-h-screen bg-black text-white max-w-screen-md mx-auto border-x border-white/5 flex flex-col">
      {/* Header */}
      <header className="glass border-b border-white/10 px-6 py-4 flex items-center gap-4 sticky top-0 z-50">
        <Link href="/studio" className="p-2 -ml-2 text-slate-400 hover:text-brand-gold transition-colors">
          <ArrowLeft size={20} />
        </Link>
        <div className="flex items-center gap-3 flex-1">
          <div className="w-10 h-10 rounded-2xl bg-white/5 flex items-center justify-center text-slate-400">
            <LayoutGrid size={20} />
          </div>
          <div>
            <h1 className="text-lg font-black tracking-tight italic uppercase">My Studio Gallery</h1>
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest text-brand-gold">
              Assets stored for 30 days
            </p>
          </div>
        </div>
        <button
          onClick={load}
          disabled={loading}
          className="p-2 text-slate-500 hover:text-brand-gold transition-colors"
          title="Refresh"
        >
          <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
        </button>
      </header>

      <main className="flex-grow p-6 overflow-y-auto no-scrollbar">
        {/* Loading */}
        {loading && (
          <div className="flex flex-col items-center justify-center h-64 gap-3 text-slate-500">
            <RefreshCw size={28} className="animate-spin" />
            <p className="text-sm font-bold uppercase tracking-widest">Loading your creations…</p>
          </div>
        )}

        {/* Error */}
        {!loading && error && (
          <div className="glass rounded-2xl border border-red-500/20 p-6 text-center space-y-2">
            <p className="text-red-400 font-bold text-sm">⚠ {error}</p>
            <button onClick={load}
              className="text-xs text-brand-gold border border-brand-gold/30 px-4 py-2 rounded-xl">
              Retry
            </button>
          </div>
        )}

        {/* Grid */}
        {!loading && !error && items.length > 0 && (
          <div className="grid grid-cols-2 gap-4">
            {items.map(item => {
              const mt  = mediaType(item.tool_slug);
              const url = item.output_url || '';
              return (
                <div
                  key={item.id}
                  className="group relative glass rounded-2xl border border-white/5 overflow-hidden aspect-[3/4] flex flex-col transition-all hover:border-brand-gold/30"
                >
                  {/* Preview */}
                  <div className="relative flex-grow bg-white/5 overflow-hidden">
                    {mt === 'image' && url && (
                      <img src={url} alt={item.prompt}
                        className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110" />
                    )}
                    {mt === 'image' && !url && (
                      <div className="w-full h-full flex items-center justify-center text-slate-600">
                        <ImageIcon size={40} />
                      </div>
                    )}
                    {mt === 'audio' && (
                      <div className="w-full h-full flex flex-col items-center justify-center gap-3
                        text-brand-gold bg-brand-gold/5 px-3">
                        <Music size={36} />
                        {url && (
                          <audio controls className="w-full max-w-[140px]" src={url}>
                            <track kind="captions" />
                          </audio>
                        )}
                      </div>
                    )}
                    {mt === 'video' && (
                      <div className="w-full h-full flex items-center justify-center text-purple-400 bg-purple-400/5">
                        <Film size={40} />
                      </div>
                    )}
                    {mt === 'text' && (
                      <div className="w-full h-full flex items-center justify-center text-blue-400 bg-blue-400/5 p-4">
                        <FileText size={36} />
                      </div>
                    )}

                    {/* Download overlay */}
                    {url && (
                      <div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                        <a href={url} download target="_blank" rel="noreferrer"
                          className="gold-gradient text-black p-3 rounded-full shadow-2xl">
                          <Download size={20} />
                        </a>
                      </div>
                    )}
                  </div>

                  {/* Footer */}
                  <div className="p-3 space-y-1 bg-black/40 backdrop-blur-md">
                    <p className="text-[10px] font-black text-brand-gold uppercase tracking-tighter line-clamp-1">
                      {toolLabel(item.tool_slug)}
                    </p>
                    <p className="text-[11px] text-slate-300 font-medium line-clamp-1 italic">
                      &ldquo;{item.prompt}&rdquo;
                    </p>
                    <div className="flex items-center gap-1 pt-1 opacity-60">
                      <Calendar size={10} className="text-slate-500" />
                      <span className="text-[9px] font-bold text-slate-500 uppercase">
                        {new Date(item.created_at).toLocaleDateString('en-GB', {
                          day: '2-digit', month: 'short', year: 'numeric',
                        })}
                      </span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Empty state */}
        {!loading && !error && items.length === 0 && (
          <div className="h-full flex flex-col items-center justify-center text-center space-y-4 opacity-30 pt-20">
            <ImageIcon size={64} strokeWidth={1} />
            <div className="space-y-1">
              <p className="text-sm font-black uppercase tracking-widest">Gallery Empty</p>
              <p className="text-xs font-medium">Your creations will appear here once you generate something.</p>
            </div>
            <Link href="/studio"
              className="text-[10px] border border-white/10 px-4 py-2 rounded-xl text-slate-400 hover:text-brand-gold transition-colors">
              ← Back to Studio
            </Link>
          </div>
        )}
      </main>
    </div>
  );
}
