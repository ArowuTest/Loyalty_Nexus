'use client';

import { useState } from 'react';
import { Loader2, Music, ChevronDown, ChevronUp, Sparkles } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

export default function MusicComposer({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg = tool.ui_config ?? {};
  const genreTags   = cfg.genre_tags   ?? ['Afrobeats', 'Pop', 'R&B', 'Hip-Hop', 'Jazz', 'Gospel', 'Classical', 'Electronic', 'Reggae', 'Rock'];
  const durations   = cfg.duration_options ?? [15, 30, 60, 120];
  const showVocals  = cfg.show_vocals_toggle ?? true;
  const showLyrics  = cfg.show_lyrics_box ?? true;

  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [prompt,       setPrompt]       = useState('');
  const [duration,     setDuration]     = useState<number>(cfg.default_duration ?? 30);
  const [vocals,       setVocals]       = useState<boolean>(cfg.default_vocals ?? true);
  const [lyrics,       setLyrics]       = useState('');
  const [showLyricsBox, setShowLyricsBox] = useState(false);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = prompt.trim().length >= 3;

  function toggleTag(tag: string) {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag],
    );
  }

  function fmtDuration(s: number) {
    return s < 60 ? `${s}s` : `${s / 60}min`;
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const tagPrefix = selectedTags.length > 0 ? `[${selectedTags.join(', ')}] ` : '';
    const payload: GeneratePayload = {
      prompt:        tagPrefix + prompt.trim(),
      duration,
      vocals:        showVocals ? vocals : undefined,
      lyrics:        showLyrics && lyrics.trim() ? lyrics.trim() : undefined,
      style_tags:    selectedTags.length > 0 ? selectedTags : undefined,
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Genre tags ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Genre</label>
        <div className="flex flex-wrap gap-1.5">
          {genreTags.map((tag) => (
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
              ? 'e.g. Upbeat Afrobeats love song, female vocals, 120 BPM, summer vibes…'
              : 'e.g. Calm piano background music for studying, minimal, relaxing…')
          }
          rows={4}
          autoFocus
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />
        <p className="text-white/25 text-[11px] mt-1">{prompt.length}/500 characters</p>
      </div>

      {/* ── Duration ── */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Duration</label>
        <div className="flex gap-2 flex-wrap">
          {durations.map((d) => (
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

      {/* ── Vocals toggle ── */}
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

      {/* ── Lyrics (collapsible) ── */}
      {showLyrics && (
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
              placeholder={cfg.lyrics_placeholder ?? 'Paste your lyrics here — verses, chorus, bridge…'}
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
          ? <><Loader2 size={15} className="animate-spin" /> Generating…</>
          : <><Sparkles size={15} /> Generate Music →</>
        }
      </button>
    </div>
  );
}
