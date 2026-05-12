/// nexus_gamification.dart
/// ─────────────────────────────────────────────────────────────────────────────
/// Shared gamification UI primitives used across the Loyalty Nexus app.
///
/// Exports:
///  • PointsEarnedOverlay  — full-screen overlay that slides in "+X pts"
///  • PointsCounter        — animated number counter widget
///  • TierProgressSection  — progress bar + next-milestone callout
///  • StreakBadge           — fire-emoji streak display
///  • NexusSkeleton        — shimmer skeleton placeholder
///  • showPointsEarned()   — convenience function to show the overlay
/// ─────────────────────────────────────────────────────────────────────────────
import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:shimmer/shimmer.dart';

import '../theme/nexus_theme.dart';

// ═══════════════════════════════════════════════════════════════════════════════
// POINTS EARNED OVERLAY
// ═══════════════════════════════════════════════════════════════════════════════

/// Shows a translucent overlay with "+N Pulse Points" flying in from the bottom.
/// Call via [showPointsEarned] convenience function.
Future<void> showPointsEarned(
  BuildContext context, {
  required int points,
  String label = 'Pulse Points Earned!',
  bool isDouble = false,
}) async {
  if (!context.mounted) return;
  HapticFeedback.heavyImpact();
  showGeneralDialog<void>(
    context:      context,
    barrierDismissible: true,
    barrierLabel: '',
    barrierColor: Colors.black.withValues(alpha: 0.45),
    transitionDuration: const Duration(milliseconds: 400),
    pageBuilder:  (ctx, anim1, anim2) => const SizedBox.shrink(),
    transitionBuilder: (ctx, anim1, anim2, child) {
      return _PointsOverlayContent(
        points:   points,
        label:    label,
        isDouble: isDouble,
        animation: anim1,
      );
    },
  );
}

class _PointsOverlayContent extends StatefulWidget {
  final int points;
  final String label;
  final bool isDouble;
  final Animation<double> animation;

  const _PointsOverlayContent({
    required this.points,
    required this.label,
    required this.isDouble,
    required this.animation,
  });

  @override
  State<_PointsOverlayContent> createState() => _PointsOverlayContentState();
}

class _PointsOverlayContentState extends State<_PointsOverlayContent>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late final Animation<double> _scale;
  late final Animation<double> _slideY;
  late final Animation<double> _opacity;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync:    this,
      duration: const Duration(milliseconds: 600),
    )..forward();

    _scale   = CurvedAnimation(parent: _ctrl, curve: Curves.elasticOut);
    _slideY  = Tween<double>(begin: 80, end: 0)
        .animate(CurvedAnimation(parent: _ctrl, curve: Curves.easeOutCubic));
    _opacity = CurvedAnimation(parent: _ctrl, curve: const Interval(0, 0.4));

    // Auto-dismiss after 2.2 s
    Future.delayed(const Duration(milliseconds: 2200), () {
      if (mounted && Navigator.canPop(context)) Navigator.pop(context);
    });
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _ctrl,
      builder: (_, __) {
        return Align(
          alignment: const Alignment(0, 0.3),
          child: Opacity(
            opacity: _opacity.value,
            child: Transform.translate(
              offset: Offset(0, _slideY.value),
              child: ScaleTransition(
                scale: _scale,
                child: Material(
                  color:        Colors.transparent,
                  borderRadius: BorderRadius.circular(24),
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 36, vertical: 28),
                    decoration: BoxDecoration(
                      color: NexusColors.surfaceHigh,
                      borderRadius: BorderRadius.circular(24),
                      border: Border.all(color: NexusColors.goldDim, width: 1.5),
                      boxShadow: [BoxShadow(color: NexusColors.gold.withValues(alpha: 0.35), blurRadius: 20, spreadRadius: 0)],
                    ),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        // Particles row
                        const _ParticleRow(),
                        const SizedBox(height: 8),

                        // Points number
                        Text(
                          '+${widget.points}',
                          style: const TextStyle(
                            color:      NexusColors.gold,
                            fontSize:   52,
                            fontWeight: FontWeight.w900,
                            letterSpacing: -1,
                            height: 1,
                          ),
                        ),

                        const SizedBox(height: 4),
                        Text(
                          widget.label,
                          style: const TextStyle(
                            color:      Colors.white,
                            fontSize:   16,
                            fontWeight: FontWeight.w600,
                          ),
                        ),

                        if (widget.isDouble) ...[
                          const SizedBox(height: 12),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 14, vertical: 6),
                            decoration: BoxDecoration(
                              color:        NexusColors.goldDim,
                              borderRadius: BorderRadius.circular(20),
                            ),
                            child: const Text(
                              '⚡ DOUBLE POINTS ACTIVE',
                              style: TextStyle(
                                color:         NexusColors.gold,
                                fontSize:      11,
                                fontWeight:    FontWeight.w800,
                                letterSpacing: 0.5,
                              ),
                            ),
                          ),
                        ],

                        const SizedBox(height: 4),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

// Tiny confetti particles above the number
class _ParticleRow extends StatefulWidget {
  const _ParticleRow();
  @override State<_ParticleRow> createState() => _ParticleRowState();
}

class _ParticleRowState extends State<_ParticleRow>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  final _rng = math.Random();

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync:    this,
      duration: const Duration(milliseconds: 900),
    )..forward();
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  static final _colors = [
    NexusColors.gold, NexusColors.primary, const Color(0xFF10b981),
    const Color(0xFFf43f5e), const Color(0xFFa78bfa),
  ];

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 32,
      width:  180,
      child: AnimatedBuilder(
        animation: _ctrl,
        builder: (_, __) {
          return CustomPaint(
            painter: _ParticlePainter(_ctrl.value, _rng, _colors),
          );
        },
      ),
    );
  }
}

