import 'dart:async';
import 'dart:io';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ── Providers ──────────────────────────────────────────────────────────────────

final _passportDetailProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getPassport();
});

final _qrProvider = FutureProvider.autoDispose<String?>((ref) async {
  try {
    final res = await ref.read(dioProvider).apiGet<Map>('/user/passport/qr');
    return (res as Map)['qr_data_url']?.toString();
  } catch (_) { return null; }
});

// ── Tier constants (mirrors webapp) ──────────────────────────────────────────

const _tierThresholds = {'BRONZE': 0, 'SILVER': 2000, 'GOLD': 10000, 'PLATINUM': 50000};
const _tierNext       = {'BRONZE': 'SILVER', 'SILVER': 'GOLD', 'GOLD': 'PLATINUM'};

const _tierConfig = {
  'BRONZE':   _TierConfig(emoji: '🛡️', label: 'Bronze',   colors: [Color(0xFFb45309), Color(0xFFd97706)], accent: Color(0xFFf59e0b)),
  'SILVER':   _TierConfig(emoji: '⭐', label: 'Silver',   colors: [Color(0xFF475569), Color(0xFF94a3b8)], accent: Color(0xFF94a3b8)),
  'GOLD':     _TierConfig(emoji: '🏆', label: 'Gold',     colors: [Color(0xFF92400e), Color(0xFFf59e0b)], accent: Color(0xFFfbbf24)),
  'PLATINUM': _TierConfig(emoji: '💎', label: 'Platinum', colors: [Color(0xFF4c1d95), Color(0xFF7c3aed)], accent: Color(0xFFa78bfa)),
};

class _TierConfig {
  final String emoji, label;
  final List<Color> colors;
  final Color accent;
  const _TierConfig({required this.emoji, required this.label, required this.colors, required this.accent});
}

_TierConfig _cfg(String tier) => _tierConfig[tier] ?? _tierConfig['BRONZE']!;

double _tierProgress(int life, String tier) {
  final min    = _tierThresholds[tier] ?? 0;
  final next   = _tierNext[tier];
  if (next == null) return 1.0;
  final nextMin = _tierThresholds[next] ?? 1;
  return ((life - min) / (nextMin - min)).clamp(0.0, 1.0);
}

// ── Badge config (mirrors webapp BADGE_RARITY) ────────────────────────────────

const _badgeRarity = {
  'first_recharge': 'common',  'streak_7': 'common',   'streak_30': 'rare',
  'streak_90': 'epic',         'spin_first': 'common', 'spin_100': 'rare',
  'studio_first': 'common',    'studio_50': 'rare',    'wars_top3': 'epic',
  'silver_tier': 'common',     'gold_tier': 'rare',    'platinum_tier': 'legendary',
  'referral_5': 'rare',        'big_winner': 'epic',
};

final _rarityColors = {
  'common':    [const Color(0x1AFFFFFF), const Color(0x1AFFFFFF)],
  'rare':      [const Color(0x1A5F72F9), const Color(0x0A5F72F9)],
  'epic':      [const Color(0x1A8B5CF6), const Color(0x0A8B5CF6)],
  'legendary': [const Color(0x1AF59E0B), const Color(0x0AF59E0B)],
};

// ── Passport screen ───────────────────────────────────────────────────────────

class PassportScreen extends ConsumerStatefulWidget {
  const PassportScreen({super.key});
  @override
  ConsumerState<PassportScreen> createState() => _PassportScreenState();
}

