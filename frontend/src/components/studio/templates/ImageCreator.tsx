'use client';

import { useState } from 'react';
import { Loader2, Sparkles, Info } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_ASPECT_RATIOS = [
  { label: 'Square',    value: '1:1',   icon: '⬜' },
  { label: 'Portrait',  value: '9:16',  icon: '📱' },
  { label: 'Landscape', value: '16:9',  icon: '🖥️' },
  { label: 'Wide',      value: '3:2',   icon: '📸' },
];

const DEFAULT_STYLE_TAGS = [
  'Photorealistic', 'Cinematic', 'Anime', 'Oil Painting', 'Watercolour',
  'Digital Art', 'Sketch', 'Minimalist', 'Dark Fantasy', 'Vintage',
  'Afrofuturist', 'Studio Portrait',
];

// Model identity info — keyed by slug, shown as a badge inside the drawer
const MODEL_IDENTITY: Record<string, { label: string; desc: string; color: string }> = {
  'ai-photo':       { label: 'FLUX',              desc: 'Fast, high-quality image generation',       color: 'text-purple-300 bg-purple-600/15 border-purple-500/30' },
  'ai-photo-pro':   { label: 'GPT-Image',          desc: 'OpenAI GPT-Image · detailed realism',       color: 'text-blue-300 bg-blue-600/15 border-blue-500/30' },
  'ai-photo-max':   { label: 'GPT-Image Large',    desc: 'Max quality · 2x detail, slower',           color: 'text-indigo-300 bg-indigo-600/15 border-indigo-500/30' },
  'ai-photo-dream': { label: 'Seedream',            desc: 'Dreamlike aesthetics · stylised outputs',  color: 'text-pink-300 bg-pink-600/15 border-pink-500/30' },
};

export default function ImageCreator({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg        = tool.ui_config ?? {};
  const aspectRatios = cfg.aspect_ratios ?? DEFAULT_ASPECT_RATIOS;
  const styleTags    = cfg.style_tags    ?? DEFAULT_STYLE_TAGS;
  const showNeg      = cfg.show_negative_prompt ?? true;
  const showStyles   = cfg.show_style_tags      ?? true;
  // HD quality only relevant for GPT-Image tools
  const showQuality  = cfg.show_quality_toggle  ?? ['ai-photo-pro', 'ai-photo-max'].includes(tool.slug ?? '');

  const [prompt,         setPrompt]         = useState('');
  const [aspect,         setAspect]         = useState(cfg.default_aspect ?? '1:1');
  const [negPrompt,      setNegPrompt]       = useState('');
  const [selectedStyles, setSelectedStyles] = useState<string[]>([]);
  const [showNegBox,     setShowNegBox]      = useState(false);
  const [quality,        setQuality]         = useState<'standard' | 'hd'>('standard');

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = prompt.trim().length >= 3;
  const modelInfo = MODEL_IDENTITY[tool.slug ?? ''];

  function toggleStyle(s: string) {
    setSelectedStyles((prev) =>
      prev.includes(s) ? prev.filter((t) => t !== s) : [...prev, s],
    );
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const stylePrefix = selectedStyles.length > 0 ? `[${selectedStyles.join(', ')}] ` : '';
    const payload: GeneratePayload = {
      prompt:          stylePrefix + prompt.trim(),
      aspect_ratio:    aspect,
      style_tags:      selectedStyles.length > 0 ? selectedStyles : undefined,
      negative_prompt: negPrompt.trim() || undefined,
      extra_params: {
        quality: showQuality ? quality : undefined,
      },
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Model identity badge ── */}
      {modelInfo && (
        <div className={cn(
          'flex items-start gap-2 rounded-xl border px-3 py-2.5',
          modelInfo.color,
        )}>
          <Info size={13} className="flex-shrink-0 mt-0.5 opacity-70" />
          <div>
            <span className="text-xs font-bold">{modelInfo.label}</span>
            <span className="text-xs opacity-70 ml-1.5">— {modelInfo.desc}</span>
          </div>
        </div>
      )}

      {/* ── Aspect ratio ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Aspect Ratio</label>
        <div className="grid grid-cols-4 gap-2">
          {aspectRatios.map((ar: { value: string; label: string; icon?: string }) => (
            <button
              key={ar.value}
              onClick={() => setAspect(ar.value)}
              className={cn(
                'flex flex-col items-center gap-1 py-2.5 rounded-xl border text-xs font-medium transition-all',
                aspect === ar.value
                  ? 'bg-purple-600/25 border-purple-500/60 text-purple-200'
                  : 'border-white/10 text-white/45 hover:border-white/25 hover:text-white/70',
              )}
            >
              {ar.icon && <span className="text-base leading-none">{ar.icon}</span>}
              <span className="text-[10px] font-semibold">{ar.label}</span>
              <span className="text-[9px] text-white/30">{ar.value}</span>
            </button>
          ))}
        </div>
      </div>

      {/* ── Quality toggle (GPT-Image only) ── */}
      {showQuality && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Quality</label>
          <div className="flex rounded-xl overflow-hidden border border-white/10 w-fit">
            <button
              onClick={() => setQuality('standard')}
              className={cn(
                'px-5 py-2 text-xs font-semibold transition-all',
                quality === 'standard' ? 'bg-purple-600 text-white' : 'text-white/55 hover:text-white/80',
              )}
            >
              Standard
            </button>
            <button
              onClick={() => setQuality('hd')}
              className={cn(
                'px-5 py-2 text-xs font-semibold transition-all flex items-center gap-1',
                quality === 'hd' ? 'bg-purple-600 text-white' : 'text-white/55 hover:text-white/80',
              )}
            >
              ✦ HD
            </button>
          </div>
          <p className="text-white/25 text-[11px] mt-1">HD uses more detail passes — slightly slower</p>
        </div>
      )}

      {/* ── Style tags ── */}
      {showStyles && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Style</label>
          <div className="flex flex-wrap gap-1.5">
            {styleTags.map((s: string) => (
              <button
                key={s}
                onClick={() => toggleStyle(s)}
                className={cn(
                  'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                  selectedStyles.includes(s)
                    ? 'bg-purple-600 text-white border-purple-500'
                    : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                )}
              >
                {s}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* ── Prompt ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
          Describe your image
        </label>
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder={cfg.prompt_placeholder ?? 'e.g. A majestic lion standing on a cliff at golden hour, dramatic lighting, ultra-detailed, 8K…'}
          rows={4}
          autoFocus
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />
        <p className="text-white/25 text-[11px] mt-1">{prompt.length}/500 characters</p>
      </div>

      {/* ── Negative prompt (collapsible) ── */}
      {showNeg && (
        <div>
          <button
            onClick={() => setShowNegBox((v) => !v)}
            className="text-white/40 text-xs hover:text-white/65 transition-colors"
          >
            {showNegBox ? '▲ Hide' : '▼ Add'} negative prompt (optional)
          </button>
          {showNegBox && (
            <textarea
              value={negPrompt}
              onChange={(e) => setNegPrompt(e.target.value)}
              placeholder={cfg.negative_prompt_placeholder ?? 'Things to avoid: blurry, low quality, watermark, extra fingers, distorted…'}
              rows={2}
              className="nexus-input resize-none w-full text-sm leading-relaxed mt-2"
            />
          )}
        </div>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-purple-600 to-pink-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-purple-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Generating…</>
          : <><Sparkles size={15} /> Generate Image →</>
        }
      </button>
    </div>
  );
}
