import 'package:flutter/material.dart';
import 'template_types.dart';

const _defaultVoices = [
  {'id': 'alloy',   'name': 'Alloy',   'tone': 'Neutral & Clear',       'category': 'Conversational'},
  {'id': 'nova',    'name': 'Nova',    'tone': 'Friendly & Warm',       'category': 'Social Media'},
  {'id': 'echo',    'name': 'Echo',    'tone': 'Deep & Warm',           'category': 'Narration'},
  {'id': 'shimmer', 'name': 'Shimmer', 'tone': 'Soft & Soothing',      'category': 'Meditation'},
  {'id': 'onyx',    'name': 'Onyx',    'tone': 'Deep & Authoritative',  'category': 'Broadcast'},
  {'id': 'fable',   'name': 'Fable',   'tone': 'Expressive & Lively',   'category': 'Storytelling'},
  {'id': 'ash',     'name': 'Ash',     'tone': 'Gentle & Calm',         'category': 'Education'},
  {'id': 'ballad',  'name': 'Ballad',  'tone': 'Smooth & Musical',      'category': 'Entertainment'},
  {'id': 'coral',   'name': 'Coral',   'tone': 'Warm & Natural',        'category': 'Podcasts'},
  {'id': 'sage',    'name': 'Sage',    'tone': 'Clear & Professional',  'category': 'Corporate'},
  {'id': 'verse',   'name': 'Verse',   'tone': 'Dynamic & Engaging',    'category': 'Advertisement'},
  {'id': 'willow',  'name': 'Willow',  'tone': 'Soft & Thoughtful',     'category': 'Audiobooks'},
  {'id': 'jessica', 'name': 'Jessica', 'tone': 'Bright & Upbeat',       'category': 'Characters'},
];

const _defaultLanguages = [
  {'code': 'en', 'label': 'English'},
  {'code': 'yo', 'label': 'Yoruba'},
  {'code': 'ha', 'label': 'Hausa'},
  {'code': 'ig', 'label': 'Igbo'},
  {'code': 'fr', 'label': 'French'},
  {'code': 'pt', 'label': 'Portuguese'},
  {'code': 'es', 'label': 'Spanish'},
];

const _speedSteps = [
  {'label': '0.75×', 'value': 0.75},
  {'label': '1×',    'value': 1.0},
  {'label': '1.25×', 'value': 1.25},
  {'label': '1.5×',  'value': 1.5},
  {'label': '2×',    'value': 2.0},
];

const _formatOptions = [
  {'label': 'MP3', 'value': 'mp3', 'desc': 'Universal'},
  {'label': 'WAV', 'value': 'wav', 'desc': 'Lossless'},
];

const _catColors = {
  'Conversational': Color(0xFF38BDF8),
  'Social Media':   Color(0xFFF472B6),
  'Narration':      Color(0xFF60A5FA),
  'Meditation':     Color(0xFF2DD4BF),
  'Broadcast':      Color(0xFFFBBF24),
  'Storytelling':   Color(0xFFFB923C),
  'Education':      Color(0xFF4ADE80),
  'Entertainment':  Color(0xFFA78BFA),
  'Podcasts':       Color(0xFFFB7185),
  'Corporate':      Color(0xFF9CA3AF),
  'Advertisement':  Color(0xFFFDE047),
  'Audiobooks':     Color(0xFF818CF8),
  'Characters':     Color(0xFFE879F9),
};

class VoiceStudioTemplate extends StatefulWidget {
  final TemplateProps props;
  const VoiceStudioTemplate({super.key, required this.props});

  @override
  State<VoiceStudioTemplate> createState() => _VoiceStudioTemplateState();
}

class _VoiceStudioTemplateState extends State<VoiceStudioTemplate> {
  final _textCtrl         = TextEditingController();
  final _voiceSearchCtrl  = TextEditingController();
  String _voiceId    = 'nova';
  String _language   = 'en';
  double _speed      = 1.0;
  String _format     = 'mp3';

  TemplateProps get p => widget.props;
  int get _maxChars  => (p.uiConfig['max_chars'] ?? 5000) as int;
  bool get _showLang => p.uiConfig['show_language_selector'] != false;
  bool get _showSpeed => p.uiConfig['show_speed_control'] != false;
  bool get _showFormat => p.uiConfig['show_format_selector'] != false;

  List<Map<String, dynamic>> get _voices {
    final raw = p.uiConfig['voices'];
    if (raw is List) return raw.cast<Map<String, dynamic>>();
    return List<Map<String, dynamic>>.from(_defaultVoices);
  }

  List<Map<String, String>> get _languages {
    final raw = p.uiConfig['languages'];
    if (raw is List) return raw.cast<Map<String, String>>();
    return List<Map<String, String>>.from(_defaultLanguages);
  }

