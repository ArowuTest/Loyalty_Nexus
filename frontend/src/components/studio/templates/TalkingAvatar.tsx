'use client';

import { useState, useRef } from 'react';
import {
  Loader2, X, ImageIcon, Sparkles, AlertTriangle,
  Mic, MicOff, Volume2, Music, User,
} from 'lucide-react';
import { useSpeechToText } from '@/hooks/useSpeechToText';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

const DEFAULT_VOICES = ['Bill', 'Cherry', 'Ethan', 'Sarah'];
const DEFAULT_ASPECTS = [
  { label: 'Portrait',  value: '9:16', icon: '📱' },
  { label: 'Square',    value: '1:1',  icon: '⬜' },
  { label: 'Landscape', value: '16:9', icon: '🖥️' },
];
const SCRIPT_IDEAS = [
  'Welcome to Loyalty Nexus! Recharge your MTN line and win amazing prizes every day.',
  'Hi there! I have exciting news about your Pulse Points — let me tell you all about it.',
  'Thank you for being a valued customer. Your next reward is just one spin away!',
];

export default function TalkingAvatar({ tool, onSubmit, isLoading, userPoints, preloadImageUrl }: TemplateProps) {
  const cfg          = tool.ui_config ?? {};
  const maxChars     = cfg.max_script_chars ?? 300;
  const voices       = cfg.avatar_voices ?? DEFAULT_VOICES;
  const aspectRatios = cfg.aspect_ratios ?? DEFAULT_ASPECTS;
  const allowAudio   = cfg.allow_audio_upload ?? true;

  // ── Photo state ────────────────────────────────────────────────────────────
  const [imageUrl,  setImageUrl]  = useState(preloadImageUrl ?? '');
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [preview,   setPreview]   = useState<string | null>(preloadImageUrl ?? null);

  // ── Script / voice / audio state ───────────────────────────────────────────
  const [script,      setScript]      = useState('');
  const [voice,       setVoice]       = useState<string>(voices[0] ?? 'Bill');
  const [aspect,      setAspect]      = useState<string>(cfg.default_aspect ?? aspectRatios[0]?.value ?? '9:16');
  const [audioFile,   setAudioFile]   = useState<File | null>(null);
  const [useOwnAudio, setUseOwnAudio] = useState(false);
  const [uploading,   setUploading]   = useState(false);
  const [showInspo,   setShowInspo]   = useState(false);
  const [error,       setError]       = useState('');

  const fileRef  = useRef<HTMLInputElement>(null);
  const audioRef = useRef<HTMLInputElement>(null);

  const { speechState, speechError, interimText, handleMicClick } =
    useSpeechToText({
      onTranscript: (t) => setScript((prev) => (prev ? prev + ' ' + t : t).slice(0, maxChars)),
      language: 'en-US',
    });

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const hasImage  = !!(imageUrl.trim() || imageFile);
  const hasScript = script.trim().length >= 2;
  const hasAudio  = !!audioFile;
  // Valid when: a photo + (a script OR — if using own audio — an audio file)
  const isValid   = hasImage && (useOwnAudio ? hasAudio : hasScript);
  const charPct   = Math.min(100, (script.length / maxChars) * 100);
  const estSecs   = Math.max(3, Math.round(script.length / 15)); // ~15 chars/sec speech

  function handleFile(file: File) {
    setError('');
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
  function clearImage() { setImageFile(null); setPreview(null); setImageUrl(''); }

  async function handleSubmit() {
    if (!isValid || isLoading || !canAfford || uploading) return;
    setUploading(true);
    setError('');
    let finalImageUrl = imageUrl.trim();
    let finalAudioUrl = '';
    try {
      if (imageFile) {
        const r = await api.uploadAsset(imageFile);
        finalImageUrl = r.url;
      }
      if (useOwnAudio && audioFile) {
        const r = await api.uploadAsset(audioFile);
        finalAudioUrl = r.url;
      }
    } catch {
      setError('Upload failed — please try again.');
      setUploading(false);
      return;
    }
    setUploading(false);

    const payload: GeneratePayload = {
      prompt:       useOwnAudio ? '' : script.trim(),
      image_url:    finalImageUrl,
      voice_id:     useOwnAudio ? undefined : voice,
      aspect_ratio: aspect,
      extra_params: finalAudioUrl ? { audio_url: finalAudioUrl } : undefined,
    };
    onSubmit(payload);
  }

  const micRecording = speechState === 'listening';

  return (
    <div className="space-y-5">
      {/* ── Engine badge ── */}
      <div className="flex items-center gap-2 bg-gradient-to-r from-fuchsia-500/10 to-cyan-500/10 border border-fuchsia-500/20 rounded-xl px-3 py-2">
        <User size={12} className="text-fuchsia-400 flex-shrink-0" />
        <p className="text-fuchsia-300/70 text-[11px]">
          <span className="font-semibold text-fuchsia-300">Nexus Talking Avatar</span>
          {' '}— upload a face, type a script, get a lip-synced talking video.
        </p>
      </div>

      {cfg.generation_warning && (
        <div className="flex items-start gap-2 bg-amber-500/8 border border-amber-500/20 rounded-xl px-3 py-2.5">
          <AlertTriangle size={13} className="text-amber-400 flex-shrink-0 mt-0.5" />
          <p className="text-amber-300/75 text-xs leading-relaxed">{cfg.generation_warning}</p>
        </div>
      )}

      {/* ── 1. Photo upload ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          1. The Face <span className="text-fuchsia-400">*</span>
        </label>
        {preview ? (
          <div className="relative rounded-2xl overflow-hidden border border-white/10 group">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={preview} alt="avatar source" className="w-full max-h-72 object-contain bg-black/40" />
            <button
              onClick={clearImage}
              className="absolute top-2 right-2 w-8 h-8 rounded-lg bg-black/60 hover:bg-black/80 flex items-center justify-center text-white/80 transition-all"
            >
              <X size={16} />
            </button>
          </div>
        ) : (
          <div
            onClick={() => fileRef.current?.click()}
            onDrop={handleDrop}
            onDragOver={(e) => e.preventDefault()}
            className="border-2 border-dashed border-white/15 hover:border-fuchsia-500/40 rounded-2xl p-8 text-center cursor-pointer transition-all bg-white/[0.02]"
          >
            <ImageIcon size={28} className="mx-auto text-white/30 mb-2" />
            <p className="text-sm text-white/60 font-medium">Drop a portrait photo or click to upload</p>
            <p className="text-[11px] text-white/30 mt-1">A clear, front-facing face works best (JPG/PNG, max 20 MB)</p>
          </div>
        )}
        <input
          ref={fileRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }}
        />
      </div>

      {/* ── 2. Script or own audio ── */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            2. {useOwnAudio ? 'Your Audio' : 'The Script'} <span className="text-fuchsia-400">*</span>
          </label>
          {allowAudio && (
            <button
              onClick={() => setUseOwnAudio((v) => !v)}
              className="text-[11px] text-cyan-400/80 hover:text-cyan-300 font-medium flex items-center gap-1"
            >
              <Music size={11} /> {useOwnAudio ? 'Type a script instead' : 'Use my own audio'}
            </button>
          )}
        </div>

        {useOwnAudio ? (
          <div
            onClick={() => audioRef.current?.click()}
            className="border-2 border-dashed border-white/15 hover:border-cyan-500/40 rounded-2xl p-6 text-center cursor-pointer transition-all bg-white/[0.02]"
          >
            <Volume2 size={22} className="mx-auto text-white/30 mb-2" />
            <p className="text-sm text-white/60 font-medium">{audioFile ? audioFile.name : 'Upload an MP3 or WAV clip'}</p>
            <p className="text-[11px] text-white/30 mt-1">The avatar will lip-sync to this audio</p>
            <input
              ref={audioRef}
              type="file"
              accept="audio/*"
              className="hidden"
              onChange={(e) => { const f = e.target.files?.[0]; if (f) setAudioFile(f); }}
            />
          </div>
        ) : (
          <div className="relative">
            <textarea
              value={script}
              onChange={(e) => setScript(e.target.value.slice(0, maxChars))}
              placeholder={cfg.prompt_placeholder ?? 'Type what the avatar should say…'}
              rows={3}
              className="w-full bg-white/[0.03] border border-white/10 focus:border-fuchsia-500/40 rounded-xl px-3 py-2.5 text-sm text-white/90 placeholder:text-white/25 resize-none outline-none transition-all"
            />
            <div className="flex items-center justify-between mt-1.5">
              <button
                onClick={handleMicClick}
                className={cn(
                  'flex items-center gap-1 text-[11px] font-medium transition-colors',
                  micRecording ? 'text-red-400' : 'text-white/40 hover:text-white/70',
                )}
              >
                {micRecording ? <MicOff size={12} /> : <Mic size={12} />}
                {micRecording ? 'Listening…' : 'Dictate'}
              </button>
              <span className={cn('text-[11px] tabular-nums', charPct > 90 ? 'text-amber-400' : 'text-white/30')}>
                {script.length}/{maxChars} · ~{estSecs}s
              </span>
            </div>
            {interimText && <p className="text-[11px] text-white/30 italic mt-1">{interimText}</p>}
            {speechError && <p className="text-[11px] text-red-400/70 mt-1">{speechError}</p>}
          </div>
        )}

        {!useOwnAudio && (
          <button
            onClick={() => setShowInspo((v) => !v)}
            className="text-[11px] text-white/40 hover:text-white/70 mt-2 flex items-center gap-1"
          >
            <Sparkles size={11} /> Need ideas?
          </button>
        )}
        {showInspo && !useOwnAudio && (
          <div className="mt-2 space-y-1.5">
            {SCRIPT_IDEAS.map((s, i) => (
              <button
                key={i}
                onClick={() => { setScript(s.slice(0, maxChars)); setShowInspo(false); }}
                className="block w-full text-left text-[12px] text-white/55 hover:text-white/85 bg-white/[0.03] hover:bg-white/[0.06] rounded-lg px-3 py-2 transition-all"
              >
                {s}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* ── 3. Voice (hidden when using own audio) ── */}
      {!useOwnAudio && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">3. Voice</label>
          <div className="flex gap-2 flex-wrap">
            {voices.map((v) => (
              <button
                key={v}
                onClick={() => setVoice(v)}
                className={cn(
                  'text-[12px] px-3 py-1.5 rounded-lg border font-semibold transition-all',
                  voice === v
                    ? 'bg-fuchsia-600 text-white border-fuchsia-500'
                    : 'bg-white/[0.03] text-white/60 border-white/10 hover:border-white/25',
                )}
              >
                {v}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* ── 4. Aspect ratio ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          {useOwnAudio ? '3.' : '4.'} Format
        </label>
        <div className="flex gap-2 flex-wrap">
          {aspectRatios.map((ar) => (
            <button
              key={ar.value}
              onClick={() => setAspect(ar.value)}
              className={cn(
                'flex items-center gap-1 text-[11px] px-3 py-1.5 rounded-lg border font-semibold transition-all',
                aspect === ar.value
                  ? 'bg-cyan-600 text-white border-cyan-500'
                  : 'bg-white/[0.03] text-white/60 border-white/10 hover:border-white/25',
              )}
            >
              {ar.icon} {ar.label}
            </button>
          ))}
        </div>
      </div>

      {error && <p className="text-[12px] text-red-400/80">{error}</p>}

      {/* ── Submit ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || uploading || !canAfford}
        className={cn(
          'w-full flex items-center justify-center gap-2 rounded-xl py-3 font-bold text-sm transition-all',
          !isValid || !canAfford
            ? 'bg-white/5 text-white/30 cursor-not-allowed'
            : 'bg-gradient-to-r from-fuchsia-600 to-cyan-600 text-white hover:opacity-90 active:scale-[0.99]',
        )}
      >
        {uploading ? (
          <><Loader2 size={16} className="animate-spin" /> Uploading…</>
        ) : isLoading ? (
          <><Loader2 size={16} className="animate-spin" /> Generating avatar…</>
        ) : !canAfford ? (
          `Need ${(tool.point_cost - userPoints).toLocaleString()} more points`
        ) : (
          <><User size={16} /> Generate Talking Avatar · {tool.is_free ? 'Free' : `${tool.point_cost} pts`}</>
        )}
      </button>

      {cfg.output_hint && (
        <p className="text-[11px] text-white/30 text-center">{cfg.output_hint}</p>
      )}
    </div>
  );
}
