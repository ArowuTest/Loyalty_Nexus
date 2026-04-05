import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/cache/cache_service.dart';
import '../../../core/theme/nexus_theme.dart';
import '../../passport/presentation/wallet_onboarding_sheet.dart';

// ── Providers ──────────────────────────────────────────────────────────────────

// ── Stale-while-revalidate cache helper ───────────────────────────────────────
// Returns cached data instantly (so UI never shows blank), then fetches fresh
// data and updates the UI automatically via Riverpod's rebuild.

final walletProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final cache = ref.read(cacheServiceProvider);
  final cached = cache.getMap(CacheKeys.wallet, maxAgeMinutes: 5);
  if (cached != null) {
    // schedule background refresh after returning cached data
    Future.microtask(() async {
      try {
        final fresh = await ref.read(userApiProvider).getWallet();
        await cache.put(CacheKeys.wallet, fresh);
      } catch (_) {}
    });
    return cached;
  }
  final data = await ref.read(userApiProvider).getWallet();
  await cache.put(CacheKeys.wallet, data);
  return data;
});

final profileProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final cache = ref.read(cacheServiceProvider);
  final cached = cache.getMap(CacheKeys.profile, maxAgeMinutes: 10);
  if (cached != null) {
    Future.microtask(() async {
      try {
        final fresh = await ref.read(userApiProvider).getProfile();
        await cache.put(CacheKeys.profile, fresh);
      } catch (_) {}
    });
    return cached;
  }
  final data = await ref.read(userApiProvider).getProfile();
  await cache.put(CacheKeys.profile, data);
  return data;
});

final passportProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final cache = ref.read(cacheServiceProvider);
  final cached = cache.getMap(CacheKeys.passport, maxAgeMinutes: 5);
  if (cached != null) {
    Future.microtask(() async {
      try {
        final fresh = await ref.read(userApiProvider).getPassport();
        await cache.put(CacheKeys.passport, fresh);
      } catch (_) {}
    });
    return cached;
  }
  final data = await ref.read(userApiProvider).getPassport();
  await cache.put(CacheKeys.passport, data);
  return data;
});

final warsLeaderboardProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  final cache = ref.read(cacheServiceProvider);
  final cached = cache.getList(CacheKeys.leaderboard, maxAgeMinutes: 3);
  if (cached != null) {
    Future.microtask(() async {
      try {
        final fresh = await ref.read(warsApiProvider).getLeaderboard();
        await cache.putList(CacheKeys.leaderboard, fresh);
      } catch (_) {}
    });
    return cached;
  }
  final data = await ref.read(warsApiProvider).getLeaderboard();
  await cache.putList(CacheKeys.leaderboard, data);
  return data;
});

final transactionsProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  final cache = ref.read(cacheServiceProvider);
  final cached = cache.getList(CacheKeys.transactions, maxAgeMinutes: 2);
  if (cached != null) {
    Future.microtask(() async {
      try {
        final fresh = await ref.read(userApiProvider).getTransactions();
        await cache.putList(CacheKeys.transactions, fresh);
      } catch (_) {}
    });
    return cached;
  }
  final data = await ref.read(userApiProvider).getTransactions();
  await cache.putList(CacheKeys.transactions, data);
  return data;
});

final bonusPulseProvider = FutureProvider.autoDispose<int>((ref) async {
  try {
    final r = await ref.read(userApiProvider).getBonusPulseAwards();
    return (r as Map)['total_bonus'] as int? ?? 0;
  } catch (_) { return 0; }
});

