'use client';

import { useState, useRef } from 'react';
import { Loader2, Upload, X, ImageIcon, Sparkles, Eye } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const EXAMPLE_QUESTIONS = [
  'What do you see in this image?',
  'Describe this image in detail',
  'What text is visible?',
  'Identify objects and their locations',
  'Explain what is happening in this scene',
  'What emotions does this convey?',
];

export default function VisionAsk({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg = tool.ui_config ?? {};
  const promptOptional = cfg.prompt_optional ?? false;

  const [imageUrl,   setImageUrl]   = useState('');
  const [imageFile,  setImageFile]  = useState<File | null>(null);
  const [preview,    setPreview]    = useState<string | null>(null);
  const [question,   setQuestion]   = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const hasImage  = imageUrl.trim() || imageFile;
  const isValid   = !!hasImage && (promptOptional || question.trim().length >= 3);

  function handleFile(file: File) {
    setImageFile(file);
    const reader = new FileReader();
    reader.onload = (e) => setPreview(e.target?.result as string);
    reader.readAsDataURL(file);
    setImageUrl('');
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
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const finalUrl = imageFile && preview ? preview : imageUrl.trim();
    const payload: GeneratePayload = {
      prompt:    question.trim() || 'Describe this image in detail',
      image_url: finalUrl,
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Image upload ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          {cfg.upload_label ?? 'Image to Analyse'}
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
                <p className="text-white/65 text-sm font-medium">Drop an image here or click to browse</p>
                <p className="text-white/30 text-xs mt-1">PNG, JPG, WebP · up to {cfg.max_file_mb ?? 10} MB</p>
              </div>
            </div>
            <input
              ref={fileRef}
              type="file"
              accept={(cfg.upload_accept ?? ['image/png', 'image/jpeg', 'image/webp']).join(',')}
              className="hidden"
              onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }}
            />
            <p className="text-white/30 text-[11px] text-center mt-2">— or paste a URL —</p>
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
            <img src={preview ?? imageUrl} alt="Source" className="w-full max-h-48 object-cover" />
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

      {/* ── Question ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
          Your question
          {promptOptional && <span className="text-white/25 normal-case font-normal ml-1">(optional)</span>}
        </label>
        <textarea
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder={cfg.prompt_placeholder ?? 'What would you like to know about this image?'}
          rows={3}
          autoFocus={!!preview}
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />
        <p className="text-white/30 text-xs mb-1.5 mt-2">Examples:</p>
        <div className="flex flex-wrap gap-1.5">
          {EXAMPLE_QUESTIONS.map((q) => (
            <button
              key={q}
              onClick={() => setQuestion(q)}
              className="text-xs px-2.5 py-1 rounded-full border border-white/12 text-white/45 hover:text-white/75 hover:border-white/25 transition-all"
            >
              {q}
            </button>
          ))}
        </div>
      </div>

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-rose-600 to-pink-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-rose-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Analysing…</>
          : <><Sparkles size={15} /> Analyse Image →</>
        }
      </button>
    </div>
  );
}
