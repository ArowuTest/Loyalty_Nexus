'use client';

import { useState } from 'react';
import { Loader2, Sparkles, Play } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_VOICES = [
  { id: 'alloy',   name: 'Alloy',   tone: 'Neutral & Clear',    category: 'Conversational' },
  { id: 'nova',    name: 'Nova',    tone: 'Friendly & Warm',    category: 'Social Media' },
  { id: 'echo',    name: 'Echo',    tone: 'Deep & Warm',        category: 'Narration' },
  { id: 'shimmer', name: 'Shimmer', tone: 'Soft & Soothing',    category: 'Meditation' },
  { id: 'onyx',    name: 'Onyx',    tone: 'Deep & Authoritative', category: 'Broadcast' },
  { id: 'fable',   name: 'Fable',   tone: 'Expressive & Lively', category: 'Storytelling' },
  { id: 'ash',     name: 'Ash',     tone: 'Gentle & Calm',      category: 'Education' },
  { id: 'ballad',  name: 'Ballad',  tone: 'Smooth & Musical',   category: 'Entertainment' },
  { id: 'coral',   name: 'Coral',   tone: 'Warm & Natural',     category: 'Podcasts' },
  { id: 'sage',    name: 'Sage',    tone: 'Clear & Professional', category: 'Corporate' },
  { id: 'verse',   name: 'Verse',   tone: 'Dynamic & Engaging', category: 'Advertisement' },
  { id: 'willow',  name: 'Willow',  tone: 'Soft & Thoughtful',  category: 'Audiobooks' },
  { id: 'jessica', name: 'Jessica', tone: 'Bright & Upbeat',    category: 'Characters' },
];

const DEFAULT_LANGUAGES = [
  { code: 'en', label: 'English' },
  { code: 'yo', label: 'Yoruba' },
  { code: 'ha', label: 'Hausa' },
  { code: 'ig', label: 'Igbo' },
  { code: 'fr', label: 'French' },
  { code: 'pt', label: 'Portuguese' },
  { code: 'es', label: 'Spanish' },
];

const SPEED_STEPS = [
  { label: '0.75×', value: 0.75 },
  { label: '1×',    value: 1.0 },
  { label: '1.25×', value: 1.25 },
  { label: '1.5×',  value: 1.5 },
  { label: '2×',    value: 2.0 },
];

const FORMAT_OPTIONS = [
  { label: 'MP3',  value: 'mp3',  desc: 'Universal' },
  { label: 'WAV',  value: 'wav',  desc: 'Lossless' },
];

// Category colour map
const CAT_COLORS: Record<string, string> = {
  'Conversational': 'text-sky-300',
  'Social Media':   'text-pink-300',
  'Narration':      'text-blue-300',
  'Meditation':     'text-teal-300',
  'Broadcast':      'text-amber-300',
  'Storytelling':   'text-orange-300',
  'Education':      'text-green-300',
  'Entertainment':  'text-purple-300',
  'Podcasts':       'text-rose-300',
  'Corporate':      'text-gray-300',
  'Advertisement':  'text-yellow-300',
  'Audiobooks':     'text-indigo-300',
  'Characters':     'text-fuchsia-300',
};

interface VoiceEntry {
  id: string;
  name: string;
  tone: string;
  category: string;
}