final myWarRankProvider = FutureProvider.autoDispose<Map<String, dynamic>?>((ref) async {
  try {
     final r = await ref.read(warsApiProvider).getMyRank();
    return Map<String, dynamic>.from(r);
  } catch (_) {}
  return null;
});
final _dashUnreadProvider = FutureProvider.autoDispose<int>((ref) async {
  try {
    final r = await ref.read(notificationsApiProvider).list(limit: 1);
    return (r['unread_count'] as int?) ?? 0;
  } catch (_) { return 0; }
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

// ── Dashboard screen ─────────────────────────────────────────────────────────────────────

class DashboardScreen extends ConsumerStatefulWidget {
  const DashboardScreen({super.key});
  @override
  ConsumerState<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends ConsumerState<DashboardScreen> {
  @override
  void initState() {
    super.initState();
    // Show wallet onboarding bottom sheet once, on the first app launch after login.
    // Uses SharedPreferences to ensure it is never shown again after dismissal.
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      if (!mounted) return;
      final show = await shouldShowWalletOnboarding();
      if (show && mounted) {
        await showWalletOnboardingSheet(context);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
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
          ref.invalidate(bonusPulseProvider);
          ref.invalidate(myWarRankProvider);
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
                // Bell with live unread badge
                Consumer(builder: (_, ref, __) {
                  final unread = ref.watch(_dashUnreadProvider).valueOrNull ?? 0;
                  return Stack(
                    clipBehavior: Clip.none,
                    children: [
                      IconButton(
                        icon: const Icon(Icons.notifications_outlined),
                        color: NexusColors.textSecondary,
                        onPressed: () => context.push('/notifications'),
                      ),
                      if (unread > 0)
                        Positioned(
                          right: 6, top: 6,
                          child: Container(
                            width: 16, height: 16,
                            decoration: const BoxDecoration(
                              color: NexusColors.red, shape: BoxShape.circle),
                            child: Center(child: Text(
                              unread > 9 ? '9+' : '$unread',
                              style: const TextStyle(color: Colors.white,
                                  fontSize: 9, fontWeight: FontWeight.w800),
                            )),
                          ),
                        ),
                    ],
                  );
                }),
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

                // ── Passport banner ──────────────────────────────────────────
                _PassportBanner(
                  walletAsync: walletAsync,
                  profileAsync: profileAsync,
                ),
                const SizedBox(height: 16),

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
                const SizedBox(height: 16),

                // ── Draws coming soon teaser ─────────────────────────────────
                _DrawsTeaser(),
                const SizedBox(height: 16),

                // ── Recharge CTA ─────────────────────────────────────────────
                _RechargeCTA(),
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

// ── Passport Banner ────────────────────────────────────────────────────────────

class _PassportBanner extends StatelessWidget {
  final AsyncValue<Map<String, dynamic>> walletAsync;
  final AsyncValue<Map<String, dynamic>> profileAsync;
  const _PassportBanner({required this.walletAsync, required this.profileAsync});

  @override
  Widget build(BuildContext context) {
    final points = walletAsync.valueOrNull?['pulse_points'] as int? ?? 0;
    final streak = profileAsync.valueOrNull?['streak_count'] as int? ?? 0;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFF1c1a2e), Color(0xFF12131f)],
          begin: Alignment.topLeft, end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: const Color(0x33a78bfa)),
        boxShadow: [BoxShadow(color: const Color(0x1A7c3aed), blurRadius: 20)],
      ),
      child: Row(children: [
        Container(
          width: 42, height: 42,
          decoration: BoxDecoration(
            color: const Color(0x1A7c3aed), borderRadius: BorderRadius.circular(12),
            border: Border.all(color: const Color(0x33a78bfa)),
          ),
          child: const Center(child: Text('🛂', style: TextStyle(fontSize: 20))),
        ),
        const SizedBox(width: 12),
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Text('Your Digital Passport is ready',
            style: TextStyle(color: Colors.white, fontWeight: FontWeight.w800, fontSize: 13)),
          const SizedBox(height: 2),
          RichText(text: TextSpan(
            style: const TextStyle(color: Color(0x99ffffff), fontSize: 11),
            children: [
              TextSpan(text: '${_fmtPts(points)} pts'),
              if (streak > 0) ...[
                const TextSpan(text: ' and '),
                TextSpan(text: 'Day $streak streak 🔥',
                  style: const TextStyle(color: Color(0xFFfb923c), fontWeight: FontWeight.w700)),
              ],
              const TextSpan(text: ' — always with you.'),
            ],
          )),
        ])),
        const SizedBox(width: 8),
        GestureDetector(
          onTap: () => context.push('/passport'),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            decoration: BoxDecoration(
              color: const Color(0x1A7c3aed),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: const Color(0x33a78bfa)),
            ),
            child: const Text('View', style: TextStyle(
              color: Color(0xFFa78bfa), fontSize: 11, fontWeight: FontWeight.w700)),
          ),
        ),
      ]),
    );
  }

  String _fmtPts(int v) {
    if (v >= 1000000) return '${(v / 1000000).toStringAsFixed(1)}M';
    if (v >= 1000) return '${(v / 1000).toStringAsFixed(1)}k';
    return '$v';
  }
}

