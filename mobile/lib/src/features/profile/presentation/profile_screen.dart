import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Providers ────────────────────────────────────────────────────────────────

final _profileProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getProfile();
});

final _walletProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getWallet();
});

// ─── Helpers ─────────────────────────────────────────────────────────────────

String _fmtPoints(int n) {
  if (n >= 1000000) return '${(n / 1000000).toStringAsFixed(1)}M';
  if (n >= 1000)    return '${(n / 1000).toStringAsFixed(1)}K';
  return '$n';
}

String _fmtDate(String iso) {
  try {
    final d = DateTime.parse(iso);
    const m = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
    return '${m[d.month - 1]} ${d.year}';
  } catch (_) { return ''; }
}

// ─── Screen ───────────────────────────────────────────────────────────────────

class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(_profileProvider);
    final walletAsync  = ref.watch(_walletProvider);
    final phone        = ref.watch(authStateProvider).phoneNumber ?? '—';

    return Scaffold(
      backgroundColor: NexusColors.background,
      body: RefreshIndicator(
        color: NexusColors.primary,
        onRefresh: () async {
          ref.invalidate(_profileProvider);
          ref.invalidate(_walletProvider);
        },
        child: CustomScrollView(
          slivers: [
            // ── SliverAppBar ──────────────────────────────────────────────
            SliverAppBar(
              backgroundColor: NexusColors.background,
              expandedHeight: 220,
              pinned: true,
              surfaceTintColor: Colors.transparent,
              flexibleSpace: FlexibleSpaceBar(
                background: profileAsync.when(
                  loading: () => _HeroShimmer(),
                  error:   (_, __) => const SizedBox.shrink(),
                  data:    (p) => _HeroBanner(profile: p, phone: phone),
                ),
              ),
              actions: [
                IconButton(
                  icon: const Icon(Icons.settings_outlined),
                  tooltip: 'Settings',
                  onPressed: () => context.push('/settings'),
                ),
              ],
            ),

            SliverToBoxAdapter(child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 32),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [

                // ── Wallet stats row ──────────────────────────────────────
                walletAsync.when(
                  loading: () => _WalletShimmer(),
                  error:   (_, __) => const SizedBox.shrink(),
                  data: (w) => _WalletRow(wallet: w,
                      streak: (w['streak_count'] as int?) ?? 0),
                ),
                const SizedBox(height: 24),

                // ── Quick actions ─────────────────────────────────────────
                NexusSectionLabel('Quick Access'),
                _ActionGrid(ref: ref),
                const SizedBox(height: 24),

                // ── Membership card ───────────────────────────────────────
                profileAsync.when(
                  loading: () => const NexusShimmer(width: double.infinity, height: 100, radius: NexusRadius.md),
                  error:   (_, __) => const SizedBox.shrink(),
                  data:    (p) => _MemberCard(profile: p, phone: phone),
                ),
                const SizedBox(height: 24),

                // ── Earning guide ──────────────────────────────────────────
                NexusSectionLabel('How You Earn'),
                _EarningGuide(),
                const SizedBox(height: 24),

                // ── USSD hint ──────────────────────────────────────────────
                _UssdBanner(),
              ]),
            )),
          ],
        ),
      ),
    );
  }
}

// ─── Hero Banner ──────────────────────────────────────────────────────────────

class _HeroBanner extends StatelessWidget {
  final Map profile;
  final String phone;
  const _HeroBanner({required this.profile, required this.phone});

