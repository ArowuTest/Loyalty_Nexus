// ─── Image Compose Template ───────────────────────────────────────────────────
// Mirrors webapp ImageCompose.tsx exactly.
// Supports: image-compose, image-composer
// Payload: prompt, image_url (primary), extra_params { image_url_2, image_url_3,
//          composition_style, aspect_ratio }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _compositionStyles = [
  {'value': 'blend',       'label': 'Blend',       'icon': '🎨', 'desc': 'Seamlessly blend images together'},
  {'value': 'collage',     'label': 'Collage',     'icon': '🖼️', 'desc': 'Create a stylish collage'},
  {'value': 'style-transfer','label': 'Style Transfer','icon': '✨', 'desc': 'Apply style from one image to another'},
  {'value': 'face-swap',   'label': 'Face Swap',   'icon': '🔄', 'desc': 'Swap faces between images'},
  {'value': 'background',  'label': 'Background',  'icon': '🌅', 'desc': 'Replace background with another image'},
  {'value': 'merge',       'label': 'Merge',       'icon': '🔗', 'desc': 'Merge elements from multiple images'},
];

const _defaultAspectRatios = [
  {'label': 'Square',    'value': '1:1'},
  {'label': 'Portrait',  'value': '9:16'},
  {'label': 'Landscape', 'value': '16:9'},
  {'label': '4:3',       'value': '4:3'},
];

class ImageComposeTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const ImageComposeTemplate({super.key, required this.props});

  @override
  ConsumerState<ImageComposeTemplate> createState() => _ImageComposeTemplateState();
}

class _ImageComposeTemplateState extends ConsumerState<ImageComposeTemplate> {
  final _promptCtrl = TextEditingController();

  String _compositionStyle = 'blend';
  String _aspect           = '1:1';