// ── Wallet hero card ───────────────────────────────────────────────────────────

class _WalletHeroCard extends ConsumerWidget {
  final AsyncValue<Map<String, dynamic>> walletAsync;
  final AsyncValue<Map<String, dynamic>> profileAsync;
  const _WalletHeroCard({required this.walletAsync, required this.profileAsync});


  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final bonusAsync = ref.watch(bonusPulseProvider);
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
          BoxShadow(color: colors[1].withValues(alpha: 0.35), blurRadius: 30, offset: const Offset(0, 8)),
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
              color: Colors.white.withValues(alpha: 0.2),
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: Colors.white.withValues(alpha: 0.3)),
            ),
            child: Text(tier, style: const TextStyle(
              color: Colors.white, fontSize: 12, fontWeight: FontWeight.w800, letterSpacing: 1)),
          ),
        ]),

        const SizedBox(height: 18),

        // Sub stats
        Row(children: [
          Expanded(child: _HeroStat(label: 'Spin Credits', value: '$spins', icon: '🎡')),
          Container(width: 1, height: 36, color: Colors.white.withValues(alpha: 0.2)),
          Expanded(child: _HeroStat(label: 'Lifetime', value: _formatPts(life), icon: '⭐')),
        ]),

        // Bonus Awards (if non-zero)
        if ((bonusAsync.valueOrNull ?? 0) > 0) ...[
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: Colors.white.withValues(alpha: 0.15)),
            ),
            child: Row(children: [
              const Text('🎁', style: TextStyle(fontSize: 13)),
              const SizedBox(width: 8),
              const Text('Bonus Awards',
                style: TextStyle(color: Colors.white70, fontSize: 11)),
              const Spacer(),
              Text(_formatPts(bonusAsync.valueOrNull ?? 0),
                style: const TextStyle(color: Colors.white,
                  fontWeight: FontWeight.w800, fontSize: 15)),
              const SizedBox(width: 6),
              GestureDetector(
                onTap: () {},
                child: const Text('History →',
                  style: TextStyle(color: Colors.white54, fontSize: 10)),
              ),
            ]),
          ),
        ],

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
              backgroundColor: Colors.white.withValues(alpha: 0.15),
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

class _QuickActionsGrid extends ConsumerWidget {
  const _QuickActionsGrid();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
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
            color: action.color.withValues(alpha: 0.12),
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
    final lbAsync     = ref.watch(warsLeaderboardProvider);
    final myRankAsync = ref.watch(myWarRankProvider);