class _ParticlePainter extends CustomPainter {
  final double t;
  final math.Random rng;
  final List<Color> colors;

  _ParticlePainter(this.t, this.rng, this.colors);

  @override
  void paint(Canvas canvas, Size size) {
    final seed = 42; // deterministic layout
    final r = math.Random(seed);
    for (var i = 0; i < 18; i++) {
      final startX  = r.nextDouble() * size.width;
      final startY  = size.height;
      final spreadX = (r.nextDouble() - 0.5) * 60;
      final height  = r.nextDouble() * size.height;
      final x = startX + spreadX * t;
      final y = startY - height * t;
      final opacity = (1.0 - t).clamp(0.0, 1.0);
      final radius  = 3.0 + r.nextDouble() * 4;
      final paint   = Paint()
        ..color  = colors[i % colors.length].withValues(alpha: opacity)
        ..style  = PaintingStyle.fill;
      canvas.drawCircle(Offset(x, y), radius, paint);
    }
  }

  @override
  bool shouldRepaint(_ParticlePainter old) => old.t != t;
}

// ═══════════════════════════════════════════════════════════════════════════════
// ANIMATED POINTS COUNTER
// ═══════════════════════════════════════════════════════════════════════════════

/// Animates from [oldValue] to [newValue] over [duration].
/// Displays with a compact format (1.2k, 45k, etc.)
class PointsCounter extends StatefulWidget {
  final int value;
  final TextStyle? style;
  final Duration duration;
  final String Function(int)? formatter;

  const PointsCounter({
    super.key,
    required this.value,
    this.style,
    this.duration = const Duration(milliseconds: 1200),
    this.formatter,
  });

  @override
  State<PointsCounter> createState() => _PointsCounterState();
}

