'use client';

import { useState, useRef } from 'react';
import { Loader2, Upload, X, ImageIcon, Sparkles, AlertTriangle } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_STYLE_TAGS = [
  'Smooth motion', 'Dramatic',  'Slow motion', 'Zoom in', 'Zoom out',
  'Pan left',      'Pan right', 'Parallax',   'Vibrant',  'Cinematic',
];

export default function VideoAnimator({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg = tool.ui_config ?? {};
  const styleTags = cfg.style_tags ?? DEFAULT_STYLE_TAGS;

  const [imageUrl,   setImageUrl]   = useState('');
  const [imageFile,  setImageFile]  = useState<File | null>(null);
  const [preview,    setPreview]    = useState<string | null>(null);
  const [motionPrompt, setMotionPrompt] = useState('');
  const [selStyles,  setSelStyles]  = useState<string[]>([]);
  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const hasImage  = imageUrl.trim() || imageFile;
  const isValid   = !!hasImage;

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

  function toggleStyle(s: string) {
    setSelStyles((prev) => prev.includes(s) ? prev.filter((t) => t !== s) : [...prev, s]);
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const finalUrl = imageFile && preview ? preview : imageUrl.trim();
    const stylePrefix = selStyles.length > 0 ? `[${selStyles.join(', ')}] ` : '';
    const payload: GeneratePayload = {
      prompt:     stylePrefix + (motionPrompt.trim() || 'Animate this image with natural motion'),
      image_url:  finalUrl,
      style_tags: selStyles.length > 0 ? selStyles : undefined,
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

      {/* ── Image upload ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          {cfg.upload_label ?? 'Source Image (required)'}
        </label>

        {!preview && !imageUrl ? (
          <>
            <div
              onDrop={handleDrop}
              onDragOver={(e) => e.preventDefault()}
              onClick={() => fileRef.current?.click()}
              className="border-2 border-dashed border-white/15 rounded-xl p-8 flex flex-col items-center gap-3 cursor-pointer
                         hover:border-cyan-500/40 hover:bg-cyan-500/5 transition-all text-center"
            >
              <div className="p-3 rounded-full bg-white/5">
                <Upload size={22} className="text-white/40" />
              </div>
              <div>
                <p className="text-white/65 text-sm font-medium">Upload the image to animate</p>
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

      {/* ── Motion style tags ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Motion style</label>
        <div className="flex flex-wrap gap-1.5">
          {styleTags.map((s) => (
            <button
              key={s}
              onClick={() => toggleStyle(s)}
              className={cn(
                'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                selStyles.includes(s)
                  ? 'bg-cyan-600 text-white border-cyan-500'
                  : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
              )}
            >
              {s}
            </button>
          ))}
        </div>
      </div>

      {/* ── Motion prompt (optional) ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
          Motion description <span className="text-white/25 normal-case font-normal">(optional)</span>
        </label>
        <textarea
          value={motionPrompt}
          onChange={(e) => setMotionPrompt(e.target.value)}
          placeholder={cfg.prompt_placeholder ?? 'Describe how to animate it — e.g. Camera slowly zooms in, trees sway in wind, water ripples…'}
          rows={3}
          autoFocus={!!preview}
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />
      </div>

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-cyan-600 to-blue-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-cyan-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Animating…</>
          : <><Sparkles size={15} /> Animate Image →</>
        }
      </button>
    </div>
  );
}
