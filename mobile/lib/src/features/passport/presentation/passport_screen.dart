import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';
import '../../../core/widgets/nexus_widgets.dart';

// ─── Providers ───────────────────────────────────────────────────────────────
final _passportProvider =
    FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(passportApiProvider).getPassport();
});

final _passportEventsProvider =
    FutureProvider.autoDispose<List<dynamic>>((ref) async {
  final raw = await ref.read(passportApiProvider).getPassportEvents(limit: 20);
  return (raw as List?) ?? [];
});

// ─── Passport Screen ──────────────────────────────────────────────────────────
class PassportScreen extends ConsumerStatefulWidget {
  const PassportScreen({super.key});
  @override
  ConsumerState<PassportScreen> createState() => _PassportScreenState();
}

class _PassportScreenState extends ConsumerState<PassportScreen>
    with TickerProviderStateMixin {
  late final AnimationController _shimmerCtrl;
  late final AnimationController _pulseCtrl;
  late final Animation<double> _shimmerAnim;
  late final Animation<double> _pulseAnim;

  @override
  void initState() {
    super.initState();
    _shimmerCtrl = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 3),
    )..repeat();
    _pulseCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1500),
    )..repeat(reverse: true);
    _shimmerAnim = Tween<double>(begin: -2, end: 2).animate(
      CurvedAnimation(parent: _shimmerCtrl, curve: Curves.easeInOut),
    );
    _pulseAnim = Tween<double>(begin: 0.95, end: 1.05).animate(
      CurvedAnimation(parent: _pulseCtrl, curve: Curves.easeInOut),
    );
  }

  @override
  void dispose() {
    _shimmerCtrl.dispose();
    _pulseCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final passportAsync = ref.watch(_passportProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        surfaceTintColor: Colors.transparent,
        title: const Text('Digital Passport'),
        centerTitle: false,
        actions: [
          IconButton(
            icon: const Icon(Icons.share_outlined,
                color: NexusColors.textSecondary),
            onPressed: () => _sharePassport(passportAsync.valueOrNull),
          ),
        ],
      ),
      body: passportAsync.when(
        loading: () => const _PassportSkeleton(),
        error: (_, __) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Text('😕', style: TextStyle(fontSize: 48)),
              const SizedBox(height: 12),
              const Text('Could not load passport',
                  style: TextStyle(color: NexusColors.textSecondary)),
              const SizedBox(height: 12),
              ElevatedButton(
                onPressed: () => ref.invalidate(_passportProvider),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
        data: (passport) => RefreshIndicator(
          color: NexusColors.primary,
          backgroundColor: NexusColors.surfaceCard,
          onRefresh: () async => ref.invalidate(_passportProvider),
          child: ListView(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 100),
            children: [
              // ── Passport Card ────────────────────────────────────────
              _PassportCard(
                passport: passport,
                shimmerAnim: _shimmerAnim,
              ),
              const SizedBox(height: 20),

              // ── QR Code ──────────────────────────────────────────────
              _QRSection(
                passport: passport,
                pulseAnim: _pulseAnim,
              ),
              const SizedBox(height: 20),

              // ── Tier Progress ────────────────────────────────────────
              _TierProgressCard(passport: passport),
              const SizedBox(height: 20),

              // ── Streak ───────────────────────────────────────────────
              _StreakCard(passport: passport),
              const SizedBox(height: 20),

              // ── Wallet CTAs ──────────────────────────────────────────
              _WalletCTAs(passport: passport),
              const SizedBox(height: 20),

              // ── Activity / Events ────────────────────────────────────
              _ActivitySection(),
            ],
          ),
        ),
      ),
    );
  }

  void _sharePassport(Map<String, dynamic>? passport) {
    if (passport == null) return;
    final passportId = passport['passport_id']?.toString() ??
        passport['id']?.toString() ?? '';
    final name = passport['display_name']?.toString() ?? 'Member';
    final tier = passport['tier']?.toString() ?? 'Bronze';
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text('Sharing $name\'s $tier passport...'),
      backgroundColor: NexusColors.surfaceCard,
      behavior: SnackBarBehavior.floating,
      shape:
          RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
    ));
  }
}

