'use client';

import { useState } from 'react';
import { Loader2, Sparkles, Play, Clock, Mic, MicOff } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import { useSpeechToText } from '@/hooks/useSpeechToText';

const DEFAULT_VOICES = [
  { id: 'alloy',   name: 'Alloy',   tone: 'Neutral & Clear',       category: 'Conversational', gender: 'N' },
  { id: 'nova',    name: 'Nova',    tone: 'Friendly & Warm',       category: 'Social Media',   gender: 'F' },
  { id: 'echo',    name: 'Echo',    tone: 'Deep & Warm',           category: 'Narration',      gender: 'M' },
  { id: 'shimmer', name: 'Shimmer', tone: 'Soft & Soothing',       category: 'Meditation',     gender: 'F' },
  { id: 'onyx',    name: 'Onyx',    tone: 'Deep & Authoritative',  category: 'Broadcast',      gender: 'M' },
  { id: 'fable',   name: 'Fable',   tone: 'Expressive & Lively',   category: 'Storytelling',   gender: 'M' },
  { id: 'ash',     name: 'Ash',     tone: 'Gentle & Calm',         category: 'Education',      gender: 'N' },
  { id: 'ballad',  name: 'Ballad',  tone: 'Smooth & Musical',      category: 'Entertainment',  gender: 'M' },
  { id: 'coral',   name: 'Coral',   tone: 'Warm & Natural',        category: 'Podcasts',       gender: 'F' },
  { id: 'sage',    name: 'Sage',    tone: 'Clear & Professional',  category: 'Corporate',      gender: 'N' },
  { id: 'verse',   name: 'Verse',   tone: 'Dynamic & Engaging',    category: 'Advertisement',  gender: 'M' },
  { id: 'willow',  name: 'Willow',  tone: 'Soft & Thoughtful',     category: 'Audiobooks',     gender: 'F' },
  { id: 'jessica', name: 'Jessica', tone: 'Bright & Upbeat',       category: 'Characters',     gender: 'F' },
];

const DEFAULT_LANGUAGES = [
  { code: 'en', label: 'English',    flag: '🇬🇧' },
  { code: 'yo', label: 'Yoruba',     flag: '🇳🇬' },
  { code: 'ha', label: 'Hausa',      flag: '🇳🇬' },
  { code: 'ig', label: 'Igbo',       flag: '🇳🇬' },
  { code: 'fr', label: 'French',     flag: '🇫🇷' },
  { code: 'pt', label: 'Portuguese', flag: '🇵🇹' },
  { code: 'es', label: 'Spanish',    flag: '🇪🇸' },
];

const SPEED_STEPS = [
  { label: '0.75×', value: 0.75 },
  { label: '1×',    value: 1.0  },
  { label: '1.25×', value: 1.25 },
  { label: '1.5×',  value: 1.5  },
  { label: '2×',    value: 2.0  },
];

const FORMAT_OPTIONS = [
  { label: 'MP3', value: 'mp3', desc: 'Universal' },
  { label: 'WAV', value: 'wav', desc: 'Lossless'  },
];

const CAT_COLORS: Record<string, string> = {
  'Conversational': 'bg-sky-600/20 border-sky-500/40 text-sky-300',
  'Social Media':   'bg-pink-600/20 border-pink-500/40 text-pink-300',
  'Narration':      'bg-blue-600/20 border-blue-500/40 text-blue-300',
  'Meditation':     'bg-teal-600/20 border-teal-500/40 text-teal-300',
  'Broadcast':      'bg-amber-600/20 border-amber-500/40 text-amber-300',
  'Storytelling':   'bg-orange-600/20 border-orange-500/40 text-orange-300',
  'Education':      'bg-green-600/20 border-green-500/40 text-green-300',
  'Entertainment':  'bg-purple-600/20 border-purple-500/40 text-purple-300',
  'Podcasts':       'bg-rose-600/20 border-rose-500/40 text-rose-300',
  'Corporate':      'bg-slate-600/20 border-slate-500/40 text-slate-300',
  'Advertisement':  'bg-yellow-600/20 border-yellow-500/40 text-yellow-300',
  'Audiobooks':     'bg-indigo-600/20 border-indigo-500/40 text-indigo-300',
  'Characters':     'bg-fuchsia-600/20 border-fuchsia-500/40 text-fuchsia-300',
};

