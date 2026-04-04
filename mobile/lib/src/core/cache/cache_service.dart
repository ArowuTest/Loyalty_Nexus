import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

// ─── Cache Service ────────────────────────────────────────────────────────────
/// Thin shared_preferences wrapper for offline-first data persistence.
/// Keys are namespaced under 'nexus_cache_'.
class CacheService {
  static const _prefix = 'nexus_cache_';

  final SharedPreferences _prefs;
  CacheService(this._prefs);

  // ── Write ─────────────────────────────────────────────────────────────────

  Future<void> put(String key, Map<String, dynamic> data) async {
    await _prefs.setString('$_prefix$key', jsonEncode({
      '_ts': DateTime.now().millisecondsSinceEpoch,
      'd':   data,
    }));
  }

  Future<void> putList(String key, List<dynamic> data) async {
    await _prefs.setString('$_prefix$key', jsonEncode({
      '_ts': DateTime.now().millisecondsSinceEpoch,
      'd':   data,
    }));
  }

  // ── Read ──────────────────────────────────────────────────────────────────

  Map<String, dynamic>? getMap(String key, {int maxAgeMinutes = 60}) {
    final raw = _prefs.getString('$_prefix$key');
    if (raw == null) return null;
    try {
      final json = jsonDecode(raw) as Map<String, dynamic>;
      final ts   = json['_ts'] as int? ?? 0;
      final age  = DateTime.now().millisecondsSinceEpoch - ts;
      if (age > maxAgeMinutes * 60 * 1000) return null; // stale
      return json['d'] as Map<String, dynamic>;
    } catch (_) { return null; }
  }

  List<dynamic>? getList(String key, {int maxAgeMinutes = 60}) {
    final raw = _prefs.getString('$_prefix$key');
    if (raw == null) return null;
    try {
      final json = jsonDecode(raw) as Map<String, dynamic>;
      final ts   = json['_ts'] as int? ?? 0;
      final age  = DateTime.now().millisecondsSinceEpoch - ts;
      if (age > maxAgeMinutes * 60 * 1000) return null;
      return json['d'] as List<dynamic>;
    } catch (_) { return null; }
  }

  // ── Clear ─────────────────────────────────────────────────────────────────

  Future<void> clear(String key) =>
      _prefs.remove('$_prefix$key');

  Future<void> clearAll() async {
    final keys = _prefs.getKeys()
        .where((k) => k.startsWith(_prefix))
        .toList();
    for (final k in keys) { await _prefs.remove(k); }
  }
}

// ─── Provider ─────────────────────────────────────────────────────────────────

final cacheServiceProvider = Provider<CacheService>((ref) {
  throw UnimplementedError(
      'cacheServiceProvider must be overridden in main.dart '
      'after SharedPreferences.getInstance()');
});

// ─── Cache keys ───────────────────────────────────────────────────────────────

class CacheKeys {
  static const wallet        = 'wallet';
  static const profile       = 'profile';
  static const passport      = 'passport';
  static const leaderboard   = 'leaderboard';
  static const draws         = 'draws';
  static const transactions  = 'transactions';
  static const notifications = 'notifications';
  static const spinWheel     = 'spin_wheel';
  static const tools         = 'studio_tools';
}
