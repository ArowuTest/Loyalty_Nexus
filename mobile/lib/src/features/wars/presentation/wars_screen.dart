import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ── Providers ──────────────────────────────────────────────────────────────────

final _leaderboardProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  return ref.read(warsApiProvider).getLeaderboard();
});

final _myRankProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(warsApiProvider).getMyRank();
});

// ── Main screen ────────────────────────────────────────────────────────────────

class WarsScreen extends ConsumerWidget {
  const WarsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lbAsync   = ref.watch(_leaderboardProvider);
    final rankAsync = ref.watch(_myRankProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        title: const Text('Regional Wars 🌍'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            color: NexusColors.textSecondary,
            onPressed: () {
              ref.invalidate(_leaderboardProvider);
              ref.invalidate(_myRankProvider);
            },
          ),
        ],
      ),
      body: RefreshIndicator(
        color: NexusColors.primary,
        onRefresh: () async {
          ref.invalidate(_leaderboardProvider);
          ref.invalidate(_myRankProvider);
        },
        child: SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(20, 8, 20, 100),
          child: Column(children: [
            // Prize pool banner (server-driven — shown only when data loads)
            lbAsync.when(
              loading: () => const SizedBox(height: 8),
              error: (_, __) => const SizedBox.shrink(),
              data: (_) => _PrizePoolBanner(),
            ),
            const SizedBox(height: 16),

            // How it works
            _HowItWorksRow(),
            const SizedBox(height: 20),

            // My rank card
            rankAsync.when(
              loading: () => const _ShimmerCard(height: 80),
              error: (_, __) => const SizedBox.shrink(), // silent — might not have state set
              data: (rank) => rank.isEmpty ? const SizedBox.shrink() : _MyRankCard(rank: rank),
            ),

            if (rankAsync.hasValue && rankAsync.value?.isNotEmpty == true)
              const SizedBox(height: 16),

            // Leaderboard
            lbAsync.when(
              loading: () => Column(children: List.generate(6, (_) => Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: _ShimmerCard(height: 64),
              ))),
              error: (e, _) => _ErrorState(onRetry: () => ref.invalidate(_leaderboardProvider)),
              data: (lb) => lb.isEmpty
                  ? _EmptyState()
                  : _Leaderboard(rows: lb),
            ),
          ]),
        ),
      ),
    );
  }
}

// ── Prize pool banner ─────────────────────────────────────────────────────────

class _PrizePoolBanner extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Container(
    width: double.infinity,
    padding: const EdgeInsets.all(20),
    decoration: BoxDecoration(
      gradient: const LinearGradient(
        colors: [Color(0xFF059669), Color(0xFF10b981)],
        begin: Alignment.topLeft, end: Alignment.bottomRight,
      ),
      borderRadius: BorderRadius.circular(22),
      boxShadow: [BoxShadow(color: const Color(0xFF10b981).withOpacity(0.3), blurRadius: 20)],
    ),
    child: Stack(children: [
      Positioned(right: 0, top: 0, bottom: 0,
        child: Center(child: Text('🌍',
          style: TextStyle(fontSize: 64, color: Colors.white.withOpacity(0.15))))),
      Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('Monthly Prize Pool', style: TextStyle(
          color: Colors.white.withOpacity(0.8), fontSize: 11, letterSpacing: 1.2)),
        const SizedBox(height: 4),
        const Text('₦500,000', style: TextStyle(
          color: Colors.white, fontSize: 32, fontWeight: FontWeight.w800, letterSpacing: -1)),
        const SizedBox(height: 6),
        Text('Top 3 states share the pool • Resets monthly',
          style: TextStyle(color: Colors.white.withOpacity(0.7), fontSize: 12)),
      ]),
    ]),
  );
}

// ── How it works row ──────────────────────────────────────────────────────────

class _HowItWorksRow extends StatelessWidget {
  static const _steps = [
    ('⚡', 'Recharge', 'Earn points on ₦200+'),
    ('👥', 'Team Up',  'Your state rank rises'),
    ('🏆', 'Win',      'Monthly prizes awarded'),
  ];

  @override
  Widget build(BuildContext context) => Row(
    children: _steps.map((s) => Expanded(child: Container(
      margin: const EdgeInsets.symmetric(horizontal: 4),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: NexusColors.border),
      ),
      child: Column(children: [
        Text(s.$1, style: const TextStyle(fontSize: 22)),
        const SizedBox(height: 6),
        Text(s.$2, style: const TextStyle(color: NexusColors.textPrimary,
          fontSize: 11, fontWeight: FontWeight.w600), textAlign: TextAlign.center),
        const SizedBox(height: 2),
        Text(s.$3, style: const TextStyle(color: NexusColors.textSecondary,
          fontSize: 10), textAlign: TextAlign.center),
      ]),
    ))).toList(),
  );
}

// ── My rank card ──────────────────────────────────────────────────────────────

class _MyRankCard extends StatelessWidget {
  final Map<String, dynamic> rank;
  const _MyRankCard({required this.rank});

