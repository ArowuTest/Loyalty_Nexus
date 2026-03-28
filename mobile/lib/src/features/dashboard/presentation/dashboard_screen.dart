import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

// ── Providers ──────────────────────────────────────────────────────────────────

final walletProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getWallet();
});

final profileProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getProfile();
});

final passportProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getPassport();
});

final warsLeaderboardProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  return ref.read(warsApiProvider).getLeaderboard();
});

final transactionsProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  return ref.read(userApiProvider).getTransactions();
});

// ── Tier helpers ───────────────────────────────────────────────────────────────

const _tierThresholds = {'BRONZE': 0, 'SILVER': 2000, 'GOLD': 10000, 'PLATINUM': 50000};
const _tierNext = {'BRONZE': 'SILVER', 'SILVER': 'GOLD', 'GOLD': 'PLATINUM'};

const _tierGradients = {
  'BRONZE':   [Color(0xFFb45309), Color(0xFFd97706)],
  'SILVER':   [Color(0xFF64748b), Color(0xFF94a3b8)],
  'GOLD':     [Color(0xFF92400e), Color(0xFFf59e0b)],
  'PLATINUM': [Color(0xFF5b21b6), Color(0xFF7c3aed)],
};

List<Color> tierGradient(String tier) =>
    _tierGradients[tier] ?? _tierGradients['BRONZE']!;

double tierProgress(int lifePoints, String tier) {
  final min   = _tierThresholds[tier] ?? 0;
  final next  = _tierNext[tier];
  if (next == null) return 1.0;
  final nextMin = _tierThresholds[next] ?? 1;
  if (nextMin == min) return 1.0;
  return ((lifePoints - min) / (nextMin - min)).clamp(0.0, 1.0);
}

// ── Dashboard screen ───────────────────────────────────────────────────────────

class DashboardScreen extends ConsumerWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final walletAsync  = ref.watch(walletProvider);
    final profileAsync = ref.watch(profileProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      body: RefreshIndicator(
        color: NexusColors.primary,
        onRefresh: () async {
          ref.invalidate(walletProvider);
          ref.invalidate(profileProvider);
          ref.invalidate(passportProvider);
          ref.invalidate(warsLeaderboardProvider);
          ref.invalidate(transactionsProvider);
        },
        child: CustomScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          slivers: [
            // App bar
            SliverAppBar(
              backgroundColor: NexusColors.background,
              pinned: true,
              elevation: 0,
              title: Row(children: [
                const Text('⚡', style: TextStyle(fontSize: 20)),
                const SizedBox(width: 8),
                Text(
                  profileAsync.valueOrNull?['full_name']?.toString().split(' ').first
                      ?? 'Loyalty Nexus',
                  style: const TextStyle(color: NexusColors.textPrimary, fontWeight: FontWeight.w700),
                ),
              ]),
              actions: [
                IconButton(
                  icon: const Icon(Icons.notifications_outlined),
                  color: NexusColors.textSecondary,
                  onPressed: () => context.push('/notifications'),
                ),
                IconButton(
                  icon: const Icon(Icons.settings_outlined),
                  color: NexusColors.textSecondary,
                  onPressed: () => context.push('/settings'),
                ),
              ],
            ),

            SliverPadding(
              padding: const EdgeInsets.fromLTRB(20, 4, 20, 100),
              sliver: SliverList(delegate: SliverChildListDelegate([

                // ── Wallet hero card ─────────────────────────────────────────
                _WalletHeroCard(
                  walletAsync: walletAsync,
                  profileAsync: profileAsync,
                ),
                const SizedBox(height: 16),

                // ── Quick actions ────────────────────────────────────────────
                _QuickActionsGrid(),
                const SizedBox(height: 20),

                // ── Passport mini-card ───────────────────────────────────────
                _PassportMiniCard(),
                const SizedBox(height: 16),

                // ── Regional Wars mini ───────────────────────────────────────
                _WarsMiniCard(),
                const SizedBox(height: 20),

                // ── Recent transactions ──────────────────────────────────────
                _RecentTransactions(),
              ])),
            ),
          ],
        ),
      ),
    );
  }
}

// ── Wallet hero card ───────────────────────────────────────────────────────────

class _WalletHeroCard extends StatelessWidget {
  final AsyncValue<Map<String, dynamic>> walletAsync;
  final AsyncValue<Map<String, dynamic>> profileAsync;
  const _WalletHeroCard({required this.walletAsync, required this.profileAsync});

