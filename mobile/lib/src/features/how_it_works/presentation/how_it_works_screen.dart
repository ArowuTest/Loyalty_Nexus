import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:gap/gap.dart';
import 'package:go_router/go_router.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Data ─────────────────────────────────────────────────────────────────────

class _Step {
  final String number;
  final String emoji;
  final Color color;
  final String title;
  final String body;
  final String stat;
  final List<String> details;
  const _Step({
    required this.number,
    required this.emoji,
    required this.color,
    required this.title,
    required this.body,
    required this.stat,
    required this.details,
  });
}

const _steps = [
  _Step(
    number: '01',
    emoji: '📱',
    color: Color(0xFF00D4FF),
    title: 'Recharge MTN',
    body:
        'Recharge ₦1,000 or more on any MTN line. Your recharge is automatically detected — no codes, no USSD, no hassle. Just recharge as you normally would.',
    stat: '₦250 = 1 Pulse Point',
    details: [
      'Works with any MTN prepaid line',
      'Minimum qualifying recharge: ₦1,000',
      'Detected automatically within seconds',
      'No extra steps required',
    ],
  ),
  _Step(
    number: '02',
    emoji: '⚡',
    color: Color(0xFFF5A623),
    title: 'Earn Pulse Points',
    body:
        'Every naira you recharge earns Pulse Points. The more you recharge, the more you earn. Accumulate points to climb tiers — Bronze, Silver, Gold, and Platinum — each unlocking better rewards.',
    stat: 'Points on every recharge',
    details: [
      'Bronze → Silver: 2,000 lifetime points',
      'Silver → Gold: 10,000 lifetime points',
      'Gold → Platinum: 50,000 lifetime points',
      'Higher tiers earn bonus multipliers',
    ],
  ),
  _Step(
    number: '03',
    emoji: '🎰',
    color: Color(0xFF10B981),
    title: 'Spin & Win',
    body:
        'Each qualifying recharge earns you a free wheel spin. Land on instant cash, data bundles, airtime, or bonus Pulse Points. Prizes are credited instantly — no waiting.',
    stat: '₦18M+ prizes distributed',
    details: [
      'One free spin per qualifying recharge',
      'Instant cash prizes paid to your wallet',
      'Data bundles credited to your MTN line',
      'Bonus spins for streak milestones',
    ],
  ),
  _Step(
    number: '04',
    emoji: '🚀',
    color: Color(0xFF8B5CF6),
    title: 'Unlock AI Studio',
    body:
        'Spend your Pulse Points to access 30+ world-class AI tools — create stunning photos, generate videos, build business plans, compose music, and more. No foreign cards, no subscriptions.',
    stat: '1.2M+ generations created',
    details: [
      '30+ AI tools across 6 categories',
      'Pay per use with Pulse Points',
      'Photo generation, video, music & more',
      'Business plans, voice-to-text, and chat',
    ],
  ),
];

class _Tier {
  final String tier;
  final String emoji;
  final Color color;
  final String pts;
  final List<String> perks;
  const _Tier({
    required this.tier,
    required this.emoji,
    required this.color,
    required this.pts,
    required this.perks,
  });
}

const _tiers = [
  _Tier(
    tier: 'Bronze',
    emoji: '🥉',
    color: Color(0xFFCD7F32),
    pts: '0+ points',
    perks: ['1× spin multiplier', 'Standard prizes', 'Basic AI Studio'],
  ),
  _Tier(
    tier: 'Silver',
    emoji: '🥈',
    color: Color(0xFFC0C0C0),
    pts: '2,000+ points',
    perks: ['1.2× spin multiplier', 'Silver prize pool', 'More AI tools'],
  ),
  _Tier(
    tier: 'Gold',
    emoji: '🥇',
    color: Color(0xFFF5A623),
    pts: '10,000+ points',
    perks: ['1.5× spin multiplier', 'Gold prize pool', 'Priority AI access'],
  ),
  _Tier(
    tier: 'Platinum',
    emoji: '💎',
    color: Color(0xFFA78BFA),
    pts: '50,000+ points',
    perks: ['2× spin multiplier', 'Platinum prize pool', 'All AI tools'],
  ),
];

class _Faq {
  final String q;
  final String a;
  const _Faq({required this.q, required this.a});
}

const _faqs = [
  _Faq(
    q: 'Which networks are supported?',
    a: 'Currently, Loyalty Nexus supports MTN Nigeria lines only. We are working to expand to other networks in the future.',
  ),
  _Faq(
    q: 'How quickly are recharges detected?',
    a: 'Recharges are typically detected within a few seconds. In rare cases it may take up to a minute.',
  ),
  _Faq(
    q: 'How do I withdraw my cash prizes?',
    a: 'Cash prizes are credited to your Loyalty Nexus wallet. Withdraw to your bank account from the Prizes section.',
  ),
  _Faq(
    q: 'Do Pulse Points expire?',
    a: 'Pulse Points do not expire as long as your account remains active (at least one recharge every 90 days).',
  ),
  _Faq(
    q: 'What AI tools are available?',
    a: 'AI Studio includes image generation, AI photo editing, video creation, music composition, business plan writing, voice-to-text, and AI chat — over 30 tools in total.',
  ),
  _Faq(
    q: 'Is there a minimum recharge amount?',
    a: 'Yes. The minimum qualifying recharge to earn Pulse Points and a free spin is ₦1,000.',
  ),
];