const AVATAR_GRADIENTS: Record<string, string> = {
  alloy: 'from-sky-600 to-blue-700', nova: 'from-pink-500 to-rose-600',
  echo: 'from-blue-600 to-indigo-700', shimmer: 'from-teal-500 to-cyan-600',
  onyx: 'from-slate-600 to-gray-700', fable: 'from-orange-500 to-amber-600',
  ash: 'from-green-600 to-emerald-700', ballad: 'from-purple-500 to-violet-600',
  coral: 'from-rose-500 to-pink-600', sage: 'from-gray-500 to-slate-600',
  verse: 'from-yellow-500 to-amber-600', willow: 'from-indigo-500 to-blue-600',
  jessica: 'from-fuchsia-500 to-pink-600',
};

interface VoiceEntry {
  id: string;
  name: string;
  tone: string;
  category: string;
  gender?: string;
}

function estimateDuration(text: string, speed: number): string {
  const words = text.trim().split(/\s+/).filter(Boolean).length;
  if (words === 0) return '—';
  const seconds = Math.round((words / 150) * 60 / speed);
  if (seconds < 60) return `~${seconds}s`;
  return `~${Math.round(seconds / 60)}m ${seconds % 60}s`;
}

export default function VoiceStudio({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg        = tool.ui_config ?? {};
  const voices     = (cfg.voices    ?? DEFAULT_VOICES) as VoiceEntry[];
  const languages  = cfg.languages  ?? DEFAULT_LANGUAGES;
  const maxChars   = cfg.max_chars  ?? 5000;
  const showLang   = cfg.show_language_selector ?? true;
  const showSpeed  = cfg.show_speed_control     ?? true;
  const showFormat = cfg.show_format_selector   ?? true;

  const [text,        setText]        = useState('');
  const [voiceId,     setVoiceId]     = useState<string>(cfg.default_voice ?? 'nova');
  const [language,    setLanguage]    = useState<string>(cfg.default_language ?? 'en');
  const [speed,       setSpeed]       = useState<number>(1.0);
  const [format,      setFormat]      = useState<string>('mp3');
  const [voiceFilter, setVoiceFilter] = useState('');

  // ── Web Speech API mic — dictate the text to narrate ─────────────────────
  const { speechState, speechError, interimText, handleMicClick, isMicBusy } =
    useSpeechToText({
      onTranscript: (t) => setText(prev => prev ? `${prev}\n\n${t}` : t),
      language: 'en-US',
    });

  const canAfford     = tool.is_free || userPoints >= tool.point_cost;
  const isValid       = text.trim().length >= 5 && text.length <= maxChars;
  const charPct       = (text.length / maxChars) * 100;
  const selectedVoice = voices.find((v) => v.id === voiceId);
  const estDuration   = estimateDuration(text, speed);

  const filteredVoices = voiceFilter
    ? voices.filter((v) =>
        v.name.toLowerCase().includes(voiceFilter.toLowerCase()) ||
        v.category.toLowerCase().includes(voiceFilter.toLowerCase()) ||
        v.tone.toLowerCase().includes(voiceFilter.toLowerCase()),
      )
    : voices;

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford || isMicBusy) return;
    const payload: GeneratePayload = {
      prompt:   text.trim(),
      voice_id: voiceId,
      language: showLang ? language : undefined,
      extra_params: {
        speed:  showSpeed  ? speed  : undefined,
        format: showFormat ? format : undefined,
      },
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Selected voice hero banner ── */}
      {selectedVoice && (
        <div className="flex items-center gap-3 p-3 rounded-xl bg-green-600/10 border border-green-500/20">
          <div className={cn(
            'w-10 h-10 rounded-full flex items-center justify-center text-sm font-bold text-white flex-shrink-0 bg-gradient-to-br',
            AVATAR_GRADIENTS[selectedVoice.id] ?? 'from-green-600 to-teal-700',
          )}>
            {selectedVoice.name[0]}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <p className="text-white font-semibold text-sm">{selectedVoice.name}</p>
              <span className={cn('text-[10px] font-semibold px-2 py-0.5 rounded-full border', CAT_COLORS[selectedVoice.category] ?? 'bg-white/10 border-white/20 text-white/50')}>
                {selectedVoice.category}
              </span>
            </div>
            <p className="text-white/40 text-[11px] truncate">{selectedVoice.tone}</p>
          </div>
          {text.trim() && (
            <div className="flex items-center gap-1 text-white/35 text-[11px] flex-shrink-0">
              <Clock size={10} />
              <span>{estDuration}</span>
            </div>
          )}
        </div>
      )}

      {/* ── Voice picker ── */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Choose Voice</label>
          <span className="text-white/25 text-[10px]">{voices.length} voices available</span>
        </div>

        {voices.length > 6 && (
          <input
            type="text"
            value={voiceFilter}
            onChange={(e) => setVoiceFilter(e.target.value)}
            placeholder="Search by name, tone, or use case…"
            className="nexus-input w-full text-xs mb-2"
          />
        )}

        <div className="max-h-56 overflow-y-auto pr-0.5 scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent">
          <div className="grid grid-cols-2 gap-1.5">
            {filteredVoices.map((v) => (
              <button
                key={v.id}
                onClick={() => setVoiceId(v.id)}
                className={cn(
                  'flex items-center gap-2.5 px-3 py-2.5 rounded-xl border text-left transition-all',
                  voiceId === v.id
                    ? 'bg-green-600/15 border-green-500/50'
                    : 'border-white/10 hover:border-white/20 hover:bg-white/[0.03]',
                )}
              >
                <div className={cn(
                  'w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0 bg-gradient-to-br',
                  voiceId === v.id
                    ? (AVATAR_GRADIENTS[v.id] ?? 'from-green-600 to-teal-700')
                    : 'from-white/10 to-white/5',
                )}>
                  <span className={voiceId === v.id ? 'text-white' : 'text-white/50'}>
                    {v.name[0]}
                  </span>
                </div>
                <div className="min-w-0">
                  <p className={cn('text-xs font-semibold truncate', voiceId === v.id ? 'text-green-200' : 'text-white/70')}>
                    {v.name}
                  </p>
                  <p className="text-[9px] text-white/30 truncate">{v.tone}</p>
                </div>
                {voiceId === v.id && (
                  <Play size={10} className="text-green-400 flex-shrink-0 ml-auto" />
                )}
              </button>
            ))}
          </div>
          {filteredVoices.length === 0 && (
            <p className="text-white/30 text-xs text-center py-4">No voices match &quot;{voiceFilter}&quot;</p>
          )}
        </div>
      </div>

      {/* ── Language ── */}
      {showLang && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Language</label>
          <div className="flex flex-wrap gap-1.5">
            {(languages as { code: string; label: string; flag?: string }[]).map((l) => (
              <button
                key={l.code}
                onClick={() => setLanguage(l.code)}
                className={cn(
                  'flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                  language === l.code
                    ? 'bg-green-600 text-white border-green-500'
                    : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                )}
              >
                {l.flag && <span className="text-sm leading-none">{l.flag}</span>}
                {l.label}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* ── Speed + Format ── */}
      <div className="grid grid-cols-2 gap-4">
        {showSpeed && (
          <div>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Speed</label>
            <div className="flex flex-wrap gap-1.5">
              {SPEED_STEPS.map((s) => (
                <button
                  key={s.value}
                  onClick={() => setSpeed(s.value)}
                  className={cn(
                    'text-xs px-2.5 py-1.5 rounded-full border font-medium transition-all',
                    speed === s.value
                      ? 'bg-green-600 text-white border-green-500'
                      : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                  )}
                >
                  {s.label}
                </button>
              ))}
            </div>
          </div>
        )}
        {showFormat && (
          <div>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Format</label>
            <div className="flex gap-1.5">
              {FORMAT_OPTIONS.map((f) => (
                <button
                  key={f.value}
                  onClick={() => setFormat(f.value)}
                  className={cn(
                    'flex-1 text-xs py-1.5 rounded-full border font-medium transition-all',
                    format === f.value
                      ? 'bg-green-600 text-white border-green-500'
                      : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                  )}
                >
                  {f.label}
                  <span className="text-[9px] opacity-60 ml-1">{f.desc}</span>
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* ── Text to narrate ── */}
      <div>
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            Text to narrate
          </label>
          <div className="flex items-center gap-2">
            {/* Web Speech API mic button */}
            <button
              type="button"
              onClick={handleMicClick}
              disabled={speechState === 'processing' || isLoading}
              title={speechState === 'listening' ? 'Stop — text will be added below' : 'Dictate the text you want narrated'}
              className={cn(
                'flex items-center gap-1 text-[10px] px-2 py-1 rounded-full border font-medium transition-all',
                speechState === 'listening'
                  ? 'bg-red-600/20 border-red-500/50 text-red-300 animate-pulse'
                  : speechState === 'processing'
                    ? 'bg-white/5 border-white/10 text-white/30 cursor-not-allowed'
                    : 'border-white/15 text-white/45 hover:border-green-500/40 hover:text-green-300',
              )}
            >
              {speechState === 'listening' ? (
                <><MicOff size={10} className="text-red-400" /> Stop</>
              ) : (
                <><Mic size={10} /> Dictate</>
              )}
            </button>
            <span className={cn('text-[11px] tabular-nums', text.length > maxChars * 0.9 ? 'text-red-400' : 'text-white/30')}>
              {text.length.toLocaleString()}/{maxChars.toLocaleString()}
            </span>
          </div>
        </div>

        {/* Mic status banners */}
        {speechState === 'listening' && (
          <div className="mb-2 flex items-center gap-2 px-3 py-2 rounded-xl bg-red-600/10 border border-red-500/20">
            <span className="w-2 h-2 rounded-full bg-red-500 animate-pulse flex-shrink-0" />
            <p className="text-red-300 text-xs flex-1">
              Listening… {interimText ? <em className="not-italic text-white/50">&ldquo;{interimText}&rdquo;</em> : 'speak clearly, then pause'}
            </p>
          </div>
        )}
        {speechState === 'error' && speechError && (
          <div className="mb-2 flex items-start gap-2 px-3 py-2 rounded-xl bg-red-600/10 border border-red-500/20">
            <MicOff size={12} className="text-red-400 flex-shrink-0 mt-0.5" />
            <p className="text-red-300 text-[11px] flex-1">{speechError}</p>
          </div>
        )}

        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={cfg.prompt_placeholder ?? 'Paste or type the text you want narrated… or tap Dictate above to speak it'}
          rows={6}
          maxLength={maxChars}
          autoFocus
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />
        {/* Character progress bar */}
        <div className="mt-1.5 h-0.5 w-full rounded-full bg-white/10 overflow-hidden">
          <div
            className={cn(
              'h-full rounded-full transition-all',
              charPct > 90 ? 'bg-red-500' : charPct > 70 ? 'bg-amber-500' : 'bg-green-500',
            )}
            style={{ width: `${Math.min(100, charPct)}%` }}
          />
        </div>
        {selectedVoice && text.trim() && (
          <p className="text-white/25 text-[11px] mt-1 flex items-center gap-1.5">
            <Mic size={9} />
            <span>Narrated by <strong className="text-white/40">{selectedVoice.name}</strong></span>
            <span className="text-white/15">·</span>
            <Clock size={9} />
            <span>{estDuration}</span>
          </p>
        )}
      </div>

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford || isMicBusy}
        className={cn(
          'w-full py-4 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford && !isMicBusy
            ? 'bg-gradient-to-r from-green-600 to-teal-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-green-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading ? (
          <><Loader2 size={15} className="animate-spin" /> Generating audio…</>
        ) : isMicBusy ? (
          <><Loader2 size={15} className="animate-spin" /> Listening…</>
        ) : (
          <><Sparkles size={15} /> Generate Voice</>
        )}
      </button>

      {!tool.is_free && (
        <p className="text-white/20 text-[11px] text-center -mt-2">
          {tool.point_cost} PulsePoints per generation · {userPoints} available
        </p>
      )}
    </div>
  );
}