// ─── Passport Card ────────────────────────────────────────────────────────────
class _PassportCard extends StatelessWidget {
  final Map<String, dynamic> passport;
  final Animation<double> shimmerAnim;
  const _PassportCard(
      {required this.passport, required this.shimmerAnim});

  @override
  Widget build(BuildContext context) {
    final tier        = passport['tier']?.toString() ?? 'BRONZE';
    final name        = passport['display_name']?.toString() ?? 'Member';
    final phone       = passport['phone_number']?.toString() ?? '';
    final passportId  = passport['passport_id']?.toString() ??
        passport['id']?.toString() ?? '';
    final totalPoints = passport['total_points'] as int? ?? 0;
    final memberSince = _fmt(passport['created_at']?.toString() ?? '');

    final (gradStart, gradEnd) = _tierGradient(tier);

    return AnimatedBuilder(
      animation: shimmerAnim,
      builder: (_, __) {
        return Container(
          height: 200,
          decoration: BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [gradStart, gradEnd],
            ),
            borderRadius: BorderRadius.circular(20),
            boxShadow: [
              BoxShadow(
                color: gradStart.withOpacity(0.4),
                blurRadius: 30,
                offset: const Offset(0, 12),
              ),
            ],
          ),
          child: Stack(
            children: [
              // Shimmer overlay
              Positioned.fill(
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(20),
                  child: CustomPaint(
                    painter: _ShimmerPainter(shimmerAnim.value),
                  ),
                ),
              ),
              // Decorative circles
              Positioned(
                top: -30,
                right: -30,
                child: Container(
                  width: 120,
                  height: 120,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: Colors.white.withOpacity(0.05),
                  ),
                ),
              ),
              Positioned(
                bottom: -20,
                left: -20,
                child: Container(
                  width: 80,
                  height: 80,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: Colors.white.withOpacity(0.05),
                  ),
                ),
              ),
              // Content
              Padding(
                padding: const EdgeInsets.all(20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 10, vertical: 4),
                          decoration: BoxDecoration(
                            color: Colors.white.withOpacity(0.2),
                            borderRadius: BorderRadius.circular(20),
                            border: Border.all(
                                color: Colors.white.withOpacity(0.3)),
                          ),
                          child: Text(
                            '${_tierEmoji(tier)} ${tier.toUpperCase()}',
                            style: const TextStyle(
                                color: Colors.white,
                                fontWeight: FontWeight.w800,
                                fontSize: 11,
                                letterSpacing: 0.5),
                          ),
                        ),
                        const Spacer(),
                        const Text('LOYALTY NEXUS',
                            style: TextStyle(
                                color: Colors.white60,
                                fontSize: 10,
                                fontWeight: FontWeight.w700,
                                letterSpacing: 1.5)),
                      ],
                    ),
                    const Spacer(),
                    Text(name,
                        style: const TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.w900,
                            fontSize: 22,
                            fontFamily: 'Syne')),
                    const SizedBox(height: 2),
                    Text(phone,
                        style: const TextStyle(
                            color: Colors.white70, fontSize: 13)),
                    const SizedBox(height: 10),
                    Row(
                      children: [
                        Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text('POINTS',
                                style: TextStyle(
                                    color: Colors.white60,
                                    fontSize: 9,
                                    fontWeight: FontWeight.w700,
                                    letterSpacing: 1)),
                            Text(_fmtNum(totalPoints),
                                style: const TextStyle(
                                    color: Colors.white,
                                    fontWeight: FontWeight.w800,
                                    fontSize: 16)),
                          ],
                        ),
                        const SizedBox(width: 24),
                        Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text('MEMBER SINCE',
                                style: TextStyle(
                                    color: Colors.white60,
                                    fontSize: 9,
                                    fontWeight: FontWeight.w700,
                                    letterSpacing: 1)),
                            Text(memberSince,
                                style: const TextStyle(
                                    color: Colors.white,
                                    fontWeight: FontWeight.w800,
                                    fontSize: 16)),
                          ],
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  (Color, Color) _tierGradient(String tier) {
    switch (tier.toUpperCase()) {
      case 'GOLD':
        return (const Color(0xFFD4A017), const Color(0xFFF5C842));
      case 'SILVER':
        return (const Color(0xFF8E9EAB), const Color(0xFFB0BEC5));
      case 'PLATINUM':
        return (const Color(0xFF4A56EE), const Color(0xFF8B5CF6));
      default: // BRONZE
        return (const Color(0xFFCD7F32), const Color(0xFFE8A44A));
    }
  }

  String _tierEmoji(String tier) {
    switch (tier.toUpperCase()) {
      case 'GOLD':     return '🥇';
      case 'SILVER':   return '🥈';
      case 'PLATINUM': return '💎';
      default:         return '🥉';
    }
  }

  String _fmt(String iso) {
    try {
      final dt = DateTime.parse(iso).toLocal();
      const months = [
        'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
        'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'
      ];
      return '${months[dt.month - 1]} ${dt.year}';
    } catch (_) {
      return '—';
    }
  }

  String _fmtNum(int n) {
    if (n >= 1000000) return '${(n / 1000000).toStringAsFixed(1)}M';
    if (n >= 1000) return '${(n / 1000).toStringAsFixed(1)}K';
    return n.toString();
  }
}

