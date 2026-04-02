'use client';

import { useState } from 'react';
import { Loader2, Music, ChevronDown, ChevronUp, Sparkles, Shuffle } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_GENRE_TAGS = [
  'Afrobeats', 'Amapiano', 'Gospel', 'Highlife', 'R&B', 'Hip-Hop',
  'Pop', 'Jazz', 'Classical', 'EDM', 'Reggae', 'Funk',
];
const DEFAULT_DURATIONS  = [15, 30, 60, 120, 180, 300];
const ENERGY_LABELS      = ['Chill', 'Relaxed', 'Balanced', 'Upbeat', 'Energetic'] as const;
const BPM_PRESETS        = [60, 80, 90, 100, 110, 120, 128, 140, 160] as const;

// Mood board colour swatches — each mood maps to a colour theme
const MOODS = [
  { label: 'Happy',     emoji: '😊', color: 'bg-yellow-500/20 border-yellow-500/40 text-yellow-200' },
  { label: 'Romantic',  emoji: '💕', color: 'bg-pink-500/20 border-pink-500/40 text-pink-200' },
  { label: 'Melancholy',emoji: '🌧️', color: 'bg-blue-500/20 border-blue-500/40 text-blue-200' },
  { label: 'Hype',      emoji: '🔥', color: 'bg-orange-500/20 border-orange-500/40 text-orange-200' },
  { label: 'Spiritual', emoji: '✨', color: 'bg-purple-500/20 border-purple-500/40 text-purple-200' },
  { label: 'Peaceful',  emoji: '🌿', color: 'bg-teal-500/20 border-teal-500/40 text-teal-200' },
];

const PROMPT_INSPIRATIONS = [
  'Upbeat Afrobeats love song, female vocals, summer vibes, catchy chorus hook',
  'Calm lo-fi piano background for studying, minimal percussion, relaxing',
  'Epic gospel choir, powerful crescendo, inspirational and uplifting',
  'Amapiano house track, deep log drum, smooth saxophone, late night energy',
  'Highlife guitar melody, nostalgic, storytelling vocals, warm and joyful',
  'Cinematic orchestral score, tension building, dramatic strings and brass',
];

