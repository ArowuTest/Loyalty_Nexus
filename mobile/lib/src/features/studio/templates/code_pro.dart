// ─── Code Pro Template ────────────────────────────────────────────────────────
// Mirrors webapp CodePro.tsx exactly.
// Supports: code-pro, code-helper, nexus-code-pro
// Payload: prompt (question), image_url (optional screenshot), extra_params {}

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _kDefaultExampleQuestions = [
  'Why is this error happening and how do I fix it?',
  'Explain what this code does and suggest improvements',
  'Convert this UI design into React + Tailwind code',
  'Debug this traceback and show the corrected code',
  'What architecture pattern is shown in this diagram?',
  'Write unit tests for this function',
  'Optimise this SQL query for performance',
  'Explain this code to a junior developer',
];

class CodeProTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const CodeProTemplate({super.key, required this.props});

  @override
  ConsumerState<CodeProTemplate> createState() => _CodeProTemplateState();
}

class _CodeProTemplateState extends ConsumerState<CodeProTemplate> {
  final _questionCtrl = TextEditingController();
  final _filterCtrl   = TextEditingController();

  String? _imageUrl;
  bool    _isUploading  = false;
  String? _uploadError;
  bool    _showUpload   = false;

  // Speech
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
            final prev = _questionCtrl.text.trim();
            _questionCtrl.text = prev.isEmpty
                ? result.recognizedWords
                : '$prev ${result.recognizedWords}';
            _micListening = false;
          });
        }
      },
      localeId: 'en_NG',
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
      setState(() { _uploadError = 'Upload failed — please try again.'; _isUploading = false; });
    }
  }

  void _clearImage() => setState(() { _imageUrl = null; _uploadError = null; });

  void _handleSubmit() {
    final p = widget.props;
    final q = _questionCtrl.text.trim();
    if (!p.canAfford || p.isLoading || q.length < 3) return;
    final payload = GeneratePayload(
      prompt:   q,
      imageUrl: _imageUrl,
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _questionCtrl.dispose();
    _filterCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final cfg = (p.tool['ui_config'] as Map?) ?? {};
    final exampleQs = (cfg['example_questions'] as List?)
        ?.map((e) => e.toString())
        .toList()
      ?? _kDefaultExampleQuestions;
    final placeholder = cfg['prompt_placeholder']?.toString()
        ?? 'Describe your code problem — or attach a screenshot of the error, UI bug, or architecture diagram…';

    final filter = _filterCtrl.text.toLowerCase();
    final filteredQs = filter.isEmpty
        ? exampleQs
        : exampleQs.where((q) => q.toLowerCase().contains(filter)).toList();

    final isValid = _questionCtrl.text.trim().length >= 3 && !_isUploading && p.canAfford;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ── Provider badge ──────────────────────────────────────────────────
        const ProviderBadge(
          label: 'Nexus Code Pro',
          description: 'AI-powered code analysis, debugging & generation',
          color: Color(0xFF7C3AED),
          icon: Icons.code_rounded,
        ),
        const SizedBox(height: 16),

        // ── Code question ────────────────────────────────────────────────────
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              'Your code question',
              style: TextStyle(
                color: Colors.white.withValues(alpha: 0.5),
                fontSize: 11,
                fontWeight: FontWeight.w600,
                letterSpacing: 0.8,
              ),
            ),
            // Mic button
            GestureDetector(
              onTap: _toggleMic,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                width: 28, height: 28,
                decoration: BoxDecoration(
                  color: _micListening
                      ? const Color(0xFFEF4444).withValues(alpha: 0.2)
                      : Colors.white.withValues(alpha: 0.05),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: _micListening
                        ? const Color(0xFFEF4444).withValues(alpha: 0.4)
                        : Colors.transparent,
                  ),
                ),
                child: Icon(
                  _micListening ? Icons.mic_off_rounded : Icons.mic_rounded,
                  size: 13,
                  color: _micListening
                      ? const Color(0xFFEF4444)
                      : Colors.white.withValues(alpha: 0.3),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        buildTextArea(
          controller: _questionCtrl,
          placeholder: placeholder,
          maxLines: 5,
          onChanged: (_) => setState(() {}),
        ),
        if (_micListening) ...[
          const SizedBox(height: 6),
          Row(children: [
            Container(width: 6, height: 6,
              decoration: const BoxDecoration(
                color: Color(0xFFEF4444),
                shape: BoxShape.circle,
              ),
            ),
            const SizedBox(width: 6),
            Text('Listening… speak your question',
              style: TextStyle(color: Colors.red.withValues(alpha: 0.75), fontSize: 11)),
          ]),
        ],
        const SizedBox(height: 10),

        // ── Example question chips ───────────────────────────────────────────
        if (exampleQs.length > 4) ...[
          TextField(
            controller: _filterCtrl,
            style: const TextStyle(color: Colors.white, fontSize: 12),
            decoration: InputDecoration(
              hintText: 'Filter examples…',
              hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 12),
              prefixIcon: Icon(Icons.search, size: 14, color: Colors.white.withValues(alpha: 0.3)),
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
                borderSide: const BorderSide(color: Color(0xFF7C3AED), width: 1.5),
              ),
              contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
            ),
            onChanged: (_) => setState(() {}),
          ),
          const SizedBox(height: 8),
        ],

        Wrap(
          spacing: 6,
          runSpacing: 6,
          children: filteredQs.map((q) {
            final isSelected = _questionCtrl.text == q;
            return GestureDetector(
              onTap: () => setState(() => _questionCtrl.text = q),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 120),
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                decoration: BoxDecoration(
                  color: isSelected
                      ? const Color(0xFF7C3AED).withValues(alpha: 0.2)
                      : Colors.white.withValues(alpha: 0.04),
                  borderRadius: BorderRadius.circular(20),
                  border: Border.all(
                    color: isSelected
                        ? const Color(0xFF7C3AED).withValues(alpha: 0.5)
                        : Colors.white.withValues(alpha: 0.1),
                  ),
                ),
                child: Text(q,
                  style: TextStyle(
                    color: isSelected ? Colors.white : Colors.white.withValues(alpha: 0.5),
                    fontSize: 11,
                    fontWeight: isSelected ? FontWeight.w600 : FontWeight.w400,
                  ),
                ),
              ),
            );
          }).toList(),
        ),
        const SizedBox(height: 16),

        // ── Screenshot upload (collapsible) ─────────────────────────────────
        GestureDetector(
          onTap: () => setState(() => _showUpload = !_showUpload),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.03),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
            ),
            child: Row(
              children: [
                Icon(Icons.upload_rounded, size: 14, color: const Color(0xFF7C3AED)),
                const SizedBox(width: 8),
                Text(
                  cfg['upload_label']?.toString() ?? 'Attach screenshot (optional)',
                  style: TextStyle(
                    color: Colors.white.withValues(alpha: 0.5),
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    letterSpacing: 0.5,
                  ),
                ),
                if (_imageUrl != null) ...[
                  const SizedBox(width: 8),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                      color: const Color(0xFF7C3AED).withValues(alpha: 0.2),
                      borderRadius: BorderRadius.circular(10),
                      border: Border.all(color: const Color(0xFF7C3AED).withValues(alpha: 0.3)),
                    ),
                    child: const Text('1 image',
                      style: TextStyle(color: Color(0xFFa78bfa), fontSize: 10, fontWeight: FontWeight.w600)),
                  ),
                ],
                const Spacer(),
                Icon(
                  _showUpload ? Icons.expand_less_rounded : Icons.expand_more_rounded,
                  size: 16,
                  color: Colors.white.withValues(alpha: 0.3),
                ),
              ],
            ),
          ),
        ),

        if (_showUpload) ...[
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.02),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.white.withValues(alpha: 0.07)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Attach a screenshot of your error, UI bug, or architecture diagram for context-aware debugging.',
                  style: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 11),
                ),
                const SizedBox(height: 12),

                if (_imageUrl == null) ...[
                  // Upload zone
                  GestureDetector(
                    onTap: _pickImage,
                    child: Container(
                      width: double.infinity,
                      padding: const EdgeInsets.symmetric(vertical: 28),
                      decoration: BoxDecoration(
                        color: Colors.white.withValues(alpha: 0.02),
                        borderRadius: BorderRadius.circular(10),
                        border: Border.all(
                          color: Colors.white.withValues(alpha: 0.1),
                          style: BorderStyle.solid,
                        ),
                      ),
                      child: Column(
                        children: [
                          Icon(Icons.image_outlined, size: 28, color: Colors.white.withValues(alpha: 0.2)),
                          const SizedBox(height: 6),
                          Text('Tap to pick screenshot from gallery',
                            style: TextStyle(color: Colors.white.withValues(alpha: 0.4), fontSize: 12)),
                          Text('PNG, JPG, WebP · up to 20 MB',
                            style: TextStyle(color: Colors.white.withValues(alpha: 0.2), fontSize: 10)),
                        ],
                      ),
                    ),
                  ),
                ] else ...[
                  // Preview
                  Stack(
                    children: [
                      ClipRRect(
                        borderRadius: BorderRadius.circular(10),
                        child: Image.network(
                          _imageUrl!,
                          width: double.infinity,
                          height: 140,
                          fit: BoxFit.cover,
                          errorBuilder: (_, __, ___) => Container(
                            height: 140,
                            color: Colors.white.withValues(alpha: 0.05),
                            child: const Center(child: Icon(Icons.broken_image, color: Colors.white38)),
                          ),
                        ),
                      ),
                      Positioned(
                        top: 8, right: 8,
                        child: GestureDetector(
                          onTap: _clearImage,
                          child: Container(
                            width: 28, height: 28,
                            decoration: BoxDecoration(
                              color: Colors.black.withValues(alpha: 0.7),
                              shape: BoxShape.circle,
                            ),
                            child: const Icon(Icons.close, size: 14, color: Colors.white),
                          ),
                        ),
                      ),
                    ],
                  ),
                ],

                // Status banners
                if (_isUploading) ...[
                  const SizedBox(height: 8),
                  Row(children: [
                    const SizedBox(width: 14, height: 14,
                      child: CircularProgressIndicator(strokeWidth: 2, color: Color(0xFF7C3AED))),
                    const SizedBox(width: 8),
                    Text('Uploading screenshot…',
                      style: TextStyle(color: Colors.white.withValues(alpha: 0.55), fontSize: 12)),
                  ]),
                ],
                if (_imageUrl != null && !_isUploading) ...[
                  const SizedBox(height: 8),
                  Row(children: [
                    const Icon(Icons.check_circle_rounded, size: 14, color: Color(0xFF10B981)),
                    const SizedBox(width: 6),
                    Text('Screenshot ready — AI will use it for visual debugging',
                      style: TextStyle(color: Colors.green.withValues(alpha: 0.75), fontSize: 11)),
                  ]),
                ],
                if (_uploadError != null) ...[
                  const SizedBox(height: 8),
                  Container(
                    padding: const EdgeInsets.all(8),
                    decoration: BoxDecoration(
                      color: Colors.red.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: Colors.red.withValues(alpha: 0.2)),
                    ),
                    child: Text(_uploadError!,
                      style: TextStyle(color: Colors.red.withValues(alpha: 0.8), fontSize: 11)),
                  ),
                ],
              ],
            ),
          ),
        ],
        const SizedBox(height: 24),

        // ── Generate button ──────────────────────────────────────────────────
        buildGenerateButton(
          label: _isUploading
              ? 'Uploading screenshot…'
              : p.isLoading
                  ? 'Generating code…'
                  : _imageUrl != null
                      ? 'Generate with Visual Context${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}'
                      : 'Generate Code${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid,
          isLoading: p.isLoading || _isUploading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF6D28D9), Color(0xFF7C3AED)],
          icon: Icons.code_rounded,
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
