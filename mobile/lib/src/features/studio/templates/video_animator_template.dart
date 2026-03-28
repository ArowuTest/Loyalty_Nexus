import 'package:flutter/material.dart';
import 'template_types.dart';

const _defaultStyleTags = [
  'Smooth motion', 'Dramatic', 'Slow motion', 'Zoom in', 'Zoom out',
  'Pan left', 'Pan right', 'Parallax', 'Vibrant', 'Cinematic',
];
const _defaultDurations   = [5, 8, 10];
const _intensityLabels    = ['Subtle', 'Moderate', 'Strong'];
const _intensityColors    = [Color(0xFF3B82F6), Color(0xFF06B6D4), Color(0xFFF97316)];

class VideoAnimatorTemplate extends StatefulWidget {
  final TemplateProps props;
  const VideoAnimatorTemplate({super.key, required this.props});

  @override
  State<VideoAnimatorTemplate> createState() => _VideoAnimatorTemplateState();
}

class _VideoAnimatorTemplateState extends State<VideoAnimatorTemplate> {
  final _urlCtrl        = TextEditingController();
  final _motionCtrl     = TextEditingController();
  bool  _hasUrl         = false;
  List<String> _selStyles = [];
  int  _duration        = 5;
  double _intensity     = 1.0; // 0=Subtle, 1=Moderate, 2=Strong

  TemplateProps get p => widget.props;

  List<String> get _styleTags {
    final raw = p.uiConfig['style_tags'];
    if (raw is List) return raw.cast<String>();
    return _defaultStyleTags;
  }

  List<int> get _durations {
    final raw = p.uiConfig['duration_options'];
    if (raw is List) return raw.cast<int>();
    return List<int>.from(_defaultDurations);
  }

  bool get _hasImage => _hasUrl && _urlCtrl.text.trim().isNotEmpty;
  bool get _isValid  => _hasImage;

  void _handleSubmit() {
    if (!_isValid || p.isLoading || !p.canAfford) return;
    final stylePrefix    = _selStyles.isNotEmpty ? '[${_selStyles.join(', ')}] ' : '';
    final intensityLabel = _intensityLabels[_intensity.round()];
    final intensityCue   = intensityLabel != 'Moderate' ? ' $intensityLabel motion.' : '';
    p.onSubmit(GeneratePayload(
      prompt:    stylePrefix + (_motionCtrl.text.trim().isNotEmpty
          ? _motionCtrl.text.trim()
          : 'Animate this image with natural motion') + intensityCue,
      imageUrl:  _urlCtrl.text.trim(),
      duration:  _duration,
      styleTags: _selStyles.isNotEmpty ? _selStyles : null,
      extraParams: {'intensity': intensityLabel},
    ));
  }

