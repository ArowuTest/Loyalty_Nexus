'use client';

/**
 * VideoEditor — Grok Imagine Video Editing
 *
 * Allows users to upload an existing video and describe an edit in plain English.
 * Grok Imagine rewrites the video while preserving the original duration and aspect ratio.
 *
 * Examples:
 *  - "Give her a silver necklace"
 *  - "Change the background to a tropical beach"
 *  - "Make it night time with neon lights"
 *  - "Add a red sports car in the background"
 */

import { useState, useRef } from 'react';
import {
  Loader2, Upload, X, Sparkles, AlertTriangle, Wand2,
  Mic, MicOff, Zap, Film, ChevronDown, ChevronUp,
} from 'lucide-react';
import { useSpeechToText } from '@/hooks/useSpeechToText';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

const EDIT_EXAMPLES = [
  { label: 'Add accessory',    text: 'Give her a gold necklace and matching earrings' },
  { label: 'Change background', text: 'Replace the background with a tropical beach at sunset' },
  { label: 'Change time of day', text: 'Make it night time with warm street lighting' },
  { label: 'Add weather',       text: 'Add light rain falling and wet streets' },
  { label: 'Change outfit',     text: 'Change the outfit to a formal black suit' },
  { label: 'Add object',        text: 'Add a glowing smartphone in their hand' },
  { label: 'Style change',      text: 'Make it look like a vintage 1970s film' },
  { label: 'Colour grade',      text: 'Apply a warm cinematic orange and teal colour grade' },
];

const CONSTRAINTS = [
  { icon: '⏱️', text: 'Input video: max 8.7 seconds' },
  { icon: '📐', text: 'Output keeps the same duration and aspect ratio' },
  { icon: '🎬', text: 'MP4 format recommended' },
];

