'use client';

import { useState } from 'react';
import { Loader2, Music, ChevronDown, ChevronUp, Sparkles, Shuffle, Plus } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

// ─── Static config ─────────────────────────────────────────────────────────────

const DEFAULT_GENRE_TAGS = [
  'Afrobeats', 'Amapiano', 'Gospel', 'Highlife', 'R&B', 'Hip-Hop',
  'Pop', 'Jazz', 'Classical', 'EDM', 'Reggae', 'Funk', 'Afro-Soul',
  'Dancehall', 'Trap', 'Lo-fi',
];

const DEFAULT_DURATIONS     = [15, 30, 60, 120, 180, 300];
const ENERGY_LABELS         = ['Chill', 'Relaxed', 'Balanced', 'Upbeat', 'Energetic'] as const;
const BPM_PRESETS           = [60, 80, 90, 100, 110, 120, 128, 140, 160] as const;

// Musical keys — matches Suno/Udio key selector
const MUSICAL_KEYS = [
  'Any', 'C major', 'C minor', 'C# major', 'D major', 'D minor',
  'Eb major', 'E major', 'E minor', 'F major', 'F minor',
  'F# major', 'G major', 'G minor', 'Ab major', 'A major', 'A minor',
  'Bb major', 'B major', 'B minor',
];

// Song structure presets — matches Suno structure control
const SONG_STRUCTURES = [
  { label: 'Auto',                  value: 'Auto' },
  { label: 'V-C-V-C',              value: 'Verse, Chorus, Verse, Chorus' },
  { label: 'V-C-V-C-B-C',         value: 'Verse, Chorus, Verse, Chorus, Bridge, Chorus' },
  { label: 'Intro-V-C-V-C-Outro', value: 'Intro, Verse, Chorus, Verse, Chorus, Outro' },
  { label: 'V-V-C-V',             value: 'Verse, Verse, Chorus, Verse' },
  { label: 'Chorus-first',        value: 'Chorus, Verse, Chorus, Bridge, Chorus' },
];

// Instrument focus presets
const INSTRUMENT_PRESETS = [
  'Piano', 'Guitar', 'Strings', 'Brass', 'Synth',
  'Drums only', 'Bass-heavy', 'A cappella', 'Full band',
];

// Jingle-specific use case tags
const JINGLE_USE_CASES = [
  'Brand intro', 'TV commercial', 'Radio ad', 'App notification',
  'Podcast intro', 'Social media', 'Hold music', 'Event fanfare',
];

// BG Music scene presets
const BG_SCENE_PRESETS = [
  'YouTube video', 'Podcast background', 'Corporate presentation',
  'Social media reel', 'Documentary', 'Study / focus', 'Meditation',
  'Workout', 'Restaurant ambience', 'Game background',
];

// Mood board
const MOODS = [
  { label: 'Happy',      emoji: '😊', color: 'bg-yellow-500/20 border-yellow-500/40 text-yellow-200' },
  { label: 'Romantic',   emoji: '💕', color: 'bg-pink-500/20 border-pink-500/40 text-pink-200' },
  { label: 'Melancholy', emoji: '🌧️', color: 'bg-blue-500/20 border-blue-500/40 text-blue-200' },
  { label: 'Hype',       emoji: '🔥', color: 'bg-orange-500/20 border-orange-500/40 text-orange-200' },
  { label: 'Spiritual',  emoji: '✨', color: 'bg-purple-500/20 border-purple-500/40 text-purple-200' },
  { label: 'Peaceful',   emoji: '🌿', color: 'bg-teal-500/20 border-teal-500/40 text-teal-200' },
];

// Lyrics section tag helpers — Suno-style
const LYRIC_SECTION_TAGS = [
  '[Verse 1]', '[Verse 2]', '[Pre-Chorus]', '[Chorus]',
  '[Bridge]', '[Outro]', '[Hook]', '[Intro]',
];

