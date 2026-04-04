// ─── Knowledge Doc Template ───────────────────────────────────────────────────
// Mirrors webapp KnowledgeDoc.tsx exactly.
// Supports: knowledge-doc, doc-writer, research-assistant
// Payload: prompt, document_url (optional), extra_params { doc_format, tone, length, language }

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:file_picker/file_picker.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _docFormats = [
  {'value': 'report',       'label': 'Report',          'icon': Icons.description_rounded},
  {'value': 'essay',        'label': 'Essay',           'icon': Icons.article_rounded},
  {'value': 'summary',      'label': 'Summary',         'icon': Icons.summarize_rounded},
  {'value': 'proposal',     'label': 'Proposal',        'icon': Icons.request_quote_rounded},
  {'value': 'letter',       'label': 'Letter',          'icon': Icons.mail_rounded},
  {'value': 'blog',         'label': 'Blog Post',       'icon': Icons.edit_note_rounded},
  {'value': 'presentation', 'label': 'Presentation',    'icon': Icons.slideshow_rounded},
  {'value': 'analysis',     'label': 'Analysis',        'icon': Icons.analytics_rounded},
];

const _toneOptions = [
  {'value': 'professional', 'label': 'Professional', 'emoji': '💼'},
  {'value': 'academic',     'label': 'Academic',     'emoji': '🎓'},
  {'value': 'casual',       'label': 'Casual',       'emoji': '😊'},
  {'value': 'persuasive',   'label': 'Persuasive',   'emoji': '🎯'},
  {'value': 'creative',     'label': 'Creative',     'emoji': '✨'},
  {'value': 'technical',    'label': 'Technical',    'emoji': '⚙️'},
];

const _lengthOptions = [
  {'value': 'brief',      'label': 'Brief',     'desc': '~200 words'},
  {'value': 'standard',   'label': 'Standard',  'desc': '~500 words'},
  {'value': 'detailed',   'label': 'Detailed',  'desc': '~1000 words'},
  {'value': 'extensive',  'label': 'Extensive', 'desc': '~2000 words'},
];

const _languageOptions = [
  {'code': 'en', 'label': 'English',    'flag': '🇬🇧'},
  {'code': 'yo', 'label': 'Yoruba',     'flag': '🇳🇬'},
  {'code': 'ha', 'label': 'Hausa',      'flag': '🇳🇬'},
  {'code': 'ig', 'label': 'Igbo',       'flag': '🇳🇬'},
  {'code': 'fr', 'label': 'French',     'flag': '🇫🇷'},
  {'code': 'pt', 'label': 'Portuguese', 'flag': '🇵🇹'},
];

const _inspirations = [
  'Write a business proposal for a mobile loyalty rewards startup in Nigeria',
  'Summarise the key trends in African fintech for 2025',
  'Write a professional email requesting a meeting with investors',
];

class KnowledgeDocTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const KnowledgeDocTemplate({super.key, required this.props});

  @override
  ConsumerState<KnowledgeDocTemplate> createState() => _KnowledgeDocTemplateState();
}

class _KnowledgeDocTemplateState extends ConsumerState<KnowledgeDocTemplate> {
  final _promptCtrl = TextEditingController();

  String _docFormat = 'report';
  String _tone      = 'professional';
  String _length    = 'standard';
  String _language  = 'en';

  String? _documentUrl;
  String? _documentName;
  bool    _isUploading = false;
  String? _uploadError;

  final stt.SpeechToText _speech = stt.SpeechToText();
  bool _micAvailable = false;
  bool _micListening = false;

  @override
  void initState() {
    super.initState();
    _initSpeech();
  }

  Future<void> _initSpeech() async {
    final available = await _speech.initialize();
    if (mounted) setState(() => _micAvailable = available);
  }

  void _toggleMic() async {
    if (!_micAvailable) return;
    if (_speech.isListening) {
      await _speech.stop();
      setState(() => _micListening = false);
      return;
    }
    setState(() => _micListening = true);
    await _speech.listen(
      onResult: (result) {
        if (result.finalResult) {
          setState(() {
            _promptCtrl.text = result.recognizedWords;
            _micListening = false;
          });
        }
      },
    );
  }