export default function MusicComposer({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg        = tool.ui_config ?? {};
  const genreTags  = cfg.genre_tags        ?? DEFAULT_GENRE_TAGS;
  const durations  = cfg.duration_options  ?? DEFAULT_DURATIONS;
  const showVocals = cfg.show_vocals_toggle ?? true;
  const showLyrics = cfg.show_lyrics_box    ?? true;
  const maxDur     = cfg.max_duration       ?? 300;
  const showBpm    = cfg.show_bpm           ?? true;
  const showEnergy = cfg.show_energy        ?? true;

  const [selectedTags,  setSelectedTags]  = useState<string[]>([]);
  const [selectedMood,  setSelectedMood]  = useState<string | null>(null);
  const [prompt,        setPrompt]        = useState('');
  const [duration,      setDuration]      = useState<number>(cfg.default_duration ?? 30);
  const [vocals,        setVocals]        = useState<boolean>(cfg.default_vocals ?? true);
  const [lyrics,        setLyrics]        = useState('');
  const [showLyricsBox, setShowLyricsBox] = useState(false);
  const [showInspo,     setShowInspo]     = useState(false);
  const [bpm,           setBpm]           = useState<number | null>(null);
  const [energy,        setEnergy]        = useState<number>(2);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = prompt.trim().length >= 3;

  const filteredDurations = durations.filter((d: number) => d <= maxDur);

  function toggleTag(tag: string) {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : prev.length < 3 ? [...prev, tag] : prev,
    );
  }

  function fmtDuration(s: number) {
    return s < 60 ? `${s}s` : `${s / 60}m`;
  }

  function surpriseMe() {
    const random = PROMPT_INSPIRATIONS[Math.floor(Math.random() * PROMPT_INSPIRATIONS.length)];
    setPrompt(random);
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const tagPrefix    = selectedTags.length > 0 ? `[${selectedTags.join(', ')}] ` : '';
    const moodCue      = selectedMood ? ` ${selectedMood} mood.` : '';
    const energyLabel  = ENERGY_LABELS[energy];
    const energyCue    = energyLabel !== 'Balanced' ? ` ${energyLabel} energy.` : '';
    const bpmCue       = bpm !== null ? ` ${bpm} BPM.` : '';
    const vocalsCue    = showVocals ? (vocals ? ' With vocals.' : ' Instrumental only.') : '';

    const payload: GeneratePayload = {
      prompt:     tagPrefix + prompt.trim() + moodCue + energyCue + bpmCue + vocalsCue,
      duration,
      vocals:     showVocals ? vocals : undefined,
      lyrics:     showLyrics && lyrics.trim() ? lyrics.trim() : undefined,
      style_tags: selectedTags.length > 0 ? selectedTags : undefined,
      extra_params: {
        bpm:    bpm ?? 'auto',
        energy: ENERGY_LABELS[energy],
        mood:   selectedMood ?? undefined,
      },
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Animated waveform loading state ── */}
      {isLoading && (
        <div className="flex items-center justify-center gap-1 py-3">
          {Array.from({ length: 20 }).map((_, i) => (
            <div
              key={i}
              className="w-1 rounded-full bg-amber-500"
              style={{
                height: `${8 + Math.sin(i * 0.8) * 12 + 8}px`,
                animation: `pulse ${0.6 + (i % 5) * 0.1}s ease-in-out infinite alternate`,
                animationDelay: `${i * 0.05}s`,
              }}
            />
          ))}
        </div>
      )}

      {/* ── Genre tags (max 3) ── */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Genre</label>
          <span className="text-white/25 text-[10px]">{selectedTags.length}/3 selected</span>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {(genreTags as string[]).map((tag) => (
            <button
              key={tag}
              onClick={() => toggleTag(tag)}
              disabled={!selectedTags.includes(tag) && selectedTags.length >= 3}
              className={cn(
                'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                selectedTags.includes(tag)
                  ? 'bg-amber-500 text-black border-amber-500 shadow-sm shadow-amber-900/30'
                  : selectedTags.length >= 3
                    ? 'text-white/20 border-white/8 cursor-not-allowed'
                    : 'text-white/55 border-white/15 hover:border-amber-500/40 hover:text-white/80',
              )}
            >
              {tag}
            </button>
          ))}
        </div>
      </div>

      {/* ── Mood board ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Mood</label>
        <div className="grid grid-cols-3 gap-2">
          {MOODS.map((mood) => (
            <button
              key={mood.label}
              onClick={() => setSelectedMood(selectedMood === mood.label ? null : mood.label)}
              className={cn(
                'flex items-center gap-2 px-3 py-2 rounded-xl border text-xs font-medium transition-all',
                selectedMood === mood.label
                  ? mood.color
                  : 'border-white/10 text-white/45 hover:border-white/20 hover:bg-white/[0.03]',
              )}
            >
              <span className="text-base leading-none">{mood.emoji}</span>
              <span>{mood.label}</span>
            </button>
          ))}
        </div>
      </div>

      {/* ── Vocals / Instrumental toggle ── */}
      {showVocals && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Style</label>
          <div className="flex rounded-xl overflow-hidden border border-white/10 w-full">
            <button
              onClick={() => setVocals(true)}
              className={cn(
                'flex-1 flex items-center justify-center gap-2 py-2.5 text-xs font-semibold transition-all',
                vocals ? 'bg-amber-500 text-black' : 'text-white/55 hover:text-white/80',
              )}
            >
              🎤 With Vocals
            </button>
            <button
              onClick={() => setVocals(false)}
              className={cn(
                'flex-1 flex items-center justify-center gap-2 py-2.5 text-xs font-semibold transition-all',
                !vocals ? 'bg-amber-500 text-black' : 'text-white/55 hover:text-white/80',
              )}
            >
              🎹 Instrumental
            </button>
          </div>
        </div>
      )}

      {/* ── Prompt ── */}
      <div>
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            Describe your music
          </label>
          <button
            onClick={surpriseMe}
            className="flex items-center gap-1 text-white/30 hover:text-amber-400 transition-colors text-[11px] font-medium"
          >
            <Shuffle size={11} /> Surprise me
          </button>
        </div>
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder={
            cfg.prompt_placeholder ??
            (vocals
              ? 'e.g. Upbeat Afrobeats love song, female vocals, summer vibes, catchy chorus hook…'
              : 'e.g. Calm lo-fi piano background for studying, minimal percussion, relaxing…')
          }
          rows={3}
          autoFocus
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />

        {/* Prompt inspirations */}
        <button
          onClick={() => setShowInspo((v) => !v)}
          className="flex items-center gap-1 text-white/25 hover:text-white/50 transition-colors text-[11px] mt-1.5"
        >
          <Music size={11} />
          {showInspo ? 'Hide' : 'Show'} prompt ideas
          {showInspo ? <ChevronUp size={11} /> : <ChevronDown size={11} />}
        </button>
        {showInspo && (
          <div className="mt-2 grid grid-cols-1 gap-1.5">
            {PROMPT_INSPIRATIONS.map((inspo) => (
              <button
                key={inspo}
                onClick={() => { setPrompt(inspo); setShowInspo(false); }}
                className="text-left text-xs text-white/40 hover:text-white/70 hover:bg-white/[0.04] px-3 py-2 rounded-lg border border-white/[0.06] hover:border-white/15 transition-all truncate"
              >
                {inspo}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* ── Energy + BPM row ── */}
      {(showEnergy || showBpm) && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">

          {showEnergy && (
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Energy</label>
                <span className={cn(
                  'text-xs font-bold px-2 py-0.5 rounded-full',
                  energy === 0 ? 'bg-blue-500/20 text-blue-300'
                  : energy === 1 ? 'bg-teal-500/20 text-teal-300'
                  : energy === 2 ? 'bg-amber-500/20 text-amber-300'
                  : energy === 3 ? 'bg-orange-500/20 text-orange-300'
                  : 'bg-red-500/20 text-red-300',
                )}>
                  {ENERGY_LABELS[energy]}
                </span>
              </div>
              <input
                type="range" min={0} max={4} step={1} value={energy}
                onChange={(e) => setEnergy(Number(e.target.value))}
                className="w-full h-1.5 rounded-full appearance-none cursor-pointer
                           bg-gradient-to-r from-blue-600 via-amber-500 to-red-500
                           [&::-webkit-slider-thumb]:appearance-none
                           [&::-webkit-slider-thumb]:w-4 [&::-webkit-slider-thumb]:h-4
                           [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-white
                           [&::-webkit-slider-thumb]:shadow-md"
              />
              <div className="flex justify-between mt-1">
                <span className="text-white/20 text-[9px]">Chill</span>
                <span className="text-white/20 text-[9px]">Energetic</span>
              </div>
            </div>
          )}

          {showBpm && (
            <div>
              <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">BPM</label>
              <div className="flex flex-wrap gap-1.5">
                <button
                  onClick={() => setBpm(null)}
                  className={cn(
                    'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                    bpm === null
                      ? 'bg-amber-500 text-black border-amber-500'
                      : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                  )}
                >
                  Auto
                </button>
                {BPM_PRESETS.map((b) => (
                  <button
                    key={b}
                    onClick={() => setBpm(b)}
                    className={cn(
                      'text-xs px-2.5 py-1.5 rounded-full border font-medium transition-all',
                      bpm === b
                        ? 'bg-amber-500 text-black border-amber-500'
                        : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                    )}
                  >
                    {b}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Duration ── */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Duration</label>
          <span className="text-white/35 text-[11px] font-mono">{fmtDuration(duration)}</span>
        </div>
        <div className="flex gap-2 flex-wrap">
          {filteredDurations.map((d: number) => (
            <button
              key={d}
              onClick={() => setDuration(d)}
              className={cn(
                'text-xs px-4 py-2 rounded-lg border font-semibold transition-all',
                duration === d
                  ? 'bg-amber-500 text-black border-amber-500 shadow-sm shadow-amber-900/30'
                  : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
              )}
            >
              {fmtDuration(d)}
            </button>
          ))}
        </div>
      </div>

      {/* ── Lyrics (collapsible) ── */}
      {showLyrics && vocals && (
        <div>
          <button
            onClick={() => setShowLyricsBox((v) => !v)}
            className="flex items-center gap-2 text-white/40 text-xs font-medium hover:text-white/70 transition-colors"
          >
            <Music size={13} />
            Add your own lyrics (optional)
            {showLyricsBox ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
          </button>
          {showLyricsBox && (
            <textarea
              value={lyrics}
              onChange={(e) => setLyrics(e.target.value)}
              placeholder={cfg.lyrics_placeholder ?? 'Paste your lyrics — verses, chorus, bridge…\n\n[Verse 1]\n…\n[Chorus]\n…'}
              rows={6}
              className="nexus-input resize-none w-full text-sm leading-relaxed mt-2"
            />
          )}
        </div>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-4 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-amber-500 to-orange-500 text-black hover:opacity-90 active:scale-[0.98] shadow-lg shadow-amber-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading ? (
          <><Loader2 size={15} className="animate-spin" /> Composing your track…</>
        ) : (
          <><Sparkles size={15} /> Generate Music</>
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
