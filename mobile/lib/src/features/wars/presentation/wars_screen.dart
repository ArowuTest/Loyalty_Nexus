import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:gap/gap.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ── Providers ──────────────────────────────────────────────────────────────────

final _leaderboardProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final raw = await ref.read(warsApiProvider).getLeaderboard();
  return {'leaderboard': raw, 'period': '', 'count': raw.length};
});

final _myRankProvider = FutureProvider.autoDispose<Map<String, dynamic>?>((ref) async {
  try {
    final r = await ref.read(warsApiProvider).getMyRank();
    return Map<String, dynamic>.from(r);
  } catch (_) {}
  return null;
});

// ── Helpers ────────────────────────────────────────────────────────────────────

String _fmtKobo(int kobo) {
  final n = kobo / 100;
  if (n >= 1000000) return '₦${(n / 1000000).toStringAsFixed(1)}M';
  if (n >= 1000)    return '₦${(n / 1000).toStringAsFixed(0)}K';
  return '₦${n.toStringAsFixed(0)}';
}

String _fmtPts(int pts) {
  if (pts >= 1000000) return '${(pts / 1000000).toStringAsFixed(1)}M';
  if (pts >= 1000)    return '${(pts / 1000).toStringAsFixed(1)}K';
  return '$pts';
}

int _daysUntilEnd(String period) {
  if (period.isEmpty) return 0;
  try {
    final parts = period.split('-').map(int.parse).toList();
    if (parts.length < 2) return 0;
    final end = DateTime(parts[0], parts[1] + 1); // 1st of next month
    return end.difference(DateTime.now()).inDays.clamp(0, 31);
  } catch (_) { return 0; }
}

// ── Main Screen ────────────────────────────────────────────────────────────────

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
        child: lbAsync.when(
          loading: () => _LoadingView(),
          error: (e, _) => _ErrorView(onRetry: () => ref.invalidate(_leaderboardProvider)),
          data: (data) {
            final lb = (data['leaderboard'] as List? ?? []);
            final period = data['period']?.toString() ?? '';

            // No active war — show animated "coming soon" state
            if (lb.isEmpty) return _NoActiveWarView();

            final daysLeft = _daysUntilEnd(period);
            final top3Kobo = lb.take(3).fold<int>(
              0, (sum, e) => sum + ((e as Map)['prize_kobo'] as int? ?? 0));

            return _ActiveWarView(
              leaderboard: lb,
              period: period,
              daysLeft: daysLeft,
              top3Kobo: top3Kobo,
              rankAsync: rankAsync,
            );
          },
        ),
      ),
    );
  }
}

// ── Active War View ────────────────────────────────────────────────────────────

class _ActiveWarView extends StatelessWidget {
  final List<dynamic> leaderboard;
  final String period;
  final int daysLeft;
  final int top3Kobo;
  final AsyncValue<Map<String, dynamic>?> rankAsync;

  const _ActiveWarView({
    required this.leaderboard,
    required this.period,
    required this.daysLeft,
    required this.top3Kobo,
    required this.rankAsync,
  });

  @override
  Widget build(BuildContext context) => SingleChildScrollView(
    physics: const AlwaysScrollableScrollPhysics(),
    padding: const EdgeInsets.fromLTRB(20, 8, 20, 100),
    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [

      // ── Prize pool hero ──────────────────────────────────────────────────
      _PrizePoolBanner(kobo: top3Kobo, daysLeft: daysLeft, period: period)
          .animate().fadeIn(duration: 400.ms).slideY(begin: -0.1, end: 0),

      const Gap(16),

      // ── My rank card ────────────────────────────────────────────────────
      rankAsync.when(
        loading: () => NexusShimmer(width: double.infinity, height: 80, radius: NexusRadius.md),
        error: (_, __) => const SizedBox.shrink(),
        data: (rank) {
          if (rank == null) return const SizedBox.shrink();
          final ranked = rank['ranked'] as bool? ?? false;
          if (!ranked) return const SizedBox.shrink();
          final entry = rank['entry'] as Map?;
          if (entry == null) return const SizedBox.shrink();
          return _MyRankCard(entry: entry)
              .animate().fadeIn(duration: 400.ms, delay: 100.ms);
        },
      ),

      const Gap(16),

      // ── How it works ─────────────────────────────────────────────────────
      const _HowItWorksRow(),

      const Gap(20),

      // ── Leaderboard ──────────────────────────────────────────────────────
      Row(children: [
        const Icon(Icons.emoji_events_rounded, size: 16, color: NexusColors.gold),
        const Gap(6),
        const Text('State Leaderboard',
            style: TextStyle(color: NexusColors.textPrimary,
                fontSize: 14, fontWeight: FontWeight.w700)),
        const Spacer(),
        Text('${leaderboard.length} states',
            style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
      ]),
      const Gap(10),

      ...List.generate(leaderboard.length, (i) {
        final row    = leaderboard[i] as Map;
        final rank   = i + 1;
        final state  = row['state']?.toString() ?? '—';
        final points = row['points'] as int? ?? row['total_points'] as int? ?? 0;
        final members = row['members'] as int? ?? row['active_members'] as int?;
        final prizeK = row['prize_kobo'] as int? ?? 0;
        final change = row['change'] as int? ?? row['rank_change'] as int? ?? 0;
        return _LeaderboardRow(
          rank: rank,
          state: state,
          points: points,
          members: members,
          prizeKobo: prizeK,
          change: change,
        ).animate().fadeIn(duration: 300.ms, delay: (i * 40).ms);
      }),
    ]),
  );
}