  List<Map<String, dynamic>> get _filteredVoices {
    final q = _voiceSearchCtrl.text.toLowerCase();
    if (q.isEmpty) return _voices;
    return _voices.where((v) =>
      v['name'].toString().toLowerCase().contains(q) ||
      v['category'].toString().toLowerCase().contains(q) ||
      v['tone'].toString().toLowerCase().contains(q),
    ).toList();
  }

  Map<String, dynamic>? get _selectedVoice =>
      _voices.where((v) => v['id'] == _voiceId).firstOrNull;

  bool get _isValid => _textCtrl.text.trim().length >= 5 &&
      _textCtrl.text.length <= _maxChars;

  void _handleSubmit() {
    if (!_isValid || p.isLoading || !p.canAfford) return;
    p.onSubmit(GeneratePayload(
      prompt:   _textCtrl.text.trim(),
      voiceId:  _voiceId,
      language: _showLang ? _language : null,
      extraParams: {
        if (_showSpeed)  'speed':  _speed,
        if (_showFormat) 'format': _format,
      },
    ));
  }

  @override
  void dispose() {
    _textCtrl.dispose();
    _voiceSearchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final voices        = _filteredVoices;
    final selectedVoice = _selectedVoice;
    final catColor      = _catColors[selectedVoice?['category']] ?? Colors.white38;
    final charPct       = _textCtrl.text.length / _maxChars;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [

        // ── Voice picker ──
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('VOICE', style: kLabelStyle),
            if (selectedVoice != null)
              Text(selectedVoice['category'] ?? '', style: TextStyle(
                fontSize: 10, fontWeight: FontWeight.w700, color: catColor,
              )),
          ],
        ),
        const SizedBox(height: 8),