// ─── Screen ───────────────────────────────────────────────────────────────────

class HowItWorksScreen extends StatelessWidget {
  const HowItWorksScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        surfaceTintColor: Colors.transparent,
        title: const Text('How It Works'),
        centerTitle: false,
      ),
      body: ListView(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        children: [
          // ── Hero ──────────────────────────────────────────────────────────
          _HeroSection(),
          const Gap(24),

          // ── Steps ─────────────────────────────────────────────────────────
          _SectionLabel(label: 'Four Steps to Everything', color: NexusColors.gold),
          const Gap(12),
          ..._steps.asMap().entries.map((e) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: _StepCard(step: e.value, isLast: e.key == _steps.length - 1),
              )),
          const Gap(24),

          // ── Tier Ladder ───────────────────────────────────────────────────
          _SectionLabel(label: 'Loyalty Tiers', color: NexusColors.purple),
          const Gap(12),
          GridView.count(
            crossAxisCount: 2,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            crossAxisSpacing: 10,
            mainAxisSpacing: 10,
            childAspectRatio: 1.1,
            children: _tiers.map((t) => _TierCard(tier: t)).toList(),
          ),
          const Gap(24),

          // ── FAQ ───────────────────────────────────────────────────────────
          _SectionLabel(label: 'Common Questions', color: NexusColors.cyan),
          const Gap(12),
          ..._faqs.map((f) => Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: _FaqCard(faq: f),
              )),
          const Gap(24),

          // ── CTA ───────────────────────────────────────────────────────────
          _CtaCard(),
          const Gap(32),
        ],
      ),
    );
  }
}

// ─── Hero ─────────────────────────────────────────────────────────────────────

class _HeroSection extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: NexusRadius.lg,
        border: Border.all(color: NexusColors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            decoration: BoxDecoration(
              color: NexusColors.gold.withValues(alpha: 0.08),
              borderRadius: NexusRadius.pill,
              border: Border.all(color: NexusColors.gold.withValues(alpha: 0.25)),
            ),
            child: const Text(
              'HOW IT WORKS',
              style: TextStyle(
                fontSize: 10,
                fontWeight: FontWeight.w700,
                color: NexusColors.gold,
                letterSpacing: 1.2,
              ),
            ),
          ),
          const Gap(12),
          RichText(
            text: const TextSpan(
              style: TextStyle(
                fontSize: 24,
                fontWeight: FontWeight.w900,
                color: NexusColors.textPrimary,
                height: 1.2,
              ),
              children: [
                TextSpan(text: 'Four steps to '),
                TextSpan(
                  text: 'everything',
                  style: TextStyle(color: NexusColors.gold),
                ),
              ],
            ),
          ),
          const Gap(8),
          const Text(
            'From your first recharge to winning cash prizes and creating with AI — here\'s the complete journey.',
            style: TextStyle(
              fontSize: 13,
              color: NexusColors.textSecondary,
              height: 1.5,
            ),
          ),
        ],
      ),
    ).animate().fadeIn(duration: 400.ms).slideY(begin: 0.1, end: 0);
  }
}

// ─── Section Label ────────────────────────────────────────────────────────────

class _SectionLabel extends StatelessWidget {
  final String label;
  final Color color;
  const _SectionLabel({required this.label, required this.color});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 3,
          height: 16,
          decoration: BoxDecoration(
            color: color,
            borderRadius: NexusRadius.pill,
          ),
        ),
        const Gap(8),
        Text(
          label.toUpperCase(),
          style: TextStyle(
            fontSize: 11,
            fontWeight: FontWeight.w700,
            color: color,
            letterSpacing: 1.0,
          ),
        ),
      ],
    );
  }
}

// ─── Step Card ────────────────────────────────────────────────────────────────

