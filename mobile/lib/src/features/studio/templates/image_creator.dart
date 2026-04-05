// ─── Image Creator Template ───────────────────────────────────────────────────
// Mirrors webapp ImageCreator.tsx exactly.
// Supports: ai-photo, ai-photo-pro, ai-photo-max, ai-photo-dream
// Payload: prompt, aspect_ratio, style_tags, negative_prompt, image_url (ref),
//          extra_params { quality, num_images, image_prompt_strength }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

// ─── Constants ────────────────────────────────────────────────────────────────

const _defaultAspectRatios = [
  {'label': 'Square',    'value': '1:1',   'icon': '⬜'},
  {'label': 'Portrait',  'value': '9:16',  'icon': '📱'},
  {'label': 'Landscape', 'value': '16:9',  'icon': '🖥️'},
  {'label': '4:3',       'value': '4:3',   'icon': '🖼️'},
  {'label': 'Wide',      'value': '21:9',  'icon': '🎬'},
];

const _defaultStyleTags = [
  'Photorealistic', 'Cinematic', 'Digital art', 'Oil painting', 'Watercolor',
  'Anime', 'Sketch', 'Vintage', 'Neon', 'Minimalist', 'Fantasy', 'Sci-Fi',
  'Portrait', 'Landscape', 'Abstract', 'Comic book', 'Dark', 'Bright',
];

const _inspirations = [
  'A majestic lion at sunset on the African savanna, golden hour, photorealistic',
  'A futuristic Lagos skyline at night, neon lights reflecting on water, cinematic',
  'A beautiful Nigerian woman in traditional attire, studio portrait, vibrant colors',
  'An abstract digital art piece with Afrobeats energy, swirling colors',
];

// ─── Provider badge config per slug ──────────────────────────────────────────

Map<String, dynamic> _providerConfig(String slug) {
  if (slug == 'ai-photo-pro') {
    return {
      'label': 'Grok Aurora',
      'desc': 'High quality · Detailed generation',
      'color': const Color(0xFF8B5CF6),
    };
  }
  if (slug == 'ai-photo-max') {
    return {
      'label': 'GPT-Image Large',
      'desc': 'Max quality · 2× detail, slower',
      'color': const Color(0xFF6366F1),
    };
  }
  if (slug == 'ai-photo-dream') {
    return {
      'label': 'Seedream 5',
      'desc': 'Dreamlike · Artistic generation',
      'color': const Color(0xFFEC4899),
    };
  }
  return {
    'label': 'FLUX',
    'desc': 'Fast, high-quality image generation',
    'color': const Color(0xFF7C3AED),
  };
}

// ─── Widget ───────────────────────────────────────────────────────────────────

class ImageCreatorTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const ImageCreatorTemplate({super.key, required this.props});

  @override
  ConsumerState<ImageCreatorTemplate> createState() => _ImageCreatorTemplateState();
}

class _ImageCreatorTemplateState extends ConsumerState<ImageCreatorTemplate> {
  final _promptCtrl  = TextEditingController();
  final _negCtrl     = TextEditingController();
  final _seedCtrl    = TextEditingController();

  String _aspect     = '1:1';
  String _quality    = 'standard';
  int    _numImages  = 1;
  double _refStrength = 0.7;
  final List<String> _selectedStyles = [];

  // Advanced settings
  bool   _useFixedSeed = false;

  // Reference image upload
  String? _refPreview;
  String? _refUploadedUrl;
  bool    _isRefUploading = false;
  String? _refUploadError;

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

