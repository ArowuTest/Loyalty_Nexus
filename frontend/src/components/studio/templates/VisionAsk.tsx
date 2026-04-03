'use client';

import { useState, useRef } from 'react';
import { Loader2, Upload, X, ImageIcon, Sparkles, Eye, Search, CheckCircle2 } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

const DEFAULT_EXAMPLE_QUESTIONS = [
  'What do you see in this image?',
  'Describe this image in full detail',
  'What text is visible?',
  'Identify all objects and their locations',
  'Explain what is happening in this scene',
  'What emotions does this image convey?',
  'Are there any brand logos present?',
  'What is the approximate location?',
];

export default function VisionAsk({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg          = tool.ui_config ?? {};
  const promptOptional = cfg.prompt_optional ?? false;
  // Example questions from ui_config (so they can be customised per tool from DB)
  const exampleQuestions = cfg.example_questions ?? DEFAULT_EXAMPLE_QUESTIONS;

  // When prompt_optional=true the tool auto-generates a full description
  const autoMode = promptOptional;

  const [imageUrl,    setImageUrl]    = useState('');
  const [imageFile,   setImageFile]   = useState<File | null>(null);
  const [preview,     setPreview]     = useState<string | null>(null);
  const [uploadedUrl, setUploadedUrl] = useState<string | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [question,    setQuestion]    = useState('');
  const [qSearch,     setQSearch]     = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const hasImage  = uploadedUrl || imageUrl.trim() || imageFile;
  // In auto mode (image-analyser) a question is not needed
  const isValid   = !!hasImage && !isUploading && (autoMode || question.trim().length >= 3);

  async function handleFile(file: File) {
    setImageFile(file);
    setUploadedUrl(null);
    setUploadError(null);
    // Show local preview immediately
    const reader = new FileReader();
    reader.onload = (e) => setPreview(e.target?.result as string);
    reader.readAsDataURL(file);
    setImageUrl('');
    // Upload to CDN so backend gets a valid HTTPS URL
    setIsUploading(true);
    try {
      const result = await api.uploadAsset(file);
      setUploadedUrl(result.url);
    } catch (err) {
      setUploadError('Upload failed — please try again or paste a URL instead.');
      console.error('[VisionAsk] upload error:', err);
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

  function handleSubmit() {
    if (!isValid || isLoading || isUploading || !canAfford) return;
    // Use CDN URL if we uploaded a file, otherwise use the pasted URL
    const finalUrl = uploadedUrl ?? imageUrl.trim();
    const payload: GeneratePayload = {
      prompt:    question.trim() || 'Describe this image in full detail — objects, people, text, setting, mood, and any notable elements.',
      image_url: finalUrl,
    };
    onSubmit(payload);
  }

  // Filter example questions
  const filteredQuestions = qSearch
    ? exampleQuestions.filter((q: string) => q.toLowerCase().includes(qSearch.toLowerCase()))
    : exampleQuestions;

  return (
    <div className="space-y-5">

      {/* ── Image upload ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          {cfg.upload_label ?? 'Image to analyse'}
        </label>

        {!preview && !imageUrl ? (
          <>
            <div
              onDrop={handleDrop}
              onDragOver={(e) => e.preventDefault()}
              onClick={() => fileRef.current?.click()}
              className="border-2 border-dashed border-white/15 rounded-xl p-8 flex flex-col items-center gap-3 cursor-pointer
                         hover:border-rose-500/40 hover:bg-rose-500/5 transition-all text-center"
            >
              <div className="p-3 rounded-full bg-white/5">
                <Eye size={22} className="text-white/40" />
              </div>
              <div>
                <p className="text-white/65 text-sm font-medium">Drop your image here or click to browse</p>
                <p className="text-white/30 text-xs mt-1">PNG, JPG, WebP, GIF · up to {cfg.max_file_mb ?? 20} MB</p>
              </div>
            </div>
            <input
              ref={fileRef}
              type="file"
              accept={(cfg.upload_accept ?? ['image/png', 'image/jpeg', 'image/webp', 'image/gif']).join(',')}
              className="hidden"
              onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }}
            />
            <p className="text-white/30 text-[11px] text-center mt-2">— or paste a public image URL —</p>
            <input
              type="url"
              value={imageUrl}
              onChange={(e) => setImageUrl(e.target.value)}
              placeholder="https://example.com/image.jpg"
              className="nexus-input w-full text-sm mt-1"
            />
          </>
        ) : (
          <div className="relative rounded-xl overflow-hidden border border-white/10 bg-black/30">
            <img src={preview ?? imageUrl} alt="Source" className="w-full max-h-52 object-cover" />
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
      </div>

      {/* ── Upload status banners ── */}
      {isUploading && (
        <div className="flex items-center gap-2 bg-rose-500/10 border border-rose-500/20 rounded-xl px-3 py-2">
          <Loader2 size={13} className="text-rose-400 animate-spin flex-shrink-0" />
          <p className="text-rose-300/80 text-xs">Uploading image…</p>
        </div>
      )}
      {uploadedUrl && !isUploading && (
        <div className="flex items-center gap-2 bg-green-500/10 border border-green-500/20 rounded-xl px-3 py-2">
          <CheckCircle2 size={13} className="text-green-400 flex-shrink-0" />
          <p className="text-green-300/80 text-xs">Image ready for analysis</p>
        </div>
      )}
      {uploadError && (
        <div className="bg-red-500/10 border border-red-500/20 rounded-xl px-3 py-2">
          <p className="text-red-300/80 text-xs">{uploadError}</p>
        </div>
      )}

      {/* ── Auto-mode banner (image-analyser) ── */}
      {autoMode && hasImage && (
        <div className="flex items-center gap-2 bg-rose-500/8 border border-rose-500/20 rounded-xl px-3 py-2.5">
          <Eye size={13} className="text-rose-400 flex-shrink-0" />
          <p className="text-rose-300/80 text-xs">
            AI will automatically describe everything visible — objects, text, colours, and scene context.
            Or type a specific question below.
          </p>
        </div>
      )}

      {/* ── Question input ── */}
      {hasImage && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
            {autoMode ? 'Ask a specific question' : 'Your question'}
            {autoMode && <span className="text-white/25 normal-case font-normal ml-1">(optional)</span>}
            {!autoMode && <span className="text-red-400 ml-1">*</span>}
          </label>

          <textarea
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            placeholder={cfg.prompt_placeholder ?? 'What would you like to know about this image?'}
            rows={3}
            autoFocus
            className="nexus-input resize-none w-full text-sm leading-relaxed"
          />

          {/* Question filter + examples */}
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

          <div className="flex flex-wrap gap-1.5">
            {filteredQuestions.map((q: string) => (
              <button
                key={q}
                onClick={() => setQuestion(q)}
                className={cn(
                  'text-xs px-2.5 py-1 rounded-full border font-medium transition-all',
                  question === q
                    ? 'bg-rose-600 text-white border-rose-500'
                    : 'text-white/45 border-white/12 hover:text-white/75 hover:border-white/25',
                )}
              >
                {q}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || isUploading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-rose-600 to-pink-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-rose-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isUploading
          ? <><Loader2 size={15} className="animate-spin" /> Uploading image…</>
          : isLoading
            ? <><Loader2 size={15} className="animate-spin" /> Analysing…</>
            : autoMode
              ? <><Eye size={15} /> Analyse Image →</>
              : <><Sparkles size={15} /> Ask About Image →</>
        }
      </button>
    </div>
  );
}