class _PassportScreenState extends ConsumerState<PassportScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  bool _showQR = false;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 2, vsync: this);
    _tabs.addListener(() => setState(() {}));
  }

  @override
  void dispose() { _tabs.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    final passportAsync = ref.watch(_passportDetailProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        title: const Text('Digital Passport 🛡️'),
        actions: [
          // Share button
          passportAsync.when(
            data: (p) => IconButton(
              icon: const Icon(Icons.share_outlined),
              color: NexusColors.textSecondary,
              onPressed: () => _share(p),
            ),
            loading: () => const SizedBox.shrink(),
            error: (_, __) => const SizedBox.shrink(),
          ),
          // QR button
          IconButton(
            icon: const Icon(Icons.qr_code_2_rounded),
            color: NexusColors.textSecondary,
            onPressed: () => setState(() => _showQR = !_showQR),
          ),
        ],
      ),
      body: passportAsync.when(
        loading: () => _PassportShimmer(),
        error: (e, _) => _ErrorView(message: e.toString(), onRetry: () => ref.invalidate(_passportDetailProvider)),
        data: (passport) => RefreshIndicator(
          color: NexusColors.primary,
          onRefresh: () async => ref.invalidate(_passportDetailProvider),
          child: SingleChildScrollView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(20, 8, 20, 100),
            child: Column(children: [
              // ── Passport card ─────────────────────────────────────────────
              _PassportCard(passport: passport),
              const SizedBox(height: 16),

              // ── QR code (toggleable) ──────────────────────────────────────
              AnimatedSize(
                duration: const Duration(milliseconds: 300),
                curve: Curves.easeInOut,
                child: _showQR ? _QRSection() : const SizedBox.shrink(),
              ),

              // ── Wallet buttons ────────────────────────────────────────────
              _WalletButtons(passport: passport),
              const SizedBox(height: 16),

              // ── Tabs: Overview / Badges ───────────────────────────────────
              _buildTabBar(passport),
              const SizedBox(height: 16),

              // ── Tab content ───────────────────────────────────────────────
              _tabs.index == 0
                  ? _OverviewTab(passport: passport)
                  : _BadgesTab(badges: (passport['badges'] as List? ?? [])),
            ]),
          ),
        ),
      ),
    );
  }

  Widget _buildTabBar(Map<String, dynamic> passport) {
    final badgeCount = (passport['badges'] as List?)?.length ?? 0;
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: NexusColors.border),
      ),
      child: TabBar(
        controller: _tabs,
        indicator: BoxDecoration(
          color: NexusColors.primary,
          borderRadius: BorderRadius.circular(10),
        ),
        labelColor: Colors.white,
        unselectedLabelColor: NexusColors.textSecondary,
        labelStyle: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13),
        unselectedLabelStyle: const TextStyle(fontSize: 13),
        dividerColor: Colors.transparent,
        indicatorSize: TabBarIndicatorSize.tab,
        tabs: [
          const Tab(text: 'Overview'),
          Tab(text: 'Badges ($badgeCount)'),
        ],
      ),
    );
  }

  void _share(Map<String, dynamic> passport) {
    final tier = passport['tier']?.toString() ?? 'BRONZE';
    final pts  = passport['lifetime_points'] as int? ?? 0;
    final text = 'I\'m a $tier member on Loyalty Nexus with ${_fmtPts(pts)} Pulse Points! ⚡';
    Clipboard.setData(ClipboardData(text: text));
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
      content: Text('Copied to clipboard! 🎯'),
      behavior: SnackBarBehavior.floating,
      backgroundColor: NexusColors.green,
    ));
  }

  String _fmtPts(int v) => v.toString().replaceAllMapped(
    RegExp(r'(\d{1,3})(?=(\d{3})+(?!\d))'), (m) => '${m[1]},');
}

// ── Passport Card ──────────────────────────────────────────────────────────────

class _PassportCard extends StatelessWidget {
  final Map<String, dynamic> passport;
  const _PassportCard({required this.passport});