  @override
  void dispose() {
    _urlCtrl.dispose();
    _motionCtrl.dispose();
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
                const Icon(Icons.warning_amber_rounded, size: 13, color: Colors.amber),
                const SizedBox(width: 8),
                Expanded(child: Text(warning,
                    style: TextStyle(color: Colors.amber.withOpacity(0.75), fontSize: 11))),
              ],
            ),
          ),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Step 1: Image URL ──
        Row(children: [
          _stepBadge(1, const Color(0xFF0891B2)),
          const SizedBox(width: 8),
          Text((p.uiConfig['upload_label'] ?? 'Photo or image to animate').toString().toUpperCase(),
              style: kLabelStyle),
        ]),
        const SizedBox(height: 8),

        if (!_hasUrl) ...[
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              border: Border.all(
                color: const Color(0xFF0891B2).withOpacity(0.3),
                style: BorderStyle.solid,
              ),
              borderRadius: BorderRadius.circular(14),
              color: const Color(0xFF0891B2).withOpacity(0.04),
            ),
            child: Column(
              children: [
                Icon(Icons.add_photo_alternate_outlined, size: 36, color: Colors.white.withOpacity(0.3)),
                const SizedBox(height: 8),
                Text('Upload the image to animate',
                    style: TextStyle(color: Colors.white.withOpacity(0.65), fontSize: 13)),
                const SizedBox(height: 4),
                Text('PNG, JPG, WebP · up to ${p.uiConfig['max_file_mb'] ?? 20} MB', style: kHintStyle),
              ],
            ),
          ),
          const SizedBox(height: 10),
          TextField(
            controller: _urlCtrl,
            keyboardType: TextInputType.url,
            style: const TextStyle(color: Colors.white, fontSize: 13),
            decoration: InputDecoration(
              hintText: 'https://example.com/photo.jpg',
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
                borderSide: const BorderSide(color: Color(0xFF0891B2), width: 1.5),
              ),
              contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
              suffixIcon: IconButton(
                icon: const Icon(Icons.arrow_forward, color: Color(0xFF0891B2)),
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
          Stack(
            children: [
              ClipRRect(
                borderRadius: BorderRadius.circular(14),
                child: Image.network(_urlCtrl.text.trim(),
                    width: double.infinity, height: 140, fit: BoxFit.cover,
                    errorBuilder: (_, __, ___) => Container(
                      height: 140, color: Colors.white.withOpacity(0.05),
                      child: const Center(child: Icon(Icons.broken_image_outlined,
                          color: Colors.white38, size: 36)),
                    )),
              ),
              Positioned(
                top: 8, right: 8,
                child: GestureDetector(
                  onTap: () => setState(() { _hasUrl = false; _urlCtrl.clear(); }),
                  child: Container(
                    padding: const EdgeInsets.all(6),
                    decoration: BoxDecoration(color: Colors.black.withOpacity(0.7), shape: BoxShape.circle),
                    child: const Icon(Icons.close, size: 14, color: Colors.white70),
                  ),
                ),
              ),
              Positioned(
                bottom: 8, left: 8,
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: Colors.black.withOpacity(0.7),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text('Source image', style: kHintStyle.copyWith(fontSize: 10)),
                ),
              ),
            ],
          ),
        ],

        // ── Step 2: Motion options (revealed once image is loaded) ──
        if (_hasImage) ...[
          const SizedBox(height: kTemplateSpacing),

          // Duration
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
                    color: sel ? const Color(0xFF0891B2) : Colors.transparent,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(
                      color: sel ? const Color(0xFF0891B2) : Colors.white.withOpacity(0.15),
                    ),
                  ),
                  child: Text('${d}s', style: TextStyle(
                    fontSize: 12, fontWeight: FontWeight.w700,
                    color: sel ? Colors.white : Colors.white.withOpacity(0.55),
                  )),
                ),
              );
            }).toList(),
          ),
          const SizedBox(height: kTemplateSpacing),

          // Motion intensity slider
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text('MOTION INTENSITY', style: kLabelStyle),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(
                  color: _intensityColors[_intensity.round()].withOpacity(0.2),
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(_intensityLabels[_intensity.round()],
                    style: TextStyle(
                      fontSize: 11, fontWeight: FontWeight.w700,
                      color: _intensityColors[_intensity.round()],
                    )),
              ),
            ],
          ),
          const SizedBox(height: 8),
          SliderTheme(
            data: SliderThemeData(
              trackHeight: 3,
              thumbColor: Colors.white,
              activeTrackColor: _intensityColors[_intensity.round()],
              inactiveTrackColor: Colors.white.withOpacity(0.1),
              overlayColor: Colors.white.withOpacity(0.1),
            ),
            child: Slider(
              min: 0, max: 2, divisions: 2, value: _intensity,
              onChanged: (v) => setState(() => _intensity = v),
            ),
          ),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text('Subtle', style: kHintStyle.copyWith(fontSize: 9)),
              Text('Strong', style: kHintStyle.copyWith(fontSize: 9)),
            ],
          ),
          const SizedBox(height: kTemplateSpacing),

          // Motion style tags
          buildSectionLabel('Motion Style'),
          Wrap(
            spacing: 6, runSpacing: 6,
            children: _styleTags.map((s) => buildChip(
              label: s,
              selected: _selStyles.contains(s),
              activeColor: const Color(0xFF0891B2),
              onTap: () => setState(() {
                _selStyles.contains(s) ? _selStyles.remove(s) : _selStyles.add(s);
              }),
            )).toList(),
          ),
          const SizedBox(height: kTemplateSpacing),

          // Motion description
          buildSectionLabel('Motion description (optional)'),
          buildTextArea(
            controller: _motionCtrl,
            placeholder: p.uiConfig['prompt_placeholder'] ??
                'Describe how to animate it — e.g. Camera slowly zooms in, trees sway, water ripples…',
            maxLines: 3,
            autoFocus: true,
          ),
          const SizedBox(height: kTemplateSpacing),

          buildGenerateButton(
            label: 'Animate Image',
            enabled: _isValid && p.canAfford,
            isLoading: p.isLoading,
            onTap: _handleSubmit,
            gradientColors: const [Color(0xFF0891B2), Color(0xFF2563EB)],
            icon: Icons.animation_rounded,
          ),
        ] else ...[
          const SizedBox(height: kTemplateSpacing),
          buildGenerateButton(
            label: 'Animate Image',
            enabled: false,
            isLoading: false,
            onTap: () {},
            gradientColors: const [Color(0xFF0891B2), Color(0xFF2563EB)],
          ),
        ],
      ],
    );
  }

  Widget _stepBadge(int n, Color color) => Container(
        width: 20, height: 20,
        decoration: BoxDecoration(color: color.withOpacity(0.25), shape: BoxShape.circle),
        alignment: Alignment.center,
        child: Text('$n', style: TextStyle(
          color: color.withOpacity(0.9), fontSize: 10, fontWeight: FontWeight.w800,
        )),
      );
}
