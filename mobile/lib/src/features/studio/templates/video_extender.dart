// ─── Video Extender Template ──────────────────────────────────────────────────
// Mirrors webapp VideoExtender.tsx exactly.
// Supports: video-extend, video-extender
// Payload: prompt, video_url, extra_params { extend_seconds, direction, match_style }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:file_picker/file_picker.dart';
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _extendDirections = [
  {'value': 'forward',  'label': 'Extend Forward',  'icon': Icons.arrow_forward_rounded, 'desc': 'Continue the video beyond its current end'},
  {'value': 'backward', 'label': 'Extend Backward', 'icon': Icons.arrow_back_rounded,    'desc': 'Add content before the video starts'},
  {'value': 'both',     'label': 'Both Directions',  'icon': Icons.swap_horiz_rounded,    'desc': 'Extend from both start and end'},
];

const _extendSeconds = [2, 4, 6, 8, 10, 15];

class VideoExtenderTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const VideoExtenderTemplate({super.key, required this.props});

  @override
  ConsumerState<VideoExtenderTemplate> createState() => _VideoExtenderTemplateState();
}

class _VideoExtenderTemplateState extends ConsumerState<VideoExtenderTemplate> {
  final _promptCtrl = TextEditingController();
  final _urlCtrl    = TextEditingController();

  String _direction     = 'forward';
  int    _extendSecs    = 4;
  bool   _matchStyle    = true;

  String? _videoUrl;
  String? _videoFileName;
  bool    _isUploading  = false;
  String? _uploadError;