  @override
  Widget build(BuildContext context) {
    final tier     = passport['tier']?.toString() ?? 'BRONZE';
    final life     = passport['lifetime_points'] as int? ?? 0;
    final streak   = passport['streak_count'] as int? ?? passport['current_streak'] as int? ?? 0;
    final badges   = (passport['badges'] as List?)?.length ?? 0;
    final nextTier = passport['next_tier']?.toString() ?? _tierNext[tier];
    final pts2next = passport['points_to_next_tier'] as int?;
    final progress = _tierProgress(life, tier);
    final cfg      = _cfg(tier);

    return Container(
      width: double.infinity,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(24),
        gradient: LinearGradient(
          colors: cfg.colors, begin: Alignment.topLeft, end: Alignment.bottomRight),
        boxShadow: [
          BoxShadow(color: cfg.colors[1].withOpacity(0.4), blurRadius: 30, offset: const Offset(0, 8)),
        ],
      ),
      child: Stack(children: [
        // Background shimmer pattern
        Positioned.fill(child: ClipRRect(
          borderRadius: BorderRadius.circular(24),
          child: CustomPaint(painter: _CardPatternPainter()),
        )),

        Padding(
          padding: const EdgeInsets.all(24),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            // Header row
            Row(children: [
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.white.withOpacity(0.2),
                  borderRadius: BorderRadius.circular(14),
                ),
                child: Text(cfg.emoji, style: const TextStyle(fontSize: 24)),
              ),
              const SizedBox(width: 14),
              Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                const Text('LOYALTY NEXUS',
                  style: TextStyle(color: Colors.white60, fontSize: 10, letterSpacing: 2.5)),
                const SizedBox(height: 3),
                Text('${cfg.label} Member',
                  style: const TextStyle(color: Colors.white, fontSize: 20,
                    fontWeight: FontWeight.w800, letterSpacing: -0.5)),
              ])),
              Text('${DateTime.now().year}',
                style: const TextStyle(color: Colors.white38, fontSize: 13)),
            ]),

            const SizedBox(height: 22),

            // Points
            const Text('Lifetime Pulse Points',
              style: TextStyle(color: Colors.white54, fontSize: 11, letterSpacing: 1)),
            const SizedBox(height: 4),
            Row(crossAxisAlignment: CrossAxisAlignment.end, children: [
              Text(_fmtPts(life),
                style: const TextStyle(color: Colors.white, fontSize: 38,
                  fontWeight: FontWeight.w800, letterSpacing: -1.5)),
              const Padding(
                padding: EdgeInsets.only(bottom: 6, left: 6),
                child: Text('pts',
                  style: TextStyle(color: Colors.white38, fontSize: 16)),
              ),
            ]),

            const SizedBox(height: 18),

            // Stats row
            Row(children: [
              _StatPill('🔥', '$streak', 'day streak'),
              const SizedBox(width: 12),
              _StatPill('🏅', '$badges', 'badge${badges != 1 ? 's' : ''}'),
              const SizedBox(width: 12),
              _StatPill('⚡', tier, ''),
            ]),

            // Tier progress
            if (nextTier != null) ...[
              const SizedBox(height: 20),
              Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                Text(cfg.label,
                  style: const TextStyle(color: Colors.white60, fontSize: 11)),
                Text(pts2next != null ? '${_fmtPts(pts2next)} to $nextTier' : 'Progress to $nextTier',
                  style: const TextStyle(color: Colors.white60, fontSize: 11)),
              ]),
              const SizedBox(height: 6),
              ClipRRect(
                borderRadius: BorderRadius.circular(4),
                child: LinearProgressIndicator(
                  value: progress, minHeight: 5,
                  backgroundColor: Colors.white.withOpacity(0.15),
                  valueColor: const AlwaysStoppedAnimation(Colors.white),
                ),
              ),
              const SizedBox(height: 4),
              Text('${(progress * 100).round()}% to $nextTier',
                style: const TextStyle(color: Colors.white30, fontSize: 10),
                textAlign: TextAlign.right),
            ] else ...[
              const SizedBox(height: 16),
              const Row(children: [
                Icon(Icons.diamond_rounded, color: Colors.white60, size: 14),
                SizedBox(width: 6),
                Text('Platinum — Maximum Tier 👑',
                  style: TextStyle(color: Colors.white60, fontSize: 12)),
              ]),
            ],
          ]),
        ),
      ]),
    );
  }

  String _fmtPts(int v) => v.toString().replaceAllMapped(
    RegExp(r'(\d{1,3})(?=(\d{3})+(?!\d))'), (m) => '${m[1]},');
}

