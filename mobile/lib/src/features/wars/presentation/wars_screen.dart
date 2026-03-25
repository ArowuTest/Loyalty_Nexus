import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

final _leaderboardProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  return ref.read(warsApiProvider).getLeaderboard();
});

final _myRankProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(warsApiProvider).getMyRank();
});

class WarsScreen extends ConsumerWidget {
  const WarsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lbAsync = ref.watch(_leaderboardProvider);
    final rankAsync = ref.watch(_myRankProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Regional Wars 🌍'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () {
              ref.invalidate(_leaderboardProvider);
              ref.invalidate(_myRankProvider);
            },
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async {
          ref.invalidate(_leaderboardProvider);
          ref.invalidate(_myRankProvider);
        },
        color: NexusColors.primary,
        child: SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.all(20),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            // Prize pool card
            rankAsync.when(
              loading: () => _prizeCard('₦—', '...'),
              error: (_, __) => _prizeCard('₦500,000', 'Prize pool'),
              data: (rank) => _prizeCard(
                '₦${_fmt(rank['prize_kobo'] as int? ?? 50000000)}',
                'Rank #${rank['rank'] ?? '?'} in ${rank['state'] ?? 'your state'}',
              ),
            ),
            const SizedBox(height: 24),
            Text('State Leaderboard', style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 12),
            lbAsync.when(
              loading: () => const Center(
                  child: CircularProgressIndicator(color: NexusColors.primary)),
              error: (e, _) => Center(
                child: Padding(
                  padding: const EdgeInsets.all(40),
                  child: Column(children: [
                    const Icon(Icons.wifi_off, color: NexusColors.textSecondary, size: 48),
                    const SizedBox(height: 12),
                    Text('Could not load leaderboard',
                        style: TextStyle(color: NexusColors.textSecondary)),
                  ]),
                ),
              ),
              data: (rows) => ListView.separated(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: rows.length,
                separatorBuilder: (_, __) => const SizedBox(height: 8),
                itemBuilder: (_, i) {
                  final row = rows[i] as Map;
                  final rank = (row['rank'] as num?)?.toInt() ?? (i + 1);
                  final medal = rank == 1
                      ? '🥇'
                      : rank == 2
                          ? '🥈'
                          : rank == 3
                              ? '🥉'
                              : '$rank';
                  return Container(
                    padding: const EdgeInsets.all(14),
                    decoration: BoxDecoration(
                      color: NexusColors.surface,
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(
                          color: rank <= 3
                              ? NexusColors.gold.withOpacity(0.3)
                              : NexusColors.border)),
                    child: Row(children: [
                      SizedBox(
                        width: 36,
                        child: Text(medal,
                            style: const TextStyle(fontSize: 20),
                            textAlign: TextAlign.center),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                          Text(row['state']?.toString() ?? '—',
                              style: const TextStyle(
                                  color: NexusColors.textPrimary,
                                  fontWeight: FontWeight.bold)),
                          Text(
                              '${row['active_members'] ?? 0} members',
                              style: const TextStyle(
                                  color: NexusColors.textSecondary, fontSize: 12)),
                        ]),
                      ),
                      Text(
                        '${_fmt((row['total_points'] as num?)?.toInt() ?? 0)} pts',
                        style: const TextStyle(
                            color: NexusColors.primary, fontWeight: FontWeight.bold),
                      ),
                    ]),
                  );
                },
              ),
            ),
          ]),
        ),
      ),
    );
  }

  Widget _prizeCard(String amount, String subtitle) => Container(
    width: double.infinity,
    padding: const EdgeInsets.all(20),
    decoration: BoxDecoration(
      gradient: const LinearGradient(
          colors: [Color(0xFF10B981), Color(0xFF059669)]),
      borderRadius: BorderRadius.circular(16)),
    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      const Text('Monthly Prize Pool',
          style: TextStyle(color: Colors.white70, fontSize: 12)),
      const SizedBox(height: 4),
      Text(amount,
          style: const TextStyle(
              color: Colors.white, fontSize: 32, fontWeight: FontWeight.bold)),
      Text(subtitle, style: const TextStyle(color: Colors.white70, fontSize: 12)),
    ]),
  );

  String _fmt(int n) {
    if (n >= 1000000) return '${(n / 1000000).toStringAsFixed(1)}M';
    if (n >= 1000) return '${(n / 1000).toStringAsFixed(1)}K';
    return n.toString();
  }
}
