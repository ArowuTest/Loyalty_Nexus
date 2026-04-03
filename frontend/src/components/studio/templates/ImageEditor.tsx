'use client';

import { useState, useRef } from 'react';
import { Loader2, Upload, X, ImageIcon, Sparkles, Sliders, ChevronDown, ChevronUp, CheckCircle2 } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

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
  'Add a cinematic film grain',
  'Make it look like a comic book',
];

const STRENGTH_LEVELS = [
  { label: 'Subtle',   value: 0.3, desc: 'Light touch' },
  { label: 'Moderate', value: 0.6, desc: 'Balanced'    },
  { label: 'Strong',   value: 0.8, desc: 'Bold change' },
  { label: 'Max',      value: 1.0, desc: 'Full rework' },
];

export default function ImageEditor({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg             = tool.ui_config ?? {};
  const editSuggestions = (cfg.edit_suggestions ?? DEFAULT_EDIT_SUGGESTIONS) as string[];
  // bg-remover sets show_edit_prompt: false and prompt_optional: true
  const showEditPrompt  = cfg.show_edit_prompt !== false; // default true
  const promptOptional  = cfg.prompt_optional === true;   // default false

  const [imageUrl,     setImageUrl]     = useState('');
  const [imageFile,    setImageFile]    = useState<File | null>(null);
  const [preview,      setPreview]      = useState<string | null>(null);
  const [uploadedUrl,  setUploadedUrl]  = useState<string | null>(null);
  const [isUploading,  setIsUploading]  = useState(false);
  const [uploadError,  setUploadError]  = useState<string | null>(null);
  const [editPrompt,   setEditPrompt]   = useState('');
  const [strength,     setStrength]     = useState(0.6);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford  = tool.is_free || userPoints >= tool.point_cost;
  const hasImage   = uploadedUrl || imageUrl.trim() || imageFile;
  const promptOk   = promptOptional || !showEditPrompt || editPrompt.trim().length >= 3;
  const isValid    = !!hasImage && promptOk && !isUploading;
  const isBusy     = isLoading || isUploading;

  async function handleFile(file: File) {
    setImageFile(file);
    setUploadedUrl(null);
    setUploadError(null);
    // Show local preview immediately
    const reader = new FileReader();
    reader.onload = (e) => setPreview(e.target?.result as string);
    reader.readAsDataURL(file);
    setImageUrl('');
    // Upload to CDN
    setIsUploading(true);
    try {
      const result = await api.uploadAsset(file);
      setUploadedUrl(result.url);
    } catch (err) {
      setUploadError('Upload failed — please try again or paste a URL instead.');
      console.error('[ImageEditor] upload error:', err);
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
    setEditPrompt('');
  }

  function handleSubmit() {
    if (!isValid || isBusy || !canAfford) return;
    // Use the CDN URL if we uploaded a file, otherwise use the pasted URL
    const finalUrl = uploadedUrl ?? imageUrl.trim();
    const payload: GeneratePayload = {
      prompt:    showEditPrompt ? (editPrompt.trim() || 'Remove the background') : 'Remove the background',
      image_url: finalUrl,
      extra_params: {
        strength,
      },
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
          /* Image preview card */
          <div className="relative rounded-xl overflow-hidden border border-white/10 bg-black/30">
            <img
              src={preview ?? imageUrl}
              alt="Original"
              className="w-full max-h-52 object-cover"
            />
            {/* Overlay labels */}
            <div className="absolute inset-0 flex pointer-events-none">
              <div className="flex-1 flex items-end justify-start p-2">
                <div className="flex items-center gap-1.5 bg-black/70 rounded-full px-2.5 py-1">
                  <ImageIcon size={11} className="text-white/60" />
                  <span className="text-white/60 text-[11px]">Original</span>
                </div>
              </div>
              <div className="flex-1 flex items-end justify-end p-2">
                <div className="flex items-center gap-1 bg-purple-600/80 rounded-full px-2.5 py-1">
                  <Sparkles size={11} className="text-white" />
                  <span className="text-white text-[11px] font-medium">AI Edit</span>
                </div>
              </div>
            </div>
            <button
              onClick={clearImage}
              className="absolute top-2 right-2 p-1.5 bg-black/70 rounded-full text-white/60 hover:text-white transition-colors"
            >
              <X size={14} />
            </button>
            <p className="text-white/30 text-[11px] px-3 py-1.5 bg-black/40">
              {imageFile?.name ?? 'URL'} · {imageFile ? `${(imageFile.size / 1024).toFixed(0)} KB` : 'External'}
            </p>
          </div>
        )}

        {/* Upload status banners */}
        {isUploading && (
          <div className="flex items-center gap-2 mt-2 bg-purple-500/10 border border-purple-500/20 rounded-xl px-3 py-2">
            <Loader2 size={13} className="text-purple-400 animate-spin flex-shrink-0" />
            <p className="text-purple-300/80 text-xs">Uploading image…</p>
          </div>
        )}
        {uploadedUrl && !isUploading && (
          <div className="flex items-center gap-2 mt-2 bg-green-500/10 border border-green-500/20 rounded-xl px-3 py-2">
            <CheckCircle2 size={13} className="text-green-400 flex-shrink-0" />
            <p className="text-green-300/80 text-xs">Image ready for editing</p>
          </div>
        )}
        {uploadError && (
          <div className="mt-2 bg-red-500/10 border border-red-500/20 rounded-xl px-3 py-2">
            <p className="text-red-300/80 text-xs">{uploadError}</p>
          </div>
        )}
        {/* Output note for bg-remover */}
        {cfg.output_note && (
          <p className="text-white/30 text-[11px] text-center mt-2">{cfg.output_note}</p>
        )}
      </div>

      {/* ── Step 2: Edit instruction (hidden for bg-remover) ── */}
      {showEditPrompt && (preview || imageUrl) && (
        <div>
          <div className="flex items-center gap-2 mb-1.5">
            <span className="w-5 h-5 rounded-full bg-purple-600/30 text-purple-300 text-[10px] font-bold flex items-center justify-center flex-shrink-0">2</span>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
              Edit instruction
              {promptOptional && <span className="text-white/25 normal-case font-normal ml-1">(optional)</span>}
            </label>
          </div>

          {/* Quick-edit chips */}
          {editSuggestions.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mb-2">
              {editSuggestions.map((s) => (
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
          )}

          <textarea
            value={editPrompt}
            onChange={(e) => setEditPrompt(e.target.value)}
            placeholder={cfg.prompt_placeholder ?? 'Or describe exactly what to change — be specific for best results…'}
            rows={3}
            autoFocus
            className="nexus-input resize-none w-full text-sm leading-relaxed"
          />

          {/* ── Advanced: Strength slider ── */}
          <button
            onClick={() => setShowAdvanced((v) => !v)}
            className="flex items-center gap-1.5 text-white/30 text-xs hover:text-white/55 transition-colors mt-2"
          >
            <Sliders size={11} />
            Advanced settings
            {showAdvanced ? <ChevronUp size={11} /> : <ChevronDown size={11} />}
          </button>

          {showAdvanced && (
            <div className="mt-3 space-y-3 p-3 rounded-xl bg-white/[0.03] border border-white/[0.06]">
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Edit Strength</label>
                  <span className="text-white/40 text-xs font-medium">
                    {STRENGTH_LEVELS.find((s) => s.value === strength)?.label ?? 'Custom'}
                  </span>
                </div>
                <div className="grid grid-cols-4 gap-1.5">
                  {STRENGTH_LEVELS.map((s) => (
                    <button
                      key={s.value}
                      onClick={() => setStrength(s.value)}
                      className={cn(
                        'flex flex-col items-center gap-0.5 py-2 rounded-xl border text-xs font-medium transition-all',
                        strength === s.value
                          ? 'bg-purple-600/25 border-purple-500/60 text-purple-200'
                          : 'border-white/10 text-white/40 hover:border-white/20 hover:text-white/65',
                      )}
                    >
                      <span className="font-bold">{s.label}</span>
                      <span className="text-[9px] opacity-60">{s.desc}</span>
                    </button>
                  ))}
                </div>
                <p className="text-white/20 text-[11px] mt-1.5">
                  Higher strength = more dramatic changes, less resemblance to original
                </p>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isBusy || !canAfford}
        className={cn(
          'w-full py-4 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isBusy && canAfford
            ? 'bg-gradient-to-r from-purple-600 to-indigo-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-purple-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isUploading
          ? <><Loader2 size={15} className="animate-spin" /> Uploading image…</>
          : isLoading
            ? <><Loader2 size={15} className="animate-spin" /> {showEditPrompt ? 'Editing image…' : 'Removing background…'}</>
            : <><Sparkles size={15} /> {showEditPrompt ? 'Apply Edit' : 'Remove Background'}</>
        }
      </button>

      {!tool.is_free && (
        <p className="text-white/20 text-[11px] text-center -mt-2">
          {tool.point_cost} PulsePoints per generation · {userPoints} available
        </p>
      )}
    </div>
  );
}
