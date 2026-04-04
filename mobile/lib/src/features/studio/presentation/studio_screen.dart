import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';
import '../templates/template_registry.dart';

// ══════════════════════════════════════════════════════════════════════════════
// Providers
// ══════════════════════════════════════════════════════════════════════════════

final _toolsProvider = FutureProvider.autoDispose<List<StudioTool>>((ref) async {
  final raw = await ref.read(studioApiProvider).listTools();
  return (raw as List).map((e) => StudioTool.fromMap(e as Map)).toList();
});

final _galleryProvider = StateNotifierProvider.autoDispose<_GalleryNotifier, _GalleryState>(
  (ref) => _GalleryNotifier(ref.read(studioApiProvider)),
);

// Loading state wrapper
class _GalleryState {
  final List<Generation> items;
  final bool loading;
  const _GalleryState({this.items = const [], this.loading = true});
  _GalleryState copyWith({List<Generation>? items, bool? loading}) =>
      _GalleryState(items: items ?? this.items, loading: loading ?? this.loading);
}

class _GalleryNotifier extends StateNotifier<_GalleryState> {
  final StudioApi _api;
  Timer? _poll;
  _GalleryNotifier(this._api) : super(const _GalleryState()) { _fetch(); }

  Future<void> _fetch() async {
    try {
      final raw = await _api.getGallery();
      final items = (raw as List).map((e) => Generation.fromMap(e as Map)).toList();
      state = state.copyWith(items: items, loading: false);
      // Auto-poll if any pending
      if (items.any((g) => g.status == 'pending' || g.status == 'processing')) {
        _poll?.cancel();
        _poll = Timer.periodic(const Duration(seconds: 4), (_) => _fetch());
      } else {
        _poll?.cancel();
      }
    } catch (_) {
      state = state.copyWith(loading: false);
    }
  }

  Future<void> refresh() async {
    _poll?.cancel();
    state = state.copyWith(loading: true);
    await _fetch();
  }

  Future<void> delete(String generationId) async {
    try {
      await _api.deleteGeneration(generationId);
      state = state.copyWith(
        items: state.items.where((g) => g.id != generationId).toList());
    } catch (_) {}
  }

  @override void dispose() { _poll?.cancel(); super.dispose(); }
}

// Note: Chat usage is now loaded directly in _StudioScreenState._loadUsage()
// and tracked locally via _chatUsage state field for optimistic updates.

// ══════════════════════════════════════════════════════════════════════════════
// Data models
// ══════════════════════════════════════════════════════════════════════════════

class StudioTool {
  final String id, slug, name, description, category;
  final int pointCost, entryPointCost;
  final bool isFree;
  final String? uiTemplate;

  const StudioTool({
    required this.id, required this.slug, required this.name,
    required this.description, required this.category,
    required this.pointCost, required this.entryPointCost,
    required this.isFree, this.uiTemplate,
  });

  factory StudioTool.fromMap(Map m) => StudioTool(
    id:             m['id']?.toString()          ?? '',
    slug:           m['slug']?.toString()        ?? '',
    name:           m['name']?.toString()        ?? '',
    description:    m['description']?.toString() ?? '',
    category:       m['category']?.toString()    ?? 'General',
    pointCost:      (m['point_cost'] as num?)?.toInt()       ?? 0,
    entryPointCost: (m['entry_point_cost'] as num?)?.toInt() ?? 0,
    isFree:         m['is_free'] as bool?        ?? false,
    uiTemplate:     m['ui_template']?.toString(),
  );

  bool get isNew => _newSlugs.contains(slug);
  bool get isChatTool =>
      slug == 'web-search-ai' ||
      slug == 'code-helper'   ||
      slug == 'nexus-chat'    ||
      slug == 'ask-nexus'     ||
      slug == 'ai-chat';
  bool get isPremium => pointCost >= 20;

  /// Converts to a Map that TemplateRegistry / TemplateProps can read.
  Map<String, dynamic> toJson() => {
    'id':               id,
    'slug':             slug,
    'name':             name,
    'description':      description,
    'category':         category,
    'point_cost':       pointCost,
    'entry_point_cost': entryPointCost,
    'is_free':          isFree,
    'ui_template':      uiTemplate,
    // ui_config is loaded from the API — the template will fall back to defaults
    // if ui_config is absent. StudioTool doesn't cache it yet; see: startGeneration.
  };
}

// Tool slugs that are new / highlighted
const _newSlugs = {
  'web-search-ai','image-analyser','ask-my-photo','code-helper',
  'narrate-pro','transcribe-african','ai-photo-pro','ai-photo-max',
  'ai-photo-dream','photo-editor','song-creator','instrumental',
  'video-cinematic','video-veo',
};
const _imageSlugs = {
  'ai-photo','ai-photo-pro','ai-photo-max','ai-photo-dream',
  'photo-editor','animate-photo','infographic','image-analyser','ask-my-photo','bg-remover',
  'image-compose','image-composer',
};
const _audioSlugs = {
  'narrate','narrate-pro','bg-music','jingle','song-creator',
  'instrumental','transcribe','transcribe-african','podcast',
};
const _videoSlugs = {
  'animate-photo','video-premium','video-cinematic','video-veo',
  'video-edit','video-editor','video-extend','video-extender',
  'video-multi-scene','video-story','video-jingle',
};
const _codeSlugs  = {'code-helper'};
const _webSlugs   = {'web-search-ai'};
const _jsonSlugs  = {'quiz','mindmap','slide-deck'};

class Generation {
  final String id, toolName, toolSlug, status;
  final String? outputUrl, outputText, prompt, errorMessage;
  final int? pointCost;
  final DateTime createdAt;

  const Generation({
    required this.id, required this.toolName, required this.toolSlug,
    required this.status, required this.createdAt,
    this.outputUrl, this.outputText, this.prompt, this.errorMessage, this.pointCost,
  });

  factory Generation.fromMap(Map m) => Generation(
    id:           m['id']?.toString()          ?? '',
    toolName:     m['tool_name']?.toString()   ?? '',
    toolSlug:     m['tool_slug']?.toString()   ?? '',
    status:       m['status']?.toString()      ?? 'pending',
    outputUrl:    m['output_url']?.toString(),
    outputText:   m['output_text']?.toString(),
    prompt:       m['prompt']?.toString(),
    errorMessage: m['error_message']?.toString(),
    pointCost:    (m['point_cost'] as num?)?.toInt(),
    createdAt:    DateTime.tryParse(m['created_at']?.toString() ?? '') ?? DateTime.now(),
  );

  bool get isImage => _imageSlugs.contains(toolSlug);
  bool get isAudio => _audioSlugs.contains(toolSlug);
  bool get isVideo => _videoSlugs.contains(toolSlug);
  bool get isCode  => _codeSlugs.contains(toolSlug);
  bool get isWeb   => _webSlugs.contains(toolSlug);
  bool get isJson  => _jsonSlugs.contains(toolSlug);

  String get displayPrompt {
    if (prompt == null) return '';
    try {
      final decoded = jsonDecode(prompt!) as Map;
      return decoded['prompt']?.toString() ?? prompt!;
    } catch (_) { return prompt!; }
  }

  String get timeAgo {
    final diff = DateTime.now().difference(createdAt);
    if (diff.inMinutes < 1)  return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24)   return '${diff.inHours}h ago';
    return '${diff.inDays}d ago';
  }
}

// ══════════════════════════════════════════════════════════════════════════════
// Category config (mirrors webapp CAT map)
// ══════════════════════════════════════════════════════════════════════════════

const _catConfig = <String, _CatConfig>{
  'Knowledge & Research': _CatConfig(icon: Icons.book_outlined,      color: Color(0xFF3b82f6)),
  'Image & Visual':       _CatConfig(icon: Icons.image_outlined,      color: Color(0xFFec4899)),
  'Audio & Voice':        _CatConfig(icon: Icons.mic_outlined,        color: Color(0xFF22c55e)),
  'Document & Business':  _CatConfig(icon: Icons.description_outlined,color: Color(0xFFf97316)),
  'Music & Entertainment':_CatConfig(icon: Icons.music_note_outlined, color: Color(0xFF8B5CF6)),
  'Language & Translation':_CatConfig(icon: Icons.language_outlined,  color: Color(0xFF06b6d4)),
  'Video & Animation':    _CatConfig(icon: Icons.videocam_outlined,   color: Color(0xFFef4444)),
  'Vision':               _CatConfig(icon: Icons.psychology_outlined, color: Color(0xFF7c3aed)),
  'Chat':                 _CatConfig(icon: Icons.chat_outlined,       color: Color(0xFF0891b2)),
  'Build':                _CatConfig(icon: Icons.code_outlined,       color: Color(0xFF84cc16)),
  'Create':               _CatConfig(icon: Icons.auto_awesome,        color: Color(0xFFf59e0b)),
};

class _CatConfig {
  final IconData icon;
  final Color color;
  const _CatConfig({required this.icon, required this.color});
}

_CatConfig _catCfg(String cat) =>
  _catConfig[cat] ?? const _CatConfig(icon: Icons.auto_fix_high, color: Color(0xFF6b7280));

// ══════════════════════════════════════════════════════════════════════════════
// Tool meta (time estimates + tips — mirrors TOOL_META in webapp)
// ══════════════════════════════════════════════════════════════════════════════

