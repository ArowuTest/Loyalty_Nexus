'use client';

/**
 * CodePro template — Nexus Code Pro
 *
 * Layout:
 *   1. Code question textarea (always visible, required)
 *   2. Example question chips (from ui_config)
 *   3. Optional screenshot / image upload (collapsed by default, expands on click)
 *   4. Generate button
 *
 * The image is truly optional — the user can ask a code question without
 * attaching any screenshot. When an image IS attached, it is sent as
 * image_url so the backend can use multimodal vision for visual debugging.
 */

import { useState, useRef } from 'react';
import {
  Loader2, Upload, X, ImageIcon, Sparkles, Code2,
  ChevronDown, ChevronUp, CheckCircle2, Mic, MicOff, Search,
} from 'lucide-react';
import { useSpeechToText } from '@/hooks/useSpeechToText';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

const DEFAULT_EXAMPLE_QUESTIONS = [
  'Why is this error happening and how do I fix it?',
  'Explain what this code does and suggest improvements',
  'Convert this UI design into React + Tailwind code',
  'Debug this traceback and show the corrected code',
  'What architecture pattern is shown in this diagram?',
  'Write unit tests for this function',
  'Optimise this SQL query for performance',
  'Explain this code to a junior developer',
];

export default function CodePro({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg              = tool.ui_config ?? {};
  const exampleQuestions = cfg.example_questions ?? DEFAULT_EXAMPLE_QUESTIONS;
  const promptPlaceholder =
    cfg.prompt_placeholder ??
    'Describe your code problem — or attach a screenshot of the error, UI bug, or architecture diagram…';

  const [question,     setQuestion]     = useState('');
  const [qSearch,      setQSearch]      = useState('');
  const [showUpload,   setShowUpload]   = useState(false);
  const [imageUrl,     setImageUrl]     = useState('');
  const [imageFile,    setImageFile]    = useState<File | null>(null);
  const [preview,      setPreview]      = useState<string | null>(null);
  const [uploadedUrl,  setUploadedUrl]  = useState<string | null>(null);
  const [isUploading,  setIsUploading]  = useState(false);
  const [uploadError,  setUploadError]  = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  const { speechState, speechError, interimText, handleMicClick } =
    useSpeechToText({
      onTranscript: (t) => setQuestion(prev => prev ? prev + ' ' + t : t),
      language: 'en-US',
    });

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const hasImage  = uploadedUrl || imageUrl.trim() || imageFile;
  const isValid   = question.trim().length >= 3 && !isUploading && canAfford;

  // ── File upload ────────────────────────────────────────────────────────────
  async function handleFile(file: File) {
    setImageFile(file);
    setUploadedUrl(null);
    setUploadError(null);
    const reader = new FileReader();
    reader.onload = (e) => setPreview(e.target?.result as string);
    reader.readAsDataURL(file);
    setImageUrl('');
    setIsUploading(true);
    try {
      const result = await api.uploadAsset(file);
      setUploadedUrl(result.url);
    } catch (err) {
      setUploadError('Upload failed — please try again or paste a URL instead.');
      console.error('[CodePro] upload error:', err);
    } finally {
      setIsUploading(false);
    }
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file && file.type.startsWith('image/')) handleFile(file);
  }

  function clearImage() {
    setImageFile(null);
    setPreview(null);
    setImageUrl('');
    setUploadedUrl(null);
    setUploadError(null);
  }

  // ── Submit ─────────────────────────────────────────────────────────────────
  function handleSubmit() {
    if (!isValid || isLoading) return;
    const finalImageUrl = uploadedUrl ?? imageUrl.trim() ?? undefined;
    const payload: GeneratePayload = {
      prompt:    question.trim(),
      image_url: finalImageUrl || undefined,
    };
    onSubmit(payload);
  }

  // ── Filter example questions ───────────────────────────────────────────────
  const filteredQuestions = qSearch
    ? exampleQuestions.filter((q: string) => q.toLowerCase().includes(qSearch.toLowerCase()))
    : exampleQuestions;

  return (
    <div className="space-y-5">

      {/* ── 1. Code question textarea (always visible) ── */}
      <div>
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            Your code question <span className="text-red-400">*</span>
          </label>
          <button
            onClick={handleMicClick}
            disabled={speechState === 'processing'}
            title={speechState === 'listening' ? 'Stop listening' : 'Speak your question'}
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
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder={promptPlaceholder}
          rows={5}
          autoFocus
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />

        {speechState === 'listening' && (
          <p className="text-[11px] text-red-300 mt-1 flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse inline-block" />
            Listening… {interimText || 'ask your question'}
          </p>
        )}
        {speechState === 'error' && speechError && (
          <p className="text-[11px] text-red-300 mt-1">{speechError}</p>
        )}

        {/* Example question chips */}
        {exampleQuestions.length > 4 && (
          <div className="relative mt-2 mb-1.5">
            <Search size={11} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-white/30 pointer-events-none" />
            <input
              type="text"
              value={qSearch}
              onChange={(e) => setQSearch(e.target.value)}
              placeholder="Filter examples…"
              className="nexus-input w-full text-xs pl-7 py-1.5"
            />
          </div>
        )}
        <div className="flex flex-wrap gap-1.5 mt-2">
          {filteredQuestions.map((q: string) => (
            <button
              key={q}
              onClick={() => setQuestion(q)}
              className={cn(
                'text-xs px-2.5 py-1 rounded-full border font-medium transition-all',
                question === q
                  ? 'bg-violet-600 text-white border-violet-500'
                  : 'text-white/45 border-white/12 hover:text-white/75 hover:border-white/25',
              )}
            >
              {q}
            </button>
          ))}
        </div>
      </div>

      {/* ── 2. Optional screenshot upload (collapsible) ── */}
      <div className="border border-white/10 rounded-xl overflow-hidden">
        <button
          type="button"
          onClick={() => setShowUpload(v => !v)}
          className="w-full flex items-center justify-between px-4 py-3 text-white/50 hover:text-white/70 transition-colors"
        >
          <div className="flex items-center gap-2">
            <Upload size={13} className="text-violet-400" />
            <span className="text-[11px] uppercase tracking-wider font-semibold">
              {cfg.upload_label ?? 'Attach screenshot (optional)'}
            </span>
            {hasImage && (
              <span className="text-[10px] bg-violet-600/30 text-violet-300 px-1.5 py-0.5 rounded-full border border-violet-500/30">
                1 image
              </span>
            )}
          </div>
          {showUpload ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        </button>

        {showUpload && (
          <div className="px-4 pb-4 space-y-3 border-t border-white/8">
            <p className="text-white/30 text-[11px] mt-3">
              Attach a screenshot of your error, UI bug, or architecture diagram for context-aware debugging.
            </p>

            {!preview && !imageUrl ? (
              <>
                <div
                  onDrop={handleDrop}
                  onDragOver={(e) => e.preventDefault()}
                  onClick={() => fileRef.current?.click()}
                  className="border-2 border-dashed border-white/12 rounded-xl p-6 flex flex-col items-center gap-2 cursor-pointer
                             hover:border-violet-500/40 hover:bg-violet-500/5 transition-all text-center"
                >
                  <div className="p-2.5 rounded-full bg-white/5">
                    <ImageIcon size={18} className="text-white/35" />
                  </div>
                  <p className="text-white/55 text-xs font-medium">Drop screenshot here or click to browse</p>
                  <p className="text-white/25 text-[11px]">PNG, JPG, WebP · up to {cfg.max_file_mb ?? 20} MB</p>
                </div>
                <input
                  ref={fileRef}
                  type="file"
                  accept={(cfg.upload_accept ?? ['image/png', 'image/jpeg', 'image/webp']).join(',')}
                  className="hidden"
                  onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }}
                />
                <p className="text-white/25 text-[11px] text-center">— or paste a public image URL —</p>
                <input
                  type="url"
                  value={imageUrl}
                  onChange={(e) => setImageUrl(e.target.value)}
                  placeholder="https://example.com/screenshot.png"
                  className="nexus-input w-full text-sm"
                />
              </>
            ) : (
              <div className="relative rounded-xl overflow-hidden border border-white/10 bg-black/30">
                <img src={preview ?? imageUrl} alt="Screenshot" className="w-full max-h-44 object-cover" />
                <button
                  onClick={clearImage}
                  className="absolute top-2 right-2 p-1.5 bg-black/70 rounded-full text-white/60 hover:text-white transition-colors"
                >
                  <X size={14} />
                </button>
                <div className="absolute bottom-2 left-2 flex items-center gap-1.5 bg-black/70 rounded-full px-2.5 py-1">
                  <ImageIcon size={11} className="text-white/60" />
                  <span className="text-white/60 text-[11px]">{imageFile?.name ?? 'URL image'}</span>
                </div>
              </div>
            )}

            {/* Upload status banners */}
            {isUploading && (
              <div className="flex items-center gap-2 bg-violet-500/10 border border-violet-500/20 rounded-xl px-3 py-2">
                <Loader2 size={13} className="text-violet-400 animate-spin flex-shrink-0" />
                <p className="text-violet-300/80 text-xs">Uploading screenshot…</p>
              </div>
            )}
            {uploadedUrl && !isUploading && (
              <div className="flex items-center gap-2 bg-green-500/10 border border-green-500/20 rounded-xl px-3 py-2">
                <CheckCircle2 size={13} className="text-green-400 flex-shrink-0" />
                <p className="text-green-300/80 text-xs">Screenshot ready — AI will use it for visual debugging</p>
              </div>
            )}
            {uploadError && (
              <div className="bg-red-500/10 border border-red-500/20 rounded-xl px-3 py-2">
                <p className="text-red-300/80 text-xs">{uploadError}</p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* ── 3. Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || isUploading}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && !isUploading
            ? 'bg-gradient-to-r from-violet-600 to-purple-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-violet-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isUploading
          ? <><Loader2 size={15} className="animate-spin" /> Uploading screenshot…</>
          : isLoading
            ? <><Loader2 size={15} className="animate-spin" /> Generating code…</>
            : hasImage
              ? <><Code2 size={15} /> Generate with Visual Context →</>
              : <><Sparkles size={15} /> Generate Code →</>
        }
      </button>
    </div>
  );
}