  @override
  Widget build(BuildContext context) {
    final wallet   = walletAsync.valueOrNull;
    final profile  = profileAsync.valueOrNull;
    final tier     = profile?['tier']?.toString() ?? 'BRONZE';
    final pts      = wallet?['pulse_points'] as int? ?? 0;
    final life     = wallet?['lifetime_points'] as int? ?? 0;
    final spins    = wallet?['spin_credits'] as int? ?? 0;
    final progress = tierProgress(life, tier);
    final nextTier = _tierNext[tier];
    final nextThreshold = nextTier != null ? _tierThresholds[nextTier] ?? 0 : null;
    final colors   = tierGradient(tier);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(22),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: colors, begin: Alignment.topLeft, end: Alignment.bottomRight),
        borderRadius: BorderRadius.circular(24),
        boxShadow: [
          BoxShadow(color: colors[1].withOpacity(0.35), blurRadius: 30, offset: const Offset(0, 8)),
        ],
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        // Tier badge + label
        Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
          Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Pulse Points', style: TextStyle(color: Colors.white60, fontSize: 12, letterSpacing: 0.8)),
            const SizedBox(height: 6),
            walletAsync.isLoading
                ? const SizedBox(width: 120, height: 36, child: _LoadingPlaceholder())
                : Text(_formatPts(pts),
                    style: const TextStyle(color: Colors.white, fontSize: 36,
                      fontWeight: FontWeight.w800, letterSpacing: -1)),
          ]),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.2),
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: Colors.white.withOpacity(0.3)),
            ),
            child: Text(tier, style: const TextStyle(
              color: Colors.white, fontSize: 12, fontWeight: FontWeight.w800, letterSpacing: 1)),
          ),
        ]),

        const SizedBox(height: 18),

        // Sub stats
        Row(children: [
          Expanded(child: _HeroStat(label: 'Spin Credits', value: '$spins', icon: '🎡')),
          Container(width: 1, height: 36, color: Colors.white.withOpacity(0.2)),
          Expanded(child: _HeroStat(label: 'Lifetime', value: _formatPts(life), icon: '⭐')),
        ]),

        // Tier progress (hide if PLATINUM)
        if (nextTier != null) ...[
          const SizedBox(height: 18),
          Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
            Text('Progress to $nextTier',
              style: const TextStyle(color: Colors.white60, fontSize: 11)),
            if (nextThreshold != null)
              Text('${_formatPts(nextThreshold - life)} to go',
                style: const TextStyle(color: Colors.white60, fontSize: 11)),
          ]),
          const SizedBox(height: 6),
          ClipRRect(
            borderRadius: BorderRadius.circular(4),
            child: LinearProgressIndicator(
              value: progress,
              minHeight: 5,
              backgroundColor: Colors.white.withOpacity(0.15),
              valueColor: const AlwaysStoppedAnimation(Colors.white),
            ),
          ),
        ] else ...[
          const SizedBox(height: 14),
          const Row(children: [
            Icon(Icons.diamond_rounded, color: Colors.white70, size: 14),
            SizedBox(width: 6),
            Text('Platinum — Maximum Tier Reached 👑',
              style: TextStyle(color: Colors.white70, fontSize: 12)),
          ]),
        ],
      ]),
    );
  }

  String _formatPts(int v) {
    if (v >= 1000) return '${(v / 1000).toStringAsFixed(1)}k';
    return v.toString();
  }
}

class _HeroStat extends StatelessWidget {
  final String label;
  final String value;
  final String icon;
  const _HeroStat({required this.label, required this.value, required this.icon});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 12),
    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text('$icon $label', style: const TextStyle(color: Colors.white54, fontSize: 11)),
      const SizedBox(height: 3),
      Text(value, style: const TextStyle(color: Colors.white, fontSize: 17, fontWeight: FontWeight.w700)),
    ]),
  );
}

// ── Quick actions ─────────────────────────────────────────────────────────────

class _QuickActionsGrid extends StatelessWidget {
  const _QuickActionsGrid();
  @override
  Widget build(BuildContext context) {
    final actions = [
      _Action('🎡', 'Spin & Win', 'Use spin credits', '/spin',     NexusColors.primary),
      _Action('🧠', 'AI Studio',  '17 free tools',   '/studio',   const Color(0xFF8B5CF6)),
      _Action('🌍', 'Regional Wars','State rank',     '/wars',     NexusColors.green),
      _Action('🏅', 'My Passport', 'Tier & badges',  '/passport', NexusColors.gold),
    ];
    return GridView.count(
      crossAxisCount: 2,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      crossAxisSpacing: 12, mainAxisSpacing: 12,
      childAspectRatio: 1.75,
      children: actions.map((a) => _ActionCard(action: a)).toList(),
    );
  }
}