const _toolMeta = <String, (String time, String tip)>{
  'ai-chat':           ('instant',  'Ask follow-ups to go deeper'),
  'web-search-ai':     ('~5 sec',   "Include 'today' or a date for current info"),
  'ai-photo':          ('~8 sec',   "Add style words: 'photorealistic', 'vibrant', 'cinematic'"),
  'ai-photo-pro':      ('~12 sec',  'Describe lighting, mood, and camera angle'),
  'ai-photo-max':      ('~20 sec',  'Be very detailed — every word affects the result'),
  'ai-photo-dream':    ('~10 sec',  "Try 'Afrofuturist', 'anime', 'oil painting' styles"),
  'photo-editor':      ('~15 sec',  "Be specific: 'remove background and replace with beach sunset'"),
  'image-analyser':    ('~4 sec',   'Works with any public image URL'),
  'animate-photo':     ('~45 sec',  'Use portraits or scenic photos for best motion'),
  'video-cinematic':   ('~90 sec',  "Describe motion: 'slow zoom in', 'camera pan left'"),
  'video-premium':     ('~2 min',   'More detail in prompt = better camera movement'),
  'video-veo':         ('~3 min',   'Describe the scene like a film director would'),
  'narrate':           ('~4 sec',   'Keep text under 500 words for best quality'),
  'narrate-pro':       ('~5 sec',   "Try 'coral' for warm tone, 'onyx' for deep voice"),
  'transcribe':        ('~6 sec',   'Paste a direct link to an MP3 or WAV file'),
  'transcribe-african':('~8 sec',   'Select language BEFORE submitting for accuracy'),
  'bg-music':          ('~30 sec',  "Describe mood: 'calm', 'energetic', 'corporate'"),
  'jingle':            ('~25 sec',  'Add brand name and target emotion in prompt'),
  'song-creator':      ('~2 min',   'Afrobeats, Gospel, Amapiano — be specific about genre'),
  'instrumental':      ('~2 min',   "Describe instruments: 'piano, strings, light percussion'"),
  'code-helper':       ('~5 sec',   'Mention the programming language in your prompt'),
  'study-guide':       ('~8 sec',   "Add 'for WAEC' or 'for university level' for focus"),
  'quiz':              ('~6 sec',   "Specify difficulty: 'easy', 'intermediate', 'expert'"),
  'mindmap':           ('~5 sec',   'One topic at a time gives the best results'),
  'research-brief':    ('~10 sec',  'Be specific about industry or location context'),
  'bizplan':           ('~12 sec',  'Include target city and startup budget for relevance'),
  'slide-deck':        ('~10 sec',  "Add audience type: 'investors', 'students', 'clients'"),
  'podcast':           ('~90 sec',  'Give a clear topic — the AI writes the full script'),
  // New & updated tools
  'image-compose':     ('~20 sec',  'Upload 2–3 images and describe how to blend them'),
  'image-composer':    ('~20 sec',  'Upload 2–3 images and describe how to blend them'),
  'video-edit':        ('~2 min',   'Be specific about the edit — style, colour, effect'),
  'video-editor':      ('~2 min',   'Be specific about the edit — style, colour, effect'),
  'video-extend':      ('~2 min',   'Describe what should happen in the extended portion'),
  'video-extender':    ('~2 min',   'Describe what should happen in the extended portion'),
  'video-multi-scene': ('~3 min',   'Plan each scene — more detail gives better results'),
  'video-story':       ('~3 min',   'Plan each scene — more detail gives better results'),
  'video-jingle':      ('~90 sec',  'Describe the brand, mood and target audience'),
  'ask-my-photo':      ('~4 sec',   'Upload any image and ask a specific question'),
  'doc-writer':        ('~10 sec',  'Be specific about format, audience and purpose'),
  'research-assistant':('~12 sec',  'Include industry, region and scope for best results'),
  'bg-remover':        ('~5 sec',   'Works best with clear subject and contrasting background'),
  'infographic':       ('~15 sec',  'List the key data points you want visualised'),
};

// ══════════════════════════════════════════════════════════════════════════════
// Chat mode
// ══════════════════════════════════════════════════════════════════════════════

enum ChatMode { general, search, code }

extension ChatModeX on ChatMode {
  String get label => switch (this) {
    ChatMode.general => 'General', ChatMode.search => 'Web Search', ChatMode.code => 'Code',
  };
  String get emoji => switch (this) {
    ChatMode.general => '🤖', ChatMode.search => '🔍', ChatMode.code => '💻',
  };
  String get placeholder => switch (this) {
    ChatMode.general => 'Ask Nexus anything…',
    ChatMode.search  => 'Search the web — ask about news, prices, facts…',
    ChatMode.code    => 'Describe what code you need…',
  };
  Color get color => switch (this) {
    ChatMode.general => NexusColors.gold,
    ChatMode.search  => const Color(0xFF0ea5e9),
    ChatMode.code    => const Color(0xFF22c55e),
  };
  String? get toolSlug => switch (this) {
    ChatMode.search => 'web-search-ai',
    ChatMode.code   => 'code-helper',
    ChatMode.general => null,
  };
}

class _Message {
  final String role, content;
  final ChatMode mode;
  final DateTime ts;
  final String? provider;  // AI provider label from backend (e.g. 'gpt-4o-mini')
  const _Message({
    required this.role, required this.content,
    this.mode = ChatMode.general, required this.ts,
    this.provider,
  });
}

// ══════════════════════════════════════════════════════════════════════════════
// Studio screen — main widget
// ══════════════════════════════════════════════════════════════════════════════

class StudioScreen extends ConsumerStatefulWidget {
  const StudioScreen({super.key});
  @override ConsumerState<StudioScreen> createState() => _StudioScreenState();
}

class _StudioScreenState extends ConsumerState<StudioScreen>
    with SingleTickerProviderStateMixin {
  late TabController _tabs;
  static const _storage = FlutterSecureStorage();
  static const _sessionKey = 'nexus_chat_session';

  // ── Chat state ──
  final List<_Message> _messages = [
    _Message(role: 'assistant', ts: DateTime.now(),
      content: "Hey! 👋 I'm Nexus AI — your personal AI assistant. "
               "I can help with business ideas, explain anything, draft content, and more. "
               "What's on your mind?"),
  ];
  ChatMode _chatMode   = ChatMode.general;
  final _chatCtrl      = TextEditingController();
  bool  _sending       = false;
  final _scrollCtrl    = ScrollController();
  String? _sessionId;          // persisted across launches for memory continuity
  Map<String, int>? _chatUsage; // {used, limit}

  // ── Tools state ──
  String  _searchQuery    = '';
  String? _activeCategory;
  StudioTool? _selectedTool;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 3, vsync: this);
    _tabs.addListener(() => setState(() {}));
    _initSession();
    _loadUsage();
  }

  /// Restore or create a session ID for backend memory continuity.
  Future<void> _initSession() async {
    String? stored = await _storage.read(key: _sessionKey);
    stored ??= 'sess_${DateTime.now().millisecondsSinceEpoch}_${math.Random().nextInt(0xFFFFFF).toRadixString(16)}';
    await _storage.write(key: _sessionKey, value: stored);
    if (mounted) setState(() => _sessionId = stored);
  }

  Future<void> _loadUsage() async {
    try {
      final r = await ref.read(studioApiProvider).getChatUsage();
      if (!mounted) return;
      final used  = (r as Map)['used'] as int? ?? 0;
      final limit = (r)['limit'] as int? ?? 100;
      setState(() => _chatUsage = {'used': used, 'limit': limit});
    } catch (_) {}
  }

  @override
  void dispose() { _tabs.dispose(); _chatCtrl.dispose(); _scrollCtrl.dispose(); super.dispose(); }

  // ── Chat send ──
  Future<void> _sendChat() async {
    final text = _chatCtrl.text.trim();
    if (text.isEmpty || _sending) return;
    _chatCtrl.clear();
    setState(() {
      _messages.add(_Message(role: 'user', content: text, mode: _chatMode, ts: DateTime.now()));
      _sending = true;
    });
    _scrollToBottom();
    try {
      final res = await ref.read(studioApiProvider).sendChat(
        text, _chatMode.toolSlug, sessionId: _sessionId);
      final reply    = (res as Map)['response']?.toString() ?? 'No response';
      final provider = res['provider']?.toString();
      final count    = res['message_count'] as int?;
      // Update session ID if backend issues a new one
      final newSid = res['session_id']?.toString();
      if (newSid != null && newSid != _sessionId) {
        _sessionId = newSid;
        await _storage.write(key: _sessionKey, value: newSid);
      }
      setState(() {
        _messages.add(_Message(
          role: 'assistant', content: reply, mode: _chatMode,
          ts: DateTime.now(), provider: provider));
        if (count != null) {
          _chatUsage = {'used': count, 'limit': _chatUsage?['limit'] ?? 100};
        } else if (_chatUsage != null) {
          _chatUsage = {'used': (_chatUsage!['used']! + 1), 'limit': _chatUsage!['limit']!};
        }
      });
    } catch (e) {
      setState(() {
        _messages.add(_Message(role: 'assistant', ts: DateTime.now(),
          content: 'I\'m having trouble connecting right now. Please try again in a moment. 🔄'));
      });
    } finally {
      setState(() => _sending = false);
      _scrollToBottom();
    }
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollCtrl.hasClients) {
        _scrollCtrl.animateTo(_scrollCtrl.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300), curve: Curves.easeOut);
      }
    });
  }

  Future<void> _clearChat() async {
    // Destroy the session so the backend starts fresh memory
    await _storage.delete(key: _sessionKey);
    _sessionId = null;
    await _initSession(); // generate a new one
    setState(() {
      _messages
        ..clear()
        ..add(_Message(role: 'assistant', ts: DateTime.now(),
          content: "Hey! 👋 I'm Nexus AI. What can I help you with?"));
    });
  }

  /// Injects a summarise prompt into the input field
  void _summariseChat() {
    if (_messages.length < 4) return;
    _chatCtrl.text =
        'Please summarise our conversation so far in 3–5 bullet points. '
        'Focus on topics covered and any decisions or advice given.';
    _chatCtrl.selection = TextSelection.fromPosition(
        TextPosition(offset: _chatCtrl.text.length));
    setState(() {});
  }

  // ── Launch tool ──
  void _openTool(StudioTool tool) {
    if (tool.slug == 'web-search-ai') {
      setState(() { _chatMode = ChatMode.search; _tabs.animateTo(0); });
    } else if (tool.slug == 'code-helper') {
      setState(() { _chatMode = ChatMode.code; _tabs.animateTo(0); });
    } else if (tool.slug == 'nexus-chat' ||
               tool.slug == 'ask-nexus'  ||
               tool.slug == 'ai-chat') {
      // Conversational tools → switch to Chat tab in general mode
      setState(() { _chatMode = ChatMode.general; _tabs.animateTo(0); });
    } else {
      setState(() => _selectedTool = tool);
    }
  }

  @override
  Widget build(BuildContext context) {
    final walletAsync = ref.watch(walletProvider);
    final userPoints = walletAsync.asData?.value ?? 0;

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        title: Row(children: [
          Container(
            width: 36, height: 36,
            decoration: BoxDecoration(
              gradient: const LinearGradient(colors: [NexusColors.gold, Color(0xFFd97706)]),
              borderRadius: BorderRadius.circular(10),
            ),
            child: const Icon(Icons.psychology_rounded, color: Colors.white, size: 20),
          ),
          const SizedBox(width: 10),
          Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Nexus AI Studio', style: TextStyle(fontSize: 15)),
            ref.watch(_toolsProvider).when(
              data: (tools) => Text('${tools.length} AI tools',
                style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
              loading: () => const SizedBox.shrink(),
              error: (_, __) => const SizedBox.shrink(),
            ),
          ]),
        ]),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(48),
          child: Container(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
            child: Container(
              padding: const EdgeInsets.all(4),
              decoration: BoxDecoration(
                color: NexusColors.surface,
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: NexusColors.border),
              ),
              child: TabBar(
                controller: _tabs,
                indicator: BoxDecoration(
                  gradient: const LinearGradient(colors: [NexusColors.gold, Color(0xFFd97706)]),
                  borderRadius: BorderRadius.circular(10),
                ),
                labelColor: Colors.white,
                unselectedLabelColor: NexusColors.textSecondary,
                labelStyle: const TextStyle(fontWeight: FontWeight.w700, fontSize: 12),
                unselectedLabelStyle: const TextStyle(fontSize: 12),
                dividerColor: Colors.transparent,
                indicatorSize: TabBarIndicatorSize.tab,
                tabs: [
                  const Tab(text: '💬 Chat'),
                  ref.watch(_toolsProvider).when(
                    data: (t) => Tab(text: '🛠 Tools (${t.length})'),
                    loading: () => const Tab(text: '🛠 Tools'),
                    error: (_, __) => const Tab(text: '🛠 Tools'),
                  ),
                  _buildGalleryTab(),
                ],
              ),
            ),
          ),
        ),
      ),
      body: Column(children: [
        // Wallet bar
        _WalletBar(points: userPoints),
        // Tab content
        Expanded(child: TabBarView(controller: _tabs, children: [
          _ChatTab(
            messages: _messages, mode: _chatMode, sending: _sending,
            controller: _chatCtrl, scrollController: _scrollCtrl,
            onSend: _sendChat, onClear: _clearChat,
            onSummarise: _summariseChat,
            onModeChange: (m) => setState(() => _chatMode = m),
            chatUsage: _chatUsage,
          ),
          _ToolsTab(
            userPoints: userPoints,
            searchQuery: _searchQuery,
            activeCategory: _activeCategory,
            onSearch: (q) => setState(() => _searchQuery = q),
            onCategoryChange: (c) => setState(() => _activeCategory = c),
            onToolTap: _openTool,
          ),
          _GalleryTab(
            galleryState: ref.watch(_galleryProvider),
            onRefresh: () => ref.read(_galleryProvider.notifier).refresh(),
            onDelete: (id) => ref.read(_galleryProvider.notifier).delete(id),
            onRegenerate: (g) {
              ref.read(_toolsProvider).whenData((tools) {
                final tool = tools.firstWhere(
                  (t) => t.slug == g.toolSlug, orElse: () => tools.first);
                _openTool(tool);
                _tabs.animateTo(1);
              });
            },
          ),
        ])),
      ]),
      // Tool drawer overlay
      extendBody: true,
      bottomSheet: _selectedTool != null
          ? _ToolDrawer(
              tool: _selectedTool!,
              userPoints: userPoints,
              onClose: () => setState(() => _selectedTool = null),
              onGenerated: () {
                ref.read(_galleryProvider.notifier).refresh();
                setState(() { _selectedTool = null; });
                _tabs.animateTo(2);
                ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
                  content: Text('⚡ Generating… check Gallery tab for result.'),
                  behavior: SnackBarBehavior.floating,
                  backgroundColor: NexusColors.primary,
                ));
              },
            )
          : null,
    );
  }

  Tab _buildGalleryTab() {
    final gs = ref.watch(_galleryProvider);
    final pending = gs.items.where((g) => g.status == 'pending' || g.status == 'processing').length;
    if (pending > 0) {
      return Tab(child: Row(mainAxisSize: MainAxisSize.min, children: [
        const Text('🖼 Gallery'),
        const SizedBox(width: 4),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
          decoration: BoxDecoration(
            color: NexusColors.primary, borderRadius: BorderRadius.circular(10)),
          child: Text('$pending', style: const TextStyle(color: Colors.white, fontSize: 9)),
        ),
      ]));
    }
    return const Tab(text: '🖼 Gallery');
  }
}

