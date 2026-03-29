import 'package:flutter/material.dart';
import 'package:shimmer/shimmer.dart';
import '../theme/nexus_theme.dart';

// ─── Nexus Card (standard card with consistent padding and border) ────────────
class NexusCard extends StatelessWidget {
  final Widget child;
  final EdgeInsetsGeometry? padding;
  final VoidCallback? onTap;
  final Color? color;

  const NexusCard({
    super.key,
    required this.child,
    this.padding,
    this.onTap,
    this.color,
  });

  @override
  Widget build(BuildContext context) {
    final container = Container(
      width: double.infinity,
      padding: padding ?? const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: color ?? NexusColors.surfaceCard,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: NexusColors.border),
      ),
      child: child,
    );
    if (onTap != null) {
      return GestureDetector(
        onTap: onTap,
        behavior: HitTestBehavior.opaque,
        child: container,
      );
    }
    return container;
  }
}

// ─── Nexus Divider ────────────────────────────────────────────────────────────
class NexusDivider extends StatelessWidget {
  const NexusDivider({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 1,
      color: NexusColors.border,
    );
  }
}

// ─── Glass Card ───────────────────────────────────────────────────────────────
/// Replicates the web glassmorphism card style
class GlassCard extends StatelessWidget {
  final Widget child;
  final EdgeInsetsGeometry? padding;
  final Color? borderColor;
  final Color? backgroundColor;
  final double radius;
  final VoidCallback? onTap;

  const GlassCard({
    super.key,
    required this.child,
    this.padding,
    this.borderColor,
    this.backgroundColor,
    this.radius = 16,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final container = Container(
      padding: padding ?? const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: backgroundColor ?? NexusColors.surfaceCard,
        borderRadius: BorderRadius.circular(radius),
        border: Border.all(color: borderColor ?? NexusColors.border),
      ),
      child: child,
    );
    if (onTap != null) {
      return GestureDetector(
        onTap: onTap,
        behavior: HitTestBehavior.opaque,
        child: container,
      );
    }
    return container;
  }
}

// ─── Gradient Card ────────────────────────────────────────────────────────────
class GradientCard extends StatelessWidget {
  final Widget child;
  final List<Color> colors;
  final Color borderColor;
  final EdgeInsetsGeometry? padding;
  final double radius;

  const GradientCard({
    super.key,
    required this.child,
    required this.colors,
    required this.borderColor,
    this.padding,
    this.radius = 20,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: padding ?? const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: colors,
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(radius),
        border: Border.all(color: borderColor),
      ),
      child: child,
    );
  }
}

// ─── Gold Button ──────────────────────────────────────────────────────────────
class GoldButton extends StatelessWidget {
  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;
  final bool isLoading;
  final double? height;

  const GoldButton({
    super.key,
    required this.label,
    this.onPressed,
    this.icon,
    this.isLoading = false,
    this.height,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: height ?? 52,
      width: double.infinity,
      child: DecoratedBox(
        decoration: BoxDecoration(
          gradient: const LinearGradient(
            colors: [Color(0xFFF5A623), Color(0xFFE8941A)],
          ),
          borderRadius: BorderRadius.circular(14),
          boxShadow: [
            BoxShadow(
              color: NexusColors.gold.withOpacity(0.3),
              blurRadius: 16,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        child: ElevatedButton(
          onPressed: isLoading ? null : onPressed,
          style: ElevatedButton.styleFrom(
            backgroundColor: Colors.transparent,
            shadowColor: Colors.transparent,
            foregroundColor: Colors.black,
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
          ),
          child: isLoading
              ? const SizedBox(
                  width: 20, height: 20,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: Colors.black),
                )
              : Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    if (icon != null) ...[
                      Icon(icon, size: 18, color: Colors.black),
                      const SizedBox(width: 8),
                    ],
                    Text(label,
                        style: const TextStyle(
                            fontSize: 15,
                            fontWeight: FontWeight.w800,
                            color: Colors.black,
                            letterSpacing: 0.2)),
                  ],
                ),
        ),
      ),
    );
  }
}

// ─── Tier Badge ───────────────────────────────────────────────────────────────
class TierBadge extends StatelessWidget {
  final String tier;
  final bool large;

  const TierBadge({super.key, required this.tier, this.large = false});