  @override
  Widget build(BuildContext context) {
    final tier       = (profile['tier'] as String? ?? 'BRONZE').toUpperCase();
    final name       = profile['full_name'] as String? ?? phone;
    final joinDate   = profile['created_at'] as String? ?? '';
    final tierColor  = NexusColors.forTier(tier);
    final tierEmoji  = NexusColors.emojiForTier(tier);

    return Container(
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [
            NexusColors.primaryDark,
            tierColor.withValues(alpha: 0.7),
            NexusColors.background,
          ],
          stops: const [0, 0.5, 1],
        ),
      ),
      padding: const EdgeInsets.fromLTRB(20, 72, 20, 20),
      child: Row(crossAxisAlignment: CrossAxisAlignment.end, children: [
        // Avatar
        Container(
          width: 68, height: 68,
          decoration: BoxDecoration(
            gradient: NexusColors.gradientBrand,
            shape: BoxShape.circle,
            border: Border.all(color: tierColor, width: 3),
            boxShadow: [BoxShadow(color: tierColor.withValues(alpha: 0.4), blurRadius: 16)],
          ),
          child: Center(child: Text(
            phone.length >= 4 ? phone.substring(phone.length - 4) : '****',
            style: const TextStyle(color: Colors.white, fontSize: 13,
                fontWeight: FontWeight.w900, letterSpacing: 1),
          )),
        ),
        const SizedBox(width: 14),
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(name,
              style: const TextStyle(color: Colors.white, fontSize: 16,
                  fontWeight: FontWeight.w800),
              maxLines: 1, overflow: TextOverflow.ellipsis),
          const SizedBox(height: 4),
          Row(children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
              decoration: BoxDecoration(
                color: tierColor.withValues(alpha: 0.2),
                borderRadius: NexusRadius.pill,
                border: Border.all(color: tierColor.withValues(alpha: 0.5)),
              ),
              child: Text('$tierEmoji  $tier',
                  style: TextStyle(color: tierColor, fontSize: 11,
                      fontWeight: FontWeight.w800)),
            ),
            const SizedBox(width: 8),
            Text('Since ${_fmtDate(joinDate)}',
                style: const TextStyle(color: Colors.white38, fontSize: 11)),
          ]),
        ])),
      ]),
    );
  }
}

class _HeroShimmer extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Container(
    color: NexusColors.surface,
    padding: const EdgeInsets.fromLTRB(20, 72, 20, 20),
    child: Row(children: [
      const NexusShimmer(width: 68, height: 68,
          radius: BorderRadius.all(Radius.circular(34))),
      const SizedBox(width: 14),
      Column(crossAxisAlignment: CrossAxisAlignment.start, children: const [
        NexusShimmer(width: 140, height: 16),
        SizedBox(height: 8),
        NexusShimmer(width: 80, height: 12),
      ]),
    ]),
  );
}

// ─── Wallet row ───────────────────────────────────────────────────────────────

class _WalletRow extends StatelessWidget {
  final Map wallet;
  final int streak;
  const _WalletRow({required this.wallet, required this.streak});

  @override
  Widget build(BuildContext context) {
    final pts     = (wallet['pulse_points'] as num?)?.toInt() ?? 0;
    final credits = (wallet['spin_credits'] as num?)?.toInt() ?? 0;

    return Row(children: [
      _StatCard(emoji: '💎', label: 'Pulse Points',  value: _fmtPoints(pts),     unit: 'pts'),
      const SizedBox(width: 10),
      _StatCard(emoji: '🎰', label: 'Spin Credits',  value: '$credits',          unit: 'left'),
      const SizedBox(width: 10),
      _StatCard(emoji: '🔥', label: 'Day Streak',    value: '$streak',
          unit: streak == 1 ? 'day' : 'days'),
    ]);
  }
}

class _WalletShimmer extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Row(children: const [
    Expanded(child: NexusShimmer(width: double.infinity, height: 80)),
    SizedBox(width: 10),
    Expanded(child: NexusShimmer(width: double.infinity, height: 80)),
    SizedBox(width: 10),
    Expanded(child: NexusShimmer(width: double.infinity, height: 80)),
  ]);
}

class _StatCard extends StatelessWidget {
  final String emoji, label, value, unit;
  const _StatCard({required this.emoji, required this.label,
      required this.value, required this.unit});
  @override
  Widget build(BuildContext context) => Expanded(child: NexusCard(
    padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 10),
    child: Column(children: [
      Text(emoji, style: const TextStyle(fontSize: 20)),
      const SizedBox(height: 4),
      Text(value, style: const TextStyle(color: NexusColors.textPrimary,
          fontSize: 18, fontWeight: FontWeight.w900)),
      Text(unit, style: NexusText.caption),
      const SizedBox(height: 2),
      Text(label, style: NexusText.label, textAlign: TextAlign.center),
    ]),
  ));
}