// ══════════════════════════════════════════════════════════════════════════════
// Wallet bar
// ══════════════════════════════════════════════════════════════════════════════

final walletProvider = FutureProvider.autoDispose<int>((ref) async {
  final w = await ref.read(userApiProvider).getWallet();
  return (w as Map)['pulse_points'] as int? ?? 0;
});

class _WalletBar extends StatelessWidget {
  final int points;
  const _WalletBar({required this.points});
  @override
  Widget build(BuildContext context) {
    final isLow = points < 50;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        color: isLow ? const Color(0xFF1c1208) : NexusColors.surface,
        border: Border(bottom: BorderSide(color: NexusColors.border)),
      ),
      child: Row(children: [
        Container(
          width: 28, height: 28,
          decoration: BoxDecoration(
            color: isLow ? const Color(0x33f59e0b) : const Color(0x1AF5A623),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Icon(Icons.bolt_rounded, size: 16,
            color: isLow ? const Color(0xFFfbbf24) : NexusColors.gold),
        ),
        const SizedBox(width: 8),
        Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Text(points.toString().replaceAllMapped(RegExp(r'(\d{1,3})(?=(\d{3})+(?!\d))'), (m) => '${m[1]},'),
              style: TextStyle(
                color: isLow ? const Color(0xFFfbbf24) : Colors.white,
                fontWeight: FontWeight.w800, fontSize: 15)),
            const SizedBox(width: 4),
            const Text('PulsePoints',
              style: TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
          ]),
          const Text('Each generation uses points once',
            style: TextStyle(color: NexusColors.textSecondary, fontSize: 9)),
        ]),
        const Spacer(),
        if (isLow)
          GestureDetector(
            onTap: () {},
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              decoration: BoxDecoration(
                color: const Color(0x33f59e0b),
                borderRadius: BorderRadius.circular(10),
                border: Border.all(color: const Color(0x66f59e0b)),
              ),
              child: const Row(mainAxisSize: MainAxisSize.min, children: [
                Icon(Icons.bolt_rounded, size: 12, color: Color(0xFFfbbf24)),
                SizedBox(width: 4),
                Text('Recharge', style: TextStyle(color: Color(0xFFfbbf24),
                  fontSize: 11, fontWeight: FontWeight.w700)),
              ]),
            ),
          )
        else
          const Row(children: [
            Icon(Icons.trending_up_rounded, size: 11, color: NexusColors.textSecondary),
            SizedBox(width: 4),
            Text('Good balance', style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
          ]),
      ]),
    );
  }
}

// ══════════════════════════════════════════════════════════════════════════════
// Chat tab
// ══════════════════════════════════════════════════════════════════════════════

class _ChatTab extends StatelessWidget {
  final List<_Message> messages;
  final ChatMode mode;
  final bool sending;
  final TextEditingController controller;
  final ScrollController scrollController;
  final VoidCallback onSend, onClear, onSummarise;
  final ValueChanged<ChatMode> onModeChange;
  final Map<String, int>? chatUsage;

  const _ChatTab({
    required this.messages, required this.mode, required this.sending,
    required this.controller, required this.scrollController,
    required this.onSend, required this.onClear, required this.onSummarise,
    required this.onModeChange, required this.chatUsage,
  });

  @override
  Widget build(BuildContext context) {
    final canSummarise = messages.length >= 4;
    return Column(children: [
      // ── Mode switcher ──────────────────────────────────────────────────────
      Padding(
        padding: const EdgeInsets.fromLTRB(16, 10, 16, 0),
        child: Row(children: ChatMode.values.map((m) {
          final active = m == mode;
          return Expanded(child: Padding(
            padding: EdgeInsets.only(right: m != ChatMode.code ? 6 : 0),
            child: GestureDetector(
              onTap: () => onModeChange(m),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                padding: const EdgeInsets.symmetric(vertical: 8),
                decoration: BoxDecoration(
                  color: active ? m.color.withValues(alpha: 0.2) : NexusColors.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: active ? m.color.withValues(alpha: 0.5) : NexusColors.border),
                ),
                child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                  Text(m.emoji, style: const TextStyle(fontSize: 12)),
                  const SizedBox(width: 5),
                  Text(m.label, style: TextStyle(
                    color: active ? m.color : NexusColors.textSecondary,
                    fontSize: 11, fontWeight: FontWeight.w600)),
                ]),
              ),
            ),
          ));
        }).toList()),
      ),

      // ── Mode description + usage counter ───────────────────────────────────
      Padding(
        padding: const EdgeInsets.fromLTRB(20, 6, 20, 0),
        child: Row(children: [
          Expanded(child: Text(
            switch (mode) {
              ChatMode.general => '🤖 General assistant — business, ideas, content, advice',
              ChatMode.search  => '🔍 Live internet — current news, prices, real-time data',
              ChatMode.code    => '💻 Qwen Coder — write, explain, debug any language',
            },
            style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10),
          )),
          if (chatUsage != null)
            Text('${chatUsage!['used']}/${chatUsage!['limit']} msgs',
                style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
        ]),
      ),

      const SizedBox(height: 8),

      // ── Messages list ──────────────────────────────────────────────────────
      Expanded(
        child: ListView.builder(
          controller: scrollController,
          padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
          itemCount: messages.length + (sending ? 1 : 0),
          itemBuilder: (ctx, i) {
            if (sending && i == messages.length) return _ThinkingBubble(mode: mode);
            return _ChatBubble(msg: messages[i])
                .animate(key: ValueKey(messages[i].ts))
                .fadeIn(duration: 250.ms)
                .slideY(begin: 0.05, end: 0);
          },
        ),
      ),

      // ── Input area ─────────────────────────────────────────────────────────
      Container(
        padding: const EdgeInsets.fromLTRB(12, 8, 12, 16),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          border: const Border(top: BorderSide(color: NexusColors.border)),
        ),
        child: Column(children: [
          // Main input row
          Row(crossAxisAlignment: CrossAxisAlignment.end, children: [
            Expanded(
              child: TextField(
                controller: controller,
                onSubmitted: (_) => onSend(),
                maxLines: 4, minLines: 1,
                style: const TextStyle(color: NexusColors.textPrimary, fontSize: 14),
                decoration: InputDecoration(
                  hintText: mode.placeholder,
                  hintStyle: const TextStyle(color: NexusColors.textSecondary, fontSize: 13),
                  filled: true, fillColor: NexusColors.background,
                  contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(14),
                    borderSide: const BorderSide(color: NexusColors.border)),
                  enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(14),
                    borderSide: const BorderSide(color: NexusColors.border)),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(14),
                    borderSide: BorderSide(color: mode.color.withValues(alpha: 0.5))),
                ),
              ),
            ),
            const SizedBox(width: 8),
            GestureDetector(
              onTap: onSend,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                width: 44, height: 44,
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    colors: [mode.color, mode.color.withValues(alpha: 0.7)],
                    begin: Alignment.topLeft, end: Alignment.bottomRight),
                  borderRadius: BorderRadius.circular(14),
                  boxShadow: [BoxShadow(
                    color: mode.color.withValues(alpha: 0.3),
                    blurRadius: 8, offset: const Offset(0, 3))],
                ),
                child: sending
                    ? const Center(child: SizedBox(width: 20, height: 20,
                        child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2)))
                    : const Icon(Icons.send_rounded, color: Colors.white, size: 18),
              ),
            ),
          ]),
          const SizedBox(height: 8),

          // ── Action row: Summarise / New chat ─────────────────────────────
          Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
            // Summarise button — only enabled when ≥4 messages
            GestureDetector(
              onTap: canSummarise ? onSummarise : null,
              child: Row(children: [
                Icon(Icons.summarize_rounded, size: 11,
                  color: canSummarise ? mode.color : NexusColors.textSecondary.withValues(alpha: 0.4)),
                const SizedBox(width: 3),
                Text('Summarise', style: TextStyle(
                  color: canSummarise ? mode.color : NexusColors.textSecondary.withValues(alpha: 0.4),
                  fontSize: 10, fontWeight: FontWeight.w600)),
              ]),
            ),
            // New chat
            GestureDetector(
              onTap: onClear,
              child: const Row(children: [
                Icon(Icons.refresh_rounded, size: 11, color: NexusColors.textSecondary),
                SizedBox(width: 3),
                Text('New chat', style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
              ]),
            ),
          ]),
        ]),
      ),
    ]);
  }
}

