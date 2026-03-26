'use client';

import { useState, useRef } from 'react';
import { Loader2, Upload, X, Mic, FileAudio, Sparkles } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_LANGUAGES = [
  { code: 'en',    label: 'English' },
  { code: 'fr',    label: 'French' },
  { code: 'es',    label: 'Spanish' },
  { code: 'pt',    label: 'Portuguese' },
  { code: 'sw',    label: 'Swahili' },
  { code: 'yo',    label: 'Yoruba' },
  { code: 'ha',    label: 'Hausa' },
  { code: 'ar',    label: 'Arabic' },
  { code: 'auto',  label: 'Auto-detect' },
];

export default function Transcribe({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg       = tool.ui_config ?? {};
  const languages = cfg.languages ?? DEFAULT_LANGUAGES;
  const showLang  = cfg.show_language_selector ?? true;
  const maxMins   = cfg.max_duration_mins ?? 60;

  const [audioUrl,  setAudioUrl]  = useState('');
  const [audioFile, setAudioFile] = useState<File | null>(null);
  const [language,  setLanguage]  = useState(cfg.default_language ?? 'auto');
  const fileRef = useRef<HTMLInputElement>(null);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const hasAudio  = audioUrl.trim() || audioFile;
  const isValid   = !!hasAudio;

  function handleFile(file: File) {
    setAudioFile(file);
    setAudioUrl('');
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file && (file.type.startsWith('audio/') || file.type.startsWith('video/'))) handleFile(file);
  }

  function clearAudio() {
    setAudioFile(null);
    setAudioUrl('');
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const finalUrl = audioFile ? `file:${audioFile.name}` : audioUrl.trim();
    const payload: GeneratePayload = {
      prompt:   finalUrl,
      language: showLang ? language : undefined,
      extra_params: audioFile ? { file_name: audioFile.name, file_type: audioFile.type } : undefined,
    };
    onSubmit(payload);
  }

  const acceptTypes = (cfg.upload_accept ?? ['audio/mp3', 'audio/wav', 'audio/m4a', 'audio/ogg', 'video/mp4']).join(',');

  return (
    <div className="space-y-5">

      {/* ── Language selector ── */}
      {showLang && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Language</label>
          <div className="flex flex-wrap gap-1.5">
            {languages.map((l) => (
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
                <p className="text-white/65 text-sm font-medium">Drop an audio file here or click to browse</p>
                <p className="text-white/30 text-xs mt-1">MP3, WAV, M4A, OGG, MP4 · up to {maxMins} min</p>
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
              placeholder="https://example.com/audio.mp3"
              className="nexus-input w-full text-sm mt-1"
            />
          </>
        ) : (
          /* File selected */
          <div className="flex items-center gap-3 bg-orange-500/8 border border-orange-500/20 rounded-xl px-4 py-3">
            <div className="p-2 rounded-lg bg-orange-500/15">
              <FileAudio size={18} className="text-orange-400" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-white/80 text-sm font-medium truncate">{audioFile?.name ?? 'URL audio'}</p>
              {audioFile && (
                <p className="text-white/35 text-xs">{(audioFile.size / (1024 * 1024)).toFixed(1)} MB</p>
              )}
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

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-orange-600 to-amber-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-orange-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Transcribing…</>
          : <><Sparkles size={15} /> Transcribe Audio →</>
        }
      </button>

      {/* Output hint */}
      {cfg.output_hint && (
        <p className="text-white/30 text-xs text-center">{cfg.output_hint}</p>
      )}
    </div>
  );
}