class _PointsCounterState extends State<PointsCounter>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late Animation<double> _anim;
  int _from = 0;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(vsync: this, duration: widget.duration);
    _anim = Tween<double>(begin: 0, end: widget.value.toDouble())
        .animate(CurvedAnimation(parent: _ctrl, curve: Curves.easeOutCubic));
    _ctrl.forward();
  }

  @override
  void didUpdateWidget(PointsCounter old) {
    super.didUpdateWidget(old);
    if (old.value != widget.value) {
      _from = old.value;
      _anim = Tween<double>(
        begin: _from.toDouble(),
        end:   widget.value.toDouble(),
      ).animate(CurvedAnimation(parent: _ctrl, curve: Curves.easeOutCubic));
      _ctrl
        ..reset()
        ..forward();
    }
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  String _fmt(int v) {
    if (widget.formatter != null) return widget.formatter!(v);
    if (v >= 1000000) return '${(v / 1000000).toStringAsFixed(1)}M';
    if (v >= 1000)    return '${(v / 1000).toStringAsFixed(1)}k';
    return v.toString();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _anim,
      builder: (_, __) {
        return Text(
          _fmt(_anim.value.round()),
          style: widget.style ??
              const TextStyle(
                color:      Colors.white,
                fontSize:   36,
                fontWeight: FontWeight.w800,
                letterSpacing: -1,
              ),
        );
      },
    );
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// TIER PROGRESS SECTION (promoted to full-width card on dashboard)
// ═══════════════════════════════════════════════════════════════════════════════

class TierProgressSection extends StatelessWidget {
  final int lifetimePoints;
  final String tier;
  final int? spinCredits;

  const TierProgressSection({
    super.key,
    required this.lifetimePoints,
    required this.tier,
    this.spinCredits,
  });

  static const _thresholds = {
    'BRONZE': 0, 'SILVER': 2000, 'GOLD': 10000, 'PLATINUM': 50000,
  };
  static const _nextTier = {
    'BRONZE': 'SILVER', 'SILVER': 'GOLD', 'GOLD': 'PLATINUM',
  };
  static const _tierEmoji = {
    'BRONZE': '🥉', 'SILVER': '🥈', 'GOLD': '🥇', 'PLATINUM': '💎',
  };

  String? get _next => _nextTier[tier.toUpperCase()];
  int get _currentMin => _thresholds[tier.toUpperCase()] ?? 0;
  int? get _nextMin => _next != null ? _thresholds[_next!] : null;

  double get _progress {
    if (_next == null) return 1.0;
    final range = (_nextMin! - _currentMin).clamp(1, 999999);
    return ((lifetimePoints - _currentMin) / range).clamp(0.0, 1.0);
  }

  int get _ptsToNext =>
      _nextMin != null ? (_nextMin! - lifetimePoints).clamp(0, 999999) : 0;

  /// How many more ₦250 recharges to reach next tier
  int get _rechargesNeeded => (_ptsToNext / 1).ceil(); // 1 pt per ₦250

  /// Milestone call-to-action message
  String get _ctaMessage {
    if (_next == null) return 'You\'ve reached the top tier! 👑';
    if (_ptsToNext <= 50) {
      return '$_ptsToNext pts to ${_tierEmoji[_next]} $_next — almost there!';
    }
    if (_ptsToNext <= 500) {
      return 'Recharge $_rechargesNeeded more times to reach $_next';
    }
    return '$_ptsToNext pts to unlock ${_tierEmoji[_next]} $_next';
  }

  @override
  Widget build(BuildContext context) {
    final next  = _next;
    final emoji = _tierEmoji[tier.toUpperCase()] ?? '🥉';

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color:        NexusColors.surface,
        borderRadius: BorderRadius.circular(16),
        border:       Border.all(color: Colors.white12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header row
          Row(
            children: [
              Text(emoji, style: const TextStyle(fontSize: 20)),
              const SizedBox(width: 8),
              Text(
                tier.toUpperCase(),
                style: const TextStyle(
                  color:      Colors.white,
                  fontWeight: FontWeight.w800,
                  fontSize:   14,
                  letterSpacing: 1,
                ),
              ),
              const Spacer(),
              if (next != null)
                Text(
                  '${_tierEmoji[next]} $next',
                  style: const TextStyle(
                    color:      Colors.white54,
                    fontSize:   12,
                    fontWeight: FontWeight.w600,
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),

          // Animated progress bar
          _AnimatedProgressBar(progress: _progress),
          const SizedBox(height: 8),

          // Points label
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                '${_fmtPts(lifetimePoints)} pts',
                style: const TextStyle(
                  color:      Colors.white70,
                  fontSize:   12,
                  fontWeight: FontWeight.w600,
                ),
              ),
              if (next != null)
                Text(
                  '${_fmtPts(_nextMin!)} pts',
                  style: const TextStyle(color: Colors.white38, fontSize: 12),
                ),
            ],
          ),
          const SizedBox(height: 12),

          // CTA callout
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color:        NexusColors.goldDim,
              borderRadius: BorderRadius.circular(10),
            ),
            child: Row(
              children: [
                const Text('⚡', style: TextStyle(fontSize: 13)),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    _ctaMessage,
                    style: const TextStyle(
                      color:      NexusColors.gold,
                      fontSize:   12,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  String _fmtPts(int v) {
    if (v >= 1000) return '${(v / 1000).toStringAsFixed(0)}k';
    return v.toString();
  }
}

class _AnimatedProgressBar extends StatefulWidget {
  final double progress;
  const _AnimatedProgressBar({required this.progress});
  @override State<_AnimatedProgressBar> createState() => _AnimatedProgressBarState();
}

class _AnimatedProgressBarState extends State<_AnimatedProgressBar>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late Animation<double> _anim;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync:    this,
      duration: const Duration(milliseconds: 1000),
    );
    _anim = Tween<double>(begin: 0, end: widget.progress)
        .animate(CurvedAnimation(parent: _ctrl, curve: Curves.easeOutCubic));
    _ctrl.forward();
  }

  @override
  void didUpdateWidget(_AnimatedProgressBar old) {
    super.didUpdateWidget(old);
    if (old.progress != widget.progress) {
      _anim = Tween<double>(begin: old.progress, end: widget.progress)
          .animate(CurvedAnimation(parent: _ctrl, curve: Curves.easeOutCubic));
      _ctrl..reset()..forward();
    }
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _anim,
      builder: (_, __) {
        return ClipRRect(
          borderRadius: BorderRadius.circular(6),
          child: Stack(
            children: [
              // Track
              Container(
                height: 10,
                color: Colors.white.withValues(alpha: 0.08),
              ),
              // Fill with shimmer
              FractionallySizedBox(
                widthFactor: _anim.value.clamp(0.0, 1.0),
                child: Container(
                  height: 10,
                  decoration: const BoxDecoration(
                    gradient: LinearGradient(
                      colors: [NexusColors.primary, NexusColors.gold],
                    ),
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// STREAK BADGE
// ═══════════════════════════════════════════════════════════════════════════════

class StreakBadge extends StatelessWidget {
  final int streak;
  final bool compact;

  const StreakBadge({super.key, required this.streak, this.compact = false});

  @override
  Widget build(BuildContext context) {
    if (streak == 0) return const SizedBox.shrink();

    return Container(
      padding: compact
          ? const EdgeInsets.symmetric(horizontal: 8, vertical: 4)
          : const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [
            const Color(0xFFFF6B35).withValues(alpha: 0.25),
            const Color(0xFFFF9F1C).withValues(alpha: 0.15),
          ],
        ),
        borderRadius: BorderRadius.circular(compact ? 8 : 12),
        border: Border.all(
          color: const Color(0xFFFF6B35).withValues(alpha: 0.5),
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '🔥',
            style: TextStyle(fontSize: compact ? 14 : 18),
          )
              .animate(onPlay: (c) => c.repeat())
              .scale(
                  duration: const Duration(milliseconds: 800),
                  begin:    const Offset(1, 1),
                  end:      const Offset(1.15, 1.15),
                  curve:    Curves.easeInOut)
              .then()
              .scale(
                  duration: const Duration(milliseconds: 800),
                  begin:    const Offset(1.15, 1.15),
                  end:      const Offset(1, 1),
                  curve:    Curves.easeInOut),
          SizedBox(width: compact ? 4 : 6),
          Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '$streak ${streak == 1 ? "day" : "days"}',
                style: TextStyle(
                  color:      const Color(0xFFFF9F1C),
                  fontSize:   compact ? 12 : 14,
                  fontWeight: FontWeight.w800,
                  height:     1,
                ),
              ),
              if (!compact)
                const Text(
                  'streak',
                  style: TextStyle(
                    color:    Color(0xFFFF6B35),
                    fontSize: 10,
                    fontWeight: FontWeight.w600,
                    height: 1,
                  ),
                ),
            ],
          ),
        ],
      ),
    );
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// SKELETON SHIMMER PRIMITIVES
// ═══════════════════════════════════════════════════════════════════════════════

class NexusSkeleton extends StatelessWidget {
  final double width;
  final double height;
  final double radius;
  final bool circle;

  const NexusSkeleton({
    super.key,
    required this.width,
    required this.height,
    this.radius = 8,
    this.circle = false,
  });

  const NexusSkeleton.text({
    super.key,
    this.width = double.infinity,
    this.height = 14,
    this.radius = 6,
    this.circle = false,
  });

  const NexusSkeleton.circle({
    super.key,
    required double size,
  })  : width  = size,
        height = size,
        radius = size / 2,
        circle = true;

  @override
  Widget build(BuildContext context) {
    return Shimmer.fromColors(
      baseColor:      const Color(0xFF1C2444),
      highlightColor: const Color(0xFF2A3560),
      child: Container(
        width:  width,
        height: height,
        decoration: BoxDecoration(
          color:        const Color(0xFF1C2444),
          borderRadius: circle
              ? BorderRadius.circular(width / 2)
              : BorderRadius.circular(radius),
        ),
      ),
    );
  }
}

// ── Pre-built skeleton layouts matching real screens ─────────────────────────

class WalletCardSkeleton extends StatelessWidget {
  const WalletCardSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(22),
      decoration: BoxDecoration(
        color:        NexusColors.surface,
        borderRadius: BorderRadius.circular(24),
        border:       Border.all(color: Colors.white12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                const NexusSkeleton.text(width: 80, height: 11),
                const SizedBox(height: 8),
                const NexusSkeleton(width: 140, height: 40),
              ]),
              const NexusSkeleton(width: 70, height: 28, radius: 14),
            ],
          ),
          const SizedBox(height: 18),
          Row(children: const [
            Expanded(child: NexusSkeleton(width: double.infinity, height: 36)),
            SizedBox(width: 12),
            Expanded(child: NexusSkeleton(width: double.infinity, height: 36)),
          ]),
          const SizedBox(height: 14),
          const NexusSkeleton(width: double.infinity, height: 6, radius: 3),
        ],
      ),
    );
  }
}

class ListItemSkeleton extends StatelessWidget {
  final int count;
  const ListItemSkeleton({super.key, this.count = 3});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: List.generate(count, (i) => Padding(
        padding: const EdgeInsets.only(bottom: 12),
        child: Row(children: [
          const NexusSkeleton.circle(size: 40),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: const [
                NexusSkeleton.text(width: 160, height: 13),
                SizedBox(height: 6),
                NexusSkeleton.text(width: 90, height: 11),
              ],
            ),
          ),
          const NexusSkeleton(width: 60, height: 20, radius: 6),
        ]),
      )),
    );
  }
}