// ── Rich message renderer ──────────────────────────────────────────────────────
// Parses: fenced ```lang\ncode\n```, **bold**, `inline`, plain text / newlines

class _RichMessage extends StatefulWidget {
  final String content;
  final ChatMode mode;
  const _RichMessage({required this.content, required this.mode});
  @override State<_RichMessage> createState() => _RichMessageState();
}

class _RichMessageState extends State<_RichMessage> {
  int? _copiedIdx;

  void _copyCode(String code, int idx) {
    Clipboard.setData(ClipboardData(text: code));
    setState(() => _copiedIdx = idx);
    Future.delayed(const Duration(milliseconds: 1800), () {
      if (mounted) setState(() => _copiedIdx = null);
    });
  }

  @override
  Widget build(BuildContext context) {
    // Split by fenced code blocks
    final regex = RegExp(r'```[\s\S]*?```');
    final parts  = widget.content.split(regex);
    final blocks = regex.allMatches(widget.content).toList();

    final widgets = <Widget>[];
    for (int i = 0; i < parts.length; i++) {
      // Plain text part
      if (parts[i].isNotEmpty) {
        widgets.add(_buildText(parts[i]));
      }
      // Code block
      if (i < blocks.length) {
        final raw        = blocks[i].group(0)!;
        final firstNL    = raw.indexOf('\n');
        final lang       = firstNL > 3 ? raw.substring(3, firstNL).trim() : 'code';
        final lastFence  = raw.lastIndexOf('```');
        final code       = firstNL >= 0 && lastFence > firstNL
            ? raw.substring(firstNL + 1, lastFence).trim()
            : '';
        final copied     = _copiedIdx == i;
        widgets.add(Container(
          margin: const EdgeInsets.symmetric(vertical: 6),
          decoration: BoxDecoration(
            color: const Color(0xFF030712),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: Colors.white.withValues(alpha: 0.1)),
          ),
          child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            // Header bar
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
              decoration: const BoxDecoration(
                color: Color(0xFF0F1726),
                borderRadius: BorderRadius.vertical(top: Radius.circular(12)),
              ),
              child: Row(children: [
                Text(lang.toUpperCase(),
                  style: const TextStyle(color: Color(0xFF6b7280),
                      fontSize: 9, fontFamily: 'monospace', letterSpacing: 0.8)),
                const Spacer(),
                GestureDetector(
                  onTap: () => _copyCode(code, i),
                  child: Row(children: [
                    Icon(copied ? Icons.check_rounded : Icons.copy_rounded,
                      size: 13, color: copied ? NexusColors.green : const Color(0xFF6b7280)),
                    const SizedBox(width: 4),
                    Text(copied ? 'Copied!' : 'Copy',
                      style: TextStyle(
                        color: copied ? NexusColors.green : const Color(0xFF6b7280),
                        fontSize: 10)),
                  ]),
                ),
              ]),
            ),
            // Code body
            SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.all(12),
              child: SelectableText(code,
                style: const TextStyle(color: Color(0xFF86efac),
                    fontFamily: 'monospace', fontSize: 11.5, height: 1.55)),
            ),
          ]),
        ));
      }
    }

    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: widgets);
  }

  /// Renders plain-text with **bold** and `inline code` support.
  Widget _buildText(String text) {
    final lines = text.split('\n');
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: lines.map((line) {
        if (line.trim().isEmpty) return const SizedBox(height: 4);
        // Split into chunks: **bold**, `inline`, plain
        final chunks = _parseInline(line);
        return Padding(
          padding: const EdgeInsets.only(bottom: 2),
          child: Text.rich(TextSpan(children: chunks),
            style: TextStyle(
              color: widget.mode == ChatMode.code
                  ? const Color(0xFF86efac).withValues(alpha: 0.9)
                  : NexusColors.textPrimary,
              fontSize: 13, height: 1.5)),
        );
      }).toList(),
    );
  }

  List<InlineSpan> _parseInline(String line) {
    final spans = <InlineSpan>[];
    // Regex: **bold** or `inline`
    final re = RegExp(r'\*\*([^*]+)\*\*|`([^`]+)`');
    int cursor = 0;
    for (final m in re.allMatches(line)) {
      if (m.start > cursor) {
        spans.add(TextSpan(text: line.substring(cursor, m.start)));
      }
      if (m.group(1) != null) {
        // **bold**
        spans.add(TextSpan(text: m.group(1),
          style: const TextStyle(fontWeight: FontWeight.w700, color: Colors.white)));
      } else if (m.group(2) != null) {
        // `inline code`
        spans.add(WidgetSpan(
          alignment: PlaceholderAlignment.middle,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.08),
              borderRadius: BorderRadius.circular(4),
              border: Border.all(color: Colors.white.withValues(alpha: 0.12)),
            ),
            child: Text(m.group(2)!,
              style: const TextStyle(fontFamily: 'monospace',
                  color: Color(0xFF86efac), fontSize: 11.5)),
          ),
        ));
      }
      cursor = m.end;
    }
    if (cursor < line.length) spans.add(TextSpan(text: line.substring(cursor)));
    return spans;
  }
}

// ── Chat bubble ────────────────────────────────────────────────────────────────

class _ChatBubble extends StatefulWidget {
  final _Message msg;
  const _ChatBubble({required this.msg});
  @override State<_ChatBubble> createState() => _ChatBubbleState();
}

class _ChatBubbleState extends State<_ChatBubble> {
  bool _copied = false;

  void _copyMessage() {
    Clipboard.setData(ClipboardData(text: widget.msg.content));
    setState(() => _copied = true);
    Future.delayed(const Duration(milliseconds: 1600), () {
      if (mounted) setState(() => _copied = false);
    });
  }

  String _formatTime(DateTime dt) {
    final h = dt.hour.toString().padLeft(2, '0');
    final m = dt.minute.toString().padLeft(2, '0');
    return '$h:$m';
  }

  @override
  Widget build(BuildContext context) {
    final msg    = widget.msg;
    final isUser = msg.role == 'user';
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 5),
      child: Column(
        crossAxisAlignment: isUser ? CrossAxisAlignment.end : CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: isUser ? MainAxisAlignment.end : MainAxisAlignment.start,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              // AI avatar
              if (!isUser) ...[
                CircleAvatar(
                  radius: 15,
                  backgroundColor: msg.mode.color.withValues(alpha: 0.12),
                  child: Text(msg.mode.emoji, style: const TextStyle(fontSize: 13)),
                ),
                const SizedBox(width: 8),
              ],
              // Bubble
              Flexible(child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                decoration: BoxDecoration(
                  gradient: isUser ? LinearGradient(
                    colors: [msg.mode.color, msg.mode.color.withValues(alpha: 0.75)],
                    begin: Alignment.topLeft, end: Alignment.bottomRight,
                  ) : null,
                  color: isUser ? null : NexusColors.surface,
                  borderRadius: BorderRadius.only(
                    topLeft:     const Radius.circular(18),
                    topRight:    const Radius.circular(18),
                    bottomLeft:  Radius.circular(isUser ? 18 : 4),
                    bottomRight: Radius.circular(isUser ? 4 : 18),
                  ),
                  border: isUser ? null : Border.all(color: NexusColors.border),
                  boxShadow: [BoxShadow(
                    color: Colors.black.withValues(alpha: 0.12),
                    blurRadius: 6, offset: const Offset(0, 2))],
                ),
                child: isUser
                    ? SelectableText(msg.content,
                        style: const TextStyle(color: Colors.white,
                            fontSize: 13, height: 1.45))
                    : _RichMessage(content: msg.content, mode: msg.mode),
              )),
              // User avatar
              if (isUser) ...[
                const SizedBox(width: 8),
                CircleAvatar(
                  radius: 15,
                  backgroundColor: NexusColors.primary.withValues(alpha: 0.15),
                  child: const Icon(Icons.person_rounded, size: 15, color: NexusColors.primary),
                ),
              ],
            ],
          ),

          // ── Meta row: provider label + timestamp + copy button ────────────
          Padding(
            padding: EdgeInsets.only(
              left: isUser ? 0 : 38,
              right: isUser ? 38 : 0,
              top: 3),
            child: Row(
              mainAxisAlignment: isUser ? MainAxisAlignment.end : MainAxisAlignment.start,
              children: [
                if (!isUser && msg.provider != null) ...[
                  Text(msg.provider!,
                    style: const TextStyle(color: NexusColors.textSecondary,
                        fontSize: 9, fontStyle: FontStyle.italic)),
                  const SizedBox(width: 6),
                ],
                Text(_formatTime(msg.ts),
                  style: const TextStyle(color: NexusColors.textSecondary, fontSize: 9)),
                // Copy button — only for assistant messages
                if (!isUser) ...[
                  const SizedBox(width: 8),
                  GestureDetector(
                    onTap: _copyMessage,
                    child: AnimatedSwitcher(
                      duration: const Duration(milliseconds: 200),
                      child: _copied
                          ? const Icon(Icons.check_rounded, size: 13, color: NexusColors.green, key: ValueKey('check'))
                          : const Icon(Icons.copy_rounded, size: 13, color: NexusColors.textSecondary, key: ValueKey('copy')),
                    ),
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ── Animated typing indicator ──────────────────────────────────────────────────

class _ThinkingBubble extends StatefulWidget {
  final ChatMode mode;
  const _ThinkingBubble({required this.mode});
  @override State<_ThinkingBubble> createState() => _ThinkingBubbleState();
}

class _ThinkingBubbleState extends State<_ThinkingBubble>
    with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(duration: const Duration(milliseconds: 900), vsync: this)
      ..repeat();
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 5),
      child: Row(crossAxisAlignment: CrossAxisAlignment.end, children: [
        CircleAvatar(
          radius: 15,
          backgroundColor: widget.mode.color.withValues(alpha: 0.12),
          child: Text(widget.mode.emoji, style: const TextStyle(fontSize: 13)),
        ),
        const SizedBox(width: 8),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          decoration: BoxDecoration(
            color: NexusColors.surface,
            borderRadius: const BorderRadius.only(
              topLeft: Radius.circular(18), topRight: Radius.circular(18),
              bottomRight: Radius.circular(18), bottomLeft: Radius.circular(4)),
            border: Border.all(color: NexusColors.border),
          ),
          child: AnimatedBuilder(
            animation: _ctrl,
            builder: (_, __) => Row(mainAxisSize: MainAxisSize.min,
              children: List.generate(3, (i) {
                // Each dot peaks at a different phase
                final phase = ((_ctrl.value * 3) - i).clamp(0.0, 1.0);
                final bounce = (phase < 0.5 ? phase * 2 : (1 - phase) * 2);
                return Transform.translate(
                  offset: Offset(0, -4 * bounce),
                  child: Container(
                    width: 7, height: 7,
                    margin: EdgeInsets.only(right: i < 2 ? 4 : 0),
                    decoration: BoxDecoration(
                      color: widget.mode.color.withValues(alpha: 0.4 + 0.6 * bounce),
                      shape: BoxShape.circle),
                  ),
                );
              }),
            ),
          ),
        ),
      ]),
    );
  }
}

