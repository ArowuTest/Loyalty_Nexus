import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Confetti Particle ────────────────────────────────────────────────────────

class _Particle {
  final double x, y, dx, dy;
  final Color color;
  const _Particle({
    required this.x,
    required this.y,
    required this.dx,
    required this.dy,
    required this.color,
  });
}

// ─── Confetti Painter ─────────────────────────────────────────────────────────

class _ConfettiPainter extends CustomPainter {
  final Animation<double> animation;
  final List<_Particle> particles;

  _ConfettiPainter(this.animation, this.particles) : super(repaint: animation);

  @override
  void paint(Canvas canvas, Size size) {
    final progress = animation.value;
    final paint    = Paint();
    for (final p in particles) {
      final x       = p.x * size.width  + p.dx * progress * 120;
      final y       = p.y * size.height + p.dy * progress * 300;
      final opacity = (1.0 - progress).clamp(0.0, 1.0);
      paint.color = p.color.withValues(alpha: opacity);
      canvas.drawCircle(Offset(x, y), 5.5, paint);
    }
  }

  @override
  bool shouldRepaint(_ConfettiPainter old) => true;
}

// ─── Reward Data ──────────────────────────────────────────────────────────────

class _RewardData {
  final int  pointsEarned;
  final bool spinEligible;
  final int  drawEntries;
  const _RewardData({
    required this.pointsEarned,
    required this.spinEligible,
    required this.drawEntries,
  });
}

// ─── Screen ───────────────────────────────────────────────────────────────────

class RechargeSuccessScreen extends ConsumerStatefulWidget {
  final String? reference;
  const RechargeSuccessScreen({super.key, this.reference});

  @override
  ConsumerState<RechargeSuccessScreen> createState() =>
      _RechargeSuccessScreenState();
}

