import 'dart:async';
import 'dart:math' as math;
import 'package:confetti/confetti.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:gap/gap.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ── Providers ─────────────────────────────────────────────────────────────────

final _wheelConfigProvider = FutureProvider.autoDispose<List<WheelSegment>>((ref) async {
  try {
    final res = await ref.read(spinApiProvider).getWheelConfig();
    final raw = (res['prizes'] ?? res['segments'] ?? []) as List;
    final segs = raw
        .where((p) => (p as Map)['is_active'] != false)
        .map((p) => WheelSegment.fromMap(p as Map))
        .toList();
    if (segs.length >= 2) return segs;
    return _fallbackSegments;
  } catch (_) {
    return _fallbackSegments;
  }
});

final _walletProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getWallet();
});

final _spinHistoryProvider = FutureProvider.autoDispose<List<SpinHistoryItem>>((ref) async {
  final res = await ref.read(spinApiProvider).getHistory();
  return (res as List).map((e) => SpinHistoryItem.fromMap(e as Map)).toList();
});

// ── Fallback segments (mirroring webapp FALLBACK_SEGMENTS) ────────────────────
const _fallbackSegments = [
  WheelSegment(label: '₦500 Airtime',  prizeType: 'airtime',       baseValue: 50000,  color: Color(0xFF10b981)),
  WheelSegment(label: 'Try Again',     prizeType: 'try_again',      baseValue: 0,      color: Color(0xFF4b5563)),
  WheelSegment(label: '100 Points',    prizeType: 'pulse_points',   baseValue: 100,    color: Color(0xFF5f72f9)),
  WheelSegment(label: '₦1k Data',      prizeType: 'data_bundle',    baseValue: 100000, color: Color(0xFF06b6d4)),
  WheelSegment(label: '50 Points',     prizeType: 'pulse_points',   baseValue: 50,     color: Color(0xFFa78bfa)),
  WheelSegment(label: '₦2k Cash',      prizeType: 'momo_cash',      baseValue: 200000, color: Color(0xFFf59e0b)),
  WheelSegment(label: 'Try Again',     prizeType: 'try_again',      baseValue: 0,      color: Color(0xFF374151)),
  WheelSegment(label: '₦5k Cash',      prizeType: 'momo_cash',      baseValue: 500000, color: Color(0xFFf43f5e)),
];

// ── Data models ───────────────────────────────────────────────────────────────

@immutable
class WheelSegment {
  final String label;
  final String prizeType;
  final int baseValue;
  final Color color;
  const WheelSegment({
    required this.label,
    required this.prizeType,
    required this.baseValue,
    required this.color,
  });
  factory WheelSegment.fromMap(Map m) => WheelSegment(
    label:     m['name'] ?? m['label'] ?? m['prize_name'] ?? 'Prize',
    prizeType: ((m['prize_type'] ?? m['type'] ?? 'try_again') as String).toLowerCase(),
    baseValue: (m['base_value'] ?? m['prize_value'] ?? m['value'] ?? 0) as int,
    color:     (m['prize_type'] ?? '') == 'try_again'
        ? const Color(0xFF374151)
        : _hexColor(m['color_hex'] ?? m['color'] ?? '#5f72f9'),
  );
}

Color _hexColor(String hex) {
  final h = hex.replaceAll('#', '');
  return Color(int.parse('FF$h', radix: 16));
}

class SpinHistoryItem {
  final String id;
  final String prizeType;
  final int prizeValue;
  final String fulfillmentStatus;
  final DateTime createdAt;
  const SpinHistoryItem({
    required this.id, required this.prizeType,
    required this.prizeValue, required this.fulfillmentStatus,
    required this.createdAt,
  });
  factory SpinHistoryItem.fromMap(Map m) => SpinHistoryItem(
    id: m['id']?.toString() ?? '',
    prizeType: (m['prize_type'] ?? 'try_again').toString().toLowerCase(),
    prizeValue: (m['prize_value'] ?? 0) as int,
    fulfillmentStatus: (m['fulfillment_status'] ?? 'n/a').toString().toLowerCase(),
    createdAt: DateTime.tryParse(m['created_at']?.toString() ?? '') ?? DateTime.now(),
  );

