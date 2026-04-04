// ─── Video Creator Template ───────────────────────────────────────────────────
// Mirrors webapp VideoCreator.tsx exactly.
// Supports: video-creator, video-short, video-cinematic
// Payload: prompt, aspect_ratio, duration, style_tags, negative_prompt,
//          image_url (optional start frame),
//          extra_params { camera_movement, scenes, audio_direction, music_style }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

// ─── Constants ────────────────────────────────────────────────────────────────

const _defaultAspectRatios = [
  {'label': 'Landscape', 'value': '16:9',  'icon': '🖥️'},
  {'label': 'Portrait',  'value': '9:16',  'icon': '📱'},
  {'label': 'Square',    'value': '1:1',   'icon': '⬜'},
  {'label': 'Cinematic', 'value': '21:9',  'icon': '🎬'},
];

const _defaultStyleTags = [
  'Cinematic', 'Documentary', 'Slow motion', 'Time-lapse',
  'Aerial drone', 'Dark', 'Vibrant', 'Vintage film', 'Sci-Fi', 'Fantasy',
];

const _defaultDurations = [5, 8, 10, 15, 30];

const _cameraMovements = [
  {'label': 'Slow zoom in',  'icon': '🔍', 'value': 'slow zoom in'},
  {'label': 'Slow zoom out', 'icon': '🔭', 'value': 'slow zoom out'},
  {'label': 'Pan left',      'icon': '⬅️', 'value': 'camera panning left'},
  {'label': 'Pan right',     'icon': '➡️', 'value': 'camera panning right'},
  {'label': 'Tilt up',       'icon': '⬆️', 'value': 'camera tilting up'},
  {'label': 'Tilt down',     'icon': '⬇️', 'value': 'camera tilting down'},
  {'label': 'Orbit',         'icon': '🔄', 'value': 'orbital camera movement'},
  {'label': 'Handheld',      'icon': '🎥', 'value': 'handheld camera shake'},
  {'label': 'Dolly in',      'icon': '🚂', 'value': 'dolly in'},
  {'label': 'Static',        'icon': '🔒', 'value': 'static camera'},
];

const _inspirations = [
  'A majestic waterfall in a tropical rainforest, slow motion, cinematic',
  'Lagos skyline at night, aerial drone shot, neon lights, 4K',
  'A cheetah running across the savanna at sunset, slow motion',
];

// ─── Widget ───────────────────────────────────────────────────────────────────

class VideoCreatorTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const VideoCreatorTemplate({super.key, required this.props});

  @override
  ConsumerState<VideoCreatorTemplate> createState() => _VideoCreatorTemplateState();
}

class _VideoCreatorTemplateState extends ConsumerState<VideoCreatorTemplate> {
  final _promptCtrl    = TextEditingController();
  final _negCtrl       = TextEditingController();
  final _audioDirCtrl  = TextEditingController();
  final _musicCtrl     = TextEditingController();

  String _aspect       = '16:9';
  int    _duration     = 8;
  String? _cameraMove;
  final List<String> _selectedStyles = [];
  final List<TextEditingController> _sceneCtrlList = [];