class _Action {
  final String emoji, label, sub, route;
  final Color color;
  const _Action(this.emoji, this.label, this.sub, this.route, this.color);
}

class _ActionCard extends StatelessWidget {
  final _Action action;
  const _ActionCard({required this.action});
  @override
  Widget build(BuildContext context) => GestureDetector(
    onTap: () => context.go(action.route),
    child: Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: NexusColors.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Container(
          width: 36, height: 36,
          decoration: BoxDecoration(
            color: action.color.withOpacity(0.12),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Center(child: Text(action.emoji, style: const TextStyle(fontSize: 18))),
        ),
        const Spacer(),
        Text(action.label,
          style: const TextStyle(color: NexusColors.textPrimary, fontSize: 13, fontWeight: FontWeight.w600)),
        const SizedBox(height: 2),
        Text(action.sub,
          style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
      ]),
    ),
  );
}

// ── Passport mini-card ─────────────────────────────────────────────────────────

class _PassportMiniCard extends ConsumerWidget {
  const _PassportMiniCard();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final passportAsync = ref.watch(passportProvider);

    return GestureDetector(
      onTap: () => context.push('/passport'),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(color: NexusColors.border),
        ),
        child: passportAsync.when(
          loading: () => const _LoadingRow(label: 'Loading Passport…'),
          error: (_, __) => _PassportEmpty(),
          data: (passport) {
            final tier      = passport['tier']?.toString() ?? 'BRONZE';
            final badges    = (passport['badges'] as List?)?.length ?? 0;
            final streak    = passport['current_streak'] as int? ?? 0;
            final colors    = tierGradient(tier);
            return Row(children: [
              Container(
                width: 44, height: 44,
                decoration: BoxDecoration(
                  gradient: LinearGradient(colors: colors),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Center(child: Text(_tierEmoji(tier),
                  style: const TextStyle(fontSize: 22))),
              ),
              const SizedBox(width: 12),
              Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('Digital Passport · $tier',
                  style: const TextStyle(color: NexusColors.textPrimary,
                    fontSize: 14, fontWeight: FontWeight.w600)),
                const SizedBox(height: 3),
                Row(children: [
                  Text('🏅 $badges badge${badges != 1 ? 's' : ''}',
                    style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
                  const SizedBox(width: 10),
                  Text('🔥 $streak day streak',
                    style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
                ]),
              ])),
              const Icon(Icons.chevron_right_rounded, color: NexusColors.textSecondary),
            ]);
          },
        ),
      ),
    );
  }

  String _tierEmoji(String t) {
    switch (t) {
      case 'SILVER':   return '⭐';
      case 'GOLD':     return '🏆';
      case 'PLATINUM': return '💎';
      default:         return '🛡️';
    }
  }
}

class _PassportEmpty extends StatelessWidget {
  @override
  Widget build(BuildContext context) => const Row(children: [
    Text('🛡️', style: TextStyle(fontSize: 28)),
    SizedBox(width: 12),
    Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text('Digital Passport', style: TextStyle(color: NexusColors.textPrimary,
        fontSize: 14, fontWeight: FontWeight.w600)),
      SizedBox(height: 3),
      Text('Tap to view your loyalty passport',
        style: TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
    ])),
    Icon(Icons.chevron_right_rounded, color: NexusColors.textSecondary),
  ]);
}

// ── Wars mini-card ─────────────────────────────────────────────────────────────

class _WarsMiniCard extends ConsumerWidget {
  const _WarsMiniCard();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final leaderboardAsync = ref.watch(warsLeaderboardProvider);

    return GestureDetector(
      onTap: () => context.go('/wars'),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(color: NexusColors.border),
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
            Text('🌍 Regional Wars', style: TextStyle(color: NexusColors.textPrimary,
              fontSize: 14, fontWeight: FontWeight.w600)),
            Text('View all →', style: TextStyle(color: NexusColors.primary, fontSize: 12)),
          ]),
          const SizedBox(height: 12),
          leaderboardAsync.when(
            loading: () => const _LoadingRow(label: 'Loading leaderboard…'),
            error: (_, __) => const Text('Could not load leaderboard',
              style: TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
            data: (lb) {
              if (lb.isEmpty) return const Text('No leaderboard data yet.',
                style: TextStyle(color: NexusColors.textSecondary, fontSize: 12));
              final top3 = lb.take(3).toList();
              return Column(children: List.generate(top3.length, (i) {
                final row = top3[i] as Map;
                final medals = ['🥇', '🥈', '🥉'];
                return Padding(
                  padding: const EdgeInsets.only(bottom: 6),
                  child: Row(children: [
                    Text(medals[i], style: const TextStyle(fontSize: 16)),
                    const SizedBox(width: 8),
                    Expanded(child: Text(row['state']?.toString() ?? '—',
                      style: const TextStyle(color: NexusColors.textPrimary,
                        fontSize: 13, fontWeight: FontWeight.w500))),
                    Text(_fmtPts(row['points'] as int? ?? row['total_points'] as int? ?? 0),
                      style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
                  ]),
                );
              }));
            },
          ),
        ]),
      ),
    );
  }

  String _fmtPts(int v) => v >= 1000 ? '${(v / 1000).toStringAsFixed(1)}k pts' : '$v pts';
}

