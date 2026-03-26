'use client';

import { useState, useRef } from 'react';
import { Loader2, Upload, X, ImageIcon, Sparkles } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_EDIT_SUGGESTIONS = [
  'Remove the background',
  'Add sunset lighting',
  'Make it look like a painting',
  'Add dramatic shadows',
  'Convert to black & white',
  'Make the colours more vibrant',
  'Add a smooth bokeh background',
  'Upscale & enhance sharpness',
  'Change background to a beach',
  'Add professional studio lighting',
];

export default function ImageEditor({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg             = tool.ui_config ?? {};
  // Quick-edit chips from DB config, falling back to defaults
  const editSuggestions = cfg.edit_suggestions ?? DEFAULT_EDIT_SUGGESTIONS;

  const [imageUrl,    setImageUrl]    = useState('');
  const [imageFile,   setImageFile]   = useState<File | null>(null);
  const [preview,     setPreview]     = useState<string | null>(null);
  const [editPrompt,  setEditPrompt]  = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const hasImage  = imageUrl.trim() || imageFile;
  const isValid   = !!hasImage && editPrompt.trim().length >= 3;

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
    setEditPrompt('');
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const finalUrl = imageFile && preview ? preview : imageUrl.trim();
    const payload: GeneratePayload = {
      prompt:    editPrompt.trim(),
      image_url: finalUrl,
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Step 1: Image upload / URL ── */}
      <div>
        <div className="flex items-center gap-2 mb-2">
          <span className="w-5 h-5 rounded-full bg-purple-600/30 text-purple-300 text-[10px] font-bold flex items-center justify-center flex-shrink-0">1</span>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            {cfg.upload_label ?? 'Upload your photo'}
          </label>
        </div>

        {!preview && !imageUrl ? (
          <>
            <div
              onDrop={handleDrop}
              onDragOver={(e) => e.preventDefault()}
              onClick={() => fileRef.current?.click()}
              className="border-2 border-dashed border-white/15 rounded-xl p-8 flex flex-col items-center gap-3 cursor-pointer
                         hover:border-purple-500/40 hover:bg-purple-500/5 transition-all text-center"
            >
              <div className="p-3 rounded-full bg-white/5">
                <Upload size={22} className="text-white/40" />
              </div>
              <div>
                <p className="text-white/65 text-sm font-medium">Drop your photo here or click to browse</p>
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
            <p className="text-white/30 text-[11px] text-center mt-2">— or paste an image URL —</p>
            <input
              type="url"
              value={imageUrl}
              onChange={(e) => setImageUrl(e.target.value)}
              placeholder="https://example.com/your-photo.jpg"
              className="nexus-input w-full text-sm mt-1"
            />
          </>
        ) : (
          /* Image preview — original preserved */
          <div className="relative rounded-xl overflow-hidden border border-white/10 bg-black/30">
            <img
              src={preview ?? imageUrl}
              alt="Original"
              className="w-full max-h-52 object-cover"
            />
            <button
              onClick={clearImage}
              className="absolute top-2 right-2 p-1.5 bg-black/70 rounded-full text-white/60 hover:text-white transition-colors"
            >
              <X size={14} />
            </button>
            {/* "Original" label */}
            <div className="absolute bottom-2 left-2 flex items-center gap-1.5 bg-black/70 rounded-full px-2.5 py-1">
              <ImageIcon size={11} className="text-white/60" />
              <span className="text-white/60 text-[11px]">Original · {imageFile?.name ?? 'URL'}</span>
            </div>
            {/* "→ Edited" hint */}
            <div className="absolute bottom-2 right-2 flex items-center gap-1 bg-purple-600/70 rounded-full px-2.5 py-1">
              <span className="text-white text-[11px] font-medium">→ AI Edit</span>
            </div>
          </div>
        )}
      </div>

      {/* ── Step 2: Edit instruction ── */}
      {(preview || imageUrl) && (
        <div>
          <div className="flex items-center gap-2 mb-1.5">
            <span className="w-5 h-5 rounded-full bg-purple-600/30 text-purple-300 text-[10px] font-bold flex items-center justify-center flex-shrink-0">2</span>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
              Edit instruction
            </label>
          </div>

          {/* Quick-edit chips (from ui_config) */}
          <div className="flex flex-wrap gap-1.5 mb-2">
            {editSuggestions.map((s: string) => (
              <button
                key={s}
                onClick={() => setEditPrompt(s)}
                className={cn(
                  'text-xs px-2.5 py-1 rounded-full border font-medium transition-all',
                  editPrompt === s
                    ? 'bg-purple-600 text-white border-purple-500'
                    : 'text-white/45 border-white/12 hover:text-white/75 hover:border-white/25',
                )}
              >
                {s}
              </button>
            ))}
          </div>

          <textarea
            value={editPrompt}
            onChange={(e) => setEditPrompt(e.target.value)}
            placeholder={cfg.prompt_placeholder ?? 'Or describe exactly what to change — be specific for best results…'}
            rows={3}
            autoFocus
            className="nexus-input resize-none w-full text-sm leading-relaxed"
          />
        </div>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-purple-600 to-indigo-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-purple-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Editing image…</>
          : <><Sparkles size={15} /> Apply Edit →</>
        }
      </button>
    </div>
  );
}
