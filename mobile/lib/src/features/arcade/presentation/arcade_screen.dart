import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Nexus Games Arcade Screen ────────────────────────────────────────────────
class ArcadeScreen extends StatelessWidget {
  const ArcadeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: NexusColors.background,
      body: CustomScrollView(
        slivers: [
          // ── Header ──────────────────────────────────────────────────────────
          SliverAppBar(
            expandedHeight: 200,
            pinned: true,
            backgroundColor: NexusColors.background,
            surfaceTintColor: Colors.transparent,
            flexibleSpace: FlexibleSpaceBar(
              background: _ArcadeHeader(),
              collapseMode: CollapseMode.parallax,
            ),
            title: const Text(
              'Nexus Games Arcade',
              style: TextStyle(
                color: NexusColors.textPrimary,
                fontWeight: FontWeight.w800,
                fontSize: 18,
              ),
            ),
          ),

          // ── Content ─────────────────────────────────────────────────────────
          SliverPadding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 100),
            sliver: SliverList(
              delegate: SliverChildListDelegate([
                // Coming soon hero
                _ComingSoonHero(),
                const SizedBox(height: 28),

                // Points earning explainer
                _EarningCard(),
                const SizedBox(height: 20),

                // Game category previews
                const _SectionLabel('Game Categories'),
                const SizedBox(height: 12),
                _GameCategoryGrid(),
                const SizedBox(height: 28),

                // Notify me CTA
                _NotifyCard(),
                const SizedBox(height: 20),
              ]),
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Arcade Header ────────────────────────────────────────────────────────────
class _ArcadeHeader extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [
            Color(0xFF1A0533),
            Color(0xFF0D1B4B),
            Color(0xFF0F1123),
          ],
        ),
      ),
      child: Stack(
        children: [
          // Decorative glow orbs
          Positioned(
            top: -40,
            right: -40,
            child: Container(
              width: 180,
              height: 180,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: RadialGradient(
                  colors: [
                    NexusColors.purple.withOpacity(0.3),
                    Colors.transparent,
                  ],
                ),
              ),
            ),
          ),
          Positioned(
            bottom: -20,
            left: -20,
            child: Container(
              width: 140,
              height: 140,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: RadialGradient(
                  colors: [
                    NexusColors.primary.withOpacity(0.25),
                    Colors.transparent,
                  ],
                ),
              ),
            ),
          ),
          // Floating game icons
          Positioned(
            top: 40,
            left: 24,
            child: Text('🎮', style: TextStyle(fontSize: 32))
                .animate(onPlay: (c) => c.repeat())
                .moveY(begin: 0, end: -8, duration: 2000.ms, curve: Curves.easeInOut)
                .then()
                .moveY(begin: -8, end: 0, duration: 2000.ms, curve: Curves.easeInOut),
          ),
          Positioned(
            top: 55,
            right: 60,
            child: Text('🕹️', style: TextStyle(fontSize: 26))
                .animate(onPlay: (c) => c.repeat())
                .moveY(begin: 0, end: -6, duration: 1800.ms, curve: Curves.easeInOut)
                .then()
                .moveY(begin: -6, end: 0, duration: 1800.ms, curve: Curves.easeInOut),
          ),
          Positioned(
            bottom: 30,
            right: 24,
            child: Text('🏆', style: TextStyle(fontSize: 28))
                .animate(onPlay: (c) => c.repeat())
                .moveY(begin: 0, end: -7, duration: 2200.ms, curve: Curves.easeInOut)
                .then()
                .moveY(begin: -7, end: 0, duration: 2200.ms, curve: Curves.easeInOut),
          ),
          // Center content
          Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const SizedBox(height: 40),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                      colors: [NexusColors.purple, NexusColors.primary],
                    ),
                    borderRadius: BorderRadius.circular(20),
                  ),
                  child: const Text(
                    'COMING SOON',
                    style: TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.w900,
                      fontSize: 11,
                      letterSpacing: 2,
                    ),
                  ),
                ),
                const SizedBox(height: 10),
                const Text(
                  'Nexus Games Arcade',
                  style: TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w900,
                    fontSize: 22,
                    letterSpacing: -0.5,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Coming Soon Hero ─────────────────────────────────────────────────────────
class _ComingSoonHero extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return NexusCard(
      gradient: const LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: [Color(0xFF1A0533), Color(0xFF0D1B4B)],
      ),
      border: Border.all(color: NexusColors.purple.withOpacity(0.4)),
      child: Column(
        children: [
          const Text('🎮', style: TextStyle(fontSize: 56))
              .animate(onPlay: (c) => c.repeat())
              .scale(begin: const Offset(1, 1), end: const Offset(1.08, 1.08), duration: 1500.ms)
              .then()
              .scale(begin: const Offset(1.08, 1.08), end: const Offset(1, 1), duration: 1500.ms),
          const SizedBox(height: 16),
          const Text(
            'Play Games. Earn Pulse Points.',
            style: TextStyle(
              color: NexusColors.textPrimary,
              fontWeight: FontWeight.w900,
              fontSize: 20,
              height: 1.2,
            ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 10),
          const Text(
            'The Nexus Games Arcade is coming soon. Play fun mini-games and earn Pulse Points that you can use across the entire platform.',
            style: TextStyle(
              color: NexusColors.textSecondary,
              fontSize: 14,
              height: 1.6,
            ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 20),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                colors: [NexusColors.purple, NexusColors.primary],
              ),
              borderRadius: BorderRadius.circular(30),
              boxShadow: [
                BoxShadow(
                  color: NexusColors.purple.withOpacity(0.4),
                  blurRadius: 20,
                  offset: const Offset(0, 6),
                ),
              ],
            ),
            child: const Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.rocket_launch_rounded, color: Colors.white, size: 16),
                SizedBox(width: 8),
                Text(
                  'Launching Soon',
                  style: TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w800,
                    fontSize: 14,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    )
        .animate()
        .fadeIn(duration: 500.ms)
        .slideY(begin: 0.1, end: 0, duration: 500.ms, curve: Curves.easeOut);
  }
}