// ══════════════════════════════════════════════════════════════════════════════
// Tools tab
// ══════════════════════════════════════════════════════════════════════════════

class _ToolsTab extends ConsumerWidget {
  final int userPoints;
  final String searchQuery;
  final String? activeCategory;
  final ValueChanged<String> onSearch;
  final ValueChanged<String?> onCategoryChange;
  final ValueChanged<StudioTool> onToolTap;

  const _ToolsTab({
    required this.userPoints, required this.searchQuery, required this.activeCategory,
    required this.onSearch, required this.onCategoryChange, required this.onToolTap,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final toolsAsync = ref.watch(_toolsProvider);
    return toolsAsync.when(
      loading: () => _loadingShimmer(),
      error: (_, __) => const Center(child: Text('Could not load tools',
        style: TextStyle(color: NexusColors.textSecondary))),
      data: (tools) {
        // Chat-tab tools are handled by the Chat tab — exclude from the tools grid
        const chatTabSlugs = {'nexus-chat', 'ask-nexus', 'ai-chat'};

        // Filter
        final filtered = tools.where((t) {
          if (chatTabSlugs.contains(t.slug)) return false;
          final matchSearch = searchQuery.isEmpty ||
            t.name.toLowerCase().contains(searchQuery.toLowerCase()) ||
            t.description.toLowerCase().contains(searchQuery.toLowerCase());
          final matchCat = activeCategory == null || t.category == activeCategory;
          return matchSearch && matchCat;
        }).toList();

        // Group by category
        final categories = tools.map((t) => t.category).toSet().toList();
        final grouped = <String, List<StudioTool>>{};
        for (final cat in categories) {
          final catTools = filtered.where((t) => t.category == cat).toList();
          if (catTools.isNotEmpty) grouped[cat] = catTools;
        }

        return Column(children: [
          // Search bar
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 10, 16, 0),
            child: TextField(
              onChanged: onSearch,
              style: const TextStyle(color: NexusColors.textPrimary, fontSize: 14),
              decoration: InputDecoration(
                hintText: 'Search tools…', hintStyle: const TextStyle(color: NexusColors.textSecondary),
                prefixIcon: const Icon(Icons.search, color: NexusColors.textSecondary, size: 18),
                filled: true, fillColor: NexusColors.surface,
                border: OutlineInputBorder(borderRadius: BorderRadius.circular(14),
                  borderSide: const BorderSide(color: NexusColors.border)),
                enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(14),
                  borderSide: const BorderSide(color: NexusColors.border)),
                contentPadding: const EdgeInsets.symmetric(vertical: 8),
              ),
            ),
          ),
          // Category chips
          SizedBox(
            height: 40,
            child: ListView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.fromLTRB(16, 6, 16, 0),
              children: [
                _CategoryChip(label: 'All', active: activeCategory == null,
                  color: NexusColors.gold, onTap: () => onCategoryChange(null)),
                ...categories.map((cat) => _CategoryChip(
                  label: cat.split(' ').first,
                  active: activeCategory == cat,
                  color: _catCfg(cat).color,
                  onTap: () => onCategoryChange(activeCategory == cat ? null : cat),
                )),
              ],
            ),
          ),
          // Per-generation pricing note
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: const Color(0x0CF5A623), borderRadius: BorderRadius.circular(10),
                border: Border.all(color: const Color(0x1AF5A623)),
              ),
              child: const Row(children: [
                Icon(Icons.bolt_rounded, size: 13, color: NexusColors.gold),
                SizedBox(width: 6),
                Expanded(child: Text(
                  'Per-generation pricing: points deducted once per Generate. Failed = auto-refunded.',
                  style: TextStyle(color: NexusColors.textSecondary, fontSize: 10))),
              ]),
            ),
          ),
          // Tool list
          Expanded(
            child: tools.isEmpty
                ? const Center(child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                    Text('✨', style: TextStyle(fontSize: 48)),
                    SizedBox(height: 12),
                    Text('No tools available yet',
                      style: TextStyle(color: NexusColors.textPrimary, fontSize: 16,
                        fontWeight: FontWeight.w600)),
                  ]))
                : grouped.isEmpty
                ? Center(child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                    const Icon(Icons.search_off_rounded, size: 48, color: NexusColors.textSecondary),
                    const SizedBox(height: 12),
                    const Text('No tools match your search',
                      style: TextStyle(color: NexusColors.textSecondary)),
                    TextButton(onPressed: () => onSearch(''),
                      child: const Text('Clear search', style: TextStyle(color: NexusColors.gold))),
                  ]))
                : ListView(
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 100),
                    children: grouped.entries.expand((entry) => [
                      Padding(
                        padding: const EdgeInsets.only(top: 12, bottom: 6),
                        child: Row(children: [
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                            decoration: BoxDecoration(
                              color: _catCfg(entry.key).color.withValues(alpha: 0.12),
                              borderRadius: BorderRadius.circular(20),
                              border: Border.all(color: _catCfg(entry.key).color.withValues(alpha: 0.3)),
                            ),
                            child: Row(mainAxisSize: MainAxisSize.min, children: [
                              Icon(_catCfg(entry.key).icon, size: 12,
                                color: _catCfg(entry.key).color),
                              const SizedBox(width: 5),
                              Text(entry.key,
                                style: TextStyle(color: _catCfg(entry.key).color,
                                  fontSize: 10, fontWeight: FontWeight.w700)),
                            ]),
                          ),
                          const SizedBox(width: 6),
                          Text('${entry.value.length} tool${entry.value.length != 1 ? 's' : ''}',
                            style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
                        ]),
                      ),
                      ...entry.value.map((tool) => Padding(
                        padding: const EdgeInsets.only(bottom: 6),
                        child: _ToolCard(tool: tool, userPoints: userPoints, onTap: () => onToolTap(tool)),
                      )),
                    ]).toList(),
                  ),
          ),
        ]);
      },
    );
  }

  Widget _loadingShimmer() => ListView.builder(
    padding: const EdgeInsets.all(16),
    itemCount: 8,
    itemBuilder: (_, __) => Container(
      height: 72, margin: const EdgeInsets.only(bottom: 6),
      decoration: BoxDecoration(
        color: NexusColors.surface, borderRadius: BorderRadius.circular(14),
        border: Border.all(color: NexusColors.border)),
    ),
  );
}

class _CategoryChip extends StatelessWidget {
  final String label;
  final bool active;
  final Color color;
  final VoidCallback onTap;
  const _CategoryChip({required this.label, required this.active,
    required this.color, required this.onTap});

  @override
  Widget build(BuildContext context) => GestureDetector(
    onTap: onTap,
    child: AnimatedContainer(
      duration: const Duration(milliseconds: 180),
      margin: const EdgeInsets.only(right: 6),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      decoration: BoxDecoration(
        color: active ? color.withValues(alpha: 0.15) : Colors.transparent,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: active ? color.withValues(alpha: 0.4) : NexusColors.border),
      ),
      child: Text(label,
        style: TextStyle(color: active ? color : NexusColors.textSecondary,
          fontSize: 11, fontWeight: FontWeight.w600)),
    ),
  );
}

