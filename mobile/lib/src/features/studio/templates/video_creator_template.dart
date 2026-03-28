import 'package:flutter/material.dart';
import 'template_types.dart';

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
  {'label': 'Orbit',         'icon': '🔄', 'value': '360 orbit around subject'},
  {'label': 'Tracking',      'icon': '🎯', 'value': 'tracking shot following subject'},
  {'label': 'Handheld',      'icon': '📷', 'value': 'handheld camera, slight shake'},
  {'label': 'Static',        'icon': '📌', 'value': 'static camera, no movement'},
];

class VideoCreatorTemplate extends StatefulWidget {
  final TemplateProps props;
  const VideoCreatorTemplate({super.key, required this.props});

  @override
  State<VideoCreatorTemplate> createState() => _VideoCreatorTemplateState();
}

class _VideoCreatorTemplateState extends State<VideoCreatorTemplate> {
  final _promptCtrl = TextEditingController();
  final _negCtrl    = TextEditingController();
  String _aspect    = '16:9';
  int    _duration  = 5;
  List<String> _selStyles = [];
  String? _cameraMove;
  bool _showNeg     = false;

  TemplateProps get p => widget.props;

  List<Map<String, String>> get _aspectRatios {
    final raw = p.uiConfig['aspect_ratios'];
    if (raw is List) return raw.cast<Map<String, String>>();
    return List<Map<String, String>>.from(_defaultAspectRatios);
  }

  List<String> get _styleTags {
    final raw = p.uiConfig['style_tags'];
    if (raw is List) return raw.cast<String>();
    return _defaultStyleTags;
  }

  List<int> get _durations {
    final max = p.uiConfig['max_duration'] ?? 15;
    final raw = p.uiConfig['duration_options'];
    final list = raw is List ? raw.cast<int>() : List<int>.from(_defaultDurations);
    return list.where((d) => d <= max).toList();
  }

  List<Map<String, String>> get _cameraMoves {
    final raw = p.uiConfig['camera_movements'];
    if (raw is List) return raw.cast<Map<String, String>>();
    return List<Map<String, String>>.from(_cameraMovements);
  }

  bool get _isValid => _promptCtrl.text.trim().length >= 3;

  void _handleSubmit() {
    if (!_isValid || p.isLoading || !p.canAfford) return;
    final stylePrefix  = _selStyles.isNotEmpty ? '[${_selStyles.join(', ')}] ' : '';
    final cameraSuffix = _cameraMove != null ? '. Camera movement: $_cameraMove.' : '';
    p.onSubmit(GeneratePayload(
      prompt:          stylePrefix + _promptCtrl.text.trim() + cameraSuffix,
      aspectRatio:    _aspect,
      duration:        _duration,
      styleTags:      _selStyles.isNotEmpty ? _selStyles : null,
      negativePrompt: _negCtrl.text.trim().isNotEmpty ? _negCtrl.text.trim() : null,
      extraParams:    {'camera_movement': _cameraMove},
    ));
  }

