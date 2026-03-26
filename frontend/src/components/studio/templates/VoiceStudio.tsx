'use client';

import { useState } from 'react';
import { Loader2, Sparkles } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_VOICES = [
  { id: 'alloy',   name: 'Alloy',   tone: 'Neutral',   category: 'Versatile' },
  { id: 'nova',    name: 'Nova',    tone: 'Warm',       category: 'Female' },
  { id: 'echo',    name: 'Echo',    tone: 'Crisp',      category: 'Male' },
  { id: 'shimmer', name: 'Shimmer', tone: 'Soft',       category: 'Female' },
  { id: 'onyx',    name: 'Onyx',    tone: 'Deep',       category: 'Male' },
  { id: 'fable',   name: 'Fable',   tone: 'Expressive', category: 'Storyteller' },
];

const DEFAULT_LANGUAGES = [
  { code: 'en', label: 'English' },
  { code: 'fr', label: 'French' },
  { code: 'es', label: 'Spanish' },
  { code: 'de', label: 'German' },
  { code: 'pt', label: 'Portuguese' },
  { code: 'sw', label: 'Swahili' },
  { code: 'yo', label: 'Yoruba' },
  { code: 'ha', label: 'Hausa' },
];

export default function VoiceStudio({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg = tool.ui_config ?? {};
  const voices    = cfg.voices    ?? DEFAULT_VOICES;
  const languages = cfg.languages ?? DEFAULT_LANGUAGES;
  const maxChars  = cfg.max_chars ?? 3000;
  const showLang  = cfg.show_language_selector ?? true;

  const [text,      setText]      = useState('');
  const [voiceId,   setVoiceId]   = useState(cfg.default_voice ?? 'nova');
  const [language,  setLanguage]  = useState(cfg.default_language ?? 'en');

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = text.trim().length >= 5 && text.length <= maxChars;

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const payload: GeneratePayload = {
      prompt:   text.trim(),
      voice_id: voiceId,
      language: showLang ? language : undefined,
    };
    onSubmit(payload);
  }

  const charPct = (text.length / maxChars) * 100;

  return (
    <div className="space-y-5">

      {/* ── Voice picker ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Voice</label>
        <div className="grid grid-cols-2 gap-2">
          {voices.map((v) => (
            <button
              key={v.id}
              onClick={() => setVoiceId(v.id)}
              className={cn(
                'flex items-center gap-2.5 px-3 py-2.5 rounded-xl border text-left transition-all',
                voiceId === v.id
                  ? 'bg-green-600/20 border-green-500/60'
                  : 'border-white/10 hover:border-white/20 hover:bg-white/3',
              )}
            >
              <div className={cn(
                'w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold flex-shrink-0',
                voiceId === v.id ? 'bg-green-600 text-white' : 'bg-white/10 text-white/50',
              )}>
                {v.name[0]}
              </div>
              <div className="min-w-0">
                <p className={cn('text-xs font-semibold truncate', voiceId === v.id ? 'text-green-200' : 'text-white/70')}>{v.name}</p>
                <p className="text-[10px] text-white/35 truncate">{v.tone} · {v.category}</p>
              </div>
            </button>
          ))}
        </div>
      </div>

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

      {/* ── Text to narrate ── */}
      <div>
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Text to narrate</label>
          <span className={cn('text-[11px]', text.length > maxChars * 0.9 ? 'text-red-400' : 'text-white/30')}>
            {text.length}/{maxChars}
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
        {/* Character bar */}
        <div className="mt-1.5 h-0.5 w-full rounded-full bg-white/10 overflow-hidden">
          <div
            className={cn(
              'h-full rounded-full transition-all',
              charPct > 90 ? 'bg-red-500' : charPct > 70 ? 'bg-amber-500' : 'bg-green-500',
            )}
            style={{ width: `${Math.min(100, charPct)}%` }}
          />
        </div>
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
