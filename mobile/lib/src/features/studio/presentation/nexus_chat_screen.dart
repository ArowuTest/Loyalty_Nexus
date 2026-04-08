// ══════════════════════════════════════════════════════════════════════════════
// NexusChatScreen — ChatGPT-style chat UI for all AI Studio chat tools
// Mirrors: frontend/src/components/studio/NexusChatUI.tsx
// ══════════════════════════════════════════════════════════════════════════════

import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:file_picker/file_picker.dart';
import 'package:image_picker/image_picker.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';
import 'studio_screen.dart';

// ─── Tool Config ──────────────────────────────────────────────────────────────

class _ChatToolConfig {
  final String label;
  final String icon;
  final Color color;
  final List<String> suggestions;
  const _ChatToolConfig({
    required this.label,
    required this.icon,
    required this.color,
    required this.suggestions,
  });
}

const _kToolConfigs = <String, _ChatToolConfig>{
  'ask-nexus': _ChatToolConfig(
    label: 'Ask Nexus', icon: '✦', color: Color(0xFFa78bfa),
    suggestions: [
      'What can I do with my MTN points?',
      'How do I earn more Pulse Points?',
      'Explain loyalty tiers',
      'Show me the best deals today',
    ],
  ),
  'nexus-chat': _ChatToolConfig(
    label: 'Nexus Chat', icon: '✦', color: Color(0xFFa78bfa),
    suggestions: [
      'Chat with Nexus AI',
      'Help me brainstorm ideas',
      'Summarise this for me',
      'Write a professional email',
    ],
  ),
  'code-helper': _ChatToolConfig(
    label: 'Code Helper', icon: '⌥', color: Color(0xFF86efac),
    suggestions: [
      'Fix a bug in my code',
      'Explain this function',
      'Write a REST API in Python',
      'Review my Flutter widget',
    ],
  ),
  'code-pro': _ChatToolConfig(
    label: 'Code Pro', icon: '⌥', color: Color(0xFF86efac),
    suggestions: [
      'Refactor for performance',
      'Generate unit tests',
      'Design a database schema',
      'Implement a sorting algorithm',
    ],
  ),
  'research-brief': _ChatToolConfig(
    label: 'Research Brief', icon: '◎', color: Color(0xFF67e8f9),
    suggestions: [
      'Research the Nigerian fintech landscape',
      'Give me a brief on renewable energy',
      'Summarise recent AI developments',
      'Find key statistics on mobile payments',
    ],
  ),
  'deep-research-brief': _ChatToolConfig(
    label: 'Deep Research', icon: '◎', color: Color(0xFF67e8f9),
    suggestions: [
      'Deep dive into blockchain in Africa',
      'Analyse Nigerian e-commerce growth',
      'Research electric vehicles in 2026',
      'Comprehensive overview of AI in healthcare',
    ],
  ),
  'mind-map': _ChatToolConfig(
    label: 'Mind Map', icon: '⬡', color: Color(0xFFfbbf24),
    suggestions: [
      'Create a mind map for a startup idea',
      'Map out a marketing strategy',
      'Visualise a project plan',
      'Organise my study topics',
    ],
  ),
  'mindmap': _ChatToolConfig(
    label: 'Mind Map', icon: '⬡', color: Color(0xFFfbbf24),
    suggestions: [
      'Create a mind map for a startup idea',
      'Map out a marketing strategy',
      'Visualise a project plan',
      'Organise my study topics',
    ],
  ),
  'study-guide': _ChatToolConfig(
    label: 'Study Guide', icon: '📖', color: Color(0xFFf472b6),
    suggestions: [
      'Create a study guide for biology',
      'Make revision notes on calculus',
      'Break down the French Revolution',
      'Explain photosynthesis step by step',
    ],
  ),
  'quiz': _ChatToolConfig(
    label: 'Quiz Me', icon: '❓', color: Color(0xFFf97316),
    suggestions: [
      'Quiz me on Nigerian history',
      'Test my JavaScript knowledge',
      'Give me a science trivia quiz',
      '10 questions on world geography',
    ],
  ),
  'quiz-me': _ChatToolConfig(
    label: 'Quiz Me', icon: '❓', color: Color(0xFFf97316),
    suggestions: [
      'Quiz me on Nigerian history',
      'Test my JavaScript knowledge',
      'Give me a science trivia quiz',
      '10 questions on world geography',
    ],
  ),
  'bizplan': _ChatToolConfig(
    label: 'Business Plan', icon: '📊', color: Color(0xFF34d399),
    suggestions: [
      'Write a business plan for a food delivery app',
      'Create a lean canvas for my startup',
      'Draft an executive summary',
      'Build a go-to-market strategy',
    ],
  ),
  'business-plan-summary': _ChatToolConfig(
    label: 'Business Plan', icon: '📊', color: Color(0xFF34d399),
    suggestions: [
      'Summarise my business idea',
      'Create a one-page business summary',
      'Pitch deck outline for investors',
      'SWOT analysis for my business',
    ],
  ),
  'doc-analyzer': _ChatToolConfig(
    label: 'Doc Analyser', icon: '📄', color: Color(0xFFa5b4fc),
    suggestions: [
      'Analyse this contract for risks',
      'Summarise this PDF',
      'Extract key points from this report',
      'Translate and explain this document',
    ],
  ),
  'voice-to-plan': _ChatToolConfig(
    label: 'Voice to Plan', icon: '🎙️', color: Color(0xFFfb7185),
    suggestions: [
      'Convert my voice note to a plan',
      'Turn my idea into action items',
      'Transcribe and structure my thoughts',
      'Voice memo to project brief',
    ],
  ),
  'local-translation': _ChatToolConfig(
    label: 'Local Translation', icon: '🌍', color: Color(0xFF22d3ee),
    suggestions: [
      'Translate to Yoruba',
      'Translate to Igbo',
      'Translate to Hausa',
      'Translate this sentence to Pidgin',
    ],
  ),
  'web-search-ai': _ChatToolConfig(
    label: 'Web Search AI', icon: '⚡', color: Color(0xFF38bdf8),
    suggestions: [
      'Search latest MTN Nigeria news',
      'What happened in tech today?',
      'Find the best smartphones under ₦200k',
      'Search for Nigerian startups 2026',
    ],
  ),
  'nexus-agent': _ChatToolConfig(
    label: 'Nexus Agent', icon: '⬡', color: Color(0xFFc084fc),
    suggestions: [
      'Automate my daily tasks',
      'Create a workflow for me',
      'Research and compile a report',
      'Help me manage my schedule',
    ],
  ),
  'localize-ui': _ChatToolConfig(
    label: 'Localise UI', icon: '🔤', color: Color(0xFF6ee7b7),
    suggestions: [
      'Localise this UI to Yoruba',
      'Adapt copy for Nigerian market',
      'Generate Hausa UI strings',
      'Translate app labels to Igbo',
    ],
  ),
  'image-analyser': _ChatToolConfig(
    label: 'Image Analyser', icon: '👁', color: Color(0xFFe879f9),
    suggestions: [
      'Describe what is in this image',
      'Read the text in this photo',
      'Analyse the chart in this image',
      'What emotions does this image convey?',
    ],
  ),
};

