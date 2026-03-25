import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

final _notifsProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(notificationsApiProvider).list();
});

class NotificationsScreen extends ConsumerWidget {
  const NotificationsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(_notifsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Notifications 🔔'),
        actions: [
          TextButton(
            onPressed: () async {
              await ref.read(notificationsApiProvider).markAllRead();
              ref.invalidate(_notifsProvider);
            },
            child: const Text('Mark all read', style: TextStyle(color: NexusColors.primary)),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(_notifsProvider),
        color: NexusColors.primary,
        child: async.when(
          loading: () => const Center(child: CircularProgressIndicator(color: NexusColors.primary)),
          error: (_, __) => const Center(child: Text('Could not load notifications')),
          data: (data) {
            final items = data['notifications'] as List? ?? [];
            if (items.isEmpty) {
              return Center(child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                const Text('🔔', style: TextStyle(fontSize: 64)),
                const SizedBox(height: 16),
                Text('No notifications yet', style: TextStyle(color: NexusColors.textSecondary, fontSize: 18)),
              ]));
            }
            return ListView.separated(
              padding: const EdgeInsets.symmetric(vertical: 8),
              itemCount: items.length,
              separatorBuilder: (_, __) => Divider(color: NexusColors.border, height: 1),
              itemBuilder: (ctx, i) {
                final n = items[i] as Map;
                final isRead = n['is_read'] as bool? ?? false;
                return ListTile(
                  tileColor: isRead ? null : NexusColors.primary.withOpacity(0.05),
                  leading: Container(
                    width: 44, height: 44,
                    decoration: BoxDecoration(
                      color: NexusColors.surface,
                      shape: BoxShape.circle,
                      border: Border.all(color: NexusColors.border)),
                    child: Center(child: Text(_typeEmoji(n['type']?.toString()),
                        style: const TextStyle(fontSize: 20))),
                  ),
                  title: Text(n['title']?.toString() ?? '',
                      style: TextStyle(
                          color: NexusColors.textPrimary,
                          fontWeight: isRead ? FontWeight.normal : FontWeight.bold)),
                  subtitle: Text(n['body']?.toString() ?? '',
                      style: const TextStyle(color: NexusColors.textSecondary, fontSize: 13),
                      maxLines: 2, overflow: TextOverflow.ellipsis),
                  trailing: Text(_formatDate(n['created_at']?.toString() ?? ''),
                      style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
                  onTap: () async {
                    if (!isRead) {
                      await ref.read(notificationsApiProvider).markRead(n['id']?.toString() ?? '');
                      ref.invalidate(_notifsProvider);
                    }
                  },
                );
              },
            );
          },
        ),
      ),
    );
  }

  String _typeEmoji(String? type) {
    switch (type) {
      case 'spin_win': return '🎰';
      case 'draw_result': return '🏆';
      case 'streak_warn': return '🔥';
      case 'subscription_warn':
      case 'subscription_expired': return '⏰';
      case 'wars_result': return '🗺️';
      case 'studio_ready': return '✨';
      case 'marketing': return '📢';
      default: return '🔔';
    }
  }

  String _formatDate(String iso) {
    try {
      final d = DateTime.parse(iso).toLocal();
      final now = DateTime.now();
      final diff = now.difference(d);
      if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
      if (diff.inHours < 24) return '${diff.inHours}h ago';
      return '${diff.inDays}d ago';
    } catch (_) { return ''; }
  }
}
