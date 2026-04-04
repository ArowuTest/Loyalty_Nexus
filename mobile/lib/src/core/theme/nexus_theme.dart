import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:google_fonts/google_fonts.dart';

// ─── Design Tokens ────────────────────────────────────────────────────────────

class NexusColors {
  // Brand
  static const primary      = Color(0xFF5F72F9);
  static const primaryDark  = Color(0xFF4A56EE);
  static const primaryGlow  = Color(0x335F72F9);

  // Surfaces
  static const background   = Color(0xFF0F1123);
  static const surface      = Color(0xFF161B35);
  static const surfaceHigh  = Color(0xFF1C2444);
  static const card         = Color(0xFF161B35);
  static const overlay      = Color(0xCC0F1123);

  // Semantic
  static const gold         = Color(0xFFF9C74F);
  static const goldDim      = Color(0x33F9C74F);
  static const green        = Color(0xFF10B981);
  static const greenDim     = Color(0x2210B981);
  static const red          = Color(0xFFF43F5E);
  static const redDim       = Color(0x22F43F5E);
  static const purple       = Color(0xFF8B5CF6);
  static const cyan         = Color(0xFF00D4FF);

  // Text
  static const textPrimary   = Color(0xFFE2E8FF);
  static const textSecondary = Color(0xFF828CB4);
  static const textMuted     = Color(0xFF4A5073);

  // Border
  static const border        = Color(0x1A5F72F9);
  static const borderMedium  = Color(0x335F72F9);

  // Tier colours — matches webapp exactly
  static const bronze   = Color(0xFFCD7F32);
  static const silver   = Color(0xFFC0C0C0);
  static const goldTier = Color(0xFFFFD700);
  static const platinum = Color(0xFFE5E4E2);
  static const diamond  = Color(0xFFB9F2FF);

  // Gradients
  static const gradientBrand = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [primaryDark, purple],
  );

  static const gradientGold = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [Color(0xFFF9C74F), Color(0xFFF5A623)],
  );

  static const gradientSuccess = LinearGradient(
    colors: [Color(0xFF10B981), Color(0xFF059669)],
  );

  static const gradientSurface = LinearGradient(
    begin: Alignment.topCenter,
    end: Alignment.bottomCenter,
    colors: [surfaceHigh, surface],
  );

  // Tier colour helper
  static Color forTier(String tier) => switch (tier.toUpperCase()) {
    'SILVER'   => silver,
    'GOLD'     => goldTier,
    'PLATINUM' => platinum,
    'DIAMOND'  => diamond,
    _          => bronze,
  };

  static String emojiForTier(String tier) => switch (tier.toUpperCase()) {
    'SILVER'   => '🥈',
    'GOLD'     => '🥇',
    'PLATINUM' => '💎',
    'DIAMOND'  => '💠',
    _          => '🥉',
  };
}

// ─── Spacing ──────────────────────────────────────────────────────────────────

class NexusSpacing {
  static const xs  = 4.0;
  static const sm  = 8.0;
  static const md  = 16.0;
  static const lg  = 24.0;
  static const xl  = 32.0;
  static const xxl = 48.0;

  static const cardPadding = EdgeInsets.all(16);
  static const pagePadding = EdgeInsets.symmetric(horizontal: 16);
  static const sectionGap  = SizedBox(height: 24);
}

// ─── Shadows ──────────────────────────────────────────────────────────────────

class NexusShadows {
  static final card = [
    BoxShadow(color: Colors.black.withValues(alpha: 0.25), blurRadius: 16, offset: const Offset(0, 4)),
  ];

  static final glow = [
    BoxShadow(color: NexusColors.primary.withValues(alpha: 0.3), blurRadius: 24, spreadRadius: 0),
  ];

  static final goldGlow = [
    BoxShadow(color: NexusColors.gold.withValues(alpha: 0.35), blurRadius: 20, spreadRadius: 0),
  ];

  static final elevated = [
    BoxShadow(color: Colors.black.withValues(alpha: 0.4), blurRadius: 32, offset: const Offset(0, 8)),
  ];
}

// ─── Radius ───────────────────────────────────────────────────────────────────

class NexusRadius {
  static const sm   = BorderRadius.all(Radius.circular(10));
  static const md   = BorderRadius.all(Radius.circular(16));
  static const lg   = BorderRadius.all(Radius.circular(20));
  static const xl   = BorderRadius.all(Radius.circular(28));
  static const pill = BorderRadius.all(Radius.circular(999));
}

// ─── Text Styles ──────────────────────────────────────────────────────────────

class NexusText {
  static TextStyle displayXl(BuildContext ctx) =>
      GoogleFonts.syne(fontSize: 40, fontWeight: FontWeight.w900, color: NexusColors.textPrimary, letterSpacing: -1);