// ─── Shimmer Painter ──────────────────────────────────────────────────────────
class _ShimmerPainter extends CustomPainter {
  final double position;
  _ShimmerPainter(this.position);

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..shader = LinearGradient(
        begin: Alignment(position - 0.5, 0),
        end: Alignment(position + 0.5, 0),
        colors: [
          Colors.transparent,
          Colors.white.withOpacity(0.08),
          Colors.transparent,
        ],
      ).createShader(Rect.fromLTWH(0, 0, size.width, size.height));
    canvas.drawRect(Rect.fromLTWH(0, 0, size.width, size.height), paint);
  }

  @override
  bool shouldRepaint(_ShimmerPainter old) => old.position != position;
}

// ─── QR Section ───────────────────────────────────────────────────────────────
class _QRSection extends StatelessWidget {
  final Map<String, dynamic> passport;
  final Animation<double> pulseAnim;
  const _QRSection(
      {required this.passport, required this.pulseAnim});

  @override
  Widget build(BuildContext context) {
    final passportId = passport['passport_id']?.toString() ??
        passport['id']?.toString() ?? 'LOYALTY-NEXUS';
    final qrData = 'loyalty-nexus://passport/$passportId';

    return NexusCard(
      child: Column(
        children: [
          const SectionHeader(title: 'Scan to Verify'),
          const SizedBox(height: 4),
          const Text(
              'Show this QR code at partner locations to earn bonus points',
              style: TextStyle(
                  color: NexusColors.textSecondary,
                  fontSize: 12,
                  height: 1.4),
              textAlign: TextAlign.center),
          const SizedBox(height: 20),
          AnimatedBuilder(
            animation: pulseAnim,
            builder: (_, child) => Transform.scale(
              scale: pulseAnim.value,
              child: child,
            ),
            child: Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(16),
                boxShadow: [
                  BoxShadow(
                    color: NexusColors.primary.withOpacity(0.2),
                    blurRadius: 20,
                    spreadRadius: 2,
                  ),
                ],
              ),
              child: QrImageView(
                data: qrData,
                version: QrVersions.auto,
                size: 180,
                backgroundColor: Colors.white,
                errorCorrectionLevel: QrErrorCorrectLevel.H,
              ),
            ),
          ),
          const SizedBox(height: 16),
          GestureDetector(
            onTap: () {
              Clipboard.setData(ClipboardData(text: passportId));
              ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
                content: Text('Passport ID copied'),
                backgroundColor: NexusColors.surfaceCard,
                behavior: SnackBarBehavior.floating,
              ));
            },
            child: Container(
              padding: const EdgeInsets.symmetric(
                  horizontal: 16, vertical: 8),
              decoration: BoxDecoration(
                color: NexusColors.primary.withOpacity(0.1),
                borderRadius: BorderRadius.circular(20),
                border: Border.all(
                    color: NexusColors.primary.withOpacity(0.2)),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    passportId.length > 20
                        ? '${passportId.substring(0, 20)}…'
                        : passportId,
                    style: const TextStyle(
                        color: NexusColors.primary,
                        fontWeight: FontWeight.w700,
                        fontSize: 13,
                        fontFamily: 'monospace'),
                  ),
                  const SizedBox(width: 8),
                  const Icon(Icons.copy_outlined,
                      color: NexusColors.primary, size: 14),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Tier Progress Card ───────────────────────────────────────────────────────
class _TierProgressCard extends StatelessWidget {
  final Map<String, dynamic> passport;
  const _TierProgressCard({required this.passport});

  @override
  Widget build(BuildContext context) {
    final tier         = passport['tier']?.toString() ?? 'BRONZE';
    final totalPoints  = passport['total_points'] as int? ?? 0;
    final nextTierPts  = passport['next_tier_points'] as int? ?? _nextTierDefault(tier);
    final progress     = nextTierPts > 0
        ? (totalPoints / nextTierPts).clamp(0.0, 1.0)
        : 1.0;
    final nextTier     = _nextTier(tier);
    final ptsNeeded    = (nextTierPts - totalPoints).clamp(0, nextTierPts);

    return NexusCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const SectionHeader(title: 'Tier Progress'),
              const Spacer(),
              TierBadge(tier: tier),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Text(_tierEmoji(tier),
                  style: const TextStyle(fontSize: 24)),
              const SizedBox(width: 8),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(tier.toUpperCase(),
                            style: const TextStyle(
                                color: NexusColors.textSecondary,
                                fontSize: 11,
                                fontWeight: FontWeight.w700)),
                        Text(nextTier.toUpperCase(),
                            style: const TextStyle(
                                color: NexusColors.textSecondary,
                                fontSize: 11,
                                fontWeight: FontWeight.w700)),
                      ],
                    ),
                    const SizedBox(height: 6),
                    ClipRRect(
                      borderRadius: BorderRadius.circular(4),
                      child: LinearProgressIndicator(
                        value: progress,
                        backgroundColor: NexusColors.border,
                        valueColor: AlwaysStoppedAnimation<Color>(
                            _tierColor(tier)),
                        minHeight: 8,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      nextTier != 'MAX'
                          ? '$ptsNeeded pts to $nextTier'
                          : 'Maximum tier reached! 🎉',
                      style: const TextStyle(
                          color: NexusColors.textSecondary,
                          fontSize: 12),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              Text(_tierEmoji(nextTier),
                  style: const TextStyle(fontSize: 24)),
            ],
          ),
          const SizedBox(height: 12),
          const NexusDivider(),
          const SizedBox(height: 12),
          // Tier benefits
          const Text('Your Tier Benefits',
              style: TextStyle(
                  color: NexusColors.textPrimary,
                  fontWeight: FontWeight.w700,
                  fontSize: 13)),
          const SizedBox(height: 8),
          ..._tierBenefits(tier).map((b) => Padding(
                padding: const EdgeInsets.only(bottom: 6),
                child: Row(
                  children: [
                    const Icon(Icons.check_circle,
                        color: NexusColors.green, size: 14),
                    const SizedBox(width: 8),
                    Text(b,
                        style: const TextStyle(
                            color: NexusColors.textSecondary,
                            fontSize: 13)),
                  ],
                ),
              )),
        ],
      ),
    );
  }

  String _tierEmoji(String tier) {
    switch (tier.toUpperCase()) {
      case 'GOLD':     return '🥇';
      case 'SILVER':   return '🥈';
      case 'PLATINUM': return '💎';
      case 'MAX':      return '👑';
      default:         return '🥉';
    }
  }

  Color _tierColor(String tier) {
    switch (tier.toUpperCase()) {
      case 'GOLD':     return NexusColors.gold;
      case 'SILVER':   return const Color(0xFFB0BEC5);
      case 'PLATINUM': return NexusColors.purple;
      default:         return const Color(0xFFCD7F32);
    }
  }

  String _nextTier(String tier) {
    switch (tier.toUpperCase()) {
      case 'BRONZE':   return 'SILVER';
      case 'SILVER':   return 'GOLD';
      case 'GOLD':     return 'PLATINUM';
      default:         return 'MAX';
    }
  }

  int _nextTierDefault(String tier) {
    switch (tier.toUpperCase()) {
      case 'BRONZE':   return 5000;
      case 'SILVER':   return 20000;
      case 'GOLD':     return 50000;
      default:         return 100000;
    }
  }

  List<String> _tierBenefits(String tier) {
    switch (tier.toUpperCase()) {
      case 'GOLD':
        return [
          '3x spin multiplier',
          'Priority prize draws',
          'Exclusive Gold challenges',
          'Dedicated support',
        ];
      case 'SILVER':
        return [
          '2x spin multiplier',
          'Silver-tier prize draws',
          'Bonus spin credits monthly',
        ];
      case 'PLATINUM':
        return [
          '5x spin multiplier',
          'VIP prize draws',
          'Personal account manager',
          'Early access to new features',
        ];
      default:
        return [
          '1x spin multiplier',
          'Standard prize draws',
          'Daily spin credits',
        ];
    }
  }
}