  Future<void> _pickRefImage() async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(source: ImageSource.gallery, imageQuality: 85);
    if (picked == null) return;
    final file = File(picked.path);
    setState(() {
      _refPreview = picked.path;
      _isRefUploading = true;
      _refUploadError = null;
    });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(file);
      setState(() { _refUploadedUrl = url; _isRefUploading = false; });
    } catch (e) {
      setState(() { _refUploadError = 'Upload failed — please try again.'; _isRefUploading = false; });
    }
  }

  void _clearRef() => setState(() {
    _refPreview = null;
    _refUploadedUrl = null;
    _refUploadError = null;
  });

  void _handleSubmit() {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _promptCtrl.text.trim().isEmpty) return;
    final stylePrefix = _selectedStyles.isNotEmpty ? '[${_selectedStyles.join(', ')}] ' : '';
    final showQuality = ['ai-photo-pro', 'ai-photo-max'].contains(p.slug);
    final payload = GeneratePayload(
      prompt: stylePrefix + _promptCtrl.text.trim(),
      aspectRatio: _aspect,
      styleTags: _selectedStyles.isNotEmpty ? List.from(_selectedStyles) : null,
      negativePrompt: _negCtrl.text.trim().isNotEmpty ? _negCtrl.text.trim() : null,
      imageUrl: _refUploadedUrl,
      extraParams: {
        if (showQuality) 'quality': _quality,
        if (_numImages > 1) 'num_images': _numImages,
        if (_refUploadedUrl != null) 'image_prompt_strength': _refStrength,
        if (_useFixedSeed && _seedCtrl.text.trim().isNotEmpty)
          'seed': int.tryParse(_seedCtrl.text.trim()) ?? 0,
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _promptCtrl.dispose();
    _negCtrl.dispose();
    _seedCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final isValid = _promptCtrl.text.trim().isNotEmpty;
    final cfg = _providerConfig(p.slug);
    final showQuality = ['ai-photo-pro', 'ai-photo-max'].contains(p.slug);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ── Provider badge ──
        ProviderBadge(
          label: cfg['label'] as String,
          description: cfg['desc'] as String,
          color: cfg['color'] as Color,
          icon: Icons.image_rounded,
        ),
        const SizedBox(height: 16),

        // ── Prompt ──
        buildSectionLabel('Describe your image'),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: 'e.g. A majestic lion at sunset on the African savanna, golden hour, photorealistic…',
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
                    color: Colors.white.withValues(alpha: 0.04),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.white.withValues(alpha: 0.1)),
                  ),
                  child: Text(
                    insp.length > 50 ? '${insp.substring(0, 50)}…' : insp,
                    style: TextStyle(color: Colors.white.withValues(alpha: 0.45), fontSize: 11),
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
            if (_selectedStyles.contains(tag)) {
              _selectedStyles.remove(tag);
            } else if (_selectedStyles.length < 4) {
              _selectedStyles.add(tag);
            }
          }),
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

        // ── Quality (pro/max only) ──
        if (showQuality) ...[
          buildSectionLabel('Quality'),
          Row(
            children: ['standard', 'hd'].map((q) => Padding(
              padding: const EdgeInsets.only(right: 8),
              child: buildChip(
                label: q == 'hd' ? 'HD' : 'Standard',
                selected: _quality == q,
                onTap: () => setState(() => _quality = q),
                activeColor: const Color(0xFF7C3AED),
              ),
            )).toList(),
          ),
          const SizedBox(height: 16),
        ],

        // ── Number of images ──
        buildSectionLabel('Number of Images'),
        Row(
          children: [1, 2, 3, 4].map((n) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: buildChip(
              label: '$n',
              selected: _numImages == n,
              onTap: () => setState(() => _numImages = n),
            ),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Reference image ──
        CollapsibleSection(
          title: 'Style Reference Image (optional)',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              UploadZone(
                label: 'Upload reference',
                sublabel: 'Tap to pick from gallery',
                icon: Icons.image_outlined,
                previewUrl: _refPreview != null && _refUploadedUrl != null
                    ? _refUploadedUrl
                    : null,
                isUploading: _isRefUploading,
                error: _refUploadError,
                onTap: _pickRefImage,
                onClear: _clearRef,
                height: 120,
              ),
              if (_refUploadError != null) ...[
                const SizedBox(height: 4),
                Text(_refUploadError!, style: const TextStyle(color: Color(0xFFEF4444), fontSize: 11)),
              ],
              if (_refUploadedUrl != null) ...[
                const SizedBox(height: 12),
                buildSectionLabel('Reference Strength — ${(_refStrength * 100).round()}%'),
                Slider(
                  value: _refStrength,
                  min: 0.1,
                  max: 1.0,
                  divisions: 9,
                  activeColor: const Color(0xFF7C3AED),
                  inactiveColor: Colors.white.withValues(alpha: 0.1),
                  onChanged: (v) => setState(() => _refStrength = v),
                ),
              ],
            ],
          ),
        ),
        const SizedBox(height: 16),

        // ── Negative prompt ──
        CollapsibleSection(
          title: 'Negative Prompt (optional)',
          child: buildTextArea(
            controller: _negCtrl,
            placeholder: 'Things to avoid: blurry, low quality, watermark, extra fingers, distorted…',
            maxLines: 2,
          ),
        ),
        const SizedBox(height: 12),

        // ── Advanced Settings (Seed control — Midjourney-style) ──
        CollapsibleSection(
          title: 'Advanced Settings',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Seed',
                          style: TextStyle(
                            color: Colors.white.withValues(alpha: 0.9),
                            fontSize: 13,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          _useFixedSeed
                              ? 'Fixed — reproduce exact results'
                              : 'Random — unique result each time',
                          style: TextStyle(
                            color: Colors.white.withValues(alpha: 0.45),
                            fontSize: 11,
                          ),
                        ),
                      ],
                    ),
                  ),
                  Switch(
                    value: _useFixedSeed,
                    onChanged: (v) => setState(() => _useFixedSeed = v),
                    activeColor: const Color(0xFF7C3AED),
                  ),
                ],
              ),
              if (_useFixedSeed) ...[  
                const SizedBox(height: 10),
                TextField(
                  controller: _seedCtrl,
                  keyboardType: TextInputType.number,
                  style: const TextStyle(color: Colors.white, fontSize: 13),
                  decoration: InputDecoration(
                    hintText: 'Enter seed number (e.g. 42)',
                    hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 13),
                    filled: true,
                    fillColor: Colors.white.withValues(alpha: 0.05),
                    contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                    ),
                    enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                    ),
                    focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: const BorderSide(color: Color(0xFF7C3AED)),
                    ),
                    suffixIcon: TextButton(
                      onPressed: () {
                        final rng = (DateTime.now().millisecondsSinceEpoch % 999983);
                        setState(() => _seedCtrl.text = rng.toString());
                      },
                      child: Text(
                        'Random',
                        style: TextStyle(
                          color: const Color(0xFF7C3AED).withValues(alpha: 0.8),
                          fontSize: 11,
                        ),
                      ),
                    ),
                  ),
                ),
              ],
            ],
          ),
        ),
        const SizedBox(height: 24),

        // ── Generate button ──
        buildGenerateButton(
          label: p.isLoading
              ? 'Generating…'
              : 'Generate${_numImages > 1 ? ' $_numImages images' : ''}${p.pointCost > 0 ? ' · ${p.pointCost * _numImages} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          icon: Icons.auto_awesome,
        ),

        if (!p.canAfford) ...[
          const SizedBox(height: 8),
          Center(
            child: Text(
              'You need ${p.pointCost * _numImages} Pulse Points to generate',
              style: TextStyle(color: Colors.red.withValues(alpha: 0.7), fontSize: 12),
            ),
          ),
        ],
      ],
    );
  }
}