  static TextStyle display(BuildContext ctx) =>
      GoogleFonts.syne(fontSize: 28, fontWeight: FontWeight.w800, color: NexusColors.textPrimary, letterSpacing: -0.5);

  static TextStyle heading(BuildContext ctx) =>
      GoogleFonts.syne(fontSize: 20, fontWeight: FontWeight.w700, color: NexusColors.textPrimary);

  static const subheading = TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: NexusColors.textPrimary);

  static const body = TextStyle(fontSize: 14, color: NexusColors.textPrimary, height: 1.5);

  static const bodySecondary = TextStyle(fontSize: 14, color: NexusColors.textSecondary, height: 1.5);

  static const caption = TextStyle(fontSize: 12, color: NexusColors.textSecondary);

  static const label = TextStyle(
    fontSize: 10, fontWeight: FontWeight.w700,
    color: NexusColors.textSecondary, letterSpacing: 0.8,
  );

  static const mono = TextStyle(
    fontFamily: 'monospace', fontSize: 14,
    color: NexusColors.textPrimary, letterSpacing: 2,
  );
}

// ─── Theme ────────────────────────────────────────────────────────────────────

class NexusTheme {
  static ThemeData dark() {
    final base = ThemeData.dark(useMaterial3: true);
    return base.copyWith(
      scaffoldBackgroundColor: NexusColors.background,

      colorScheme: const ColorScheme.dark(
        primary:        NexusColors.primary,
        secondary:      NexusColors.gold,
        surface:        NexusColors.surface,
        onPrimary:      Colors.white,
        onSurface:      NexusColors.textPrimary,
        error:          NexusColors.red,
        onError:        Colors.white,
        outline:        NexusColors.border,
      ),

      textTheme: GoogleFonts.interTextTheme(base.textTheme).copyWith(
        displayLarge:  GoogleFonts.syne(fontSize: 40, fontWeight: FontWeight.w900, color: NexusColors.textPrimary),
        displayMedium: GoogleFonts.syne(fontSize: 28, fontWeight: FontWeight.w800, color: NexusColors.textPrimary),
        displaySmall:  GoogleFonts.syne(fontSize: 22, fontWeight: FontWeight.w700, color: NexusColors.textPrimary),
        headlineLarge: GoogleFonts.syne(fontSize: 20, fontWeight: FontWeight.w700, color: NexusColors.textPrimary),
        headlineMedium:const TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: NexusColors.textPrimary),
        headlineSmall: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: NexusColors.textPrimary),
        titleLarge:    const TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: NexusColors.textPrimary),
        titleMedium:   const TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: NexusColors.textPrimary),
        bodyLarge:     const TextStyle(fontSize: 15, color: NexusColors.textPrimary, height: 1.6),
        bodyMedium:    const TextStyle(fontSize: 14, color: NexusColors.textPrimary, height: 1.5),
        bodySmall:     const TextStyle(fontSize: 12, color: NexusColors.textSecondary, height: 1.4),
        labelLarge:    const TextStyle(fontSize: 12, fontWeight: FontWeight.w700, color: NexusColors.textPrimary),
        labelMedium:   const TextStyle(fontSize: 11, fontWeight: FontWeight.w600, color: NexusColors.textSecondary),
        labelSmall:    const TextStyle(fontSize: 10, fontWeight: FontWeight.w700, color: NexusColors.textSecondary, letterSpacing: 0.8),
      ),

      appBarTheme: AppBarTheme(
        backgroundColor:    NexusColors.background,
        surfaceTintColor:   Colors.transparent,
        elevation:          0,
        scrolledUnderElevation: 0,
        centerTitle:        false,
        titleTextStyle:     GoogleFonts.syne(
          fontSize: 20, fontWeight: FontWeight.w700, color: NexusColors.textPrimary),
        iconTheme: const IconThemeData(color: NexusColors.textPrimary, size: 22),
        systemOverlayStyle: const SystemUiOverlayStyle(
          statusBarColor:           Colors.transparent,
          statusBarIconBrightness:  Brightness.light,
        ),
      ),

      cardTheme: CardTheme(
        color:     NexusColors.surface,
        elevation: 0,
        shape:     RoundedRectangleBorder(
          borderRadius: NexusRadius.md,
          side: const BorderSide(color: NexusColors.border),
        ),
        margin: EdgeInsets.zero,
      ),

      inputDecorationTheme: InputDecorationTheme(
        filled:    true,
        fillColor: NexusColors.background,
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        border:           OutlineInputBorder(borderRadius: NexusRadius.md, borderSide: const BorderSide(color: NexusColors.border)),
        enabledBorder:    OutlineInputBorder(borderRadius: NexusRadius.md, borderSide: const BorderSide(color: NexusColors.border)),
        focusedBorder:    OutlineInputBorder(borderRadius: NexusRadius.md, borderSide: const BorderSide(color: NexusColors.primary, width: 1.5)),
        errorBorder:      OutlineInputBorder(borderRadius: NexusRadius.md, borderSide: const BorderSide(color: NexusColors.red)),
        focusedErrorBorder: OutlineInputBorder(borderRadius: NexusRadius.md, borderSide: const BorderSide(color: NexusColors.red, width: 1.5)),
        hintStyle:   const TextStyle(color: NexusColors.textMuted, fontSize: 14),
        labelStyle:  const TextStyle(color: NexusColors.textSecondary, fontSize: 14),
        errorStyle:  const TextStyle(color: NexusColors.red, fontSize: 12),
        prefixIconColor: NexusColors.textSecondary,
        suffixIconColor: NexusColors.textSecondary,
      ),

      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: NexusColors.primary,
          foregroundColor: Colors.white,
          disabledBackgroundColor: NexusColors.primary.withValues(alpha: 0.4),
          disabledForegroundColor: Colors.white54,
          minimumSize:    const Size(double.infinity, 52),
          shape:          RoundedRectangleBorder(borderRadius: NexusRadius.md),
          textStyle:      const TextStyle(fontSize: 15, fontWeight: FontWeight.w700, letterSpacing: 0.3),
          elevation: 0,
        ),
      ),

      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: NexusColors.textPrimary,
          side: const BorderSide(color: NexusColors.border),
          minimumSize: const Size(double.infinity, 52),
          shape: RoundedRectangleBorder(borderRadius: NexusRadius.md),
          textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
        ),
      ),

      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: NexusColors.primary,
          textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
        ),
      ),

      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((s) =>
            s.contains(WidgetState.selected) ? Colors.white : NexusColors.textMuted),
        trackColor: WidgetStateProperty.resolveWith((s) =>
            s.contains(WidgetState.selected)
                ? NexusColors.primary
                : NexusColors.border),
        trackOutlineColor: WidgetStateProperty.all(Colors.transparent),
      ),

      chipTheme: ChipThemeData(
        backgroundColor: NexusColors.surfaceHigh,
        selectedColor:   NexusColors.primary.withValues(alpha: 0.2),
        labelStyle: const TextStyle(fontSize: 12, color: NexusColors.textPrimary),
        side: const BorderSide(color: NexusColors.border),
        shape: const StadiumBorder(),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      ),

      tabBarTheme: const TabBarTheme(
        labelColor:         NexusColors.textPrimary,
        unselectedLabelColor: NexusColors.textSecondary,
        indicatorColor:     NexusColors.primary,
        dividerColor:       NexusColors.border,
        labelStyle:         TextStyle(fontSize: 13, fontWeight: FontWeight.w700),
        unselectedLabelStyle: TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
      ),

      dialogTheme: DialogTheme(
        backgroundColor: NexusColors.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 8,
        shape: RoundedRectangleBorder(borderRadius: NexusRadius.lg),
        titleTextStyle: const TextStyle(fontSize: 18, fontWeight: FontWeight.w800, color: NexusColors.textPrimary),
        contentTextStyle: const TextStyle(fontSize: 14, color: NexusColors.textSecondary, height: 1.5),
      ),

      bottomSheetTheme: const BottomSheetThemeData(
        backgroundColor:     NexusColors.surface,
        surfaceTintColor:    Colors.transparent,
        modalBackgroundColor: NexusColors.surface,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
        ),
      ),

      snackBarTheme: SnackBarThemeData(
        backgroundColor:    NexusColors.surfaceHigh,
        contentTextStyle:   const TextStyle(color: NexusColors.textPrimary, fontSize: 14),
        behavior:           SnackBarBehavior.floating,
        shape:              RoundedRectangleBorder(borderRadius: NexusRadius.md),
        actionTextColor:    NexusColors.primary,
      ),

      navigationBarTheme: NavigationBarThemeData(
        backgroundColor:  NexusColors.surface,
        indicatorColor:   NexusColors.primaryGlow,
        surfaceTintColor: Colors.transparent,
        labelTextStyle: WidgetStateProperty.resolveWith((s) =>
          TextStyle(
            fontSize: 10,
            fontWeight: FontWeight.w700,
            color: s.contains(WidgetState.selected)
                ? NexusColors.primary
                : NexusColors.textSecondary,
          ),
        ),
        iconTheme: WidgetStateProperty.resolveWith((s) => IconThemeData(
          color: s.contains(WidgetState.selected)
              ? NexusColors.primary
              : NexusColors.textSecondary,
          size: 22,
        )),
      ),

      dividerTheme: const DividerThemeData(
        color: NexusColors.border,
        thickness: 1,
        space: 1,
      ),

      progressIndicatorTheme: const ProgressIndicatorThemeData(
        color: NexusColors.primary,
        linearMinHeight: 2,
      ),
    );
  }
}