  String get displayLabel {
    switch (prizeType) {
      case 'try_again':    return 'Try Again';
      case 'pulse_points': return '$prizeValue Points';
      case 'airtime':      return '₦${(prizeValue / 100).toStringAsFixed(0)} Airtime';
      case 'data_bundle':  return 'Data Bundle';
      case 'momo_cash':    return '₦${(prizeValue / 100).toStringAsFixed(0)} Cash';
      default:             return prizeType;
    }
  }

  bool get isWin => prizeType != 'try_again';
}

// ── Main screen ───────────────────────────────────────────────────────────────

class SpinScreen extends ConsumerStatefulWidget {
  const SpinScreen({super.key});
  @override ConsumerState<SpinScreen> createState() => _SpinScreenState();
}

class _SpinScreenState extends ConsumerState<SpinScreen>
    with TickerProviderStateMixin {
  late final AnimationController _wheelCtrl;
  late final Animation<double> _wheelAnim;
  late final ConfettiController _confettiCtrl;

  double _currentAngle = 0;
  bool _spinning = false;
  bool _spun = false;
  Map<String, dynamic>? _outcome;
  bool _showResult = false;

  @override
  void initState() {
    super.initState();
    _confettiCtrl = ConfettiController(duration: const Duration(seconds: 3));
    _wheelCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 4600),
    );
    _wheelAnim = CurvedAnimation(
      parent: _wheelCtrl,
      curve: Curves.easeOutExpo,
    );
    _wheelCtrl.addListener(() => setState(() {}));
    _wheelCtrl.addStatusListener((s) {
      if (s == AnimationStatus.completed) _onSpinComplete();
    });
  }

  @override
  void dispose() {
    _confettiCtrl.dispose();
    _wheelCtrl.dispose();
    super.dispose();
  }

  // ── Compute target angle from server slot_index ─────────────────────────
  double _targetRotation(List<WheelSegment> segs, int slotIndex) {
    final segAngle = 2 * math.pi / segs.length;
    final target   = slotIndex * segAngle + segAngle / 2;
    final extra    = (6 + math.Random().nextDouble() * 2) * 2 * math.pi;
    return extra + (2 * math.pi - target);
  }

  Future<void> _handleSpin(List<WheelSegment> segs, int spinCredits) async {
    if (_spinning || _spun) return;
    if (spinCredits < 1) {
      _showSnack('No spin credits! Recharge to earn spins 💫', isError: true);
      return;
    }
    HapticFeedback.mediumImpact();
    setState(() { _spinning = true; _showResult = false; });

    try {
      final res = await ref.read(spinApiProvider).play();
      _outcome = Map<String, dynamic>.from(res);
      final slotIdx = (res['slot_index'] ?? 0) as int;
      final delta = _targetRotation(segs, slotIdx);
      _currentAngle += delta;
      _wheelCtrl.reset();
      _wheelCtrl.forward();
    } catch (e) {
      setState(() => _spinning = false);
      _showSnack(e.toString(), isError: true);
    }
  }

  void _onSpinComplete() {
    HapticFeedback.heavyImpact();
    setState(() {
      _spinning = false;
      _spun = true;
      _showResult = true;
    });
    final prizeType = _outcome?['spin_result']?['prize_type'] ?? 'try_again';
    if (prizeType != 'try_again') {
      _confettiCtrl.play(); // 🎉 fire confetti on any real win
      _showSnack('🎉 ${_outcome?['prize_label'] ?? 'You won!'}');
    }
    // Refresh wallet + history
    ref.invalidate(_walletProvider);
    ref.invalidate(_spinHistoryProvider);
  }

  void _handleReset() {
    setState(() { _spun = false; _outcome = null; _showResult = false; });
  }

  void _showSnack(String msg, {bool isError = false}) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: isError ? NexusColors.red : NexusColors.green,
      behavior: SnackBarBehavior.floating,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
    ));
  }

  @override
  Widget build(BuildContext context) {
    final wheelAsync   = ref.watch(_wheelConfigProvider);
    final walletAsync  = ref.watch(_walletProvider);
    final historyAsync = ref.watch(_spinHistoryProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        title: const Text('Spin & Win ✨'),
        actions: [
          TextButton.icon(
            onPressed: () => context.push('/prizes'),
            icon: const Icon(Icons.history_rounded, size: 16),
            label: const Text('My Prizes', style: TextStyle(fontSize: 12)),
            style: TextButton.styleFrom(foregroundColor: NexusColors.primary),
          ),
        ],
      ),
      body: Stack(children: [
        wheelAsync.when(
        loading: () => _SpinShimmer(),
        error:   (_, __) => _buildErrorState(),
        data: (segs) {
          final credits = walletAsync.valueOrNull?['spin_credits'] as int? ?? 0;
          return RefreshIndicator(
            color: NexusColors.primary,
            onRefresh: () async {
              ref.invalidate(_walletProvider);
              ref.invalidate(_spinHistoryProvider);
            },
            child: SingleChildScrollView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(20, 8, 20, 100),
              child: Column(children: [
                // Credits badge
                _CreditsBadge(credits: credits, walletLoaded: walletAsync.hasValue),
                const SizedBox(height: 20),

                // Wheel card
                Container(
                  padding: const EdgeInsets.all(24),
                  decoration: BoxDecoration(
                    color: NexusColors.surface,
                    borderRadius: BorderRadius.circular(24),
                    border: Border.all(color: NexusColors.border),
                    boxShadow: _spinning
                        ? [BoxShadow(color: NexusColors.primary.withOpacity(0.3), blurRadius: 40, spreadRadius: 4)]
                        : [],
                  ),
                  child: Column(children: [
                    // Wheel
                    _WheelWidget(
                      segments: segs,
                      angle: _currentAngle,
                      animValue: _wheelAnim.value,
                      baseAngle: _currentAngle,
                      spinning: _spinning,
                    ),
                    const SizedBox(height: 20),

                    // Result
                    AnimatedSwitcher(
                      duration: const Duration(milliseconds: 350),
                      child: _showResult && _outcome != null
                          ? _ResultCard(outcome: _outcome!, onReset: _handleReset)
                          : const SizedBox.shrink(),
                    ),

                    // Spin button
                    if (!_spun) ...[
                      const SizedBox(height: 4),
                      _SpinButton(
                        onTap: () => _handleSpin(segs, credits),
                        spinning: _spinning,
                        credits: credits,
                      ),
                    ],
                  ]),
                ),

                const SizedBox(height: 20),

                // Prizes key
                _PrizesKey(segments: segs),

                const SizedBox(height: 20),

                // History
                historyAsync.when(
                  loading: () => const SizedBox.shrink(),
                  error:   (_, __) => const SizedBox.shrink(),
                  data: (history) => history.isEmpty
                      ? const SizedBox.shrink()
                      : _HistorySection(history: history),
                ),
              ]),
            ),
          );
        },
        ),
        // Confetti cannon — fires from top-center on win
        Align(
          alignment: Alignment.topCenter,
          child: ConfettiWidget(
            confettiController: _confettiCtrl,
            blastDirectionality: BlastDirectionality.explosive,
            numberOfParticles: 40,
            gravity: 0.3,
            emissionFrequency: 0.05,
            colors: const [
              Color(0xFF5f72f9), Color(0xFFf9c74f), Color(0xFF10b981),
              Color(0xFFf43f5e), Color(0xFF06b6d4), Color(0xFFa78bfa),
            ],
          ),
        ),
      ]),
    );
  }

  Widget _buildErrorState() => Center(
    child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
      const Icon(Icons.wifi_off_rounded, size: 56, color: NexusColors.textSecondary),
      const SizedBox(height: 16),
      const Text('Could not load wheel', style: TextStyle(color: NexusColors.textSecondary, fontSize: 16)),
      const SizedBox(height: 12),
      ElevatedButton(
        onPressed: () => ref.invalidate(_wheelConfigProvider),
        child: const Text('Retry'),
      ),
    ]),
  );
}