class _StatPill extends StatelessWidget {
  final String emoji, value, sub;
  const _StatPill(this.emoji, this.value, this.sub);
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
    decoration: BoxDecoration(
      color: Colors.white.withOpacity(0.15),
      borderRadius: BorderRadius.circular(20),
      border: Border.all(color: Colors.white.withOpacity(0.2)),
    ),
    child: Row(mainAxisSize: MainAxisSize.min, children: [
      Text(emoji, style: const TextStyle(fontSize: 13)),
      const SizedBox(width: 5),
      Text(sub.isNotEmpty ? '$value $sub' : value,
        style: const TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.w600)),
    ]),
  );
}

class _CardPatternPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size s) {
    final paint = Paint()
      ..color = Colors.white.withOpacity(0.04)
      ..style = PaintingStyle.fill;
    for (int i = 0; i < 5; i++) {
      canvas.drawCircle(Offset(s.width * 0.85, s.height * 0.15 + i * 30), 40 + i * 20.0, paint);
    }
  }
  @override bool shouldRepaint(_) => false;
}

// ── QR Section ─────────────────────────────────────────────────────────────────

class _QRSection extends ConsumerWidget {
  const _QRSection();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final qrAsync = ref.watch(_qrProvider);
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Container(
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(color: NexusColors.border),
        ),
        child: Column(children: [
          const Text('Scan to verify your passport',
            style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
          const SizedBox(height: 16),
          qrAsync.when(
            loading: () => const NexusShimmer(
              width: 160, height: 160,
              radius: BorderRadius.all(Radius.circular(12))),
            error: (_, __) => const Text('QR code unavailable',
              style: TextStyle(color: NexusColors.textSecondary)),
            data: (qrUrl) => qrUrl == null
                ? const Text('QR code unavailable',
                    style: TextStyle(color: NexusColors.textSecondary))
                : Container(
                    width: 160, height: 160,
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    // QR is a base64 data URL from the API — display as Image.network
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(12),
                      child: Image.network(qrUrl, fit: BoxFit.cover,
                        errorBuilder: (_, __, ___) => const Icon(Icons.qr_code_2_rounded,
                          color: Colors.black, size: 120)),
                    ),
                  ),
          ),
        ]),
      ),
    );
  }
}

// ── Wallet Buttons ─────────────────────────────────────────────────────────────

class _WalletButtons extends StatelessWidget {
  final Map<String, dynamic> passport;
  const _WalletButtons({required this.passport});

  @override
  Widget build(BuildContext context) => Row(children: [
    // Apple Wallet — iOS only
    if (Platform.isIOS) ...[
      Expanded(child: _WalletBtn(
        label: 'Apple Wallet',
        icon: Icons.phone_iphone_rounded,
        color: Colors.black,
        borderColor: Colors.white24,
        textColor: Colors.white,
        onTap: () => launchUrl(Uri.parse('/user/passport/apple-wallet')),
      )),
      const SizedBox(width: 12),
    ],
    // Google Wallet — Android
    if (Platform.isAndroid)
      Expanded(child: _WalletBtn(
        label: 'Google Wallet',
        icon: Icons.account_balance_wallet_outlined,
        color: const Color(0xFF1a73e8),
        borderColor: const Color(0xFF1a73e8),
        textColor: Colors.white,
        onTap: () async {
          final url = passport['google_wallet_url']?.toString();
          if (url != null) await launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication);
        },
      )),
    // Download PDF fallback
    const SizedBox(width: 12),
    Expanded(child: _WalletBtn(
      label: 'Download PDF',
      icon: Icons.download_rounded,
      color: NexusColors.surface,
      borderColor: NexusColors.border,
      textColor: NexusColors.textSecondary,
      onTap: () async {
        await launchUrl(Uri.parse('/user/passport/download'), mode: LaunchMode.externalApplication);
      },
    )),
  ]);
}

