'use client';

import { useState, useRef } from 'react';
import { Loader2, Upload, X, ImageIcon, Sparkles, AlertTriangle, Mic, MicOff } from 'lucide-react';
import { useSpeechToText } from '@/hooks/useSpeechToText';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

const DEFAULT_STYLE_TAGS = [
  'Smooth motion', 'Dramatic', 'Slow motion', 'Zoom in', 'Zoom out',
  'Pan left',      'Pan right', 'Parallax',  'Vibrant',  'Cinematic',
];

const DEFAULT_DURATIONS = [5, 8, 10];

const INTENSITY_LABELS = ['Subtle', 'Moderate', 'Strong'] as const;

export default function VideoAnimator({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg       = tool.ui_config ?? {};
  const styleTags = cfg.style_tags       ?? DEFAULT_STYLE_TAGS;
  const durations = cfg.duration_options ?? DEFAULT_DURATIONS;

  const [imageUrl,      setImageUrl]      = useState('');
  const [imageFile,     setImageFile]     = useState<File | null>(null);
  const [preview,       setPreview]       = useState<string | null>(null);
  const [motionPrompt,  setMotionPrompt]  = useState('');

  // Web Speech API mic
  const { speechState, speechError, interimText, handleMicClick } =
    useSpeechToText({
      onTranscript: (t) => setMotionPrompt(prev => prev ? prev + ' ' + t : t),
      language: 'en-US',
    });
  const [selStyles,     setSelStyles]     = useState<string[]>([]);
  const [duration,      setDuration]      = useState<number>(cfg.default_duration ?? 5);
  const [intensity,     setIntensity]     = useState<number>(1); // 0=Subtle, 1=Moderate, 2=Strong
  const [aspectRatio,   setAspectRatio]   = useState<string>(cfg.default_aspect ?? '16:9');
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

  const [uploading, setUploading] = useState(false);

  async function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    let finalUrl = imageUrl.trim();
    // If a file was selected, upload it to get a real HTTPS URL (base64 data URIs
    // are rejected by Pollinations and FAL video providers)
    if (imageFile) {
      setUploading(true);
      try {
        const result = await api.uploadAsset(imageFile);
        finalUrl = result.url;
      } catch (err) {
        console.error('Image upload failed:', err);
        setUploading(false);
        return;
      }
      setUploading(false);
    }
    const stylePrefix    = selStyles.length > 0 ? `[${selStyles.join(', ')}] ` : '';
    const intensityLabel = INTENSITY_LABELS[intensity];
    const intensityCue   = intensityLabel !== 'Moderate' ? ` ${intensityLabel} motion.` : '';
    const payload: GeneratePayload = {
      prompt:     stylePrefix + (motionPrompt.trim() || 'Animate this image with natural motion') + intensityCue,
      image_url:  finalUrl,
      duration,
      aspect_ratio: aspectRatio,
      style_tags: selStyles.length > 0 ? selStyles : undefined,
      extra_params: {
        intensity: intensityLabel,
      },
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

      {/* ── Step 1: Image upload ── */}
      <div>
        <div className="flex items-center gap-2 mb-2">
          <span className="w-5 h-5 rounded-full bg-cyan-600/30 text-cyan-300 text-[10px] font-bold flex items-center justify-center flex-shrink-0">1</span>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            {cfg.upload_label ?? 'Photo or image to animate'}
          </label>
        </div>

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
                <p className="text-white/30 text-xs mt-1">PNG, JPG, WebP · up to {cfg.max_file_mb ?? 20} MB</p>
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
              placeholder="https://example.com/photo.jpg"
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

      {/* ── Step 2: Motion options (revealed once image is loaded) ── */}
      {hasImage && (
        <>
          {/* Aspect Ratio */}
          <div>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Aspect Ratio</label>
            <div className="flex gap-2 flex-wrap">
              {(['16:9', '9:16', '1:1', '4:3'] as const).map((ar) => (
                <button
                  key={ar}
                  onClick={() => setAspectRatio(ar)}
                  className={cn(
                    'text-xs px-4 py-2 rounded-lg border font-semibold transition-all',
                    aspectRatio === ar
                      ? 'bg-cyan-600 text-white border-cyan-500'
                      : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                  )}
                >
                  {ar === '16:9' ? '16:9 Landscape' : ar === '9:16' ? '9:16 Portrait' : ar === '1:1' ? '1:1 Square' : '4:3 Standard'}
                </button>
              ))}
            </div>
          </div>

          {/* Duration */}
          <div>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Duration</label>
            <div className="flex gap-2 flex-wrap">
              {durations.map((d: number) => (
                <button
                  key={d}
                  onClick={() => setDuration(d)}
                  className={cn(
                    'text-xs px-4 py-2 rounded-lg border font-semibold transition-all',
                    duration === d
                      ? 'bg-cyan-600 text-white border-cyan-500'
                      : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                  )}
                >
                  {d}s
                </button>
              ))}
            </div>
          </div>

          {/* Motion intensity slider */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Motion Intensity</label>
              <span className={cn(
                'text-xs font-bold px-2 py-0.5 rounded-full',
                intensity === 0 ? 'bg-blue-500/20 text-blue-300'
                : intensity === 1 ? 'bg-cyan-500/20 text-cyan-300'
                : 'bg-orange-500/20 text-orange-300',
              )}>
                {INTENSITY_LABELS[intensity]}
              </span>
            </div>
            <input
              type="range"
              min={0}
              max={2}
              step={1}
              value={intensity}
              onChange={(e) => setIntensity(Number(e.target.value))}
              className="w-full h-1.5 rounded-full appearance-none cursor-pointer
                         bg-gradient-to-r from-blue-600 via-cyan-500 to-orange-500
                         [&::-webkit-slider-thumb]:appearance-none
                         [&::-webkit-slider-thumb]:w-4
                         [&::-webkit-slider-thumb]:h-4
                         [&::-webkit-slider-thumb]:rounded-full
                         [&::-webkit-slider-thumb]:bg-white
                         [&::-webkit-slider-thumb]:shadow-md"
            />
            <div className="flex justify-between mt-1">
              <span className="text-white/20 text-[9px]">Subtle</span>
              <span className="text-white/20 text-[9px]">Strong</span>
            </div>
          </div>

          {/* Motion style tags */}
          <div>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Motion Style</label>
            <div className="flex flex-wrap gap-1.5">
              {styleTags.map((s: string) => (
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

          {/* Motion description */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
                Motion description <span className="text-white/25 normal-case font-normal">(optional)</span>
              </label>
              <button
                onClick={handleMicClick}
                disabled={speechState === 'processing'}
                title={speechState === 'listening' ? 'Stop listening' : 'Speak the motion'}
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
              value={motionPrompt}
              onChange={(e) => setMotionPrompt(e.target.value)}
              placeholder={cfg.prompt_placeholder ?? 'Describe how to animate it — e.g. Camera slowly zooms in, trees sway in wind, water ripples…'}
              rows={3}
              autoFocus
              className="nexus-input resize-none w-full text-sm leading-relaxed"
            />
            {speechState === 'listening' && (
              <p className="text-[11px] text-red-300 mt-1 flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse inline-block" />
                Listening… {interimText || 'describe the motion'}
              </p>
            )}
            {speechState === 'error' && speechError && (
              <p className="text-[11px] text-red-300 mt-1">{speechError}</p>
            )}
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
            ? 'bg-gradient-to-r from-cyan-600 to-blue-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-cyan-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {uploading
          ? <><Loader2 size={15} className="animate-spin" /> Uploading image…</>
          : isLoading
            ? <><Loader2 size={15} className="animate-spin" /> Animating…</>
            : <><Sparkles size={15} /> Animate Image →</>
        }
      </button>
    </div>
  );
}
