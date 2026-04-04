// ─── Image Editor Template ────────────────────────────────────────────────────
// Mirrors webapp ImageEditor.tsx exactly.
// Supports: photo-editor, background-remover
// Payload: prompt, image_url, extra_params { strength }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ImageEditorTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const ImageEditorTemplate({super.key, required this.props});

  @override
  ConsumerState<ImageEditorTemplate> createState() => _ImageEditorTemplateState();
}

class _ImageEditorTemplateState extends ConsumerState<ImageEditorTemplate> {
  final _editPromptCtrl = TextEditingController();
  final _urlCtrl        = TextEditingController();

  String? _uploadedUrl;
  bool    _isUploading  = false;
  String? _uploadError;
  double  _strength     = 0.85;

  bool get _isBackgroundRemover => widget.props.slug == 'background-remover';

  Future<void> _pickImage() async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(source: ImageSource.gallery, imageQuality: 85);
    if (picked == null) return;
    setState(() {

      _isUploading = true;
      _uploadError = null;
      _uploadedUrl = null;
    });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(File(picked.path));
      setState(() { _uploadedUrl = url; _isUploading = false; });
    } catch (e) {
      setState(() { _uploadError = 'Upload failed — please try again.'; _isUploading = false; });
    }
  }

  void _clearImage() => setState(() {

    _uploadedUrl = null;
    _uploadError = null;
    _urlCtrl.clear();
  });

  void _handleSubmit() {
    final p = widget.props;
    final finalUrl = _uploadedUrl ?? _urlCtrl.text.trim();
    if (finalUrl.isEmpty || !p.canAfford || p.isLoading) return;
    final editPrompt = _editPromptCtrl.text.trim();
    final payload = GeneratePayload(
      prompt: _isBackgroundRemover
          ? 'Remove the background'
          : (editPrompt.isNotEmpty ? editPrompt : 'Remove the background'),
      imageUrl: finalUrl,
      extraParams: {'strength': _strength},
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _editPromptCtrl.dispose();
    _urlCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final finalUrl = _uploadedUrl ?? _urlCtrl.text.trim();
    final isValid = finalUrl.isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ── Provider badge ──
        ProviderBadge(
          label: _isBackgroundRemover ? 'rembg / BiRefNet' : 'Pollinations p-edit',
          description: _isBackgroundRemover
              ? 'Precise background removal'
              : 'AI-powered image editing',
          color: const Color(0xFF06B6D4),
          icon: Icons.edit_rounded,
        ),
        const SizedBox(height: 16),

        // ── Image upload ──
        buildSectionLabel('Upload Image'),
        UploadZone(
          label: 'Tap to upload',
          sublabel: 'PNG, JPG, WEBP',
          icon: Icons.image_outlined,
          previewUrl: _uploadedUrl,
          isUploading: _isUploading,
          error: _uploadError,
          onTap: _pickImage,
          onClear: _clearImage,
          height: 140,
          accentColor: const Color(0xFF06B6D4),
        ),
        if (_uploadError != null) ...[
          const SizedBox(height: 4),
          Text(_uploadError!, style: const TextStyle(color: Color(0xFFEF4444), fontSize: 11)),
        ],
        const SizedBox(height: 12),

        // ── Or paste URL ──
        buildSectionLabel('Or Paste Image URL'),
        TextField(
          controller: _urlCtrl,
          style: const TextStyle(color: Colors.white, fontSize: 13),
          decoration: InputDecoration(
            hintText: 'https://example.com/image.jpg',
            hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 12),
            filled: true,
            fillColor: Colors.white.withValues(alpha: 0.04),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: const BorderSide(color: Color(0xFF06B6D4), width: 1.5),
            ),
            contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          ),
          onChanged: (_) => setState(() {}),
        ),
        const SizedBox(height: 16),

        // ── Edit prompt (non-background-remover) ──
        if (!_isBackgroundRemover) ...[
          buildSectionLabel('Edit Instructions'),
          buildTextArea(
            controller: _editPromptCtrl,
            placeholder: 'e.g. Make the background a beach sunset, add dramatic lighting…',
            maxLines: 3,
          ),
          const SizedBox(height: 16),
        ],

        // ── Strength ──
        buildSectionLabel('Edit Strength — ${(_strength * 100).round()}%'),
        Slider(
          value: _strength,
          min: 0.1,
          max: 1.0,
          divisions: 9,
          activeColor: const Color(0xFF06B6D4),
          inactiveColor: Colors.white.withValues(alpha: 0.1),
          onChanged: (v) => setState(() => _strength = v),
        ),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('Subtle', style: TextStyle(color: Colors.white.withValues(alpha: 0.35), fontSize: 10)),
            Text('Strong', style: TextStyle(color: Colors.white.withValues(alpha: 0.35), fontSize: 10)),
          ],
        ),
        const SizedBox(height: 24),

        // ── Generate button ──
        buildGenerateButton(
          label: p.isLoading
              ? 'Processing…'
              : (_isBackgroundRemover ? 'Remove Background' : 'Edit Image') +
                  (p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''),
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF0891B2), Color(0xFF06B6D4)],
          icon: Icons.auto_fix_high_rounded,
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