// ── Recent transactions ────────────────────────────────────────────────────────

class _RecentTransactions extends ConsumerWidget {
  const _RecentTransactions();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final txAsync = ref.watch(transactionsProvider);
    return txAsync.when(
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
      data: (txns) {
        if (txns.isEmpty) return const SizedBox.shrink();
        final recent = txns.take(4).toList();
        return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Text('Recent Activity',
            style: TextStyle(color: NexusColors.textSecondary, fontSize: 12,
              fontWeight: FontWeight.w700, letterSpacing: 0.8)),
          const SizedBox(height: 10),
          ...recent.map((t) => _TxRow(tx: t as Map)),
          const SizedBox(height: 6),
          Center(child: TextButton(
            onPressed: () => context.push('/profile'),
            style: TextButton.styleFrom(foregroundColor: NexusColors.primary),
            child: const Text('View all transactions →', style: TextStyle(fontSize: 12)),
          )),
        ]);
      },
    );
  }
}

class _TxRow extends StatelessWidget {
  final Map tx;
  const _TxRow({required this.tx});
  @override
  Widget build(BuildContext context) {
    final type   = tx['transaction_type']?.toString() ?? '';
    final pts    = tx['points'] as int? ?? 0;
    final isEarn = pts > 0;
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: NexusColors.border),
      ),
      child: Row(children: [
        Container(
          width: 36, height: 36,
          decoration: BoxDecoration(
            color: (isEarn ? NexusColors.green : NexusColors.red).withOpacity(0.1),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Center(child: Text(_txEmoji(type), style: const TextStyle(fontSize: 16))),
        ),
        const SizedBox(width: 12),
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(_txLabel(type), style: const TextStyle(color: NexusColors.textPrimary,
            fontSize: 13, fontWeight: FontWeight.w500)),
          if (tx['description'] != null)
            Text(tx['description'].toString(),
              style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11),
              maxLines: 1, overflow: TextOverflow.ellipsis),
        ])),
        Text('${isEarn ? '+' : ''}$pts pts',
          style: TextStyle(
            color: isEarn ? NexusColors.green : NexusColors.red,
            fontSize: 14, fontWeight: FontWeight.w700)),
      ]),
    );
  }

  String _txEmoji(String t) {
    if (t.contains('recharge')) return '⚡';
    if (t.contains('spin'))     return '🎡';
    if (t.contains('studio'))   return '🧠';
    if (t.contains('referral')) return '👥';
    if (t.contains('bonus'))    return '🎁';
    return '📊';
  }

  String _txLabel(String t) {
    if (t.contains('recharge')) return 'Recharge Earned';
    if (t.contains('spin'))     return 'Spin Used';
    if (t.contains('studio'))   return 'AI Studio';
    if (t.contains('referral')) return 'Referral Bonus';
    if (t.contains('bonus'))    return 'Bonus Award';
    return t.isNotEmpty ? t.replaceAll('_', ' ').toUpperCase() : 'Transaction';
  }
}

// ── Shared helpers ─────────────────────────────────────────────────────────────

class _LoadingPlaceholder extends StatelessWidget {
  const _LoadingPlaceholder();
  @override
  Widget build(BuildContext context) => Container(
    decoration: BoxDecoration(
      color: Colors.white.withOpacity(0.15),
      borderRadius: BorderRadius.circular(8),
    ),
  );
}

class _LoadingRow extends StatelessWidget {
  final String label;
  const _LoadingRow({required this.label});
  @override
  Widget build(BuildContext context) => Row(children: [
    const SizedBox(width: 16, height: 16,
      child: CircularProgressIndicator(strokeWidth: 2, color: NexusColors.primary)),
    const SizedBox(width: 10),
    Text(label, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
  ]);
}
