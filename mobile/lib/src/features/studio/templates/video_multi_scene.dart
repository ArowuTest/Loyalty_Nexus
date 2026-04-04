// ─── Video Multi-Scene Template ───────────────────────────────────────────────
// Mirrors webapp VideoMultiScene.tsx exactly.
// Supports: video-multi-scene, video-story
// Payload: prompt (overall story), aspect_ratio, duration,
//          extra_params { scenes: [{description, duration}], style, music_style }

import 'package:flutter/material.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';

const _defaultStyles = [
  {'value': 'cinematic',    'label': 'Cinematic',    'icon': '🎬'},
  {'value': 'documentary',  'label': 'Documentary',  'icon': '📹'},
  {'value': 'commercial',   'label': 'Commercial',   'icon': '📢'},
  {'value': 'social-media', 'label': 'Social Media', 'icon': '📱'},
  {'value': 'music-video',  'label': 'Music Video',  'icon': '🎵'},
  {'value': 'short-film',   'label': 'Short Film',   'icon': '🎥'},
];

const _defaultAspectRatios = [
  {'label': 'Landscape', 'value': '16:9'},
  {'label': 'Portrait',  'value': '9:16'},
  {'label': 'Square',    'value': '1:1'},
];

const _musicStyles = [
  'Afrobeats', 'Cinematic', 'Lo-fi', 'Electronic', 'Gospel', 'Hip-Hop', 'None',
];

class VideoMultiSceneTemplate extends StatefulWidget {
  final TemplateProps props;
  const VideoMultiSceneTemplate({super.key, required this.props});

  @override
  State<VideoMultiSceneTemplate> createState() => _VideoMultiSceneTemplateState();
}

class _VideoMultiSceneTemplateState extends State<VideoMultiSceneTemplate> {
  final _storyCtrl  = TextEditingController();

  String _style       = 'cinematic';
  String _aspect      = '16:9';
  String _musicStyle  = 'Cinematic';

  final List<_SceneData> _scenes = [
    _SceneData(),
    _SceneData(),
  ];

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
          setState(() { _storyCtrl.text = result.recognizedWords; _micListening = false; });
        }
      },
    );
  }

  void _addScene() {
    if (_scenes.length >= 8) return;
    setState(() => _scenes.add(_SceneData()));
  }

  void _removeScene(int i) {
    if (_scenes.length <= 1) return;
    _scenes[i].ctrl.dispose();
    setState(() => _scenes.removeAt(i));
  }

  void _handleSubmit() {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _storyCtrl.text.trim().isEmpty) return;
    final scenes = _scenes
        .where((s) => s.ctrl.text.trim().isNotEmpty)
        .map((s) => {'description': s.ctrl.text.trim(), 'duration': s.duration})
        .toList();
    final payload = GeneratePayload(
      prompt: _storyCtrl.text.trim(),
      aspectRatio: _aspect,
      extraParams: {
        'scenes':       scenes,
        'style':        _style,
        'music_style':  _musicStyle == 'None' ? '' : _musicStyle,
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _storyCtrl.dispose();
    for (final s in _scenes) { s.ctrl.dispose(); }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final isValid = _storyCtrl.text.trim().isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const ProviderBadge(
          label: 'Grok / Kling',
          description: 'Multi-scene AI video storytelling',
          color: Color(0xFFF59E0B),
          icon: Icons.movie_creation_rounded,
        ),
        const SizedBox(height: 16),

        // ── Overall story ──
        buildSectionLabel('Overall Story / Concept'),
        buildTextArea(
          controller: _storyCtrl,
          placeholder: 'Describe the overall narrative or concept for your video…',
          maxLines: 3,
          maxLength: 1000,
          onMicTap: _micAvailable ? _toggleMic : null,
          micActive: _micListening,
        ),
        const SizedBox(height: 16),

        // ── Video style ──
        buildSectionLabel('Video Style'),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: _defaultStyles.map((s) => buildChip(
            label: '${s['icon']} ${s['label']}',
            selected: _style == s['value'],
            onTap: () => setState(() => _style = s['value']!),
            activeColor: const Color(0xFFF59E0B),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Aspect ratio ──
        buildSectionLabel('Aspect Ratio'),
        Row(
          children: _defaultAspectRatios.map((a) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: buildChip(
              label: a['label']!,
              selected: _aspect == a['value'],
              onTap: () => setState(() => _aspect = a['value']!),
              activeColor: const Color(0xFFF59E0B),
            ),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Scene builder ──
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            buildSectionLabel('Scenes (${_scenes.length}/8)'),
            if (_scenes.length < 8)
              GestureDetector(
                onTap: _addScene,
                child: Row(
                  children: [
                    const Icon(Icons.add, size: 14, color: Color(0xFFF59E0B)),
                    const SizedBox(width: 4),
                    const Text('Add Scene', style: TextStyle(color: Color(0xFFF59E0B), fontSize: 12, fontWeight: FontWeight.w600)),
                  ],
                ),
              ),
          ],
        ),
        const SizedBox(height: 8),
        ...(_scenes.asMap().entries.map((e) => _SceneCard(
          index: e.key,
          data: e.value,
          onRemove: _scenes.length > 1 ? () => _removeScene(e.key) : null,
          onChanged: () => setState(() {}),
        ))),
        const SizedBox(height: 16),

        // ── Music style ──
        buildSectionLabel('Background Music'),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: _musicStyles.map((m) => buildChip(
            label: m,
            selected: _musicStyle == m,
            onTap: () => setState(() => _musicStyle = m),
          )).toList(),
        ),
        const SizedBox(height: 24),

        buildGenerateButton(
          label: p.isLoading
              ? 'Creating…'
              : 'Create Video${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFFD97706), Color(0xFFF59E0B)],
          icon: Icons.movie_creation_rounded,
        ),

        if (!p.canAfford) ...[
          const SizedBox(height: 8),
          Center(child: Text('You need ${p.pointCost} Pulse Points to use this tool', style: TextStyle(color: Colors.red.withValues(alpha: 0.7), fontSize: 12))),
        ],
      ],
    );
  }
}

