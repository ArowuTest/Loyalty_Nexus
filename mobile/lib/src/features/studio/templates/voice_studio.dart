// ─── Voice Studio Template ────────────────────────────────────────────────────
// Mirrors webapp VoiceStudio.tsx exactly.
// Supports: narrate, narrate-pro
// Payload: prompt (text), voice_id, language, extra_params { speed, format }

import 'package:flutter/material.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';

// ─── Constants ────────────────────────────────────────────────────────────────

const _defaultVoices = [
  {'id': 'alloy',   'name': 'Alloy',   'tone': 'Neutral & Clear',      'category': 'Conversational', 'gender': 'N'},
  {'id': 'nova',    'name': 'Nova',    'tone': 'Friendly & Warm',      'category': 'Social Media',   'gender': 'F'},
  {'id': 'echo',    'name': 'Echo',    'tone': 'Deep & Warm',          'category': 'Narration',      'gender': 'M'},
  {'id': 'shimmer', 'name': 'Shimmer', 'tone': 'Soft & Soothing',      'category': 'Meditation',     'gender': 'F'},
  {'id': 'onyx',    'name': 'Onyx',    'tone': 'Deep & Authoritative', 'category': 'Broadcast',      'gender': 'M'},
  {'id': 'fable',   'name': 'Fable',   'tone': 'Expressive & Lively',  'category': 'Storytelling',   'gender': 'M'},
  {'id': 'ash',     'name': 'Ash',     'tone': 'Gentle & Calm',        'category': 'Education',      'gender': 'N'},
  {'id': 'ballad',  'name': 'Ballad',  'tone': 'Smooth & Musical',     'category': 'Entertainment',  'gender': 'M'},
  {'id': 'coral',   'name': 'Coral',   'tone': 'Warm & Natural',       'category': 'Podcasts',       'gender': 'F'},
  {'id': 'sage',    'name': 'Sage',    'tone': 'Clear & Professional', 'category': 'Corporate',      'gender': 'N'},
  {'id': 'verse',   'name': 'Verse',   'tone': 'Dynamic & Engaging',   'category': 'Advertisement',  'gender': 'M'},
  {'id': 'willow',  'name': 'Willow',  'tone': 'Soft & Thoughtful',    'category': 'Audiobooks',     'gender': 'F'},
  {'id': 'jessica', 'name': 'Jessica', 'tone': 'Bright & Upbeat',      'category': 'Characters',     'gender': 'F'},
];

const _defaultLanguages = [
  {'code': 'en', 'label': 'English',    'flag': '🇬🇧'},
  {'code': 'yo', 'label': 'Yoruba',     'flag': '🇳🇬'},
  {'code': 'ha', 'label': 'Hausa',      'flag': '🇳🇬'},
  {'code': 'ig', 'label': 'Igbo',       'flag': '🇳🇬'},
  {'code': 'fr', 'label': 'French',     'flag': '🇫🇷'},
  {'code': 'pt', 'label': 'Portuguese', 'flag': '🇵🇹'},
  {'code': 'es', 'label': 'Spanish',    'flag': '🇪🇸'},
];

const _speedSteps = [0.75, 1.0, 1.25, 1.5, 2.0];
const _speedLabels = ['0.75×', '1×', '1.25×', '1.5×', '2×'];

const _catColors = {
  'Conversational': Color(0xFF0EA5E9),
  'Social Media':   Color(0xFFEC4899),
  'Narration':      Color(0xFF8B5CF6),
  'Meditation':     Color(0xFF10B981),
  'Broadcast':      Color(0xFFF59E0B),
  'Storytelling':   Color(0xFFEF4444),
  'Education':      Color(0xFF06B6D4),
  'Entertainment':  Color(0xFFD946EF),
  'Podcasts':       Color(0xFF84CC16),
  'Corporate':      Color(0xFF64748B),
  'Advertisement':  Color(0xFFF97316),
  'Audiobooks':     Color(0xFFA78BFA),
  'Characters':     Color(0xFFFBBF24),
};