  @override
  Widget build(BuildContext context) {
    final state     = rank['state']?.toString();
    final position  = rank['rank'] as int? ?? rank['position'] as int?;
    final points    = rank['points'] as int? ?? rank['total_points'] as int? ?? 0;

    if (state == null) return const SizedBox.shrink();

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        gradient: LinearGradient(colors: [
          NexusColors.primary.withOpacity(0.15),
          NexusColors.primary.withOpacity(0.05),
        ]),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: NexusColors.primary.withOpacity(0.3)),
      ),
      child: Row(children: [
        const Text('📍', style: TextStyle(fontSize: 28)),
        const SizedBox(width: 12),
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text('Your State: $state',
            style: const TextStyle(color: NexusColors.textPrimary,
              fontSize: 14, fontWeight: FontWeight.w600)),
          const SizedBox(height: 3),
          Text('${_fmtPts(points)} contribution',
            style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
        ])),
        if (position != null)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
            decoration: BoxDecoration(
              color: NexusColors.primary.withOpacity(0.2),
              borderRadius: BorderRadius.circular(20),
            ),
            child: Text('#$position',
              style: const TextStyle(color: NexusColors.primary,
                fontSize: 18, fontWeight: FontWeight.w800)),
          ),
      ]),
    );
  }

  String _fmtPts(int v) => v >= 1000 ? '${(v / 1000).toStringAsFixed(1)}k pts' : '$v pts';
}

// ── Leaderboard ────────────────────────────────────────────────────────────────

class _Leaderboard extends StatelessWidget {
  final List<dynamic> rows;
  const _Leaderboard({required this.rows});

  static const _medals = ['🥇', '🥈', '🥉'];

  @override
  Widget build(BuildContext context) => Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
    const Row(children: [
      Icon(Icons.emoji_events_rounded, size: 16, color: NexusColors.gold),
      SizedBox(width: 6),
      Text('State Leaderboard',
        style: TextStyle(color: NexusColors.textPrimary, fontSize: 14, fontWeight: FontWeight.w700)),
    ]),
    const SizedBox(height: 10),
    ...List.generate(rows.length, (i) {
      final row    = rows[i] as Map;
      final rank   = i + 1;
      final state  = row['state']?.toString() ?? '—';
      final points = row['points'] as int? ?? row['total_points'] as int? ?? 0;
      final members= row['members'] as int? ?? row['member_count'] as int?;
      final change = row['change'] as int? ?? row['rank_change'] as int? ?? 0;
      return _LeaderboardRow(
        rank:    rank,
        medal:   rank <= 3 ? _medals[rank - 1] : null,
        state:   state,
        points:  points,
        members: members,
        change:  change,
      );
    }),
  ]);
}

class _LeaderboardRow extends StatelessWidget {
  final int rank;
  final String? medal;
  final String state;
  final int points;
  final int? members;
  final int change;
  const _LeaderboardRow({
    required this.rank, this.medal, required this.state,
    required this.points, this.members, required this.change,
  });

  @override
  Widget build(BuildContext context) => Container(
    margin: const EdgeInsets.only(bottom: 8),
    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
    decoration: BoxDecoration(
      color: NexusColors.surface,
      borderRadius: BorderRadius.circular(16),
      border: Border.all(color: rank <= 3 ? NexusColors.gold.withOpacity(0.25) : NexusColors.border),
    ),
    child: Row(children: [
      // Rank
      SizedBox(width: 36, child: Center(
        child: medal != null
            ? Text(medal!, style: const TextStyle(fontSize: 20))
            : Text('$rank', style: const TextStyle(
                color: NexusColors.textSecondary, fontSize: 13, fontWeight: FontWeight.w700)),
      )),
      const SizedBox(width: 8),
      // State info
      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(state, style: const TextStyle(color: NexusColors.textPrimary,
          fontSize: 14, fontWeight: FontWeight.w600)),
        if (members != null)
          Text('${_fmtNum(members!)} members',
            style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
      ])),
      // Points + change
      Column(crossAxisAlignment: CrossAxisAlignment.end, children: [
        Text(_fmtPts(points),
          style: const TextStyle(color: NexusColors.textPrimary,
            fontSize: 14, fontWeight: FontWeight.w700)),
        if (change != 0 || true) ...[
          const SizedBox(height: 2),
          Text(
            change > 0 ? '▲$change' : change < 0 ? '▼${change.abs()}' : '—',
            style: TextStyle(
              color: change > 0 ? NexusColors.green
                   : change < 0 ? NexusColors.red
                   : NexusColors.textSecondary,
              fontSize: 11, fontWeight: FontWeight.w600),
          ),
        ],
      ]),
    ]),
  );

  String _fmtPts(int v) => v >= 1000 ? '${(v / 1000).toStringAsFixed(1)}k' : '$v';
  String _fmtNum(int v) => v >= 1000 ? '${(v / 1000).toStringAsFixed(1)}k' : '$v';
}

// ── Empty & error states ──────────────────────────────────────────────────────

class _EmptyState extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 40),
    child: Column(children: [
      const Text('🌍', style: TextStyle(fontSize: 56)),
      const SizedBox(height: 16),
      const Text('Wars haven't begun yet',
        style: TextStyle(color: NexusColors.textPrimary, fontSize: 18, fontWeight: FontWeight.w600)),
      const SizedBox(height: 8),
      const Text('Recharge to start earning points for your state.',
        style: TextStyle(color: NexusColors.textSecondary, fontSize: 13),
        textAlign: TextAlign.center),
    ]),
  );
}

class _ErrorState extends StatelessWidget {
  final VoidCallback onRetry;
  const _ErrorState({required this.onRetry});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 40),
    child: Column(children: [
      const Icon(Icons.wifi_off_rounded, size: 48, color: NexusColors.textSecondary),
      const SizedBox(height: 16),
      const Text('Could not load leaderboard',
        style: TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
      const SizedBox(height: 12),
      ElevatedButton.icon(
        onPressed: onRetry,
        icon: const Icon(Icons.refresh_rounded, size: 16),
        label: const Text('Retry'),
      ),
    ]),
  );
}

// ── Shimmer placeholder ────────────────────────────────────────────────────────

class _ShimmerCard extends StatelessWidget {
  final double height;
  const _ShimmerCard({required this.height});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 8),
    child: NexusShimmer(width: double.infinity, height: height, radius: NexusRadius.md),
  );
}