export default function VideoEditor({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg = tool.ui_config ?? {};

  // ── Video upload state ─────────────────────────────────────────────────────
  const [videoFile,    setVideoFile]    = useState<File | null>(null);
  const [videoUrl,     setVideoUrl]     = useState('');
  const [videoPreview, setVideoPreview] = useState<string | null>(null);
  const [uploading,    setUploading]    = useState(false);
  const [uploadedUrl,  setUploadedUrl]  = useState('');

  // ── Edit instruction ───────────────────────────────────────────────────────
  const [editPrompt,   setEditPrompt]   = useState('');
  const [showExamples, setShowExamples] = useState(false);

  // ── Voice input ────────────────────────────────────────────────────────────
  const { speechState, speechError, interimText, handleMicClick } =
    useSpeechToText({
      onTranscript: (t) => setEditPrompt(prev => prev ? prev + ' ' + t : t),
      language: 'en-US',
    });

  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford  = tool.is_free || userPoints >= tool.point_cost;
  const hasVideo   = uploadedUrl || videoUrl.trim();
  const isValid    = !!hasVideo && editPrompt.trim().length >= 5;

  // ── File handler ───────────────────────────────────────────────────────────
  async function handleVideoFile(file: File) {
    setVideoFile(file);
    // Create local preview URL
    const localUrl = URL.createObjectURL(file);
    setVideoPreview(localUrl);
    setVideoUrl('');
    setUploadedUrl('');

    // Upload immediately so we have a CDN URL ready
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
    if (!isValid || isLoading || !canAfford || uploading) return;

    const finalVideoUrl = uploadedUrl || videoUrl.trim();

    const payload: GeneratePayload = {
      prompt:    editPrompt.trim(),
      image_url: finalVideoUrl, // backend reads video_url from extra_params first, falls back to image_url
      extra_params: {
        video_url: finalVideoUrl,
      },
    };
    onSubmit(payload);
  }

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <div className="space-y-5">

      {/* ── Grok badge ── */}
      <div className="flex items-center gap-2 bg-gradient-to-r from-violet-500/10 to-pink-500/10 border border-violet-500/20 rounded-xl px-3 py-2.5">
        <Zap size={13} className="text-violet-400 flex-shrink-0" />
        <p className="text-violet-300/80 text-xs leading-relaxed">
          <span className="font-semibold text-violet-300">Grok Imagine Video Editing</span>
          {' '}— Describe your edit in plain English and Grok rewrites the video. No timeline. No keyframes. Just words.
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
          <span className="w-5 h-5 rounded-full bg-violet-600/30 text-violet-300 text-[10px] font-bold flex items-center justify-center flex-shrink-0">1</span>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            Upload your video to edit
          </label>
        </div>

        {!videoPreview && !videoUrl ? (
          <>
            <div
              onDrop={handleDrop}
              onDragOver={(e) => e.preventDefault()}
              onClick={() => fileRef.current?.click()}
              className="border-2 border-dashed border-white/15 rounded-xl p-8 flex flex-col items-center gap-3 cursor-pointer
                         hover:border-violet-500/40 hover:bg-violet-500/5 transition-all text-center"
            >
              <div className="p-3 rounded-full bg-white/5">
                <Film size={22} className="text-white/40" />
              </div>
              <div>
                <p className="text-white/65 text-sm font-medium">Upload the video to edit</p>
                <p className="text-white/30 text-xs mt-1">MP4, MOV, WebM · max 8.7 seconds · up to {cfg.max_file_mb ?? 100} MB</p>
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
                <Film size={16} className="text-violet-400 flex-shrink-0" />
                <span className="text-white/60 text-sm truncate">{videoUrl}</span>
              </div>
            )}
            <button
              onClick={clearVideo}
              className="absolute top-2 right-2 p-1.5 bg-black/70 rounded-full text-white/60 hover:text-white transition-colors"
            >
              <X size={14} />
            </button>
            {/* Upload status */}
            <div className="absolute bottom-2 left-2 flex items-center gap-1.5 bg-black/70 rounded-full px-2.5 py-1">
              {uploading ? (
                <><Loader2 size={10} className="text-violet-400 animate-spin" /><span className="text-white/50 text-[11px]">Uploading…</span></>
              ) : uploadedUrl ? (
                <><span className="w-1.5 h-1.5 rounded-full bg-green-500 inline-block" /><span className="text-white/50 text-[11px]">{videoFile?.name ?? 'Video ready'}</span></>
              ) : (
                <><Film size={10} className="text-violet-400" /><span className="text-white/50 text-[11px]">{videoFile?.name ?? 'Video URL'}</span></>
              )}
            </div>
          </div>
        )}
      </div>

      {/* ── Step 2: Edit instruction ── */}
      {(hasVideo || videoUrl) && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="w-5 h-5 rounded-full bg-pink-600/30 text-pink-300 text-[10px] font-bold flex items-center justify-center flex-shrink-0">2</span>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
              Describe your edit
            </label>
            <button
              onClick={handleMicClick}
              disabled={speechState === 'processing'}
              title={speechState === 'listening' ? 'Stop listening' : 'Speak your edit'}
              className={cn(
                'ml-auto w-7 h-7 rounded-lg flex items-center justify-center transition-all border',
                speechState === 'listening'
                  ? 'bg-red-500/20 text-red-400 border-red-500/40 animate-pulse'
                  : 'bg-white/5 text-white/30 hover:text-white/60 hover:bg-white/10 border-transparent',
              )}
            >
              {speechState === 'listening' ? <MicOff size={12} /> : <Mic size={12} />}
            </button>
          </div>

          <textarea
            value={editPrompt}
            onChange={(e) => setEditPrompt(e.target.value)}
            placeholder={cfg.prompt_placeholder ?? 'Describe the edit in plain English — e.g. "Give her a gold necklace" or "Change the background to a beach"'}
            rows={3}
            className="nexus-input resize-none w-full text-sm leading-relaxed"
          />
          {speechState === 'listening' && (
            <p className="text-[11px] text-red-300 mt-1 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse inline-block" />
              Listening… {interimText || 'describe your edit'}
            </p>
          )}
          {speechState === 'error' && speechError && (
            <p className="text-[11px] text-red-300 mt-1">{speechError}</p>
          )}

          {/* Character count */}
          <div className="flex justify-end mt-1">
            <span className={cn(
              'text-[10px] font-mono',
              editPrompt.length > 400 ? 'text-amber-400' : 'text-white/20',
            )}>
              {editPrompt.length}/500
            </span>
          </div>

          {/* Edit examples */}
          <div className="mt-2">
            <button
              onClick={() => setShowExamples(!showExamples)}
              className="flex items-center gap-1.5 text-white/35 text-[11px] hover:text-white/60 transition-colors"
            >
              {showExamples ? <ChevronUp size={11} /> : <ChevronDown size={11} />}
              {showExamples ? 'Hide examples' : 'Show edit examples'}
            </button>
            {showExamples && (
              <div className="mt-2 grid grid-cols-2 gap-1.5">
                {EDIT_EXAMPLES.map((ex) => (
                  <button
                    key={ex.label}
                    onClick={() => { setEditPrompt(ex.text); setShowExamples(false); }}
                    className="text-left px-3 py-2 rounded-lg bg-white/[0.04] border border-white/8 hover:border-pink-500/30 hover:bg-pink-500/5 transition-all group"
                  >
                    <p className="text-white/60 text-[10px] font-semibold group-hover:text-pink-300 transition-colors">{ex.label}</p>
                    <p className="text-white/35 text-[10px] mt-0.5 leading-tight line-clamp-2">{ex.text}</p>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford || uploading}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford && !uploading
            ? 'bg-gradient-to-r from-violet-600 to-pink-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-violet-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {uploading
          ? <><Loader2 size={15} className="animate-spin" /> Uploading video…</>
          : isLoading
            ? <><Loader2 size={15} className="animate-spin" /> Editing video…</>
            : !hasVideo
              ? <><Film size={15} className="opacity-50" /> Upload a video first</>
              : editPrompt.trim().length < 5
                ? <><Wand2 size={15} className="opacity-50" /> Describe your edit</>
                : <><Sparkles size={15} /> Apply Edit with Grok →</>
        }
      </button>
    </div>
  );
}
