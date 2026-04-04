// ─── Vision Ask Template ──────────────────────────────────────────────────────
// Mirrors webapp VisionAsk.tsx exactly.
// Supports: vision-ask, image-analyser
// Payload: prompt (question), image_url, extra_params { analysis_mode }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _analysisModes = [
  {'value': 'general',    'label': 'General Analysis',   'icon': Icons.auto_awesome_rounded,  'desc': 'Comprehensive image description'},
  {'value': 'detailed',   'label': 'Detailed Analysis',  'icon': Icons.search_rounded,         'desc': 'In-depth object & scene analysis'},
  {'value': 'ocr',        'label': 'Text Extraction',    'icon': Icons.text_fields_rounded,    'desc': 'Extract text from the image'},
  {'value': 'qa',         'label': 'Q&A',                'icon': Icons.question_answer_rounded,'desc': 'Ask specific questions'},
  {'value': 'comparison', 'label': 'Comparison',         'icon': Icons.compare_rounded,        'desc': 'Compare elements in image'},
];

const _quickQuestions = [
  'What is in this image?',
  'Describe this image in detail',
  'What text can you see?',
  'What are the main colours?',
  'Is there anything unusual?',
  'What emotions does this convey?',
];

class VisionAskTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const VisionAskTemplate({super.key, required this.props});

  @override
  ConsumerState<VisionAskTemplate> createState() => _VisionAskTemplateState();
}

class _VisionAskTemplateState extends ConsumerState<VisionAskTemplate> {
  final _questionCtrl = TextEditingController();
  final _urlCtrl      = TextEditingController();

  String _analysisMode = 'general';

  String? _imageUrl;
  bool    _isUploading = false;
  String? _uploadError;

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
            _questionCtrl.text = result.recognizedWords;
            _micListening = false;
          });
        }
      },
    );
  }

  Future<void> _pickImage() async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(source: ImageSource.gallery, imageQuality: 85);
    if (picked == null) return;
    setState(() { _isUploading = true; _uploadError = null; });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(File(picked.path));
      setState(() { _imageUrl = url; _isUploading = false; });
    } catch (e) {
      setState(() { _uploadError = 'Upload failed.'; _isUploading = false; });
    }
  }

  void _clearImage() => setState(() { _imageUrl = null; _uploadError = null; _urlCtrl.clear(); });

  void _handleSubmit() {
    final p = widget.props;
    final finalUrl = _imageUrl ?? _urlCtrl.text.trim();
    if (!p.canAfford || p.isLoading || finalUrl.isEmpty) return;
    final question = _questionCtrl.text.trim().isNotEmpty
        ? _questionCtrl.text.trim()
        : 'Analyse this image';
    final payload = GeneratePayload(
      prompt: question,
      imageUrl: finalUrl,
      extraParams: {'analysis_mode': _analysisMode},
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _questionCtrl.dispose();
    _urlCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final finalUrl = _imageUrl ?? _urlCtrl.text.trim();
    final isValid = finalUrl.isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const ProviderBadge(
          label: 'Pollinations Vision',
          description: 'Multimodal image understanding',
          color: Color(0xFF6366F1),
          icon: Icons.visibility_rounded,
        ),
        const SizedBox(height: 16),

        // ── Image upload ──
        buildSectionLabel('Upload Image'),
        UploadZone(
          label: 'Tap to upload',
          sublabel: 'PNG, JPG, WEBP',
          icon: Icons.image_search_rounded,
          previewUrl: _imageUrl,
          isUploading: _isUploading,
          error: _uploadError,
          onTap: _pickImage,
          onClear: _clearImage,
          height: 140,
          accentColor: const Color(0xFF6366F1),
        ),
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
              borderSide: const BorderSide(color: Color(0xFF6366F1), width: 1.5),
            ),
            contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          ),
          onChanged: (_) => setState(() {}),
        ),
        const SizedBox(height: 16),

        // ── Analysis mode ──
        buildSectionLabel('Analysis Mode'),
        ...(_analysisModes.map((mode) {
          final isSelected = _analysisMode == mode['value'];
          return Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: GestureDetector(
              onTap: () => setState(() => _analysisMode = mode['value'] as String),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
                decoration: BoxDecoration(
                  color: isSelected ? const Color(0xFF6366F1).withValues(alpha: 0.1) : Colors.white.withValues(alpha: 0.03),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: isSelected ? const Color(0xFF6366F1).withValues(alpha: 0.4) : Colors.white.withValues(alpha: 0.08),
                    width: isSelected ? 1.5 : 1,
                  ),
                ),
                child: Row(
                  children: [
                    Icon(
                      mode['icon'] as IconData,
                      size: 16,
                      color: isSelected ? const Color(0xFF6366F1) : Colors.white.withValues(alpha: 0.4),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            mode['label'] as String,
                            style: TextStyle(
                              color: isSelected ? Colors.white : Colors.white.withValues(alpha: 0.7),
                              fontWeight: FontWeight.w600,
                              fontSize: 13,
                            ),
                          ),
                          Text(
                            mode['desc'] as String,
                            style: TextStyle(color: Colors.white.withValues(alpha: 0.35), fontSize: 11),
                          ),
                        ],
                      ),
                    ),
                    if (isSelected)
                      const Icon(Icons.check_circle_rounded, size: 16, color: Color(0xFF6366F1)),
                  ],
                ),
              ),
            ),
          );
        })),
        const SizedBox(height: 16),

        // ── Question ──
        buildSectionLabel('Your Question (optional)'),
        buildTextArea(
          controller: _questionCtrl,
          placeholder: 'Ask anything about the image…',
          maxLines: 3,
          onMicTap: _micAvailable ? _toggleMic : null,
          micActive: _micListening,
        ),
        const SizedBox(height: 8),

        // ── Quick questions ──
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: _quickQuestions.map((q) => Padding(
              padding: const EdgeInsets.only(right: 8),
              child: GestureDetector(
                onTap: () => setState(() => _questionCtrl.text = q),
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.04),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.white.withValues(alpha: 0.1)),
                  ),
                  child: Text(q, style: TextStyle(color: Colors.white.withValues(alpha: 0.45), fontSize: 11)),
                ),
              ),
            )).toList(),
          ),
        ),
        const SizedBox(height: 24),

        buildGenerateButton(
          label: p.isLoading
              ? 'Analysing…'
              : 'Analyse Image${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF4F46E5), Color(0xFF6366F1)],
          icon: Icons.visibility_rounded,
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