// Prompt inspirations per tool type
const PROMPT_INSPIRATIONS: Record<string, string[]> = {
  'song-creator': [
    'Upbeat Afrobeats love song, female vocals, summer vibes, catchy chorus hook',
    'Epic gospel choir, powerful crescendo, inspirational and uplifting',
    'Amapiano house track, deep log drum, smooth saxophone, late night energy',
    'Highlife guitar melody, nostalgic, storytelling vocals, warm and joyful',
    'Trap beat with melodic hook, dark 808s, atmospheric pads',
    'Dancehall riddim, bouncy rhythm, party anthem, Caribbean vibes',
  ],
  'bg-music': [
    'Calm lo-fi piano background for studying, minimal percussion, relaxing',
    'Uplifting corporate background music, professional, motivational',
    'Cinematic orchestral score, tension building, dramatic strings and brass',
    'Chill Afrobeats instrumental, smooth and laid-back, no vocals',
    'Ambient electronic background, atmospheric pads, focus music',
    'Acoustic guitar loop, warm and cosy, coffee shop vibe',
  ],
  'jingle': [
    '15-second energetic jingle for a fintech brand, catchy and memorable',
    'Upbeat 10-second app notification sound, modern and friendly',
    'Radio ad jingle, 30 seconds, fun and singable, product launch',
    'Corporate brand intro, 5 seconds, professional and confident',
    'Podcast intro music, 20 seconds, engaging and dynamic',
    'Retail store hold music, cheerful and non-intrusive',
  ],
  'instrumental': [
    'Calm piano background music for studying, 60 seconds, minimal',
    'Cinematic orchestral piece, epic and dramatic, full orchestra',
    'Smooth jazz instrumental, saxophone lead, late night club feel',
    'Electronic ambient track, floating pads, meditative and deep',
    'Afrobeats instrumental, log drum and guitar, no vocals',
    'Classical string quartet, elegant and sophisticated',
  ],
};

// ─── Context-aware tool config ─────────────────────────────────────────────────

function getToolContext(slug: string) {
  const s = slug.toLowerCase();
  if (s.includes('jingle') || s.includes('marketing')) {
    return {
      mode:          'jingle' as const,
      defaultVocals: true,
      showLyrics:    false,
      showVocals:    false,
      showStructure: false,
      defaultDur:    15,
      promptLabel:   'Describe your jingle',
      promptHint:    'e.g. 15-second energetic jingle for a fintech brand called Nexus, catchy and memorable…',
      accentColor:   'green',
    };
  }
  if (s.includes('bg-music') || s.includes('background')) {
    return {
      mode:          'bg-music' as const,
      defaultVocals: false,
      showLyrics:    false,
      showVocals:    false,
      showStructure: false,
      defaultDur:    30,
      promptLabel:   'Describe the scene or mood',
      promptHint:    'e.g. Calm lo-fi piano background for studying, minimal percussion, relaxing…',
      accentColor:   'blue',
    };
  }
  if (s.includes('instrumental')) {
    return {
      mode:          'instrumental' as const,
      defaultVocals: false,
      showLyrics:    false,
      showVocals:    false,
      showStructure: true,
      defaultDur:    60,
      promptLabel:   'Describe the instrumental',
      promptHint:    'e.g. Calm piano background music for studying, 60 seconds, minimal percussion…',
      accentColor:   'violet',
    };
  }
  // Default: song-creator
  return {
    mode:          'song-creator' as const,
    defaultVocals: true,
    showLyrics:    true,
    showVocals:    true,
    showStructure: true,
    defaultDur:    30,
    promptLabel:   'Music Style & Direction',
    promptHint:    'e.g. Upbeat Afrobeats love song, female vocals, summer vibes, catchy chorus hook…',
    accentColor:   'amber',
  };
}

// ─── Component ─────────────────────────────────────────────────────────────────