  // Start-frame image upload
  String? _imageUrl;
  String? _imagePreview;
  bool    _isImageUploading = false;
  String? _imageUploadError;

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
            _promptCtrl.text = result.recognizedWords;
            _micListening = false;
          });
        }
      },
      localeId: 'en_NG',
    );
  }

  Future<void> _pickStartFrame() async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(source: ImageSource.gallery, imageQuality: 85);
    if (picked == null) return;
    setState(() { _imagePreview = picked.path; _isImageUploading = true; _imageUploadError = null; });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(File(picked.path));
      setState(() { _imageUrl = url; _isImageUploading = false; });
    } catch (e) {
      setState(() { _imageUploadError = 'Upload failed.'; _isImageUploading = false; });
    }
  }

  void _clearStartFrame() => setState(() { _imageUrl = null; _imagePreview = null; _imageUploadError = null; });

  void _addScene() {
    if (_sceneCtrlList.length >= 5) return;
    setState(() => _sceneCtrlList.add(TextEditingController()));
  }

  void _removeScene(int i) {
    _sceneCtrlList[i].dispose();
    setState(() => _sceneCtrlList.removeAt(i));
  }

  void _handleSubmit() {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _promptCtrl.text.trim().isEmpty) return;
    final selStyles = _selectedStyles;
    final stylePrefix = selStyles.isNotEmpty ? '[${selStyles.join(', ')}] ' : '';
    final cameraSuffix = _cameraMove != null ? '. Camera: $_cameraMove.' : '';
    final scenes = _sceneCtrlList.map((c) => c.text.trim()).where((s) => s.isNotEmpty).toList();
    final scenesSuffix = scenes.isNotEmpty
        ? '\n\nScene breakdown:\n${scenes.asMap().entries.map((e) => 'Scene ${e.key + 1}: ${e.value}').join('\n')}'
        : '';
    final audioDirSuffix = _audioDirCtrl.text.trim().isNotEmpty ? '. Audio: ${_audioDirCtrl.text.trim()}.' : '';

    final payload = GeneratePayload(
      prompt: stylePrefix + _promptCtrl.text.trim() + cameraSuffix + scenesSuffix + audioDirSuffix,
      aspectRatio: _aspect,
      duration: _duration,
      styleTags: selStyles.isNotEmpty ? List.from(selStyles) : null,
      negativePrompt: _negCtrl.text.trim().isNotEmpty ? _negCtrl.text.trim() : null,
      imageUrl: _imageUrl,
      extraParams: {
        if (_cameraMove != null) 'camera_movement': _cameraMove,
        if (scenes.isNotEmpty) 'scenes': scenes,
        if (_audioDirCtrl.text.trim().isNotEmpty) 'audio_direction': _audioDirCtrl.text.trim(),
        if (_musicCtrl.text.trim().isNotEmpty) 'music_style': _musicCtrl.text.trim(),
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _promptCtrl.dispose();
    _negCtrl.dispose();
    _audioDirCtrl.dispose();
    _musicCtrl.dispose();
    for (final c in _sceneCtrlList) c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final isValid = _promptCtrl.text.trim().isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ── Warning ──
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: const Color(0xFFF59E0B).withOpacity(0.08),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: const Color(0xFFF59E0B).withOpacity(0.2)),
          ),
          child: Row(
            children: [
              const Icon(Icons.warning_amber_rounded, size: 13, color: Color(0xFFF59E0B)),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  'Video generation takes 2–5 minutes. You\'ll be notified when it\'s ready.',
                  style: TextStyle(color: const Color(0xFFF59E0B).withOpacity(0.8), fontSize: 11, height: 1.4),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),

        // ── Provider badge ──
        const ProviderBadge(
          label: 'Grok / Kling',
          description: 'High-quality AI video generation',
          color: Color(0xFFEF4444),
          icon: Icons.videocam_rounded,
        ),
        const SizedBox(height: 16),

        // ── Prompt ──
        buildSectionLabel('Describe your video'),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: 'e.g. A majestic waterfall in a tropical rainforest, slow motion, cinematic…',
          maxLines: 4,
          maxLength: 1000,
          onMicTap: _micAvailable ? _toggleMic : null,
          micActive: _micListening,
        ),
        const SizedBox(height: 8),

        // ── Inspirations ──
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: _inspirations.map((insp) => Padding(
              padding: const EdgeInsets.only(right: 8),
              child: GestureDetector(
                onTap: () => setState(() => _promptCtrl.text = insp),
                child: Container(
                  constraints: const BoxConstraints(maxWidth: 200),
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  decoration: BoxDecoration(
                    color: Colors.white.withOpacity(0.04),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.white.withOpacity(0.1)),
                  ),
                  child: Text(
                    insp.length > 50 ? '${insp.substring(0, 50)}…' : insp,
                    style: TextStyle(color: Colors.white.withOpacity(0.45), fontSize: 11),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ),
            )).toList(),
          ),
        ),
        const SizedBox(height: 16),

        // ── Style tags ──
        buildSectionLabel('Style Tags (optional)'),
        buildChipRow(
          options: _defaultStyleTags,
          selected: _selectedStyles,
          onToggle: (tag) => setState(() {
            if (_selectedStyles.contains(tag)) _selectedStyles.remove(tag);
            else if (_selectedStyles.length < 3) _selectedStyles.add(tag);
          }),
          activeColor: const Color(0xFFEF4444),
        ),
        const SizedBox(height: 16),

        // ── Aspect ratio ──
        buildSectionLabel('Aspect Ratio'),
        AspectRatioSelector(
          options: _defaultAspectRatios.cast<Map<String, String>>(),
          selected: _aspect,
          onSelect: (v) => setState(() => _aspect = v),
        ),
        const SizedBox(height: 16),

        // ── Duration ──
        buildSectionLabel('Duration'),
        Row(
          children: _defaultDurations.map((d) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: buildChip(
              label: '${d}s',
              selected: _duration == d,
              onTap: () => setState(() => _duration = d),
              activeColor: const Color(0xFFEF4444),
            ),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Camera movement ──
        buildSectionLabel('Camera Movement (optional)'),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: _cameraMovements.map((cm) => Padding(
              padding: const EdgeInsets.only(right: 8),
              child: buildChip(
                label: '${cm['icon']} ${cm['label']}',
                selected: _cameraMove == cm['value'],
                onTap: () => setState(() =>
                    _cameraMove = _cameraMove == cm['value'] ? null : cm['value']),
                activeColor: const Color(0xFFEF4444),
              ),
            )).toList(),
          ),
        ),
        const SizedBox(height: 16),

        // ── Start frame image ──
        CollapsibleSection(
          title: 'Start Frame Image (optional)',
          child: Column(
            children: [
              UploadZone(
                label: 'Upload start frame',
                sublabel: 'First frame of your video',
                icon: Icons.image_outlined,
                previewUrl: _imageUrl,
                isUploading: _isImageUploading,
                error: _imageUploadError,
                onTap: _pickStartFrame,
                onClear: _clearStartFrame,
                height: 120,
                accentColor: const Color(0xFFEF4444),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),

        // ── Scene breakdown ──
        CollapsibleSection(
          title: 'Scene Breakdown (optional)',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              ..._sceneCtrlList.asMap().entries.map((e) => Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Row(
                  children: [
                    Container(
                      width: 24,
                      height: 24,
                      decoration: BoxDecoration(
                        color: const Color(0xFFEF4444).withOpacity(0.15),
                        shape: BoxShape.circle,
                      ),
                      child: Center(
                        child: Text(
                          '${e.key + 1}',
                          style: const TextStyle(color: Color(0xFFEF4444), fontSize: 11, fontWeight: FontWeight.w700),
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: TextField(
                        controller: e.value,
                        style: const TextStyle(color: Colors.white, fontSize: 13),
                        decoration: InputDecoration(
                          hintText: 'Describe scene ${e.key + 1}…',
                          hintStyle: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 12),
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
                            borderSide: const BorderSide(color: Color(0xFFEF4444), width: 1.5),
                          ),
                          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    GestureDetector(
                      onTap: () => _removeScene(e.key),
                      child: Icon(Icons.remove_circle_outline, size: 18, color: Colors.white.withOpacity(0.3)),
                    ),
                  ],
                ),
              )),
              if (_sceneCtrlList.length < 5)
                GestureDetector(
                  onTap: _addScene,
                  child: Container(
                    padding: const EdgeInsets.symmetric(vertical: 10),
                    decoration: BoxDecoration(
                      color: Colors.white.withOpacity(0.03),
                      borderRadius: BorderRadius.circular(10),
                      border: Border.all(color: Colors.white.withOpacity(0.1)),
                    ),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(Icons.add, size: 14, color: Colors.white.withOpacity(0.4)),
                        const SizedBox(width: 6),
                        Text('Add Scene', style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 12)),
                      ],
                    ),
                  ),
                ),
            ],
          ),
        ),
        const SizedBox(height: 16),

        // ── Audio direction ──
        CollapsibleSection(
          title: 'Audio Direction (optional)',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              buildTextArea(
                controller: _audioDirCtrl,
                placeholder: 'e.g. Epic orchestral music, building tension…',
                maxLines: 2,
              ),
              const SizedBox(height: 10),
              buildSectionLabel('Music Style'),
              buildTextArea(
                controller: _musicCtrl,
                placeholder: 'e.g. Afrobeats, cinematic score, lo-fi…',
                maxLines: 1,
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),

        // ── Negative prompt ──
        CollapsibleSection(
          title: 'Negative Prompt (optional)',
          child: buildTextArea(
            controller: _negCtrl,
            placeholder: 'Things to avoid: blurry, low quality, watermark…',
            maxLines: 2,
          ),
        ),
        const SizedBox(height: 24),

        // ── Generate button ──
        buildGenerateButton(
          label: p.isLoading
              ? 'Generating…'
              : 'Generate Video${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFFDC2626), Color(0xFFEF4444)],
          icon: Icons.videocam_rounded,
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
