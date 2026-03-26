'use client';

import { useState } from 'react';
import { Loader2, Music, ChevronDown, ChevronUp, Sparkles } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

const DEFAULT_GENRE_TAGS = [
  'Afrobeats', 'Amapiano', 'Gospel', 'Highlife', 'R&B', 'Hip-Hop',
  'Pop', 'Jazz', 'Classical', 'EDM', 'Reggae', 'Funk',
];
const DEFAULT_DURATIONS  = [15, 30, 60, 120, 180, 300];
const ENERGY_LABELS      = ['Chill', 'Relaxed', 'Balanced', 'Upbeat', 'Energetic'] as const;
const BPM_PRESETS        = [60, 80, 90, 100, 110, 120, 128, 140, 160] as const;

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
  const [prompt,        setPrompt]        = useState('');
  const [duration,      setDuration]      = useState<number>(cfg.default_duration ?? 30);
  const [vocals,        setVocals]        = useState<boolean>(cfg.default_vocals ?? true);
  const [lyrics,        setLyrics]        = useState('');
  const [showLyricsBox, setShowLyricsBox] = useState(false);
  // BPM — null = "Auto"
  const [bpm,           setBpm]           = useState<number | null>(null);
  // Energy 0–4 index
  const [energy,        setEnergy]        = useState<number>(2);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = prompt.trim().length >= 3;

  const filteredDurations = durations.filter((d: number) => d <= maxDur);

  function toggleTag(tag: string) {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag],
    );
  }

  function fmtDuration(s: number) {
    return s < 60 ? `${s}s` : `${s / 60}m`;
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;

    // Build an enriched prompt that includes genre, energy, and BPM cues
    const tagPrefix    = selectedTags.length > 0 ? `[${selectedTags.join(', ')}] ` : '';
    const energyLabel  = ENERGY_LABELS[energy];
    const energyCue    = energyLabel !== 'Balanced' ? ` ${energyLabel} energy.` : '';
    const bpmCue       = bpm !== null ? ` ${bpm} BPM.` : '';
    const vocalsCue    = showVocals ? (vocals ? ' With vocals.' : ' Instrumental only.') : '';

    const payload: GeneratePayload = {
      prompt:     tagPrefix + prompt.trim() + energyCue + bpmCue + vocalsCue,
      duration,
      vocals:     showVocals ? vocals : undefined,
      lyrics:     showLyrics && lyrics.trim() ? lyrics.trim() : undefined,
      style_tags: selectedTags.length > 0 ? selectedTags : undefined,
      extra_params: {
        bpm:    bpm ?? 'auto',
        energy: ENERGY_LABELS[energy],
      },
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Genre tags ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Genre</label>
        <div className="flex flex-wrap gap-1.5">
          {genreTags.map((tag: string) => (
            <button
              key={tag}
              onClick={() => toggleTag(tag)}
              className={cn(
                'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                selectedTags.includes(tag)
                  ? 'bg-amber-500 text-black border-amber-500'
                  : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
              )}
            >
              {tag}
            </button>
          ))}
        </div>
      </div>

      {/* ── Vocals / Instrumental toggle ── */}
      {showVocals && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Style</label>
          <div className="flex rounded-xl overflow-hidden border border-white/10 w-fit">
            <button
              onClick={() => setVocals(true)}
              className={cn(
                'px-5 py-2 text-xs font-semibold transition-all',
                vocals ? 'bg-amber-500 text-black' : 'text-white/55 hover:text-white/80',
              )}
            >
              🎤 With Vocals
            </button>
            <button
              onClick={() => setVocals(false)}
              className={cn(
                'px-5 py-2 text-xs font-semibold transition-all',
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
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
          Describe your music
        </label>
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
        <p className="text-white/25 text-[11px] mt-1">{prompt.length}/500 characters</p>
      </div>

      {/* ── Energy + BPM row ── */}
      {(showEnergy || showBpm) && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">

          {/* Energy slider */}
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
                type="range"
                min={0}
                max={4}
                step={1}
                value={energy}
                onChange={(e) => setEnergy(Number(e.target.value))}
                className="w-full h-1.5 rounded-full appearance-none cursor-pointer
                           bg-gradient-to-r from-blue-600 via-amber-500 to-red-500
                           [&::-webkit-slider-thumb]:appearance-none
                           [&::-webkit-slider-thumb]:w-4
                           [&::-webkit-slider-thumb]:h-4
                           [&::-webkit-slider-thumb]:rounded-full
                           [&::-webkit-slider-thumb]:bg-white
                           [&::-webkit-slider-thumb]:shadow-md"
              />
              <div className="flex justify-between mt-1">
                <span className="text-white/20 text-[9px]">Chill</span>
                <span className="text-white/20 text-[9px]">Energetic</span>
              </div>
            </div>
          )}

          {/* BPM picker */}
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
                      'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
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
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Duration</label>
        <div className="flex gap-2 flex-wrap">
          {filteredDurations.map((d: number) => (
            <button
              key={d}
              onClick={() => setDuration(d)}
              className={cn(
                'text-xs px-4 py-2 rounded-lg border font-semibold transition-all',
                duration === d
                  ? 'bg-amber-500 text-black border-amber-500'
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
            className="flex items-center gap-2 text-white/45 text-xs font-medium hover:text-white/75 transition-colors"
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
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-amber-500 to-orange-500 text-black hover:opacity-90 active:scale-[0.98] shadow-lg shadow-amber-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Composing…</>
          : <><Sparkles size={15} /> Generate Music →</>
        }
      </button>
    </div>
  );
}
