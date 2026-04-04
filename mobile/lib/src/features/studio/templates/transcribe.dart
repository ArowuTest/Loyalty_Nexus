// ─── Transcribe Template ──────────────────────────────────────────────────────
// Mirrors webapp Transcribe.tsx exactly.
// Supports: transcribe, african-transcribe
// Payload: audio_url, language, extra_params { output_format, speaker_labels, timestamps }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:file_picker/file_picker.dart';
import 'package:record/record.dart';
import 'package:path_provider/path_provider.dart';
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _defaultLanguages = [
  {'code': 'auto', 'label': 'Auto-detect',  'flag': '🌍'},
  {'code': 'en',   'label': 'English',       'flag': '🇬🇧'},
  {'code': 'yo',   'label': 'Yoruba',        'flag': '🇳🇬'},
  {'code': 'ha',   'label': 'Hausa',         'flag': '🇳🇬'},
  {'code': 'ig',   'label': 'Igbo',          'flag': '🇳🇬'},
  {'code': 'pcm',  'label': 'Pidgin',        'flag': '🇳🇬'},
  {'code': 'fr',   'label': 'French',        'flag': '🇫🇷'},
  {'code': 'sw',   'label': 'Swahili',       'flag': '🇰🇪'},
  {'code': 'am',   'label': 'Amharic',       'flag': '🇪🇹'},
  {'code': 'ar',   'label': 'Arabic',        'flag': '🇸🇦'},
];

const _outputFormats = [
  {'value': 'text',  'label': 'Plain Text',  'icon': Icons.text_fields_rounded},
  {'value': 'srt',   'label': 'SRT Subtitles','icon': Icons.subtitles_rounded},
  {'value': 'vtt',   'label': 'VTT Captions','icon': Icons.closed_caption_rounded},
  {'value': 'json',  'label': 'JSON',        'icon': Icons.data_object_rounded},
];

class TranscribeTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const TranscribeTemplate({super.key, required this.props});

  @override
  ConsumerState<TranscribeTemplate> createState() => _TranscribeTemplateState();
}

class _TranscribeTemplateState extends ConsumerState<TranscribeTemplate> {
  String _language       = 'auto';
  String _outputFormat   = 'text';
  bool   _speakerLabels  = false;
  bool   _timestamps     = false;

  String? _audioUrl;
  String? _audioFileName;
  bool    _isUploading   = false;
  String? _uploadError;

  // Recording
  final AudioRecorder _recorder = AudioRecorder();
  bool   _isRecording   = false;
  String? _recordingPath;

  bool get _isAfricanMode => widget.props.slug == 'african-transcribe';

