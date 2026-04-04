// ─── Studio Template Shared Types ─────────────────────────────────────────────
// Mirrors the webapp's templates/types.ts exactly.

import 'package:flutter/material.dart';

/// Payload sent to the API after the user fills in a template.
/// Mirrors webapp's GeneratePayload interface.
class GeneratePayload {
  final String prompt;
  final String? aspectRatio;
  final int? duration;
  final String? voiceId;
  final String? language;
  final bool? vocals;
  final String? lyrics;
  final List<String>? styleTags;
  final String? negativePrompt;
  final String? imageUrl;
  final String? documentUrl;
  final Map<String, dynamic>? extraParams;

  const GeneratePayload({
    required this.prompt,
    this.aspectRatio,
    this.duration,
    this.voiceId,
    this.language,
    this.vocals,
    this.lyrics,
    this.styleTags,
    this.negativePrompt,
    this.imageUrl,
    this.documentUrl,
    this.extraParams,
  });

  Map<String, dynamic> toJson() => {
        'prompt': prompt,
        if (aspectRatio != null) 'aspect_ratio': aspectRatio,
        if (duration != null) 'duration': duration,
        if (voiceId != null) 'voice_id': voiceId,
        if (language != null) 'language': language,
        if (vocals != null) 'vocals': vocals,
        if (lyrics != null) 'lyrics': lyrics,
        if (styleTags != null && styleTags!.isNotEmpty) 'style_tags': styleTags,
        if (negativePrompt != null) 'negative_prompt': negativePrompt,
        if (imageUrl != null) 'image_url': imageUrl,
        if (documentUrl != null) 'document_url': documentUrl,
        if (extraParams != null) ...extraParams!,
      };
}

/// Props every template widget receives — mirrors webapp's TemplateProps.
class TemplateProps {
  final Map<String, dynamic> tool;
  final void Function(GeneratePayload) onSubmit;
  final bool isLoading;
  final int userPoints;

  const TemplateProps({
    required this.tool,
    required this.onSubmit,
    required this.isLoading,
    required this.userPoints,
  });

  Map<String, dynamic> get uiConfig =>
      (tool['ui_config'] as Map<String, dynamic>?) ?? {};

  bool get isFree => tool['is_free'] == true || (tool['point_cost'] ?? 0) == 0;
  bool get canAfford => isFree || userPoints >= (tool['point_cost'] ?? 0);
  String get slug => tool['slug'] ?? '';
  String get name => (tool['name'] as String?) ?? '';
  int get pointCost => (tool['point_cost'] as int?) ?? 0;
}

// ─── Shared Design Tokens ─────────────────────────────────────────────────────

const kTemplateSpacing = 20.0;

/// Dark card background used by template sections.
const kSectionBg = Color(0xFF1A1A2E);

/// Shared label style (matches webapp's text-white/50 uppercase tracking-wider).
const TextStyle kLabelStyle = TextStyle(
  color: Color(0x80FFFFFF),
  fontSize: 10,
  fontWeight: FontWeight.w700,
  letterSpacing: 1.2,
);

/// Shared hint style (matches webapp's text-white/25).
const TextStyle kHintStyle = TextStyle(
  color: Color(0x40FFFFFF),
  fontSize: 11,
);

// ─── Shared Chip Builder ──────────────────────────────────────────────────────