// ─── Streak Card ──────────────────────────────────────────────────────────────
class _StreakCard extends StatelessWidget {
  final Map<String, dynamic> passport;
  const _StreakCard({required this.passport});

  @override
  Widget build(BuildContext context) {
    final streak    = passport['streak_count'] as int? ??
        passport['current_streak'] as int? ?? 0;
    final bestStreak = passport['best_streak'] as int? ?? streak;
    final lastSpin  = _fmt(passport['last_spin_at']?.toString() ?? '');

    return NexusCard(
      child: Row(
        children: [
          Container(
            width: 56,
            height: 56,
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                colors: [Color(0xFFFF6B35), Color(0xFFFF4500)],
              ),
              borderRadius: BorderRadius.circular(16),
            ),
            child: const Center(
                child: Text('🔥', style: TextStyle(fontSize: 28))),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('Daily Streak',
                    style: TextStyle(
                        color: NexusColors.textSecondary,
                        fontSize: 12,
                        fontWeight: FontWeight.w600)),
                const SizedBox(height: 2),
                Text('$streak days',
                    style: const TextStyle(
                        color: NexusColors.textPrimary,
                        fontWeight: FontWeight.w900,
                        fontSize: 24,
                        fontFamily: 'Syne')),
                if (lastSpin.isNotEmpty)
                  Text('Last spin: $lastSpin',
                      style: const TextStyle(
                          color: NexusColors.textSecondary,
                          fontSize: 11)),
              ],
            ),
          ),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              const Text('BEST',
                  style: TextStyle(
                      color: NexusColors.textSecondary,
                      fontSize: 10,
                      fontWeight: FontWeight.w700,
                      letterSpacing: 0.5)),
              Text('$bestStreak',
                  style: const TextStyle(
                      color: NexusColors.gold,
                      fontWeight: FontWeight.w800,
                      fontSize: 20)),
              const Text('days',
                  style: TextStyle(
                      color: NexusColors.textSecondary, fontSize: 10)),
            ],
          ),
        ],
      ),
    );
  }

  String _fmt(String iso) {
    try {
      final dt  = DateTime.parse(iso).toLocal();
      final now = DateTime.now();
      final diff = now.difference(dt);
      if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
      if (diff.inHours < 24) return '${diff.inHours}h ago';
      return '${diff.inDays}d ago';
    } catch (_) {
      return '';
    }
  }
}