// ── Credits badge ──────────────────────────────────────────────────────────────

class _CreditsBadge extends StatelessWidget {
  final int credits;
  final bool walletLoaded;
  const _CreditsBadge({required this.credits, required this.walletLoaded});

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    decoration: BoxDecoration(
      color: NexusColors.surface,
      borderRadius: BorderRadius.circular(16),
      border: Border.all(color: credits > 0 ? NexusColors.primary.withOpacity(0.3) : NexusColors.border),
    ),
    child: Row(children: [
      Icon(Icons.bolt_rounded,
        color: credits > 0 ? NexusColors.gold : NexusColors.textSecondary.withOpacity(0.4),
        size: 20),
      const SizedBox(width: 10),
      const Text('Available Spins', style: TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
      const Spacer(),
      if (!walletLoaded)
        const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2, color: NexusColors.primary))
      else ...[
        Text('$credits',
          style: TextStyle(
            color: credits > 0 ? NexusColors.gold : NexusColors.textSecondary,
            fontSize: 24, fontWeight: FontWeight.w800,
          )),
        if (credits == 0) ...[
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              border: Border.all(color: NexusColors.border),
              borderRadius: BorderRadius.circular(20),
            ),
            child: const Text('Recharge to earn',
              style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
          ),
        ],
      ],
    ]),
  );
}