// ─── Earning Card ─────────────────────────────────────────────────────────────
class _EarningCard extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return NexusCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  gradient: const LinearGradient(
                    colors: [NexusColors.gold, Color(0xFFF5A623)],
                  ),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Center(
                  child: Text('⚡', style: TextStyle(fontSize: 22)),
                ),
              ),
              const SizedBox(width: 12),
              const Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Earn Pulse Points',
                      style: TextStyle(
                        color: NexusColors.textPrimary,
                        fontWeight: FontWeight.w800,
                        fontSize: 15,
                      ),
                    ),
                    Text(
                      'Every game you play earns you points',
                      style: TextStyle(
                        color: NexusColors.textSecondary,
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          const _EarningRow(icon: '🎯', label: 'Complete a game level', points: '+5 pts'),
          const _EarningRow(icon: '🏆', label: 'Win a game tournament', points: '+50 pts'),
          const _EarningRow(icon: '⭐', label: 'Daily game streak bonus', points: '+10 pts'),
          const _EarningRow(icon: '🎪', label: 'First game of the day', points: '+2 pts'),
        ],
      ),
    )
        .animate()
        .fadeIn(delay: 150.ms, duration: 500.ms)
        .slideY(begin: 0.1, end: 0, duration: 500.ms, curve: Curves.easeOut);
  }
}

class _EarningRow extends StatelessWidget {
  final String icon, label, points;
  const _EarningRow({required this.icon, required this.label, required this.points});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(
        children: [
          Text(icon, style: const TextStyle(fontSize: 16)),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              label,
              style: const TextStyle(
                color: NexusColors.textSecondary,
                fontSize: 13,
              ),
            ),
          ),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: NexusColors.gold.withOpacity(0.15),
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: NexusColors.gold.withOpacity(0.3)),
            ),
            child: Text(
              points,
              style: const TextStyle(
                color: NexusColors.gold,
                fontWeight: FontWeight.w700,
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Game Category Grid ───────────────────────────────────────────────────────
class _GameCategoryGrid extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    const categories = [
      _GameCategory(emoji: '🧩', name: 'Puzzle Games', desc: 'Brain teasers & logic'),
      _GameCategory(emoji: '🎲', name: 'Chance Games', desc: 'Luck-based fun'),
      _GameCategory(emoji: '⚡', name: 'Speed Games', desc: 'Quick reaction games'),
      _GameCategory(emoji: '🃏', name: 'Card Games', desc: 'Classic card play'),
      _GameCategory(emoji: '🎯', name: 'Skill Games', desc: 'Test your accuracy'),
      _GameCategory(emoji: '🌍', name: 'Trivia', desc: 'Nigeria & world knowledge'),
    ];

    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        crossAxisSpacing: 12,
        mainAxisSpacing: 12,
        childAspectRatio: 1.5,
      ),
      itemCount: categories.length,
      itemBuilder: (context, i) {
        final cat = categories[i];
        return _GameCategoryCard(category: cat)
            .animate()
            .fadeIn(delay: Duration(milliseconds: 100 * i), duration: 400.ms)
            .slideY(begin: 0.1, end: 0, duration: 400.ms, curve: Curves.easeOut);
      },
    );
  }
}