    return Container(
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFF0f1a12), Color(0xFF0d0e14)],
          begin: Alignment.topLeft, end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: const Color(0x2E10b981)),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        // Accent top line
        Container(height: 2, decoration: const BoxDecoration(
          gradient: LinearGradient(colors: [Colors.transparent, Color(0x8010b981), Colors.transparent]),
          borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
        )),

        Padding(padding: const EdgeInsets.fromLTRB(16, 14, 16, 16), child: Column(
          crossAxisAlignment: CrossAxisAlignment.start, children: [
          // Header
          Row(children: [
            Container(
              width: 36, height: 36,
              decoration: BoxDecoration(
                color: const Color(0x1A10b981), borderRadius: BorderRadius.circular(10),
                border: Border.all(color: const Color(0x3310b981))),
              child: const Icon(Icons.outlined_flag_rounded, size: 18, color: Color(0xFF34d399))),
            const SizedBox(width: 10),
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              const Text('Regional Wars',
                style: TextStyle(color: Colors.white, fontWeight: FontWeight.w800, fontSize: 14)),
              const Text('₦500K monthly prize pool',
                style: TextStyle(color: Color(0xFF6ee7b7), fontSize: 11)),
            ])),
            GestureDetector(
              onTap: () => context.go('/wars'),
              child: const Row(mainAxisSize: MainAxisSize.min, children: [
                Text('View all', style: TextStyle(color: Color(0xFF34d399),
                  fontSize: 11, fontWeight: FontWeight.w800)),
                SizedBox(width: 3),
                Icon(Icons.arrow_forward_rounded, size: 12, color: Color(0xFF34d399)),
              ]),
            ),
          ]),

          const SizedBox(height: 14),

          // My rank card
          myRankAsync.when(
            loading: () => const SizedBox.shrink(),
            error: (_, __) => const SizedBox.shrink(),
            data: (rank) {
              final ranked = (rank?['ranked'] as bool?) == true;
              final entry  = ranked ? (rank?['entry'] as Map?) : null;
              if (ranked && entry != null) {
                final pts = entry['total_points'] as int? ?? 0;
                return Container(
                  padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
                  margin: const EdgeInsets.only(bottom: 10),
                  decoration: BoxDecoration(
                    color: const Color(0x1510b981),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0x2210b981)),
                  ),
                  child: Row(children: [
                    const Icon(Icons.location_on_rounded, size: 16, color: Color(0xFF34d399)),
                    const SizedBox(width: 8),
                    Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Text(entry['state']?.toString() ?? '',
                        style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 13)),
                      Text(_fmtPts(pts),
                        style: const TextStyle(color: Color(0xFF6ee7b7), fontSize: 11)),
                    ])),
                    Column(crossAxisAlignment: CrossAxisAlignment.end, children: [
                      const Text('Your rank', style: TextStyle(color: Color(0xFF6ee7b7), fontSize: 10)),
                      Text('#${entry['rank']}',
                        style: const TextStyle(color: Color(0xFF34d399),
                          fontSize: 20, fontWeight: FontWeight.w900)),
                    ]),
                  ]),
                );
              }
              // Not ranked
              return Container(
                padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
                margin: const EdgeInsets.only(bottom: 10),
                decoration: BoxDecoration(
                  color: const Color(0x0A10b981), borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: const Color(0x1410b981))),
                child: const Row(children: [
                  Icon(Icons.location_on_outlined, size: 14, color: Color(0x7734d399)),
                  SizedBox(width: 8),
                  Expanded(child: Text('Recharge to earn points and join your state\'s battle',
                    style: TextStyle(color: Color(0xFF6b7280), fontSize: 11))),
                ]),
              );
            },
          ),

          // Top 3 leaderboard
          lbAsync.when(
            loading: () => const _LoadingRow(label: 'Loading leaderboard…'),
            error: (_, __) => const SizedBox.shrink(),
            data: (lb) {
              if (lb.isEmpty) return const SizedBox.shrink();
              final medals = ['🥇', '🥈', '🥉'];
              final colors = [Color(0xFFF5A623), Color(0xFFC0C0C0), Color(0xFFCD7F32)];
              return Column(
                children: List.generate(lb.take(3).length, (i) {
                  final row  = lb[i] as Map;
                  final pts  = row['total_points'] as int? ?? 0;
                  final prize = row['prize_kobo'] as int?;
                  final prizeStr = (prize != null && prize > 0)
                    ? '₦${(prize / 100).toStringAsFixed(0)}'
                    : null;
                  return Container(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
                    margin: const EdgeInsets.only(bottom: 5),
                    decoration: BoxDecoration(
                      color: const Color(0x07ffffff),
                      borderRadius: BorderRadius.circular(10)),
                    child: Row(children: [
                      Text(medals[i], style: const TextStyle(fontSize: 15)),
                      const SizedBox(width: 10),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text(row['state']?.toString() ?? '—',
                          style: const TextStyle(color: Colors.white,
                            fontWeight: FontWeight.w700, fontSize: 13)),
                        Text(_fmtPts(pts),
                          style: const TextStyle(color: Color(0x99ffffff),
                            fontSize: 10, fontFamily: 'Courier')),
                      ])),
                      if (prizeStr != null)
                        Text(prizeStr,
                          style: TextStyle(color: colors[i],
                            fontWeight: FontWeight.w800, fontSize: 11)),
                    ]),
                  );
                }),
              );
            },
          ),

          const SizedBox(height: 10),

          // Individual draw info
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: const Color(0x0BF5A623), borderRadius: BorderRadius.circular(10),
              border: Border.all(color: const Color(0x19F5A623))),
            child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
              const Icon(Icons.card_giftcard_rounded, size: 13,
                color: NexusColors.gold),
              const SizedBox(width: 8),
              const Expanded(child: Text(
                'Individual draw: One random member from each top-3 state wins a personal MoMo cash payout at month end.',
                style: TextStyle(color: Color(0xFF9ca3af), fontSize: 10, height: 1.4))),
            ]),
          ),
        ])),
      ]),
    );
  }

  String _fmtPts(int v) => v >= 1000 ? '${(v / 1000).toStringAsFixed(1)}k pts' : '$v pts';
}