// ── Wheel widget with real CustomPainter ───────────────────────────────────────

class _WheelWidget extends StatelessWidget {
  final List<WheelSegment> segments;
  final double angle;
  final double animValue;
  final double baseAngle;
  final bool spinning;

  const _WheelWidget({
    required this.segments,
    required this.angle,
    required this.animValue,
    required this.baseAngle,
    required this.spinning,
  });

  @override
  Widget build(BuildContext context) {
    const size = 270.0;
    final displayAngle = baseAngle * animValue;
    return Column(children: [
      // Pointer
      const _WheelPointer(),
      const SizedBox(height: 4),
      SizedBox(
        width: size, height: size,
        child: Stack(alignment: Alignment.center, children: [
          // Glow
          if (spinning)
            Container(
              width: size, height: size,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                boxShadow: [
                  BoxShadow(color: NexusColors.primary.withOpacity(0.4), blurRadius: 50, spreadRadius: 10),
                  BoxShadow(color: NexusColors.gold.withOpacity(0.2), blurRadius: 80, spreadRadius: 20),
                ],
              ),
            ),
          // Wheel disc
          Transform.rotate(
            angle: displayAngle,
            child: CustomPaint(
              size: const Size(size, size),
              painter: _WheelPainter(segments: segments),
            ),
          ),
          // Outer ring
          Container(
            width: size, height: size,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              border: Border.all(
                color: spinning ? NexusColors.primary : NexusColors.border,
                width: spinning ? 3 : 2,
              ),
            ),
          ),
          // Centre hub
          _WheelHub(spinning: spinning),
        ]),
      ),
    ]);
  }
}

class _WheelPointer extends StatelessWidget {
  const _WheelPointer();
  @override
  Widget build(BuildContext context) => CustomPaint(
    size: const Size(24, 28),
    painter: _PointerPainter(),
  );
}

class _PointerPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = NexusColors.gold
      ..style = PaintingStyle.fill;
    final path = Path()
      ..moveTo(size.width / 2, size.height)
      ..lineTo(0, 0)
      ..lineTo(size.width, 0)
      ..close();
    canvas.drawPath(path, paint);
    canvas.drawShadow(path, NexusColors.gold.withOpacity(0.6), 6, false);
  }
  @override bool shouldRepaint(_) => false;
}

class _WheelPainter extends CustomPainter {
  final List<WheelSegment> segments;
  const _WheelPainter({required this.segments});

  @override
  void paint(Canvas canvas, Size size) {
    final cx = size.width / 2;
    final cy = size.height / 2;
    final r  = size.width / 2;
    final n  = segments.length;
    final sweep = 2 * math.pi / n;

    for (int i = 0; i < n; i++) {
      final start = -math.pi / 2 + i * sweep;
      final seg = segments[i];

      // Sector fill
      final paint = Paint()
        ..color = seg.color
        ..style = PaintingStyle.fill;
      final path = Path()
        ..moveTo(cx, cy)
        ..arcTo(Rect.fromCircle(center: Offset(cx, cy), radius: r - 2),
          start, sweep, false)
        ..close();
      canvas.drawPath(path, paint);

      // Sector border
      canvas.drawPath(path, Paint()
        ..color = Colors.black.withOpacity(0.25)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.2);

      // Label
      canvas.save();
      canvas.translate(cx, cy);
      canvas.rotate(start + sweep / 2);
      final textR = r * 0.68;
      final tp = TextPainter(
        text: TextSpan(
          text: seg.label,
          style: TextStyle(
            color: Colors.white.withOpacity(0.95),
            fontSize: r < 140 ? 8.5 : 9.5,
            fontWeight: FontWeight.w700,
            shadows: const [Shadow(color: Colors.black, blurRadius: 4)],
          ),
        ),
        textDirection: TextDirection.ltr,
        textAlign: TextAlign.center,
      );
      tp.layout(maxWidth: 56);
      tp.paint(canvas, Offset(textR - tp.width / 2, -tp.height / 2));
      canvas.restore();
    }
  }

