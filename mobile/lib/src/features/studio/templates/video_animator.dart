// ─── Video Animator Template ──────────────────────────────────────────────────
// Mirrors webapp VideoAnimator.tsx exactly.
// Supports: video-animator, photo-to-video
// Payload: prompt, image_url (required), duration, aspect_ratio,
//          extra_params { animation_style, motion_intensity }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _animationStyles = [
  {'label': 'Cinematic',   'icon': '🎬', 'value': 'cinematic'},
  {'label': 'Anime',       'icon': '✨', 'value': 'anime'},
  {'label': 'Realistic',   'icon': '📷', 'value': 'realistic'},
  {'label': 'Artistic',    'icon': '🎨', 'value': 'artistic'},
  {'label': 'Slow Motion', 'icon': '🌊', 'value': 'slow-motion'},
  {'label': 'Timelapse',   'icon': '⏩', 'value': 'timelapse'},
];

const _motionIntensities = [
  {'label': 'Subtle',   'value': 'subtle'},
  {'label': 'Moderate', 'value': 'moderate'},
  {'label': 'Dynamic',  'value': 'dynamic'},
  {'label': 'Extreme',  'value': 'extreme'},
];

const _defaultDurations = [3, 5, 8, 10];

const _inspirations = [
  'Bring this portrait to life with gentle head movement and blinking eyes',
  'Animate this landscape with flowing water and swaying trees',
  'Make this character walk forward with natural movement',
];

class VideoAnimatorTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const VideoAnimatorTemplate({super.key, required this.props});

  @override
  ConsumerState<VideoAnimatorTemplate> createState() => _VideoAnimatorTemplateState();
}

class _VideoAnimatorTemplateState extends ConsumerState<VideoAnimatorTemplate> {
  final _promptCtrl = TextEditingController();
  final _negCtrl    = TextEditingController();

  String  _animStyle     = 'cinematic';
  String  _motionIntensity = 'moderate';
  int     _duration      = 5;
  String  _aspect        = '16:9';

  String? _imageUrl;
  String? _imagePreview;
  bool    _isUploading   = false;
  String? _uploadError;

  Future<void> _pickImage() async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(source: ImageSource.gallery, imageQuality: 85);
    if (picked == null) return;
    setState(() { _imagePreview = picked.path; _isUploading = true; _uploadError = null; });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(File(picked.path));
      setState(() { _imageUrl = url; _isUploading = false; });
    } catch (e) {
      setState(() { _uploadError = 'Upload failed — please try again.'; _isUploading = false; });
    }
  }

  void _clearImage() => setState(() { _imageUrl = null; _imagePreview = null; _uploadError = null; });

  void _handleSubmit() {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _imageUrl == null || _promptCtrl.text.trim().isEmpty) return;
    final payload = GeneratePayload(
      prompt: _promptCtrl.text.trim(),
      imageUrl: _imageUrl,
      duration: _duration,
      aspectRatio: _aspect,
      negativePrompt: _negCtrl.text.trim().isNotEmpty ? _negCtrl.text.trim() : null,
      extraParams: {
        'animation_style':  _animStyle,
        'motion_intensity': _motionIntensity,
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _promptCtrl.dispose();
    _negCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final isValid = _imageUrl != null && _promptCtrl.text.trim().isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const ProviderBadge(
          label: 'Kling / Wan',
          description: 'Image-to-video animation',
          color: Color(0xFFF97316),
          icon: Icons.animation_rounded,
        ),
        const SizedBox(height: 16),

        // ── Image upload (required) ──
        buildSectionLabel('Upload Image to Animate *'),
        UploadZone(
          label: 'Tap to upload image',
          sublabel: 'PNG, JPG, WEBP — required',
          icon: Icons.image_outlined,
          previewUrl: _imageUrl,
          isUploading: _isUploading,
          error: _uploadError,
          onTap: _pickImage,
          onClear: _clearImage,
          height: 150,
          accentColor: const Color(0xFFF97316),
          required: true,
        ),
        if (_uploadError != null) ...[
          const SizedBox(height: 4),
          Text(_uploadError!, style: const TextStyle(color: Color(0xFFEF4444), fontSize: 11)),
        ],
        const SizedBox(height: 16),

        // ── Animation prompt ──
        buildSectionLabel('Animation Description'),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: 'Describe how you want the image to animate…',
          maxLines: 3,
          maxLength: 500,
        ),
        const SizedBox(height: 8),
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

        // ── Animation style ──
        buildSectionLabel('Animation Style'),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: _animationStyles.map((s) => buildChip(
            label: '${s['icon']} ${s['label']}',
            selected: _animStyle == s['value'],
            onTap: () => setState(() => _animStyle = s['value']!),
            activeColor: const Color(0xFFF97316),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Motion intensity ──
        buildSectionLabel('Motion Intensity'),
        Row(
          children: _motionIntensities.map((m) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: buildChip(
              label: m['label']!,
              selected: _motionIntensity == m['value'],
              onTap: () => setState(() => _motionIntensity = m['value']!),
              activeColor: const Color(0xFFF97316),
            ),
          )).toList(),
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
            ),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Aspect ratio ──
        buildSectionLabel('Aspect Ratio'),
        Row(
          children: [
            {'label': '16:9', 'value': '16:9'},
            {'label': '9:16', 'value': '9:16'},
            {'label': '1:1',  'value': '1:1'},
          ].map((a) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: buildChip(
              label: a['label']!,
              selected: _aspect == a['value'],
              onTap: () => setState(() => _aspect = a['value']!),
            ),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Negative prompt ──
        CollapsibleSection(
          title: 'Negative Prompt (optional)',
          child: buildTextArea(
            controller: _negCtrl,
            placeholder: 'Things to avoid: blurry, distorted, low quality…',
            maxLines: 2,
          ),
        ),
        const SizedBox(height: 24),

        buildGenerateButton(
          label: p.isLoading
              ? 'Animating…'
              : 'Animate Image${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFFEA580C), Color(0xFFF97316)],
          icon: Icons.animation_rounded,
        ),

        if (_imageUrl == null && !p.isLoading) ...[
          const SizedBox(height: 8),
          Center(
            child: Text(
              'Please upload an image to animate',
              style: TextStyle(color: Colors.orange.withOpacity(0.7), fontSize: 12),
            ),
          ),
        ],
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
