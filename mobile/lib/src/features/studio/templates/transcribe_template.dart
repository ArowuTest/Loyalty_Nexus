import 'package:flutter/material.dart';
import 'template_types.dart';

const _defaultLanguages = [
  {'code': 'auto', 'label': 'Auto-detect'},
  {'code': 'en',   'label': 'English'},
  {'code': 'yo',   'label': 'Yoruba'},
  {'code': 'ha',   'label': 'Hausa'},
  {'code': 'ig',   'label': 'Igbo'},
  {'code': 'fr',   'label': 'French'},
  {'code': 'pcm',  'label': 'Pidgin'},
];

const _outputFormats = [
  {'value': 'plain',       'label': 'Plain text',    'desc': 'Clean transcript, no timestamps'},
  {'value': 'timestamped', 'label': 'Timestamped',   'desc': 'With time markers per sentence'},
  {'value': 'srt',         'label': 'SRT Subtitles', 'desc': 'Ready to use as subtitles'},
];

class TranscribeTemplate extends StatefulWidget {
  final TemplateProps props;
  const TranscribeTemplate({super.key, required this.props});

  @override
  State<TranscribeTemplate> createState() => _TranscribeTemplateState();
}

class _TranscribeTemplateState extends State<TranscribeTemplate> {
  final _urlCtrl    = TextEditingController();
  bool  _hasUrl     = false;
  String _language  = 'auto';
  bool   _speakLabels = true;
  String _outFormat   = 'plain';

  TemplateProps get p => widget.props;

  List<Map<String, String>> get _languages {
    final raw = p.uiConfig['languages'];
    if (raw is List) return raw.cast<Map<String, String>>();
    return List<Map<String, String>>.from(_defaultLanguages);
  }

  bool get _showLang     => p.uiConfig['show_language_selector'] != false;
  bool get _showSpeakers => p.uiConfig['show_speaker_labels']    != false;
  bool get _showFormat   => p.uiConfig['show_output_format']     != false;

  bool get _isValid => _hasUrl && _urlCtrl.text.trim().isNotEmpty;

  void _handleSubmit() {
    if (!_isValid || p.isLoading || !p.canAfford) return;
    p.onSubmit(GeneratePayload(
      prompt:    _urlCtrl.text.trim(),
      language:  _showLang ? _language : null,
      extraParams: {
        'speaker_labels': _showSpeakers ? _speakLabels : false,
        'output_format':  _outFormat,
      },
    ));
  }

