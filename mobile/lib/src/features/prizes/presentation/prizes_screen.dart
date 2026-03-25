import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

final _drawsProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  return ref.read(drawsApiProvider).listUpcoming();
});

class PrizesScreen extends ConsumerWidget {
  const PrizesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final drawsAsync = ref.watch(_drawsProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Prizes & Draws 🏆')),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(_drawsProvider),
        color: NexusColors.primary,
        child: drawsAsync.when(
          loading: () => const Center(
              child: CircularProgressIndicator(color: NexusColors.primary)),
          error: (e, _) => Center(
            child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
              const Icon(Icons.wifi_off, size: 64, color: NexusColors.textSecondary),
              const SizedBox(height: 12),
              const Text('Could not load draws',
                  style: TextStyle(color: NexusColors.textSecondary)),
              const SizedBox(height: 12),
              ElevatedButton(
                  onPressed: () => ref.invalidate(_drawsProvider),
                  child: const Text('Retry')),
            ]),
          ),
          data: (draws) => draws.isEmpty
              ? Center(
                  child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                    const Text('🎁', style: TextStyle(fontSize: 64)),
                    const SizedBox(height: 16),
                    Text('No upcoming draws',
                        style: TextStyle(color: NexusColors.textSecondary,
                            fontSize: 18)),
                    const SizedBox(height: 8),
                    Text('Keep spinning to earn draw entries!',
                        style: TextStyle(color: NexusColors.textSecondary)),
                  ]))
              : ListView.separated(
                  padding: const EdgeInsets.all(20),
                  itemCount: draws.length,
                  separatorBuilder: (_, __) => const SizedBox(height: 12),
                  itemBuilder: (ctx, i) {
                    final draw = draws[i] as Map;
                    return _DrawCard(draw: draw);
                  },
                ),
        ),
      ),
    );
  }
}

class _DrawCard extends StatelessWidget {
  final Map draw;
  const _DrawCard({required this.draw});

  @override
  Widget build(BuildContext context) {
    final prizeKobo = (draw['prize_pool_kobo'] as num?)?.toInt() ?? 0;
    final prizeNaira = (prizeKobo / 100).toStringAsFixed(0);
    final entries = draw['entry_count'] ?? 0;
    final drawDate = draw['draw_date']?.toString() ?? '';
    final status = draw['status']?.toString() ?? 'SCHEDULED';

    Color statusColor;
    switch (status) {
      case 'SCHEDULED': statusColor = NexusColors.primary; break;
      case 'IN_PROGRESS': statusColor = NexusColors.gold; break;
      case 'COMPLETED': statusColor = NexusColors.green; break;
      default: statusColor = NexusColors.textSecondary;
    }

    return Container(
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: NexusColors.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        // Header gradient
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            gradient: const LinearGradient(
                colors: [Color(0xFF4A56EE), Color(0xFF8B5CF6)]),
            borderRadius: const BorderRadius.vertical(top: Radius.circular(16))),
          child: Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
            Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(draw['name']?.toString() ?? 'Monthly Draw',
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 18,
                      fontWeight: FontWeight.bold)),
              const SizedBox(height: 4),
              Text('₦$prizeNaira prize pool',
                  style: const TextStyle(color: Colors.white70, fontSize: 13)),
            ]),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              decoration: BoxDecoration(
                  color: statusColor.withOpacity(0.2),
                  borderRadius: BorderRadius.circular(20),
                  border: Border.all(color: statusColor)),
              child: Text(status,
                  style: TextStyle(
                      color: statusColor, fontSize: 12, fontWeight: FontWeight.bold)),
            ),
          ]),
        ),
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(children: [
            _infoChip('📅', _formatDate(drawDate)),
            const SizedBox(width: 12),
            _infoChip('🎟', '$entries entries'),
          ]),
        ),
      ]),
    );
  }

  Widget _infoChip(String icon, String label) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
    decoration: BoxDecoration(
        color: NexusColors.background,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: NexusColors.border)),
    child: Row(mainAxisSize: MainAxisSize.min, children: [
      Text(icon, style: const TextStyle(fontSize: 14)),
      const SizedBox(width: 6),
      Text(label, style: const TextStyle(
          color: NexusColors.textSecondary, fontSize: 12)),
    ]),
  );

  String _formatDate(String iso) {
    try {
      final d = DateTime.parse(iso).toLocal();
      return '${d.day}/${d.month}/${d.year}';
    } catch (_) { return iso; }
  }
}