// ─── Widget ───────────────────────────────────────────────────────────────────

class VoiceStudioTemplate extends StatefulWidget {
  final TemplateProps props;
  const VoiceStudioTemplate({super.key, required this.props});

  @override
  State<VoiceStudioTemplate> createState() => _VoiceStudioTemplateState();
}

class _VoiceStudioTemplateState extends State<VoiceStudioTemplate> {
  final _textCtrl   = TextEditingController();
  final _filterCtrl = TextEditingController();

  String _voiceId  = 'nova';
  String _language = 'en';
  double _speed    = 1.0;
  String _format   = 'mp3';

  // Speech-to-text
  final stt.SpeechToText _speech = stt.SpeechToText();
  bool _micAvailable = false;
  bool _micListening = false;

  @override
  void initState() {
    super.initState();
    _initSpeech();
  }

  Future<void> _initSpeech() async {
    final available = await _speech.initialize();
    if (mounted) setState(() => _micAvailable = available);
  }

  void _toggleMic() async {
    if (!_micAvailable) return;
    if (_speech.isListening) {
      await _speech.stop();
      setState(() => _micListening = false);
      return;
    }
    setState(() => _micListening = true);
    await _speech.listen(
      onResult: (result) {
        if (result.finalResult) {
          setState(() {
            _textCtrl.text = result.recognizedWords;
            _micListening = false;
          });
        }
      },
      localeId: 'en_NG',
    );
  }

  void _handleSubmit() {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _textCtrl.text.trim().isEmpty) return;
    final payload = GeneratePayload(
      prompt: _textCtrl.text.trim(),
      voiceId: _voiceId,
      language: _language,
      extraParams: {
        'speed':  _speed,
        'format': _format,
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _textCtrl.dispose();
    _filterCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final isValid = _textCtrl.text.trim().isNotEmpty;
    final filter = _filterCtrl.text.toLowerCase();
    final voices = filter.isEmpty
        ? _defaultVoices
        : _defaultVoices.where((v) =>
            v['name']!.toLowerCase().contains(filter) ||
            v['category']!.toLowerCase().contains(filter) ||
            v['tone']!.toLowerCase().contains(filter)).toList();

    final selectedVoice = _defaultVoices.firstWhere(
      (v) => v['id'] == _voiceId,
      orElse: () => _defaultVoices.first,
    );
    final catColor = _catColors[selectedVoice['category']] ?? const Color(0xFF7C3AED);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ── Provider badge ──
        ProviderBadge(
          label: widget.props.slug.contains('pro') ? 'Pollinations TTS Full' : 'Google Cloud TTS',
          description: 'Natural voice synthesis with ${_defaultVoices.length} voices',
          color: const Color(0xFF8B5CF6),
          icon: Icons.record_voice_over_rounded,
        ),
        const SizedBox(height: 16),

        // ── Selected voice hero ──
        Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: catColor.withOpacity(0.08),
            borderRadius: BorderRadius.circular(14),
            border: Border.all(color: catColor.withOpacity(0.25)),
          ),
          child: Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: catColor.withOpacity(0.15),
                  shape: BoxShape.circle,
                ),
                child: Center(
                  child: Text(
                    selectedVoice['name']![0],
                    style: TextStyle(
                      color: catColor,
                      fontSize: 18,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      selectedVoice['name']!,
                      style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 15),
                    ),
                    Text(
                      selectedVoice['tone']!,
                      style: TextStyle(color: Colors.white.withOpacity(0.55), fontSize: 12),
                    ),
                  ],
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: catColor.withOpacity(0.15),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Text(
                  selectedVoice['category']!,
                  style: TextStyle(color: catColor, fontSize: 10, fontWeight: FontWeight.w700),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),

