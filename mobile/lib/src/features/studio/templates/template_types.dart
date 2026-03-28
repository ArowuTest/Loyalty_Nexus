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
      child: Text(
        label,
        style: TextStyle(
          color: selected ? activeText : Colors.white.withOpacity(0.55),
          fontSize: 12,
          fontWeight: FontWeight.w600,
        ),
      ),
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
}) {
  return TextField(
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
      contentPadding: const EdgeInsets.all(14),
      counterStyle: const TextStyle(color: Color(0x40FFFFFF), fontSize: 11),
    ),
  );
}

// ─── Section Label ────────────────────────────────────────────────────────────

Widget buildSectionLabel(String text) => Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Text(text.toUpperCase(), style: kLabelStyle),
    );

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