class _ToolCard extends StatelessWidget {
  final StudioTool tool;
  final int userPoints;
  final VoidCallback onTap;
  const _ToolCard({required this.tool, required this.userPoints, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final cfg = _catCfg(tool.category);
    final meta = _toolMeta[tool.slug];
    final entryLocked = !tool.isFree && tool.entryPointCost > 0 && userPoints < tool.entryPointCost;

    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: NexusColors.surface, borderRadius: BorderRadius.circular(14),
          border: Border.all(color: NexusColors.border),
        ),
        child: Stack(children: [
          Row(children: [
            Container(
              width: 40, height: 40,
              decoration: BoxDecoration(
                color: cfg.color.withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(11),
                border: Border.all(color: cfg.color.withValues(alpha: 0.25)),
              ),
              child: Icon(cfg.icon, size: 18, color: cfg.color),
            ),
            const SizedBox(width: 12),
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Flexible(child: Text(tool.name,
                  style: const TextStyle(color: NexusColors.textPrimary,
                    fontWeight: FontWeight.w600, fontSize: 13),
                  maxLines: 1, overflow: TextOverflow.ellipsis)),
                if (tool.isNew) ...[const SizedBox(width: 4), _Badge('NEW', const Color(0xFF8B5CF6))],
                if (tool.isFree) ...[const SizedBox(width: 4), _Badge('FREE', NexusColors.green)],
                if (tool.isChatTool) ...[const SizedBox(width: 4), _Badge('💬 Chat', const Color(0xFF0891b2))],
              ]),
              const SizedBox(height: 2),
              Text(tool.description,
                style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11),
                maxLines: 1, overflow: TextOverflow.ellipsis),
              const SizedBox(height: 4),
              Row(children: [
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: tool.isFree ? NexusColors.green.withValues(alpha: 0.12)
                        : NexusColors.gold.withValues(alpha: 0.12),
                    borderRadius: BorderRadius.circular(6)),
                  child: Text(tool.isFree ? 'Free' : '${tool.pointCost} pts/gen',
                    style: TextStyle(
                      color: tool.isFree ? NexusColors.green : NexusColors.gold,
                      fontSize: 9, fontWeight: FontWeight.w700)),
                ),
                if (meta != null) ...[
                  const SizedBox(width: 6),
                  Icon(Icons.schedule_rounded, size: 9, color: NexusColors.textSecondary),
                  const SizedBox(width: 2),
                  Text(meta.$1, style: const TextStyle(
                    color: NexusColors.textSecondary, fontSize: 9)),
                ],
              ]),
            ])),
            Icon(Icons.chevron_right_rounded, color: NexusColors.textSecondary.withValues(alpha: 0.5), size: 18),
          ]),
          if (entryLocked)
            Positioned.fill(child: Container(
              decoration: BoxDecoration(
                color: Colors.black.withValues(alpha: 0.55),
                borderRadius: BorderRadius.circular(14),
              ),
              child: Align(
                alignment: Alignment.centerRight,
                child: Padding(
                  padding: const EdgeInsets.only(right: 10),
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                    decoration: BoxDecoration(
                      color: const Color(0x33f59e0b),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: const Color(0x66f59e0b)),
                    ),
                    child: Row(mainAxisSize: MainAxisSize.min, children: [
                      const Icon(Icons.lock_rounded, size: 10, color: Color(0xFFfbbf24)),
                      const SizedBox(width: 3),
                      Text('${tool.entryPointCost} pts to unlock',
                        style: const TextStyle(color: Color(0xFFfbbf24),
                          fontSize: 9, fontWeight: FontWeight.w700)),
                    ]),
                  ),
                ),
              ),
            )),
        ]),
      ),
    );
  }
}

class _Badge extends StatelessWidget {
  final String label;
  final Color color;
  const _Badge(this.label, this.color);
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
    decoration: BoxDecoration(
      color: color.withValues(alpha: 0.15),
      borderRadius: BorderRadius.circular(4),
      border: Border.all(color: color.withValues(alpha: 0.3)),
    ),
    child: Text(label, style: TextStyle(color: color, fontSize: 8, fontWeight: FontWeight.w800)),
  );
}

// ══════════════════════════════════════════════════════════════════════════════
// Gallery tab
// ══════════════════════════════════════════════════════════════════════════════

class _GalleryTab extends StatelessWidget {
  final _GalleryState galleryState;
  final Future<void> Function() onRefresh;
  final ValueChanged<Generation> onRegenerate;
  final ValueChanged<String> onDelete;
  const _GalleryTab({
    required this.galleryState, required this.onRefresh,
    required this.onRegenerate, required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    // Loading skeleton
    if (galleryState.loading) {
      return ListView.builder(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 100),
        itemCount: 4,
        itemBuilder: (_, __) => Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: NexusShimmer(width: double.infinity, height: 130, radius: NexusRadius.md),
        ),
      );
    }

    final gallery = galleryState.items;

    // Empty state
    if (gallery.isEmpty) { return RefreshIndicator(
      onRefresh: onRefresh,
      color: NexusColors.primary,
      child: ListView(padding: const EdgeInsets.all(32), children: [
        const SizedBox(height: 60),
        const Icon(Icons.photo_library_outlined, size: 64, color: NexusColors.textSecondary),
        const SizedBox(height: 16),
        const Text('No generations yet',
          style: TextStyle(color: NexusColors.textPrimary, fontSize: 18, fontWeight: FontWeight.w600),
          textAlign: TextAlign.center),
        const SizedBox(height: 8),
        const Text('Use a tool from the Tools tab to create something amazing',
          style: TextStyle(color: NexusColors.textSecondary, fontSize: 13),
          textAlign: TextAlign.center),
      ]),
    ); }

    return Column(children: [
      Padding(
        padding: const EdgeInsets.fromLTRB(16, 10, 16, 4),
        child: Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
          Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text('${gallery.length} generation${gallery.length != 1 ? 's' : ''}',
              style: const TextStyle(color: NexusColors.textPrimary, fontWeight: FontWeight.w700, fontSize: 13)),
            const Text('Failed items are auto-refunded',
              style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
          ]),
          IconButton(onPressed: onRefresh, icon: const Icon(Icons.refresh_rounded),
            color: NexusColors.textSecondary, iconSize: 18),
        ]),
      ),
      Expanded(
        child: RefreshIndicator(
          onRefresh: onRefresh,
          color: NexusColors.primary,
          child: ListView.builder(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 100),
            itemCount: gallery.length,
            itemBuilder: (_, i) => Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: _GenerationCard(
                gen: gallery[i],
                onRegenerate: onRegenerate,
                onDelete: () => _confirmDelete(context, gallery[i].id),
              ),
            ),
          ),
        ),
      ),
    ]);
  }

  void _confirmDelete(BuildContext context, String id) {
    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        backgroundColor: NexusColors.surface,
        title: const Text('Delete generation?',
            style: TextStyle(color: NexusColors.textPrimary, fontSize: 16, fontWeight: FontWeight.w700)),
        content: const Text('This cannot be undone.',
            style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context),
              child: const Text('Cancel')),
          TextButton(
            onPressed: () { Navigator.pop(context); onDelete(id); },
            child: const Text('Delete', style: TextStyle(color: NexusColors.red)),
          ),
        ],
      ),
    );
  }
}

class _GenerationCard extends StatelessWidget {
  final Generation gen;
  final ValueChanged<Generation> onRegenerate;
  final VoidCallback onDelete;
  const _GenerationCard({required this.gen, required this.onRegenerate, required this.onDelete});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: NexusColors.surface, borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: gen.status == 'failed'
            ? const Color(0x33ef4444) : NexusColors.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(14, 12, 14, 0),
          child: Row(children: [
            Expanded(child: Text(gen.toolName,
              style: const TextStyle(color: NexusColors.textPrimary,
                fontWeight: FontWeight.w600, fontSize: 13),
              maxLines: 1, overflow: TextOverflow.ellipsis)),
            Text(gen.timeAgo, style: const TextStyle(
              color: NexusColors.textSecondary, fontSize: 10)),
            const SizedBox(width: 6),
            _StatusPill(status: gen.status),
            const SizedBox(width: 4),
            GestureDetector(
              onTap: onDelete,
              child: const Icon(Icons.delete_outline_rounded,
                  size: 16, color: NexusColors.textMuted),
            ),
          ]),
        ),

        // Prompt preview
        if (gen.displayPrompt.isNotEmpty)
          Padding(
            padding: const EdgeInsets.fromLTRB(14, 4, 14, 0),
            child: Text('"${gen.displayPrompt}"',
              style: const TextStyle(color: NexusColors.textSecondary,
                fontSize: 11, fontStyle: FontStyle.italic),
              maxLines: 2, overflow: TextOverflow.ellipsis),
          ),

        const SizedBox(height: 10),

        // Content by type
        if (gen.status == 'processing' || gen.status == 'pending')
          _ProcessingView(gen: gen)
        else if (gen.status == 'completed')
          _CompletedView(gen: gen, onRegenerate: onRegenerate)
        else if (gen.status == 'failed')
          _FailedView(gen: gen),

        const SizedBox(height: 12),

        // Footer
        if (gen.status == 'completed')
          Padding(
            padding: const EdgeInsets.fromLTRB(14, 0, 14, 0),
            child: Row(children: [
              const Text('Generated by Nexus AI',
                style: TextStyle(color: NexusColors.textSecondary, fontSize: 9)),
              const Spacer(),
              if (gen.pointCost != null)
                Text(gen.pointCost == 0 ? 'Free generation' : '${gen.pointCost} pts',
                  style: const TextStyle(color: NexusColors.textSecondary, fontSize: 9)),
            ]),
          ),
      ]),
    );
  }
}

class _StatusPill extends StatelessWidget {
  final String status;
  const _StatusPill({required this.status});
  @override
  Widget build(BuildContext context) {
    final (label, color, icon) = switch (status) {
      'pending'    => ('Queued',     const Color(0xFFfbbf24), Icons.schedule_rounded),
      'processing' => ('Generating', NexusColors.primary,     Icons.sync_rounded),
      'completed'  => ('Done',       NexusColors.green,       Icons.check_circle_rounded),
      'failed'     => ('Failed',     const Color(0xFFef4444), Icons.error_outline_rounded),
      _            => ('Unknown',    NexusColors.textSecondary, Icons.help_outline_rounded),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: color.withValues(alpha: 0.3)),
      ),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Icon(icon, size: 10, color: color),
        const SizedBox(width: 3),
        Text(label, style: TextStyle(color: color, fontSize: 9, fontWeight: FontWeight.w700)),
      ]),
    );
  }
}

class _ProcessingView extends StatelessWidget {
  final Generation gen;
  const _ProcessingView({required this.gen});
  @override
  Widget build(BuildContext context) {
    final meta = _toolMeta[gen.toolSlug];
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        ClipRRect(
          borderRadius: BorderRadius.circular(4),
          child: const LinearProgressIndicator(
            minHeight: 3,
            backgroundColor: Color(0x1AF5A623),
            valueColor: AlwaysStoppedAnimation(NexusColors.gold),
          ),
        ),
        if (meta != null) ...[
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: const Color(0x0CF5A623), borderRadius: BorderRadius.circular(10),
              border: Border.all(color: const Color(0x1AF5A623))),
            child: Row(children: [
              const Text('💡', style: TextStyle(fontSize: 13)),
              const SizedBox(width: 6),
              Expanded(child: Text('Did you know? ${meta.$2}',
                style: const TextStyle(color: Color(0xFFfbbf24), fontSize: 11))),
              Text('~${meta.$1}',
                style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
            ]),
          ),
        ],
      ]),
    );
  }
}