class _WalletBtn extends StatelessWidget {
  final String label;
  final IconData icon;
  final Color color, borderColor, textColor;
  final VoidCallback onTap;
  const _WalletBtn({required this.label, required this.icon, required this.color,
    required this.borderColor, required this.textColor, required this.onTap});

  @override
  Widget build(BuildContext context) => GestureDetector(
    onTap: onTap,
    child: Container(
      padding: const EdgeInsets.symmetric(vertical: 12),
      decoration: BoxDecoration(
        color: color, borderRadius: BorderRadius.circular(14),
        border: Border.all(color: borderColor),
      ),
      child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
        Icon(icon, size: 16, color: textColor),
        const SizedBox(width: 7),
        Text(label, style: TextStyle(color: textColor, fontSize: 12, fontWeight: FontWeight.w600)),
      ]),
    ),
  );
}

// ── Overview tab ───────────────────────────────────────────────────────────────

class _OverviewTab extends StatelessWidget {
  final Map<String, dynamic> passport;
  const _OverviewTab({required this.passport});

  @override
  Widget build(BuildContext context) {
    final tier        = passport['tier']?.toString() ?? 'BRONZE';
    final life        = passport['lifetime_points'] as int? ?? 0;
    final current     = passport['pulse_points'] as int? ?? 0;
    final streak      = passport['streak_count'] as int? ?? 0;
    final longestStreak = passport['longest_streak'] as int? ?? 0;
    final totalSpins  = passport['total_spins'] as int? ?? 0;
    final phone       = passport['phone_number']?.toString() ?? '';
    final state       = passport['state']?.toString();

    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      // Stats grid
      _SectionTitle('Your Stats'),
      const SizedBox(height: 10),
      GridView.count(
        crossAxisCount: 2, shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        crossAxisSpacing: 12, mainAxisSpacing: 12,
        childAspectRatio: 1.7,
        children: [
          _StatCard('⚡', 'Current Points', '$current pts', NexusColors.primary),
          _StatCard('⭐', 'Lifetime Points', '$life pts', NexusColors.gold),
          _StatCard('🔥', 'Current Streak', '$streak days', const Color(0xFFf97316)),
          _StatCard('📈', 'Longest Streak', '$longestStreak days', NexusColors.green),
          _StatCard('🎡', 'Total Spins', '$totalSpins', const Color(0xFF8B5CF6)),
        ],
      ),

      const SizedBox(height: 20),

      // Account info
      _SectionTitle('Account Info'),
      const SizedBox(height: 10),
      _InfoCard(items: [
        _InfoRow(icon: Icons.phone_android_rounded, label: 'Phone', value: phone),
        if (state != null) _InfoRow(icon: Icons.location_on_outlined, label: 'State', value: state),
        _InfoRow(icon: Icons.shield_outlined, label: 'Tier', value: tier),
      ]),

      const SizedBox(height: 20),

      // Tier progression
      _SectionTitle('Tier Progression'),
      const SizedBox(height: 10),
      _TierProgressionCard(lifetimePoints: life),

      const SizedBox(height: 20),

      // Points sparkline
      _SectionTitle('Points Journey'),
      const SizedBox(height: 10),
      _PointsSparkline(lifetimePoints: life),
    ]);
  }
}

class _SectionTitle extends StatelessWidget {
  final String text;
  const _SectionTitle(this.text);
  @override
  Widget build(BuildContext context) => Text(text,
    style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12,
      fontWeight: FontWeight.w700, letterSpacing: 0.8));
}

class _StatCard extends StatelessWidget {
  final String emoji, label, value;
  final Color accent;
  const _StatCard(this.emoji, this.label, this.value, this.accent);
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(14),
    decoration: BoxDecoration(
      color: NexusColors.surface, borderRadius: BorderRadius.circular(16),
      border: Border.all(color: NexusColors.border),
    ),
    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Text(emoji, style: const TextStyle(fontSize: 18)),
        const Spacer(),
        Container(width: 6, height: 6, decoration: BoxDecoration(
          color: accent, shape: BoxShape.circle)),
      ]),
      const Spacer(),
      Text(value, style: TextStyle(color: accent, fontSize: 15, fontWeight: FontWeight.w700)),
      const SizedBox(height: 2),
      Text(label, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
    ]),
  );
}

