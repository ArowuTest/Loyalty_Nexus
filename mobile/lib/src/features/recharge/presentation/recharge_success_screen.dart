import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

class RechargeSuccessScreen extends ConsumerWidget {
  final String? reference;
  const RechargeSuccessScreen({super.key, this.reference});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authStateProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            children: [
              const Spacer(flex: 2),

              // ── Success icon ──
              Container(
                width:  100,
                height: 100,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: Colors.green.withValues(alpha: 0.12),
                  border: Border.all(color: Colors.green, width: 2),
                ),
                child: const Icon(
                  Icons.check_rounded,
                  color: Colors.green,
                  size:  52,
                ),
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

              const SizedBox(height: 32),

              // ── Double points callout ──
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color:        NexusColors.goldDim,
                  borderRadius: BorderRadius.circular(16),
                  border:       Border.all(color: NexusColors.gold.withValues(alpha: 0.4)),
                ),
                child: const Row(
                  children: [
                    Text('⚡', style: TextStyle(fontSize: 24)),
                    SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'DOUBLE POINTS INCOMING',
                            style: TextStyle(
                              color:      NexusColors.gold,
                              fontSize:   13,
                              fontWeight: FontWeight.w800,
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

              if (reference != null) ...[
                const SizedBox(height: 20),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
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
                        'Ref: $reference',
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

              const Spacer(flex: 3),

              // ── CTAs ──
              Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // View points only if logged in
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
                        style: TextStyle(fontWeight: FontWeight.w700, fontSize: 16),
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
                        style: TextStyle(fontWeight: FontWeight.w700, fontSize: 16),
                      ),
                    ),

                  const SizedBox(height: 12),

                  OutlinedButton(
                    onPressed: () => context.go('/recharge'),
                    style: OutlinedButton.styleFrom(
                      foregroundColor:  Colors.white70,
                      side:             const BorderSide(color: Colors.white24),
                      minimumSize:      const Size.fromHeight(48),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(14),
                      ),
                    ),
                    child: const Text(
                      'Recharge Again',
                      style: TextStyle(fontWeight: FontWeight.w600, fontSize: 15),
                    ),
                  ),
                ],
              ),

              const SizedBox(height: 8),
            ],
          ),
        ),
      ),
    );
  }
}
