import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Provider ─────────────────────────────────────────────────────────────────

final _bonusProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getBonusPulseAwards();
});

// ─── Helpers ─────────────────────────────────────────────────────────────────

String _formatPoints(int pts) {
  if (pts >= 1000000) return '${(pts / 1000000).toStringAsFixed(1)}M';
  if (pts >= 1000)    return '${(pts / 1000).toStringAsFixed(1)}K';
  return '$pts';
}

String _formatDate(String iso) {
  try {
    final d = DateTime.parse(iso).toLocal();
    const months = ['Jan','Feb','Mar','Apr','May','Jun',
                    'Jul','Aug','Sep','Oct','Nov','Dec'];
    final h = d.hour.toString().padLeft(2, '0');
    final m = d.minute.toString().padLeft(2, '0');
    return '${d.day} ${months[d.month - 1]} ${d.year}, $h:$m';
  } catch (_) { return iso; }
}

// ─── Screen ───────────────────────────────────────────────────────────────────

class PulseAwardsScreen extends ConsumerWidget {
  const PulseAwardsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final dataAsync = ref.watch(_bonusProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.surface,
        title: const Text('Bonus Awards 🎁'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () => ref.invalidate(_bonusProvider),
          ),
        ],
      ),
      body: dataAsync.when(
        loading: () => _PulseShimmer(),
        error: (e, _) => Center(child: Column(
          mainAxisAlignment: MainAxisAlignment.center, children: [
            const Icon(Icons.wifi_off_rounded, size: 52, color: NexusColors.textSecondary),
            const SizedBox(height: 12),
            const Text('Could not load awards',
                style: TextStyle(color: NexusColors.textSecondary)),
            const SizedBox(height: 12),
            ElevatedButton(
              onPressed: () => ref.invalidate(_bonusProvider),
              child: const Text('Retry'),
            ),
          ],
        )),
        data: (data) {
          final total  = (data['total_bonus'] as num?)?.toInt() ?? 0;
          final awards = (data['awards'] as List?)?.cast<Map<String, dynamic>>() ?? [];

          return RefreshIndicator(
            color: NexusColors.gold,
            onRefresh: () async => ref.invalidate(_bonusProvider),
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: [
                // ── Total banner ──
                _TotalBanner(total: total, count: awards.length),
                const SizedBox(height: 20),

                // ── History ──
                if (awards.isEmpty) ...[
                  const SizedBox(height: 40),
                  Center(child: Column(children: [
                    const Icon(Icons.card_giftcard_rounded, size: 52, color: NexusColors.textSecondary),
                    const SizedBox(height: 16),
                    const Text('No bonus awards yet',
                        style: TextStyle(color: NexusColors.textPrimary, fontSize: 16,
                            fontWeight: FontWeight.w700)),
                    const SizedBox(height: 6),
                    const Text('Bonus Pulse Points from campaigns\nand promotions will appear here.',
                        textAlign: TextAlign.center,
                        style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
                  ])),
                ] else ...[
                  Text('${awards.length} Award${awards.length != 1 ? 's' : ''}',
                      style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11,
                          fontWeight: FontWeight.w700, letterSpacing: 0.8)),
                  const SizedBox(height: 8),
                  ...awards.asMap().entries.map((e) => Padding(
                    padding: const EdgeInsets.only(bottom: 10),
                    child: _AwardCard(award: e.value, index: e.key),
                  )),
                ],
              ],
            ),
          );
        },
      ),
    );
  }
}

// ─── Total Banner ─────────────────────────────────────────────────────────────

class _TotalBanner extends StatelessWidget {
  final int total, count;
  const _TotalBanner({required this.total, required this.count});

  @override
  Widget build(BuildContext context) => Container(
    width: double.infinity,
    padding: const EdgeInsets.all(22),
    decoration: BoxDecoration(
      borderRadius: BorderRadius.circular(20),
      gradient: const LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: [Color(0xFF4A56EE), Color(0xFF8B5CF6), Color(0x4DF9C74F)],
      ),
    ),
    child: Stack(children: [
      // Radial highlight
      Positioned(top: -20, right: -20,
        child: Container(
          width: 100, height: 100,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            gradient: RadialGradient(colors: [
              Colors.white.withValues(alpha: 0.15), Colors.transparent,
            ]),
          ),
        ),
      ),
      Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('Total Bonus Points Received'.toUpperCase(),
            style: TextStyle(color: Colors.white.withValues(alpha: 0.7), fontSize: 10,
                fontWeight: FontWeight.w700, letterSpacing: 0.8)),
        const SizedBox(height: 8),
        Text(_formatPoints(total),
            style: const TextStyle(color: Colors.white, fontSize: 40,
                fontWeight: FontWeight.w900, height: 1)),
        const SizedBox(height: 4),
        Text('$count award${count != 1 ? "s" : ""} received',
            style: TextStyle(color: Colors.white.withValues(alpha: 0.6), fontSize: 12)),
      ]),
    ]),
  );
}

// ─── Award Card ───────────────────────────────────────────────────────────────

class _AwardCard extends StatelessWidget {
  final Map<String, dynamic> award;
  final int index;
  const _AwardCard({required this.award, required this.index});

  @override
  Widget build(BuildContext context) {
    final campaign  = award['campaign'] as String? ?? 'Campaign Award';
    final note      = award['note'] as String?;
    final points    = (award['points'] as num?)?.toInt() ?? 0;
    final createdAt = award['created_at'] as String? ?? '';

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: Colors.white.withValues(alpha: 0.06)),
      ),
      child: Row(children: [
        // Rank badge
        Container(
          width: 40, height: 40,
          decoration: BoxDecoration(
            color: NexusColors.primary.withValues(alpha: 0.15),
            borderRadius: BorderRadius.circular(12),
          ),
          child: const Center(child: Text('🎁', style: TextStyle(fontSize: 18))),
        ),
        const SizedBox(width: 12),

        // Details
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(campaign,
              style: const TextStyle(color: NexusColors.textPrimary, fontSize: 13,
                  fontWeight: FontWeight.w700),
              maxLines: 1, overflow: TextOverflow.ellipsis),
          if (note != null) ...[
            const SizedBox(height: 2),
            Text(note, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11),
                maxLines: 1, overflow: TextOverflow.ellipsis),
          ],
          const SizedBox(height: 3),
          Text(_formatDate(createdAt),
              style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
        ])),

        // Points
        Column(crossAxisAlignment: CrossAxisAlignment.end, children: [
          Text('+${_formatPoints(points)}',
              style: const TextStyle(color: NexusColors.primary, fontSize: 18,
                  fontWeight: FontWeight.w900)),
          const Text('pts', style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
        ]),
      ]),
    );
  }
}

// ── Shimmer ───────────────────────────────────────────────────────────────────
class _PulseShimmer extends StatelessWidget {
  @override
  Widget build(BuildContext context) => ListView.builder(
    padding: const EdgeInsets.all(16),
    itemCount: 6,
    itemBuilder: (_, __) => Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: NexusShimmer(width: double.infinity, height: 76, radius: NexusRadius.md),
    ),
  );
}