  Future<void> _pickDocument() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['pdf', 'doc', 'docx', 'txt'],
    );
    if (result == null || result.files.isEmpty) return;
    final file = File(result.files.first.path!);
    setState(() { _documentName = result.files.first.name; _isUploading = true; _uploadError = null; });
    try {
      final studioApi = ref.read(studioApiProvider);
      final url = await studioApi.uploadAsset(file);
      setState(() { _documentUrl = url; _isUploading = false; });
    } catch (e) {
      setState(() { _uploadError = 'Upload failed.'; _isUploading = false; });
    }
  }

  void _clearDocument() => setState(() { _documentUrl = null; _documentName = null; _uploadError = null; });

  void _handleSubmit() {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _promptCtrl.text.trim().isEmpty) return;
    final payload = GeneratePayload(
      prompt: _promptCtrl.text.trim(),
      documentUrl: _documentUrl,
      language: _language,
      extraParams: {
        'doc_format': _docFormat,
        'tone':       _tone,
        'length':     _length,
      },
    );
    p.onSubmit(payload);
  }

  @override
  void dispose() {
    _promptCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final isValid = _promptCtrl.text.trim().isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const ProviderBadge(
          label: 'Gemini Flash',
          description: 'Long-form document generation & analysis',
          color: Color(0xFF10B981),
          icon: Icons.description_rounded,
        ),
        const SizedBox(height: 16),

        // ── Prompt ──
        buildSectionLabel('What do you need?'),
        buildTextArea(
          controller: _promptCtrl,
          placeholder: 'e.g. Write a business proposal for a mobile loyalty rewards startup in Nigeria…',
          maxLines: 4,
          maxLength: 2000,
          onMicTap: _micAvailable ? _toggleMic : null,
          micActive: _micListening,
        ),
        const SizedBox(height: 8),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: _inspirations.map((insp) => Padding(
              padding: const EdgeInsets.only(right: 8),
              child: GestureDetector(
                onTap: () => setState(() => _promptCtrl.text = insp),
                child: Container(
                  constraints: const BoxConstraints(maxWidth: 200),
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  decoration: BoxDecoration(
                    color: Colors.white.withOpacity(0.04),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.white.withOpacity(0.1)),
                  ),
                  child: Text(
                    insp.length > 50 ? '${insp.substring(0, 50)}…' : insp,
                    style: TextStyle(color: Colors.white.withOpacity(0.45), fontSize: 11),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ),
            )).toList(),
          ),
        ),
        const SizedBox(height: 16),

        // ── Document format ──
        buildSectionLabel('Document Format'),
        GridView.builder(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: 4,
            childAspectRatio: 1.1,
            crossAxisSpacing: 8,
            mainAxisSpacing: 8,
          ),
          itemCount: _docFormats.length,
          itemBuilder: (context, i) {
            final fmt = _docFormats[i];
            final isSelected = _docFormat == fmt['value'];
            return GestureDetector(
              onTap: () => setState(() => _docFormat = fmt['value'] as String),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                decoration: BoxDecoration(
                  color: isSelected ? const Color(0xFF10B981).withOpacity(0.12) : Colors.white.withOpacity(0.03),
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(
                    color: isSelected ? const Color(0xFF10B981).withOpacity(0.4) : Colors.white.withOpacity(0.08),
                    width: isSelected ? 1.5 : 1,
                  ),
                ),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(
                      fmt['icon'] as IconData,
                      size: 18,
                      color: isSelected ? const Color(0xFF10B981) : Colors.white.withOpacity(0.4),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      fmt['label'] as String,
                      style: TextStyle(
                        color: isSelected ? Colors.white : Colors.white.withOpacity(0.45),
                        fontSize: 10,
                        fontWeight: FontWeight.w600,
                      ),
                      textAlign: TextAlign.center,
                    ),
                  ],
                ),
              ),
            );
          },
        ),
        const SizedBox(height: 16),

        // ── Tone ──
        buildSectionLabel('Tone'),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: _toneOptions.map((t) => buildChip(
            label: '${t['emoji']} ${t['label']}',
            selected: _tone == t['value'],
            onTap: () => setState(() => _tone = t['value']!),
            activeColor: const Color(0xFF10B981),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Length ──
        buildSectionLabel('Length'),
        Row(
          children: _lengthOptions.map((l) => Expanded(
            child: Padding(
              padding: const EdgeInsets.only(right: 6),
              child: GestureDetector(
                onTap: () => setState(() => _length = l['value']!),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 150),
                  padding: const EdgeInsets.symmetric(vertical: 10),
                  decoration: BoxDecoration(
                    color: _length == l['value']
                        ? const Color(0xFF10B981).withOpacity(0.12)
                        : Colors.white.withOpacity(0.03),
                    borderRadius: BorderRadius.circular(10),
                    border: Border.all(
                      color: _length == l['value']
                          ? const Color(0xFF10B981).withOpacity(0.4)
                          : Colors.white.withOpacity(0.08),
                    ),
                  ),
                  child: Column(
                    children: [
                      Text(
                        l['label']!,
                        style: TextStyle(
                          color: _length == l['value'] ? Colors.white : Colors.white.withOpacity(0.5),
                          fontWeight: FontWeight.w700,
                          fontSize: 11,
                        ),
                      ),
                      Text(
                        l['desc']!,
                        style: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 9),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Language ──
        buildSectionLabel('Output Language'),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: _languageOptions.map((lang) => buildChip(
            label: '${lang['flag']} ${lang['label']}',
            selected: _language == lang['code'],
            onTap: () => setState(() => _language = lang['code']!),
          )).toList(),
        ),
        const SizedBox(height: 16),

        // ── Reference document ──
        CollapsibleSection(
          title: 'Reference Document (optional)',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              GestureDetector(
                onTap: _isUploading ? null : _pickDocument,
                child: Container(
                  padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 16),
                  decoration: BoxDecoration(
                    color: Colors.white.withOpacity(0.04),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: Colors.white.withOpacity(0.1)),
                  ),
                  child: Row(
                    children: [
                      Icon(Icons.upload_file_rounded, size: 18, color: Colors.white.withOpacity(0.4)),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Text(
                          _documentName ?? 'Upload PDF, DOC, DOCX, TXT',
                          style: TextStyle(
                            color: _documentName != null ? Colors.white : Colors.white.withOpacity(0.35),
                            fontSize: 12,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      if (_documentUrl != null)
                        GestureDetector(
                          onTap: _clearDocument,
                          child: Icon(Icons.close, size: 14, color: Colors.white.withOpacity(0.3)),
                        ),
                    ],
                  ),
                ),
              ),
              if (_isUploading) ...[
                const SizedBox(height: 8),
                const LinearProgressIndicator(color: Color(0xFF10B981)),
              ],
              if (_uploadError != null) ...[
                const SizedBox(height: 4),
                Text(_uploadError!, style: const TextStyle(color: Color(0xFFEF4444), fontSize: 11)),
              ],
            ],
          ),
        ),
        const SizedBox(height: 24),

        buildGenerateButton(
          label: p.isLoading
              ? 'Writing…'
              : 'Generate Document${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF059669), Color(0xFF10B981)],
          icon: Icons.description_rounded,
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
}
