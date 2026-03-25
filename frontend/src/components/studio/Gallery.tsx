"use client";

import React, { useState } from 'react';
import { LayoutGrid, Download, ArrowLeft, Image as ImageIcon, Music, FileText, Calendar, Trash2 } from 'lucide-react';
import Link from 'next/link';

interface GalleryItem {
  id: string;
  type: 'image' | 'audio' | 'pdf';
  toolName: string;
  prompt: string;
  url: string;
  createdAt: Date;
}

const MOCK_GALLERY: GalleryItem[] = [
  {
    id: '1',
    type: 'image',
    toolName: 'My AI Photo',
    prompt: 'Tech founder in Lagos rooftop garden...',
    url: 'https://static-s3.skyworkcdn.com/fe/skywork-site-assets/images/skybot/avatar1-new.png',
    createdAt: new Date(),
  },
  {
    id: '2',
    type: 'pdf',
    toolName: 'Study Guide',
    prompt: 'WAEC Chemistry: Organic Reactions',
    url: 'https://cdn.loyalty-nexus.ai/learning/mock.pdf',
    createdAt: new Date(),
  },
  {
    id: '3',
    type: 'audio',
    toolName: 'My Podcast',
    prompt: 'Benefits of Solar Energy in Nigeria',
    url: 'https://cdn.loyalty-nexus.ai/learning/mock.mp3',
    createdAt: new Date(),
  }
];

export default function Gallery() {
  const [items] = useState<GalleryItem[]>(MOCK_GALLERY);

  return (
    <div className="min-h-screen bg-black text-white max-w-screen-md mx-auto border-x border-white/5 flex flex-col">
      {/* Header */}
      <header className="glass border-b border-white/10 px-6 py-4 flex items-center gap-4 sticky top-0 z-50">
        <Link href="/studio" className="p-2 -ml-2 text-slate-400 hover:text-brand-gold transition-colors">
          <ArrowLeft size={20} />
        </Link>
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-2xl bg-white/5 flex items-center justify-center text-slate-400">
            <LayoutGrid size={20} />
          </div>
          <div>
            <h1 className="text-lg font-black tracking-tight italic uppercase">My Studio Gallery</h1>
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest text-brand-gold">Assets stored for 30 days</p>
          </div>
        </div>
      </header>

      <main className="flex-grow p-6 overflow-y-auto no-scrollbar">
        {items.length > 0 ? (
          <div className="grid grid-cols-2 gap-4">
            {items.map((item) => (
              <div key={item.id} className="group relative glass rounded-2xl border border-white/5 overflow-hidden aspect-[3/4] flex flex-col transition-all hover:border-brand-gold/30">
                <div className="relative flex-grow bg-white/5 overflow-hidden">
                  {item.type === 'image' && (
                    <img src={item.url} alt={item.prompt} className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110" />
                  )}
                  {item.type === 'audio' && (
                    <div className="w-full h-full flex items-center justify-center text-brand-gold bg-brand-gold/5">
                      <Music size={40} />
                    </div>
                  )}
                  {item.type === 'pdf' && (
                    <div className="w-full h-full flex items-center justify-center text-blue-400 bg-blue-400/5">
                      <FileText size={40} />
                    </div>
                  )}
                  
                  {/* Download Overlay */}
                  <div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                    <button className="gold-gradient text-black p-3 rounded-full shadow-2xl">
                      <Download size={20} />
                    </button>
                  </div>
                </div>

                <div className="p-3 space-y-1 bg-black/40 backdrop-blur-md">
                  <p className="text-[10px] font-black text-brand-gold uppercase tracking-tighter line-clamp-1">{item.toolName}</p>
                  <p className="text-[11px] text-slate-300 font-medium line-clamp-1 italic">"{item.prompt}"</p>
                  <div className="flex items-center justify-between pt-1 opacity-60">
                    <div className="flex items-center gap-1 text-[9px] font-bold text-slate-500 uppercase">
                      <Calendar size={10} /> {item.createdAt.toLocaleDateString()}
                    </div>
                    <button className="text-slate-500 hover:text-red-400 transition-colors">
                      <Trash2 size={12} />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="h-full flex flex-col items-center justify-center text-center space-y-4 opacity-30">
            <ImageIcon size={64} strokeWidth={1} />
            <div className="space-y-1">
              <p className="text-sm font-black uppercase tracking-widest">Gallery Empty</p>
              <p className="text-xs font-medium">Your creations will be saved here.</p>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