  @override
  bool shouldRepaint(_WheelPainter old) => old.segments != segments;
}

class _WheelHub extends StatelessWidget {
  final bool spinning;
  const _WheelHub({required this.spinning});
  @override
  Widget build(BuildContext context) => Container(
    width: 52, height: 52,
    decoration: BoxDecoration(
      color: NexusColors.background,
      shape: BoxShape.circle,
      border: Border.all(
        color: spinning ? NexusColors.primary : NexusColors.border, width: 3),
      boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.5), blurRadius: 12)],
    ),
    child: Center(
      child: spinning
          ? const SizedBox(width: 18, height: 18,
              child: CircularProgressIndicator(strokeWidth: 2.5, color: NexusColors.primary))
          : const Icon(Icons.stars_rounded, color: NexusColors.primary, size: 22),
    ),
  );
}

// ── Spin button ───────────────────────────────────────────────────────────────

class _SpinButton extends StatelessWidget {
  final VoidCallback onTap;
  final bool spinning;
  final int credits;
  const _SpinButton({required this.onTap, required this.spinning, required this.credits});

  @override
  Widget build(BuildContext context) {
    final canSpin = !spinning && credits > 0;
    return Column(children: [
      SizedBox(
        width: double.infinity,
        child: ElevatedButton(
          onPressed: canSpin ? onTap : null,
          style: ElevatedButton.styleFrom(
            padding: const EdgeInsets.symmetric(vertical: 16),
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18)),
            backgroundColor: NexusColors.primary,
            foregroundColor: Colors.white,
            disabledBackgroundColor: NexusColors.primary.withOpacity(0.3),
            elevation: 0,
          ),
          child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
            if (spinning) ...[
              const SizedBox(width: 18, height: 18,
                child: CircularProgressIndicator(strokeWidth: 2.5, color: Colors.white)),
              const SizedBox(width: 10),
              const Text('Spinning…', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
            ] else if (credits > 0) ...[
              const Icon(Icons.bolt_rounded, size: 20),
              const SizedBox(width: 8),
              Text('Spin Now! ($credits left)',
                style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
            ] else
              const Text('No Spins — Recharge to Earn',
                style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
          ]),
        ),
      ),
      if (credits == 0)
        const Padding(
          padding: EdgeInsets.only(top: 8),
          child: Text('Every ₦1,000 recharge = 1 spin credit',
            style: TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
        ),
    ]);
  }
}

// ── Result card ────────────────────────────────────────────────────────────────

class _ResultCard extends StatelessWidget {
  final Map<String, dynamic> outcome;
  final VoidCallback onReset;
  const _ResultCard({required this.outcome, required this.onReset});

  static const _prizeIcons = {
    'airtime':      ('📱', Color(0xFF60a5fa)),
    'data_bundle':  ('📶', Color(0xFFa78bfa)),
    'pulse_points': ('⚡', Color(0xFFf9c74f)),
    'momo_cash':    ('💰', Color(0xFF4ade80)),
    'try_again':    ('🔄', Color(0xFF6b7280)),
  };

