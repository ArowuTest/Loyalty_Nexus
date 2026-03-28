// ─── Template Registry ────────────────────────────────────────────────────────
// Mirrors webapp's renderTemplate() switch exactly.
// Usage: TemplateRegistry.build(tool, onSubmit, isLoading, userPoints)

library studio_templates;

export 'template_types.dart';
export 'image_creator_template.dart';
export 'image_editor_template.dart';
export 'video_creator_template.dart';
export 'video_animator_template.dart';
export 'voice_studio_template.dart';
export 'music_composer_template.dart';
export 'transcribe_template.dart';
export 'vision_ask_template.dart';
export 'knowledge_doc_template.dart';

import 'package:flutter/material.dart';
import 'template_types.dart';
import 'image_creator_template.dart';
import 'image_editor_template.dart';
import 'video_creator_template.dart';
import 'video_animator_template.dart';
import 'voice_studio_template.dart';
import 'music_composer_template.dart';
import 'transcribe_template.dart';
import 'vision_ask_template.dart';
import 'knowledge_doc_template.dart';

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

    switch (tool['ui_template'] as String? ?? '') {
      case 'music-composer':
        return MusicComposerTemplate(props: props);
      case 'image-creator':
        return ImageCreatorTemplate(props: props);
      case 'image-editor':
        return ImageEditorTemplate(props: props);
      case 'video-creator':
        return VideoCreatorTemplate(props: props);
      case 'video-animator':
        return VideoAnimatorTemplate(props: props);
      case 'voice-studio':
        return VoiceStudioTemplate(props: props);
      case 'transcribe':
        return TranscribeTemplate(props: props);
      case 'vision-ask':
        return VisionAskTemplate(props: props);
      case 'knowledge-doc':
      default:
        return KnowledgeDocTemplate(props: props);
    }
  }
}
