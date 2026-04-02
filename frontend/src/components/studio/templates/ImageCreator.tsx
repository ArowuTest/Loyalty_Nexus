'use client';

import { useState } from 'react';
import { Loader2, Sparkles, Info, Shuffle, ChevronDown, ChevronUp, Wand2 } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_ASPECT_RATIOS = [
  { label: 'Square',    value: '1:1',   w: 1,   h: 1   },
  { label: 'Portrait',  value: '9:16',  w: 9,   h: 16  },
  { label: 'Landscape', value: '16:9',  w: 16,  h: 9   },
  { label: 'Wide',      value: '3:2',   w: 3,   h: 2   },
];

const DEFAULT_STYLE_TAGS = [
  'Photorealistic', 'Cinematic', 'Anime', 'Oil Painting', 'Watercolour',
  'Digital Art', 'Sketch', 'Minimalist', 'Dark Fantasy', 'Vintage',
  'Afrofuturist', 'Studio Portrait',
];

const PROMPT_INSPIRATIONS = [
  'A majestic lion standing on a cliff at golden hour, dramatic lighting, ultra-detailed',
  'A futuristic Lagos skyline at night, neon lights reflecting on rain-soaked streets, cinematic',
  'A serene Yoruba village at dawn, mist over the river, oil painting style',
  'Portrait of a Nigerian queen in traditional attire, studio lighting, 8K detail',
  'An astronaut floating above Earth, Africa visible below, photorealistic',
  'A vibrant Afrobeats concert crowd, colourful lights, motion blur, energetic',
];

const MODEL_IDENTITY: Record<string, { label: string; desc: string; color: string; dot: string }> = {
  'ai-photo':       { label: 'FLUX',           desc: 'Fast, high-quality image generation',     color: 'text-purple-300 bg-purple-600/15 border-purple-500/30',  dot: 'bg-purple-400' },
  'ai-photo-pro':   { label: 'GPT-Image',       desc: 'OpenAI GPT-Image · detailed realism',     color: 'text-blue-300 bg-blue-600/15 border-blue-500/30',        dot: 'bg-blue-400' },
  'ai-photo-max':   { label: 'GPT-Image Large', desc: 'Max quality · 2× detail, slower',         color: 'text-indigo-300 bg-indigo-600/15 border-indigo-500/30',  dot: 'bg-indigo-400' },
  'ai-photo-dream': { label: 'Seedream',        desc: 'Dreamlike aesthetics · stylised outputs', color: 'text-pink-300 bg-pink-600/15 border-pink-500/30',        dot: 'bg-pink-400' },
};