  @override
  Widget build(BuildContext context) {
    final prizeType  = outcome['spin_result']?['prize_type']?.toString() ?? 'try_again';
    final prizeLabel = outcome['prize_label']?.toString() ?? '';
    final needsMomo  = outcome['needs_momo_setup'] as bool? ?? false;
    final isWin      = prizeType != 'try_again';
    final iconData   = _prizeIcons[prizeType] ?? ('🎁', NexusColors.primary);

    String subText;
    if (needsMomo) {
      subText = '⚠️ Add your MoMo number in Settings to receive your cash prize.';
    } else if (prizeType == 'momo_cash') {
      subText = 'Cash will be sent to your linked MoMo number within 24 hours.';
    } else if (prizeType == 'pulse_points') {
      subText = 'Points added to your wallet instantly! ⚡';
    } else if (prizeType == 'airtime') {
      subText = 'Airtime credited to your MTN line within 5–10 minutes.';
    } else if (prizeType == 'data_bundle') {
      subText = 'Data bundle activated on your MTN line within 5–10 minutes.';
    } else {
      subText = 'Better luck next time! Recharge to earn another spin credit.';
    }

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: isWin
            ? LinearGradient(
                colors: [NexusColors.primary.withOpacity(0.12), iconData.$2.withOpacity(0.08)],
                begin: Alignment.topLeft, end: Alignment.bottomRight)
            : null,
        color: isWin ? null : NexusColors.surface,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(
          color: isWin ? iconData.$2.withOpacity(0.4) : NexusColors.border),
      ),
      child: Column(children: [
        if (isWin) ...[
          Text(iconData.$1, style: const TextStyle(fontSize: 48))
              .animate()
              .scale(begin: const Offset(0.5, 0.5), end: const Offset(1, 1),
                  curve: Curves.elasticOut, duration: 600.ms),
          const Gap(8),
          const Text('YOU WON!',
            style: TextStyle(color: NexusColors.gold, fontSize: 11,
              fontWeight: FontWeight.w900, letterSpacing: 2)),
          const Gap(6),
          Text(prizeLabel,
            style: TextStyle(color: iconData.$2, fontSize: 24,
              fontWeight: FontWeight.w900),
            textAlign: TextAlign.center),
        ] else ...[
          Text(iconData.$1, style: const TextStyle(fontSize: 40))
              .animate().shake(hz: 3, curve: Curves.easeInOut),
          const Gap(8),
          const Text('Not this time…',
            style: TextStyle(color: NexusColors.textSecondary, fontSize: 16,
              fontWeight: FontWeight.w600)),
        ],
        const Gap(10),
        Text(subText,
          style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12, height: 1.5),
          textAlign: TextAlign.center),
        if (needsMomo) ...[
          const Gap(12),
          FilledButton.icon(
            onPressed: () => context.push('/settings'),
            icon: const Icon(Icons.phone_android_rounded, size: 14),
            label: const Text('Add MoMo Number'),
            style: FilledButton.styleFrom(
              backgroundColor: NexusColors.primary,
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
            ),
          ),
        ],
        const Gap(12),
        TextButton(
          onPressed: onReset,
          style: TextButton.styleFrom(foregroundColor: NexusColors.textSecondary),
          child: const Text('Spin Again →', style: TextStyle(fontSize: 13)),
        ),
      ]),
    ).animate().fadeIn(duration: 350.ms).slideY(begin: 0.1, end: 0);
  }
}

// ── Prizes key ─────────────────────────────────────────────────────────────────

class _PrizesKey extends StatelessWidget {
  final List<WheelSegment> segments;
  const _PrizesKey({required this.segments});

  @override
  Widget build(BuildContext context) {
    final unique = <String>{};
    final distinctSegs = segments.where((s) => unique.add(s.prizeType + s.label)).toList();

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: NexusColors.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Row(children: [
          Icon(Icons.info_outline_rounded, size: 14, color: NexusColors.textSecondary),
          SizedBox(width: 6),
          Text('Possible Prizes',
            style: TextStyle(color: NexusColors.textSecondary, fontSize: 12,
              fontWeight: FontWeight.w700, letterSpacing: 0.8)),
        ]),
        const SizedBox(height: 12),
        Wrap(
          spacing: 10, runSpacing: 8,
          children: distinctSegs.map((s) => Row(mainAxisSize: MainAxisSize.min, children: [
            Container(width: 10, height: 10, decoration: BoxDecoration(
              color: s.color, shape: BoxShape.circle)),
            const SizedBox(width: 6),
            Text(s.label,
              style: TextStyle(
                color: s.prizeType == 'try_again'
                    ? NexusColors.textSecondary.withOpacity(0.4)
                    : NexusColors.textSecondary,
                fontSize: 12,
                fontStyle: s.prizeType == 'try_again' ? FontStyle.italic : FontStyle.normal,
              )),
          ])).toList(),
        ),
      ]),
    );
  }
}