        // ── Voice search ──
        buildSectionLabel('Select Voice (${_defaultVoices.length} available)'),
        TextField(
          controller: _filterCtrl,
          style: const TextStyle(color: Colors.white, fontSize: 13),
          decoration: InputDecoration(
            hintText: 'Search voices…',
            hintStyle: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 12),
            prefixIcon: Icon(Icons.search, size: 16, color: Colors.white.withOpacity(0.3)),
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
              borderSide: const BorderSide(color: Color(0xFF8B5CF6), width: 1.5),
            ),
            contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          ),
          onChanged: (_) => setState(() {}),
        ),
        const SizedBox(height: 10),

        // ── Voice grid ──
        SizedBox(
          height: 200,
          child: GridView.builder(
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              childAspectRatio: 2.8,
              crossAxisSpacing: 8,
              mainAxisSpacing: 8,
            ),
            itemCount: voices.length,
            itemBuilder: (context, i) {
              final v = voices[i];
              final isSelected = _voiceId == v['id'];
              final vColor = _catColors[v['category']] ?? const Color(0xFF7C3AED);
              return GestureDetector(
                onTap: () => setState(() => _voiceId = v['id']!),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 150),
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                  decoration: BoxDecoration(
                    color: isSelected ? vColor.withOpacity(0.12) : Colors.white.withOpacity(0.03),
                    borderRadius: BorderRadius.circular(10),
                    border: Border.all(
                      color: isSelected ? vColor.withOpacity(0.5) : Colors.white.withOpacity(0.08),
                      width: isSelected ? 1.5 : 1,
                    ),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Text(
                        v['name']!,
                        style: TextStyle(
                          color: isSelected ? Colors.white : Colors.white.withOpacity(0.75),
                          fontWeight: FontWeight.w700,
                          fontSize: 12,
                        ),
                      ),
                      Text(
                        v['tone']!,
                        style: TextStyle(
                          color: isSelected ? vColor.withOpacity(0.8) : Colors.white.withOpacity(0.35),
                          fontSize: 10,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ),
        const SizedBox(height: 16),

        // ── Text to narrate ──
        buildSectionLabel('Text to Narrate'),
        buildTextArea(
          controller: _textCtrl,
          placeholder: 'Enter the text you want to convert to speech…',
          maxLines: 5,
          maxLength: 4000,
          onMicTap: _micAvailable ? _toggleMic : null,
          micActive: _micListening,
        ),
        const SizedBox(height: 16),

        // ── Language ──
        buildSectionLabel('Language'),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: _defaultLanguages.map((lang) => Padding(
              padding: const EdgeInsets.only(right: 8),
              child: buildChip(
                label: '${lang['flag']} ${lang['label']}',
                selected: _language == lang['code'],
                onTap: () => setState(() => _language = lang['code']!),
                activeColor: const Color(0xFF8B5CF6),
              ),
            )).toList(),
          ),
        ),
        const SizedBox(height: 16),

        // ── Speed ──
        buildSectionLabel('Speed — ${_speedLabels[_speedSteps.indexOf(_speed)]}'),
        Row(
          children: List.generate(_speedSteps.length, (i) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: buildChip(
              label: _speedLabels[i],
              selected: _speed == _speedSteps[i],
              onTap: () => setState(() => _speed = _speedSteps[i]),
            ),
          )),
        ),
        const SizedBox(height: 16),

        // ── Format ──
        buildSectionLabel('Output Format'),
        Row(
          children: [
            buildChip(label: 'MP3', selected: _format == 'mp3', onTap: () => setState(() => _format = 'mp3')),
            const SizedBox(width: 8),
            buildChip(label: 'WAV', selected: _format == 'wav', onTap: () => setState(() => _format = 'wav')),
          ],
        ),
        const SizedBox(height: 24),

        // ── Generate button ──
        buildGenerateButton(
          label: p.isLoading
              ? 'Generating…'
              : 'Generate Audio${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF7C3AED), Color(0xFF8B5CF6)],
          icon: Icons.record_voice_over_rounded,
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
}