class _GameCategory {
  final String emoji, name, desc;
  const _GameCategory({required this.emoji, required this.name, required this.desc});
}

class _GameCategoryCard extends StatelessWidget {
  final _GameCategory category;
  const _GameCategoryCard({required this.category});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: NexusColors.border),
      ),
      child: Stack(
        children: [
          // Coming soon overlay
          Positioned.fill(
            child: Container(
              decoration: BoxDecoration(
                color: NexusColors.background.withOpacity(0.4),
                borderRadius: BorderRadius.circular(16),
              ),
            ),
          ),
          // Content
          Padding(
            padding: const EdgeInsets.all(14),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text(category.emoji, style: const TextStyle(fontSize: 28)),
                const SizedBox(height: 6),
                Text(
                  category.name,
                  style: const TextStyle(
                    color: NexusColors.textPrimary,
                    fontWeight: FontWeight.w700,
                    fontSize: 13,
                  ),
                ),
                Text(
                  category.desc,
                  style: const TextStyle(
                    color: NexusColors.textSecondary,
                    fontSize: 11,
                  ),
                ),
              ],
            ),
          ),
          // Lock badge
          Positioned(
            top: 8,
            right: 8,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                color: NexusColors.surfaceHigh,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: NexusColors.border),
              ),
              child: const Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.lock_outline_rounded,
                      size: 10, color: NexusColors.textMuted),
                  SizedBox(width: 3),
                  Text('Soon',
                      style: TextStyle(
                          color: NexusColors.textMuted,
                          fontSize: 9,
                          fontWeight: FontWeight.w700)),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Notify Card ──────────────────────────────────────────────────────────────
class _NotifyCard extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return NexusCard(
      gradient: LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: [
          NexusColors.primary.withOpacity(0.15),
          NexusColors.purple.withOpacity(0.1),
        ],
      ),
      border: Border.all(color: NexusColors.primary.withOpacity(0.3)),
      child: Row(
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: NexusColors.primary.withOpacity(0.15),
              borderRadius: BorderRadius.circular(14),
            ),
            child: const Center(
              child: Icon(Icons.notifications_outlined,
                  color: NexusColors.primary, size: 24),
            ),
          ),
          const SizedBox(width: 14),
          const Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Get Notified at Launch',
                  style: TextStyle(
                    color: NexusColors.textPrimary,
                    fontWeight: FontWeight.w700,
                    fontSize: 14,
                  ),
                ),
                SizedBox(height: 2),
                Text(
                  'We\'ll send you a push notification the moment the Arcade goes live.',
                  style: TextStyle(
                    color: NexusColors.textSecondary,
                    fontSize: 12,
                    height: 1.4,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 10),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                colors: [NexusColors.primary, NexusColors.primaryDark],
              ),
              borderRadius: BorderRadius.circular(10),
            ),
            child: const Text(
              'Notify Me',
              style: TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.w700,
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    )
        .animate()
        .fadeIn(delay: 400.ms, duration: 500.ms)
        .slideY(begin: 0.1, end: 0, duration: 500.ms, curve: Curves.easeOut);
  }
}

// ─── Section Label ────────────────────────────────────────────────────────────
class _SectionLabel extends StatelessWidget {
  final String text;
  const _SectionLabel(this.text);

  @override
  Widget build(BuildContext context) {
    return Text(
      text.toUpperCase(),
      style: const TextStyle(
        color: NexusColors.textSecondary,
        fontSize: 11,
        fontWeight: FontWeight.w700,
        letterSpacing: 1.2,
      ),
    );
  }
}