  @override
  void dispose() {
    _urlCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [

        // ── Language ──
        if (_showLang) ...[
          buildSectionLabel('Language'),
          Wrap(
            spacing: 6, runSpacing: 6,
            children: _languages.map((l) => buildChip(
              label: l['label']!,
              selected: _language == l['code'],
              activeColor: const Color(0xFFEA580C),
              onTap: () => setState(() => _language = l['code']!),
            )).toList(),
          ),
          const SizedBox(height: 4),
          RichText(text: TextSpan(children: [
            TextSpan(text: 'Select ', style: kHintStyle),
            TextSpan(text: 'Auto-detect',
                style: kHintStyle.copyWith(fontWeight: FontWeight.w700, color: Colors.white.withOpacity(0.4))),
            TextSpan(text: ' if unsure — or pick the language for better accuracy', style: kHintStyle),
          ])),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Output format grid ──
        if (_showFormat) ...[
          buildSectionLabel('Output Format'),
          Row(
            children: _outputFormats.map((f) {
              final sel = _outFormat == f['value'];
              return Expanded(
                child: GestureDetector(
                  onTap: () => setState(() => _outFormat = f['value']!),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 150),
                    margin: const EdgeInsets.only(right: 6),
                    padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 8),
                    decoration: BoxDecoration(
                      color: sel ? const Color(0xFFEA580C).withOpacity(0.2) : Colors.transparent,
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(
                        color: sel ? const Color(0xFFEA580C).withOpacity(0.6)
                                   : Colors.white.withOpacity(0.1),
                      ),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(f['label']!,
                            style: TextStyle(
                              fontSize: 10, fontWeight: FontWeight.w700,
                              color: sel ? const Color(0xFFFED7AA) : Colors.white.withOpacity(0.55),
                            )),
                        const SizedBox(height: 2),
                        Text(f['desc']!,
                            style: kHintStyle.copyWith(fontSize: 8)),
                      ],
                    ),
                  ),
                ),
              );
            }).toList(),
          ),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Speaker labels toggle ──
        if (_showSpeakers) ...[
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.03),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.white.withOpacity(0.08)),
            ),
            child: Row(
              children: [
                Icon(Icons.people_outline, size: 15, color: Colors.white.withOpacity(0.4)),
                const SizedBox(width: 10),
                Expanded(child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Speaker labels',
                        style: TextStyle(color: Colors.white.withOpacity(0.7),
                            fontSize: 12, fontWeight: FontWeight.w700)),
                    Text('Identify who is talking (Speaker A, Speaker B…)',
                        style: kHintStyle.copyWith(fontSize: 10)),
                  ],
                )),
                GestureDetector(
                  onTap: () => setState(() => _speakLabels = !_speakLabels),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 200),
                    width: 40, height: 22,
                    padding: const EdgeInsets.all(2),
                    decoration: BoxDecoration(
                      color: _speakLabels ? const Color(0xFFEA580C) : Colors.white.withOpacity(0.15),
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: AnimatedAlign(
                      duration: const Duration(milliseconds: 200),
                      alignment: _speakLabels ? Alignment.centerRight : Alignment.centerLeft,
                      child: Container(
                        width: 18, height: 18,
                        decoration: const BoxDecoration(color: Colors.white, shape: BoxShape.circle),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Audio URL ──
        buildSectionLabel(p.uiConfig['upload_label']?.toString() ?? 'Audio File URL'),
        const SizedBox(height: 8),

        if (!_hasUrl) ...[
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              border: Border.all(
                color: const Color(0xFFEA580C).withOpacity(0.3),
                style: BorderStyle.solid,
              ),
              borderRadius: BorderRadius.circular(14),
              color: const Color(0xFFEA580C).withOpacity(0.04),
            ),
            child: Column(
              children: [
                Icon(Icons.mic_outlined, size: 36, color: Colors.white.withOpacity(0.3)),
                const SizedBox(height: 8),
                Text('Paste audio URL below',
                    style: TextStyle(color: Colors.white.withOpacity(0.65), fontSize: 13)),
                const SizedBox(height: 4),
                Text('MP3, WAV, M4A, FLAC, OGG · up to ${p.uiConfig['max_duration_mins'] ?? 60} min',
                    style: kHintStyle),
              ],
            ),
          ),
          const SizedBox(height: 10),
          TextField(
            controller: _urlCtrl,
            keyboardType: TextInputType.url,
            style: const TextStyle(color: Colors.white, fontSize: 13),
            decoration: InputDecoration(
              hintText: 'https://example.com/recording.mp3',
              hintStyle: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 12),
              filled: true,
              fillColor: Colors.white.withOpacity(0.04),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
              ),
              enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: const BorderSide(color: Color(0xFFEA580C), width: 1.5),
              ),
              contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
              suffixIcon: IconButton(
                icon: const Icon(Icons.arrow_forward, color: Color(0xFFEA580C)),
                onPressed: () {
                  if (_urlCtrl.text.trim().isNotEmpty) setState(() => _hasUrl = true);
                },
              ),
            ),
            onSubmitted: (_) {
              if (_urlCtrl.text.trim().isNotEmpty) setState(() => _hasUrl = true);
            },
          ),
        ] else ...[
          // Audio file selected card
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            decoration: BoxDecoration(
              color: const Color(0xFFEA580C).withOpacity(0.08),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: const Color(0xFFEA580C).withOpacity(0.2)),
            ),
            child: Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: const Color(0xFFEA580C).withOpacity(0.15),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: const Icon(Icons.audio_file_rounded, size: 18, color: Color(0xFFFB923C)),
                ),
                const SizedBox(width: 12),
                Expanded(child: Text(
                  _urlCtrl.text.trim(),
                  style: TextStyle(color: Colors.white.withOpacity(0.75), fontSize: 12),
                  overflow: TextOverflow.ellipsis,
                )),
                GestureDetector(
                  onTap: () => setState(() { _hasUrl = false; _urlCtrl.clear(); }),
                  child: Icon(Icons.close, size: 16, color: Colors.white.withOpacity(0.4)),
                ),
              ],
            ),
          ),
        ],
        const SizedBox(height: kTemplateSpacing),

        // ── Transcribe button ──
        buildGenerateButton(
          label: 'Transcribe Audio',
          enabled: _isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFFEA580C), Color(0xFFD97706)],
          icon: Icons.subtitles_rounded,
        ),

        // ── Output hint ──
        if (p.uiConfig['output_hint'] != null) ...[
          const SizedBox(height: 8),
          Text(p.uiConfig['output_hint'].toString(), style: kHintStyle, textAlign: TextAlign.center),
        ],
      ],
    );
  }
}
