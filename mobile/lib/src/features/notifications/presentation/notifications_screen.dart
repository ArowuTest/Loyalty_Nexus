import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ── Providers ──────────────────────────────────────────────────────────────────

final _notifsProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(notificationsApiProvider).list();
});

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
    final notifsAsync = ref.watch(_notifsProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        title: Row(children: [
          const Text('Notifications 🔔'),
          const SizedBox(width: 8),
          notifsAsync.when(
            data: (d) {
              final unread = d['unread_count'] as int? ?? 0;
              return unread > 0
                  ? Container(
                      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
                      decoration: BoxDecoration(
                        color: NexusColors.primary,
                        borderRadius: BorderRadius.circular(20),
                      ),
                      child: Text('$unread',
                        style: const TextStyle(color: Colors.white, fontSize: 11,
                          fontWeight: FontWeight.w800)),
                    )
                  : const SizedBox.shrink();
            },
            loading: () => const SizedBox.shrink(),
            error: (_, __) => const SizedBox.shrink(),
          ),
        ]),
        actions: [
          notifsAsync.when(
            data: (d) {
              final unread = d['unread_count'] as int? ?? 0;
              return unread > 0
                  ? TextButton(
                      onPressed: () async {
                        await ref.read(notificationsApiProvider).markAllRead();
                        ref.invalidate(_notifsProvider);
                      },
                      style: TextButton.styleFrom(foregroundColor: NexusColors.primary),
                      child: const Text('Mark all read', style: TextStyle(fontSize: 12)),
                    )
                  : const SizedBox.shrink();
            },
            loading: () => const SizedBox.shrink(),
            error: (_, __) => const SizedBox.shrink(),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(_notifsProvider),
        color: NexusColors.primary,
        child: notifsAsync.when(
          loading: () => _LoadingList(),
          error: (_, __) => _EmptyState(),
          data: (data) {
            final items = data['notifications'] as List? ?? [];

            if (items.isEmpty) return _EmptyState();

            // Group: today vs earlier
            final now   = DateTime.now();
            final today = DateTime(now.year, now.month, now.day);
            final todayItems    = <Map>[];
            final earlierItems  = <Map>[];
            for (final item in items) {
              final n = item as Map;
              final d = DateTime.tryParse(n['created_at']?.toString() ?? '') ?? DateTime(2000);
              if (d.isAfter(today)) { todayItems.add(n); } else { earlierItems.add(n); }
            }

            return ListView(children: [
              if (todayItems.isNotEmpty) ...[
                _GroupHeader('Today'),
                ...todayItems.map((n) => _NotifTile(notif: n, ref: ref)),
              ],
              if (earlierItems.isNotEmpty) ...[
                _GroupHeader('Earlier'),
                ...earlierItems.map((n) => _NotifTile(notif: n, ref: ref)),
              ],
              const SizedBox(height: 100),
            ]);
          },
        ),
      ),
    );
  }
}

// ── Notification tile ─────────────────────────────────────────────────────────

class _NotifTile extends StatelessWidget {
  final Map notif;
  final WidgetRef ref;
  const _NotifTile({required this.notif, required this.ref});

  @override
  Widget build(BuildContext context) {
    final isRead = notif['is_read'] as bool? ?? false;
    final type   = notif['type']?.toString();
    final cfg    = _cfg(type);
    final date   = _formatDate(notif['created_at']?.toString() ?? '');

    return InkWell(
      onTap: !isRead ? () async {
        await ref.read(notificationsApiProvider).markRead(notif['id']?.toString() ?? '');
        ref.invalidate(_notifsProvider);
      } : null,
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