// ─── Scene data model ─────────────────────────────────────────────────────────

class _SceneData {
  final TextEditingController ctrl = TextEditingController();
  int duration = 5;
}

// ─── Scene card widget ────────────────────────────────────────────────────────

class _SceneCard extends StatefulWidget {
  final int        index;
  final _SceneData data;
  final VoidCallback? onRemove;
  final VoidCallback  onChanged;

  const _SceneCard({
    required this.index,
    required this.data,
    required this.onRemove,
    required this.onChanged,
  });

  @override
  State<_SceneCard> createState() => _SceneCardState();
}

class _SceneCardState extends State<_SceneCard> {
  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.03),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 24,
                height: 24,
                decoration: BoxDecoration(
                  color: const Color(0xFFF59E0B).withValues(alpha: 0.15),
                  shape: BoxShape.circle,
                ),
                child: Center(
                  child: Text(
                    '${widget.index + 1}',
                    style: const TextStyle(color: Color(0xFFF59E0B), fontSize: 11, fontWeight: FontWeight.w800),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              Text(
                'Scene ${widget.index + 1}',
                style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 13),
              ),
              const Spacer(),
              // Duration selector
              Row(
                children: [3, 5, 8, 10].map((d) => Padding(
                  padding: const EdgeInsets.only(left: 4),
                  child: GestureDetector(
                    onTap: () => setState(() => widget.data.duration = d),
                    child: AnimatedContainer(
                      duration: const Duration(milliseconds: 120),
                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
                      decoration: BoxDecoration(
                        color: widget.data.duration == d
                            ? const Color(0xFFF59E0B).withValues(alpha: 0.15)
                            : Colors.transparent,
                        borderRadius: BorderRadius.circular(6),
                        border: Border.all(
                          color: widget.data.duration == d
                              ? const Color(0xFFF59E0B).withValues(alpha: 0.4)
                              : Colors.white.withValues(alpha: 0.08),
                        ),
                      ),
                      child: Text(
                        '${d}s',
                        style: TextStyle(
                          color: widget.data.duration == d ? const Color(0xFFF59E0B) : Colors.white.withValues(alpha: 0.35),
                          fontSize: 10,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                  ),
                )).toList(),
              ),
              if (widget.onRemove != null) ...[
                const SizedBox(width: 8),
                GestureDetector(
                  onTap: widget.onRemove,
                  child: Icon(Icons.remove_circle_outline, size: 16, color: Colors.white.withValues(alpha: 0.25)),
                ),
              ],
            ],
          ),
          const SizedBox(height: 8),
          TextField(
            controller: widget.data.ctrl,
            style: const TextStyle(color: Colors.white, fontSize: 12),
            maxLines: 2,
            decoration: InputDecoration(
              hintText: 'Describe scene ${widget.index + 1}…',
              hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.25), fontSize: 12),
              filled: true,
              fillColor: Colors.white.withValues(alpha: 0.03),
              border: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.08))),
              enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.08))),
              focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: Color(0xFFF59E0B), width: 1.5)),
              contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
            ),
            onChanged: (_) => widget.onChanged(),
          ),
        ],
      ),
    );
  }
}
