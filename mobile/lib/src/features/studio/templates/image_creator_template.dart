import 'package:flutter/material.dart';
import 'template_types.dart';

// Model identity — mirrors webapp's MODEL_IDENTITY map exactly
const _modelIdentity = {
  'ai-photo': (
    label: 'FLUX',
    desc: 'Fast, high-quality image generation',
    color: Color(0xFF7C3AED),
  ),
  'ai-photo-pro': (
    label: 'GPT-Image',
    desc: 'OpenAI GPT-Image · detailed realism',
    color: Color(0xFF2563EB),
  ),
  'ai-photo-max': (
    label: 'GPT-Image Large',
    desc: 'Max quality · 2x detail, slower',
    color: Color(0xFF4338CA),
  ),
  'ai-photo-dream': (
    label: 'Seedream',
    desc: 'Dreamlike aesthetics · stylised outputs',
    color: Color(0xFFDB2777),
  ),
};

const _defaultAspectRatios = [
  {'label': 'Square',    'value': '1:1',   'icon': '⬜'},
  {'label': 'Portrait',  'value': '9:16',  'icon': '📱'},
  {'label': 'Landscape', 'value': '16:9',  'icon': '🖥️'},
  {'label': 'Wide',      'value': '3:2',   'icon': '📸'},
];

const _defaultStyleTags = [
  'Photorealistic', 'Cinematic', 'Anime', 'Oil Painting', 'Watercolour',
  'Digital Art', 'Sketch', 'Minimalist', 'Dark Fantasy', 'Vintage',
  'Afrofuturist', 'Studio Portrait',
];

class ImageCreatorTemplate extends StatefulWidget {
  final TemplateProps props;
  const ImageCreatorTemplate({super.key, required this.props});

  @override
  State<ImageCreatorTemplate> createState() => _ImageCreatorTemplateState();
}

class _ImageCreatorTemplateState extends State<ImageCreatorTemplate> {
  final _promptCtrl = TextEditingController();
  final _negCtrl    = TextEditingController();
  String _aspect    = '1:1';
  List<String> _selectedStyles = [];
  bool _showNeg     = false;
  String _quality   = 'standard'; // standard | hd

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

  bool get _showQuality {
    final override = p.uiConfig['show_quality_toggle'];
    if (override is bool) return override;
    return p.slug == 'ai-photo-pro' || p.slug == 'ai-photo-max';
  }

  bool get _showStyles => p.uiConfig['show_style_tags'] != false;
  bool get _showNegPpt => p.uiConfig['show_negative_prompt'] != false;

  bool get _isValid => _promptCtrl.text.trim().length >= 3;