// ── Prize Pool Banner ─────────────────────────────────────────────────────────

class _PrizePoolBanner extends StatelessWidget {
  final int kobo;
  final int daysLeft;
  final String period;
  const _PrizePoolBanner({required this.kobo, required this.daysLeft, required this.period});

  @override
  Widget build(BuildContext context) => Container(
    width: double.infinity,
    padding: const EdgeInsets.all(20),
    decoration: BoxDecoration(
      gradient: const LinearGradient(
        colors: [Color(0xFF1A1F3A), Color(0xFF0D1020)],
        begin: Alignment.topLeft, end: Alignment.bottomRight,
      ),
      borderRadius: NexusRadius.xl,
      border: Border.all(color: NexusColors.gold.withValues(alpha: 0.3)),
      boxShadow: [BoxShadow(color: NexusColors.gold.withValues(alpha: 0.08),
          blurRadius: 24, spreadRadius: 2)],
    ),
    child: Column(children: [
      // Title row
      Row(mainAxisAlignment: MainAxisAlignment.center, children: [
        const Text('🏆', style: TextStyle(fontSize: 20)),
        const Gap(8),
        const Text('Top 3 States Prize Pool',
            style: TextStyle(color: NexusColors.gold,
                fontSize: 12, fontWeight: FontWeight.w700, letterSpacing: 0.5)),
      ]),
      const Gap(8),
      // Prize amount
      Text(kobo > 0 ? _fmtKobo(kobo) : 'Prize TBD',
          style: const TextStyle(color: Colors.white,
              fontSize: 36, fontWeight: FontWeight.w900, letterSpacing: -1)),
      const Gap(12),
      // Stats row
      Row(mainAxisAlignment: MainAxisAlignment.center, children: [
        if (period.isNotEmpty) ...[
          _StatChip(icon: Icons.calendar_month_rounded,
              label: period, color: NexusColors.textSecondary),
          const Gap(12),
        ],
        if (daysLeft > 0)
          _StatChip(
            icon: Icons.timer_outlined,
            label: '$daysLeft days left',
            color: daysLeft <= 5 ? NexusColors.red : NexusColors.green,
          ),
      ]),
    ]),
  );
}

class _StatChip extends StatelessWidget {
  final IconData icon;
  final String label;
  final Color color;
  const _StatChip({required this.icon, required this.label, required this.color});

  @override
  Widget build(BuildContext context) => Row(mainAxisSize: MainAxisSize.min, children: [
    Icon(icon, size: 12, color: color),
    const Gap(4),
    Text(label, style: TextStyle(color: color, fontSize: 11, fontWeight: FontWeight.w600)),
  ]);
}

// ── My Rank Card ──────────────────────────────────────────────────────────────

class _MyRankCard extends StatelessWidget {
  final Map entry;
  const _MyRankCard({required this.entry});

  @override
  Widget build(BuildContext context) {
    final state   = entry['state']?.toString() ?? '—';
    final rank    = entry['rank'] as int? ?? 0;
    final points  = entry['total_points'] as int? ?? 0;
    final prizeK  = entry['prize_kobo'] as int? ?? 0;
    final medal   = rank == 1 ? '🥇' : rank == 2 ? '🥈' : rank == 3 ? '🥉' : null;

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: NexusColors.primaryGlow,
        borderRadius: NexusRadius.lg,
        border: Border.all(color: NexusColors.primary.withValues(alpha: 0.4)),
      ),
      child: Row(children: [
        Text(medal ?? '🏅', style: const TextStyle(fontSize: 24)),
        const Gap(12),
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text('$state — Rank #$rank',
              style: const TextStyle(color: NexusColors.textPrimary,
                  fontSize: 14, fontWeight: FontWeight.w700)),
          Text('${_fmtPts(points)} pts${prizeK > 0 ? " · ${_fmtKobo(prizeK)} prize" : ""}',
              style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
        ])),
        const Icon(Icons.emoji_events_rounded, color: NexusColors.gold, size: 20),
      ]),
    );
  }
}