_ChatToolConfig _configFor(String slug) =>
    _kToolConfigs[slug] ??
    const _ChatToolConfig(
      label: 'Nexus AI', icon: '✦', color: Color(0xFFa78bfa),
      suggestions: [
        'What can you help me with?',
        'Tell me something interesting',
        'Help me write something',
        'Explain a concept',
      ],
    );

// ─── Message model ────────────────────────────────────────────────────────────

enum _MsgRole { user, ai }

class _Msg {
  final _MsgRole role;
  final String content;
  final DateTime time;
  const _Msg({required this.role, required this.content, required this.time});
}

// ══════════════════════════════════════════════════════════════════════════════
// NexusChatScreen
// ══════════════════════════════════════════════════════════════════════════════

class NexusChatScreen extends ConsumerStatefulWidget {
  final StudioTool tool;
  const NexusChatScreen({super.key, required this.tool});

  @override
  ConsumerState<NexusChatScreen> createState() => _NexusChatScreenState();
}

class _NexusChatScreenState extends ConsumerState<NexusChatScreen>
    with TickerProviderStateMixin {
  late final _ChatToolConfig _cfg;
  final _messages = <_Msg>[];
  final _ctrl = TextEditingController();
  final _scrollCtrl = ScrollController();
  String? _sessionId;
  bool _isTyping = false;
  String _streamingText = '';
  bool _isStreaming = false;
  Timer? _streamTimer;

  // Typing indicator animation
  late AnimationController _dotAnim;

  @override
  void initState() {
    super.initState();
    _cfg = _configFor(widget.tool.slug);
    _dotAnim = AnimationController(
      vsync: this, duration: const Duration(milliseconds: 900))
      ..repeat();
    _loadSession();
  }

  @override
  void dispose() {
    _ctrl.dispose();
    _scrollCtrl.dispose();
    _dotAnim.dispose();
    _streamTimer?.cancel();
    super.dispose();
  }

  Future<void> _loadSession() async {
    final prefs = await SharedPreferences.getInstance();
    final key = 'nexus_chat_session_${widget.tool.slug}';
    setState(() => _sessionId = prefs.getString(key));
  }

  Future<void> _saveSession(String id) async {
    final prefs = await SharedPreferences.getInstance();
    final key = 'nexus_chat_session_${widget.tool.slug}';
    await prefs.setString(key, id);
    _sessionId = id;
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollCtrl.hasClients) {
        _scrollCtrl.animateTo(
          _scrollCtrl.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        );
      }
    });
  }

  Future<void> _send({String? override}) async {
    final text = (override ?? _ctrl.text).trim();
    if (text.isEmpty || _isTyping || _isStreaming) return;

    _ctrl.clear();
    setState(() {
      _messages.add(_Msg(role: _MsgRole.user, content: text, time: DateTime.now()));
      _isTyping = true;
    });
    _scrollToBottom();

    try {
      final api = ref.read(studioApiProvider);
      final res = await api.sendChat(
        text,
        sessionId: _sessionId,
        toolSlug: widget.tool.slug,
      );
      final reply = res['message']?.toString() ??
          res['response']?.toString() ??
          res['text']?.toString() ??
          'I\'m here to help! Ask me anything.';
      final newSession = res['session_id']?.toString();
      if (newSession != null && newSession != _sessionId) {
        await _saveSession(newSession);
      }
      setState(() => _isTyping = false);
      _startStreaming(reply);
    } catch (e) {
      setState(() {
        _isTyping = false;
        _messages.add(_Msg(
          role: _MsgRole.ai,
          content: 'Sorry, something went wrong. Please try again.',
          time: DateTime.now(),
        ));
      });
      _scrollToBottom();
    }
  }

  void _startStreaming(String fullText) {
    final words = fullText.split(' ');
    int i = 0;
    setState(() {
      _isStreaming = true;
      _streamingText = '';
    });
    _streamTimer = Timer.periodic(const Duration(milliseconds: 35), (t) {
      if (i >= words.length) {
        t.cancel();
        setState(() {
          _isStreaming = false;
          _messages.add(_Msg(role: _MsgRole.ai, content: fullText, time: DateTime.now()));
          _streamingText = '';
        });
        _scrollToBottom();
        return;
      }
      setState(() {
        _streamingText += (i == 0 ? '' : ' ') + words[i];
        i++;
      });
      _scrollToBottom();
    });
  }

  Future<void> _pickFile() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['pdf', 'doc', 'docx', 'txt'],
    );
    if (result != null && result.files.isNotEmpty) {
      final name = result.files.first.name;
      _send(override: '[Attached file: $name] Please analyse this document.');
    }
  }

  Future<void> _pickImage() async {
    final picker = ImagePicker();
    final img = await picker.pickImage(source: ImageSource.gallery, imageQuality: 80);
    if (img != null) {
      _send(override: '[Attached image: ${img.name}] Please analyse this image.');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0C0C10),
      appBar: _buildAppBar(),
      body: Column(
        children: [
          Expanded(child: _buildMessageList()),
          if (_isTyping) _buildTypingIndicator(),
          _buildInputBar(),
        ],
      ),
    );
  }

  AppBar _buildAppBar() {
    return AppBar(
      backgroundColor: const Color(0xFF0C0C10),
      elevation: 0,
      leading: IconButton(
        icon: const Icon(Icons.arrow_back_rounded, color: NexusColors.textPrimary),
        onPressed: () => Navigator.of(context).pop(),
      ),
      title: Row(children: [
        _ToolAvatar(icon: _cfg.icon, color: _cfg.color, size: 32),
        const SizedBox(width: 10),
        Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(_cfg.label,
            style: const TextStyle(
              fontSize: 15, fontWeight: FontWeight.w700,
              color: NexusColors.textPrimary)),
          Row(children: [
            Container(
              width: 7, height: 7,
              decoration: const BoxDecoration(
                color: NexusColors.green,
                shape: BoxShape.circle,
              ),
            ),
            const SizedBox(width: 4),
            const Text('Online',
              style: TextStyle(fontSize: 11, color: NexusColors.green)),
          ]),
        ]),
      ]),
      actions: [
        IconButton(
          icon: const Icon(Icons.more_vert, color: NexusColors.textSecondary, size: 20),
          onPressed: () => _showMenu(context),
        ),
      ],
    );
  }

  void _showMenu(BuildContext ctx) {
    showModalBottomSheet(
      context: ctx,
      backgroundColor: NexusColors.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (_) => SafeArea(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          ListTile(
            leading: const Icon(Icons.delete_outline, color: NexusColors.red),
            title: const Text('Clear conversation',
              style: TextStyle(color: NexusColors.textPrimary)),
            onTap: () {
              Navigator.pop(ctx);
              setState(() { _messages.clear(); _sessionId = null; });
            },
          ),
          ListTile(
            leading: const Icon(Icons.copy_rounded, color: NexusColors.textSecondary),
            title: const Text('Copy last reply',
              style: TextStyle(color: NexusColors.textPrimary)),
            onTap: () {
              Navigator.pop(ctx);
              final last = _messages.lastWhere(
                (m) => m.role == _MsgRole.ai, orElse: () => _messages.last);
              Clipboard.setData(ClipboardData(text: last.content));
              ScaffoldMessenger.of(ctx).showSnackBar(
                const SnackBar(content: Text('Copied to clipboard')));
            },
          ),
        ]),
      ),
    );
  }

  Widget _buildMessageList() {
    final showEmpty = _messages.isEmpty && !_isTyping && !_isStreaming;
    return showEmpty
        ? _buildEmptyState()
        : ListView.builder(
            controller: _scrollCtrl,
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            itemCount: _messages.length + (_isStreaming ? 1 : 0),
            itemBuilder: (_, i) {
              if (_isStreaming && i == _messages.length) {
                return _buildAiMessage(_streamingText, streaming: true);
              }
              final msg = _messages[i];
              return msg.role == _MsgRole.user
                  ? _buildUserMessage(msg.content)
                  : _buildAiMessage(msg.content);
            },
          );
  }

  Widget _buildEmptyState() {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(children: [
        const SizedBox(height: 32),
        _ToolAvatar(icon: _cfg.icon, color: _cfg.color, size: 56),
        const SizedBox(height: 16),
        Text('Hello! I\'m ${_cfg.label}',
          style: const TextStyle(
            fontSize: 20, fontWeight: FontWeight.w700,
            color: NexusColors.textPrimary)),
        const SizedBox(height: 8),
        Text('Ask me anything or try one of the suggestions below',
          style: const TextStyle(fontSize: 14, color: NexusColors.textSecondary),
          textAlign: TextAlign.center),
        const SizedBox(height: 32),
        GridView.count(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          crossAxisCount: 2,
          mainAxisSpacing: 10,
          crossAxisSpacing: 10,
          childAspectRatio: 2.2,
          children: _cfg.suggestions.map((s) => _SuggestionCard(
            text: s,
            color: _cfg.color,
            onTap: () => _send(override: s),
          )).toList(),
        ),
      ]),
    );
  }

  Widget _buildUserMessage(String text) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          Flexible(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
              decoration: BoxDecoration(
                color: Colors.white.withOpacity(0.09),
                borderRadius: const BorderRadius.all(Radius.circular(18)),
                border: Border.all(color: Colors.white.withOpacity(0.08)),
              ),
              child: _RichText(text: text, baseColor: NexusColors.textPrimary),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildAiMessage(String text, {bool streaming = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _ToolAvatar(icon: _cfg.icon, color: _cfg.color, size: 28),
          const SizedBox(width: 10),
          Flexible(
            child: GestureDetector(
              onLongPress: () {
                Clipboard.setData(ClipboardData(text: text));
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Message copied')));
              },
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _RichText(text: text, baseColor: NexusColors.textPrimary),
                  if (streaming)
                    Container(
                      margin: const EdgeInsets.only(top: 4),
                      width: 6, height: 14,
                      color: _cfg.color,
                    ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildTypingIndicator() {
    return Padding(
      padding: const EdgeInsets.only(left: 16, bottom: 8),
      child: Row(children: [
        _ToolAvatar(icon: _cfg.icon, color: _cfg.color, size: 28),
        const SizedBox(width: 10),
        AnimatedBuilder(
          animation: _dotAnim,
          builder: (_, __) => Row(children: List.generate(3, (i) {
            final delay = i * 0.25;
            final t = (_dotAnim.value + delay) % 1.0;
            final scale = 0.6 + 0.4 * (t < 0.5 ? t * 2 : (1 - t) * 2);
            return Container(
              margin: const EdgeInsets.symmetric(horizontal: 2),
              width: 7 * scale, height: 7 * scale,
              decoration: BoxDecoration(
                color: _cfg.color.withOpacity(0.8),
                shape: BoxShape.circle,
              ),
            );
          })),
        ),
      ]),
    );
  }

  Widget _buildInputBar() {
    return Container(
      decoration: const BoxDecoration(
        color: Color(0xFF0C0C10),
        border: Border(top: BorderSide(color: Color(0x1AFFFFFF))),
      ),
      padding: EdgeInsets.only(
        left: 12, right: 12, top: 10,
        bottom: MediaQuery.of(context).viewInsets.bottom + 10,
      ),
      child: Row(children: [
        IconButton(
          icon: Icon(Icons.attach_file_rounded,
            color: NexusColors.textSecondary.withOpacity(0.7), size: 20),
          onPressed: _pickFile,
        ),
        IconButton(
          icon: Icon(Icons.image_outlined,
            color: NexusColors.textSecondary.withOpacity(0.7), size: 20),
          onPressed: _pickImage,
        ),
        Expanded(
          child: Container(
            decoration: BoxDecoration(
              color: const Color(0xFF1A1A2E),
              borderRadius: BorderRadius.circular(22),
              border: Border.all(color: const Color(0x1AFFFFFF)),
            ),
            child: TextField(
              controller: _ctrl,
              maxLines: 4,
              minLines: 1,
              style: const TextStyle(color: NexusColors.textPrimary, fontSize: 14),
              decoration: const InputDecoration(
                hintText: 'Message…',
                hintStyle: TextStyle(color: NexusColors.textMuted),
                border: InputBorder.none,
                contentPadding: EdgeInsets.symmetric(horizontal: 14, vertical: 10),
              ),
              onSubmitted: (_) => _send(),
              textInputAction: TextInputAction.send,
            ),
          ),
        ),
        const SizedBox(width: 8),
        GestureDetector(
          onTap: _send,
          child: Container(
            width: 40, height: 40,
            decoration: BoxDecoration(
              gradient: LinearGradient(colors: [_cfg.color, _cfg.color.withOpacity(0.7)]),
              shape: BoxShape.circle,
            ),
            child: const Icon(Icons.send_rounded, color: Colors.white, size: 18),
          ),
        ),
      ]),
    );
  }
}

// ─── Tool Avatar ──────────────────────────────────────────────────────────────

class _ToolAvatar extends StatelessWidget {
  final String icon;
  final Color color;
  final double size;
  const _ToolAvatar({required this.icon, required this.color, required this.size});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size, height: size,
      decoration: BoxDecoration(
        color: color.withOpacity(0.15),
        borderRadius: BorderRadius.circular(size * 0.28),
        border: Border.all(color: color.withOpacity(0.3)),
      ),
      child: Center(
        child: Text(icon,
          style: TextStyle(fontSize: size * 0.44, height: 1)),
      ),
    );
  }
}

// ─── Suggestion Card ──────────────────────────────────────────────────────────

class _SuggestionCard extends StatelessWidget {
  final String text;
  final Color color;
  final VoidCallback onTap;
  const _SuggestionCard({required this.text, required this.color, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: color.withOpacity(0.08),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: color.withOpacity(0.2)),
        ),
        child: Text(text,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            fontSize: 12, color: color, fontWeight: FontWeight.w500, height: 1.3)),
      ),
    );
  }
}