class _StepCard extends StatelessWidget {
  final _Step step;
  final bool isLast;
  const _StepCard({required this.step, required this.isLast});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: NexusRadius.lg,
        border: Border.all(color: step.color.withValues(alpha: 0.2)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header row
          Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: step.color.withValues(alpha: 0.1),
                  borderRadius: NexusRadius.md,
                  border: Border.all(color: step.color.withValues(alpha: 0.3)),
                ),
                child: Center(
                  child: Text(step.emoji, style: const TextStyle(fontSize: 22)),
                ),
              ),
              const Gap(12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      step.title,
                      style: const TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w800,
                        color: NexusColors.textPrimary,
                      ),
                    ),
                    const Gap(2),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: step.color.withValues(alpha: 0.1),
                        borderRadius: NexusRadius.pill,
                        border: Border.all(color: step.color.withValues(alpha: 0.25)),
                      ),
                      child: Text(
                        step.stat,
                        style: TextStyle(
                          fontSize: 10,
                          fontWeight: FontWeight.w700,
                          color: step.color,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              Text(
                step.number,
                style: const TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w900,
                  color: Color(0x22E2E8FF),
                  fontFamily: 'monospace',
                ),
              ),
            ],
          ),
          const Gap(12),
          // Body
          Text(
            step.body,
            style: const TextStyle(
              fontSize: 13,
              color: NexusColors.textSecondary,
              height: 1.5,
            ),
          ),
          const Gap(12),
          // Details
          ...step.details.map((d) => Padding(
                padding: const EdgeInsets.only(bottom: 6),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Icon(Icons.chevron_right_rounded, size: 14, color: step.color),
                    const Gap(4),
                    Expanded(
                      child: Text(
                        d,
                        style: const TextStyle(
                          fontSize: 12,
                          color: NexusColors.textSecondary,
                          height: 1.4,
                        ),
                      ),
                    ),
                  ],
                ),
              )),
        ],
      ),
    ).animate().fadeIn(duration: 400.ms).slideY(begin: 0.08, end: 0);
  }
}

// ─── Tier Card ────────────────────────────────────────────────────────────────

class _TierCard extends StatelessWidget {
  final _Tier tier;
  const _TierCard({required this.tier});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: NexusRadius.lg,
        border: Border.all(color: tier.color.withValues(alpha: 0.2)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(tier.emoji, style: const TextStyle(fontSize: 26)),
          const Gap(6),
          Text(
            tier.tier,
            style: const TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w800,
              color: NexusColors.textPrimary,
            ),
          ),
          Text(
            tier.pts,
            style: TextStyle(
              fontSize: 10,
              fontWeight: FontWeight.w700,
              color: tier.color,
            ),
          ),
          const Gap(8),
          ...tier.perks.map((p) => Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Container(
                      width: 4,
                      height: 4,
                      margin: const EdgeInsets.only(top: 5, right: 6),
                      decoration: BoxDecoration(
                        color: tier.color,
                        shape: BoxShape.circle,
                      ),
                    ),
                    Expanded(
                      child: Text(
                        p,
                        style: const TextStyle(
                          fontSize: 10,
                          color: NexusColors.textSecondary,
                          height: 1.4,
                        ),
                      ),
                    ),
                  ],
                ),
              )),
        ],
      ),
    );
  }
}

// ─── FAQ Card ─────────────────────────────────────────────────────────────────

class _FaqCard extends StatefulWidget {
  final _Faq faq;
  const _FaqCard({required this.faq});

  @override
  State<_FaqCard> createState() => _FaqCardState();
}

class _FaqCardState extends State<_FaqCard> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () => setState(() => _expanded = !_expanded),
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          borderRadius: NexusRadius.md,
          border: Border.all(color: NexusColors.border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    widget.faq.q,
                    style: const TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w700,
                      color: NexusColors.textPrimary,
                    ),
                  ),
                ),
                Icon(
                  _expanded ? Icons.expand_less_rounded : Icons.expand_more_rounded,
                  color: NexusColors.textSecondary,
                  size: 20,
                ),
              ],
            ),
            if (_expanded) ...[
              const Gap(8),
              Text(
                widget.faq.a,
                style: const TextStyle(
                  fontSize: 13,
                  color: NexusColors.textSecondary,
                  height: 1.5,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

// ─── CTA Card ─────────────────────────────────────────────────────────────────

class _CtaCard extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: NexusColors.gold.withValues(alpha: 0.05),
        borderRadius: NexusRadius.xl,
        border: Border.all(color: NexusColors.gold.withValues(alpha: 0.2)),
      ),
      child: Column(
        children: [
          const Text('⚡', style: TextStyle(fontSize: 36)),
          const Gap(12),
          const Text(
            'Ready to start earning?',
            style: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.w900,
              color: NexusColors.textPrimary,
            ),
            textAlign: TextAlign.center,
          ),
          const Gap(8),
          const Text(
            'Recharge your MTN line and earn Pulse Points, free spins, and AI Studio access — starting today.',
            style: TextStyle(
              fontSize: 13,
              color: NexusColors.textSecondary,
              height: 1.5,
            ),
            textAlign: TextAlign.center,
          ),
          const Gap(16),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton(
              onPressed: () => context.go('/spin'),
              style: ElevatedButton.styleFrom(
                backgroundColor: NexusColors.gold,
                foregroundColor: Colors.black,
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(borderRadius: NexusRadius.md),
                textStyle: const TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w800,
                ),
              ),
              child: const Text('Go to Spin & Win'),
            ),
          ),
        ],
      ),
    );
  }
}