class _InfoRow {
  final IconData icon;
  final String label, value;
  const _InfoRow({required this.icon, required this.label, required this.value});
}

class _InfoCard extends StatelessWidget {
  final List<_InfoRow> items;
  const _InfoCard({required this.items});
  @override
  Widget build(BuildContext context) => Container(
    decoration: BoxDecoration(
      color: NexusColors.surface, borderRadius: BorderRadius.circular(16),
      border: Border.all(color: NexusColors.border),
    ),
    child: Column(children: List.generate(items.length, (i) => Column(children: [
      Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 13),
        child: Row(children: [
          Icon(items[i].icon, size: 16, color: NexusColors.textSecondary),
          const SizedBox(width: 12),
          Text(items[i].label,
            style: const TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
          const Spacer(),
          Text(items[i].value,
            style: const TextStyle(color: NexusColors.textPrimary, fontSize: 13,
              fontWeight: FontWeight.w500)),
        ]),
      ),
      if (i < items.length - 1)
        Divider(height: 1, color: NexusColors.border),
    ]))),
  );
}

class _TierProgressionCard extends StatelessWidget {
  final int lifetimePoints;
  const _TierProgressionCard({required this.lifetimePoints});

  static const _tiers = ['BRONZE', 'SILVER', 'GOLD', 'PLATINUM'];
  static const _emojis = ['🛡️', '⭐', '🏆', '💎'];

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(16),
    decoration: BoxDecoration(
      color: NexusColors.surface, borderRadius: BorderRadius.circular(16),
      border: Border.all(color: NexusColors.border),
    ),
    child: Column(children: List.generate(_tiers.length, (i) {
      final tier      = _tiers[i];
      final threshold = _tierThresholds[tier] ?? 0;
      final unlocked  = lifetimePoints >= threshold;
      final cfg       = _cfg(tier);
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Row(children: [
          Text(_emojis[i], style: TextStyle(
            fontSize: 22, color: unlocked ? null : const Color(0xFF374151))),
          const SizedBox(width: 12),
          Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(cfg.label,
              style: TextStyle(
                color: unlocked ? NexusColors.textPrimary : NexusColors.textSecondary,
                fontSize: 13, fontWeight: FontWeight.w600)),
            Text(threshold == 0 ? 'Starting tier' : '${_fmt(threshold)} lifetime pts',
              style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
          ])),
          if (unlocked)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: cfg.accent.withOpacity(0.15), borderRadius: BorderRadius.circular(20)),
              child: Text('Unlocked', style: TextStyle(
                color: cfg.accent, fontSize: 10, fontWeight: FontWeight.w700)),
            )
          else
            Text('${_fmt(threshold - lifetimePoints)} to go',
              style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
        ]),
      );
    })),
  );

  String _fmt(int v) => v.toString().replaceAllMapped(
    RegExp(r'(\d{1,3})(?=(\d{3})+(?!\d))'), (m) => '${m[1]},');
}

// ── Badges tab ─────────────────────────────────────────────────────────────────

class _BadgesTab extends StatelessWidget {
  final List<dynamic> badges;
  const _BadgesTab({required this.badges});

  @override
  Widget build(BuildContext context) {
    if (badges.isEmpty) return _EmptyBadges();
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      _SectionTitle('${badges.length} badge${badges.length != 1 ? 's' : ''} earned'),
      const SizedBox(height: 12),
      GridView.builder(
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 3, crossAxisSpacing: 10, mainAxisSpacing: 10, childAspectRatio: 0.85),
        itemCount: badges.length,
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        itemBuilder: (_, i) => _BadgeCard(badge: badges[i] as Map),
      ),
    ]);
  }
}

class _BadgeCard extends StatelessWidget {
  final Map badge;
  const _BadgeCard({required this.badge});