  @override
  Widget build(BuildContext context) {
    final color = NexusColors.tierColor(tier);
    final emoji = NexusColors.tierEmoji(tier);
    return Container(
      padding: EdgeInsets.symmetric(
          horizontal: large ? 12 : 8, vertical: large ? 6 : 3),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(large ? 12 : 8),
        border: Border.all(color: color.withOpacity(0.3)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(emoji, style: TextStyle(fontSize: large ? 14 : 11)),
          const SizedBox(width: 4),
          Text(
            tier.toUpperCase(),
            style: TextStyle(
              color: color,
              fontSize: large ? 13 : 10,
              fontWeight: FontWeight.w800,
              letterSpacing: 0.5,
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Shimmer Loading Box ──────────────────────────────────────────────────────
class ShimmerBox extends StatelessWidget {
  final double width;
  final double height;
  final double radius;

  const ShimmerBox({
    super.key,
    this.width = double.infinity,
    required this.height,
    this.radius = 8,
  });

  @override
  Widget build(BuildContext context) {
    return Shimmer.fromColors(
      baseColor: NexusColors.surfaceCard,
      highlightColor: NexusColors.surfaceElevated,
      child: Container(
        width: width,
        height: height,
        decoration: BoxDecoration(
          color: NexusColors.surfaceCard,
          borderRadius: BorderRadius.circular(radius),
        ),
      ),
    );
  }
}

// ─── Section Header ───────────────────────────────────────────────────────────
class SectionHeader extends StatelessWidget {
  final String title;
  final String? actionLabel;
  final VoidCallback? onAction;

  const SectionHeader({
    super.key,
    required this.title,
    this.actionLabel,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(title, style: Theme.of(context).textTheme.titleMedium),
        if (actionLabel != null)
          TextButton(
            onPressed: onAction,
            style: TextButton.styleFrom(
              padding: EdgeInsets.zero,
              minimumSize: Size.zero,
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            ),
            child: Row(
              children: [
                Text(actionLabel!,
                    style: const TextStyle(
                        color: NexusColors.primary,
                        fontSize: 12,
                        fontWeight: FontWeight.w600)),
                const SizedBox(width: 2),
                const Icon(Icons.chevron_right,
                    color: NexusColors.primary, size: 14),
              ],
            ),
          ),
      ],
    );
  }
}

// ─── Empty State ──────────────────────────────────────────────────────────────
class EmptyState extends StatelessWidget {
  final String emoji;
  final String title;
  final String subtitle;
  final String? buttonLabel;
  final VoidCallback? onButton;

  const EmptyState({
    super.key,
    required this.emoji,
    required this.title,
    required this.subtitle,
    this.buttonLabel,
    this.onButton,
  });

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(40),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(emoji, style: const TextStyle(fontSize: 56)),
            const SizedBox(height: 16),
            Text(title,
                style: Theme.of(context).textTheme.titleLarge,
                textAlign: TextAlign.center),
            const SizedBox(height: 8),
            Text(subtitle,
                style: Theme.of(context).textTheme.bodyMedium,
                textAlign: TextAlign.center),
            if (buttonLabel != null) ...[
              const SizedBox(height: 24),
              SizedBox(
                width: 180,
                child: FilledButton(
                  onPressed: onButton,
                  child: Text(buttonLabel!),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

// ─── Stat Chip ────────────────────────────────────────────────────────────────
class StatChip extends StatelessWidget {
  final String label;
  final String value;
  final Color? color;

  const StatChip({
    super.key,
    required this.label,
    required this.value,
    this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: NexusColors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label,
              style: const TextStyle(
                  color: NexusColors.textSecondary,
                  fontSize: 10,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 0.5)),
          const SizedBox(height: 2),
          Text(value,
              style: TextStyle(
                  color: color ?? NexusColors.textPrimary,
                  fontSize: 18,
                  fontWeight: FontWeight.w800)),
        ],
      ),
    );
  }
}

// ─── Top Shimmer Line ─────────────────────────────────────────────────────────
class TopShimmerLine extends StatelessWidget {
  final Color color;
  const TopShimmerLine({super.key, required this.color});

  @override
  Widget build(BuildContext context) {
    return Positioned(
      top: 0, left: 0, right: 0,
      child: Container(
        height: 2,
        decoration: BoxDecoration(
          gradient: LinearGradient(
            colors: [
              Colors.transparent,
              color.withOpacity(0.6),
              Colors.transparent,
            ],
          ),
        ),
      ),
    );
  }
}

// ─── Number formatter ─────────────────────────────────────────────────────────
String fmtPoints(int n) {
  if (n >= 1000000) return '${(n / 1000000).toStringAsFixed(1)}M';
  if (n >= 1000) return '${(n / 1000).toStringAsFixed(1)}K';
  return n.toString();
}

String fmtNaira(int kobo) {
  final naira = kobo ~/ 100;
  if (naira >= 1000000) return '₦${(naira / 1000000).toStringAsFixed(1)}M';
  if (naira >= 1000) return '₦${(naira / 1000).toStringAsFixed(0)}K';
  return '₦${naira.toLocaleString()}';
}

extension IntFormat on int {
  String toLocaleString() {
    final s = toString();
    final buffer = StringBuffer();
    for (var i = 0; i < s.length; i++) {
      if (i > 0 && (s.length - i) % 3 == 0) buffer.write(',');
      buffer.write(s[i]);
    }
    return buffer.toString();
  }
}