class _CompletedView extends StatelessWidget {
  final Generation gen;
  final ValueChanged<Generation> onRegenerate;
  const _CompletedView({required this.gen, required this.onRegenerate});

  @override
  Widget build(BuildContext context) {
    if (gen.outputUrl != null) {
      if (gen.isImage && !gen.isVideo) return _ImageOutput(gen: gen, onRegenerate: onRegenerate);
      if (gen.isVideo) return _VideoOutput(gen: gen, onRegenerate: onRegenerate);
      if (gen.isAudio && !gen.isVideo) return _AudioOutput(gen: gen, onRegenerate: onRegenerate);
      return _UrlOutput(gen: gen);
    }
    if (gen.outputText != null) return _TextOutput(gen: gen, onRegenerate: onRegenerate);
    return const SizedBox.shrink();
  }
}

class _ImageOutput extends StatelessWidget {
  final Generation gen;
  final ValueChanged<Generation> onRegenerate;
  const _ImageOutput({required this.gen, required this.onRegenerate});
  @override
  Widget build(BuildContext context) => Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
    ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: Image.network(gen.outputUrl!,
        width: double.infinity, fit: BoxFit.cover,
        loadingBuilder: (_, child, progress) => progress == null ? child
            : Container(height: 200, color: NexusColors.background,
                child: const Center(child: CircularProgressIndicator(color: NexusColors.primary))),
        errorBuilder: (_, __, ___) => Container(height: 120, color: NexusColors.background,
          child: const Center(child: Icon(Icons.broken_image_rounded,
            color: NexusColors.textSecondary, size: 48))),
      ),
    ),
    const SizedBox(height: 8),
    Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14),
      child: Row(children: [
        _ActionBtn('Download', Icons.download_rounded,
          () => launchUrl(Uri.parse(gen.outputUrl!))),
        const SizedBox(width: 8),
        _ActionBtn('Regenerate', Icons.refresh_rounded, () => onRegenerate(gen)),
      ]),
    ),
  ]);
}

class _AudioOutput extends StatelessWidget {
  final Generation gen;
  final ValueChanged<Generation> onRegenerate;
  const _AudioOutput({required this.gen, required this.onRegenerate});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 14),
    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: NexusColors.green.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: NexusColors.green.withValues(alpha: 0.2)),
        ),
        child: Row(children: [
          const Icon(Icons.audio_file_rounded, color: NexusColors.green, size: 28),
          const SizedBox(width: 12),
          Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(gen.toolName, style: const TextStyle(color: NexusColors.textPrimary,
              fontWeight: FontWeight.w600, fontSize: 13)),
            const Text('Audio file ready',
              style: TextStyle(color: NexusColors.green, fontSize: 11)),
          ])),
        ]),
      ),
      const SizedBox(height: 8),
      Row(children: [
        _ActionBtn('Download MP3', Icons.download_rounded,
          () => launchUrl(Uri.parse(gen.outputUrl!))),
        const SizedBox(width: 8),
        _ActionBtn('Regenerate', Icons.refresh_rounded, () => onRegenerate(gen)),
      ]),
    ]),
  );
}

class _VideoOutput extends StatelessWidget {
  final Generation gen;
  final ValueChanged<Generation> onRegenerate;
  const _VideoOutput({required this.gen, required this.onRegenerate});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 14),
    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Container(
        height: 140,
        decoration: BoxDecoration(
          color: const Color(0x1Aef4444),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: const Color(0x33ef4444)),
        ),
        child: const Center(child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
          Icon(Icons.play_circle_filled_rounded, size: 56, color: Color(0xFFef4444)),
          SizedBox(height: 6),
          Text('Video ready', style: TextStyle(color: Color(0xFFef4444), fontSize: 12)),
        ])),
      ),
      const SizedBox(height: 8),
      Row(children: [
        _ActionBtn('Download Video', Icons.download_rounded,
          () => launchUrl(Uri.parse(gen.outputUrl!), mode: LaunchMode.externalApplication)),
        const SizedBox(width: 8),
        _ActionBtn('Regenerate', Icons.refresh_rounded, () => onRegenerate(gen)),
      ]),
    ]),
  );
}

class _UrlOutput extends StatelessWidget {
  final Generation gen;
  const _UrlOutput({required this.gen});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 14),
    child: GestureDetector(
      onTap: () => launchUrl(Uri.parse(gen.outputUrl!), mode: LaunchMode.externalApplication),
      child: Row(children: [
        const Icon(Icons.open_in_new_rounded, size: 14, color: NexusColors.primary),
        const SizedBox(width: 6),
        Text('View result', style: const TextStyle(color: NexusColors.primary, fontSize: 13)),
      ]),
    ),
  );
}

class _TextOutput extends StatelessWidget {
  final Generation gen;
  final ValueChanged<Generation> onRegenerate;
  const _TextOutput({required this.gen, required this.onRegenerate});

  @override
  Widget build(BuildContext context) {
    final text = gen.outputText!;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        if (gen.isCode) ...[
          Container(
            decoration: BoxDecoration(
              color: const Color(0xFF0a0f0a),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: const Color(0x33ffffff)),
            ),
            child: Column(children: [
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                child: Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                  const Text('Code output', style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
                  GestureDetector(
                    onTap: () { Clipboard.setData(ClipboardData(text: text));
                      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
                        content: Text('Code copied'), behavior: SnackBarBehavior.floating));
                    },
                    child: const Row(children: [
                      Icon(Icons.copy_rounded, size: 11, color: NexusColors.textSecondary),
                      SizedBox(width: 3),
                      Text('Copy', style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
                    ]),
                  ),
                ]),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                child: Text(text, style: const TextStyle(
                  color: Color(0xFF86efac), fontSize: 11,
                  fontFamily: 'Courier', height: 1.5)),
              ),
            ]),
          ),
        ] else ...[
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: NexusColors.background, borderRadius: BorderRadius.circular(10),
              border: Border.all(color: NexusColors.border),
            ),
            child: SelectableText(text, style: const TextStyle(
              color: NexusColors.textPrimary, fontSize: 12, height: 1.5)),
          ),
          const SizedBox(height: 8),
          Row(children: [
            _ActionBtn('Copy', Icons.copy_rounded, () {
              Clipboard.setData(ClipboardData(text: text));
              ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
                content: Text('Copied'), behavior: SnackBarBehavior.floating));
            }),
            const SizedBox(width: 8),
            _ActionBtn('Regenerate', Icons.refresh_rounded, () => onRegenerate(gen)),
          ]),
        ],
      ]),
    );
  }
}

class _ActionBtn extends StatelessWidget {
  final String label;
  final IconData icon;
  final VoidCallback onTap;
  const _ActionBtn(this.label, this.icon, this.onTap);
  @override
  Widget build(BuildContext context) => GestureDetector(
    onTap: onTap,
    child: Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: NexusColors.background, borderRadius: BorderRadius.circular(8),
        border: Border.all(color: NexusColors.border)),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Icon(icon, size: 12, color: NexusColors.textSecondary),
        const SizedBox(width: 5),
        Text(label, style: const TextStyle(color: NexusColors.textSecondary,
          fontSize: 11, fontWeight: FontWeight.w500)),
      ]),
    ),
  );
}

class _FailedView extends StatelessWidget {
  final Generation gen;
  const _FailedView({required this.gen});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 14),
    child: Column(children: [
      Container(
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: const Color(0x1Aef4444), borderRadius: BorderRadius.circular(10),
          border: Border.all(color: const Color(0x33ef4444))),
        child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Icon(Icons.error_outline_rounded, size: 14, color: Color(0xFFef4444)),
          const SizedBox(width: 8),
          Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Generation failed',
              style: TextStyle(color: Color(0xFFef4444),
                fontWeight: FontWeight.w600, fontSize: 12)),
            if (gen.errorMessage != null)
              Text(gen.errorMessage!, style: const TextStyle(
                color: Color(0xAAef4444), fontSize: 11)),
          ])),
        ]),
      ),
      if ((gen.pointCost ?? 0) > 0) ...[
        const SizedBox(height: 6),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: BoxDecoration(
            color: NexusColors.green.withValues(alpha: 0.08),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: NexusColors.green.withValues(alpha: 0.2))),
          child: Row(children: [
            const Icon(Icons.check_circle_rounded, size: 12, color: NexusColors.green),
            const SizedBox(width: 6),
            Text('${gen.pointCost} pts refunded automatically',
              style: const TextStyle(color: NexusColors.green,
                fontSize: 11, fontWeight: FontWeight.w600)),
          ]),
        ),
      ],
    ]),
  );
}

// ══════════════════════════════════════════════════════════════════════════════
// Tool drawer — bottom sheet with confirm modal
// ══════════════════════════════════════════════════════════════════════════════

class _ToolDrawer extends ConsumerStatefulWidget {
  final StudioTool tool;
  final int userPoints;
  final VoidCallback onClose, onGenerated;
  const _ToolDrawer({required this.tool, required this.userPoints,
    required this.onClose, required this.onGenerated});
  @override ConsumerState<_ToolDrawer> createState() => _ToolDrawerState();
}

class _ToolDrawerState extends ConsumerState<_ToolDrawer> {
  final _promptCtrl = TextEditingController();
  final _urlCtrl    = TextEditingController();
  bool _generating  = false;
  bool _showConfirm = false;

  @override void dispose() { _promptCtrl.dispose(); _urlCtrl.dispose(); super.dispose(); }

  bool get _isFree     => widget.tool.isFree || widget.tool.pointCost == 0;
  bool get _canAfford  => _isFree || widget.userPoints >= widget.tool.pointCost;
  bool get _entryLocked => !widget.tool.isFree &&
      widget.tool.entryPointCost > 0 && widget.userPoints < widget.tool.entryPointCost;