// ── Spin history section ───────────────────────────────────────────────────────

class _HistorySection extends StatelessWidget {
  final List<SpinHistoryItem> history;
  const _HistorySection({required this.history});

  @override
  Widget build(BuildContext context) {
    final recent = history.take(5).toList();
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      const Row(children: [
        Icon(Icons.history_rounded, size: 14, color: NexusColors.textSecondary),
        SizedBox(width: 6),
        Text('Recent Spins',
          style: TextStyle(color: NexusColors.textSecondary, fontSize: 12,
            fontWeight: FontWeight.w700, letterSpacing: 0.8)),
      ]),
      const SizedBox(height: 10),
      ...recent.map((item) => _HistoryRow(item: item)),
      const SizedBox(height: 6),
      Center(
        child: TextButton(
          onPressed: () => context.push('/prizes'),
          style: TextButton.styleFrom(foregroundColor: NexusColors.primary),
          child: const Text('View all spin history →', style: TextStyle(fontSize: 13)),
        ),
      ),
    ]);
  }
}

class _HistoryRow extends StatelessWidget {
  final SpinHistoryItem item;
  const _HistoryRow({required this.item});

  @override
  Widget build(BuildContext context) {
    final statusCfg = _statusConfig(item.fulfillmentStatus);
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: NexusColors.border),
      ),
      child: Row(children: [
        Icon(item.isWin ? Icons.card_giftcard_rounded : Icons.close_rounded,
          size: 16,
          color: item.isWin ? NexusColors.primary : NexusColors.textSecondary.withOpacity(0.3)),
        const SizedBox(width: 10),
        Expanded(
          child: Text(item.displayLabel,
            style: TextStyle(
              color: item.isWin ? NexusColors.textPrimary : NexusColors.textSecondary.withOpacity(0.5),
              fontSize: 13, fontWeight: FontWeight.w500)),
        ),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
          decoration: BoxDecoration(
            color: statusCfg.$1.withOpacity(0.15),
            borderRadius: BorderRadius.circular(20),
          ),
          child: Text(statusCfg.$2,
            style: TextStyle(color: statusCfg.$1, fontSize: 10, fontWeight: FontWeight.w700)),
        ),
        const SizedBox(width: 8),
        Text(_formatDate(item.createdAt),
          style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
      ]),
    );
  }

  (Color, String) _statusConfig(String status) {
    switch (status) {
      case 'completed':    return (NexusColors.green, 'Credited');
      case 'pending':
      case 'pending_momo': return (NexusColors.gold,  'Pending');
      case 'failed':       return (NexusColors.red,   'Failed');
      case 'n/a':
      default:             return (NexusColors.textSecondary.withOpacity(0.3), 'No Prize');
    }
  }

  String _formatDate(DateTime d) {
    final months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
    return '${d.day} ${months[d.month - 1]}';
  }
}

// ── Spin shimmer ──────────────────────────────────────────────────────────────
class _SpinShimmer extends StatelessWidget {
  @override
  Widget build(BuildContext context) => SingleChildScrollView(
    padding: const EdgeInsets.all(24),
    child: Column(children: const [
      // Wheel placeholder
      NexusShimmer(width: 300, height: 300, radius: BorderRadius.all(Radius.circular(150))),
      SizedBox(height: 28),
      // Credits bar
      NexusShimmer(width: double.infinity, height: 52, radius: NexusRadius.md),
      SizedBox(height: 20),
      // Spin button
      NexusShimmer(width: double.infinity, height: 56, radius: NexusRadius.lg),
      SizedBox(height: 24),
      // History rows
      NexusShimmer(width: double.infinity, height: 44, radius: NexusRadius.sm),
      SizedBox(height: 8),
      NexusShimmer(width: double.infinity, height: 44, radius: NexusRadius.sm),
      SizedBox(height: 8),
      NexusShimmer(width: double.infinity, height: 44, radius: NexusRadius.sm),
    ]),
  );
}