// ─── Rich Text Renderer ───────────────────────────────────────────────────────

class _RichText extends StatelessWidget {
  final String text;
  final Color baseColor;
  const _RichText({required this.text, required this.baseColor});

  @override
  Widget build(BuildContext context) {
    // Simple rich text: parse **bold**, `code`, bullet lines, numbered lists
    final lines = text.split('\n');
    final spans = <InlineSpan>[];

    for (var li = 0; li < lines.length; li++) {
      if (li > 0) spans.add(const TextSpan(text: '\n'));
      final line = lines[li];

      // Code block start/end
      if (line.startsWith('```')) {
        spans.add(TextSpan(
          text: line,
          style: TextStyle(
            fontFamily: 'monospace', fontSize: 13,
            color: const Color(0xFF86efac),
            backgroundColor: const Color(0xFF1A1A2E),
          ),
        ));
        continue;
      }

      // Heading
      if (line.startsWith('### ')) {
        spans.add(TextSpan(
          text: line.substring(4),
          style: TextStyle(
            fontSize: 16, fontWeight: FontWeight.w700, color: baseColor),
        ));
        continue;
      }
      if (line.startsWith('## ')) {
        spans.add(TextSpan(
          text: line.substring(3),
          style: TextStyle(
            fontSize: 17, fontWeight: FontWeight.w800, color: baseColor),
        ));
        continue;
      }
      if (line.startsWith('# ')) {
        spans.add(TextSpan(
          text: line.substring(2),
          style: TextStyle(
            fontSize: 18, fontWeight: FontWeight.w900, color: baseColor),
        ));
        continue;
      }

      // Bullet
      String lineText = line;
      if (line.startsWith('- ') || line.startsWith('• ')) {
        lineText = '• ${line.substring(2)}';
      }

      // Inline parsing: **bold** and `code`
      spans.addAll(_parseInline(lineText, baseColor));
    }

    return Text.rich(TextSpan(children: spans,
      style: TextStyle(color: baseColor, fontSize: 14, height: 1.55)));
  }

  List<InlineSpan> _parseInline(String line, Color base) {
    final result = <InlineSpan>[];
    // Match **bold** or `code`
    final pattern = RegExp(r'\*\*(.+?)\*\*|`(.+?)`');
    int last = 0;
    for (final m in pattern.allMatches(line)) {
      if (m.start > last) {
        result.add(TextSpan(text: line.substring(last, m.start),
          style: TextStyle(color: base)));
      }
      if (m.group(1) != null) {
        result.add(TextSpan(text: m.group(1),
          style: TextStyle(
            color: base, fontWeight: FontWeight.w700)));
      } else if (m.group(2) != null) {
        result.add(TextSpan(text: m.group(2),
          style: const TextStyle(
            fontFamily: 'monospace', fontSize: 13,
            color: Color(0xFF86efac),
            backgroundColor: Color(0xFF1A1A2E),
          )));
      }
      last = m.end;
    }
    if (last < line.length) {
      result.add(TextSpan(text: line.substring(last),
        style: TextStyle(color: base)));
    }
    return result.isEmpty
        ? [TextSpan(text: line, style: TextStyle(color: base))]
        : result;
  }
}
