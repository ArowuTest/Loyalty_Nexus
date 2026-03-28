import 'package:flutter/material.dart';
import 'template_types.dart';

const _defaultExampleQuestions = [
  'What do you see in this image?',
  'Describe this image in full detail',
  'What text is visible?',
  'Identify all objects and their locations',
  'Explain what is happening in this scene',
  'What emotions does this image convey?',
  'Are there any brand logos present?',
  'What is the approximate location?',
];

class VisionAskTemplate extends StatefulWidget {
  final TemplateProps props;
  const VisionAskTemplate({super.key, required this.props});

  @override
  State<VisionAskTemplate> createState() => _VisionAskTemplateState();
}

class _VisionAskTemplateState extends State<VisionAskTemplate> {
  final _urlCtrl        = TextEditingController();
  final _questionCtrl   = TextEditingController();
  final _qSearchCtrl    = TextEditingController();
  bool  _hasUrl         = false;

  TemplateProps get p => widget.props;

  bool get _autoMode => p.uiConfig['prompt_optional'] == true;

  List<String> get _exampleQuestions {
    final raw = p.uiConfig['example_questions'];
    if (raw is List) return raw.cast<String>();
    return _defaultExampleQuestions;
  }

  List<String> get _filteredQuestions {
    final q = _qSearchCtrl.text.toLowerCase();
    if (q.isEmpty) return _exampleQuestions;
    return _exampleQuestions.where((e) => e.toLowerCase().contains(q)).toList();
  }

  bool get _hasImage => _hasUrl && _urlCtrl.text.trim().isNotEmpty;
  bool get _isValid  =>
      _hasImage && (_autoMode || _questionCtrl.text.trim().length >= 3);

  void _handleSubmit() {
    if (!_isValid || p.isLoading || !p.canAfford) return;
    p.onSubmit(GeneratePayload(
      prompt:   _questionCtrl.text.trim().isNotEmpty
          ? _questionCtrl.text.trim()
          : 'Describe this image in full detail — objects, people, text, setting, mood, and any notable elements.',
      imageUrl: _urlCtrl.text.trim(),
    ));
  }

