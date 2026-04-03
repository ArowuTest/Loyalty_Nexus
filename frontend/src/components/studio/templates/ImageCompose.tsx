'use client';

/**
 * ImageCompose.tsx — Whisk-style multi-reference image composition
 *
 * Allows users to upload up to three reference images (subject, scene, style)
 * and write a composition prompt. The backend routes this to Grok Aurora
 * (grok-imagine-image) as Tier 1, which accepts up to 5 reference images natively
 * via the image_urls array. FAL Flux Pro 1.1 Ultra is used as a fallback.
 *
 * This mirrors Google Whisk's core UX: pick a character, pick a scene, pick a
 * style, and let the AI compose them into a new image.
 */

import { useState, useRef } from 'react';
import { Loader2, Upload, X, Sparkles, Mic, MicOff, User, Mountain, Palette, ChevronDown, ChevronUp, CheckCircle2, Wand2, Info } from 'lucide-react';
import { useSpeechToText } from '@/hooks/useSpeechToText';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

// ── Types ──────────────────────────────────────────────────────────────────

type SlotKey = 'subject' | 'scene' | 'style';

interface Slot {
  key:       SlotKey;
  label:     string;
  sublabel:  string;
  icon:      React.ReactNode;
  required:  boolean;
  color:     string;
  hint:      string;
}

interface SlotState {
  preview:    string | null;
  uploadedUrl: string | null;
  isUploading: boolean;
  error:       string | null;
}

// ── Constants ──────────────────────────────────────────────────────────────

const SLOTS: Slot[] = [
  {
    key:      'subject',
    label:    'Subject',
    sublabel: 'Character / Person',
    icon:     <User size={16} />,
    required: true,
    color:    'border-purple-500/40 bg-purple-600/8',
    hint:     'Upload the character, person, or main object to feature in the image.',
  },
  {
    key:      'scene',
    label:    'Scene',
    sublabel: 'Background / Setting',
    icon:     <Mountain size={16} />,
    required: false,
    color:    'border-blue-500/40 bg-blue-600/8',
    hint:     'Upload a background or environment to place the subject in.',
  },
  {
    key:      'style',
    label:    'Style',
    sublabel: 'Aesthetic Reference',
    icon:     <Palette size={16} />,
    required: false,
    color:    'border-pink-500/40 bg-pink-600/8',
    hint:     'Upload an image whose visual style, colour palette, or mood you want to match.',
  },
];

const COMPOSITION_PROMPTS = [
  'Place the subject in the scene with natural lighting and cinematic depth',
  'Compose the subject in the scene, matching the style of the reference image',
  'The subject stands in the scene, rendered in the aesthetic of the style image',
  'Portrait of the subject in the scene environment, photorealistic, 8K detail',
  'The character from the subject image in the background setting, anime style',
  'A dramatic composition of the subject in the scene, oil painting aesthetic',
];

const NUM_IMAGE_OPTIONS = [1, 2, 4] as const;
type NumImages = typeof NUM_IMAGE_OPTIONS[number];

// ── Component ──────────────────────────────────────────────────────────────