// ── How It Works Row ──────────────────────────────────────────────────────────

class _HowItWorksRow extends StatelessWidget {
  const _HowItWorksRow();

  static const _steps = [
    (icon: Icons.bolt_rounded,     label: 'Earn Points',  sub: 'Recharge MTN',   color: Color(0xFF5f72f9)),
    (icon: Icons.flag_rounded,     label: 'Set State',    sub: 'In Settings',    color: Color(0xFF10b981)),
    (icon: Icons.emoji_events_rounded, label: 'Win Prize',sub: 'Top 3 states',   color: Color(0xFFf9c74f)),
  ];

  @override
  Widget build(BuildContext context) => Row(
    children: _steps.map((s) => Expanded(child: Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 10),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          borderRadius: NexusRadius.md,
          border: Border.all(color: s.color.withValues(alpha: 0.2)),
        ),
        child: Column(children: [
          Icon(s.icon, color: s.color, size: 18),
          const Gap(4),
          Text(s.label, style: const TextStyle(color: NexusColors.textPrimary,
              fontSize: 11, fontWeight: FontWeight.w600)),
          Text(s.sub, style: const TextStyle(color: NexusColors.textSecondary,
              fontSize: 9)),
        ]),
      ),
    ))).toList(),
  );
}

// ── Leaderboard Row ───────────────────────────────────────────────────────────

class _LeaderboardRow extends StatelessWidget {
  final int rank;
  final String state;
  final int points;
  final int? members;
  final int prizeKobo;
  final int change;
  const _LeaderboardRow({
    required this.rank, required this.state, required this.points,
    this.members, required this.prizeKobo, required this.change,
  });

  static const _medals = ['🥇', '🥈', '🥉'];

  @override
  Widget build(BuildContext context) {
    final isTop3   = rank <= 3;
    final medal    = isTop3 ? _medals[rank - 1] : null;

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: isTop3 ? NexusColors.gold.withValues(alpha: 0.3) : NexusColors.border,
          width: isTop3 ? 1.2 : 1,
        ),
      ),
      child: Row(children: [
        // Rank / Medal
        SizedBox(width: 36, child: Center(
          child: medal != null
              ? Text(medal, style: const TextStyle(fontSize: 20))
              : Text('$rank', style: const TextStyle(
                  color: NexusColors.textSecondary, fontSize: 13, fontWeight: FontWeight.w700)),
        )),
        const Gap(8),
        // State info
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(state, style: const TextStyle(color: NexusColors.textPrimary,
              fontSize: 14, fontWeight: FontWeight.w600)),
          if (members != null)
            Text('${_fmt(members!)} members',
                style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
        ])),
        // Points
        Column(crossAxisAlignment: CrossAxisAlignment.end, children: [
          Text(_fmtPts(points), style: const TextStyle(color: NexusColors.textPrimary,
              fontSize: 13, fontWeight: FontWeight.w700)),
          if (prizeKobo > 0)
            Text(_fmtKobo(prizeKobo),
                style: const TextStyle(color: NexusColors.gold, fontSize: 10, fontWeight: FontWeight.w600)),
          if (change != 0) ...[
            const Gap(2),
            Text(
              change > 0 ? '▲$change' : '▼${change.abs()}',
              style: TextStyle(
                color: change > 0 ? NexusColors.green : NexusColors.red,
                fontSize: 10, fontWeight: FontWeight.w700),
            ),
          ],
        ]),
      ]),
    );
  }

  String _fmt(int v) => v >= 1000 ? '${(v / 1000).toStringAsFixed(1)}k' : '$v';
}

// ── No Active War ─────────────────────────────────────────────────────────────

class _NoActiveWarView extends StatefulWidget {
  @override
  State<_NoActiveWarView> createState() => _NoActiveWarViewState();
}

