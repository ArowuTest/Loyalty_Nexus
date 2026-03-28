import 'package:flutter/material.dart';
import 'template_types.dart';

const _defaultEditSuggestions = [
  'Remove the background',
  'Add sunset lighting',
  'Make it look like a painting',
  'Add dramatic shadows',
  'Convert to black & white',
  'Make the colours more vibrant',
  'Add a smooth bokeh background',
  'Upscale & enhance sharpness',
  'Change background to a beach',
  'Add professional studio lighting',
];

class ImageEditorTemplate extends StatefulWidget {
  final TemplateProps props;
  const ImageEditorTemplate({super.key, required this.props});

  @override
  State<ImageEditorTemplate> createState() => _ImageEditorTemplateState();
}

class _ImageEditorTemplateState extends State<ImageEditorTemplate> {
  final _urlCtrl    = TextEditingController();
  final _editCtrl   = TextEditingController();
  bool _hasUrl      = false;  // user entered a URL

  TemplateProps get p => widget.props;

  List<String> get _suggestions {
    final raw = p.uiConfig['edit_suggestions'];
    if (raw is List) return raw.cast<String>();
    return _defaultEditSuggestions;
  }

  bool get _hasImage => _hasUrl && _urlCtrl.text.trim().isNotEmpty;
  bool get _isValid  => _hasImage && _editCtrl.text.trim().length >= 3;

  void _handleSubmit() {
    if (!_isValid || p.isLoading || !p.canAfford) return;
    p.onSubmit(GeneratePayload(
      prompt:   _editCtrl.text.trim(),
      imageUrl: _urlCtrl.text.trim(),
    ));
  }

  @override
  void dispose() {
    _urlCtrl.dispose();
    _editCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [

        // ── Step 1: Image URL ──
        _StepHeader(step: 1, label: p.uiConfig['upload_label'] ?? 'Your photo URL'),
        const SizedBox(height: 8),

        if (!_hasUrl) ...[
          // URL input card
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              border: Border.all(
                color: Colors.white.withOpacity(0.12),
                style: BorderStyle.solid,
              ),
              borderRadius: BorderRadius.circular(14),
              color: Colors.white.withOpacity(0.02),
            ),
            child: Column(
              children: [
                Icon(Icons.add_photo_alternate_outlined,
                    size: 36, color: Colors.white.withOpacity(0.3)),
                const SizedBox(height: 8),
                Text('Paste a public image URL',
                    style: TextStyle(color: Colors.white.withOpacity(0.65), fontSize: 13)),
                const SizedBox(height: 4),
                Text('PNG, JPG, WebP · up to ${p.uiConfig['max_file_mb'] ?? 10} MB',
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
              hintText: 'https://example.com/your-photo.jpg',
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
                borderSide: const BorderSide(color: Color(0xFF7C3AED), width: 1.5),
              ),
              contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
              suffixIcon: IconButton(
                icon: const Icon(Icons.arrow_forward, color: Color(0xFF7C3AED)),
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
          // Image preview card
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
                      child: Icon(Icons.broken_image_outlined, color: Colors.white38, size: 40),
                    ),
                  ),
                ),
              ),
              // Clear button
              Positioned(
                top: 8, right: 8,
                child: GestureDetector(
                  onTap: () => setState(() {
                    _hasUrl = false;
                    _urlCtrl.clear();
                    _editCtrl.clear();
                  }),
                  child: Container(
                    padding: const EdgeInsets.all(6),
                    decoration: BoxDecoration(
                      color: Colors.black.withOpacity(0.7),
                      shape: BoxShape.circle,
                    ),
                    child: const Icon(Icons.close, size: 14, color: Colors.white70),
                  ),
                ),
              ),
              // Labels
              Positioned(
                bottom: 8, left: 8,
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: Colors.black.withOpacity(0.7),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(Icons.image_outlined, size: 11, color: Colors.white.withOpacity(0.6)),
                      const SizedBox(width: 4),
                      Text('Original', style: TextStyle(color: Colors.white.withOpacity(0.6), fontSize: 11)),
                    ],
                  ),
                ),
              ),
              Positioned(
                bottom: 8, right: 8,
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: const Color(0xFF7C3AED).withOpacity(0.7),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: const Text('→ AI Edit',
                      style: TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.w600)),
                ),
              ),
            ],
          ),
        ],

        const SizedBox(height: kTemplateSpacing),

        // ── Step 2: Edit instruction (revealed after image) ──
        if (_hasImage) ...[
          _StepHeader(step: 2, label: 'Edit instruction'),
          const SizedBox(height: 8),

          // Quick-edit chips
          Wrap(
            spacing: 6, runSpacing: 6,
            children: _suggestions.map((s) {
              final selected = _editCtrl.text == s;
              return GestureDetector(
                onTap: () => setState(() => _editCtrl.text = s),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 150),
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                  decoration: BoxDecoration(
                    color: selected ? const Color(0xFF7C3AED) : Colors.transparent,
                    borderRadius: BorderRadius.circular(999),
                    border: Border.all(
                      color: selected
                          ? const Color(0xFF7C3AED)
                          : Colors.white.withOpacity(0.12),
                    ),
                  ),
                  child: Text(s,
                      style: TextStyle(
                        fontSize: 11, fontWeight: FontWeight.w600,
                        color: selected ? Colors.white : Colors.white.withOpacity(0.45),
                      )),
                ),
              );
            }).toList(),
          ),
          const SizedBox(height: 10),

          buildTextArea(
            controller: _editCtrl,
            placeholder: p.uiConfig['prompt_placeholder'] ??
                'Or describe exactly what to change — be specific for best results…',
            maxLines: 3,
            autoFocus: true,
          ),
          const SizedBox(height: kTemplateSpacing),

          // ── Generate button ──
          ValueListenableBuilder(
            valueListenable: _editCtrl,
            builder: (_, __, ___) => buildGenerateButton(
              label: 'Apply Edit',
              enabled: _isValid && p.canAfford,
              isLoading: p.isLoading,
              onTap: _handleSubmit,
              gradientColors: const [Color(0xFF7C3AED), Color(0xFF4338CA)],
              icon: Icons.auto_fix_high,
            ),
          ),
        ] else ...[
          buildGenerateButton(
            label: 'Apply Edit',
            enabled: false,
            isLoading: false,
            onTap: () {},
            gradientColors: const [Color(0xFF7C3AED), Color(0xFF4338CA)],
          ),
        ],
      ],
    );
  }
}

class _StepHeader extends StatelessWidget {
  final int step;
  final String label;
  const _StepHeader({required this.step, required this.label});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 20, height: 20,
          decoration: const BoxDecoration(
            color: Color(0x4D7C3AED),
            shape: BoxShape.circle,
          ),
          alignment: Alignment.center,
          child: Text('$step',
              style: const TextStyle(
                color: Color(0xFFDDD6FE), fontSize: 10, fontWeight: FontWeight.w800,
              )),
        ),
        const SizedBox(width: 8),
        Text(label.toUpperCase(), style: kLabelStyle),
      ],
    );
  }
}
