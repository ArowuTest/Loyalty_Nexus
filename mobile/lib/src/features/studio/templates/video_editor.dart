// ─── Video Editor Template ────────────────────────────────────────────────────
// Mirrors webapp VideoEditor.tsx exactly.
// Supports: video-edit, video-editor
// Payload: prompt (edit instructions), video_url, extra_params { edit_type, strength }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:file_picker/file_picker.dart';
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _editTypes = [
  {'value': 'style',       'label': 'Style Change',    'icon': Icons.palette_rounded,         'desc': 'Apply a new visual style'},
  {'value': 'enhance',     'label': 'Enhance',         'icon': Icons.auto_fix_high_rounded,   'desc': 'Improve quality and clarity'},
  {'value': 'color',       'label': 'Color Grade',     'icon': Icons.color_lens_rounded,      'desc': 'Adjust colours and tone'},
  {'value': 'slow-motion', 'label': 'Slow Motion',     'icon': Icons.slow_motion_video_rounded,'desc': 'Create slow-motion effect'},
  {'value': 'upscale',     'label': 'Upscale',         'icon': Icons.hd_rounded,              'desc': 'Increase resolution'},
  {'value': 'loop',        'label': 'Loop',            'icon': Icons.loop_rounded,            'desc': 'Create a seamless loop'},
];

class VideoEditorTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const VideoEditorTemplate({super.key, required this.props});

  @override
  ConsumerState<VideoEditorTemplate> createState() => _VideoEditorTemplateState();
}

class _VideoEditorTemplateState extends ConsumerState<VideoEditorTemplate> {
  final _promptCtrl = TextEditingController();
  final _urlCtrl    = TextEditingController();

  String _editType = 'style';
  double _strength = 0.7;

  String? _videoUrl;
  String? _videoFileName;
  bool    _isUploading  = false;
  String? _uploadError;