  @override
  void dispose() {
    _promptCtrl.dispose();
    _negCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final warning = p.uiConfig['generation_warning'] as String?;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [

        // ── Warning banner ──
        if (warning != null) ...[
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            decoration: BoxDecoration(
              color: const Color(0xFFF59E0B).withOpacity(0.08),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: const Color(0xFFF59E0B).withOpacity(0.2)),
            ),
            child: Row(
              children: [
                Icon(Icons.warning_amber_rounded, size: 13, color: Colors.amber.withOpacity(0.8)),
                const SizedBox(width: 8),
                Expanded(child: Text(warning,
                    style: TextStyle(color: Colors.amber.withOpacity(0.75), fontSize: 11))),
              ],
            ),
          ),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Aspect ratio ──
        buildSectionLabel('Aspect Ratio'),
        Row(
          children: _aspectRatios.map((ar) {
            final sel = _aspect == ar['value'];
            return Expanded(
              child: GestureDetector(
                onTap: () => setState(() => _aspect = ar['value']!),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 150),
                  margin: const EdgeInsets.only(right: 6),
                  padding: const EdgeInsets.symmetric(vertical: 10),
                  decoration: BoxDecoration(
                    color: sel ? const Color(0xFF2563EB).withOpacity(0.2) : Colors.transparent,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(
                      color: sel ? const Color(0xFF2563EB).withOpacity(0.6)
                                 : Colors.white.withOpacity(0.1),
                    ),
                  ),
                  child: Column(
                    children: [
                      Text(ar['icon'] ?? '', style: const TextStyle(fontSize: 16)),
                      const SizedBox(height: 2),
                      Text(ar['label']!,
                          style: TextStyle(
                            fontSize: 9, fontWeight: FontWeight.w700,
                            color: sel ? const Color(0xFFBFDBFE) : Colors.white.withOpacity(0.45),
                          )),
                      Text(ar['value']!,
                          style: TextStyle(fontSize: 8, color: Colors.white.withOpacity(0.25))),
                    ],
                  ),
                ),
              ),
            );
          }).toList(),
        ),
        const SizedBox(height: kTemplateSpacing),

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
                  color: sel ? const Color(0xFF2563EB) : Colors.transparent,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: sel ? const Color(0xFF2563EB) : Colors.white.withOpacity(0.15),
                  ),
                ),
                child: Text('${d}s',
                    style: TextStyle(
                      fontSize: 12, fontWeight: FontWeight.w700,
                      color: sel ? Colors.white : Colors.white.withOpacity(0.55),
                    )),
              ),
            );
          }).toList(),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Style tags ──
        buildSectionLabel('Style'),
        Wrap(
          spacing: 6, runSpacing: 6,
          children: _styleTags.map((s) => buildChip(
            label: s,
            selected: _selStyles.contains(s),
            activeColor: const Color(0xFF2563EB),
            onTap: () => setState(() {
              _selStyles.contains(s) ? _selStyles.remove(s) : _selStyles.add(s);
            }),
          )).toList(),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Camera movement presets ──
        buildSectionLabel('Camera Movement (optional)'),
        GridView.count(
          crossAxisCount: 3,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          crossAxisSpacing: 6, mainAxisSpacing: 6,
          childAspectRatio: 2.8,
          children: _cameraMoves.map((cm) {
            final sel = _cameraMove == cm['value'];
            return GestureDetector(
              onTap: () => setState(() => _cameraMove = sel ? null : cm['value']),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(horizontal: 8),
                decoration: BoxDecoration(
                  color: sel ? const Color(0xFF2563EB).withOpacity(0.2) : Colors.transparent,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: sel ? const Color(0xFF2563EB).withOpacity(0.6)
                               : Colors.white.withOpacity(0.1),
                  ),
                ),
                child: Row(
                  children: [
                    Text(cm['icon'] ?? '', style: const TextStyle(fontSize: 12)),
                    const SizedBox(width: 4),
                    Expanded(
                      child: Text(cm['label']!,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 10, fontWeight: FontWeight.w600,
                            color: sel ? const Color(0xFFBFDBFE)
                                       : Colors.white.withOpacity(0.45),
                          )),
                    ),
                  ],
                ),
              ),
            );
          }).toList(),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Scene prompt ──
        buildSectionLabel('Scene description'),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: p.uiConfig['prompt_placeholder'] ??
              'Describe the scene — subject, setting, lighting, atmosphere…\ne.g. A hawk soaring over Lagos skyline at dusk',
          maxLines: 4,
          autoFocus: true,
        ),
        const SizedBox(height: 4),
        ValueListenableBuilder(
          valueListenable: _promptCtrl,
          builder: (_, __, ___) => Text('${_promptCtrl.text.length}/500 characters', style: kHintStyle),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Negative prompt ──
        GestureDetector(
          onTap: () => setState(() => _showNeg = !_showNeg),
          child: Text(
            _showNeg ? '▲ Hide negative prompt' : '▼ Add negative prompt (optional)',
            style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 12),
          ),
        ),
        if (_showNeg) ...[
          const SizedBox(height: 8),
          buildTextArea(
            controller: _negCtrl,
            placeholder: p.uiConfig['negative_prompt_placeholder'] ??
                'Things to avoid: shaky camera, blurry, text overlays, watermark…',
            maxLines: 2,
          ),
        ],
        const SizedBox(height: kTemplateSpacing),

        // ── Generate button ──
        ValueListenableBuilder(
          valueListenable: _promptCtrl,
          builder: (_, __, ___) => buildGenerateButton(
            label: 'Generate Video',
            enabled: _isValid && p.canAfford,
            isLoading: p.isLoading,
            onTap: _handleSubmit,
            gradientColors: const [Color(0xFF2563EB), Color(0xFF0891B2)],
            icon: Icons.videocam_rounded,
          ),
        ),
      ],
    );
  }
}
