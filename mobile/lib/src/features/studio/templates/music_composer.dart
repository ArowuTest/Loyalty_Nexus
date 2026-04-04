// ─── Music Composer Template ──────────────────────────────────────────────────
// Mirrors webapp MusicComposer.tsx exactly.
// Supports modes: song-creator, jingle, bg-music, instrumental.
// Primary provider: Suno AI. Fallback: ElevenLabs / Pollinations.
// Payload: prompt, duration, vocals, lyrics, style_tags, negative_prompt,
//          extra_params { bpm, energy, mood, key, structure, instruments,
//                         brand_name, jingle_use_case, bg_scene, tool_mode,
//                         genre, title, vocal_gender }

import 'package:flutter/material.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';

// ─── Constants ────────────────────────────────────────────────────────────────

const _defaultGenreTags = [
  'Afrobeats', 'Amapiano', 'Highlife', 'Afropop', 'Afro-Soul', 'Afro-Jazz',
  'Hip-Hop', 'R&B', 'Pop', 'Gospel', 'Reggae', 'Dancehall', 'Electronic',
  'Jazz', 'Classical', 'Country', 'Rock', 'Funk', 'Blues', 'Lo-fi',
];

const _defaultDurations = [30, 60, 90, 120, 180, 240];

const _energyLabels = ['Calm', 'Mellow', 'Balanced', 'Energetic', 'Intense'];

const _moodOptions = [
  {'label': 'Happy',       'emoji': '😊'},
  {'label': 'Sad',         'emoji': '😢'},
  {'label': 'Romantic',    'emoji': '💕'},
  {'label': 'Angry',       'emoji': '😤'},
  {'label': 'Chill',       'emoji': '😎'},
  {'label': 'Epic',        'emoji': '⚡'},
  {'label': 'Mysterious',  'emoji': '🌙'},
  {'label': 'Nostalgic',   'emoji': '🌅'},
  {'label': 'Motivational','emoji': '🔥'},
  {'label': 'Peaceful',    'emoji': '🕊️'},
];

const _keyOptions = ['Any', 'C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B'];
const _structureOptions = ['Auto', 'Verse-Chorus', 'Verse-Chorus-Bridge', 'Intro-Verse-Chorus-Outro', 'Loop'];
const _vocalGenders = ['female', 'male', 'mixed'];
const _bgScenes = ['YouTube video', 'Podcast', 'Meditation', 'Corporate presentation', 'Social media reel', 'Game', 'Film score'];
const _jingleUseCases = ['TV ad', 'Radio ad', 'Social media', 'Brand identity', 'Product launch', 'App notification'];

const _songInspiration = [
  'Upbeat Afrobeats love song, female vocals, summer vibes, catchy chorus hook',
  'Amapiano house track, deep log drum, smooth saxophone, late night energy',
  'Highlife guitar melody, nostalgic, storytelling vocals, warm and joyful',
];
const _jingleInspiration = [
  '15-second energetic jingle for a fintech brand, catchy and memorable',
  'Radio ad jingle, 30 seconds, fun and singable, product launch',
];
const _instrumentalInspiration = [
  'Afrobeats instrumental, log drum and guitar, no vocals',
  'Smooth jazz instrumental, saxophone lead, late night club feel',
];
const _bgInspiration = [
  'Calm lo-fi beats, study music, no vocals',
  'Upbeat background music for a YouTube vlog, positive energy',
];

// ─── Mode context ─────────────────────────────────────────────────────────────

class _ModeCtx {
  final String mode;
  final String promptLabel;
  final String promptHint;
  final bool showVocals;
  final bool showLyrics;
  final int defaultDur;
  final bool defaultVocals;
  final List<String> inspirations;

  const _ModeCtx({
    required this.mode,
    required this.promptLabel,
    required this.promptHint,
    required this.showVocals,
    required this.showLyrics,
    required this.defaultDur,
    required this.defaultVocals,
    required this.inspirations,
  });
}

