// ─── Voice Studio Template ────────────────────────────────────────────────────
// Mirrors webapp VoiceStudio.tsx exactly.
// Supports: narrate, narrate-pro
// Payload: prompt (text), voice_id, language, extra_params { speed, format }

import 'package:flutter/material.dart';
import 'package:just_audio/just_audio.dart';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';

const _kBaseUrl = String.fromEnvironment(
  'API_URL', defaultValue: 'https://loyalty-nexus-api.onrender.com/api/v1');

const _kPreviewPhrase = 'Hello! I am your AI voice assistant, ready to bring your words to life.';

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

  // Voice preview (ElevenLabs-style)
  final AudioPlayer _previewPlayer = AudioPlayer();
  final Map<String, String> _previewCache = {}; // voiceId → audio URL
  String? _previewingVoiceId;
  bool _previewLoading = false;

  // Speech-to-text
  final stt.SpeechToText _speech = stt.SpeechToText();
  bool _micAvailable = false;
  bool _micListening = false;

  @override
  void initState() {
    super.initState();
    _initSpeech();
    _previewPlayer.playerStateStream.listen((state) {
      if (state.processingState == ProcessingState.completed) {
        if (mounted) setState(() => _previewingVoiceId = null);
      }
    });
  }

  Future<void> _previewVoice(String voiceId) async {
    // Stop if already playing this voice
    if (_previewingVoiceId == voiceId) {
      await _previewPlayer.stop();
      if (mounted) setState(() { _previewingVoiceId = null; _previewLoading = false; });
      return;
    }
    // Stop any other preview
    await _previewPlayer.stop();
    if (mounted) setState(() { _previewingVoiceId = voiceId; _previewLoading = true; });

    try {
      String? audioUrl = _previewCache[voiceId];
      if (audioUrl == null) {
        // Fetch from backend TTS preview endpoint (no points charged)
        const storage = FlutterSecureStorage();
        final token = await storage.read(key: 'nexus_token');
        final dio = Dio(BaseOptions(
          baseUrl: _kBaseUrl,
          connectTimeout: const Duration(seconds: 15),
          receiveTimeout: const Duration(seconds: 30),
          headers: {
            'Content-Type': 'application/json',
            if (token != null) 'Authorization': 'Bearer \$token',
          },
        ));
        final resp = await dio.post<Map<String, dynamic>>(
          '/studio/tts-preview',
          data: {'voice_id': voiceId, 'text': _kPreviewPhrase},
        );
        audioUrl = (resp.data?['output_url'] ?? resp.data?['url']) as String?;
        if (audioUrl != null) _previewCache[voiceId] = audioUrl;
      }
      if (audioUrl != null && mounted) {
        await _previewPlayer.setUrl(audioUrl);
        await _previewPlayer.play();
        if (mounted) setState(() => _previewLoading = false);
      } else {
        if (mounted) setState(() { _previewingVoiceId = null; _previewLoading = false; });
      }
    } catch (_) {
      if (mounted) setState(() { _previewingVoiceId = null; _previewLoading = false; });
    }
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
    _previewPlayer.dispose();
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
            color: catColor.withValues(alpha: 0.08),
            borderRadius: BorderRadius.circular(14),
            border: Border.all(color: catColor.withValues(alpha: 0.25)),
          ),
          child: Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: catColor.withValues(alpha: 0.15),
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
                      style: TextStyle(color: Colors.white.withValues(alpha: 0.55), fontSize: 12),
                    ),
                  ],
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: catColor.withValues(alpha: 0.15),
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
            hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 12),
            prefixIcon: Icon(Icons.search, size: 16, color: Colors.white.withValues(alpha: 0.3)),
            filled: true,
            fillColor: Colors.white.withValues(alpha: 0.04),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
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
              final isPreviewing = _previewingVoiceId == v['id'];
              final isPreviewLoading = _previewLoading && isPreviewing;
              return GestureDetector(
                onTap: () => setState(() => _voiceId = v['id']!),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 150),
                  padding: const EdgeInsets.only(left: 10, right: 4, top: 6, bottom: 6),
                  decoration: BoxDecoration(
                    color: isSelected ? vColor.withValues(alpha: 0.12) : Colors.white.withValues(alpha: 0.03),
                    borderRadius: BorderRadius.circular(10),
                    border: Border.all(
                      color: isSelected ? vColor.withValues(alpha: 0.5) : Colors.white.withValues(alpha: 0.08),
                      width: isSelected ? 1.5 : 1,
                    ),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Text(
                              v['name']!,
                              style: TextStyle(
                                color: isSelected ? Colors.white : Colors.white.withValues(alpha: 0.75),
                                fontWeight: FontWeight.w700,
                                fontSize: 12,
                              ),
                            ),
                            Text(
                              v['tone']!,
                              style: TextStyle(
                                color: isSelected ? vColor.withValues(alpha: 0.8) : Colors.white.withValues(alpha: 0.35),
                                fontSize: 10,
                              ),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ],
                        ),
                      ),
                      // ElevenLabs-style preview button
                      GestureDetector(
                        onTap: () => _previewVoice(v['id']!),
                        child: Container(
                          width: 26,
                          height: 26,
                          decoration: BoxDecoration(
                            color: isPreviewing
                                ? vColor.withValues(alpha: 0.25)
                                : Colors.white.withValues(alpha: 0.06),
                            shape: BoxShape.circle,
                          ),
                          child: isPreviewLoading
                              ? Padding(
                                  padding: const EdgeInsets.all(6),
                                  child: CircularProgressIndicator(
                                    strokeWidth: 1.5,
                                    color: vColor,
                                  ),
                                )
                              : Icon(
                                  isPreviewing ? Icons.stop_rounded : Icons.play_arrow_rounded,
                                  size: 14,
                                  color: isPreviewing ? vColor : Colors.white.withValues(alpha: 0.4),
                                ),
                        ),
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
              style: TextStyle(color: Colors.red.withValues(alpha: 0.7), fontSize: 12),
            ),
          ),
        ],
      ],
    );
  }
}
