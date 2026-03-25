import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

class NexusColors {
  static const primary    = Color(0xFF5F72F9);
  static const primaryDark= Color(0xFF4A56EE);
  static const surface    = Color(0xFF1C2038);
  static const background = Color(0xFF0F1123);
  static const card       = Color(0xFF1C2038);
  static const gold       = Color(0xFFF9C74F);
  static const green      = Color(0xFF10B981);
  static const red        = Color(0xFFF43F5E);
  static const textPrimary  = Color(0xFFE2E8FF);
  static const textSecondary= Color(0xFF828CB4);
  static const border     = Color(0x265F72F9); // 15% opacity
}

class NexusTheme {
  static ThemeData dark() {
    return ThemeData.dark().copyWith(
      scaffoldBackgroundColor: NexusColors.background,
      colorScheme: const ColorScheme.dark(
        primary:   NexusColors.primary,
        secondary: NexusColors.gold,
        surface:   NexusColors.surface,
        onPrimary: Colors.white,
        onSurface: NexusColors.textPrimary,
      ),
      textTheme: GoogleFonts.interTextTheme(ThemeData.dark().textTheme).copyWith(
        displayLarge: GoogleFonts.syne(
          fontSize: 32, fontWeight: FontWeight.w800, color: NexusColors.textPrimary,
        ),
        displayMedium: GoogleFonts.syne(
          fontSize: 24, fontWeight: FontWeight.w700, color: NexusColors.textPrimary,
        ),
        titleLarge: const TextStyle(
          fontSize: 18, fontWeight: FontWeight.w600, color: NexusColors.textPrimary,
        ),
        bodyLarge: const TextStyle(
          fontSize: 16, color: NexusColors.textPrimary,
        ),
        bodyMedium: const TextStyle(
          fontSize: 14, color: NexusColors.textSecondary,
        ),
        bodySmall: const TextStyle(
          fontSize: 12, color: NexusColors.textSecondary,
        ),
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: NexusColors.background,
        elevation: 0,
        centerTitle: false,
        titleTextStyle: TextStyle(
          fontFamily: 'Syne',
          fontSize: 20,
          fontWeight: FontWeight.w700,
          color: NexusColors.textPrimary,
        ),
        iconTheme: IconThemeData(color: NexusColors.textPrimary),
      ),
      cardTheme: CardTheme(
        color: NexusColors.card,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: const BorderSide(color: NexusColors.border),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: const Color(0xFF161830),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: NexusColors.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: NexusColors.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: NexusColors.primary, width: 1.5),
        ),
        hintStyle: const TextStyle(color: NexusColors.textSecondary),
        labelStyle: const TextStyle(color: NexusColors.textSecondary),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: NexusColors.primary,
          foregroundColor: Colors.white,
          minimumSize: const Size(double.infinity, 52),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
          textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
          elevation: 0,
        ),
      ),
      bottomNavigationBarTheme: const BottomNavigationBarThemeData(
        backgroundColor: NexusColors.surface,
        selectedItemColor: NexusColors.primary,
        unselectedItemColor: NexusColors.textSecondary,
        type: BottomNavigationBarType.fixed,
        elevation: 0,
        selectedLabelStyle: TextStyle(fontSize: 10, fontWeight: FontWeight.w600),
        unselectedLabelStyle: TextStyle(fontSize: 10),
      ),
      dividerColor: NexusColors.border,
    );
  }
}