  Future<void> _pickVideo() async {
    final result = await FilePicker.platform.pickFiles(type: FileType.video, allowMultiple: false);
    if (result == null || result.files.isEmpty) return;
    final file = File(result.files.first.path!);
    setState(() { _videoFileName = result.files.first.name; _isUploading = true; _uploadError = null; });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(file);
      setState(() { _videoUrl = url; _isUploading = false; });
    } catch (e) {
      setState(() { _uploadError = 'Upload failed.'; _isUploading = false; });
    }
  }

  void _clearVideo() => setState(() { _videoUrl = null; _videoFileName = null; _uploadError = null; _urlCtrl.clear(); });

  void _handleSubmit() {
    final p = widget.props;
    final finalUrl = _videoUrl ?? _urlCtrl.text.trim();
    if (!p.canAfford || p.isLoading || finalUrl.isEmpty) return;
    final payload = GeneratePayload(
      prompt: _promptCtrl.text.trim().isNotEmpty
          ? _promptCtrl.text.trim()
          : 'Continue the video naturally',
      videoUrl: finalUrl,
      extraParams: {
        'extend_seconds': _extendSecs,
        'direction':      _direction,
        'match_style':    _matchStyle,
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
    final isValid = finalUrl.isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const ProviderBadge(
          label: 'Kling / Wan',
          description: 'Seamless video extension',
          color: Color(0xFF0EA5E9),
          icon: Icons.expand_rounded,
        ),
        const SizedBox(height: 16),

        // ── Video upload ──
        buildSectionLabel('Upload Video to Extend'),
        GestureDetector(
          onTap: _isUploading ? null : _pickVideo,
          child: Container(
            padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 16),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.04),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: _videoUrl != null ? const Color(0xFF0EA5E9).withOpacity(0.4) : Colors.white.withOpacity(0.1),
              ),
            ),
            child: Row(
              children: [
                Icon(
                  _videoUrl != null ? Icons.videocam_rounded : Icons.upload_file_rounded,
                  size: 22,
                  color: _videoUrl != null ? const Color(0xFF0EA5E9) : Colors.white.withOpacity(0.4),
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
                      Text('MP4, MOV, AVI, WEBM', style: TextStyle(color: Colors.white.withOpacity(0.25), fontSize: 10)),
                    ],
                  ),
                ),
                if (_videoUrl != null)
                  GestureDetector(onTap: _clearVideo, child: Icon(Icons.close, size: 16, color: Colors.white.withOpacity(0.3))),
              ],
            ),
          ),
        ),
        if (_isUploading) ...[const SizedBox(height: 8), const LinearProgressIndicator(color: Color(0xFF0EA5E9))],
        if (_uploadError != null) ...[const SizedBox(height: 4), Text(_uploadError!, style: const TextStyle(color: Color(0xFFEF4444), fontSize: 11))],
        const SizedBox(height: 12),

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
            focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: const BorderSide(color: Color(0xFF0EA5E9), width: 1.5)),
            contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          ),
          onChanged: (_) => setState(() {}),
        ),
        const SizedBox(height: 16),

        // ── Direction ──
        buildSectionLabel('Extension Direction'),
        ...(_extendDirections.map((d) {
          final isSelected = _direction == d['value'];
          return Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: GestureDetector(
              onTap: () => setState(() => _direction = d['value'] as String),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
                decoration: BoxDecoration(
                  color: isSelected ? const Color(0xFF0EA5E9).withOpacity(0.1) : Colors.white.withOpacity(0.03),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: isSelected ? const Color(0xFF0EA5E9).withOpacity(0.4) : Colors.white.withOpacity(0.08),
                    width: isSelected ? 1.5 : 1,
                  ),
                ),
                child: Row(
                  children: [
                    Icon(d['icon'] as IconData, size: 16, color: isSelected ? const Color(0xFF0EA5E9) : Colors.white.withOpacity(0.4)),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(d['label'] as String, style: TextStyle(color: isSelected ? Colors.white : Colors.white.withOpacity(0.7), fontWeight: FontWeight.w700, fontSize: 13)),
                          Text(d['desc'] as String, style: TextStyle(color: Colors.white.withOpacity(0.35), fontSize: 11)),
                        ],
                      ),
                    ),
                    if (isSelected) const Icon(Icons.check_circle_rounded, size: 16, color: Color(0xFF0EA5E9)),
                  ],
                ),
              ),
            ),
          );
        })),
        const SizedBox(height: 16),

        // ── Extend seconds ──
        buildSectionLabel('Extend By'),
        Row(
          children: _extendSeconds.map((s) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: buildChip(
              label: '${s}s',
              selected: _extendSecs == s,
              onTap: () => setState(() => _extendSecs = s),
              activeColor: const Color(0xFF0EA5E9),
            ),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Match style toggle ──
        GestureDetector(
          onTap: () => setState(() => _matchStyle = !_matchStyle),
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 150),
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            decoration: BoxDecoration(
              color: _matchStyle ? const Color(0xFF0EA5E9).withOpacity(0.08) : Colors.white.withOpacity(0.03),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: _matchStyle ? const Color(0xFF0EA5E9).withOpacity(0.3) : Colors.white.withOpacity(0.08),
              ),
            ),
            child: Row(
              children: [
                Icon(Icons.auto_awesome_rounded, size: 16, color: _matchStyle ? const Color(0xFF0EA5E9) : Colors.white.withOpacity(0.4)),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Match Original Style', style: TextStyle(color: _matchStyle ? Colors.white : Colors.white.withOpacity(0.6), fontWeight: FontWeight.w700, fontSize: 13)),
                      Text('Maintain visual consistency with the original', style: TextStyle(color: Colors.white.withOpacity(0.35), fontSize: 11)),
                    ],
                  ),
                ),
                Switch(
                  value: _matchStyle,
                  onChanged: (v) => setState(() => _matchStyle = v),
                  activeColor: const Color(0xFF0EA5E9),
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 16),

        // ── Continuation prompt ──
        CollapsibleSection(
          title: 'Continuation Instructions (optional)',
          child: buildTextArea(
            controller: _promptCtrl,
            placeholder: 'Describe what should happen in the extended portion…',
            maxLines: 3,
          ),
        ),
        const SizedBox(height: 24),

        buildGenerateButton(
          label: p.isLoading ? 'Extending…' : 'Extend Video${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF0284C7), Color(0xFF0EA5E9)],
          icon: Icons.expand_rounded,
        ),

        if (!p.canAfford) ...[
          const SizedBox(height: 8),
          Center(child: Text('You need ${p.pointCost} Pulse Points to use this tool', style: TextStyle(color: Colors.red.withOpacity(0.7), fontSize: 12))),
        ],
      ],
    );
  }
}