// ─── Wallet CTAs ──────────────────────────────────────────────────────────────
class _WalletCTAs extends ConsumerStatefulWidget {
  final Map<String, dynamic> passport;
  const _WalletCTAs({required this.passport});
  @override
  ConsumerState<_WalletCTAs> createState() => _WalletCTAsState();
}

class _WalletCTAsState extends ConsumerState<_WalletCTAs> {
  bool _appleLoading  = false;
  bool _googleLoading = false;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const SectionHeader(title: 'Add to Wallet'),
        const SizedBox(height: 4),
        Text(
          'Your passport stays on your lock screen and receives live updates.',
          style: TextStyle(
            color: NexusColors.textSecondary,
            fontSize: 12,
          ),
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: _WalletButton(
                icon: '🍎',
                label: 'Apple Wallet',
                color: const Color(0xFF1C1C1E),
                loading: _appleLoading,
                onTap: _addAppleWallet,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: _WalletButton(
                icon: '🤖',
                label: 'Google Wallet',
                color: const Color(0xFF1A73E8),
                loading: _googleLoading,
                onTap: _addGoogleWallet,
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        SizedBox(
          width: double.infinity,
          height: 48,
          child: OutlinedButton.icon(
            onPressed: _downloadPassport,
            icon: const Icon(Icons.download_outlined,
                color: NexusColors.primary),
            label: const Text('Download Passport PDF',
                style: TextStyle(
                    color: NexusColors.primary,
                    fontWeight: FontWeight.w700)),
            style: OutlinedButton.styleFrom(
              side: const BorderSide(color: NexusColors.primary),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12)),
            ),
          ),
        ),
      ],
    );
  }

  Future<void> _addAppleWallet() async {
    if (_appleLoading) return;
    setState(() => _appleLoading = true);
    try {
      final url = await ref.read(passportApiProvider).getApplePKPassURL();
      final uri = Uri.parse(url);
      if (await canLaunchUrl(uri)) {
        // iOS intercepts .pkpass URLs and opens Wallet automatically
        await launchUrl(uri, mode: LaunchMode.externalApplication);
      } else {
        _showError('Apple Wallet is not available on this device');
      }
    } catch (e) {
      _showError('Could not open Apple Wallet: ${e.toString()}');
    } finally {
      if (mounted) setState(() => _appleLoading = false);
    }
  }

  Future<void> _addGoogleWallet() async {
    if (_googleLoading) return;
    setState(() => _googleLoading = true);
    try {
      final urls = await ref.read(passportApiProvider).getWalletPassURLs();
      final googleUrl = urls['google_wallet_url']?.toString() ?? '';
      if (googleUrl.isEmpty) {
        _showError('Google Wallet is not available right now');
        return;
      }
      final uri = Uri.parse(googleUrl);
      if (await canLaunchUrl(uri)) {
        await launchUrl(uri, mode: LaunchMode.externalApplication);
      } else {
        _showError('Could not open Google Wallet');
      }
    } catch (e) {
      _showError('Could not open Google Wallet: ${e.toString()}');
    } finally {
      if (mounted) setState(() => _googleLoading = false);
    }
  }

  Future<void> _downloadPassport() async {
    try {
      final urls = await ref.read(passportApiProvider).getWalletPassURLs();
      final pdfUrl = urls['pdf_url']?.toString() ?? '';
      if (pdfUrl.isNotEmpty) {
        await launchUrl(Uri.parse(pdfUrl),
            mode: LaunchMode.externalApplication);
      } else {
        _showError('PDF download is not available right now');
      }
    } catch (e) {
      _showError('Could not download passport');
    }
  }

  void _showError(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(message),
      backgroundColor: NexusColors.surfaceCard,
      behavior: SnackBarBehavior.floating,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
    ));
  }
}