// ─── Shimmer extension ───────────────────────────────────────────────────────

class NexusShimmer extends StatefulWidget {
  final double width, height;
  final BorderRadius? radius;
  const NexusShimmer({
    super.key,
    required this.width,
    required this.height,
    this.radius,
  });

  @override
  State<NexusShimmer> createState() => _NexusShimmerState();
}

class _NexusShimmerState extends State<NexusShimmer>
    with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;
  late Animation<double> _anim;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 1200));
    _anim = Tween<double>(begin: -2, end: 2).animate(
        CurvedAnimation(parent: _ctrl, curve: Curves.easeInOut));
    _ctrl.repeat();
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _anim,
      builder: (_, __) => Container(
        width: widget.width,
        height: widget.height,
        decoration: BoxDecoration(
          borderRadius: widget.radius ?? NexusRadius.sm,
          gradient: LinearGradient(
            begin: Alignment(_anim.value - 1, 0),
            end: Alignment(_anim.value, 0),
            colors: const [
              Color(0xFF1C2444),
              Color(0xFF2A3260),
              Color(0xFF1C2444),
            ],
          ),
        ),
      ),
    );
  }
}

// ─── Card widget ──────────────────────────────────────────────────────────────

class NexusCard extends StatelessWidget {
  final Widget child;
  final EdgeInsetsGeometry? padding;
  final Color? color;
  final BorderRadius? radius;
  final List<BoxShadow>? shadows;
  final Gradient? gradient;
  final VoidCallback? onTap;
  final Border? border;

