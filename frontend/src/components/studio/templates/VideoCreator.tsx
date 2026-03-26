'use client';

import { useState } from 'react';
import { Loader2, Sparkles, AlertTriangle } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_ASPECT_RATIOS = [
  { label: 'Landscape', value: '16:9',  icon: '🖥️' },
  { label: 'Portrait',  value: '9:16',  icon: '📱' },
  { label: 'Square',    value: '1:1',   icon: '⬜' },
  { label: 'Cinematic', value: '21:9',  icon: '🎬' },
];

const DEFAULT_STYLE_TAGS = [
  'Cinematic', 'Documentary', 'Slow motion', 'Time-lapse',
  'Aerial drone', 'Dark', 'Vibrant', 'Vintage film', 'Sci-Fi', 'Fantasy',
];

const DEFAULT_DURATIONS = [5, 8, 10, 15, 30];

// Camera movement presets — appended to prompt for best results
const CAMERA_MOVEMENTS = [
  { label: 'Slow zoom in',  icon: '🔍', value: 'slow zoom in' },
  { label: 'Slow zoom out', icon: '🔭', value: 'slow zoom out' },
  { label: 'Pan left',      icon: '⬅️', value: 'camera panning left' },
  { label: 'Pan right',     icon: '➡️', value: 'camera panning right' },
  { label: 'Tilt up',       icon: '⬆️', value: 'camera tilting up' },
  { label: 'Orbit shot',    icon: '🔄', value: '360 orbit around subject' },
  { label: 'Tracking',      icon: '🎯', value: 'tracking shot following subject' },
  { label: 'Handheld',      icon: '📷', value: 'handheld camera, slight shake' },
  { label: 'Static',        icon: '📌', value: 'static camera, no movement' },
];

export default function VideoCreator({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg          = tool.ui_config ?? {};
  const aspectRatios = cfg.aspect_ratios           ?? DEFAULT_ASPECT_RATIOS;
  const styleTags    = cfg.style_tags              ?? DEFAULT_STYLE_TAGS;
  const durations    = cfg.duration_options        ?? DEFAULT_DURATIONS;
  const showNeg      = cfg.show_negative_prompt    ?? true;
  // Allow max duration override per tool (video-veo supports 30s)
  const maxDuration  = cfg.max_duration            ?? 15;
  const cameraPresets = cfg.camera_movements       ?? CAMERA_MOVEMENTS;

  const [prompt,       setPrompt]      = useState('');
  const [aspect,       setAspect]      = useState(cfg.default_aspect ?? '16:9');
  const [duration,     setDuration]    = useState(cfg.default_duration ?? 5);
  const [selStyles,    setSelStyles]   = useState<string[]>([]);
  const [negPrompt,    setNegPrompt]   = useState('');
  const [showNegBox,   setShowNegBox]  = useState(false);
  const [cameraMove,   setCameraMove]  = useState<string | null>(null);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = prompt.trim().length >= 3;

  const filteredDurations = durations.filter((d: number) => d <= maxDuration);

  function toggleStyle(s: string) {
    setSelStyles((prev) => prev.includes(s) ? prev.filter((t) => t !== s) : [...prev, s]);
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const stylePrefix  = selStyles.length > 0 ? `[${selStyles.join(', ')}] ` : '';
    const cameraSuffix = cameraMove ? `. Camera movement: ${cameraMove}.` : '';
    const payload: GeneratePayload = {
      prompt:          stylePrefix + prompt.trim() + cameraSuffix,
      aspect_ratio:    aspect,
      duration:        duration,
      style_tags:      selStyles.length > 0 ? selStyles : undefined,
      negative_prompt: negPrompt.trim() || undefined,
      extra_params: {
        camera_movement: cameraMove ?? undefined,
      },
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Generation warning ── */}
      {cfg.generation_warning && (
        <div className="flex items-start gap-2 bg-amber-500/8 border border-amber-500/20 rounded-xl px-3 py-2.5">
          <AlertTriangle size={13} className="text-amber-400 flex-shrink-0 mt-0.5" />
          <p className="text-amber-300/75 text-xs leading-relaxed">{cfg.generation_warning}</p>
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
                  ? 'bg-blue-600/25 border-blue-500/60 text-blue-200'
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

      {/* ── Duration ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Duration</label>
        <div className="flex gap-2 flex-wrap">
          {(filteredDurations.length > 0 ? filteredDurations : durations).map((d: number) => (
            <button
              key={d}
              onClick={() => setDuration(d)}
              className={cn(
                'text-xs px-4 py-2 rounded-lg border font-semibold transition-all',
                duration === d
                  ? 'bg-blue-600 text-white border-blue-500'
                  : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
              )}
            >
              {d}s
            </button>
          ))}
        </div>
      </div>

      {/* ── Style tags ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Style</label>
        <div className="flex flex-wrap gap-1.5">
          {styleTags.map((s: string) => (
            <button
              key={s}
              onClick={() => toggleStyle(s)}
              className={cn(
                'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                selStyles.includes(s)
                  ? 'bg-blue-600 text-white border-blue-500'
                  : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
              )}
            >
              {s}
            </button>
          ))}
        </div>
      </div>

      {/* ── Camera movement presets ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          Camera Movement <span className="text-white/25 normal-case font-normal">(optional)</span>
        </label>
        <div className="grid grid-cols-3 gap-1.5">
          {cameraPresets.map((cm: { label: string; icon?: string; value: string }) => (
            <button
              key={cm.value}
              onClick={() => setCameraMove(cameraMove === cm.value ? null : cm.value)}
              className={cn(
                'flex items-center gap-1.5 px-2.5 py-2 rounded-xl border text-xs font-medium transition-all text-left',
                cameraMove === cm.value
                  ? 'bg-blue-600/25 border-blue-500/60 text-blue-200'
                  : 'border-white/10 text-white/45 hover:border-white/25 hover:text-white/70',
              )}
            >
              {cm.icon && <span className="text-sm leading-none">{cm.icon}</span>}
              <span className="text-[11px] truncate">{cm.label}</span>
            </button>
          ))}
        </div>
      </div>

      {/* ── Scene prompt ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
          Scene description
        </label>
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder={
            cfg.prompt_placeholder ??
            'Describe the scene — subject, setting, lighting, atmosphere…\ne.g. A hawk soaring over Lagos skyline at dusk, golden light, cinematic'
          }
          rows={4}
          autoFocus
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />
        <p className="text-white/25 text-[11px] mt-1">{prompt.length}/500 characters</p>
      </div>

      {/* ── Negative prompt ── */}
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
              placeholder={cfg.negative_prompt_placeholder ?? 'Things to avoid: shaky camera, blurry, text overlays, watermark…'}
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
            ? 'bg-gradient-to-r from-blue-600 to-cyan-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-blue-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Generating…</>
          : <><Sparkles size={15} /> Generate Video →</>
        }
      </button>
    </div>
  );
}
