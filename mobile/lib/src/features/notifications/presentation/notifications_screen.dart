import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:gap/gap.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ── Providers ──────────────────────────────────────────────────────────────────

// ── Paginated state ───────────────────────────────────────────────────────────

class _NotifsState {
  final List<Map> items;
  final int unreadCount;
  final String? cursor;
  final bool loadingMore;
  const _NotifsState({
    this.items = const [], this.unreadCount = 0,
    this.cursor, this.loadingMore = false,
  });
  _NotifsState copyWith({List<Map>? items, int? unreadCount, String? cursor,
    bool clearCursor = false, bool? loadingMore}) =>
      _NotifsState(
        items: items ?? this.items,
        unreadCount: unreadCount ?? this.unreadCount,
        cursor: clearCursor ? null : (cursor ?? this.cursor),
        loadingMore: loadingMore ?? this.loadingMore,
      );
}

final _notifsProvider = StateNotifierProvider.autoDispose<_NotifsNotifier, _NotifsState>(
  (ref) => _NotifsNotifier(ref.read(notificationsApiProvider)),
);

class _NotifsNotifier extends StateNotifier<_NotifsState> {
  final NotificationsApi _api;
  _NotifsNotifier(this._api) : super(const _NotifsState()) { _load(); }

  Future<void> _load({String? afterCursor}) async {
    try {
      final Map res = await _api.list() as Map;
      final items = ((res['notifications'] ?? []) as List).cast<Map>();
      state = _NotifsState(
        items: items,
        unreadCount: res['unread_count'] as int? ?? 0,
        cursor: res['cursor']?.toString(),
      );
    } catch (_) {}
  }

  Future<void> refresh() => _load();

  Future<void> loadMore() async {
    if (state.cursor == null || state.loadingMore) return;
    state = state.copyWith(loadingMore: true);
    try {
      final Map res = await _api.list() as Map; // API will support cursor later
      final more = ((res['notifications'] ?? []) as List).cast<Map>();
      state = state.copyWith(
        items: [...state.items, ...more],
        cursor: more.isEmpty ? null : res['cursor']?.toString(),
        clearCursor: more.isEmpty,
        loadingMore: false,
      );
    } catch (_) {
      state = state.copyWith(loadingMore: false);
    }
  }

  Future<void> markRead(String id) async {
    try {
      await _api.markRead(id);
      state = state.copyWith(
        items: state.items.map((n) =>
          n['id'] == id ? {...n, 'is_read': true} : n).toList(),
        unreadCount: (state.unreadCount - 1).clamp(0, 9999),
      );
    } catch (_) {}
  }

  Future<void> markAllRead() async {
    try {
      await _api.markAllRead();
      state = state.copyWith(
        items: state.items.map((n) => {...n, 'is_read': true}).toList(),
        unreadCount: 0,
      );
    } catch (_) {}
  }
}

// ── Type → icon/colour mapping (no subscription types) ────────────────────────

const _typeConfig = {
  'spin_win':     (emoji: '🏆', color: Color(0xFFf59e0b)),
  'draw_result':  (emoji: '🎁', color: Color(0xFFec4899)),
  'streak_warn':  (emoji: '🔥', color: Color(0xFFf97316)),
  'studio_ready': (emoji: '✨', color: Color(0xFF8B5CF6)),
  'wars_result':  (emoji: '🌍', color: Color(0xFF10b981)),
  'bonus_pulse':  (emoji: '⚡', color: Color(0xFF5F72F9)),
  'marketing':    (emoji: '📢', color: Color(0xFF06b6d4)),
  'referral':     (emoji: '👥', color: Color(0xFF22c55e)),
};

({String emoji, Color color}) _cfg(String? type) =>
  _typeConfig[type] ?? (emoji: '🔔', color: NexusColors.textSecondary);

// ── Notifications screen ───────────────────────────────────────────────────────

class NotificationsScreen extends ConsumerWidget {
  const NotificationsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state    = ref.watch(_notifsProvider);
    final notifier = ref.read(_notifsProvider.notifier);
    final items    = state.items;
    final unread   = state.unreadCount;

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        title: Row(children: [
          const Text('Notifications 🔔'),
          const Gap(8),
          if (unread > 0)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
              decoration: BoxDecoration(
                color: NexusColors.primary,
                borderRadius: BorderRadius.circular(20),
              ),
              child: Text('$unread',
                style: const TextStyle(color: Colors.white, fontSize: 11,
                  fontWeight: FontWeight.w800)),
            ).animate().scale(),
        ]),
        actions: [
          if (unread > 0)
            TextButton(
              onPressed: () => notifier.markAllRead(),
              style: TextButton.styleFrom(foregroundColor: NexusColors.primary),
              child: const Text('Mark all read', style: TextStyle(fontSize: 12)),
            ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () => notifier.refresh(),
        color: NexusColors.primary,
        child: items.isEmpty
            ? ListView(children: [const Gap(80), const _EmptyState()])
            : _NotifList(items: items, state: state, notifier: notifier),
      ),
    );
  }
}

class _NotifList extends StatelessWidget {
  final List<Map> items;
  final _NotifsState state;
  final _NotifsNotifier notifier;
  const _NotifList({required this.items, required this.state, required this.notifier});

