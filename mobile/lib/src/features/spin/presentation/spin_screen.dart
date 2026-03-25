import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'dart:math' as math;
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

class SpinScreen extends ConsumerStatefulWidget {
  const SpinScreen({super.key});
  @override ConsumerState<SpinScreen> createState() => _State();
}

class _State extends ConsumerState<SpinScreen> with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;
  late Animation<double> _anim;
  bool spinning = false;
  String? prizeLabel;
  bool? isWin;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(vsync: this, duration: const Duration(milliseconds: 3500));
    _anim = CurvedAnimation(parent: _ctrl, curve: Curves.elasticOut);
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  Future<void> spin() async {
    if (spinning) return;
    setState(() { spinning = true; prizeLabel = null; });
    final target = 6 + math.Random().nextDouble() * 4;
    _ctrl.reset();
    _ctrl.animateTo(1.0, duration: const Duration(milliseconds: 3500));
    try {
      final result = await ref.read(dioProvider).apiPost<Map<String, dynamic>>('/spin/play', data: {});
      await Future.delayed(const Duration(milliseconds: 3500));
      setState(() {
        prizeLabel = result['prize_label'] as String? ?? 'Try Again';
        isWin = result['is_win'] as bool? ?? false;
        spinning = false;
      });
    } on ApiException catch (e) {
      await Future.delayed(const Duration(milliseconds: 3500));
      setState(() { prizeLabel = e.message; isWin = false; spinning = false; });
    }
  }

  @override
  Widget build(BuildContext ctx) {
    return Scaffold(
      appBar: AppBar(title: const Text('Spin & Win')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(children: [
          const Spacer(),
          // Animated wheel
          AnimatedBuilder(
            animation: _anim,
            builder: (_, __) => Transform.rotate(
              angle: _anim.value * 6 * math.pi,
              child: Container(
                width: 260, height: 260,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  gradient: SweepGradient(colors: [
                    NexusColors.primary, const Color(0xFF8B5CF6),
                    NexusColors.gold, const Color(0xFF06B6D4),
                    NexusColors.green, const Color(0xFFF43F5E),
                    const Color(0xFFFB923C), NexusColors.primary,
                  ]),
                  boxShadow: [BoxShadow(
                    color: NexusColors.primary.withOpacity(0.4),
                    blurRadius: 30, spreadRadius: 5)],
                ),
                child: Center(child: Container(
                  width: 64, height: 64,
                  decoration: BoxDecoration(
                    color: NexusColors.background, shape: BoxShape.circle,
                    border: Border.all(color: NexusColors.primary, width: 3)),
                  child: const Icon(Icons.bolt, color: NexusColors.primary, size: 32))))),
          ),
          const SizedBox(height: 24),
          if (prizeLabel != null)
            AnimatedOpacity(
              opacity: 1, duration: const Duration(milliseconds: 500),
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
                decoration: BoxDecoration(
                  color: NexusColors.surface,
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(
                    color: (isWin ?? false) ? NexusColors.gold : NexusColors.border)),
                child: Column(children: [
                  Text((isWin ?? false) ? '🎉' : '😔', style: const TextStyle(fontSize: 32)),
                  const SizedBox(height: 8),
                  Text(prizeLabel!, style: const TextStyle(
                    color: NexusColors.textPrimary, fontSize: 18, fontWeight: FontWeight.bold)),
                ]))),
          const Spacer(),
          ElevatedButton.icon(
            onPressed: spinning ? null : spin,
            icon: spinning
              ? const SizedBox(width: 20, height: 20,
                  child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
              : const Icon(Icons.casino_rounded),
            label: Text(spinning ? 'Spinning…' : 'Spin Now'),
          ),
          const SizedBox(height: 24),
        ]),
      ),
    );
  }
}