// ── Draws Coming Soon Teaser ───────────────────────────────────────────────────

class _DrawsTeaser extends StatelessWidget {
  const _DrawsTeaser();

  @override
  Widget build(BuildContext context) {
    const items = [
      (icon: Icons.access_time_rounded, color: Color(0xFF00D4FF),
       title: 'Daily Draw', body: 'Win prizes daily just for being active.'),
      (icon: Icons.star_rounded, color: Color(0xFF8B5CF6),
       title: 'Weekly Jackpot', body: 'Bigger prizes for top rechargees.'),
    ];
    return Row(children: items.map((item) => Expanded(
      child: Container(
        margin: EdgeInsets.only(left: item.icon == Icons.access_time_rounded ? 0 : 10),
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: NexusColors.surface, borderRadius: BorderRadius.circular(18),
          border: Border.all(color: NexusColors.border),
        ),
        child: Opacity(opacity: 0.75, child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
            Container(
              width: 32, height: 32,
              decoration: BoxDecoration(
                color: item.color.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(10),
                border: Border.all(color: item.color.withValues(alpha: 0.2))),
              child: Icon(item.icon, size: 15, color: item.color)),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                color: item.color.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(20),
                border: Border.all(color: item.color.withValues(alpha: 0.2))),
              child: Text('SOON', style: TextStyle(color: item.color,
                fontSize: 8, fontWeight: FontWeight.w900, letterSpacing: 0.5))),
          ]),
          const SizedBox(height: 10),
          Text(item.title,
            style: const TextStyle(color: NexusColors.textPrimary,
              fontSize: 12, fontWeight: FontWeight.w800)),
          const SizedBox(height: 3),
          Text(item.body, style: const TextStyle(color: NexusColors.textSecondary,
            fontSize: 10, height: 1.4)),
        ])),
      ),
    )).toList());
  }
}

// ── Recharge CTA ───────────────────────────────────────────────────────────────

class _RechargeCTA extends StatelessWidget {
  const _RechargeCTA();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 14),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0x14F5A623), Color(0x08F5A623)],
          begin: Alignment.topLeft, end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: const Color(0x2EF5A623)),
      ),
      child: Row(children: [
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Text('Recharge to earn more ⚡',
            style: TextStyle(color: Colors.white, fontWeight: FontWeight.w800, fontSize: 13)),
          const SizedBox(height: 3),
          const Text('₦250 = 1 Pulse Point · ₦1,000+ = free spin',
            style: TextStyle(color: Color(0xFF9ca3af), fontSize: 11)),
        ])),
        const SizedBox(width: 12),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            gradient: const LinearGradient(
              colors: [NexusColors.gold, Color(0xFFd97706)]),
            borderRadius: BorderRadius.circular(14),
          ),
          child: const Row(mainAxisSize: MainAxisSize.min, children: [
            Icon(Icons.bolt_rounded, size: 14, color: Colors.white),
            SizedBox(width: 4),
            Text('Recharge', style: TextStyle(color: Colors.white,
              fontWeight: FontWeight.w800, fontSize: 12)),
          ]),
        ),
      ]),
    );
  }
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
            color: (isEarn ? NexusColors.green : NexusColors.red).withValues(alpha: 0.1),
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
    if (t.contains('bonus'))    return '🎁';
    return '📊';
  }

  String _txLabel(String t) {
    if (t.contains('recharge')) return 'Recharge Earned';
    if (t.contains('spin'))     return 'Spin Used';
    if (t.contains('studio'))   return 'AI Studio';
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
      color: Colors.white.withValues(alpha: 0.15),
      borderRadius: BorderRadius.circular(8),
    ),
  );
}

class _LoadingRow extends StatelessWidget {
  final String label;
  const _LoadingRow({required this.label});
  @override
  Widget build(BuildContext context) => Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
    const NexusShimmer(width: double.infinity, height: 80, radius: NexusRadius.md),
    const SizedBox(height: 10),
    NexusShimmer(width: 200, height: 14, radius: NexusRadius.sm),
  ]);
}