  @override
  Widget build(BuildContext context) {
    final cfg = _catCfg(widget.tool.category);
    final meta = _toolMeta[widget.tool.slug];
    final after = widget.userPoints - widget.tool.pointCost;

    return Stack(children: [
      // Backdrop
      GestureDetector(
        onTap: widget.onClose,
        child: Container(color: Colors.black54),
      ),
      // Sheet
      Align(
        alignment: Alignment.bottomCenter,
        child: Material(
          color: NexusColors.surface,
          borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
          child: Container(
            constraints: BoxConstraints(maxHeight: MediaQuery.of(context).size.height * 0.85),
            decoration: const BoxDecoration(
              color: NexusColors.surface,
              borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
            ),
            child: Column(mainAxisSize: MainAxisSize.min, children: [
              // Handle
              Container(
                width: 40, height: 4, margin: const EdgeInsets.only(top: 12, bottom: 4),
                decoration: BoxDecoration(
                  color: NexusColors.border, borderRadius: BorderRadius.circular(2)),
              ),
              // Accent stripe
              Container(height: 3,
                decoration: BoxDecoration(
                  gradient: LinearGradient(colors: [Colors.transparent, cfg.color, Colors.transparent]),
                  borderRadius: const BorderRadius.horizontal(left: Radius.circular(2), right: Radius.circular(2)))),
              // Body
              Flexible(child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(20, 16, 20, 32),
                child: _entryLocked ? _buildEntryGate(cfg) : _buildBody(cfg, meta, after),
              )),
            ]),
          ),
        ),
      ),
      // Confirm modal on top
      if (_showConfirm) _buildConfirmModal(after),
    ]);
  }

  Widget _buildEntryGate(_CatConfig cfg) => Column(children: [
    Container(
      width: 64, height: 64,
      decoration: BoxDecoration(
        color: const Color(0x33f59e0b), borderRadius: BorderRadius.circular(18),
        border: Border.all(color: const Color(0x66f59e0b))),
      child: const Icon(Icons.lock_rounded, size: 32, color: Color(0xFFfbbf24))),
    const SizedBox(height: 16),
    Text(widget.tool.name, style: const TextStyle(color: NexusColors.textPrimary,
      fontSize: 20, fontWeight: FontWeight.w800)),
    const SizedBox(height: 6),
    const Text('Requires minimum balance to unlock',
      style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
    const SizedBox(height: 20),
    Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0x0CF5A623), borderRadius: BorderRadius.circular(14),
        border: Border.all(color: const Color(0x1AF5A623))),
      child: Column(children: [
        _PointRow('Required balance', '${widget.tool.entryPointCost} pts', const Color(0xFFfbbf24)),
        _PointRow('Your balance', '${widget.userPoints} pts', const Color(0xFFef4444)),
        const Divider(color: NexusColors.border, height: 16),
        _PointRow('You need', '${widget.tool.entryPointCost - widget.userPoints} more pts', const Color(0xFFef4444)),
      ]),
    ),
    const SizedBox(height: 20),
    Row(children: [
      Expanded(child: OutlinedButton(
        onPressed: widget.onClose,
        style: OutlinedButton.styleFrom(
          foregroundColor: NexusColors.textSecondary,
          side: const BorderSide(color: NexusColors.border),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14))),
        child: const Text('Back'))),
      const SizedBox(width: 12),
      Expanded(child: ElevatedButton(
        onPressed: () {},
        style: ElevatedButton.styleFrom(
          backgroundColor: const Color(0xFFd97706),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14))),
        child: const Text('Recharge MTN'))),
    ]),
  ]);

  Widget _buildBody(_CatConfig cfg, (String, String)? meta, int after) => Column(
    crossAxisAlignment: CrossAxisAlignment.start, children: [
    // Header
    Row(children: [
      Container(width: 44, height: 44,
        decoration: BoxDecoration(
          color: cfg.color.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(14),
          border: Border.all(color: cfg.color.withValues(alpha: 0.25))),
        child: Icon(cfg.icon, size: 22, color: cfg.color)),
      const SizedBox(width: 12),
      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Flexible(child: Text(widget.tool.name,
            style: const TextStyle(color: NexusColors.textPrimary,
              fontSize: 17, fontWeight: FontWeight.w800))),
          if (widget.tool.isNew) ...[const SizedBox(width: 6),
            _Badge('NEW', const Color(0xFF8B5CF6))],
          if (_isFree) ...[const SizedBox(width: 6), _Badge('FREE', NexusColors.green)],
        ]),
        if (meta != null)
          Text('${_outputEmoji()} Outputs 1 ${_outputNoun()} · ~${meta.$1}',
            style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
      ])),
      IconButton(onPressed: widget.onClose, icon: const Icon(Icons.close_rounded),
        color: NexusColors.textSecondary),
    ]),

    // Tip
    if (meta != null) ...[
      const SizedBox(height: 12),
      Container(
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: const Color(0x0CF5A623), borderRadius: BorderRadius.circular(10),
          border: Border.all(color: const Color(0x1AF5A623))),
        child: Row(children: [
          const Text('💡', style: TextStyle(fontSize: 13)),
          const SizedBox(width: 8),
          Expanded(child: Text('Tip: ${meta.$2}',
            style: const TextStyle(color: Color(0xFFfbbf24), fontSize: 11))),
        ]),
      ),
    ],

    const SizedBox(height: 16),

    // ── Points balance bar (always visible above template) ──
    Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: _isFree ? NexusColors.green.withValues(alpha: 0.05)
            : _canAfford ? const Color(0x0CF5A623)
            : const Color(0x1Aef4444),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: _isFree ? NexusColors.green.withValues(alpha: 0.2)
              : _canAfford ? const Color(0x1AF5A623)
              : const Color(0x33ef4444))),
      child: _isFree
          ? const Row(children: [
              Icon(Icons.check_circle_rounded, size: 13, color: NexusColors.green),
              SizedBox(width: 8),
              Text('Free generation — no points used',
                style: TextStyle(color: NexusColors.green, fontSize: 12, fontWeight: FontWeight.w600)),
            ])
          : Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
              _PointRow('Cost', '−${widget.tool.pointCost} pts', Colors.white70),
              _PointRow('Balance', '${widget.userPoints} pts', NexusColors.gold),
              _PointRow('After',
                _canAfford
                    ? '${widget.userPoints - widget.tool.pointCost} pts'
                    : 'Need ${widget.tool.pointCost - widget.userPoints} more',
                _canAfford ? NexusColors.gold : const Color(0xFFef4444)),
            ]),
    ),

    const SizedBox(height: 20),

    // ── Purpose-built template (has its own Generate button) ──
    TemplateRegistry.build(
      tool: widget.tool.toJson(),
      onSubmit: (payload) => _doGenerateWithPayload(payload),
      isLoading: _generating,
      userPoints: widget.userPoints,
    ),
  ]);

  /// Called by TemplateRegistry templates when the user taps Generate.
  void _doGenerateWithPayload(GeneratePayload payload) {
    if (_generating) return;
    _doGenerateRaw(payload.toJson());
  }

  Widget _buildConfirmModal(int after) => GestureDetector(
    onTap: () => setState(() => _showConfirm = false),
    child: Container(
      color: Colors.black.withValues(alpha: 0.7),
      child: Center(child: GestureDetector(
        onTap: () {},
        child: Container(
          margin: const EdgeInsets.all(24),
          padding: const EdgeInsets.all(24),
          decoration: BoxDecoration(
            color: NexusColors.surface, borderRadius: BorderRadius.circular(20),
            border: Border.all(color: NexusColors.border)),
          child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(widget.tool.name, style: const TextStyle(color: NexusColors.textPrimary,
              fontSize: 18, fontWeight: FontWeight.w800)),
            const SizedBox(height: 6),
            Text('"${_promptCtrl.text.trim()}"',
              style: const TextStyle(color: NexusColors.textSecondary,
                fontSize: 12, fontStyle: FontStyle.italic),
              maxLines: 3, overflow: TextOverflow.ellipsis),
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: const Color(0x0CF5A623), borderRadius: BorderRadius.circular(12),
                border: Border.all(color: const Color(0x1AF5A623))),
              child: Column(children: [
                _PointRow('Generation cost', '−${widget.tool.pointCost} pts', Colors.white),
                const SizedBox(height: 4),
                _PointRow('Your balance', '${widget.userPoints} pts', NexusColors.gold),
                const Divider(color: NexusColors.border, height: 12),
                _PointRow('After', '$after pts', NexusColors.gold),
              ]),
            ),
            const SizedBox(height: 6),
            const Text('⚡ Points deducted once when generation starts. Failed = auto-refunded.',
              style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
            const SizedBox(height: 16),
            Row(children: [
              Expanded(child: OutlinedButton(
                onPressed: () => setState(() => _showConfirm = false),
                style: OutlinedButton.styleFrom(
                  foregroundColor: NexusColors.textSecondary,
                  side: const BorderSide(color: NexusColors.border),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12))),
                child: const Text('Cancel'))),
              const SizedBox(width: 10),
              Expanded(child: ElevatedButton(
                onPressed: _generating ? null : _doGenerate,
                style: ElevatedButton.styleFrom(
                  backgroundColor: NexusColors.gold, foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12))),
                child: _generating
                    ? const SizedBox(width: 18, height: 18,
                        child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
                    : Text(_isFree ? 'Generate Free' : 'Confirm & Generate'),
              )),
            ]),
          ]),
        ),
      )),
    ),
  );

  Future<void> _doGenerateRaw(Map<String, dynamic> payload) async {
    setState(() { _generating = true; _showConfirm = false; });
    try {
      await ref.read(studioApiProvider)
          .startGeneration(widget.tool.slug, payload);
      widget.onGenerated();
    } catch (e) {
      setState(() => _generating = false);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text('Failed: ${e.toString()}'),
          behavior: SnackBarBehavior.floating,
          backgroundColor: const Color(0xFFef4444)));
      }
    }
  }

  Future<void> _doGenerate() async {
    // Legacy path (used by confirm modal). Build minimal payload from old form.
    final prompt = _promptCtrl.text.trim();
    await _doGenerateRaw({'prompt': prompt});
  }

  String _outputEmoji() {
    final slug = widget.tool.slug;
    if (_videoSlugs.contains(slug)) return '🎬';
    if (_audioSlugs.contains(slug)) return '🎵';
    if (_imageSlugs.contains(slug)) return '🖼️';
    if (_codeSlugs.contains(slug))  return '💻';
    return '📄';
  }

  String _outputNoun() {
    final slug = widget.tool.slug;
    if (_videoSlugs.contains(slug)) return 'video';
    if (_audioSlugs.contains(slug)) return 'audio';
    if (_imageSlugs.contains(slug)) return 'image';
    if (_codeSlugs.contains(slug))  return 'code block';
    return 'text';
  }
}

// Helper for point rows in confirm/entry-gate boxes
class _PointRow extends StatelessWidget {
  final String label, value;
  final Color valueColor;
  const _PointRow(this.label, this.value, this.valueColor);
  @override
  Widget build(BuildContext context) => Row(
    mainAxisAlignment: MainAxisAlignment.spaceBetween,
    children: [
      Text(label, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
      Text(value, style: TextStyle(color: valueColor, fontSize: 12, fontWeight: FontWeight.w700)),
    ],
  );
}