export default function ImageCreator({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg          = tool.ui_config ?? {};
  const aspectRatios = cfg.aspect_ratios ?? DEFAULT_ASPECT_RATIOS;
  const styleTags    = cfg.style_tags    ?? DEFAULT_STYLE_TAGS;
  const showNeg      = cfg.show_negative_prompt ?? true;
  const showStyles   = cfg.show_style_tags      ?? true;
  const showQuality  = cfg.show_quality_toggle  ?? ['ai-photo-pro', 'ai-photo-max'].includes(tool.slug ?? '');

  const [prompt,         setPrompt]         = useState('');
  const [aspect,         setAspect]         = useState(cfg.default_aspect ?? '1:1');
  const [negPrompt,      setNegPrompt]      = useState('');
  const [selectedStyles, setSelectedStyles] = useState<string[]>([]);
  const [showNegBox,     setShowNegBox]     = useState(false);
  const [showInspo,      setShowInspo]      = useState(false);
  const [quality,        setQuality]        = useState<'standard' | 'hd'>('standard');

  const canAfford  = tool.is_free || userPoints >= tool.point_cost;
  const isValid    = prompt.trim().length >= 3;
  const modelInfo  = MODEL_IDENTITY[tool.slug ?? ''];
  const charPct    = Math.min(100, (prompt.length / 500) * 100);

  // Find current aspect ratio dimensions for the live preview box
  const currentAR = (aspectRatios as { value: string; w?: number; h?: number; label: string }[])
    .find((ar) => ar.value === aspect) ?? { w: 1, h: 1, label: 'Square', value: '1:1' };
  const previewW = 56;
  const previewH = currentAR.w && currentAR.h
    ? Math.round((currentAR.h / currentAR.w) * previewW)
    : previewW;

  function toggleStyle(s: string) {
    setSelectedStyles((prev) =>
      prev.includes(s) ? prev.filter((t) => t !== s) : prev.length < 4 ? [...prev, s] : prev,
    );
  }

  function surpriseMe() {
    const random = PROMPT_INSPIRATIONS[Math.floor(Math.random() * PROMPT_INSPIRATIONS.length)];
    setPrompt(random);
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
        <div className={cn('flex items-center gap-2.5 rounded-xl border px-3 py-2.5', modelInfo.color)}>
          <span className={cn('w-2 h-2 rounded-full flex-shrink-0', modelInfo.dot)} />
          <div className="flex items-center gap-1.5 flex-1 min-w-0">
            <span className="text-xs font-bold">{modelInfo.label}</span>
            <span className="text-xs opacity-60">—</span>
            <span className="text-xs opacity-70 truncate">{modelInfo.desc}</span>
          </div>
          <Info size={12} className="opacity-40 flex-shrink-0" />
        </div>
      )}

      {/* ── Aspect ratio with live canvas preview ── */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Aspect Ratio</label>
          {/* Live canvas preview */}
          <div className="flex items-center gap-2">
            <div
              className="border-2 border-purple-500/50 bg-purple-600/10 rounded transition-all duration-300"
              style={{ width: `${previewW}px`, height: `${previewH}px`, minHeight: '20px' }}
            />
            <span className="text-white/35 text-[11px] font-mono">{aspect}</span>
          </div>
        </div>
        <div className="grid grid-cols-4 gap-2">
          {(aspectRatios as { value: string; label: string; icon?: string }[]).map((ar) => (
            <button
              key={ar.value}
              onClick={() => setAspect(ar.value)}
              className={cn(
                'flex flex-col items-center gap-1 py-3 rounded-xl border text-xs font-medium transition-all',
                aspect === ar.value
                  ? 'bg-purple-600/25 border-purple-500/60 text-purple-200 shadow-sm shadow-purple-900/30'
                  : 'border-white/10 text-white/45 hover:border-white/25 hover:text-white/70 hover:bg-white/[0.03]',
              )}
            >
              {ar.icon && <span className="text-base leading-none">{ar.icon}</span>}
              <span className="text-[10px] font-semibold">{ar.label}</span>
              <span className="text-[9px] text-white/30 font-mono">{ar.value}</span>
            </button>
          ))}
        </div>
      </div>

      {/* ── Quality toggle (GPT-Image only) ── */}
      {showQuality && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Quality</label>
          <div className="flex rounded-xl overflow-hidden border border-white/10 w-fit">
            {(['standard', 'hd'] as const).map((q) => (
              <button
                key={q}
                onClick={() => setQuality(q)}
                className={cn(
                  'px-5 py-2 text-xs font-semibold transition-all',
                  quality === q ? 'bg-purple-600 text-white' : 'text-white/55 hover:text-white/80',
                )}
              >
                {q === 'hd' ? '✦ HD' : 'Standard'}
              </button>
            ))}
          </div>
          <p className="text-white/25 text-[11px] mt-1">HD uses more detail passes — slightly slower</p>
        </div>
      )}

      {/* ── Style tags (max 4) ── */}
      {showStyles && (
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Style</label>
            <span className="text-white/25 text-[10px]">{selectedStyles.length}/4 selected</span>
          </div>
          <div className="flex flex-wrap gap-1.5">
            {(styleTags as string[]).map((s) => (
              <button
                key={s}
                onClick={() => toggleStyle(s)}
                disabled={!selectedStyles.includes(s) && selectedStyles.length >= 4}
                className={cn(
                  'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                  selectedStyles.includes(s)
                    ? 'bg-purple-600 text-white border-purple-500 shadow-sm shadow-purple-900/30'
                    : selectedStyles.length >= 4
                      ? 'text-white/20 border-white/8 cursor-not-allowed'
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
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            Describe your image
          </label>
          <div className="flex items-center gap-2">
            {/* Character ring */}
            <svg width="20" height="20" viewBox="0 0 20 20" className="flex-shrink-0">
              <circle cx="10" cy="10" r="8" fill="none" stroke="rgba(255,255,255,0.08)" strokeWidth="2.5" />
              <circle
                cx="10" cy="10" r="8" fill="none"
                stroke={charPct > 90 ? '#f87171' : charPct > 70 ? '#f59e0b' : '#a855f7'}
                strokeWidth="2.5"
                strokeDasharray={`${2 * Math.PI * 8}`}
                strokeDashoffset={`${2 * Math.PI * 8 * (1 - charPct / 100)}`}
                strokeLinecap="round"
                transform="rotate(-90 10 10)"
                className="transition-all"
              />
            </svg>
            <button
              onClick={surpriseMe}
              title="Surprise me with a random prompt"
              className="flex items-center gap-1 text-white/35 hover:text-purple-300 transition-colors text-[11px] font-medium"
            >
              <Shuffle size={12} /> Surprise me
            </button>
          </div>
        </div>
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value.slice(0, 500))}
          placeholder={cfg.prompt_placeholder ?? 'e.g. A majestic lion standing on a cliff at golden hour, dramatic lighting, ultra-detailed, 8K…'}
          rows={4}
          autoFocus
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />

        {/* Prompt inspirations */}
        <button
          onClick={() => setShowInspo((v) => !v)}
          className="flex items-center gap-1 text-white/30 hover:text-white/55 transition-colors text-[11px] mt-1.5"
        >
          <Wand2 size={11} />
          {showInspo ? 'Hide' : 'Show'} prompt ideas
          {showInspo ? <ChevronUp size={11} /> : <ChevronDown size={11} />}
        </button>
        {showInspo && (
          <div className="mt-2 grid grid-cols-1 gap-1.5">
            {PROMPT_INSPIRATIONS.map((inspo) => (
              <button
                key={inspo}
                onClick={() => { setPrompt(inspo); setShowInspo(false); }}
                className="text-left text-xs text-white/45 hover:text-white/75 hover:bg-white/[0.04] px-3 py-2 rounded-lg border border-white/[0.06] hover:border-white/15 transition-all truncate"
              >
                {inspo}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* ── Negative prompt (collapsible) ── */}
      {showNeg && (
        <div>
          <button
            onClick={() => setShowNegBox((v) => !v)}
            className="flex items-center gap-1.5 text-white/35 text-xs hover:text-white/60 transition-colors"
          >
            {showNegBox ? <ChevronUp size={11} /> : <ChevronDown size={11} />}
            {showNegBox ? 'Hide' : 'Add'} negative prompt <span className="text-white/20">(optional)</span>
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
          'w-full py-4 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-purple-600 to-pink-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-purple-900/40'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading ? (
          <>
            <Loader2 size={15} className="animate-spin" />
            <span>Generating your image…</span>
          </>
        ) : (
          <>
            <Sparkles size={15} />
            <span>Generate Image</span>
            {!canAfford && <span className="text-xs opacity-60 ml-1">(insufficient points)</span>}
          </>
        )}
      </button>

      {/* Cost hint */}
      {!tool.is_free && (
        <p className="text-white/20 text-[11px] text-center -mt-2">
          {tool.point_cost} PulsePoints per generation · {userPoints} available
        </p>
      )}
    </div>
  );
}