  @override
  Widget build(BuildContext context) {
    final now   = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final todayItems   = items.where((n) {
      final d = DateTime.tryParse(n['created_at']?.toString() ?? '') ?? DateTime(2000);
      return d.isAfter(today);
    }).toList();
    final earlierItems = items.where((n) {
      final d = DateTime.tryParse(n['created_at']?.toString() ?? '') ?? DateTime(2000);
      return !d.isAfter(today);
    }).toList();

    return ListView(children: [
      if (todayItems.isNotEmpty) ..[
        const _GroupHeader('Today'),
        ...todayItems.asMap().entries.map((e) =>
          _NotifTile(notif: e.value, notifier: notifier)
              .animate(delay: (e.key * 30).ms).fadeIn().slideX(begin: -0.03, end: 0)),
      ],
      if (earlierItems.isNotEmpty) ..[
        const _GroupHeader('Earlier'),
        ...earlierItems.asMap().entries.map((e) =>
          _NotifTile(notif: e.value, notifier: notifier)
              .animate(delay: (e.key * 20).ms).fadeIn()),
      ],
      // Load More
      if (state.cursor != null)
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
          child: state.loadingMore
              ? const Center(child: NexusShimmer(width: double.infinity, height: 44, radius: NexusRadius.md))
              : OutlinedButton(
                  onPressed: () => notifier.loadMore(),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: NexusColors.primary,
                    side: const BorderSide(color: NexusColors.border),
                    minimumSize: const Size(double.infinity, 0),
                    padding: const EdgeInsets.symmetric(vertical: 12),
                    shape: RoundedRectangleBorder(borderRadius: NexusRadius.md),
                  ),
                  child: const Text('Load More', style: TextStyle(fontSize: 13)),
                ),
        ),
      const Gap(100),
    ]);
  }
}

// ── Notification tile ─────────────────────────────────────────────────────────

class _NotifTile extends StatelessWidget {
  final Map notif;
  final _NotifsNotifier notifier;
  const _NotifTile({required this.notif, required this.notifier});

  @override
  Widget build(BuildContext context) {
    final isRead = notif['is_read'] as bool? ?? false;
    final type   = notif['type']?.toString();
    final cfg    = _cfg(type);
    final date   = _formatDate(notif['created_at']?.toString() ?? '');

    return InkWell(
      onTap: !isRead ? () => notifier.markRead(notif['id']?.toString() ?? '') : null,
      child: Container(
        margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: isRead ? NexusColors.surface : cfg.color.withOpacity(0.05),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: isRead ? NexusColors.border : cfg.color.withOpacity(0.2)),
        ),
        child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
          // Type icon
          Container(
            width: 40, height: 40,
            decoration: BoxDecoration(
              color: cfg.color.withOpacity(0.12),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: cfg.color.withOpacity(0.2)),
            ),
            child: Center(child: Text(cfg.emoji, style: const TextStyle(fontSize: 18))),
          ),
          const SizedBox(width: 12),

          // Content
          Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              Expanded(child: Text(
                notif['title']?.toString() ?? '',
                style: TextStyle(
                  color: NexusColors.textPrimary,
                  fontWeight: isRead ? FontWeight.w400 : FontWeight.w600,
                  fontSize: 13),
              )),
              Text(date, style: const TextStyle(
                color: NexusColors.textSecondary, fontSize: 10)),
            ]),
            const SizedBox(height: 4),
            Text(
              notif['body']?.toString() ?? '',
              style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12),
              maxLines: 2, overflow: TextOverflow.ellipsis,
            ),
          ])),

          // Unread dot
          if (!isRead) ...[
            const SizedBox(width: 8),
            Container(
              width: 8, height: 8, margin: const EdgeInsets.only(top: 4),
              decoration: const BoxDecoration(
                color: NexusColors.primary, shape: BoxShape.circle),
            ),
          ],
        ]),
      ),
    );
  }

  String _formatDate(String iso) {
    try {
      final d = DateTime.parse(iso).toLocal();
      final now = DateTime.now();
      final diff = now.difference(d);
      if (diff.inMinutes < 1)  return 'Now';
      if (diff.inMinutes < 60) return '${diff.inMinutes}m';
      if (diff.inHours < 24)   return '${diff.inHours}h';
      return '${diff.inDays}d';
    } catch (_) { return ''; }
  }
}

// ── Group header ───────────────────────────────────────────────────────────────

class _GroupHeader extends StatelessWidget {
  final String title;
  const _GroupHeader(this.title);
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(20, 16, 20, 6),
    child: Text(title,
      style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11,
        fontWeight: FontWeight.w700, letterSpacing: 1.2)),
  );
}

// ── Empty state ────────────────────────────────────────────────────────────────

class _EmptyState extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(32),
      child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
        Container(
          width: 80, height: 80,
          decoration: BoxDecoration(
            color: NexusColors.surface,
            shape: BoxShape.circle,
            border: Border.all(color: NexusColors.border),
          ),
          child: const Center(child: Text('🔕', style: TextStyle(fontSize: 36))),
        ),
        const SizedBox(height: 20),
        const Text('All caught up',
          style: TextStyle(color: NexusColors.textPrimary, fontSize: 18,
            fontWeight: FontWeight.w600)),
        const SizedBox(height: 8),
        const Text(
          'You\'ll be notified when you win a spin, when your AI generation is ready, or when your streak is about to expire.',
          style: TextStyle(color: NexusColors.textSecondary, fontSize: 13),
          textAlign: TextAlign.center),
      ]),
    ),
  );
}

// ── Loading shimmer list ───────────────────────────────────────────────────────

class _LoadingList extends StatelessWidget {
  @override
  Widget build(BuildContext context) => ListView.builder(
    padding: const EdgeInsets.all(16),
    itemCount: 6,
    itemBuilder: (_, __) => Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: NexusShimmer(width: double.infinity, height: 72, radius: NexusRadius.md),
    ),
  );
}