  const NexusCard({
    super.key,
    required this.child,
    this.padding,
    this.color,
    this.radius,
    this.shadows,
    this.gradient,
    this.onTap,
    this.border,
  });

  @override
  Widget build(BuildContext context) {
    Widget content = Container(
      padding: padding ?? NexusSpacing.cardPadding,
      decoration: BoxDecoration(
        color: gradient == null ? (color ?? NexusColors.surface) : null,
        gradient: gradient,
        borderRadius: radius ?? NexusRadius.md,
        border: border ?? Border.all(color: NexusColors.border),
        boxShadow: shadows ?? NexusShadows.card,
      ),
      child: child,
    );

    if (onTap != null) {
      content = GestureDetector(onTap: onTap, child: content);
    }
    return content;
  }
}

// ─── Section label ───────────────────────────────────────────────────────────

class NexusSectionLabel extends StatelessWidget {
  final String text;
  final Widget? trailing;
  const NexusSectionLabel(this.text, {super.key, this.trailing});

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 10, left: 2),
    child: Row(children: [
      Expanded(child: Text(text.toUpperCase(), style: NexusText.label)),
      if (trailing != null) trailing!,
    ]),
  );
}

// ─── Error / empty states ────────────────────────────────────────────────────

class NexusErrorState extends StatelessWidget {
  final String message;
  final VoidCallback? onRetry;
  const NexusErrorState({super.key, required this.message, this.onRetry});

  @override
  Widget build(BuildContext context) => Center(child: Column(
    mainAxisAlignment: MainAxisAlignment.center,
    children: [
      const Icon(Icons.wifi_off_rounded, size: 52, color: NexusColors.textSecondary),
      const SizedBox(height: 16),
      Text(message, style: NexusText.bodySecondary, textAlign: TextAlign.center),
      if (onRetry != null) ...[
        const SizedBox(height: 20),
        OutlinedButton.icon(
          onPressed: onRetry,
          icon: const Icon(Icons.refresh_rounded, size: 16),
          label: const Text('Retry'),
          style: OutlinedButton.styleFrom(
            minimumSize: const Size(140, 44),
            foregroundColor: NexusColors.primary,
            side: const BorderSide(color: NexusColors.primary),
          ),
        ),
      ],
    ],
  ));
}

class NexusEmptyState extends StatelessWidget {
  final String emoji, title, subtitle;
  const NexusEmptyState({super.key, required this.emoji, required this.title, required this.subtitle});

  @override
  Widget build(BuildContext context) => Center(child: Column(
    mainAxisAlignment: MainAxisAlignment.center,
    children: [
      Text(emoji, style: const TextStyle(fontSize: 56)),
      const SizedBox(height: 16),
      Text(title, style: const TextStyle(
          color: NexusColors.textPrimary, fontSize: 16, fontWeight: FontWeight.w700)),
      const SizedBox(height: 6),
      Text(subtitle, style: NexusText.bodySecondary, textAlign: TextAlign.center),
    ],
  ));
}