Widget buildChip({
  required String label,
  required bool selected,
  required VoidCallback onTap,
  Color activeColor = const Color(0xFF7C3AED),
  Color activeText = Colors.white,
  String? emoji,
}) {
  return GestureDetector(
    onTap: onTap,
    child: AnimatedContainer(
      duration: const Duration(milliseconds: 150),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: selected ? activeColor : Colors.transparent,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(
          color: selected ? activeColor : Colors.white.withOpacity(0.15),
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (emoji != null) ...[Text(emoji, style: const TextStyle(fontSize: 12)), const SizedBox(width: 4)],
          Text(
            label,
            style: TextStyle(
              color: selected ? activeText : Colors.white.withOpacity(0.55),
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    ),
  );
}

/// Horizontal scrollable chip row
Widget buildChipRow({
  required List<String> options,
  required List<String> selected,
  required void Function(String) onToggle,
  Color activeColor = const Color(0xFF7C3AED),
}) {
  return SingleChildScrollView(
    scrollDirection: Axis.horizontal,
    child: Row(
      children: options.map((opt) {
        final isSelected = selected.contains(opt);
        return Padding(
          padding: const EdgeInsets.only(right: 8),
          child: buildChip(
            label: opt,
            selected: isSelected,
            onTap: () => onToggle(opt),
            activeColor: activeColor,
          ),
        );
      }).toList(),
    ),
  );
}

// ─── Shared TextArea Builder ──────────────────────────────────────────────────

Widget buildTextArea({
  required TextEditingController controller,
  required String placeholder,
  int maxLines = 4,
  bool autoFocus = false,
  int? maxLength,
  VoidCallback? onMicTap,
  bool micActive = false,
}) {
  return Stack(
    children: [
      TextField(
        controller: controller,
        autofocus: autoFocus,
        maxLines: maxLines,
        maxLength: maxLength,
        style: const TextStyle(color: Colors.white, fontSize: 14, height: 1.5),
        decoration: InputDecoration(
          hintText: placeholder,
          hintStyle: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 13),
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
          contentPadding: EdgeInsets.only(
            left: 14, top: 14, bottom: 14,
            right: onMicTap != null ? 44 : 14,
          ),
          counterStyle: const TextStyle(color: Color(0x40FFFFFF), fontSize: 11),
        ),
      ),
      if (onMicTap != null)
        Positioned(
          right: 8,
          bottom: 8,
          child: GestureDetector(
            onTap: onMicTap,
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 200),
              width: 32,
              height: 32,
              decoration: BoxDecoration(
                color: micActive
                    ? const Color(0xFFEF4444).withOpacity(0.2)
                    : Colors.white.withOpacity(0.06),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                  color: micActive
                      ? const Color(0xFFEF4444).withOpacity(0.5)
                      : Colors.white.withOpacity(0.1),
                ),
              ),
              child: Icon(
                micActive ? Icons.mic : Icons.mic_none_rounded,
                size: 16,
                color: micActive ? const Color(0xFFEF4444) : Colors.white.withOpacity(0.4),
              ),
            ),
          ),
        ),
    ],
  );
}

// ─── Section Label ────────────────────────────────────────────────────────────

Widget buildSectionLabel(String text, {Widget? trailing}) => Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(children: [
        Expanded(child: Text(text.toUpperCase(), style: kLabelStyle)),
        if (trailing != null) trailing,
      ]),
    );

// ─── UploadZone ───────────────────────────────────────────────────────────────

class UploadZone extends StatelessWidget {
  final String label;
  final String sublabel;
  final IconData icon;
  final String? previewUrl;
  final bool isUploading;
  final String? error;
  final VoidCallback onTap;
  final VoidCallback? onClear;
  final Color accentColor;
  final double height;

  const UploadZone({
    super.key,
    required this.label,
    required this.sublabel,
    required this.icon,
    this.previewUrl,
    this.isUploading = false,
    this.error,
    required this.onTap,
    this.onClear,
    this.accentColor = const Color(0xFF7C3AED),
    this.height = 90,
  });

  @override
  Widget build(BuildContext context) {
    final hasFile = previewUrl != null;
    return GestureDetector(
      onTap: isUploading ? null : onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 200),
        height: height,
        decoration: BoxDecoration(
          color: hasFile ? accentColor.withOpacity(0.08) : Colors.white.withOpacity(0.03),
          borderRadius: BorderRadius.circular(14),
          border: Border.all(
            color: error != null
                ? const Color(0xFFEF4444).withOpacity(0.5)
                : hasFile
                    ? accentColor.withOpacity(0.4)
                    : Colors.white.withOpacity(0.1),
            width: hasFile ? 1.5 : 1,
          ),
        ),
        child: Stack(
          children: [
            if (hasFile && previewUrl!.startsWith('http'))
              ClipRRect(
                borderRadius: BorderRadius.circular(13),
                child: Image.network(
                  previewUrl!,
                  width: double.infinity,
                  height: double.infinity,
                  fit: BoxFit.cover,
                  errorBuilder: (_, __, ___) => _placeholder(),
                ),
              )
            else
              _placeholder(),
            if (hasFile && onClear != null)
              Positioned(
                top: 6, right: 6,
                child: GestureDetector(
                  onTap: onClear,
                  child: Container(
                    width: 22, height: 22,
                    decoration: BoxDecoration(
                      color: Colors.black.withOpacity(0.6),
                      shape: BoxShape.circle,
                    ),
                    child: const Icon(Icons.close, size: 12, color: Colors.white),
                  ),
                ),
              ),
            if (isUploading)
              Container(
                decoration: BoxDecoration(
                  color: Colors.black.withOpacity(0.5),
                  borderRadius: BorderRadius.circular(13),
                ),
                child: const Center(
                  child: SizedBox(
                    width: 20, height: 20,
                    child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _placeholder() => Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, size: 22, color: Colors.white.withOpacity(0.3)),
            const SizedBox(height: 6),
            Text(label, style: TextStyle(color: Colors.white.withOpacity(0.6), fontSize: 12, fontWeight: FontWeight.w600)),
            Text(sublabel, style: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 10)),
          ],
        ),
      );
}

// ─── ProviderBadge ────────────────────────────────────────────────────────────

class ProviderBadge extends StatelessWidget {
  final String label;
  final String description;
  final Color color;
  final IconData icon;