class _WalletButton extends StatelessWidget {
  final String icon;
  final String label;
  final Color color;
  final VoidCallback onTap;
  final bool loading;
  const _WalletButton({
    required this.icon,
    required this.label,
    required this.color,
    required this.onTap,
    this.loading = false,
  });
  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: loading ? null : onTap,
      child: AnimatedOpacity(
        opacity: loading ? 0.7 : 1.0,
        duration: const Duration(milliseconds: 200),
        child: Container(
          height: 52,
          decoration: BoxDecoration(
            color: color,
            borderRadius: BorderRadius.circular(12),
            boxShadow: [
              BoxShadow(
                color: color.withOpacity(0.3),
                blurRadius: 12,
                offset: const Offset(0, 4),
              ),
            ],
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              if (loading)
                const SizedBox(
                  width: 18, height: 18,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: Colors.white,
                  ),
                )
              else
                Text(icon, style: const TextStyle(fontSize: 18)),
              const SizedBox(width: 8),
              Text(label,
                  style: const TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.w700,
                      fontSize: 13)),
            ],
          ),
        ),
      ),
    );
  }
}

// ─── Activity Section ───────────────────────────────────────────────────────────────
class _ActivitySection extends ConsumerWidget {
  const _ActivitySection();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final eventsAsync = ref.watch(_passportEventsProvider);