  void _handleSubmit() {
    if (!_isValid || p.isLoading || !p.canAfford) return;
    final stylePrefix = _selectedStyles.isNotEmpty ? '[${_selectedStyles.join(', ')}] ' : '';
    p.onSubmit(GeneratePayload(
      prompt: stylePrefix + _promptCtrl.text.trim(),
      aspectRatio: _aspect,
      styleTags: _selectedStyles.isNotEmpty ? _selectedStyles : null,
      negativePrompt: _negCtrl.text.trim().isNotEmpty ? _negCtrl.text.trim() : null,
      extraParams: _showQuality ? {'quality': _quality} : null,
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
    final modelInfo = _modelIdentity[p.slug];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [

        // ── Model identity badge ──
        if (modelInfo != null) ...[
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            decoration: BoxDecoration(
              color: modelInfo.color.withOpacity(0.12),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: modelInfo.color.withOpacity(0.3)),
            ),
            child: Row(
              children: [
                Icon(Icons.info_outline, size: 13, color: modelInfo.color.withOpacity(0.8)),
                const SizedBox(width: 8),
                Expanded(
                  child: RichText(
                    text: TextSpan(children: [
                      TextSpan(
                        text: modelInfo.label,
                        style: TextStyle(
                          color: modelInfo.color.withOpacity(0.9),
                          fontWeight: FontWeight.w800, fontSize: 12,
                        ),
                      ),
                      TextSpan(
                        text: ' — ${modelInfo.desc}',
                        style: TextStyle(color: Colors.white.withOpacity(0.5), fontSize: 11),
                      ),
                    ]),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Aspect ratio ──
        buildSectionLabel('Aspect Ratio'),
        Row(
          children: _aspectRatios.map((ar) {
            final selected = _aspect == ar['value'];
            return Expanded(
              child: GestureDetector(
                onTap: () => setState(() => _aspect = ar['value']!),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 150),
                  margin: const EdgeInsets.only(right: 6),
                  padding: const EdgeInsets.symmetric(vertical: 10),
                  decoration: BoxDecoration(
                    color: selected ? const Color(0xFF7C3AED).withOpacity(0.2) : Colors.transparent,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(
                      color: selected
                          ? const Color(0xFF7C3AED).withOpacity(0.6)
                          : Colors.white.withOpacity(0.1),
                    ),
                  ),
                  child: Column(
                    children: [
                      Text(ar['icon'] ?? '', style: const TextStyle(fontSize: 16)),
                      const SizedBox(height: 2),
                      Text(ar['label']!,
                          style: TextStyle(
                            fontSize: 9,
                            fontWeight: FontWeight.w700,
                            color: selected ? const Color(0xFFDDD6FE) : Colors.white.withOpacity(0.45),
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

        // ── Quality toggle (GPT-Image only) ──
        if (_showQuality) ...[
          buildSectionLabel('Quality'),
          Container(
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.white.withOpacity(0.1)),
            ),
            clipBehavior: Clip.hardEdge,
            child: Row(
              children: ['standard', 'hd'].map((q) {
                final sel = _quality == q;
                return Expanded(
                  child: GestureDetector(
                    onTap: () => setState(() => _quality = q),
                    child: Container(
                      padding: const EdgeInsets.symmetric(vertical: 10),
                      color: sel ? const Color(0xFF7C3AED) : Colors.transparent,
                      alignment: Alignment.center,
                      child: Text(
                        q == 'hd' ? '✦ HD' : 'Standard',
                        style: TextStyle(
                          fontSize: 12, fontWeight: FontWeight.w700,
                          color: sel ? Colors.white : Colors.white.withOpacity(0.55),
                        ),
                      ),
                    ),
                  ),
                );
              }).toList(),
            ),
          ),
          const SizedBox(height: 4),
          Text('HD uses more detail passes — slightly slower', style: kHintStyle),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Style tags ──
        if (_showStyles) ...[
          buildSectionLabel('Style'),
          Wrap(
            spacing: 6, runSpacing: 6,
            children: _styleTags.map((s) => buildChip(
              label: s,
              selected: _selectedStyles.contains(s),
              onTap: () => setState(() {
                _selectedStyles.contains(s)
                    ? _selectedStyles.remove(s)
                    : _selectedStyles.add(s);
              }),
            )).toList(),
          ),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Prompt ──
        buildSectionLabel('Describe your image'),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: p.uiConfig['prompt_placeholder'] ??
              'e.g. A majestic lion standing on a cliff at golden hour, dramatic lighting, ultra-detailed…',
          maxLines: 4,
          autoFocus: true,
        ),
        const SizedBox(height: 4),
        ValueListenableBuilder(
          valueListenable: _promptCtrl,
          builder: (_, __, ___) => Text(
            '${_promptCtrl.text.length}/500 characters',
            style: kHintStyle,
          ),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Negative prompt (collapsible) ──
        if (_showNegPpt) ...[
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
                  'Things to avoid: blurry, low quality, watermark, extra fingers…',
              maxLines: 2,
            ),
          ],
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Generate button ──
        ValueListenableBuilder(
          valueListenable: _promptCtrl,
          builder: (_, __, ___) => buildGenerateButton(
            label: 'Generate Image',
            enabled: _isValid && p.canAfford,
            isLoading: p.isLoading,
            onTap: _handleSubmit,
          ),
        ),
      ],
    );
  }
}