class _RechargeSuccessScreenState extends ConsumerState<RechargeSuccessScreen>
    with TickerProviderStateMixin {

  // Theme colours
  static const _gold  = Color(0xFFF5A623);
  static const _green = Color(0xFF4CAF50);
  static const _blue  = Color(0xFF2196F3);

  // Animation controllers
  late final AnimationController _confettiCtrl;
  late final AnimationController _rewardCtrl;

  // Reward card animations
  late final Animation<double> _rewardSlide;
  late final Animation<double> _rewardFade;

  // Particles
  late final List<_Particle> _particles;

  // Reward data
  _RewardData? _rewardData;
  bool         _loadingReward = false;

  // ── Init ──────────────────────────────────────────────────────────────────
  @override
  void initState() {
    super.initState();

    // Confetti: plays once over 2.5 s
    _confettiCtrl = AnimationController(
      vsync:    this,
      duration: const Duration(milliseconds: 2500),
    )..forward();

    // Reward card: slides up + fades in over 600 ms
    _rewardCtrl = AnimationController(
      vsync:    this,
      duration: const Duration(milliseconds: 600),
    );
    _rewardSlide = Tween<double>(begin: 60, end: 0).animate(
      CurvedAnimation(parent: _rewardCtrl, curve: Curves.easeOut),
    );
    _rewardFade = CurvedAnimation(parent: _rewardCtrl, curve: Curves.easeIn);

    // Generate 20 particles
    final rng    = Random();
    final colors = [_gold, _green, _blue, Colors.white70, Colors.pinkAccent];
    _particles   = List.generate(20, (_) => _Particle(
      x:     0.2 + rng.nextDouble() * 0.6,
      y:     0.3 + rng.nextDouble() * 0.3,
      dx:    (rng.nextDouble() - 0.5) * 2,
      dy:    -(0.5 + rng.nextDouble()),
      color: colors[rng.nextInt(colors.length)],
    ));

    // Fetch reward data (300 ms delay so the success UI renders first)
    Future.delayed(const Duration(milliseconds: 300), _fetchReward);
  }

  // ── Fetch reward data ─────────────────────────────────────────────────────
  Future<void> _fetchReward() async {
    if (widget.reference == null || !mounted) return;
    setState(() => _loadingReward = true);
    try {
      final dio  = ref.read(dioProvider);
      final resp = await dio.get<Map<String, dynamic>>(
        '/recharge/status/${widget.reference}',
      );
      final data = resp.data ?? {};
      if (!mounted) return;
      setState(() {
        _rewardData = _RewardData(
          pointsEarned: (data['points_earned'] as num?)?.toInt() ?? 0,
          spinEligible: data['spin_eligible'] == true,
          drawEntries:  (data['draw_entries']  as num?)?.toInt() ?? 0,
        );
        _loadingReward = false;
      });
      // Start card animation 500 ms after data arrives
      await Future.delayed(const Duration(milliseconds: 500));
      if (mounted) _rewardCtrl.forward();
    } catch (_) {
      if (mounted) setState(() => _loadingReward = false);
    }
  }

  // ── Dispose ───────────────────────────────────────────────────────────────
  @override
  void dispose() {
    _confettiCtrl.dispose();
    _rewardCtrl.dispose();
    super.dispose();
  }

  // ── Build ─────────────────────────────────────────────────────────────────
  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authStateProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      body: Stack(
        children: [
          // Confetti layer (full screen, behind content)
          Positioned.fill(
            child: CustomPaint(
              painter: _ConfettiPainter(_confettiCtrl, _particles),
            ),
          ),

          // Main scrollable content
          SafeArea(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                children: [
                  const Spacer(flex: 2),

                  // Success icon
                  Container(
                    width:  100,
                    height: 100,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color:  _green.withValues(alpha: 0.12),
                      border: Border.all(color: _green, width: 2),
                    ),
                    child: const Icon(Icons.check_rounded, color: _green, size: 52),
                  ),
                  const SizedBox(height: 24),

                  const Text(
                    'Payment Initiated!',
                    style: TextStyle(
                      color:      Colors.white,
                      fontSize:   26,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    'Your recharge is being processed.\nTop-up usually completes in under 30 seconds.',
                    textAlign: TextAlign.center,
                    style: TextStyle(
                      color:    Colors.white.withValues(alpha: 0.6),
                      fontSize: 15,
                      height:   1.5,
                    ),
                  ),

                  const SizedBox(height: 28),

                  // Double-points callout
                  Container(
                    padding:    const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color:        NexusColors.goldDim,
                      borderRadius: BorderRadius.circular(16),
                      border:       Border.all(
                        color: NexusColors.gold.withValues(alpha: 0.4),
                      ),
                    ),
                    child: const Row(
                      children: [
                        Text('\u26A1', style: TextStyle(fontSize: 24)),
                        SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                'DOUBLE POINTS INCOMING',
                                style: TextStyle(
                                  color:         NexusColors.gold,
                                  fontSize:      13,
                                  fontWeight:    FontWeight.w800,
                                  letterSpacing: 0.5,
                                ),
                              ),
                              SizedBox(height: 4),
                              Text(
                                'You earn Pulse Points from your Paystack payment PLUS MTN will award additional points when the recharge hits your line.',
                                style: TextStyle(
                                  color:    Colors.white70,
                                  fontSize: 12,
                                  height:   1.4,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),

                  // Reference chip
                  if (widget.reference != null) ...[
                    const SizedBox(height: 20),
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 10),
                      decoration: BoxDecoration(
                        color:        NexusColors.surface,
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Icon(Icons.receipt_long_rounded,
                              color: Colors.white38, size: 16),
                          const SizedBox(width: 8),
                          Text(
                            'Ref: ${widget.reference}',
                            style: const TextStyle(
                              color:      Colors.white54,
                              fontSize:   12,
                              fontFamily: 'monospace',
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],

                  const SizedBox(height: 28),

                  // Reward summary card (animated slide-up + fade)
                  if (_rewardData != null)
                    AnimatedBuilder(
                      animation: _rewardCtrl,
                      builder: (_, __) => Transform.translate(
                        offset: Offset(0, _rewardSlide.value),
                        child: Opacity(
                          opacity: _rewardFade.value,
                          child:   _buildRewardCard(_rewardData!),
                        ),
                      ),
                    )
                  else if (_loadingReward)
                    const SizedBox(
                      width:  24,
                      height: 24,
                      child:  CircularProgressIndicator(
                        color:       _gold,
                        strokeWidth: 2,
                      ),
                    ),

                  const Spacer(flex: 3),

                  // CTA buttons
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      if (auth.isAuthenticated)
                        ElevatedButton(
                          onPressed: () => context.go('/dashboard'),
                          style: ElevatedButton.styleFrom(
                            backgroundColor: NexusColors.primary,
                            foregroundColor: Colors.white,
                            minimumSize:     const Size.fromHeight(52),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(14),
                            ),
                            elevation: 0,
                          ),
                          child: const Text(
                            'View My Points',
                            style: TextStyle(
                              fontWeight: FontWeight.w700,
                              fontSize:   16,
                            ),
                          ),
                        )
                      else
                        ElevatedButton(
                          onPressed: () => context.go('/'),
                          style: ElevatedButton.styleFrom(
                            backgroundColor: NexusColors.primary,
                            foregroundColor: Colors.white,
                            minimumSize:     const Size.fromHeight(52),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(14),
                            ),
                            elevation: 0,
                          ),
                          child: const Text(
                            'Sign In to Track Your Points',
                            style: TextStyle(
                              fontWeight: FontWeight.w700,
                              fontSize:   16,
                            ),
                          ),
                        ),

                      const SizedBox(height: 12),

                      OutlinedButton(
                        onPressed: () => context.go('/recharge'),
                        style: OutlinedButton.styleFrom(
                          foregroundColor: Colors.white70,
                          side:            const BorderSide(color: Colors.white24),
                          minimumSize:     const Size.fromHeight(48),
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(14),
                          ),
                        ),
                        child: const Text(
                          'Recharge Again',
                          style: TextStyle(
                            fontWeight: FontWeight.w600,
                            fontSize:   15,
                          ),
                        ),
                      ),
                    ],
                  ),

                  const SizedBox(height: 8),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  // ── Reward card ───────────────────────────────────────────────────────────
  Widget _buildRewardCard(_RewardData data) {
    return Container(
      width:      double.infinity,
      padding:    const EdgeInsets.symmetric(horizontal: 20, vertical: 18),
      decoration: BoxDecoration(
        color:        const Color(0xFF1A1A2E),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
          color: _gold.withValues(alpha: 0.45),
          width: 1.5,
        ),
        boxShadow: [
          BoxShadow(
            color:       _gold.withValues(alpha: 0.12),
            blurRadius:  20,
            spreadRadius: 2,
          ),
        ],
      ),
      child: Column(
        children: [
          Text(
            '\uD83C\uDF89 Your Rewards',
            style: TextStyle(
              color:    Colors.white.withValues(alpha: 0.7),
              fontSize: 13,
            ),
          ),
          const SizedBox(height: 14),

          if (data.pointsEarned > 0) ...[
            Text(
              '+${data.pointsEarned}',
              style: const TextStyle(
                color:      _gold,
                fontSize:   52,
                fontWeight: FontWeight.bold,
                height:     1.0,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              'Pulse Points',
              style: TextStyle(
                color:    Colors.white.withValues(alpha: 0.55),
                fontSize: 14,
              ),
            ),
            const SizedBox(height: 14),
          ],

          if (data.drawEntries > 0) ...[
            _rewardChip(
              Icons.confirmation_number_outlined,
              _blue,
              '${data.drawEntries} Draw '
              '${data.drawEntries == 1 ? "Entry" : "Entries"}',
            ),
            const SizedBox(height: 8),
          ],

          if (data.spinEligible)
            _rewardChip(Icons.casino_outlined, _gold, 'Spin Eligible!'),
        ],
      ),
    );
  }

  Widget _rewardChip(IconData icon, Color color, String label) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        color:        color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(22),
        border: Border.all(color: color.withValues(alpha: 0.40)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, color: color, size: 16),
          const SizedBox(width: 7),
          Text(
            label,
            style: TextStyle(
              color:      color,
              fontWeight: FontWeight.w700,
              fontSize:   13,
            ),
          ),
        ],
      ),
    );
  }
}