class _NoActiveWarViewState extends State<_NoActiveWarView>
    with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;
  late Animation<double> _scale;
  late Animation<double> _rotate;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(duration: const Duration(seconds: 3), vsync: this)
      ..repeat(reverse: true);
    _scale  = Tween(begin: 1.0, end: 1.05).animate(
        CurvedAnimation(parent: _ctrl, curve: Curves.easeInOut));
    _rotate = Tween(begin: -0.05, end: 0.05).animate(
        CurvedAnimation(parent: _ctrl, curve: Curves.easeInOut));
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) => ListView(
    padding: const EdgeInsets.fromLTRB(24, 40, 24, 100),
    children: [
      // Animated globe
      Center(child: AnimatedBuilder(
        animation: _ctrl,
        builder: (_, __) => Transform.scale(
          scale: _scale.value,
          child: Transform.rotate(
            angle: _rotate.value,
            child: const Text('🌍', style: TextStyle(fontSize: 72)),
          ),
        ),
      )),
      const Gap(20),

      // Heading
      const Text('No Active War Yet',
          textAlign: TextAlign.center,
          style: TextStyle(color: NexusColors.textPrimary,
              fontSize: 22, fontWeight: FontWeight.w900)),
      const Gap(8),
      const Text(
        'Regional Wars kick off monthly. Keep recharging and building your points — your state will need you when the battle begins!',
        textAlign: TextAlign.center,
        style: TextStyle(color: NexusColors.textSecondary, fontSize: 13, height: 1.5),
      ),
      const Gap(24),

      // Watch out card
      Container(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          borderRadius: NexusRadius.lg,
          border: Border.all(color: NexusColors.gold.withValues(alpha: 0.2)),
        ),
        child: const Column(children: [
          Text('WATCH OUT FOR THE NEXT EVENT',
              style: TextStyle(color: NexusColors.gold, fontSize: 10,
                  fontWeight: FontWeight.w900, letterSpacing: 1)),
          Gap(4),
          Text('Admins announce new wars each month',
              style: TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
        ]),
      ),
      const Gap(24),

      // Step cards
      Row(children: [
        _StepCard(
          icon: Icons.bolt_rounded, label: 'Earn Points',
          sub: 'Recharge to earn', color: NexusColors.primary),
        const Gap(8),
        _StepCard(
          icon: Icons.flag_rounded, label: 'Set State',
          sub: 'In Settings', color: NexusColors.green,
          onTap: () => context.push('/settings')),
        const Gap(8),
        _StepCard(
          icon: Icons.emoji_events_rounded, label: 'Win Prizes',
          sub: 'When war starts', color: NexusColors.gold),
      ]),
      const Gap(24),

      // CTA to settings to set state
      OutlinedButton.icon(
        onPressed: () => context.push('/settings'),
        icon: const Icon(Icons.flag_rounded, size: 16),
        label: const Text('Set Your State Now'),
        style: OutlinedButton.styleFrom(
          foregroundColor: NexusColors.primary,
          side: const BorderSide(color: NexusColors.primary),
          padding: const EdgeInsets.symmetric(vertical: 14),
          minimumSize: const Size(double.infinity, 0),
          shape: RoundedRectangleBorder(borderRadius: NexusRadius.md),
        ),
      ),
    ],
  );
}

class _StepCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String sub;
  final Color color;
  final VoidCallback? onTap;
  const _StepCard({required this.icon, required this.label, required this.sub,
    required this.color, this.onTap});

  @override
  Widget build(BuildContext context) => Expanded(child: GestureDetector(
    onTap: onTap,
    child: Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 12),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: NexusRadius.md,
        border: Border.all(color: color.withValues(alpha: 0.2)),
      ),
      child: Column(children: [
        Icon(icon, color: color, size: 22),
        const Gap(6),
        Text(label, style: const TextStyle(color: NexusColors.textPrimary,
            fontSize: 11, fontWeight: FontWeight.w700), textAlign: TextAlign.center),
        Text(sub, style: const TextStyle(color: NexusColors.textSecondary,
            fontSize: 9), textAlign: TextAlign.center),
      ]),
    ),
  ));
}

// ── Loading / Error ───────────────────────────────────────────────────────────

class _LoadingView extends StatelessWidget {
  @override
  Widget build(BuildContext context) => ListView(
    padding: const EdgeInsets.fromLTRB(20, 8, 20, 100),
    children: [
      NexusShimmer(width: double.infinity, height: 140, radius: NexusRadius.xl),
      const Gap(16),
      ...List.generate(6, (_) => Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: NexusShimmer(width: double.infinity, height: 64, radius: NexusRadius.md),
      )),
    ],
  );
}

class _ErrorView extends StatelessWidget {
  final VoidCallback onRetry;
  const _ErrorView({required this.onRetry});

  @override
  Widget build(BuildContext context) => Center(child: Padding(
    padding: const EdgeInsets.all(32),
    child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
      const Icon(Icons.wifi_off_rounded, size: 56, color: NexusColors.textSecondary),
      const Gap(16),
      const Text('Could not load leaderboard',
          style: TextStyle(color: NexusColors.textSecondary, fontSize: 15),
          textAlign: TextAlign.center),
      const Gap(16),
      FilledButton.icon(
        onPressed: onRetry,
        icon: const Icon(Icons.refresh_rounded, size: 16),
        label: const Text('Retry'),
        style: FilledButton.styleFrom(backgroundColor: NexusColors.primary),
      ),
    ]),
  ));
}