  Future<void> _pickAudio() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.audio,
      allowMultiple: false,
    );
    if (result == null || result.files.isEmpty) return;
    final file = File(result.files.first.path!);
    setState(() {
      _audioFileName = result.files.first.name;
      _isUploading = true;
      _uploadError = null;
    });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(file);
      setState(() { _audioUrl = url; _isUploading = false; });
    } catch (e) {
      setState(() { _uploadError = 'Upload failed — please try again.'; _isUploading = false; });
    }
  }

  Future<void> _toggleRecording() async {
    if (_isRecording) {
      final path = await _recorder.stop();
      if (path != null) {
        setState(() { _isRecording = false; _recordingPath = path; _isUploading = true; });
        try {
          final studioApi = ref.read(studioApiProvider);
          final url = await studioApi.uploadAsset(File(path));
          setState(() { _audioUrl = url; _audioFileName = 'Recording'; _isUploading = false; });
        } catch (e) {
          setState(() { _uploadError = 'Upload failed.'; _isUploading = false; });
        }
      }
    } else {
      final hasPermission = await _recorder.hasPermission();
      if (!hasPermission) return;
      final dir = await getTemporaryDirectory();
      final path = '${dir.path}/recording_${DateTime.now().millisecondsSinceEpoch}.m4a';
      await _recorder.start(const RecordConfig(), path: path);
      setState(() { _isRecording = true; _recordingPath = null; });
    }
  }

  void _clearAudio() => setState(() {
    _audioUrl = null;
    _audioFileName = null;
    _uploadError = null;
    _recordingPath = null;
  });

  void _handleSubmit() {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _audioUrl == null) return;
    final payload = GeneratePayload(
      prompt: 'Transcribe the audio',
      audioUrl: _audioUrl,
      language: _language,
      extraParams: {
        'output_format':   _outputFormat,
        'speaker_labels':  _speakerLabels,
        'timestamps':      _timestamps,
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _recorder.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final isValid = _audioUrl != null;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        ProviderBadge(
          label: _isAfricanMode ? 'Pollinations Whisper' : 'OpenAI Whisper',
          description: _isAfricanMode
              ? 'Optimised for African languages & Pidgin'
              : 'High-accuracy speech recognition',
          color: const Color(0xFF0EA5E9),
          icon: Icons.mic_rounded,
        ),
        const SizedBox(height: 16),

        // ── Upload or record ──
        buildSectionLabel('Audio File'),
        Row(
          children: [
            Expanded(
              child: GestureDetector(
                onTap: _isUploading ? null : _pickAudio,
                child: Container(
                  padding: const EdgeInsets.symmetric(vertical: 14),
                  decoration: BoxDecoration(
                    color: Colors.white.withOpacity(0.04),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: Colors.white.withOpacity(0.1)),
                  ),
                  child: Column(
                    children: [
                      Icon(Icons.upload_file_rounded, size: 22, color: Colors.white.withOpacity(0.5)),
                      const SizedBox(height: 6),
                      Text(
                        _audioFileName ?? 'Upload Audio',
                        style: TextStyle(
                          color: _audioFileName != null ? Colors.white : Colors.white.withOpacity(0.4),
                          fontSize: 12,
                          fontWeight: _audioFileName != null ? FontWeight.w600 : FontWeight.normal,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      Text(
                        'MP3, WAV, M4A, OGG',
                        style: TextStyle(color: Colors.white.withOpacity(0.25), fontSize: 10),
                      ),
                    ],
                  ),
                ),
              ),
            ),
            const SizedBox(width: 10),
            GestureDetector(
              onTap: _toggleRecording,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                width: 72,
                padding: const EdgeInsets.symmetric(vertical: 14),
                decoration: BoxDecoration(
                  color: _isRecording
                      ? const Color(0xFFEF4444).withOpacity(0.15)
                      : Colors.white.withOpacity(0.04),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: _isRecording
                        ? const Color(0xFFEF4444).withOpacity(0.5)
                        : Colors.white.withOpacity(0.1),
                  ),
                ),
                child: Column(
                  children: [
                    Icon(
                      _isRecording ? Icons.stop_rounded : Icons.mic_rounded,
                      size: 22,
                      color: _isRecording ? const Color(0xFFEF4444) : Colors.white.withOpacity(0.5),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      _isRecording ? 'Stop' : 'Record',
                      style: TextStyle(
                        color: _isRecording ? const Color(0xFFEF4444) : Colors.white.withOpacity(0.4),
                        fontSize: 11,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
        if (_isUploading) ...[
          const SizedBox(height: 8),
          const LinearProgressIndicator(color: Color(0xFF0EA5E9)),
        ],
        if (_uploadError != null) ...[
          const SizedBox(height: 4),
          Text(_uploadError!, style: const TextStyle(color: Color(0xFFEF4444), fontSize: 11)),
        ],
        if (_audioUrl != null) ...[
          const SizedBox(height: 8),
          Row(
            children: [
              const Icon(Icons.check_circle, size: 14, color: Color(0xFF10B981)),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  _audioFileName ?? 'Audio ready',
                  style: const TextStyle(color: Color(0xFF10B981), fontSize: 12),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              GestureDetector(
                onTap: _clearAudio,
                child: Icon(Icons.close, size: 14, color: Colors.white.withOpacity(0.3)),
              ),
            ],
          ),
        ],
        const SizedBox(height: 16),

        // ── Language ──
        buildSectionLabel('Language'),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: _defaultLanguages.map((lang) => buildChip(
            label: '${lang['flag']} ${lang['label']}',
            selected: _language == lang['code'],
            onTap: () => setState(() => _language = lang['code']!),
            activeColor: const Color(0xFF0EA5E9),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Output format ──
        buildSectionLabel('Output Format'),
        Row(
          children: _outputFormats.map((f) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: GestureDetector(
              onTap: () => setState(() => _outputFormat = f['value'] as String),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                decoration: BoxDecoration(
                  color: _outputFormat == f['value']
                      ? const Color(0xFF0EA5E9).withOpacity(0.12)
                      : Colors.white.withOpacity(0.04),
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(
                    color: _outputFormat == f['value']
                        ? const Color(0xFF0EA5E9).withOpacity(0.4)
                        : Colors.white.withOpacity(0.08),
                  ),
                ),
                child: Column(
                  children: [
                    Icon(
                      f['icon'] as IconData,
                      size: 16,
                      color: _outputFormat == f['value']
                          ? const Color(0xFF0EA5E9)
                          : Colors.white.withOpacity(0.4),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      f['label'] as String,
                      style: TextStyle(
                        color: _outputFormat == f['value']
                            ? Colors.white
                            : Colors.white.withOpacity(0.4),
                        fontSize: 10,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Options ──
        buildSectionLabel('Options'),
        Row(
          children: [
            Expanded(
              child: _toggleOption(
                'Speaker Labels',
                Icons.people_rounded,
                _speakerLabels,
                (v) => setState(() => _speakerLabels = v),
              ),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: _toggleOption(
                'Timestamps',
                Icons.access_time_rounded,
                _timestamps,
                (v) => setState(() => _timestamps = v),
              ),
            ),
          ],
        ),
        const SizedBox(height: 24),

        buildGenerateButton(
          label: p.isLoading
              ? 'Transcribing…'
              : 'Transcribe${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF0284C7), Color(0xFF0EA5E9)],
          icon: Icons.mic_rounded,
        ),

        if (!p.canAfford) ...[
          const SizedBox(height: 8),
          Center(
            child: Text(
              'You need ${p.pointCost} Pulse Points to use this tool',
              style: TextStyle(color: Colors.red.withOpacity(0.7), fontSize: 12),
            ),
          ),
        ],
      ],
    );
  }

  Widget _toggleOption(String label, IconData icon, bool value, ValueChanged<bool> onChanged) {
    return GestureDetector(
      onTap: () => onChanged(!value),
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: value ? const Color(0xFF0EA5E9).withOpacity(0.1) : Colors.white.withOpacity(0.04),
          borderRadius: BorderRadius.circular(10),
          border: Border.all(
            color: value ? const Color(0xFF0EA5E9).withOpacity(0.4) : Colors.white.withOpacity(0.08),
          ),
        ),
        child: Row(
          children: [
            Icon(icon, size: 14, color: value ? const Color(0xFF0EA5E9) : Colors.white.withOpacity(0.4)),
            const SizedBox(width: 6),
            Expanded(
              child: Text(
                label,
                style: TextStyle(
                  color: value ? Colors.white : Colors.white.withOpacity(0.45),
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            Icon(
              value ? Icons.check_box_rounded : Icons.check_box_outline_blank_rounded,
              size: 14,
              color: value ? const Color(0xFF0EA5E9) : Colors.white.withOpacity(0.2),
            ),
          ],
        ),
      ),
    );
  }
}