_ModeCtx _ctxForSlug(String slug) {
  if (slug.contains('jingle') || slug.contains('marketing')) {
    return const _ModeCtx(
      mode: 'jingle',
      promptLabel: 'Describe your jingle',
      promptHint: 'e.g. 15-second energetic jingle for a fintech brand called Nexus, catchy and memorable…',
      showVocals: false,
      showLyrics: false,
      defaultDur: 30,
      defaultVocals: false,
      inspirations: _jingleInspiration,
    );
  }
  if (slug.contains('bg-music') || slug.contains('background')) {
    return const _ModeCtx(
      mode: 'bg-music',
      promptLabel: 'Describe the background music',
      promptHint: 'e.g. Calm lo-fi beats for a study session, no vocals…',
      showVocals: false,
      showLyrics: false,
      defaultDur: 120,
      defaultVocals: false,
      inspirations: _bgInspiration,
    );
  }
  if (slug.contains('instrumental')) {
    return const _ModeCtx(
      mode: 'instrumental',
      promptLabel: 'Describe the instrumental',
      promptHint: 'e.g. Afrobeats instrumental, log drum and guitar, no vocals…',
      showVocals: false,
      showLyrics: false,
      defaultDur: 120,
      defaultVocals: false,
      inspirations: _instrumentalInspiration,
    );
  }
  return const _ModeCtx(
    mode: 'song-creator',
    promptLabel: 'Describe your song',
    promptHint: 'e.g. Upbeat Afrobeats love song, female vocals, summer vibes, catchy chorus hook…',
    showVocals: true,
    showLyrics: true,
    defaultDur: 120,
    defaultVocals: true,
    inspirations: _songInspiration,
  );
}

// ─── Widget ───────────────────────────────────────────────────────────────────

class MusicComposerTemplate extends StatefulWidget {
  final TemplateProps props;
  const MusicComposerTemplate({super.key, required this.props});

  @override
  State<MusicComposerTemplate> createState() => _MusicComposerTemplateState();
}

class _MusicComposerTemplateState extends State<MusicComposerTemplate> {
  late final _ModeCtx ctx;

  final _promptCtrl    = TextEditingController();
  final _lyricsCtrl    = TextEditingController();
  final _negCtrl       = TextEditingController();
  final _instrumentCtrl = TextEditingController();
  final _brandCtrl     = TextEditingController();
  final _titleCtrl     = TextEditingController();

  late int    _duration;
  late bool   _vocals;
  int?        _bpm;
  int         _energy = 2;
  String?     _mood;
  String      _key = 'Any';
  String      _structure = 'Auto';
  String      _vocalGender = 'female';
  String?     _jingleUseCase;
  String?     _bgScene;

  final List<String> _selectedTags = [];

  // Speech-to-text
  final stt.SpeechToText _speech = stt.SpeechToText();
  bool _micAvailable = false;
  bool _micListening = false;
  bool _lyricsMicListening = false;

  @override
  void initState() {
    super.initState();
    ctx = _ctxForSlug(widget.props.slug);
    _duration = ctx.defaultDur;
    _vocals   = ctx.defaultVocals;
    _initSpeech();
  }

  Future<void> _initSpeech() async {
    final available = await _speech.initialize();
    if (mounted) setState(() => _micAvailable = available);
  }

  void _toggleMic(bool isLyrics) async {
    if (!_micAvailable) return;
    if (_speech.isListening) {
      await _speech.stop();
      setState(() { _micListening = false; _lyricsMicListening = false; });
      return;
    }
    setState(() {
      _micListening = !isLyrics;
      _lyricsMicListening = isLyrics;
    });
    await _speech.listen(
      onResult: (result) {
        if (result.finalResult) {
          setState(() {
            if (isLyrics) {
              _lyricsCtrl.text = result.recognizedWords;
              _lyricsMicListening = false;
            } else {
              _promptCtrl.text = result.recognizedWords;
              _micListening = false;
            }
          });
        }
      },
      localeId: 'en_NG',
    );
  }

  void _handleSubmit() {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _promptCtrl.text.trim().isEmpty) return;