  Future<void> _pickVideo() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.video,
      allowMultiple: false,
    );
    if (result == null || result.files.isEmpty) return;
    final file = File(result.files.first.path!);
    setState(() { _videoFileName = result.files.first.name; _isUploading = true; _uploadError = null; });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(file);
      setState(() { _videoUrl = url; _isUploading = false; });
    } catch (e) {
      setState(() { _uploadError = 'Upload failed — please try again.'; _isUploading = false; });
    }
  }

  void _clearVideo() => setState(() { _videoUrl = null; _videoFileName = null; _uploadError = null; _urlCtrl.clear(); });

  void _handleSubmit() {
    final p = widget.props;
    final finalUrl = _videoUrl ?? _urlCtrl.text.trim();
    if (!p.canAfford || p.isLoading || finalUrl.isEmpty || _promptCtrl.text.trim().isEmpty) return;
    final payload = GeneratePayload(
      prompt: _promptCtrl.text.trim(),
      videoUrl: finalUrl,
      extraParams: {
        'edit_type': _editType,
        'strength':  _strength,
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _promptCtrl.dispose();
    _urlCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final finalUrl = _videoUrl ?? _urlCtrl.text.trim();
    final isValid = finalUrl.isNotEmpty && _promptCtrl.text.trim().isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const ProviderBadge(
          label: 'Kling / Wan',
          description: 'AI video editing and enhancement',
          color: Color(0xFF7C3AED),
          icon: Icons.video_settings_rounded,
        ),
        const SizedBox(height: 16),

        // ── Video upload ──
        buildSectionLabel('Upload Video'),
        GestureDetector(
          onTap: _isUploading ? null : _pickVideo,
          child: Container(
            padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 16),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.04),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: _videoUrl != null
                    ? const Color(0xFF7C3AED).withOpacity(0.4)
                    : Colors.white.withOpacity(0.1),
              ),
            ),
            child: Row(
              children: [
                Icon(
                  _videoUrl != null ? Icons.videocam_rounded : Icons.upload_file_rounded,
                  size: 22,
                  color: _videoUrl != null ? const Color(0xFF7C3AED) : Colors.white.withOpacity(0.4),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        _videoFileName ?? 'Upload video file',
                        style: TextStyle(
                          color: _videoFileName != null ? Colors.white : Colors.white.withOpacity(0.4),
                          fontSize: 13,
                          fontWeight: _videoFileName != null ? FontWeight.w600 : FontWeight.normal,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      Text(
                        'MP4, MOV, AVI, WEBM',
                        style: TextStyle(color: Colors.white.withOpacity(0.25), fontSize: 10),
                      ),
                    ],
                  ),
                ),
                if (_videoUrl != null)
                  GestureDetector(
                    onTap: _clearVideo,
                    child: Icon(Icons.close, size: 16, color: Colors.white.withOpacity(0.3)),
                  ),
              ],
            ),
          ),
        ),
        if (_isUploading) ...[
          const SizedBox(height: 8),
          const LinearProgressIndicator(color: Color(0xFF7C3AED)),
        ],
        if (_uploadError != null) ...[
          const SizedBox(height: 4),
          Text(_uploadError!, style: const TextStyle(color: Color(0xFFEF4444), fontSize: 11)),
        ],
        const SizedBox(height: 12),

        // ── Or paste URL ──
        buildSectionLabel('Or Paste Video URL'),
        TextField(
          controller: _urlCtrl,
          style: const TextStyle(color: Colors.white, fontSize: 13),
          decoration: InputDecoration(
            hintText: 'https://example.com/video.mp4',
            hintStyle: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 12),
            filled: true,
            fillColor: Colors.white.withOpacity(0.04),
            border: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: BorderSide(color: Colors.white.withOpacity(0.1))),
            enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: BorderSide(color: Colors.white.withOpacity(0.1))),
            focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: const BorderSide(color: Color(0xFF7C3AED), width: 1.5)),
            contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          ),
          onChanged: (_) => setState(() {}),
        ),
        const SizedBox(height: 16),

        // ── Edit type ──
        buildSectionLabel('Edit Type'),
        ...(_editTypes.map((et) {
          final isSelected = _editType == et['value'];
          return Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: GestureDetector(
              onTap: () => setState(() => _editType = et['value'] as String),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
                decoration: BoxDecoration(
                  color: isSelected ? const Color(0xFF7C3AED).withOpacity(0.1) : Colors.white.withOpacity(0.03),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: isSelected ? const Color(0xFF7C3AED).withOpacity(0.4) : Colors.white.withOpacity(0.08),
                    width: isSelected ? 1.5 : 1,
                  ),
                ),
                child: Row(
                  children: [
                    Icon(et['icon'] as IconData, size: 16, color: isSelected ? const Color(0xFF7C3AED) : Colors.white.withOpacity(0.4)),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(et['label'] as String, style: TextStyle(color: isSelected ? Colors.white : Colors.white.withOpacity(0.7), fontWeight: FontWeight.w700, fontSize: 13)),
                          Text(et['desc'] as String, style: TextStyle(color: Colors.white.withOpacity(0.35), fontSize: 11)),
                        ],
                      ),
                    ),
                    if (isSelected) const Icon(Icons.check_circle_rounded, size: 16, color: Color(0xFF7C3AED)),
                  ],
                ),
              ),
            ),
          );
        })),
        const SizedBox(height: 16),

        // ── Edit instructions ──
        buildSectionLabel('Edit Instructions'),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: 'Describe the changes you want to make…',
          maxLines: 3,
        ),
        const SizedBox(height: 16),

        // ── Strength ──
        buildSectionLabel('Edit Strength — ${(_strength * 100).round()}%'),
        Slider(
          value: _strength,
          min: 0.1,
          max: 1.0,
          divisions: 9,
          activeColor: const Color(0xFF7C3AED),
          inactiveColor: Colors.white.withOpacity(0.1),
          onChanged: (v) => setState(() => _strength = v),
        ),
        const SizedBox(height: 24),

        buildGenerateButton(
          label: p.isLoading ? 'Editing…' : 'Edit Video${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF6D28D9), Color(0xFF7C3AED)],
          icon: Icons.video_settings_rounded,
        ),

        if (!p.canAfford) ...[
          const SizedBox(height: 8),
          Center(child: Text('You need ${p.pointCost} Pulse Points to use this tool', style: TextStyle(color: Colors.red.withOpacity(0.7), fontSize: 12))),
        ],
      ],
    );
  }
}