  // Up to 3 images
  final List<String?> _imageUrls    = [null, null, null];
  final List<String?> _imagePreviews = [null, null, null];
  final List<bool>    _isUploading  = [false, false, false];
  final List<String?> _uploadErrors = [null, null, null];

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
          setState(() { _promptCtrl.text = result.recognizedWords; _micListening = false; });
        }
      },
    );
  }

  Future<void> _pickImage(int index) async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(source: ImageSource.gallery, imageQuality: 85);
    if (picked == null) return;
    setState(() { _imagePreviews[index] = picked.path; _isUploading[index] = true; _uploadErrors[index] = null; });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(File(picked.path));
      setState(() { _imageUrls[index] = url; _isUploading[index] = false; });
    } catch (e) {
      setState(() { _uploadErrors[index] = 'Upload failed.'; _isUploading[index] = false; });
    }
  }

  void _clearImage(int index) => setState(() {
    _imageUrls[index] = null;
    _imagePreviews[index] = null;
    _uploadErrors[index] = null;
  });

  void _handleSubmit() {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _imageUrls[0] == null || _promptCtrl.text.trim().isEmpty) return;
    final payload = GeneratePayload(
      prompt: _promptCtrl.text.trim(),
      imageUrl: _imageUrls[0],
      aspectRatio: _aspect,
      extraParams: {
        'composition_style': _compositionStyle,
        if (_imageUrls[1] != null) 'image_url_2': _imageUrls[1],
        if (_imageUrls[2] != null) 'image_url_3': _imageUrls[2],
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _promptCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final isValid = _imageUrls[0] != null && _promptCtrl.text.trim().isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const ProviderBadge(
          label: 'Grok Aurora',
          description: 'Multi-image composition & blending',
          color: Color(0xFFDB2777),
          icon: Icons.photo_library_rounded,
        ),
        const SizedBox(height: 16),

        // ── Image slots ──
        buildSectionLabel('Upload Images (up to 3)'),
        Row(
          children: List.generate(3, (i) => Expanded(
            child: Padding(
              padding: EdgeInsets.only(right: i < 2 ? 8 : 0),
              child: _ImageSlot(
                index: i,
                imageUrl: _imageUrls[i],
                previewPath: _imagePreviews[i],
                isUploading: _isUploading[i],
                error: _uploadErrors[i],
                onTap: () => _pickImage(i),
                onClear: () => _clearImage(i),
                required: i == 0,
              ),
            ),
          )),
        ),
        const SizedBox(height: 16),

        // ── Composition style ──
        buildSectionLabel('Composition Style'),
        ...(_compositionStyles.map((s) {
          final isSelected = _compositionStyle == s['value'];
          return Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: GestureDetector(
              onTap: () => setState(() => _compositionStyle = s['value'] as String),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
                decoration: BoxDecoration(
                  color: isSelected ? const Color(0xFFDB2777).withOpacity(0.1) : Colors.white.withOpacity(0.03),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: isSelected ? const Color(0xFFDB2777).withOpacity(0.4) : Colors.white.withOpacity(0.08),
                    width: isSelected ? 1.5 : 1,
                  ),
                ),
                child: Row(
                  children: [
                    Text(s['icon'] as String, style: const TextStyle(fontSize: 18)),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            s['label'] as String,
                            style: TextStyle(
                              color: isSelected ? Colors.white : Colors.white.withOpacity(0.7),
                              fontWeight: FontWeight.w700,
                              fontSize: 13,
                            ),
                          ),
                          Text(
                            s['desc'] as String,
                            style: TextStyle(color: Colors.white.withOpacity(0.35), fontSize: 11),
                          ),
                        ],
                      ),
                    ),
                    if (isSelected)
                      const Icon(Icons.check_circle_rounded, size: 16, color: Color(0xFFDB2777)),
                  ],
                ),
              ),
            ),
          );
        })),
        const SizedBox(height: 16),

        // ── Prompt ──
        buildSectionLabel('Composition Instructions'),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: 'Describe how you want the images to be composed…',
          maxLines: 3,
          onMicTap: _micAvailable ? _toggleMic : null,
          micActive: _micListening,
        ),
        const SizedBox(height: 16),

        // ── Aspect ratio ──
        buildSectionLabel('Output Aspect Ratio'),
        Row(
          children: _defaultAspectRatios.map((a) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: buildChip(
              label: a['label']!,
              selected: _aspect == a['value'],
              onTap: () => setState(() => _aspect = a['value']!),
              activeColor: const Color(0xFFDB2777),
            ),
          )).toList(),
        ),
        const SizedBox(height: 24),

        buildGenerateButton(
          label: p.isLoading
              ? 'Composing…'
              : 'Compose Images${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFFBE185D), Color(0xFFDB2777)],
          icon: Icons.photo_library_rounded,
        ),

        if (_imageUrls[0] == null && !p.isLoading) ...[
          const SizedBox(height: 8),
          Center(
            child: Text(
              'Please upload at least one image',
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

// ─── Image Slot Widget ────────────────────────────────────────────────────────

class _ImageSlot extends StatelessWidget {
  final int     index;
  final String? imageUrl;
  final String? previewPath;
  final bool    isUploading;
  final String? error;
  final bool    required;
  final VoidCallback onTap;
  final VoidCallback onClear;

  const _ImageSlot({
    required this.index,
    required this.imageUrl,
    required this.previewPath,
    required this.isUploading,
    required this.error,
    required this.onTap,
    required this.onClear,
    this.required = false,
  });

  @override
  Widget build(BuildContext context) {
    final hasImage = imageUrl != null;
    return GestureDetector(
      onTap: hasImage ? null : onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 200),
        height: 90,
        decoration: BoxDecoration(
          color: hasImage
              ? Colors.transparent
              : Colors.white.withOpacity(0.04),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: error != null
                ? const Color(0xFFEF4444).withOpacity(0.5)
                : hasImage
                    ? const Color(0xFFDB2777).withOpacity(0.4)
                    : Colors.white.withOpacity(0.1),
            width: hasImage ? 1.5 : 1,
          ),
        ),
        child: ClipRRect(
          borderRadius: BorderRadius.circular(11),
          child: isUploading
              ? const Center(child: SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2, color: Color(0xFFDB2777))))
              : hasImage && previewPath != null
                  ? Stack(
                      fit: StackFit.expand,
                      children: [
                        Image.file(File(previewPath!), fit: BoxFit.cover),
                        Positioned(
                          top: 4,
                          right: 4,
                          child: GestureDetector(
                            onTap: onClear,
                            child: Container(
                              width: 18,
                              height: 18,
                              decoration: BoxDecoration(
                                color: Colors.black.withOpacity(0.6),
                                shape: BoxShape.circle,
                              ),
                              child: const Icon(Icons.close, size: 10, color: Colors.white),
                            ),
                          ),
                        ),
                      ],
                    )
                  : Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(
                          Icons.add_photo_alternate_outlined,
                          size: 20,
                          color: Colors.white.withOpacity(0.3),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          index == 0 ? 'Primary *' : 'Image ${index + 1}',
                          style: TextStyle(
                            color: Colors.white.withOpacity(0.3),
                            fontSize: 10,
                            fontWeight: index == 0 ? FontWeight.w700 : FontWeight.normal,
                          ),
                        ),
                      ],
                    ),
        ),
      ),
    );
  }
}