export default function MusicComposer({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg     = tool.ui_config ?? {};
  const ctx     = getToolContext(tool.slug);
  const mode    = ctx.mode;

  const genreTags = cfg.genre_tags       ?? DEFAULT_GENRE_TAGS;
  const durations = cfg.duration_options ?? DEFAULT_DURATIONS;
  const maxDur    = cfg.max_duration     ?? 300;
  const showBpm   = cfg.show_bpm         ?? true;
  const showEnergy = cfg.show_energy     ?? true;

  // Core state
  const [selectedTags,  setSelectedTags]  = useState<string[]>([]);
  const [selectedMood,  setSelectedMood]  = useState<string | null>(null);
  const [prompt,        setPrompt]        = useState('');
  const [duration,      setDuration]      = useState<number>(cfg.default_duration ?? ctx.defaultDur);
  const [vocals,        setVocals]        = useState<boolean>(ctx.defaultVocals);
  const [lyrics,        setLyrics]        = useState('');
  const [bpm,           setBpm]           = useState<number | null>(null);
  const [energy,        setEnergy]        = useState<number>(2);

  // Advanced / best-in-class controls
  const [selectedKey,       setSelectedKey]       = useState('Any');
  const [selectedStructure, setSelectedStructure] = useState('Auto');
  const [instruments,       setInstruments]       = useState('');
  const [negativePrompt,    setNegativePrompt]    = useState('');

  // Jingle-specific
  const [brandName,    setBrandName]    = useState('');
  const [jingleUseCase, setJingleUseCase] = useState('');

  // BG Music scene
  const [bgScene, setBgScene] = useState('');

  // UI toggles — lyrics always open for song-creator
  const [showLyricsBox, setShowLyricsBox] = useState(ctx.mode === 'song-creator');
  const [showInspo,     setShowInspo]     = useState(false);
  const [showAdvanced,  setShowAdvanced]  = useState(false);

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const isValid   = prompt.trim().length >= 3;
  const filteredDurations = (durations as number[]).filter((d) => d <= maxDur);

  const inspirations = PROMPT_INSPIRATIONS[mode] ?? PROMPT_INSPIRATIONS['song-creator'];

  // Accent colour classes
  const accent = {
    amber:  { tag: 'bg-amber-500 text-black border-amber-500', btn: 'from-amber-500 to-orange-500', hover: 'hover:border-amber-500/40' },
    green:  { tag: 'bg-green-500 text-black border-green-500', btn: 'from-green-500 to-emerald-500', hover: 'hover:border-green-500/40' },
    blue:   { tag: 'bg-blue-500 text-white border-blue-500',   btn: 'from-blue-500 to-cyan-500',    hover: 'hover:border-blue-500/40' },
    violet: { tag: 'bg-violet-500 text-white border-violet-500', btn: 'from-violet-500 to-purple-500', hover: 'hover:border-violet-500/40' },
  }[ctx.accentColor] ?? { tag: 'bg-amber-500 text-black border-amber-500', btn: 'from-amber-500 to-orange-500', hover: 'hover:border-amber-500/40' };

  function toggleTag(tag: string) {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : prev.length < 3 ? [...prev, tag] : prev,
    );
  }

  function fmtDuration(s: number) {
    return s < 60 ? `${s}s` : `${s / 60}m`;
  }

  function surpriseMe() {
    const random = inspirations[Math.floor(Math.random() * inspirations.length)];
    setPrompt(random);
  }

  function insertLyricTag(tag: string) {
    setLyrics((prev) => (prev ? `${prev}\n\n${tag}\n` : `${tag}\n`));
    if (!showLyricsBox) setShowLyricsBox(true);
  }

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;

    const tagPrefix   = selectedTags.length > 0 ? `[${selectedTags.join(', ')}] ` : '';
    const moodCue     = selectedMood ? ` ${selectedMood} mood.` : '';
    const energyLabel = ENERGY_LABELS[energy];
    const energyCue   = energyLabel !== 'Balanced' ? ` ${energyLabel} energy.` : '';
    const bpmCue      = bpm !== null ? ` ${bpm} BPM.` : '';

    // Mode-specific prompt enrichment
    let modeCue = '';
    if (mode === 'jingle') {
      if (brandName.trim()) modeCue += ` Brand: ${brandName.trim()}.`;
      if (jingleUseCase)    modeCue += ` Use case: ${jingleUseCase}.`;
      modeCue += ' No vocals required, catchy and memorable.';
    } else if (mode === 'bg-music') {
      if (bgScene) modeCue += ` Scene: ${bgScene}.`;
      modeCue += ' No vocals, loop-friendly, background music.';
    } else if (mode === 'instrumental') {
      modeCue += ' Instrumental only, no vocals.';
    } else {
      // song-creator
      const vocalsCue = ctx.showVocals ? (vocals ? ' With vocals.' : ' Instrumental only.') : '';
      modeCue = vocalsCue;
    }

    const payload: GeneratePayload = {
      prompt:          tagPrefix + prompt.trim() + moodCue + energyCue + bpmCue + modeCue,
      duration,
      vocals:          ctx.showVocals ? vocals : (mode === 'song-creator' ? true : false),
      lyrics:          ctx.showLyrics && lyrics.trim() ? lyrics.trim() : undefined,
      style_tags:      selectedTags.length > 0 ? selectedTags : undefined,
      negative_prompt: negativePrompt.trim() || undefined,
      extra_params: {
        bpm:        bpm ?? 'auto',
        energy:     ENERGY_LABELS[energy],
        mood:       selectedMood ?? undefined,
        key:        selectedKey !== 'Any' ? selectedKey : undefined,
        structure:  selectedStructure !== 'Auto' ? selectedStructure : undefined,
        instruments: instruments.trim() || undefined,
        brand_name:  brandName.trim() || undefined,
        jingle_use_case: jingleUseCase || undefined,
        bg_scene:    bgScene || undefined,
        tool_mode:   mode,
      },
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">

      {/* ── Animated waveform loading state ── */}
      {isLoading && (
        <div className="flex items-center justify-center gap-1 py-3">
          {Array.from({ length: 24 }).map((_, i) => (
            <div
              key={i}
              className={cn('w-1 rounded-full', ctx.accentColor === 'amber' ? 'bg-amber-500' : ctx.accentColor === 'green' ? 'bg-green-500' : ctx.accentColor === 'blue' ? 'bg-blue-500' : 'bg-violet-500')}
              style={{
                height: `${8 + Math.sin(i * 0.7) * 14 + 6}px`,
                animation: `pulse ${0.5 + (i % 6) * 0.08}s ease-in-out infinite alternate`,
                animationDelay: `${i * 0.04}s`,
              }}
            />
          ))}
        </div>
      )}

      {/* ── Jingle: Brand name + Use case ── */}
      {mode === 'jingle' && (
        <div className="space-y-3">
          <div>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
              Brand / Product Name <span className="text-white/25 normal-case font-normal">(optional)</span>
            </label>
            <input
              type="text"
              value={brandName}
              onChange={(e) => setBrandName(e.target.value)}
              placeholder="e.g. Nexus, Konga, MTN, Paystack…"
              className="nexus-input w-full text-sm"
            />
          </div>
          <div>
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
              Use Case
            </label>
            <div className="flex flex-wrap gap-1.5">
              {JINGLE_USE_CASES.map((uc) => (
                <button
                  key={uc}
                  onClick={() => setJingleUseCase(jingleUseCase === uc ? '' : uc)}
                  className={cn(
                    'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                    jingleUseCase === uc
                      ? accent.tag
                      : `text-white/55 border-white/15 ${accent.hover} hover:text-white/80`,
                  )}
                >
                  {uc}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ── BG Music: Scene presets ── */}
      {mode === 'bg-music' && (
        <div>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
            Scene / Use Case
          </label>
          <div className="flex flex-wrap gap-1.5">
            {BG_SCENE_PRESETS.map((scene) => (
              <button
                key={scene}
                onClick={() => setBgScene(bgScene === scene ? '' : scene)}
                className={cn(
                  'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                  bgScene === scene
                    ? accent.tag
                    : `text-white/55 border-white/15 ${accent.hover} hover:text-white/80`,
                )}
              >
                {scene}
              </button>
            ))}
          </div>
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
                  ? accent.tag
                  : selectedTags.length >= 3
                    ? 'text-white/20 border-white/8 cursor-not-allowed'
                    : `text-white/55 border-white/15 ${accent.hover} hover:text-white/80`,
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

      {/* ── Vocals / Instrumental toggle (song-creator only) ── */}
      {ctx.showVocals && (
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
        <div className="flex items-center justify-between mb-1">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
            {ctx.promptLabel}
          </label>
          <button
            onClick={surpriseMe}
            className="flex items-center gap-1 text-white/30 hover:text-amber-400 transition-colors text-[11px] font-medium"
          >
            <Shuffle size={11} /> Surprise me
          </button>
        </div>
        {ctx.mode === 'song-creator' && (
          <p className="text-white/30 text-[10px] mb-1.5">
            Describe the <span className="text-amber-400/70 font-medium">vibe, genre &amp; feel</span> — not the words. For your actual lyrics, use the section below.
          </p>
        )}
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder={cfg.prompt_placeholder ?? ctx.promptHint}
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
            {inspirations.map((inspo) => (
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
                    bpm === null ? accent.tag : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
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
                      bpm === b ? accent.tag : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
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
          {filteredDurations.map((d) => (
            <button
              key={d}
              onClick={() => setDuration(d)}
              className={cn(
                'text-xs px-4 py-2 rounded-lg border font-semibold transition-all',
                duration === d ? accent.tag : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
              )}
            >
              {fmtDuration(d)}
            </button>
          ))}
        </div>
      </div>

      {/* ── Advanced controls (Key, Structure, Instruments, Negative Prompt) ── */}
      <div>
        <button
          onClick={() => setShowAdvanced((v) => !v)}
          className="flex items-center gap-2 text-white/40 text-xs font-medium hover:text-white/70 transition-colors w-full"
        >
          <Sparkles size={13} />
          Advanced controls
          <span className="ml-auto text-white/20 text-[10px]">
            {[selectedKey !== 'Any' && selectedKey, selectedStructure !== 'Auto' && selectedStructure, instruments].filter(Boolean).join(' · ') || 'Key, Structure, Instruments'}
          </span>
          {showAdvanced ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
        </button>

        {showAdvanced && (
          <div className="mt-3 space-y-4 pl-1 border-l-2 border-white/[0.06]">

            {/* Musical Key */}
            <div>
              <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
                Musical Key
              </label>
              <div className="flex flex-wrap gap-1.5">
                {MUSICAL_KEYS.slice(0, 10).map((k) => (
                  <button
                    key={k}
                    onClick={() => setSelectedKey(k)}
                    className={cn(
                      'text-xs px-2.5 py-1.5 rounded-full border font-medium transition-all',
                      selectedKey === k
                        ? 'bg-violet-500 text-white border-violet-500'
                        : 'text-white/55 border-white/15 hover:border-violet-500/40 hover:text-white/80',
                    )}
                  >
                    {k}
                  </button>
                ))}
                <select
                  value={selectedKey}
                  onChange={(e) => setSelectedKey(e.target.value)}
                  className="text-xs px-2.5 py-1.5 rounded-full border border-white/15 bg-transparent text-white/55 hover:border-violet-500/40 transition-all cursor-pointer"
                >
                  {MUSICAL_KEYS.map((k) => (
                    <option key={k} value={k} className="bg-gray-900">{k}</option>
                  ))}
                </select>
              </div>
            </div>

            {/* Song Structure (song-creator + instrumental only) */}
            {ctx.showStructure && (
              <div>
                <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
                  Song Structure
                </label>
                <div className="flex flex-wrap gap-1.5">
                  {SONG_STRUCTURES.map((s) => (
                    <button
                      key={s.label}
                      onClick={() => setSelectedStructure(s.value)}
                      className={cn(
                        'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                        selectedStructure === s.value
                          ? 'bg-violet-500 text-white border-violet-500'
                          : 'text-white/55 border-white/15 hover:border-violet-500/40 hover:text-white/80',
                      )}
                    >
                      {s.label}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Instrument Focus */}
            <div>
              <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">
                Instrument Focus
              </label>
              <div className="flex flex-wrap gap-1.5 mb-2">
                {INSTRUMENT_PRESETS.map((inst) => (
                  <button
                    key={inst}
                    onClick={() => setInstruments(instruments === inst ? '' : inst)}
                    className={cn(
                      'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                      instruments === inst
                        ? 'bg-violet-500 text-white border-violet-500'
                        : 'text-white/55 border-white/15 hover:border-violet-500/40 hover:text-white/80',
                    )}
                  >
                    {inst}
                  </button>
                ))}
              </div>
              <input
                type="text"
                value={instruments}
                onChange={(e) => setInstruments(e.target.value)}
                placeholder="Or type custom instruments (e.g. kora, talking drum, oud)…"
                className="nexus-input w-full text-xs"
              />
            </div>

            {/* Negative Prompt */}
            <div>
              <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
                Negative Prompt <span className="text-white/25 normal-case font-normal">(what to avoid)</span>
              </label>
              <input
                type="text"
                value={negativePrompt}
                onChange={(e) => setNegativePrompt(e.target.value)}
                placeholder="e.g. no drums, no distortion, avoid heavy bass, no rap…"
                className="nexus-input w-full text-xs"
              />
            </div>
          </div>
        )}
      </div>

      {/* ── Lyrics editor (song-creator with vocals only) ── */}
      {ctx.showLyrics && (mode !== 'song-creator' || vocals) && (
        <div className="rounded-xl border border-white/10 bg-white/[0.02] p-4 space-y-3">

          {/* Header row */}
          <div className="flex items-center justify-between">
            <div>
              <label className="text-white/70 text-[11px] uppercase tracking-wider font-semibold flex items-center gap-1.5">
                <Music size={12} className="text-amber-400" />
                Your Lyrics
                <span className="text-white/25 normal-case font-normal text-[10px]">(optional)</span>
              </label>
              <p className="text-white/30 text-[10px] mt-0.5">
                Paste your verses, chorus and bridge here — the AI will sing them exactly as written.
              </p>
            </div>
            <button
              onClick={() => setShowLyricsBox((v) => !v)}
              className="text-white/25 hover:text-white/50 transition-colors"
            >
              {showLyricsBox ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
            </button>
          </div>

          {showLyricsBox && (
            <>
              {/* Section tag quick-insert helpers */}
              <div>
                <p className="text-white/25 text-[10px] mb-1.5">Click to insert section tags:</p>
                <div className="flex flex-wrap gap-1.5">
                  {LYRIC_SECTION_TAGS.map((tag) => (
                    <button
                      key={tag}
                      onClick={() => insertLyricTag(tag)}
                      className="flex items-center gap-1 text-[10px] px-2 py-1 rounded-md border border-white/10 text-white/40 hover:border-amber-500/40 hover:text-amber-400 transition-all font-mono"
                    >
                      <Plus size={9} />
                      {tag}
                    </button>
                  ))}
                </div>
              </div>

              <textarea
                value={lyrics}
                onChange={(e) => setLyrics(e.target.value)}
                placeholder={
                  cfg.lyrics_placeholder ??
                  '[Verse 1]\nWrite your verse here…\n\n[Chorus]\nWrite your chorus here…\n\n[Bridge]\nWrite your bridge here…'
                }
                rows={8}
                className="nexus-input resize-none w-full text-sm leading-relaxed font-mono"
              />
              <p className="text-white/20 text-[10px]">
                Leave blank to let the AI write its own lyrics based on your style direction above.
              </p>
            </>
          )}

          {/* Collapsed preview when hidden */}
          {!showLyricsBox && (
            <button
              onClick={() => setShowLyricsBox(true)}
              className="w-full text-left text-xs text-white/30 hover:text-amber-400 transition-colors py-1"
            >
              {lyrics.trim() ? (
                <span className="text-amber-400/70">✓ Lyrics added ({lyrics.trim().split('\n').length} lines) — click to edit</span>
              ) : (
                <span>+ Click to add your own lyrics…</span>
              )}
            </button>
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
            ? `bg-gradient-to-r ${accent.btn} text-black hover:opacity-90 active:scale-[0.98] shadow-lg`
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading ? (
          <><Loader2 size={15} className="animate-spin" /> {mode === 'jingle' ? 'Creating your jingle…' : mode === 'bg-music' ? 'Generating background music…' : mode === 'instrumental' ? 'Composing instrumental…' : 'Composing your track…'}</>
        ) : (
          <><Sparkles size={15} /> {mode === 'jingle' ? 'Generate Jingle' : mode === 'bg-music' ? 'Generate Background Music' : mode === 'instrumental' ? 'Generate Instrumental' : 'Generate Music'}</>
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
