'use client';

/**
 * VideoExtender — Grok Imagine Video Extension
 *
 * Upload an existing video (2–15 seconds) and Grok continues it seamlessly
 * from the last frame, adding 2–10 more seconds of AI-generated content.
 *
 * Use cases:
 *  - Extend a product reveal video
 *  - Continue a cinematic scene
 *  - Add a longer outro to a brand video
 *  - Make a short clip longer for social media
 */

import { useState, useRef } from 'react';
import {
  Loader2, Upload, X, Sparkles, AlertTriangle,
  Mic, MicOff, Zap, Film, ArrowRight,
} from 'lucide-react';
import { useSpeechToText } from '@/hooks/useSpeechToText';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

const EXTENSION_DURATIONS = [2, 4, 6, 8, 10];

const ASPECT_RATIOS = [
  { value: '16:9', label: 'Landscape', icon: '🖥️' },
  { value: '9:16', label: 'Portrait',  icon: '📱' },
  { value: '1:1',  label: 'Square',    icon: '⬜' },
];

const EXTENSION_EXAMPLES = [
  'Continue the scene naturally with the same lighting and mood',
  'Zoom out slowly to reveal the wider environment',
  'The character walks forward and exits the frame',
  'The camera pans right to reveal a cityscape',
  'Fade to a wide aerial shot of the location',
  'The product rotates 360 degrees and comes to rest',
];

const CONSTRAINTS = [
  { icon: '⏱️', text: 'Input: 2–15 seconds' },
  { icon: '➕', text: 'Extension: 2–10 seconds' },
  { icon: '🔗', text: 'Seamless continuation from last frame' },
];