export default function ImageCompose({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const [slots, setSlots] = useState<Record<SlotKey, SlotState>>({
    subject: { preview: null, uploadedUrl: null, isUploading: false, error: null },
    scene:   { preview: null, uploadedUrl: null, isUploading: false, error: null },
    style:   { preview: null, uploadedUrl: null, isUploading: false, error: null },
  });
  const [prompt,      setPrompt]      = useState('');
  const [numImages,   setNumImages]   = useState<NumImages>(1);
  const [showInspo,   setShowInspo]   = useState(false);
  const [showHints,   setShowHints]   = useState(false);
  const [refStrength, setRefStrength] = useState(0.35); // how strongly the subject image guides output

  const fileRefs = {
    subject: useRef<HTMLInputElement>(null),
    scene:   useRef<HTMLInputElement>(null),
    style:   useRef<HTMLInputElement>(null),
  };

  const { speechState, speechError, interimText, handleMicClick } =
    useSpeechToText({
      onTranscript: (t) => setPrompt(prev => (prev ? prev + ' ' + t : t).slice(0, 500)),
      language: 'en-US',
    });

  const subjectReady = !!slots.subject.uploadedUrl;
  const anyUploading = Object.values(slots).some((s) => s.isUploading);
  const canAfford    = tool.is_free || userPoints >= tool.point_cost * numImages;
  const isValid      = subjectReady && prompt.trim().length >= 3 && !anyUploading;
  const totalCost    = tool.point_cost * numImages;

  // ── Upload handler ─────────────────────────────────────────────────────

  async function handleFile(key: SlotKey, file: File) {
    // Show preview immediately
    const reader = new FileReader();
    reader.onload = (e) =>
      setSlots((prev) => ({ ...prev, [key]: { ...prev[key], preview: e.target?.result as string, error: null } }));
    reader.readAsDataURL(file);

    setSlots((prev) => ({ ...prev, [key]: { ...prev[key], isUploading: true, uploadedUrl: null, error: null } }));
    try {
      const result = await api.uploadAsset(file);
      setSlots((prev) => ({ ...prev, [key]: { ...prev[key], isUploading: false, uploadedUrl: result.url } }));
    } catch {
      setSlots((prev) => ({
        ...prev,
        [key]: { ...prev[key], isUploading: false, error: 'Upload failed — please try again.' },
      }));
    }
  }

  function clearSlot(key: SlotKey) {
    setSlots((prev) => ({
      ...prev,
      [key]: { preview: null, uploadedUrl: null, isUploading: false, error: null },
    }));
  }

  // ── Submit ─────────────────────────────────────────────────────────────

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;

    // Build a composite prompt that describes all the references
    const parts: string[] = [];
    if (slots.scene.uploadedUrl)  parts.push('scene reference provided');
    if (slots.style.uploadedUrl)  parts.push('style reference provided');
    const compositePrompt = prompt.trim() + (parts.length ? ` [${parts.join(', ')}]` : '');

    const payload: GeneratePayload = {
      prompt:    compositePrompt,
      image_url: slots.subject.uploadedUrl!,
      extra_params: {
        num_images:            numImages > 1 ? numImages : undefined,
        image_prompt_strength: refStrength,
        // Pass scene and style image URLs as additional context
        scene_image_url:       slots.scene.uploadedUrl  ?? undefined,
        style_image_url:       slots.style.uploadedUrl  ?? undefined,
        compose_mode:          true,
      },
    };
    onSubmit(payload);
  }

  // ── Render ─────────────────────────────────────────────────────────────

  return (
    <div className="space-y-5">

      {/* ── How it works banner ── */}
      <div className="flex items-start gap-2.5 bg-purple-600/8 border border-purple-500/20 rounded-xl px-3 py-2.5">
        <Info size={13} className="text-purple-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="text-purple-200/80 text-xs font-medium">Whisk-style composition</p>
          <p className="text-purple-200/45 text-[11px] mt-0.5 leading-relaxed">
            Upload a <span className="text-purple-300">subject</span> (required), an optional <span className="text-blue-300">scene</span>, and an optional <span className="text-pink-300">style</span> reference. All three images are sent to <span className="text-purple-300 font-medium">Grok Aurora</span> for true multi-image composition.
          </p>
        </div>
      </div>

      {/* ── Reference image slots ── */}
      <div className="grid grid-cols-3 gap-2.5">
        {SLOTS.map((slot) => {
          const state = slots[slot.key];
          return (
            <div key={slot.key} className="space-y-1.5">
              {/* Slot header */}
              <div className="flex items-center gap-1.5">
                <span className={cn(
                  'p-1 rounded-lg',
                  slot.key === 'subject' ? 'bg-purple-600/20 text-purple-300' :
                  slot.key === 'scene'   ? 'bg-blue-600/20 text-blue-300' :
                                           'bg-pink-600/20 text-pink-300',
                )}>
                  {slot.icon}
                </span>
                <div>
                  <p className="text-white/65 text-[11px] font-semibold leading-tight">{slot.label}</p>
                  <p className="text-white/25 text-[9px] leading-tight">{slot.sublabel}</p>
                </div>
                {slot.required && (
                  <span className="ml-auto text-[9px] text-red-400/60 font-semibold">REQ</span>
                )}
              </div>

              {/* Upload zone / preview */}
              {!state.preview ? (
                <div
                  onClick={() => fileRefs[slot.key].current?.click()}
                  className={cn(
                    'aspect-square rounded-xl border-2 border-dashed flex flex-col items-center justify-center gap-1.5 cursor-pointer transition-all',
                    slot.color,
                    'hover:opacity-80',
                  )}
                >
                  <Upload size={14} className="text-white/30" />
                  <p className="text-white/30 text-[10px] text-center px-1">Upload</p>
                </div>
              ) : (
                <div className="relative aspect-square rounded-xl overflow-hidden border border-white/10">
                  <img
                    src={state.preview}
                    alt={slot.label}
                    className="w-full h-full object-cover"
                  />
                  {state.isUploading && (
                    <div className="absolute inset-0 bg-black/60 flex items-center justify-center">
                      <Loader2 size={16} className="text-white animate-spin" />
                    </div>
                  )}
                  {state.uploadedUrl && !state.isUploading && (
                    <div className="absolute bottom-1 right-1">
                      <CheckCircle2 size={12} className="text-green-400 drop-shadow" />
                    </div>
                  )}
                  <button
                    onClick={() => clearSlot(slot.key)}
                    className="absolute top-1 right-1 p-1 bg-black/70 rounded-full text-white/60 hover:text-white transition-colors"
                  >
                    <X size={10} />
                  </button>
                </div>
              )}

              <input
                ref={fileRefs[slot.key]}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                className="hidden"
                onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(slot.key, f); }}
              />

              {state.error && (
                <p className="text-red-300/70 text-[10px]">{state.error}</p>
              )}
            </div>
          );
        })}
      </div>

      {/* ── Slot hints (collapsible) ── */}
      <button
        onClick={() => setShowHints((v) => !v)}
        className="flex items-center gap-1.5 text-white/25 hover:text-white/45 text-[11px] transition-colors"
      >
        {showHints ? <ChevronUp size={11} /> : <ChevronDown size={11} />}
        {showHints ? 'Hide' : 'Show'} slot descriptions
      </button>
      {showHints && (
        <div className="space-y-1.5 p-3 rounded-xl bg-white/[0.02] border border-white/[0.05]">
          {SLOTS.map((slot) => (
            <div key={slot.key} className="flex items-start gap-2">
              <span className={cn(
                'text-[10px] font-bold mt-0.5 flex-shrink-0 w-12',
                slot.key === 'subject' ? 'text-purple-300' :
                slot.key === 'scene'   ? 'text-blue-300' :
                                         'text-pink-300',
              )}>
                {slot.label}
              </span>
              <p className="text-white/35 text-[11px] leading-relaxed">{slot.hint}</p>
            </div>
          ))}
        </div>
      )}

      {/* ── Number of images ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          Number of Images
        </label>
        <div className="flex gap-2">
          {NUM_IMAGE_OPTIONS.map((n) => (
            <button
              key={n}
              onClick={() => setNumImages(n)}
              className={cn(
                'flex-1 py-2.5 rounded-xl border text-sm font-semibold transition-all',
                numImages === n
                  ? 'bg-purple-600/25 border-purple-500/60 text-purple-200 shadow-sm shadow-purple-900/30'
                  : 'border-white/10 text-white/45 hover:border-white/25 hover:text-white/70',
              )}
            >
              {n === 1 ? '× 1' : n === 2 ? '× 2' : '× 4'}
            </button>
          ))}
        </div>
        {numImages > 1 && (
          <p className="text-white/25 text-[11px] mt-1.5">
            {numImages} composition variations · costs {totalCost} PulsePoints
          </p>
        )}
      </div>

      {/* ── Subject influence strength ── */}
      {subjectReady && (
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="text-white/40 text-[11px] font-semibold uppercase tracking-wider">Reference Influence</label>
            <span className="text-white/40 text-[11px] font-mono">{Math.round(refStrength * 100)}%</span>
          </div>
          <input
            type="range"
            min={0.1}
            max={0.9}
            step={0.05}
            value={refStrength}
            onChange={(e) => setRefStrength(parseFloat(e.target.value))}
            className="w-full accent-purple-500"
          />
          <div className="flex justify-between text-[10px] text-white/20 mt-0.5">
            <span>Loose (more creative)</span>
            <span>Strict (more faithful)</span>
          </div>
        </div>
      )}

      {/* ── Composition prompt ── */}
      <div>
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            Composition prompt
          </label>
          <button
            onClick={handleMicClick}
            disabled={speechState === 'processing'}
            title={speechState === 'listening' ? 'Stop listening' : 'Speak your prompt'}
            className={cn(
              'w-7 h-7 rounded-lg flex items-center justify-center transition-all border flex-shrink-0',
              speechState === 'listening'
                ? 'bg-red-500/20 text-red-400 border-red-500/40 animate-pulse'
                : 'bg-white/5 text-white/30 hover:text-white/60 hover:bg-white/10 border-transparent',
            )}
          >
            {speechState === 'listening' ? <MicOff size={12} /> : <Mic size={12} />}
          </button>
        </div>

        {/* Prompt inspirations */}
        <button
          onClick={() => setShowInspo((v) => !v)}
          className="flex items-center gap-1 text-white/25 hover:text-white/45 text-[11px] transition-colors mb-2"
        >
          <Wand2 size={11} />
          {showInspo ? 'Hide' : 'Show'} prompt ideas
          {showInspo ? <ChevronUp size={11} /> : <ChevronDown size={11} />}
        </button>
        {showInspo && (
          <div className="grid grid-cols-1 gap-1.5 mb-2">
            {COMPOSITION_PROMPTS.map((p) => (
              <button
                key={p}
                onClick={() => { setPrompt(p); setShowInspo(false); }}
                className="text-left text-xs text-white/40 hover:text-white/70 hover:bg-white/[0.04] px-3 py-2 rounded-lg border border-white/[0.06] hover:border-white/15 transition-all"
              >
                {p}
              </button>
            ))}
          </div>
        )}

        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value.slice(0, 500))}
          placeholder="Describe how to compose the images — e.g. 'Place the subject in the scene with cinematic lighting, matching the style reference'"
          rows={3}
          autoFocus
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />

        {speechState === 'listening' && (
          <p className="text-[11px] text-red-300 mt-1 flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse inline-block" />
            Listening… {interimText || 'describe the composition'}
          </p>
        )}
        {speechState === 'error' && speechError && (
          <p className="text-[11px] text-red-300 mt-1">{speechError}</p>
        )}
      </div>

      {/* ── Validation hint ── */}
      {!subjectReady && (
        <p className="text-amber-400/60 text-[11px] flex items-center gap-1.5">
          <span className="w-1.5 h-1.5 rounded-full bg-amber-400/60 flex-shrink-0" />
          Upload a subject image to continue
        </p>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-4 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-purple-600 via-pink-600 to-blue-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-purple-900/40'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading ? (
          <>
            <Loader2 size={15} className="animate-spin" />
            <span>Composing {numImages > 1 ? `${numImages} images` : 'image'}…</span>
          </>
        ) : anyUploading ? (
          <>
            <Loader2 size={15} className="animate-spin" />
            <span>Uploading references…</span>
          </>
        ) : (
          <>
            <Sparkles size={15} />
            <span>Compose {numImages > 1 ? `${numImages} Images` : 'Image'}</span>
            {!canAfford && <span className="text-xs opacity-60 ml-1">(insufficient points)</span>}
          </>
        )}
      </button>

      {!tool.is_free && (
        <p className="text-white/20 text-[11px] text-center -mt-2">
          {totalCost} PulsePoints per generation · {userPoints} available
        </p>
      )}
    </div>
  );
}
