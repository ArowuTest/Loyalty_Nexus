'use client';

import { useState } from 'react';
import { Loader2, Sparkles, AlertTriangle, Film, Plus, X, ChevronDown, ChevronUp } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_ASPECT_RATIOS = [
  { label: 'Landscape', value: '16:9',  icon: '🖥️', w: 16, h: 9  },
  { label: 'Portrait',  value: '9:16',  icon: '📱', w: 9,  h: 16 },
  { label: 'Square',    value: '1:1',   icon: '⬜', w: 1,  h: 1  },
  { label: 'Cinematic', value: '21:9',  icon: '🎬', w: 21, h: 9  },
];

const DEFAULT_STYLE_TAGS = [
  'Cinematic', 'Documentary', 'Slow motion', 'Time-lapse',
  'Aerial drone', 'Dark', 'Vibrant', 'Vintage film', 'Sci-Fi', 'Fantasy',
];

const DEFAULT_DURATIONS = [5, 8, 10, 15, 30];

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

const PROMPT_INSPIRATIONS = [
  'A hawk soaring over Lagos skyline at dusk, golden light, cinematic',
  'Rain falling on a busy Lagos market, slow motion, dramatic lighting',
  'A couple walking on a beach at sunset, romantic, warm tones, tracking shot',
  'Futuristic city at night, neon lights, aerial drone shot, cyberpunk',
  'A lion running through the savanna, slow motion, dust particles, epic',
];

