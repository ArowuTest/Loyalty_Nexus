'use client';

/**
 * VideoMultiScene — Multi-Image Story Video Builder
 *
 * Uses FAL.ai fal-ai/kling-video/v1.6/standard/multi-image-to-video
 * Accepts 2–4 images. Each image can have its own scene description.
 * The AI weaves them into a single video with smooth transitions.
 */

import { useState, useRef } from 'react';
import {
  Upload, X, ImageIcon, Sparkles, AlertTriangle,
  GripVertical, ChevronDown, ChevronUp, Film,
} from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

const MAX_IMAGES = 4;
const MIN_IMAGES = 2;

interface SceneSlot {
  id:       string;
  file:     File | null;
  preview:  string | null;
  url:      string;          // remote URL (if pasted)
  caption:  string;          // per-scene description
}

function emptySlot(id: string): SceneSlot {
  return { id, file: null, preview: null, url: '', caption: '' };
}

export default function VideoMultiScene({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg       = tool.ui_config ?? {};
  const maxImages = cfg.max_images ?? MAX_IMAGES;
  const durations = cfg.duration_options ?? [5, 10];

  const [scenes,      setScenes]      = useState<SceneSlot[]>([emptySlot('1'), emptySlot('2')]);
  const [duration,    setDuration]    = useState<number>(cfg.default_duration ?? 5);
  const [aspectRatio, setAspectRatio] = useState<string>(cfg.default_aspect ?? '16:9');
  const [storyPrompt, setStoryPrompt] = useState('');
  const [uploading,   setUploading]   = useState(false);
  const [expandedId,  setExpandedId]  = useState<string | null>(null);

  const fileRefs = useRef<Record<string, HTMLInputElement | null>>({});

  const canAfford   = tool.is_free || userPoints >= tool.point_cost;
  const filledCount = scenes.filter(s => s.file || s.url.trim()).length;
  const isValid     = filledCount >= MIN_IMAGES;

  // ── Scene management ────────────────────────────────────────────────────────
  function addScene() {
    if (scenes.length >= maxImages) return;
    setScenes(prev => [...prev, emptySlot(String(Date.now()))]);
  }

  function removeScene(id: string) {
    if (scenes.length <= MIN_IMAGES) return;
    setScenes(prev => prev.filter(s => s.id !== id));
  }

  function updateScene(id: string, patch: Partial<SceneSlot>) {
    setScenes(prev => prev.map(s => s.id === id ? { ...s, ...patch } : s));
  }

  function handleFile(id: string, file: File) {
    const reader = new FileReader();
    reader.onload = (e) => {
      updateScene(id, { file, preview: e.target?.result as string, url: '' });
    };
    reader.readAsDataURL(file);
  }

  function handleDrop(id: string, e: React.DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file && file.type.startsWith('image/')) handleFile(id, file);
  }

  // ── Submit ──────────────────────────────────────────────────────────────────
  async function handleSubmit() {
    if (!isValid || isLoading || !canAfford || uploading) return;

    setUploading(true);
    const uploadedUrls: string[] = [];
    const captions: string[]     = [];

    try {
      for (const scene of scenes) {
        if (!scene.file && !scene.url.trim()) continue;
        let url = scene.url.trim();
        if (scene.file) {
          const result = await api.uploadAsset(scene.file);
          url = result.url;
        }
        uploadedUrls.push(url);
        captions.push(scene.caption.trim());
      }
    } catch (err) {
      console.error('Image upload failed:', err);
      setUploading(false);
      return;
    }
    setUploading(false);

    // Build a combined prompt: story overview + per-scene captions
    const sceneParts = captions
      .map((c, i) => c ? `Scene ${i + 1}: ${c}` : `Scene ${i + 1}`)
      .join('. ');
    const fullPrompt = [storyPrompt.trim(), sceneParts].filter(Boolean).join(' — ');

    const payload: GeneratePayload = {
      prompt:       fullPrompt || 'A cinematic story video with smooth transitions',
      duration,
      aspect_ratio: aspectRatio,
      extra_params: {
        image_urls: uploadedUrls,
        captions,
      },
    };
    onSubmit(payload);
  }

  // ── Render ──────────────────────────────────────────────────────────────────
  return (
    <div className="space-y-5">

      {/* ── Warning ── */}
      {cfg.generation_warning && (
        <div className="flex items-start gap-2 bg-amber-500/8 border border-amber-500/20 rounded-xl px-3 py-2.5">
          <AlertTriangle size={13} className="text-amber-400 flex-shrink-0 mt-0.5" />
          <p className="text-amber-300/75 text-xs leading-relaxed">{cfg.generation_warning}</p>
        </div>
      )}

      {/* ── Header info ── */}
      <div className="flex items-start gap-3 bg-white/4 border border-white/8 rounded-xl px-4 py-3">
        <Film size={16} className="text-cyan-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="text-white/75 text-sm font-medium">Upload 2–{maxImages} images</p>
          <p className="text-white/35 text-xs mt-0.5 leading-relaxed">
            Each image becomes a scene. Add a caption per scene to guide the AI on how to animate it.
            The model weaves them into one continuous video.
          </p>
        </div>
      </div>

      {/* ── Scene slots ── */}
      <div className="space-y-3">
        {scenes.map((scene, idx) => {
          const hasMaterial = scene.file || scene.url.trim();
          const isExpanded  = expandedId === scene.id || !hasMaterial;

          return (
            <div
              key={scene.id}
              className={cn(
                'border rounded-xl overflow-hidden transition-all',
                hasMaterial
                  ? 'border-cyan-500/25 bg-cyan-500/4'
                  : 'border-white/10 bg-white/2',
              )}
            >
              {/* Scene header */}
              <div className="flex items-center gap-2 px-3 py-2.5">
                <GripVertical size={14} className="text-white/20 flex-shrink-0" />
                <div className={cn(
                  'w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold flex-shrink-0',
                  hasMaterial ? 'bg-cyan-600/40 text-cyan-300' : 'bg-white/8 text-white/40',
                )}>
                  {idx + 1}
                </div>
                <span className="text-white/50 text-xs font-medium flex-1">
                  {hasMaterial
                    ? (scene.file?.name ?? 'Scene ' + (idx + 1))
                    : `Scene ${idx + 1} — upload an image`}
                </span>
                <div className="flex items-center gap-1">
                  {hasMaterial && (
                    <button
                      onClick={() => setExpandedId(isExpanded ? null : scene.id)}
                      className="p-1 text-white/30 hover:text-white/60 transition-colors"
                    >
                      {isExpanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
                    </button>
                  )}
                  {scenes.length > MIN_IMAGES && (
                    <button
                      onClick={() => removeScene(scene.id)}
                      className="p-1 text-white/25 hover:text-red-400 transition-colors"
                    >
                      <X size={14} />
                    </button>
                  )}
                </div>
              </div>

              {/* Scene body */}
              {isExpanded && (
                <div className="px-3 pb-3 space-y-2.5 border-t border-white/6 pt-2.5">
                  {/* Image upload */}
                  {!hasMaterial ? (
                    <>
                      <div
                        onDrop={(e) => handleDrop(scene.id, e)}
                        onDragOver={(e) => e.preventDefault()}
                        onClick={() => fileRefs.current[scene.id]?.click()}
                        className="border-2 border-dashed border-white/12 rounded-lg p-6 flex flex-col items-center gap-2
                                   cursor-pointer hover:border-cyan-500/35 hover:bg-cyan-500/4 transition-all text-center"
                      >
                        <Upload size={18} className="text-white/35" />
                        <p className="text-white/50 text-xs">Click or drag image here</p>
                      </div>
                      <input
                        ref={(el) => { fileRefs.current[scene.id] = el; }}
                        type="file"
                        accept="image/png,image/jpeg,image/webp"
                        className="hidden"
                        onChange={(e) => {
                          const f = e.target.files?.[0];
                          if (f) handleFile(scene.id, f);
                        }}
                      />
                      <p className="text-white/25 text-[10px] text-center">— or paste a URL —</p>
                      <input
                        type="url"
                        value={scene.url}
                        onChange={(e) => updateScene(scene.id, { url: e.target.value })}
                        placeholder="https://example.com/scene.jpg"
                        className="nexus-input w-full text-xs"
                      />
                    </>
                  ) : (
                    <div className="relative rounded-lg overflow-hidden border border-white/10 bg-black/40">
                      <img
                        src={scene.preview ?? scene.url}
                        alt={`Scene ${idx + 1}`}
                        className="w-full max-h-40 object-contain"
                      />
                      <button
                        onClick={() => updateScene(scene.id, { file: null, preview: null, url: '' })}
                        className="absolute top-1.5 right-1.5 p-1 bg-black/70 rounded-full text-white/50 hover:text-white transition-colors"
                      >
                        <X size={12} />
                      </button>
                    </div>
                  )}

                  {/* Per-scene caption */}
                  <div>
                    <label className="text-white/35 text-[10px] uppercase tracking-wider font-semibold mb-1 block">
                      Scene {idx + 1} description <span className="normal-case font-normal text-white/20">(optional)</span>
                    </label>
                    <textarea
                      value={scene.caption}
                      onChange={(e) => updateScene(scene.id, { caption: e.target.value })}
                      placeholder={`Describe what happens in scene ${idx + 1}…`}
                      rows={2}
                      className="nexus-input w-full text-xs resize-none"
                    />
                  </div>
                </div>
              )}
            </div>
          );
        })}

        {/* Add scene button */}
        {scenes.length < maxImages && (
          <button
            onClick={addScene}
            className="w-full border-2 border-dashed border-white/10 rounded-xl py-3 flex items-center justify-center gap-2
                       text-white/35 text-xs hover:border-cyan-500/30 hover:text-cyan-400 transition-all"
          >
            <Upload size={13} />
            Add scene {scenes.length + 1} of {maxImages}
          </button>
        )}
      </div>

      {/* ── Story overview prompt ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          Story Overview <span className="normal-case font-normal text-white/25">(optional)</span>
        </label>
        <textarea
          value={storyPrompt}
          onChange={(e) => setStoryPrompt(e.target.value)}
          placeholder={cfg.placeholder_prompt ?? 'Describe the overall mood, style, or narrative of the video…'}
          rows={3}
          className="nexus-input w-full text-sm resize-none"
        />
      </div>

      {/* ── Duration ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          Duration
        </label>
        <div className="flex gap-2">
          {durations.map((d: number) => (
            <button
              key={d}
              onClick={() => setDuration(d)}
              className={cn(
                'px-4 py-2 rounded-lg text-sm font-medium border transition-all',
                duration === d
                  ? 'bg-cyan-600/30 border-cyan-500/50 text-cyan-300'
                  : 'bg-white/4 border-white/10 text-white/50 hover:border-white/20',
              )}
            >
              {d}s
            </button>
          ))}
        </div>
      </div>

      {/* ── Aspect ratio ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          Aspect Ratio
        </label>
        <div className="flex gap-2">
          {(cfg.aspect_ratio_options
            ? cfg.aspect_ratio_options.map((o) => (typeof o === 'string' ? o : o.value))
            : ['16:9', '9:16', '1:1']
          ).map((ar: string) => (
            <button
              key={ar}
              onClick={() => setAspectRatio(ar)}
              className={cn(
                'px-4 py-2 rounded-lg text-sm font-medium border transition-all',
                aspectRatio === ar
                  ? 'bg-cyan-600/30 border-cyan-500/50 text-cyan-300'
                  : 'bg-white/4 border-white/10 text-white/50 hover:border-white/20',
              )}
            >
              {ar}
            </button>
          ))}
        </div>
      </div>

      {/* ── Scene count indicator ── */}
      {filledCount > 0 && filledCount < MIN_IMAGES && (
        <p className="text-amber-400/70 text-xs flex items-center gap-1.5">
          <AlertTriangle size={12} />
          Add at least {MIN_IMAGES} images to generate a story video
        </p>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford || uploading}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && canAfford && !uploading
            ? 'bg-gradient-to-r from-cyan-600 to-purple-600 text-white hover:opacity-90 active:scale-[0.98]'
            : 'bg-white/6 text-white/25 cursor-not-allowed',
        )}
      >
        {uploading ? (
          <>
            <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            Uploading images…
          </>
        ) : isLoading ? (
          <>
            <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            Building story video…
          </>
        ) : !canAfford ? (
          'Not enough points'
        ) : (
          <>
            <Sparkles size={15} />
            {cfg.submit_label ?? `Build Story Video (${filledCount} scene${filledCount !== 1 ? 's' : ''})`}
          </>
        )}
      </button>

      {!canAfford && (
        <p className="text-white/30 text-xs text-center">
          This tool costs {tool.point_cost} points. You have {userPoints} points.
        </p>
      )}
    </div>
  );
}
