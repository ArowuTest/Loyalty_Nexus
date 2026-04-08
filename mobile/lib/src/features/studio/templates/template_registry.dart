// ─── Template Registry ────────────────────────────────────────────────────────
// Mirrors webapp's renderTemplate() switch exactly.
// All 13 templates registered — 9 updated + 4 new.
// Usage: TemplateRegistry.build(tool, onSubmit, isLoading, userPoints)

export 'template_types.dart';
export 'music_composer.dart';
export 'image_creator.dart';
export 'image_editor.dart';
export 'image_compose.dart';
export 'video_creator.dart';
export 'video_animator.dart';
export 'video_editor.dart';
export 'video_extender.dart';
export 'video_multi_scene.dart';
export 'video_script.dart';
export 'voice_studio.dart';
export 'transcribe.dart';
export 'vision_ask.dart';
export 'knowledge_doc.dart';

import 'package:flutter/material.dart';
import 'template_types.dart';
import 'music_composer.dart';
import 'image_creator.dart';
import 'image_editor.dart';
import 'image_compose.dart';
import 'video_creator.dart';
import 'video_animator.dart';
import 'video_editor.dart';
import 'video_extender.dart';
import 'video_multi_scene.dart';
import 'video_script.dart';
import 'voice_studio.dart';
import 'transcribe.dart';
import 'vision_ask.dart';
import 'knowledge_doc.dart';

class TemplateRegistry {
  /// Picks the purpose-built input widget based on [tool.ui_template].
  /// Falls back to KnowledgeDocTemplate for any unknown template.
  /// Mirrors: webapp/src/app/studio/page.tsx → renderTemplate()
  static Widget build({
    required Map<String, dynamic> tool,
    required void Function(GeneratePayload) onSubmit,
    required bool isLoading,
    required int userPoints,
  }) {
    final props = TemplateProps(
      tool:       tool,
      onSubmit:   onSubmit,
      isLoading:  isLoading,
      userPoints: userPoints,
    );

    // Normalise: DB may still have PascalCase values (e.g. 'MusicComposer').
    // Convert to kebab-case so both old and new values match.
    final rawTpl = (tool['ui_template'] as String? ?? '').trim();
    final tpl = rawTpl
        .replaceAllMapped(RegExp(r'([A-Z])'), (m) {
          final i = rawTpl.indexOf(m.group(0)!);
          return (i > 0 ? '-' : '') + m.group(0)!.toLowerCase();
        })
        .replaceFirst(RegExp(r'^-'), '');

    switch (tpl) {
      // ── Music ──────────────────────────────────────────────────────────────────────
      case 'music-composer':
        return MusicComposerTemplate(props: props);

      // ── Image ──────────────────────────────────────────────────────────────────────
      case 'image-creator':
        return ImageCreatorTemplate(props: props);
      case 'image-editor':
        return ImageEditorTemplate(props: props);
      case 'image-compose':
        return ImageComposeTemplate(props: props);

      // ── Video ──────────────────────────────────────────────────────────────────────
      case 'video-creator':
        return VideoCreatorTemplate(props: props);
      case 'video-animator':
        return VideoAnimatorTemplate(props: props);
      case 'video-editor':
        return VideoEditorTemplate(props: props);
      case 'video-extender':
        return VideoExtenderTemplate(props: props);
      case 'video-multi-scene':
        return VideoMultiSceneTemplate(props: props);
      case 'video-script':
        return VideoScriptTemplate(props: props);

      // ── Voice & Audio ──────────────────────────────────────────────────────────────
      case 'voice-studio':
        return VoiceStudioTemplate(props: props);
      case 'transcribe':
        return TranscribeTemplate(props: props);

      // ── Vision & Analysis ──────────────────────────────────────────────────────────────
      case 'vision-ask':
        return VisionAskTemplate(props: props);

      case 'chat':
        return KnowledgeDocTemplate(props: props);

      // ── Website Builder (safety net — routed directly in studio_screen) ─────────
      case 'website-builder':
        return KnowledgeDocTemplate(props: props);

      // ── Document / Knowledge (default) ──────────────────────────────────────────────────────────────
      case 'knowledge-doc':
      default:
        return KnowledgeDocTemplate(props: props);
    }
  }
}