export default function VideoCreator({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg           = tool.ui_config ?? {};
  const aspectRatios  = cfg.aspect_ratios        ?? DEFAULT_ASPECT_RATIOS;
  const styleTags     = cfg.style_tags           ?? DEFAULT_STYLE_TAGS;
  const durations     = cfg.duration_options     ?? DEFAULT_DURATIONS;
  const showNeg       = cfg.show_negative_prompt ?? true;
  const maxDuration   = cfg.max_duration         ?? 15;
  const cameraPresets = cfg.camera_movements     ?? CAMERA_MOVEMENTS;

  const [prompt,     setPrompt]     = useState('');
  const [aspect,     setAspect]     = useState(cfg.default_aspect ?? '16:9');
  const [duration,   setDuration]   = useState(cfg.default_duration ?? 5);
  const [selStyles,  setSelStyles]  = useState<string[]>([]);
  const [negPrompt,  setNegPrompt]  = useState('');
  const [showNegBox, setShowNegBox] = useState(false);
  const [cameraMove, setCameraMove] = useState<string | null>(null);
  const [showInspo,  setShowInspo]  = useState(false);
  // Multi-scene builder
  const [scenes,     setScenes]     = useState<string[]>([]);
  const [newScene,   setNewScene]   = useState('');
  const [showScenes, setShowScenes] = useState(false);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = prompt.trim().length >= 3;

  const filteredDurations = durations.filter((d: number) => d <= maxDuration);

  // Find current AR for live preview
  const currentAR = (aspectRatios as { value: string; w?: number; h?: number }[])
    .find((ar) => ar.value === aspect) ?? { w: 16, h: 9 };
  const previewW = 64;
  const previewH = currentAR.w && currentAR.h
    ? Math.round((currentAR.h / currentAR.w) * previewW)
    : 36;

  function toggleStyle(s: string) {
    setSelStyles((prev) => prev.includes(s) ? prev.filter((t) => t !== s) : [...prev, s]);
  }

  function addScene() {
    if (newScene.trim().length >= 3) {
      setScenes((prev) => [...prev, newScene.trim()]);
      setNewScene('');
    }
  }

  function removeScene(i: number) {
    setScenes((prev) => prev.filter((_, idx) => idx !== i));
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const stylePrefix  = selStyles.length > 0 ? `[${selStyles.join(', ')}] ` : '';
    const cameraSuffix = cameraMove ? `. Camera movement: ${cameraMove}.` : '';
    const scenesSuffix = scenes.length > 0
      ? '\n\nScene breakdown:\n' + scenes.map((s, i) => `Scene ${i + 1}: ${s}`).join('\n')
      : '';
    const payload: GeneratePayload = {
      prompt:          stylePrefix + prompt.trim() + cameraSuffix + scenesSuffix,
      aspect_ratio:    aspect,
      duration,
      style_tags:      selStyles.length > 0 ? selStyles : undefined,
      negative_prompt: negPrompt.trim() || undefined,
      extra_params: {
        camera_movement: cameraMove ?? undefined,
        scenes:          scenes.length > 0 ? scenes : undefined,
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

      {/* ── Aspect ratio with live canvas preview ── */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Aspect Ratio</label>
          <div className="flex items-center gap-2">
            <div
              className="border-2 border-blue-500/50 bg-blue-600/10 rounded transition-all duration-300"
              style={{ width: `${previewW}px`, height: `${previewH}px`, minHeight: '16px' }}
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
                'flex flex-col items-center gap-1 py-2.5 rounded-xl border text-xs font-medium transition-all',
                aspect === ar.value
                  ? 'bg-blue-600/25 border-blue-500/60 text-blue-200 shadow-sm shadow-blue-900/30'
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

      {/* ── Duration ── */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Duration</label>
          <span className="text-white/35 text-[11px] font-mono">{duration}s</span>
        </div>
        <div className="flex gap-2 flex-wrap">
          {(filteredDurations.length > 0 ? filteredDurations : durations).map((d: number) => (
            <button
              key={d}
              onClick={() => setDuration(d)}
              className={cn(
                'text-xs px-4 py-2 rounded-lg border font-semibold transition-all',
                duration === d
                  ? 'bg-blue-600 text-white border-blue-500 shadow-sm shadow-blue-900/30'
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
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Style</label>
          {selStyles.length > 0 && (
            <button onClick={() => setSelStyles([])} className="text-white/25 text-[10px] hover:text-white/50 transition-colors">
              Clear all
            </button>
          )}
        </div>
        <div className="flex flex-wrap gap-1.5">
          {(styleTags as string[]).map((s) => (
            <button
              key={s}
              onClick={() => toggleStyle(s)}
              className={cn(
                'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                selStyles.includes(s)
                  ? 'bg-blue-600 text-white border-blue-500 shadow-sm shadow-blue-900/30'
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
          {(cameraPresets as { label: string; icon?: string; value: string }[]).map((cm) => (
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
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            Scene description
          </label>
          <button
            onClick={() => setShowInspo((v) => !v)}
            className="flex items-center gap-1 text-white/25 hover:text-white/50 transition-colors text-[11px]"
          >
            <Film size={11} />
            {showInspo ? 'Hide' : 'Show'} ideas
            {showInspo ? <ChevronUp size={11} /> : <ChevronDown size={11} />}
          </button>
        </div>
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

        {showInspo && (
          <div className="mt-2 grid grid-cols-1 gap-1.5">
            {PROMPT_INSPIRATIONS.map((inspo) => (
              <button
                key={inspo}
                onClick={() => { setPrompt(inspo); setShowInspo(false); }}
                className="text-left text-xs text-white/40 hover:text-white/70 hover:bg-white/[0.04] px-3 py-2 rounded-lg border border-white/[0.06] hover:border-white/15 transition-all truncate"
              >
                {inspo}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* ── Multi-scene builder (collapsible) ── */}
      <div>
        <button
          onClick={() => setShowScenes((v) => !v)}
          className="flex items-center gap-2 text-white/35 text-xs font-medium hover:text-white/60 transition-colors"
        >
          <Film size={13} />
          Multi-scene storyboard
          {scenes.length > 0 && (
            <span className="bg-blue-600/30 text-blue-300 text-[10px] font-bold px-1.5 py-0.5 rounded-full">
              {scenes.length}
            </span>
          )}
          {showScenes ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
        </button>

        {showScenes && (
          <div className="mt-2 space-y-2">
            <p className="text-white/25 text-[11px]">Add individual scenes to build a storyboard — each scene will be described in the prompt.</p>
            {scenes.map((scene, i) => (
              <div key={i} className="flex items-start gap-2 bg-blue-600/8 border border-blue-500/20 rounded-xl px-3 py-2.5">
                <span className="text-blue-400/60 text-[11px] font-bold flex-shrink-0 mt-0.5">S{i + 1}</span>
                <p className="text-white/65 text-xs flex-1">{scene}</p>
                <button onClick={() => removeScene(i)} className="text-white/25 hover:text-white/60 transition-colors flex-shrink-0">
                  <X size={13} />
                </button>
              </div>
            ))}
            <div className="flex gap-2">
              <input
                type="text"
                value={newScene}
                onChange={(e) => setNewScene(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addScene(); } }}
                placeholder={`Scene ${scenes.length + 1} description…`}
                className="nexus-input flex-1 text-xs"
              />
              <button
                onClick={addScene}
                disabled={newScene.trim().length < 3}
                className="flex items-center gap-1 px-3 py-2 rounded-xl bg-blue-600/20 border border-blue-500/30 text-blue-300 text-xs font-medium hover:bg-blue-600/30 transition-all disabled:opacity-30 disabled:cursor-not-allowed"
              >
                <Plus size={13} /> Add
              </button>
            </div>
          </div>
        )}
      </div>

      {/* ── Negative prompt ── */}
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
          'w-full py-4 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-blue-600 to-cyan-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-blue-900/40'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading ? (
          <><Loader2 size={15} className="animate-spin" /> Generating video…</>
        ) : (
          <><Sparkles size={15} /> Generate Video</>
        )}
      </button>

      {!tool.is_free && (
        <p className="text-white/20 text-[11px] text-center -mt-2">
          {tool.point_cost} PulsePoints per generation · {userPoints} available
        </p>
      )}
    </div>
  );
}