    final tagPrefix   = _selectedTags.isNotEmpty ? '[${_selectedTags.join(', ')}] ' : '';
    final moodCue     = _mood != null ? ' $_mood mood.' : '';
    final energyLabel = _energyLabels[_energy];
    final energyCue   = energyLabel != 'Balanced' ? ' $energyLabel energy.' : '';
    final bpmCue      = _bpm != null ? ' $_bpm BPM.' : '';

    String modeCue = '';
    if (ctx.mode == 'jingle') {
      if (_brandCtrl.text.trim().isNotEmpty) modeCue += ' Brand: ${_brandCtrl.text.trim()}.';
      if (_jingleUseCase != null) modeCue += ' Use case: $_jingleUseCase.';
      modeCue += ' No vocals required, catchy and memorable.';
    } else if (ctx.mode == 'bg-music') {
      if (_bgScene != null) modeCue += ' Scene: $_bgScene.';
      modeCue += ' No vocals, loop-friendly, background music.';
    } else if (ctx.mode == 'instrumental') {
      modeCue += ' Instrumental only, no vocals.';
    } else {
      final vocalsCue = ctx.showVocals ? (_vocals ? ' With vocals.' : ' Instrumental only.') : '';
      modeCue = vocalsCue;
    }

    final payload = GeneratePayload(
      prompt: tagPrefix + _promptCtrl.text.trim() + moodCue + energyCue + bpmCue + modeCue,
      duration: _duration,
      vocals: ctx.showVocals ? _vocals : (ctx.mode == 'song-creator' ? true : false),
      lyrics: ctx.showLyrics && _lyricsCtrl.text.trim().isNotEmpty
          ? _lyricsCtrl.text.trim()
          : null,
      styleTags: _selectedTags.isNotEmpty ? List.from(_selectedTags) : null,
      negativePrompt: _negCtrl.text.trim().isNotEmpty ? _negCtrl.text.trim() : null,
      extraParams: {
        'bpm':              _bpm ?? 'auto',
        'energy':           energyLabel,
        if (_mood != null) 'mood': _mood,
        if (_key != 'Any') 'key': _key,
        if (_structure != 'Auto') 'structure': _structure,
        if (_instrumentCtrl.text.trim().isNotEmpty) 'instruments': _instrumentCtrl.text.trim(),
        if (_brandCtrl.text.trim().isNotEmpty) 'brand_name': _brandCtrl.text.trim(),
        if (_jingleUseCase != null) 'jingle_use_case': _jingleUseCase,
        if (_bgScene != null) 'bg_scene': _bgScene,
        'tool_mode': ctx.mode,
        if (_selectedTags.isNotEmpty) 'genre': _selectedTags.first,
        if (_titleCtrl.text.trim().isNotEmpty) 'title': _titleCtrl.text.trim(),
        if (ctx.mode == 'song-creator' && _vocals) 'vocal_gender': _vocalGender,
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _promptCtrl.dispose();
    _lyricsCtrl.dispose();
    _negCtrl.dispose();
    _instrumentCtrl.dispose();
    _brandCtrl.dispose();
    _titleCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final isValid = _promptCtrl.text.trim().isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ── Provider badge ──
        const ProviderBadge(
          label: 'Suno AI',
          description: 'Professional music generation with full song structure',
          color: Color(0xFF10B981),
          icon: Icons.music_note_rounded,
        ),
        const SizedBox(height: 16),

        // ── Genre tags ──
        buildSectionLabel('Genre / Style'),
        buildChipRow(
          options: _defaultGenreTags,
          selected: _selectedTags,
          onToggle: (tag) => setState(() {
            if (_selectedTags.contains(tag)) {
              _selectedTags.remove(tag);
            } else if (_selectedTags.length < 3) {
              _selectedTags.add(tag);
            }
          }),
          activeColor: const Color(0xFF10B981),
        ),
        const SizedBox(height: 16),

        // ── Main prompt ──
        buildSectionLabel(ctx.promptLabel),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: ctx.promptHint,
          maxLines: 4,
          maxLength: 500,
          onMicTap: _micAvailable ? () => _toggleMic(false) : null,
          micActive: _micListening,
        ),
        const SizedBox(height: 8),

        // ── Inspirations ──
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: ctx.inspirations.map((insp) => Padding(
              padding: const EdgeInsets.only(right: 8),
              child: GestureDetector(
                onTap: () => setState(() => _promptCtrl.text = insp),
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  decoration: BoxDecoration(
                    color: Colors.white.withOpacity(0.04),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.white.withOpacity(0.1)),
                  ),
                  child: Text(
                    insp.length > 40 ? '${insp.substring(0, 40)}…' : insp,
                    style: TextStyle(color: Colors.white.withOpacity(0.45), fontSize: 11),
                  ),
                ),
              ),
            )).toList(),
          ),
        ),
        const SizedBox(height: 16),

        // ── Duration ──
        buildSectionLabel('Duration'),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: _defaultDurations.map((d) {
              final label = d < 60 ? '${d}s' : '${d ~/ 60}m${d % 60 > 0 ? ' ${d % 60}s' : ''}';
              return Padding(
                padding: const EdgeInsets.only(right: 8),
                child: buildChip(
                  label: label,
                  selected: _duration == d,
                  onTap: () => setState(() => _duration = d),
                  activeColor: const Color(0xFF10B981),
                ),
              );
            }).toList(),
          ),
        ),
        const SizedBox(height: 16),

        // ── Vocals (song-creator only) ──
        if (ctx.showVocals) ...[
          buildSectionLabel('Vocals'),
          Row(
            children: [
              _voiceToggle('With Vocals', true),
              const SizedBox(width: 8),
              _voiceToggle('Instrumental', false),
            ],
          ),
          const SizedBox(height: 16),
        ],

        // ── Vocal gender (song-creator + vocals) ──
        if (ctx.mode == 'song-creator' && _vocals) ...[
          buildSectionLabel('Vocal Gender'),
          Row(
            children: _vocalGenders.map((g) => Padding(
              padding: const EdgeInsets.only(right: 8),
              child: buildChip(
                label: g[0].toUpperCase() + g.substring(1),
                selected: _vocalGender == g,
                onTap: () => setState(() => _vocalGender = g),
                activeColor: const Color(0xFFDB2777),
              ),
            )).toList(),
          ),
          const SizedBox(height: 16),
        ],

        // ── Mood ──
        buildSectionLabel('Mood (optional)'),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: _moodOptions.map((m) => Padding(
              padding: const EdgeInsets.only(right: 8),
              child: buildChip(
                label: m['label']!,
                emoji: m['emoji'],
                selected: _mood == m['label'],
                onTap: () => setState(() => _mood = _mood == m['label'] ? null : m['label']),
                activeColor: const Color(0xFFF59E0B),
              ),
            )).toList(),
          ),
        ),
        const SizedBox(height: 16),

        // ── Energy ──
        buildSectionLabel('Energy — ${_energyLabels[_energy]}'),
        Slider(
          value: _energy.toDouble(),
          min: 0,
          max: 4,
          divisions: 4,
          activeColor: const Color(0xFF10B981),
          inactiveColor: Colors.white.withOpacity(0.1),
          onChanged: (v) => setState(() => _energy = v.round()),
        ),
        const SizedBox(height: 16),

        // ── Jingle-specific fields ──
        if (ctx.mode == 'jingle') ...[
          buildSectionLabel('Brand Name (optional)'),
          buildTextArea(controller: _brandCtrl, placeholder: 'e.g. Loyalty Nexus', maxLines: 1),
          const SizedBox(height: 12),
          buildSectionLabel('Use Case'),
          buildChipRow(
            options: _jingleUseCases,
            selected: _jingleUseCase != null ? [_jingleUseCase!] : [],
            onToggle: (v) => setState(() => _jingleUseCase = _jingleUseCase == v ? null : v),
            activeColor: const Color(0xFF10B981),
          ),
          const SizedBox(height: 16),
        ],

        // ── BG music scene ──
        if (ctx.mode == 'bg-music') ...[
          buildSectionLabel('Scene (optional)'),
          buildChipRow(
            options: _bgScenes,
            selected: _bgScene != null ? [_bgScene!] : [],
            onToggle: (v) => setState(() => _bgScene = _bgScene == v ? null : v),
            activeColor: const Color(0xFF10B981),
          ),
          const SizedBox(height: 16),
        ],

        // ── Lyrics (song-creator) ──
        if (ctx.showLyrics) ...[
          CollapsibleSection(
            title: 'Custom Lyrics (optional)',
            child: buildTextArea(
              controller: _lyricsCtrl,
              placeholder: 'Paste or type your lyrics here…',
              maxLines: 6,
              onMicTap: _micAvailable ? () => _toggleMic(true) : null,
              micActive: _lyricsMicListening,
            ),
          ),
          const SizedBox(height: 16),
        ],

        // ── Advanced options ──
        CollapsibleSection(
          title: 'Advanced Options',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Song title
              if (ctx.mode == 'song-creator') ...[
                buildSectionLabel('Song Title (optional)'),
                buildTextArea(controller: _titleCtrl, placeholder: 'e.g. Lagos Nights', maxLines: 1),
                const SizedBox(height: 12),
              ],
              // BPM
              buildSectionLabel('BPM (optional)'),
              Row(
                children: [80, 100, 120, 140, 160].map((bpm) => Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: buildChip(
                    label: '$bpm',
                    selected: _bpm == bpm,
                    onTap: () => setState(() => _bpm = _bpm == bpm ? null : bpm),
                  ),
                )).toList(),
              ),
              const SizedBox(height: 12),
              // Key
              buildSectionLabel('Key'),
              buildChipRow(
                options: _keyOptions,
                selected: [_key],
                onToggle: (v) => setState(() => _key = v),
              ),
              const SizedBox(height: 12),
              // Structure
              buildSectionLabel('Structure'),
              buildChipRow(
                options: _structureOptions,
                selected: [_structure],
                onToggle: (v) => setState(() => _structure = v),
              ),
              const SizedBox(height: 12),
              // Instruments
              buildSectionLabel('Featured Instruments (optional)'),
              buildTextArea(
                controller: _instrumentCtrl,
                placeholder: 'e.g. guitar, saxophone, log drum',
                maxLines: 1,
              ),
              const SizedBox(height: 12),
              // Negative prompt
              buildSectionLabel('Avoid (optional)'),
              buildTextArea(
                controller: _negCtrl,
                placeholder: 'e.g. no drums, no distortion',
                maxLines: 2,
              ),
            ],
          ),
        ),
        const SizedBox(height: 24),

        // ── Generate button ──
        buildGenerateButton(
          label: p.isLoading
              ? 'Composing…'
              : 'Compose${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF059669), Color(0xFF10B981)],
          icon: Icons.music_note_rounded,
        ),

        if (!p.canAfford) ...[
          const SizedBox(height: 8),
          Center(
            child: Text(
              'You need ${p.pointCost} Pulse Points to use this tool',
              style: TextStyle(color: Colors.red.withOpacity(0.7), fontSize: 12),
            ),
          ),
        ],
      ],
    );
  }

  Widget _voiceToggle(String label, bool value) {
    final selected = _vocals == value;
    return Expanded(
      child: GestureDetector(
        onTap: () => setState(() => _vocals = value),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          padding: const EdgeInsets.symmetric(vertical: 10),
          decoration: BoxDecoration(
            color: selected ? const Color(0xFF10B981).withOpacity(0.15) : Colors.transparent,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(
              color: selected ? const Color(0xFF10B981).withOpacity(0.5) : Colors.white.withOpacity(0.12),
            ),
          ),
          child: Center(
            child: Text(
              label,
              style: TextStyle(
                color: selected ? Colors.white : Colors.white.withOpacity(0.45),
                fontWeight: FontWeight.w600,
                fontSize: 13,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
