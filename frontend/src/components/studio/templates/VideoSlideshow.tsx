'use client';

import { useState, useRef } from 'react';
import {
  Loader2, X, Plus, Film, AlertTriangle, GripVertical,
} from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

const DEFAULT_ASPECTS = [
  { label: 'Portrait',  value: '9:16' },
  { label: 'Square',    value: '1:1'  },
  { label: 'Landscape', value: '16:9' },
];

interface Slide { url: string; preview: string; }

export default function VideoSlideshow({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg       = tool.ui_config ?? {};
  const maxImages = cfg.max_images ?? 6;
  const minImages = cfg.min_images ?? 3;
  const aspects   = cfg.aspect_ratios ?? DEFAULT_ASPECTS;

  const [slides,   setSlides]   = useState<Slide[]>([]);
  const [caption,  setCaption]  = useState('');
  const [aspect,   setAspect]   = useState<string>(cfg.default_aspect ?? aspects[0]?.value ?? '9:16');
  const [uploading, setUploading] = useState(false);
  const [error,    setError]    = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = slides.length >= minImages;

  async function addFiles(files: FileList) {
    setError('');
    const room = maxImages - slides.length;
    const picked = Array.from(files).slice(0, room);
    setUploading(true);
    try {
      for (const f of picked) {
        if (!f.type.startsWith('image/')) continue;
        const preview = URL.createObjectURL(f);
        const r = await api.uploadAsset(f);
        setSlides((prev) => (prev.length < maxImages ? [...prev, { url: r.url, preview }] : prev));
      }
    } catch {
      setError('Upload failed — please try again.');
    } finally {
      setUploading(false);
      if (fileRef.current) fileRef.current.value = '';
    }
  }

  function removeSlide(i: number) {
    setSlides((prev) => prev.filter((_, idx) => idx !== i));
  }
  function move(i: number, dir: -1 | 1) {
    setSlides((prev) => {
      const j = i + dir;
      if (j < 0 || j >= prev.length) return prev;
      const next = [...prev];
      [next[i], next[j]] = [next[j], next[i]];
      return next;
    });
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford || uploading) return;
    const payload: GeneratePayload = {
      prompt:       caption.trim(),
      aspect_ratio: aspect,
      extra_params: {
        images:  slides.map((s) => s.url),
        caption: caption.trim() || undefined,
      },
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center gap-2 bg-gradient-to-r from-fuchsia-500/10 to-cyan-500/10 border border-fuchsia-500/20 rounded-xl px-3 py-2">
        <Film size={12} className="text-fuchsia-400 flex-shrink-0" />
        <p className="text-fuchsia-300/70 text-[11px]">
          <span className="font-semibold text-fuchsia-300">Video Slideshow</span>
          {' '}— turn your images into a montage with transitions and captions.
        </p>
      </div>

      {cfg.coming_soon_note && (
        <div className="flex items-start gap-2 bg-amber-500/8 border border-amber-500/20 rounded-xl px-3 py-2.5">
          <AlertTriangle size={13} className="text-amber-400 flex-shrink-0 mt-0.5" />
          <p className="text-amber-300/75 text-xs leading-relaxed">{cfg.coming_soon_note}</p>
        </div>
      )}

      {/* ── 1. Images ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          1. Images <span className="text-fuchsia-400">*</span>
          <span className="text-white/30 normal-case tracking-normal ml-2">{slides.length}/{maxImages} · min {minImages}</span>
        </label>
        <div className="grid grid-cols-3 gap-2">
          {slides.map((s, i) => (
            <div key={i} className="relative rounded-xl overflow-hidden border border-white/10 aspect-square group">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={s.preview} alt={`slide ${i + 1}`} className="w-full h-full object-cover" />
              <div className="absolute inset-0 bg-black/0 group-hover:bg-black/40 transition-all flex items-center justify-center gap-1 opacity-0 group-hover:opacity-100">
                <button onClick={() => move(i, -1)} className="w-7 h-7 rounded-lg bg-black/70 text-white/80 flex items-center justify-center text-xs">↑</button>
                <button onClick={() => move(i, 1)}  className="w-7 h-7 rounded-lg bg-black/70 text-white/80 flex items-center justify-center text-xs">↓</button>
                <button onClick={() => removeSlide(i)} className="w-7 h-7 rounded-lg bg-red-600/80 text-white flex items-center justify-center"><X size={13} /></button>
              </div>
              <span className="absolute top-1 left-1 w-5 h-5 rounded-md bg-black/70 text-white/80 text-[10px] font-bold flex items-center justify-center">{i + 1}</span>
            </div>
          ))}
          {slides.length < maxImages && (
            <button
              onClick={() => fileRef.current?.click()}
              className="aspect-square rounded-xl border-2 border-dashed border-white/15 hover:border-fuchsia-500/40 flex flex-col items-center justify-center text-white/40 hover:text-white/70 transition-all"
            >
              {uploading ? <Loader2 size={20} className="animate-spin" /> : <><Plus size={20} /><span className="text-[10px] mt-1">Add</span></>}
            </button>
          )}
        </div>
        <input ref={fileRef} type="file" accept="image/*" multiple className="hidden"
          onChange={(e) => { if (e.target.files) addFiles(e.target.files); }} />
        <p className="text-[11px] text-white/30 mt-2 flex items-center gap-1"><GripVertical size={11} /> Hover a tile to reorder or remove.</p>
      </div>

      {/* ── 2. Caption ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">2. Caption <span className="text-white/30 normal-case">(optional)</span></label>
        <input
          value={caption}
          onChange={(e) => setCaption(e.target.value.slice(0, 80))}
          placeholder="e.g. Recharge & Win with Loyalty Nexus"
          className="w-full bg-white/[0.03] border border-white/10 focus:border-fuchsia-500/40 rounded-xl px-3 py-2.5 text-sm text-white/90 placeholder:text-white/25 outline-none transition-all"
        />
      </div>

      {/* ── 3. Format ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">3. Format</label>
        <div className="flex gap-2 flex-wrap">
          {aspects.map((ar) => (
            <button key={ar.value} onClick={() => setAspect(ar.value)}
              className={cn('text-[12px] px-3 py-1.5 rounded-lg border font-semibold transition-all',
                aspect === ar.value ? 'bg-cyan-600 text-white border-cyan-500' : 'bg-white/[0.03] text-white/60 border-white/10 hover:border-white/25')}>
              {ar.label}
            </button>
          ))}
        </div>
      </div>

      {error && <p className="text-[12px] text-red-400/80">{error}</p>}

      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || uploading || !canAfford}
        className={cn('w-full flex items-center justify-center gap-2 rounded-xl py-3 font-bold text-sm transition-all',
          !isValid || !canAfford ? 'bg-white/5 text-white/30 cursor-not-allowed'
            : 'bg-gradient-to-r from-fuchsia-600 to-cyan-600 text-white hover:opacity-90 active:scale-[0.99]')}
      >
        {uploading ? <><Loader2 size={16} className="animate-spin" /> Uploading…</>
          : isLoading ? <><Loader2 size={16} className="animate-spin" /> Rendering video…</>
          : !isValid ? `Add at least ${minImages} images`
          : !canAfford ? `Need ${(tool.point_cost - userPoints).toLocaleString()} more points`
          : <><Film size={16} /> Create Slideshow · {tool.is_free ? 'Free' : `${tool.point_cost} pts`}</>}
      </button>
    </div>
  );
}