// ─── Action Grid ──────────────────────────────────────────────────────────────

class _ActionGrid extends StatelessWidget {
  final WidgetRef ref;
  const _ActionGrid({required this.ref});

  static const _actions = [
    (icon: Icons.qr_code_rounded,        label: 'Digital\nPassport',  route: '/passport',      color: NexusColors.primary),
    (icon: Icons.notifications_outlined, label: 'Notifications',      route: '/notifications', color: NexusColors.gold),
    (icon: Icons.emoji_events_rounded,   label: 'My Prizes',          route: '/prizes',        color: NexusColors.green),
    (icon: Icons.calendar_today_rounded, label: 'Draws',              route: '/draws',         color: NexusColors.purple),
    (icon: Icons.card_giftcard_rounded,  label: 'Bonus Awards',       route: '/pulse-awards',  color: NexusColors.cyan),
    (icon: Icons.settings_outlined,      label: 'Settings',           route: '/settings',      color: NexusColors.textSecondary),
  ];

  @override
  Widget build(BuildContext context) => GridView.count(
    crossAxisCount: 3,
    shrinkWrap:     true,
    physics: const NeverScrollableScrollPhysics(),
    crossAxisSpacing: 10,
    mainAxisSpacing: 10,
    childAspectRatio: 1.05,
    children: _actions.map((a) => _ActionTile(
      icon: a.icon, label: a.label,
      color: a.color,
      onTap: () => context.push(a.route),
    )).toList(),
  );
}

class _ActionTile extends StatelessWidget {
  final IconData icon;
  final String label;
  final Color color;
  final VoidCallback onTap;
  const _ActionTile({required this.icon, required this.label,
      required this.color, required this.onTap});
  @override
  Widget build(BuildContext context) => GestureDetector(
    onTap: onTap,
    child: NexusCard(
      padding: const EdgeInsets.all(12),
      child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
        Container(
          width: 38, height: 38,
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.12),
            borderRadius: NexusRadius.sm,
          ),
          child: Icon(icon, color: color, size: 20),
        ),
        const SizedBox(height: 6),
        Text(label, style: const TextStyle(color: NexusColors.textPrimary,
            fontSize: 10, fontWeight: FontWeight.w700, height: 1.3),
            textAlign: TextAlign.center, maxLines: 2),
      ]),
    ),
  );
}

// ─── Membership card ──────────────────────────────────────────────────────────

class _MemberCard extends StatelessWidget {
  final Map profile;
  final String phone;
  const _MemberCard({required this.profile, required this.phone});

  @override
  Widget build(BuildContext context) {
    final tier        = (profile['tier'] as String? ?? 'BRONZE').toUpperCase();
    final momoVerified= profile['momo_verified'] as bool? ?? false;
    final momoNumber  = profile['momo_number'] as String? ?? '';
    final state       = profile['state'] as String? ?? 'Not set';
    final tierColor   = NexusColors.forTier(tier);

    return NexusCard(
      gradient: LinearGradient(
        colors: [NexusColors.surfaceHigh, NexusColors.surface],
        begin: Alignment.topLeft, end: Alignment.bottomRight,
      ),
      border: Border.all(color: tierColor.withValues(alpha: 0.3)),
      child: Column(children: [
        // Header
        Row(children: [
          const Icon(Icons.badge_outlined, color: NexusColors.textSecondary, size: 16),
          const SizedBox(width: 8),
          const Text('Membership Info', style: NexusText.subheading),
        ]),
        const SizedBox(height: 14),
        const Divider(color: NexusColors.border, height: 1),
        const SizedBox(height: 14),
        _InfoRow(label: 'Phone',     value: phone),
        _InfoRow(label: 'State',     value: state),
        _InfoRow(label: 'Tier',      value: tier,
            valueColor: tierColor),
        _InfoRow(label: 'MoMo Wallet',
            value: momoVerified
                ? '✅ Verified'
                : momoNumber.isNotEmpty ? '⏳ Pending' : '⚠️ Not linked',
            valueColor: momoVerified ? NexusColors.green
                : momoNumber.isNotEmpty ? NexusColors.gold : NexusColors.red),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          onPressed: () => context.push('/settings'),
          icon: const Icon(Icons.edit_outlined, size: 15),
          label: const Text('Edit in Settings'),
          style: OutlinedButton.styleFrom(
            minimumSize: const Size(double.infinity, 44),
            foregroundColor: NexusColors.primary,
            side: const BorderSide(color: NexusColors.primary),
          ),
        ),
      ]),
    );
  }
}