        // Voice search
        if (_voices.length > 6) ...[
          TextField(
            controller: _voiceSearchCtrl,
            style: const TextStyle(color: Colors.white, fontSize: 12),
            onChanged: (_) => setState(() {}),
            decoration: InputDecoration(
              hintText: 'Search voices — name, tone, or use case…',
              hintStyle: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 11),
              prefixIcon: Icon(Icons.search, size: 15, color: Colors.white.withOpacity(0.3)),
              filled: true,
              fillColor: Colors.white.withOpacity(0.04),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(10),
                borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
              ),
              enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(10),
                borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(10),
                borderSide: const BorderSide(color: Color(0xFF059669), width: 1.5),
              ),
              contentPadding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
            ),
          ),
          const SizedBox(height: 8),
        ],

        // Voice grid (scrollable list of 2-column cards)
        SizedBox(
          height: 200,
          child: voices.isEmpty
              ? Center(child: Text(
                  'No voices match "${_voiceSearchCtrl.text}"',
                  style: kHintStyle,
                ))
              : GridView.builder(
                  itemCount: voices.length,
                  gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                    crossAxisCount: 2,
                    crossAxisSpacing: 6,
                    mainAxisSpacing: 6,
                    childAspectRatio: 3.2,
                  ),
                  itemBuilder: (_, i) {
                    final v   = voices[i];
                    final sel = _voiceId == v['id'];
                    return GestureDetector(
                      onTap: () => setState(() => _voiceId = v['id']),
                      child: AnimatedContainer(
                        duration: const Duration(milliseconds: 150),
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
                        decoration: BoxDecoration(
                          color: sel ? const Color(0xFF059669).withOpacity(0.15) : Colors.transparent,
                          borderRadius: BorderRadius.circular(12),
                          border: Border.all(
                            color: sel ? const Color(0xFF059669).withOpacity(0.6)
                                       : Colors.white.withOpacity(0.1),
                          ),
                        ),
                        child: Row(
                          children: [
                            Container(
                              width: 28, height: 28,
                              decoration: BoxDecoration(
                                color: sel ? const Color(0xFF059669) : Colors.white.withOpacity(0.1),
                                shape: BoxShape.circle,
                              ),
                              alignment: Alignment.center,
                              child: Text(
                                v['name'][0],
                                style: TextStyle(
                                  fontSize: 12, fontWeight: FontWeight.w800,
                                  color: sel ? Colors.white : Colors.white.withOpacity(0.5),
                                ),
                              ),
                            ),
                            const SizedBox(width: 6),
                            Expanded(child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              mainAxisAlignment: MainAxisAlignment.center,
                              children: [
                                Text(v['name'], overflow: TextOverflow.ellipsis,
                                    style: TextStyle(
                                      fontSize: 11, fontWeight: FontWeight.w700,
                                      color: sel ? const Color(0xFFD1FAE5) : Colors.white.withOpacity(0.7),
                                    )),
                                Text(v['tone'], overflow: TextOverflow.ellipsis,
                                    style: kHintStyle.copyWith(fontSize: 9)),
                              ],
                            )),
                          ],
                        ),
                      ),
                    );
                  },
                ),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Language ──
        if (_showLang) ...[
          buildSectionLabel('Language'),
          Wrap(
            spacing: 6, runSpacing: 6,
            children: _languages.map((l) => buildChip(
              label: l['label']!,
              selected: _language == l['code'],
              activeColor: const Color(0xFF059669),
              onTap: () => setState(() => _language = l['code']!),
            )).toList(),
          ),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Speed + Format ──
        if (_showSpeed || _showFormat) ...[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (_showSpeed) Expanded(child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  buildSectionLabel('Speed'),
                  Wrap(
                    spacing: 6, runSpacing: 6,
                    children: _speedSteps.map((s) {
                      final sel = _speed == s['value'];
                      return GestureDetector(
                        onTap: () => setState(() => _speed = s['value'] as double),
                        child: AnimatedContainer(
                          duration: const Duration(milliseconds: 150),
                          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                          decoration: BoxDecoration(
                            color: sel ? const Color(0xFF059669) : Colors.transparent,
                            borderRadius: BorderRadius.circular(999),
                            border: Border.all(
                              color: sel ? const Color(0xFF059669) : Colors.white.withOpacity(0.15),
                            ),
                          ),
                          child: Text(s['label'] as String, style: TextStyle(
                            fontSize: 11, fontWeight: FontWeight.w700,
                            color: sel ? Colors.white : Colors.white.withOpacity(0.55),
                          )),
                        ),
                      );
                    }).toList(),
                  ),
                ],
              )),
              if (_showSpeed && _showFormat) const SizedBox(width: 16),
              if (_showFormat) Expanded(child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  buildSectionLabel('Format'),
                  Wrap(
                    spacing: 6, runSpacing: 6,
                    children: _formatOptions.map((f) {
                      final sel = _format == f['value'];
                      return GestureDetector(
                        onTap: () => setState(() => _format = f['value'] as String),
                        child: AnimatedContainer(
                          duration: const Duration(milliseconds: 150),
                          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                          decoration: BoxDecoration(
                            color: sel ? const Color(0xFF059669) : Colors.transparent,
                            borderRadius: BorderRadius.circular(999),
                            border: Border.all(
                              color: sel ? const Color(0xFF059669) : Colors.white.withOpacity(0.15),
                            ),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Text(f['label'] as String, style: TextStyle(
                                fontSize: 11, fontWeight: FontWeight.w700,
                                color: sel ? Colors.white : Colors.white.withOpacity(0.55),
                              )),
                              const SizedBox(width: 4),
                              Text(f['desc'] as String, style: kHintStyle.copyWith(fontSize: 9)),
                            ],
                          ),
                        ),
                      );
                    }).toList(),
                  ),
                ],
              )),
            ],
          ),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Text to narrate ──
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('TEXT TO NARRATE', style: kLabelStyle),
            ValueListenableBuilder(
              valueListenable: _textCtrl,
              builder: (_, __, ___) => Text(
                '${_textCtrl.text.length.toString().padLeft(4)}/${_maxChars.toString()}',
                style: kHintStyle.copyWith(
                  color: _textCtrl.text.length > _maxChars * 0.9
                      ? Colors.redAccent
                      : Colors.white.withOpacity(0.3),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        buildTextArea(
          controller: _textCtrl,
          placeholder: p.uiConfig['prompt_placeholder'] ??
              'Paste or type the text you want narrated…',
          maxLines: 6,
          autoFocus: true,
          maxLength: _maxChars,
        ),
        const SizedBox(height: 6),
        // Char progress bar
        ValueListenableBuilder(
          valueListenable: _textCtrl,
          builder: (_, __, ___) {
            final pct = (_textCtrl.text.length / _maxChars).clamp(0.0, 1.0);
            return ClipRRect(
              borderRadius: BorderRadius.circular(2),
              child: LinearProgressIndicator(
                value: pct,
                backgroundColor: Colors.white.withOpacity(0.1),
                valueColor: AlwaysStoppedAnimation(
                  pct > 0.9 ? Colors.redAccent
                      : pct > 0.7 ? Colors.amber
                      : const Color(0xFF059669),
                ),
                minHeight: 3,
              ),
            );
          },
        ),
        const SizedBox(height: 4),
        if (selectedVoice != null)
          Row(children: [
            const Icon(Icons.play_circle_outline, size: 11, color: Colors.white24),
            const SizedBox(width: 4),
            Text('Will be narrated by ',
                style: kHintStyle),
            Text(selectedVoice['name'], style: kHintStyle.copyWith(
              fontWeight: FontWeight.w700, color: Colors.white.withOpacity(0.4),
            )),
            Text(' · ${selectedVoice['tone']}', style: kHintStyle),
          ]),
        const SizedBox(height: kTemplateSpacing),

        // ── Generate button ──
        ValueListenableBuilder(
          valueListenable: _textCtrl,
          builder: (_, __, ___) => buildGenerateButton(
            label: 'Generate Voice',
            enabled: _isValid && p.canAfford,
            isLoading: p.isLoading,
            onTap: _handleSubmit,
            gradientColors: const [Color(0xFF059669), Color(0xFF0D9488)],
            icon: Icons.record_voice_over_rounded,
          ),
        ),
      ],
    );
  }
}