class BundleListSkeleton extends StatelessWidget {
  final int count;
  const BundleListSkeleton({super.key, this.count = 4});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: List.generate(count, (i) => Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          decoration: BoxDecoration(
            color:        NexusColors.surface,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Row(children: const [
            NexusSkeleton.circle(size: 20),
            SizedBox(width: 12),
            Expanded(child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                NexusSkeleton.text(width: 140, height: 14),
                SizedBox(height: 5),
                NexusSkeleton.text(width: 80, height: 11),
              ],
            )),
            NexusSkeleton(width: 50, height: 20, radius: 6),
          ]),
        ),
      )),
    );
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// SPIN DRUM-ROLL WIDGET
// ═══════════════════════════════════════════════════════════════════════════════

/// Shows a pulsing "Get ready…" build-up above the spin button.
/// Displayed during the 800ms before the wheel actually starts.
class SpinAnticipationOverlay extends StatelessWidget {
  final bool visible;
  const SpinAnticipationOverlay({super.key, required this.visible});

  @override
  Widget build(BuildContext context) {
    return AnimatedSwitcher(
      duration: const Duration(milliseconds: 300),
      child: visible
          ? const _AnticipationContent()
          : const SizedBox.shrink(),
    );
  }
}

class _AnticipationContent extends StatefulWidget {
  const _AnticipationContent();
  @override State<_AnticipationContent> createState() => _AnticipationContentState();
}

