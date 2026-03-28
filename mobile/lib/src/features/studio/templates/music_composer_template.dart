import 'package:flutter/material.dart';
import 'template_types.dart';

const _defaultGenreTags = [
  'Afrobeats', 'Amapiano', 'Gospel', 'Highlife', 'R&B', 'Hip-Hop',
  'Pop', 'Jazz', 'Classical', 'EDM', 'Reggae', 'Funk',
];
const _defaultDurations = [15, 30, 60, 120, 180, 300];
const _energyLabels     = ['Chill', 'Relaxed', 'Balanced', 'Upbeat', 'Energetic'];
const _energyColors     = [
  Color(0xFF3B82F6), Color(0xFF14B8A6), Color(0xFFF59E0B),
  Color(0xFFF97316), Color(0xFFEF4444),
];
const _bpmPresets       = [60, 80, 90, 100, 110, 120, 128, 140, 160];

class MusicComposerTemplate extends StatefulWidget {
  final TemplateProps props;
  const MusicComposerTemplate({super.key, required this.props});

  @override
  State<MusicComposerTemplate> createState() => _MusicComposerTemplateState();
}

class _MusicComposerTemplateState extends State<MusicComposerTemplate> {
  final _promptCtrl  = TextEditingController();
  final _lyricsCtrl  = TextEditingController();
  List<String> _selectedTags = [];
  int    _duration     = 30;
  bool   _vocals       = true;
  bool   _showLyrics   = false;
  int?   _bpm;           // null = Auto
  double _energy       = 2.0; // 0–4

  TemplateProps get p => widget.props;

  List<String> get _genreTags {
    final raw = p.uiConfig['genre_tags'];
    if (raw is List) return raw.cast<String>();
    return _defaultGenreTags;
  }

  List<int> get _durations {
    final maxDur = (p.uiConfig['max_duration'] ?? 300) as int;
    final raw    = p.uiConfig['duration_options'];
    final list   = raw is List ? raw.cast<int>() : List<int>.from(_defaultDurations);
    return list.where((d) => d <= maxDur).toList();
  }

  bool get _showVocals  => p.uiConfig['show_vocals_toggle'] != false;
  bool get _showLyrBox  => p.uiConfig['show_lyrics_box'] != false;
  bool get _showBpm     => p.uiConfig['show_bpm'] != false;
  bool get _showEnergy  => p.uiConfig['show_energy'] != false;

  bool get _isValid => _promptCtrl.text.trim().length >= 3;

  String _fmtDuration(int s) => s < 60 ? '${s}s' : '${s ~/ 60}m';

  void _handleSubmit() {
    if (!_isValid || p.isLoading || !p.canAfford) return;
    final tagPrefix   = _selectedTags.isNotEmpty ? '[${_selectedTags.join(', ')}] ' : '';
    final energyLabel = _energyLabels[_energy.round()];
    final energyCue   = energyLabel != 'Balanced' ? ' $energyLabel energy.' : '';
    final bpmCue      = _bpm != null ? ' $_bpm BPM.' : '';
    final vocalsCue   = _showVocals ? (_vocals ? ' With vocals.' : ' Instrumental only.') : '';
    p.onSubmit(GeneratePayload(
      prompt:    tagPrefix + _promptCtrl.text.trim() + energyCue + bpmCue + vocalsCue,
      duration:  _duration,
      vocals:    _showVocals ? _vocals : null,
      lyrics:    _showLyrBox && _lyricsCtrl.text.trim().isNotEmpty ? _lyricsCtrl.text.trim() : null,
      styleTags: _selectedTags.isNotEmpty ? _selectedTags : null,
      extraParams: {
        'bpm':    _bpm ?? 'auto',
        'energy': energyLabel,
      },
    ));
  }