  @override
  Widget build(BuildContext context) {
    final slug    = badge['badge_type']?.toString() ?? badge['slug']?.toString() ?? '';
    final name    = badge['name']?.toString() ?? slug.replaceAll('_', ' ');
    final icon    = badge['icon']?.toString() ?? '🏅';
    final rarity  = _badgeRarity[slug] ?? 'common';
    final colors  = _rarityColors[rarity] ?? _rarityColors['common']!;

    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: colors[0], borderRadius: BorderRadius.circular(14),
        border: Border.all(color: colors[1].withOpacity(4)),
      ),
      child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
        Text(icon, style: const TextStyle(fontSize: 28)),
        const SizedBox(height: 6),
        Text(name, style: const TextStyle(color: NexusColors.textPrimary,
          fontSize: 10, fontWeight: FontWeight.w600), textAlign: TextAlign.center,
          maxLines: 2, overflow: TextOverflow.ellipsis),
        const SizedBox(height: 3),
        Text(rarity.toUpperCase(),
          style: TextStyle(
            color: rarity == 'legendary' ? NexusColors.gold
                 : rarity == 'epic'      ? const Color(0xFF8B5CF6)
                 : rarity == 'rare'      ? NexusColors.primary
                 : NexusColors.textSecondary,
            fontSize: 8, fontWeight: FontWeight.w800, letterSpacing: 0.8)),
      ]),
    );
  }
}

class _EmptyBadges extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 40),
    child: Column(children: [
      const Text('🏅', style: TextStyle(fontSize: 56)),
      const SizedBox(height: 16),
      const Text('No badges yet', style: TextStyle(color: NexusColors.textPrimary,
        fontSize: 18, fontWeight: FontWeight.w600)),
      const SizedBox(height: 8),
      const Text('Earn badges by recharging, spinning, and hitting milestones.',
        style: TextStyle(color: NexusColors.textSecondary, fontSize: 13),
        textAlign: TextAlign.center),
    ]),
  );
}

// ── Error view ─────────────────────────────────────────────────────────────────

class _ErrorView extends StatelessWidget {
  final String message;
  final VoidCallback onRetry;
  const _ErrorView({required this.message, required this.onRetry});
  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(32),
      child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
        const Icon(Icons.error_outline_rounded, size: 56, color: NexusColors.textSecondary),
        const SizedBox(height: 16),
        Text(message, style: const TextStyle(color: NexusColors.textSecondary),
          textAlign: TextAlign.center),
        const SizedBox(height: 16),
        ElevatedButton.icon(
          onPressed: onRetry,
          icon: const Icon(Icons.refresh_rounded, size: 16),
          label: const Text('Retry'),
        ),
      ]),
    ),
  );
}

// ── Section title helper (used in OverviewTab, reused here) ───────────────────
class _SectionTitle extends StatelessWidget {
  final String text;
  const _SectionTitle(this.text);
  @override
  Widget build(BuildContext context) => Text(text,
    style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12,
      fontWeight: FontWeight.w700, letterSpacing: 0.8));
}


// ── Passport shimmer skeleton ─────────────────────────────────────────────────
class _PassportShimmer extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.all(20),
    child: Column(children: const [
      NexusShimmer(width: double.infinity, height: 200, radius: NexusRadius.lg),
      SizedBox(height: 16),
      NexusShimmer(width: double.infinity, height: 100, radius: NexusRadius.md),
      SizedBox(height: 12),
      NexusShimmer(width: double.infinity, height: 80,  radius: NexusRadius.md),
      SizedBox(height: 12),
      NexusShimmer(width: double.infinity, height: 200, radius: NexusRadius.md),
    ]),
  );
}

// ── Points Journey sparkline ───────────────────────────────────────────────────
/// Shows tier milestones (0, Silver@2000, Gold@10000, Platinum@50000) as
/// waypoints and plots the user's current lifetime points on the curve.
class _PointsSparkline extends StatelessWidget {
  final int lifetimePoints;
  const _PointsSparkline({required this.lifetimePoints});