export default function VideoExtender({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg = tool.ui_config ?? {};

  // ── Video upload state ─────────────────────────────────────────────────────
  const [videoFile,    setVideoFile]    = useState<File | null>(null);
  const [videoUrl,     setVideoUrl]     = useState('');
  const [videoPreview, setVideoPreview] = useState<string | null>(null);
  const [uploading,    setUploading]    = useState(false);
  const [uploadedUrl,  setUploadedUrl]  = useState('');

  // ── Extension controls ─────────────────────────────────────────────────────
  const [extendPrompt, setExtendPrompt] = useState('');
  const [duration,     setDuration]     = useState(6);
  const [aspectRatio,  setAspectRatio]  = useState('16:9');

  // ── Voice input ────────────────────────────────────────────────────────────
  const { speechState, speechError, interimText, handleMicClick } =
    useSpeechToText({
      onTranscript: (t) => setExtendPrompt(prev => prev ? prev + ' ' + t : t),
      language: 'en-US',
    });

  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const hasVideo  = uploadedUrl || videoUrl.trim();
  const isValid   = !!hasVideo && !uploading;

  // ── File handler ───────────────────────────────────────────────────────────
  async function handleVideoFile(file: File) {
    setVideoFile(file);
    const localUrl = URL.createObjectURL(file);
    setVideoPreview(localUrl);
    setVideoUrl('');
    setUploadedUrl('');

    setUploading(true);
    try {
      const result = await api.uploadAsset(file);
      setUploadedUrl(result.url);
    } catch (err) {
      console.error('Video upload failed:', err);
    } finally {
      setUploading(false);
    }
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file && file.type.startsWith('video/')) handleVideoFile(file);
  }

  function clearVideo() {
    setVideoFile(null);
    setVideoPreview(null);
    setVideoUrl('');
    setUploadedUrl('');
  }

  // ── Submit ─────────────────────────────────────────────────────────────────
  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;

    const finalVideoUrl = uploadedUrl || videoUrl.trim();
    const prompt = extendPrompt.trim() || 'Continue the video naturally, maintaining the same style, lighting, and motion';

    const payload: GeneratePayload = {
      prompt,
      image_url:    finalVideoUrl,
      duration,
      aspect_ratio: aspectRatio,
      extra_params: {
        video_url: finalVideoUrl,
        extend:    true,
      },
    };
    onSubmit(payload);
  }

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <div className="space-y-5">

      {/* ── Grok badge ── */}
      <div className="flex items-center gap-2 bg-gradient-to-r from-emerald-500/10 to-cyan-500/10 border border-emerald-500/20 rounded-xl px-3 py-2.5">
        <Zap size={13} className="text-emerald-400 flex-shrink-0" />
        <p className="text-emerald-300/80 text-xs leading-relaxed">
          <span className="font-semibold text-emerald-300">Grok Imagine Video Extension</span>
          {' '}— Upload a video and Grok seamlessly continues it from the last frame. No cuts. No jump. Just more.
        </p>
      </div>

      {/* ── Constraints info ── */}
      <div className="grid grid-cols-3 gap-2">
        {CONSTRAINTS.map((c) => (
          <div key={c.text} className="bg-white/[0.03] border border-white/8 rounded-lg px-2.5 py-2 text-center">
            <div className="text-base mb-1">{c.icon}</div>
            <p className="text-white/40 text-[10px] leading-tight">{c.text}</p>
          </div>
        ))}
      </div>

      {/* ── Generation warning ── */}
      {cfg.generation_warning && (
        <div className="flex items-start gap-2 bg-amber-500/8 border border-amber-500/20 rounded-xl px-3 py-2.5">
          <AlertTriangle size={13} className="text-amber-400 flex-shrink-0 mt-0.5" />
          <p className="text-amber-300/75 text-xs leading-relaxed">{cfg.generation_warning}</p>
        </div>
      )}

      {/* ── Step 1: Video upload ── */}
      <div>
        <div className="flex items-center gap-2 mb-2">
          <span className="w-5 h-5 rounded-full bg-emerald-600/30 text-emerald-300 text-[10px] font-bold flex items-center justify-center flex-shrink-0">1</span>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            Upload your video to extend
          </label>
        </div>

        {!videoPreview && !videoUrl ? (
          <>
            <div
              onDrop={handleDrop}
              onDragOver={(e) => e.preventDefault()}
              onClick={() => fileRef.current?.click()}
              className="border-2 border-dashed border-white/15 rounded-xl p-8 flex flex-col items-center gap-3 cursor-pointer
                         hover:border-emerald-500/40 hover:bg-emerald-500/5 transition-all text-center"
            >
              <div className="p-3 rounded-full bg-white/5">
                <Film size={22} className="text-white/40" />
              </div>
              <div>
                <p className="text-white/65 text-sm font-medium">Upload the video to extend</p>
                <p className="text-white/30 text-xs mt-1">MP4, MOV, WebM · 2–15 seconds · up to {cfg.max_file_mb ?? 100} MB</p>
              </div>
            </div>
            <input
              ref={fileRef}
              type="file"
              accept="video/mp4,video/quicktime,video/webm,video/*"
              className="hidden"
              onChange={(e) => { const f = e.target.files?.[0]; if (f) handleVideoFile(f); }}
            />
            <p className="text-white/30 text-[11px] text-center mt-2">— or paste a video URL —</p>
            <input
              type="url"
              value={videoUrl}
              onChange={(e) => setVideoUrl(e.target.value)}
              placeholder="https://example.com/video.mp4"
              className="nexus-input w-full text-sm mt-1"
            />
          </>
        ) : (
          <div className="relative rounded-xl overflow-hidden border border-white/10 bg-black/50">
            {videoPreview ? (
              <video
                src={videoPreview}
                controls
                className="w-full max-h-56 object-contain"
              />
            ) : (
              <div className="flex items-center gap-3 px-4 py-3">
                <Film size={16} className="text-emerald-400 flex-shrink-0" />
                <span className="text-white/60 text-sm truncate">{videoUrl}</span>
              </div>
            )}
            <button
              onClick={clearVideo}
              className="absolute top-2 right-2 p-1.5 bg-black/70 rounded-full text-white/60 hover:text-white transition-colors"
            >
              <X size={14} />
            </button>
            <div className="absolute bottom-2 left-2 flex items-center gap-1.5 bg-black/70 rounded-full px-2.5 py-1">
              {uploading ? (
                <><Loader2 size={10} className="text-emerald-400 animate-spin" /><span className="text-white/50 text-[11px]">Uploading…</span></>
              ) : uploadedUrl ? (
                <><span className="w-1.5 h-1.5 rounded-full bg-green-500 inline-block" /><span className="text-white/50 text-[11px]">{videoFile?.name ?? 'Video ready'}</span></>
              ) : (
                <><Film size={10} className="text-emerald-400" /><span className="text-white/50 text-[11px]">{videoFile?.name ?? 'Video URL'}</span></>
              )}
            </div>
          </div>
        )}
      </div>

      {/* ── Step 2: Extension controls (shown once video is loaded) ── */}
      {hasVideo && (
        <>
          {/* Extension duration */}
          <div>
            <div className="flex items-center gap-2 mb-2">
              <span className="w-5 h-5 rounded-full bg-cyan-600/30 text-cyan-300 text-[10px] font-bold flex items-center justify-center flex-shrink-0">2</span>
              <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
                How much to add
              </label>
              <span className="ml-auto text-white/35 text-[11px] font-mono">{duration}s extension</span>
            </div>
            <div className="flex gap-2 flex-wrap">
              {EXTENSION_DURATIONS.map((d) => (
                <button
                  key={d}
                  onClick={() => setDuration(d)}
                  className={cn(
                    'text-xs px-4 py-2 rounded-lg border font-semibold transition-all',
                    duration === d
                      ? 'bg-emerald-600 text-white border-emerald-500'
                      : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                  )}
                >
                  +{d}s
                </button>
              ))}
            </div>
          </div>

          {/* Aspect ratio */}
          <div>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Aspect Ratio</label>
            <div className="flex gap-2 flex-wrap">
              {ASPECT_RATIOS.map((ar) => (
                <button
                  key={ar.value}
                  onClick={() => setAspectRatio(ar.value)}
                  className={cn(
                    'flex items-center gap-1.5 text-xs px-4 py-2 rounded-lg border font-semibold transition-all',
                    aspectRatio === ar.value
                      ? 'bg-emerald-600 text-white border-emerald-500'
                      : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                  )}
                >
                  <span>{ar.icon}</span>
                  <span>{ar.label}</span>
                  <span className="text-[9px] font-mono opacity-60">{ar.value}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Continuation direction */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <div className="flex items-center gap-2">
                <span className="w-5 h-5 rounded-full bg-blue-600/30 text-blue-300 text-[10px] font-bold flex items-center justify-center flex-shrink-0">3</span>
                <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
                  What happens next <span className="text-white/25 normal-case font-normal">(optional)</span>
                </label>
              </div>
              <button
                onClick={handleMicClick}
                disabled={speechState === 'processing'}
                title={speechState === 'listening' ? 'Stop listening' : 'Speak the continuation'}
                className={cn(
                  'w-7 h-7 rounded-lg flex items-center justify-center transition-all border',
                  speechState === 'listening'
                    ? 'bg-red-500/20 text-red-400 border-red-500/40 animate-pulse'
                    : 'bg-white/5 text-white/30 hover:text-white/60 hover:bg-white/10 border-transparent',
                )}
              >
                {speechState === 'listening' ? <MicOff size={12} /> : <Mic size={12} />}
              </button>
            </div>
            <textarea
              value={extendPrompt}
              onChange={(e) => setExtendPrompt(e.target.value)}
              placeholder="Describe what should happen in the extension — or leave blank for a natural continuation"
              rows={2}
              className="nexus-input resize-none w-full text-sm leading-relaxed"
            />
            {speechState === 'listening' && (
              <p className="text-[11px] text-red-300 mt-1 flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse inline-block" />
                Listening… {interimText || 'describe the continuation'}
              </p>
            )}
            {speechState === 'error' && speechError && (
              <p className="text-[11px] text-red-300 mt-1">{speechError}</p>
            )}

            {/* Quick examples */}
            <div className="mt-2 flex flex-wrap gap-1.5">
              {EXTENSION_EXAMPLES.slice(0, 4).map((ex) => (
                <button
                  key={ex}
                  onClick={() => setExtendPrompt(ex)}
                  className="text-[10px] px-2.5 py-1 rounded-full border border-white/10 text-white/40 hover:border-emerald-500/30 hover:text-emerald-300 hover:bg-emerald-500/5 transition-all"
                >
                  {ex.length > 40 ? ex.slice(0, 40) + '…' : ex}
                </button>
              ))}
            </div>
          </div>

          {/* Visual summary */}
          <div className="flex items-center gap-2 bg-white/[0.03] border border-white/8 rounded-xl px-4 py-3">
            <div className="flex-1 text-center">
              <p className="text-white/30 text-[10px] uppercase tracking-wider mb-1">Your video</p>
              <div className="h-8 bg-emerald-600/20 border border-emerald-500/30 rounded-lg flex items-center justify-center">
                <Film size={12} className="text-emerald-400" />
              </div>
            </div>
            <ArrowRight size={14} className="text-white/20 flex-shrink-0" />
            <div className="flex-1 text-center">
              <p className="text-white/30 text-[10px] uppercase tracking-wider mb-1">+{duration}s added</p>
              <div className="h-8 bg-cyan-600/20 border border-cyan-500/30 border-dashed rounded-lg flex items-center justify-center">
                <Zap size={12} className="text-cyan-400" />
              </div>
            </div>
            <ArrowRight size={14} className="text-white/20 flex-shrink-0" />
            <div className="flex-1 text-center">
              <p className="text-white/30 text-[10px] uppercase tracking-wider mb-1">Extended video</p>
              <div className="h-8 bg-gradient-to-r from-emerald-600/20 to-cyan-600/20 border border-emerald-500/30 rounded-lg flex items-center justify-center">
                <Sparkles size={12} className="text-white/60" />
              </div>
            </div>
          </div>
        </>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-emerald-600 to-cyan-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-emerald-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {uploading
          ? <><Loader2 size={15} className="animate-spin" /> Uploading video…</>
          : isLoading
            ? <><Loader2 size={15} className="animate-spin" /> Extending video…</>
            : !hasVideo
              ? <><Film size={15} className="opacity-50" /> Upload a video first</>
              : <><Sparkles size={15} /> Extend with Grok (+{duration}s) →</>
        }
      </button>
    </div>
  );
}