  @override
  void dispose() {
    _promptCtrl.dispose();
    _lyricsCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [

        // ── Genre tags ──
        buildSectionLabel('Genre'),
        Wrap(
          spacing: 6, runSpacing: 6,
          children: _genreTags.map((tag) => buildChip(
            label: tag,
            selected: _selectedTags.contains(tag),
            activeColor: const Color(0xFFF59E0B),
            activeText: Colors.black,
            onTap: () => setState(() {
              _selectedTags.contains(tag)
                  ? _selectedTags.remove(tag)
                  : _selectedTags.add(tag);
            }),
          )).toList(),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Vocals / Instrumental toggle ──
        if (_showVocals) ...[
          buildSectionLabel('Style'),
          Container(
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.white.withOpacity(0.1)),
            ),
            clipBehavior: Clip.hardEdge,
            child: Row(
              children: [
                Expanded(
                  child: GestureDetector(
                    onTap: () => setState(() => _vocals = true),
                    child: Container(
                      padding: const EdgeInsets.symmetric(vertical: 11),
                      color: _vocals ? const Color(0xFFF59E0B) : Colors.transparent,
                      alignment: Alignment.center,
                      child: Text('🎤 With Vocals',
                          style: TextStyle(
                            fontSize: 12, fontWeight: FontWeight.w700,
                            color: _vocals ? Colors.black : Colors.white.withOpacity(0.55),
                          )),
                    ),
                  ),
                ),
                Expanded(
                  child: GestureDetector(
                    onTap: () => setState(() => _vocals = false),
                    child: Container(
                      padding: const EdgeInsets.symmetric(vertical: 11),
                      color: !_vocals ? const Color(0xFFF59E0B) : Colors.transparent,
                      alignment: Alignment.center,
                      child: Text('🎹 Instrumental',
                          style: TextStyle(
                            fontSize: 12, fontWeight: FontWeight.w700,
                            color: !_vocals ? Colors.black : Colors.white.withOpacity(0.55),
                          )),
                    ),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Describe your music ──
        buildSectionLabel('Describe your music'),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: p.uiConfig['prompt_placeholder'] ??
              (_vocals
                  ? 'e.g. Upbeat Afrobeats love song, female vocals, summer vibes, catchy chorus hook…'
                  : 'e.g. Calm lo-fi piano background for studying, minimal percussion, relaxing…'),
          maxLines: 3,
          autoFocus: true,
        ),
        const SizedBox(height: 4),
        ValueListenableBuilder(
          valueListenable: _promptCtrl,
          builder: (_, __, ___) => Text('${_promptCtrl.text.length}/500 characters', style: kHintStyle),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Energy + BPM ──
        if (_showEnergy || _showBpm) ...[
          if (_showEnergy) ...[
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text('ENERGY', style: kLabelStyle),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: _energyColors[_energy.round()].withOpacity(0.2),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(_energyLabels[_energy.round()], style: TextStyle(
                    fontSize: 11, fontWeight: FontWeight.w700,
                    color: _energyColors[_energy.round()],
                  )),
                ),
              ],
            ),
            const SizedBox(height: 8),
            SliderTheme(
              data: SliderThemeData(
                trackHeight: 3,
                thumbColor: Colors.white,
                activeTrackColor: _energyColors[_energy.round()],
                inactiveTrackColor: Colors.white.withOpacity(0.1),
                overlayColor: Colors.white.withOpacity(0.1),
              ),
              child: Slider(
                min: 0, max: 4, divisions: 4, value: _energy,
                onChanged: (v) => setState(() => _energy = v),
              ),
            ),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text('Chill', style: kHintStyle.copyWith(fontSize: 9)),
                Text('Energetic', style: kHintStyle.copyWith(fontSize: 9)),
              ],
            ),
            const SizedBox(height: kTemplateSpacing),
          ],

          if (_showBpm) ...[
            buildSectionLabel('BPM'),
            Wrap(
              spacing: 6, runSpacing: 6,
              children: [
                // Auto chip
                GestureDetector(
                  onTap: () => setState(() => _bpm = null),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 150),
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                    decoration: BoxDecoration(
                      color: _bpm == null ? const Color(0xFFF59E0B) : Colors.transparent,
                      borderRadius: BorderRadius.circular(999),
                      border: Border.all(
                        color: _bpm == null ? const Color(0xFFF59E0B) : Colors.white.withOpacity(0.15),
                      ),
                    ),
                    child: Text('Auto', style: TextStyle(
                      fontSize: 12, fontWeight: FontWeight.w700,
                      color: _bpm == null ? Colors.black : Colors.white.withOpacity(0.55),
                    )),
                  ),
                ),
                ..._bpmPresets.map((b) => GestureDetector(
                  onTap: () => setState(() => _bpm = b),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 150),
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                    decoration: BoxDecoration(
                      color: _bpm == b ? const Color(0xFFF59E0B) : Colors.transparent,
                      borderRadius: BorderRadius.circular(999),
                      border: Border.all(
                        color: _bpm == b ? const Color(0xFFF59E0B) : Colors.white.withOpacity(0.15),
                      ),
                    ),
                    child: Text('$b', style: TextStyle(
                      fontSize: 12, fontWeight: FontWeight.w700,
                      color: _bpm == b ? Colors.black : Colors.white.withOpacity(0.55),
                    )),
                  ),
                )),
              ],
            ),
            const SizedBox(height: kTemplateSpacing),
          ],
        ],

        // ── Duration ──
        buildSectionLabel('Duration'),
        Wrap(
          spacing: 8, runSpacing: 8,
          children: _durations.map((d) {
            final sel = _duration == d;
            return GestureDetector(
              onTap: () => setState(() => _duration = d),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                decoration: BoxDecoration(
                  color: sel ? const Color(0xFFF59E0B) : Colors.transparent,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: sel ? const Color(0xFFF59E0B) : Colors.white.withOpacity(0.15),
                  ),
                ),
                child: Text(_fmtDuration(d), style: TextStyle(
                  fontSize: 12, fontWeight: FontWeight.w700,
                  color: sel ? Colors.black : Colors.white.withOpacity(0.55),
                )),
              ),
            );
          }).toList(),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Lyrics (collapsible, only with vocals) ──
        if (_showLyrBox && _vocals) ...[
          GestureDetector(
            onTap: () => setState(() => _showLyrics = !_showLyrics),
            child: Row(children: [
              const Icon(Icons.music_note, size: 13, color: Colors.white38),
              const SizedBox(width: 6),
              Text('Add your own lyrics (optional)',
                  style: TextStyle(color: Colors.white.withOpacity(0.45), fontSize: 12, fontWeight: FontWeight.w500)),
              const SizedBox(width: 4),
              Icon(_showLyrics ? Icons.expand_less : Icons.expand_more,
                  size: 16, color: Colors.white38),
            ]),
          ),
          if (_showLyrics) ...[
            const SizedBox(height: 8),
            buildTextArea(
              controller: _lyricsCtrl,
              placeholder: p.uiConfig['lyrics_placeholder'] ??
                  'Paste your lyrics — verses, chorus, bridge…\n\n[Verse 1]\n…\n[Chorus]\n…',
              maxLines: 6,
            ),
          ],
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Generate button ──
        ValueListenableBuilder(
          valueListenable: _promptCtrl,
          builder: (_, __, ___) => buildGenerateButton(
            label: 'Generate Music',
            enabled: _isValid && p.canAfford,
            isLoading: p.isLoading,
            onTap: _handleSubmit,
            gradientColors: const [Color(0xFFF59E0B), Color(0xFFF97316)],
            icon: Icons.library_music_rounded,
          ),
        ),
      ],
    );
  }
}
