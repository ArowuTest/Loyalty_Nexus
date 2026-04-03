'use client';

import { useState, useRef } from 'react';
import { Loader2, Upload, X, Mic, FileAudio, Sparkles, Users, CheckCircle2 } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

const DEFAULT_LANGUAGES = [
  { code: 'auto', label: 'Auto-detect' },
  { code: 'en',   label: 'English' },
  { code: 'yo',   label: 'Yoruba' },
  { code: 'ha',   label: 'Hausa' },
  { code: 'ig',   label: 'Igbo' },
  { code: 'fr',   label: 'French' },
  { code: 'pcm',  label: 'Pidgin' },
];

const OUTPUT_FORMATS = [
  { value: 'plain',       label: 'Plain text',   desc: 'Clean transcript, no timestamps' },
  { value: 'timestamped', label: 'Timestamped',  desc: 'With time markers per sentence' },
  { value: 'srt',         label: 'SRT Subtitles', desc: 'Ready to use as subtitles' },
];

function formatFileSize(bytes: number): string {
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function Transcribe({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg        = tool.ui_config ?? {};
  const languages  = cfg.languages  ?? DEFAULT_LANGUAGES;
  const showLang   = cfg.show_language_selector ?? true;
  const showSpeakers = cfg.show_speaker_labels  ?? true;
  const showFormat = cfg.show_output_format     ?? true;
  const maxMins    = cfg.max_duration_mins      ?? 60;

  const [audioUrl,     setAudioUrl]     = useState('');
  const [audioFile,    setAudioFile]    = useState<File | null>(null);
  const [uploadedUrl,  setUploadedUrl]  = useState<string | null>(null);
  const [isUploading,  setIsUploading]  = useState(false);
  const [uploadError,  setUploadError]  = useState<string | null>(null);
  const [language,     setLanguage]     = useState<string>(cfg.default_language ?? 'auto');
  const [speakLabels,  setSpeakLabels]  = useState<boolean>(true);
  const [outFormat,    setOutFormat]    = useState<string>('plain');
  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const hasAudio  = uploadedUrl || audioUrl.trim() || audioFile;
  const isValid   = !!hasAudio && !isUploading;
  const isBusy    = isLoading || isUploading;

  async function handleFile(file: File) {
    setAudioFile(file);
    setUploadedUrl(null);
    setUploadError(null);
    setAudioUrl('');
    // Upload to CDN so backend gets a valid HTTPS URL
    setIsUploading(true);
    try {
      const result = await api.uploadAsset(file);
      setUploadedUrl(result.url);
    } catch (err) {
      setUploadError('Upload failed — please try again or paste a URL instead.');
      console.error('[Transcribe] upload error:', err);
    } finally {
      setIsUploading(false);
    }
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file && (file.type.startsWith('audio/') || file.type.startsWith('video/'))) handleFile(file);
  }

  function clearAudio() {
    setAudioFile(null);
    setAudioUrl('');
    setUploadedUrl(null);
    setUploadError(null);
  }

  function handleSubmit() {
    if (!isValid || isBusy || !canAfford) return;
    // Use the CDN URL if we uploaded a file, otherwise use the pasted URL
    const finalUrl = uploadedUrl ?? audioUrl.trim();
    const payload: GeneratePayload = {
      prompt:   finalUrl,
      language: showLang ? language : undefined,
      extra_params: {
        speaker_labels: showSpeakers ? speakLabels : false,
        output_format:  outFormat,
      },
    };
    onSubmit(payload);
  }

  const acceptTypes = (cfg.upload_accept ?? [
    'audio/mp3', 'audio/mpeg', 'audio/wav', 'audio/m4a',
    'audio/ogg', 'audio/flac', 'video/mp4',
  ]).join(',');

  return (
    <div className="space-y-5">

      {/* ── Language selector (first — set BEFORE uploading) ── */}
      {showLang && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Language</label>
          <div className="flex flex-wrap gap-1.5">
            {(languages as { code: string; label: string }[]).map((l) => (
              <button
                key={l.code}
                onClick={() => setLanguage(l.code)}
                className={cn(
                  'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                  language === l.code
                    ? 'bg-orange-600 text-white border-orange-500'
                    : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                )}
              >
                {l.label}
              </button>
            ))}
          </div>
          <p className="text-white/25 text-[11px] mt-1">
            Select <strong className="text-white/40">Auto-detect</strong> if unsure — or pick the language for better accuracy
          </p>
        </div>
      )}

      {/* ── Output format ── */}
      {showFormat && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Output Format</label>
          <div className="grid grid-cols-3 gap-2">
            {OUTPUT_FORMATS.map((f) => (
              <button
                key={f.value}
                onClick={() => setOutFormat(f.value)}
                className={cn(
                  'flex flex-col gap-0.5 p-2.5 rounded-xl border text-left transition-all',
                  outFormat === f.value
                    ? 'bg-orange-600/20 border-orange-500/60 text-orange-200'
                    : 'border-white/10 text-white/45 hover:border-white/25 hover:text-white/70',
                )}
              >
                <span className="text-[11px] font-semibold">{f.label}</span>
                <span className="text-[9px] opacity-60">{f.desc}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* ── Speaker labels toggle ── */}
      {showSpeakers && (
        <div className="flex items-center justify-between bg-white/3 border border-white/8 rounded-xl px-4 py-3">
          <div className="flex items-center gap-2.5">
            <Users size={14} className="text-white/40" />
            <div>
              <p className="text-white/70 text-xs font-semibold">Speaker labels</p>
              <p className="text-white/30 text-[10px]">Identify who is talking (Speaker A, Speaker B…)</p>
            </div>
          </div>
          <button
            onClick={() => setSpeakLabels((v) => !v)}
            className={cn(
              'w-10 h-5.5 rounded-full transition-all relative flex-shrink-0',
              speakLabels ? 'bg-orange-600' : 'bg-white/15',
            )}
          >
            <span className={cn(
              'absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-all',
              speakLabels ? 'left-5.5' : 'left-0.5',
            )} />
          </button>
        </div>
      )}

      {/* ── Audio upload ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
          {cfg.upload_label ?? 'Audio File'}
        </label>

        {!audioFile && !audioUrl ? (
          <>
            <div
              onDrop={handleDrop}
              onDragOver={(e) => e.preventDefault()}
              onClick={() => fileRef.current?.click()}
              className="border-2 border-dashed border-white/15 rounded-xl p-8 flex flex-col items-center gap-3 cursor-pointer
                         hover:border-orange-500/40 hover:bg-orange-500/5 transition-all text-center"
            >
              <div className="p-3 rounded-full bg-white/5">
                <Mic size={22} className="text-white/40" />
              </div>
              <div>
                <p className="text-white/65 text-sm font-medium">Drop your audio here or click to browse</p>
                <p className="text-white/30 text-xs mt-1">MP3, WAV, M4A, FLAC, OGG · up to {maxMins} min</p>
              </div>
            </div>
            <input
              ref={fileRef}
              type="file"
              accept={acceptTypes}
              className="hidden"
              onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }}
            />
            <p className="text-white/30 text-[11px] text-center mt-2">— or paste a URL —</p>
            <input
              type="url"
              value={audioUrl}
              onChange={(e) => setAudioUrl(e.target.value)}
              placeholder="https://example.com/recording.mp3"
              className="nexus-input w-full text-sm mt-1"
            />
          </>
        ) : (
          /* File selected card */
          <div className="flex items-center gap-3 bg-orange-500/8 border border-orange-500/20 rounded-xl px-4 py-3">
            <div className="p-2 rounded-lg bg-orange-500/15">
              <FileAudio size={18} className="text-orange-400" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-white/80 text-sm font-medium truncate">{audioFile?.name ?? 'URL audio'}</p>
              <div className="flex items-center gap-2 mt-0.5">
                {audioFile && (
                  <p className="text-white/35 text-xs">{formatFileSize(audioFile.size)}</p>
                )}
                {audioFile && (
                  <p className="text-white/25 text-xs">· {audioFile.type.split('/')[1]?.toUpperCase()}</p>
                )}
              </div>
            </div>
            <button
              onClick={clearAudio}
              className="p-1.5 text-white/40 hover:text-white/80 transition-colors flex-shrink-0"
            >
              <X size={15} />
            </button>
          </div>
        )}
      </div>

      {/* ── Upload status banners ── */}
      {isUploading && (
        <div className="flex items-center gap-2 bg-orange-500/10 border border-orange-500/20 rounded-xl px-3 py-2">
          <Loader2 size={13} className="text-orange-400 animate-spin flex-shrink-0" />
          <p className="text-orange-300/80 text-xs">Uploading audio…</p>
        </div>
      )}
      {uploadedUrl && !isUploading && (
        <div className="flex items-center gap-2 bg-green-500/10 border border-green-500/20 rounded-xl px-3 py-2">
          <CheckCircle2 size={13} className="text-green-400 flex-shrink-0" />
          <p className="text-green-300/80 text-xs">Audio ready for transcription</p>
        </div>
      )}
      {uploadError && (
        <div className="bg-red-500/10 border border-red-500/20 rounded-xl px-3 py-2">
          <p className="text-red-300/80 text-xs">{uploadError}</p>
        </div>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isBusy || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isBusy && canAfford
            ? 'bg-gradient-to-r from-orange-600 to-amber-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-orange-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isUploading
          ? <><Loader2 size={15} className="animate-spin" /> Uploading audio…</>
          : isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Transcribing…</>
          : <><Sparkles size={15} /> Transcribe Audio →</>
        }
      </button>

      {/* Output hint */}
      {cfg.output_hint && (
        <p className="text-white/30 text-xs text-center leading-relaxed">{cfg.output_hint}</p>
      )}
    </div>
  );
}