class _InfoRow extends StatelessWidget {
  final String label, value;
  final Color? valueColor;
  const _InfoRow({required this.label, required this.value, this.valueColor});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 5),
    child: Row(children: [
      SizedBox(width: 110,
          child: Text(label, style: NexusText.caption)),
      Expanded(child: Text(value, style: TextStyle(
          color: valueColor ?? NexusColors.textPrimary,
          fontSize: 13, fontWeight: FontWeight.w600))),
    ]),
  );
}

// ─── Earning guide ────────────────────────────────────────────────────────────

class _EarningGuide extends StatelessWidget {
  static const _items = [
    (emoji: '💎', title: 'Pulse Points',    body: '1 point per ₦250 recharged — used for AI Studio tools'),
    (emoji: '🎰', title: 'Spin Credits',    body: '1 credit per ₦1,000 recharged — spin the wheel to win'),
    (emoji: '🔥', title: 'Recharge Streak', body: 'Recharge within 36 hrs to keep your streak & unlock bonuses'),
    (emoji: '⚔️', title: 'Regional Wars',   body: 'Points from recharges rank your state — top states win cash'),
  ];

  @override
  Widget build(BuildContext context) => NexusCard(
    child: Column(children: [
      for (int i = 0; i < _items.length; i++) ...[
        if (i > 0) const Divider(color: NexusColors.border, height: 1),
        _EarnRow(emoji: _items[i].emoji, title: _items[i].title, body: _items[i].body),
      ],
    ]),
  );
}

class _EarnRow extends StatelessWidget {
  final String emoji, title, body;
  const _EarnRow({required this.emoji, required this.title, required this.body});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 4),
    child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text(emoji, style: const TextStyle(fontSize: 20)),
      const SizedBox(width: 12),
      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title, style: const TextStyle(color: NexusColors.textPrimary,
            fontSize: 13, fontWeight: FontWeight.w700)),
        const SizedBox(height: 2),
        Text(body, style: NexusText.caption),
      ])),
    ]),
  );
}

// ─── USSD banner ─────────────────────────────────────────────────────────────

class _UssdBanner extends StatelessWidget {
  @override
  Widget build(BuildContext context) => NexusCard(
    gradient: const LinearGradient(
      colors: [Color(0xFF1A2040), Color(0xFF141830)],
    ),
    border: Border.all(color: NexusColors.primary.withValues(alpha: 0.2)),
    child: Row(children: [
      Container(
        width: 44, height: 44,
        decoration: BoxDecoration(
          color: NexusColors.primaryGlow,
          borderRadius: NexusRadius.md,
        ),
        child: const Center(child: Text('📟', style: TextStyle(fontSize: 22))),
      ),
      const SizedBox(width: 14),
      const Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('USSD Access', style: TextStyle(color: NexusColors.textPrimary,
            fontSize: 13, fontWeight: FontWeight.w700)),
        SizedBox(height: 2),
        Text('Check balance & spin on any phone — even without internet.',
            style: TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
        SizedBox(height: 4),
        Text('Dial *789*6398#', style: TextStyle(color: NexusColors.gold,
            fontSize: 12, fontWeight: FontWeight.w700)),
      ])),
    ]),
  );
}