  @override
  void dispose() {
    _urlCtrl.dispose();
    _questionCtrl.dispose();
    _qSearchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [

        // ── Image upload (URL) ──
        buildSectionLabel(p.uiConfig['upload_label']?.toString() ?? 'Image to analyse'),
        const SizedBox(height: 8),

        if (!_hasUrl) ...[
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              border: Border.all(
                color: const Color(0xFFBE185D).withOpacity(0.3),
                style: BorderStyle.solid,
              ),
              borderRadius: BorderRadius.circular(14),
              color: const Color(0xFFBE185D).withOpacity(0.04),
            ),
            child: Column(
              children: [
                Icon(Icons.remove_red_eye_outlined, size: 36, color: Colors.white.withOpacity(0.3)),
                const SizedBox(height: 8),
                Text('Paste a public image URL',
                    style: TextStyle(color: Colors.white.withOpacity(0.65), fontSize: 13)),
                const SizedBox(height: 4),
                Text('PNG, JPG, WebP, GIF · up to ${p.uiConfig['max_file_mb'] ?? 20} MB',
                    style: kHintStyle),
              ],
            ),
          ),
          const SizedBox(height: 10),
          TextField(
            controller: _urlCtrl,
            keyboardType: TextInputType.url,
            style: const TextStyle(color: Colors.white, fontSize: 13),
            decoration: InputDecoration(
              hintText: 'https://example.com/image.jpg',
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
                borderSide: const BorderSide(color: Color(0xFFBE185D), width: 1.5),
              ),
              contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
              suffixIcon: IconButton(
                icon: const Icon(Icons.arrow_forward, color: Color(0xFFBE185D)),
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
                child: Image.network(
                  _urlCtrl.text.trim(),
                  width: double.infinity,
                  height: 160,
                  fit: BoxFit.cover,
                  errorBuilder: (_, __, ___) => Container(
                    height: 160,
                    color: Colors.white.withOpacity(0.05),
                    child: const Center(
                      child: Icon(Icons.broken_image_outlined, color: Colors.white38, size: 36),
                    ),
                  ),
                ),
              ),
              Positioned(
                top: 8, right: 8,
                child: GestureDetector(
                  onTap: () => setState(() { _hasUrl = false; _urlCtrl.clear(); }),
                  child: Container(
                    padding: const EdgeInsets.all(6),
                    decoration: BoxDecoration(
                      color: Colors.black.withOpacity(0.7), shape: BoxShape.circle,
                    ),
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
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(Icons.image_outlined, size: 11, color: Colors.white.withOpacity(0.6)),
                      const SizedBox(width: 4),
                      Text('Source image', style: kHintStyle.copyWith(fontSize: 10)),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ],

        // ── Auto-mode banner (image-analyser) ──
        if (_autoMode && _hasImage) ...[
          const SizedBox(height: 10),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            decoration: BoxDecoration(
              color: const Color(0xFFBE185D).withOpacity(0.08),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: const Color(0xFFBE185D).withOpacity(0.2)),
            ),
            child: Row(
              children: [
                const Icon(Icons.remove_red_eye_outlined, size: 13, color: Color(0xFFFDA4AF)),
                const SizedBox(width: 8),
                const Expanded(child: Text(
                  'AI will automatically describe everything visible — objects, text, colours, and scene context. Or type a specific question below.',
                  style: TextStyle(color: Color(0xFFFDA4AF), fontSize: 11),
                )),
              ],
            ),
          ),
        ],

        // ── Question input (after image is set) ──
        if (_hasImage) ...[
          const SizedBox(height: kTemplateSpacing),

          Row(children: [
            Text(
              _autoMode ? 'ASK A SPECIFIC QUESTION' : 'YOUR QUESTION',
              style: kLabelStyle,
            ),
            if (_autoMode)
              Text(' (optional)', style: kHintStyle)
            else
              const Text(' *', style: TextStyle(color: Colors.redAccent, fontSize: 12)),
          ]),
          const SizedBox(height: 8),

          buildTextArea(
            controller: _questionCtrl,
            placeholder: p.uiConfig['prompt_placeholder']?.toString() ??
                'What would you like to know about this image?',
            maxLines: 3,
            autoFocus: true,
          ),

          // Question search + filter (if many examples)
          if (_exampleQuestions.length > 4) ...[
            const SizedBox(height: 8),
            TextField(
              controller: _qSearchCtrl,
              style: const TextStyle(color: Colors.white, fontSize: 12),
              onChanged: (_) => setState(() {}),
              decoration: InputDecoration(
                hintText: 'Filter examples…',
                hintStyle: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 11),
                prefixIcon: Icon(Icons.search, size: 14, color: Colors.white.withOpacity(0.3)),
                filled: true,
                fillColor: Colors.white.withOpacity(0.03),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: BorderSide(color: Colors.white.withOpacity(0.08)),
                ),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: BorderSide(color: Colors.white.withOpacity(0.08)),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: const BorderSide(color: Color(0xFFBE185D), width: 1.5),
                ),
                contentPadding: const EdgeInsets.symmetric(vertical: 6, horizontal: 12),
              ),
            ),
          ],
          const SizedBox(height: 8),

          // Example questions
          Wrap(
            spacing: 6, runSpacing: 6,
            children: _filteredQuestions.map((q) {
              final sel = _questionCtrl.text == q;
              return GestureDetector(
                onTap: () => setState(() => _questionCtrl.text = q),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 150),
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                  decoration: BoxDecoration(
                    color: sel ? const Color(0xFFBE185D) : Colors.transparent,
                    borderRadius: BorderRadius.circular(999),
                    border: Border.all(
                      color: sel ? const Color(0xFFBE185D) : Colors.white.withOpacity(0.12),
                    ),
                  ),
                  child: Text(q, style: TextStyle(
                    fontSize: 11, fontWeight: FontWeight.w600,
                    color: sel ? Colors.white : Colors.white.withOpacity(0.45),
                  )),
                ),
              );
            }).toList(),
          ),
          const SizedBox(height: kTemplateSpacing),

          buildGenerateButton(
            label: _autoMode ? 'Analyse Image' : 'Ask About Image',
            enabled: _isValid && p.canAfford,
            isLoading: p.isLoading,
            onTap: _handleSubmit,
            gradientColors: const [Color(0xFFBE185D), Color(0xFFDB2777)],
            icon: _autoMode ? Icons.remove_red_eye_rounded : Icons.auto_awesome,
          ),
        ] else ...[
          const SizedBox(height: kTemplateSpacing),
          buildGenerateButton(
            label: _autoMode ? 'Analyse Image' : 'Ask About Image',
            enabled: false,
            isLoading: false,
            onTap: () {},
            gradientColors: const [Color(0xFFBE185D), Color(0xFFDB2777)],
          ),
        ],
      ],
    );
  }
}