    return NexusCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SectionHeader(title: 'Activity History'),
          const SizedBox(height: 4),
          const Text(
            'Your recent passport activity and milestones',
            style: TextStyle(
                color: NexusColors.textSecondary,
                fontSize: 12,
                height: 1.4),
          ),
          const SizedBox(height: 16),
          eventsAsync.when(
            loading: () => Column(
              children: List.generate(3, (_) =>
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: ShimmerBox(height: 52, radius: 10),
                ),
              ),
            ),
            error: (_, __) => const Center(
              child: Padding(
                padding: EdgeInsets.symmetric(vertical: 16),
                child: Text(
                  'Could not load activity',
                  style: TextStyle(color: NexusColors.textSecondary, fontSize: 13),
                ),
              ),
            ),
            data: (events) {
              if (events.isEmpty) {
                return const Center(
                  child: Padding(
                    padding: EdgeInsets.symmetric(vertical: 24),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text('🏆', style: TextStyle(fontSize: 36)),
                        SizedBox(height: 8),
                        Text(
                          'No activity yet',
                          style: TextStyle(
                              color: NexusColors.textSecondary,
                              fontWeight: FontWeight.w600),
                        ),
                        SizedBox(height: 4),
                        Text(
                          'Earn points and spin to see your history here',
                          style: TextStyle(
                              color: NexusColors.textSecondary,
                              fontSize: 12),
                          textAlign: TextAlign.center,
                        ),
                      ],
                    ),
                  ),
                );
              }
              return Column(
                children: events.take(10).map((e) {
                  final event = e as Map<String, dynamic>;
                  return _ActivityItem(event: event);
                }).toList(),
              );
            },
          ),
        ],
      ),
    );
  }
}

class _ActivityItem extends StatelessWidget {
  final Map<String, dynamic> event;
  const _ActivityItem({required this.event});

  @override
  Widget build(BuildContext context) {
    final eventType = event['event_type']?.toString() ?? '';
    final detail    = event['detail']?.toString() ?? '';
    final createdAt = event['created_at']?.toString() ?? '';
    final (icon, color) = _eventStyle(eventType);

    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: color.withOpacity(0.12),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Center(
              child: Text(icon, style: const TextStyle(fontSize: 18)),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  _eventLabel(eventType),
                  style: const TextStyle(
                      color: NexusColors.textPrimary,
                      fontWeight: FontWeight.w600,
                      fontSize: 13),
                ),
                if (detail.isNotEmpty)
                  Text(
                    detail,
                    style: const TextStyle(
                        color: NexusColors.textSecondary,
                        fontSize: 11),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
              ],
            ),
          ),
          Text(
            _timeAgo(createdAt),
            style: const TextStyle(
                color: NexusColors.textSecondary,
                fontSize: 11),
          ),
        ],
      ),
    );
  }

  (String, Color) _eventStyle(String type) {
    switch (type) {
      case 'tier_upgrade':   return ('🏆', NexusColors.gold);
      case 'badge_earned':   return ('🏅', NexusColors.purple);
      case 'qr_scan':        return ('📱', NexusColors.primary);
      case 'streak_milestone': return ('🔥', const Color(0xFFFF6B35));
      case 'wallet_install': return ('💳', const Color(0xFF1A73E8));
      default:               return ('✨', NexusColors.textSecondary);
    }
  }

  String _eventLabel(String type) {
    switch (type) {
      case 'tier_upgrade':     return 'Tier Upgraded';
      case 'badge_earned':     return 'Badge Earned';
      case 'qr_scan':          return 'QR Code Scanned';
      case 'streak_milestone': return 'Streak Milestone';
      case 'wallet_install':   return 'Wallet Pass Installed';
      default:                 return type.replaceAll('_', ' ').toUpperCase();
    }
  }

  String _timeAgo(String iso) {
    try {
      final dt   = DateTime.parse(iso).toLocal();
      final diff = DateTime.now().difference(dt);
      if (diff.inMinutes < 1)  return 'just now';
      if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
      if (diff.inHours < 24)   return '${diff.inHours}h ago';
      if (diff.inDays < 7)     return '${diff.inDays}d ago';
      return '${(diff.inDays / 7).floor()}w ago';
    } catch (_) {
      return '';
    }
  }
}

// ─── Skeleton ─────────────────────────────────────────────────────────────────
class _PassportSkeleton extends StatelessWidget {
  const _PassportSkeleton();

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        ShimmerBox(height: 200, radius: 20),
        const SizedBox(height: 20),
        ShimmerBox(height: 280, radius: 16),
        const SizedBox(height: 20),
        ShimmerBox(height: 160, radius: 16),
        const SizedBox(height: 20),
        ShimmerBox(height: 80, radius: 16),
      ],
    );
  }
}