export default function VoiceStudio({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg      = tool.ui_config ?? {};
  const voices   = (cfg.voices    ?? DEFAULT_VOICES) as VoiceEntry[];
  const languages = cfg.languages ?? DEFAULT_LANGUAGES;
  const maxChars  = cfg.max_chars  ?? 5000;
  const showLang  = cfg.show_language_selector ?? true;
  const showSpeed = cfg.show_speed_control ?? true;
  const showFormat = cfg.show_format_selector ?? true;

  const [text,     setText]     = useState('');
  const [voiceId,  setVoiceId]  = useState<string>(cfg.default_voice ?? 'nova');
  const [language, setLanguage] = useState<string>(cfg.default_language ?? 'en');
  const [speed,    setSpeed]    = useState<number>(1.0);
  const [format,   setFormat]   = useState<string>('mp3');
  const [voiceFilter, setVoiceFilter] = useState('');

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = text.trim().length >= 5 && text.length <= maxChars;
  const charPct   = (text.length / maxChars) * 100;

  const filteredVoices = voiceFilter
    ? voices.filter((v) =>
        v.name.toLowerCase().includes(voiceFilter.toLowerCase()) ||
        v.category.toLowerCase().includes(voiceFilter.toLowerCase()) ||
        v.tone.toLowerCase().includes(voiceFilter.toLowerCase()),
      )
    : voices;

  const selectedVoice = voices.find((v) => v.id === voiceId);

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const payload: GeneratePayload = {
      prompt:   text.trim(),
      voice_id: voiceId,
      language: showLang ? language : undefined,
      extra_params: {
        speed:  showSpeed ? speed : undefined,
        format: showFormat ? format : undefined,
      },
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Voice picker ── */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Voice</label>
          {selectedVoice && (
            <span className={cn('text-[10px] font-semibold', CAT_COLORS[selectedVoice.category] ?? 'text-white/40')}>
              {selectedVoice.category}
            </span>
          )}
        </div>

        {/* Search filter */}
        {voices.length > 6 && (
          <input
            type="text"
            value={voiceFilter}
            onChange={(e) => setVoiceFilter(e.target.value)}
            placeholder="Search voices — name, tone, or use case…"
            className="nexus-input w-full text-xs mb-2"
          />
        )}

        {/* Scrollable voice grid */}
        <div className="max-h-52 overflow-y-auto pr-0.5 space-y-1 scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent">
          <div className="grid grid-cols-2 gap-1.5">
            {filteredVoices.map((v) => (
              <button
                key={v.id}
                onClick={() => setVoiceId(v.id)}
                className={cn(
                  'flex items-center gap-2 px-3 py-2.5 rounded-xl border text-left transition-all',
                  voiceId === v.id
                    ? 'bg-green-600/20 border-green-500/60'
                    : 'border-white/10 hover:border-white/20 hover:bg-white/3',
                )}
              >
                <div className={cn(
                  'w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0',
                  voiceId === v.id
                    ? 'bg-green-600 text-white'
                    : 'bg-white/10 text-white/50',
                )}>
                  {v.name[0]}
                </div>
                <div className="min-w-0">
                  <p className={cn('text-xs font-semibold truncate', voiceId === v.id ? 'text-green-200' : 'text-white/70')}>
                    {v.name}
                  </p>
                  <p className="text-[9px] text-white/30 truncate">{v.tone}</p>
                </div>
              </button>
            ))}
          </div>
          {filteredVoices.length === 0 && (
            <p className="text-white/30 text-xs text-center py-4">No voices match "{voiceFilter}"</p>
          )}
        </div>
      </div>

      {/* ── Language + Speed + Format row ── */}
      <div className="grid grid-cols-1 gap-4">
        {/* Language */}
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
                      ? 'bg-green-600 text-white border-green-500'
                      : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                  )}
                >
                  {l.label}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Speed + Format side by side */}
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
                      'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
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
                      'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
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
      </div>

      {/* ── Text to narrate ── */}
      <div>
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Text to narrate</label>
          <span className={cn('text-[11px] tabular-nums', text.length > maxChars * 0.9 ? 'text-red-400' : 'text-white/30')}>
            {text.length.toLocaleString()}/{maxChars.toLocaleString()}
          </span>
        </div>
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={cfg.prompt_placeholder ?? 'Paste or type the text you want narrated…'}
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
        {selectedVoice && (
          <p className="text-white/25 text-[11px] mt-1 flex items-center gap-1">
            <Play size={9} />
            Will be narrated by <strong className="text-white/40">{selectedVoice.name}</strong> · {selectedVoice.tone}
          </p>
        )}
      </div>

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-green-600 to-teal-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-green-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Generating audio…</>
          : <><Sparkles size={15} /> Generate Voice →</>
        }
      </button>
    </div>
  );
}