  const ProviderBadge({
    super.key,
    required this.label,
    required this.description,
    required this.color,
    this.icon = Icons.bolt,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: color.withOpacity(0.08),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: color.withOpacity(0.25)),
      ),
      child: Row(
        children: [
          Icon(icon, size: 13, color: color),
          const SizedBox(width: 8),
          Expanded(
            child: RichText(
              text: TextSpan(
                style: TextStyle(color: color.withOpacity(0.8), fontSize: 11, height: 1.4),
                children: [
                  TextSpan(text: label, style: TextStyle(fontWeight: FontWeight.w700, color: color)),
                  TextSpan(text: ' — $description'),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ─── AspectRatioSelector ──────────────────────────────────────────────────────

class AspectRatioSelector extends StatelessWidget {
  final List<Map<String, String>> options;
  final String selected;
  final void Function(String) onSelect;

  const AspectRatioSelector({
    super.key,
    required this.options,
    required this.selected,
    required this.onSelect,
  });

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        children: options.map((opt) {
          final val = opt['value'] ?? '';
          final isSelected = selected == val;
          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: GestureDetector(
              onTap: () => onSelect(val),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                decoration: BoxDecoration(
                  color: isSelected ? const Color(0xFF7C3AED).withOpacity(0.15) : Colors.transparent,
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(
                    color: isSelected ? const Color(0xFF7C3AED).withOpacity(0.6) : Colors.white.withOpacity(0.12),
                  ),
                ),
                child: Column(
                  children: [
                    Text(opt['icon'] ?? '', style: const TextStyle(fontSize: 14)),
                    const SizedBox(height: 2),
                    Text(
                      opt['label'] ?? val,
                      style: TextStyle(
                        color: isSelected ? Colors.white : Colors.white.withOpacity(0.5),
                        fontSize: 10,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        }).toList(),
      ),
    );
  }
}

// ─── CollapsibleSection ───────────────────────────────────────────────────────

class CollapsibleSection extends StatefulWidget {
  final String title;
  final Widget child;
  final bool initiallyExpanded;

  const CollapsibleSection({
    super.key,
    required this.title,
    required this.child,
    this.initiallyExpanded = false,
  });

  @override
  State<CollapsibleSection> createState() => _CollapsibleSectionState();
}

class _CollapsibleSectionState extends State<CollapsibleSection> {
  late bool _expanded;

  @override
  void initState() {
    super.initState();
    _expanded = widget.initiallyExpanded;
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        GestureDetector(
          onTap: () => setState(() => _expanded = !_expanded),
          child: Row(
            children: [
              Expanded(child: Text(widget.title.toUpperCase(), style: kLabelStyle)),
              Icon(
                _expanded ? Icons.keyboard_arrow_up_rounded : Icons.keyboard_arrow_down_rounded,
                size: 16,
                color: Colors.white.withOpacity(0.4),
              ),
            ],
          ),
        ),
        if (_expanded) ...[const SizedBox(height: 10), widget.child],
      ],
    );
  }
}

// ─── Generate Button ─────────────────────────────────────────────────────────

Widget buildGenerateButton({
  required String label,
  required bool enabled,
  required bool isLoading,
  required VoidCallback onTap,
  List<Color> gradientColors = const [Color(0xFF7C3AED), Color(0xFFDB2777)],
  IconData icon = Icons.auto_awesome,
}) {
  return GestureDetector(
    onTap: enabled && !isLoading ? onTap : null,
    child: AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 14),
      decoration: BoxDecoration(
        gradient: enabled && !isLoading
            ? LinearGradient(colors: gradientColors)
            : null,
        color: enabled && !isLoading ? null : Colors.white.withOpacity(0.05),
        borderRadius: BorderRadius.circular(14),
        boxShadow: enabled && !isLoading
            ? [BoxShadow(color: gradientColors.first.withOpacity(0.3), blurRadius: 12, offset: const Offset(0, 4))]
            : null,
      ),
      child: isLoading
          ? const Center(
              child: SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2),
              ),
            )
          : Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(icon, color: enabled ? Colors.white : Colors.white.withOpacity(0.2), size: 16),
                const SizedBox(width: 8),
                Text(
                  label,
                  style: TextStyle(
                    color: enabled ? Colors.white : Colors.white.withOpacity(0.2),
                    fontWeight: FontWeight.w700,
                    fontSize: 14,
                  ),
                ),
              ],
            ),
    ),
  );
}