class _AnticipationContentState extends State<_AnticipationContent>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync:    this,
      duration: const Duration(milliseconds: 600),
    )..repeat(reverse: true);
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _ctrl,
      builder: (_, __) => Opacity(
        opacity: 0.6 + 0.4 * _ctrl.value,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
          margin:  const EdgeInsets.only(bottom: 8),
          decoration: BoxDecoration(
            color:        NexusColors.goldDim,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: NexusColors.gold.withValues(alpha: 0.4 + 0.4 * _ctrl.value),
            ),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Text('🎰', style: TextStyle(fontSize: 18)),
              const SizedBox(width: 10),
              Text(
                'Get ready…',
                style: TextStyle(
                  color:      NexusColors.gold,
                  fontSize:   15,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 0.5 + 0.5 * _ctrl.value,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// WHEEL POINTER TICK — visual deceleration feedback
// ═══════════════════════════════════════════════════════════════════════════════

/// Renders an animated pointer that ticks left/right as the wheel decelerates.
/// Add above the wheel and bind to [tickValue] (increments per segment crossing).
class WheelPointerTick extends StatefulWidget {
  final int tickCount;    // bumps this each time a segment passes the pointer
  const WheelPointerTick({super.key, required this.tickCount});
  @override State<WheelPointerTick> createState() => _WheelPointerTickState();
}

class _WheelPointerTickState extends State<WheelPointerTick>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late Animation<double> _shake;
  int _lastTick = 0;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync:    this,
      duration: const Duration(milliseconds: 80),
    );
    _shake = Tween<double>(begin: 0, end: 1).animate(_ctrl);
  }

  @override
  void didUpdateWidget(WheelPointerTick old) {
    super.didUpdateWidget(old);
    if (widget.tickCount != _lastTick) {
      _lastTick = widget.tickCount;
      _ctrl.forward(from: 0);
      HapticFeedback.selectionClick(); // subtle tick haptic
    }
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _shake,
      builder: (_, child) {
        final offset = math.sin(_shake.value * math.pi) * 4;
        return Transform.translate(
          offset: Offset(offset, 0),
          child: child,
        );
      },
      child: const Icon(
        Icons.arrow_drop_down_rounded,
        color: NexusColors.gold,
        size:  42,
      ),
    );
  }
}