  static const _milestones = [
    (label: 'Start',    pts: 0),
    (label: 'Silver',   pts: 2000),
    (label: 'Gold',     pts: 10000),
    (label: 'Platinum', pts: 50000),
  ];

  @override
  Widget build(BuildContext context) {
    // Build a curve through milestone y-values; add user point
    final sorted = [
      ..._milestones.map((m) => FlSpot(m.pts.toDouble(), m.pts.toDouble())),
    ];

    // User's current progress spot — clamp to 50000 max for display
    final clampedPts = lifetimePoints.clamp(0, 50000).toDouble();
    final userSpot   = FlSpot(clampedPts, clampedPts);

    return Container(
      height: 160,
      padding: const EdgeInsets.fromLTRB(0, 8, 16, 8),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: NexusColors.border),
      ),
      child: LineChart(LineChartData(
        minX: 0, maxX: 50000,
        minY: 0, maxY: 50000,
        gridData: FlGridData(
          show: true,
          drawVerticalLine: false,
          getDrawingHorizontalLine: (_) => const FlLine(
              color: NexusColors.border, strokeWidth: 0.5),
        ),
        borderData: FlBorderData(show: false),
        titlesData: FlTitlesData(
          leftTitles: const AxisTitles(sideTitles: SideTitles(showTitles: false)),
          rightTitles: const AxisTitles(sideTitles: SideTitles(showTitles: false)),
          topTitles: const AxisTitles(sideTitles: SideTitles(showTitles: false)),
          bottomTitles: AxisTitles(sideTitles: SideTitles(
            showTitles: true,
            getTitlesWidget: (v, meta) {
              final m = _milestones.firstWhere(
                  (m) => (m.pts.toDouble() - v).abs() < 1,
                  orElse: () => (label: '', pts: -1));
              if (m.pts == -1 || m.label.isEmpty) return const SizedBox.shrink();
              return Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Text(m.label,
                    style: const TextStyle(color: NexusColors.textMuted,
                        fontSize: 10, fontWeight: FontWeight.w600)),
              );
            },
            reservedSize: 24,
          )),
        ),
        lineBarsData: [
          // Milestone baseline
          LineChartBarData(
            spots: sorted,
            isCurved: true,
            color: NexusColors.border,
            barWidth: 1.5,
            dotData: FlDotData(
              show: true,
              getDotPainter: (s, _, __, ___) => FlDotCirclePainter(
                  radius: 3, color: NexusColors.surface,
                  strokeWidth: 1.5, strokeColor: NexusColors.textMuted),
            ),
            belowBarData: BarAreaData(show: false),
          ),
          // User progress line
          LineChartBarData(
            spots: [FlSpot(0, 0), userSpot],
            isCurved: true,
            gradient: LinearGradient(colors: [NexusColors.primary, NexusColors.gold]),
            barWidth: 3,
            dotData: FlDotData(
              show: true,
              getDotPainter: (s, _, __, ___) {
                if ((s.x - clampedPts).abs() < 1) {
                  return FlDotCirclePainter(
                    radius: 6, color: NexusColors.primary,
                    strokeWidth: 2, strokeColor: Colors.white);
                }
                return FlDotCirclePainter(radius: 0, color: Colors.transparent);
              },
            ),
            belowBarData: BarAreaData(
              show: true,
              gradient: LinearGradient(
                begin: Alignment.topCenter, end: Alignment.bottomCenter,
                colors: [
                  NexusColors.primary.withOpacity(0.25),
                  NexusColors.primary.withOpacity(0.0),
                ],
              ),
            ),
          ),
        ],
        lineTouchData: LineTouchData(
          touchTooltipData: LineTouchTooltipData(
            getTooltipItems: (spots) => spots.map((s) => LineTooltipItem(
              '${s.y.toInt()} pts',
              const TextStyle(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 11),
            )).toList(),
          ),
        ),
      )),
    );
  }
}

