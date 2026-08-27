// Remote kill switch, force-update and feature gates.
//
// WHY THIS EXISTS
// You cannot un-ship a mobile binary. Once a build is on a device it stays there
// until the user updates. That makes this the ONLY mitigation lever that does not
// require a store review cycle — closer to infrastructure than to a feature.
//
// It also solves a real problem during the tester round: when a fix ships for a
// bug a tester found, min-version gating tells them to update instead of us
// chasing people individually.
//
// ── THE GOVERNING RULE: FAIL OPEN ────────────────────────────────────────────
// Every failure path must leave the app FULLY USABLE. A misconfigured or
// unreachable config service must never brick the app — that would turn the
// safety mechanism into the outage. Concretely:
//   · fetch timeout / network failure  -> use last cached values, else defaults
//   · defaults are permissive          -> no maintenance, no forced update
//   · a malformed version string       -> treated as "no constraint"
//   · any exception                    -> swallowed, app proceeds normally
// The ONLY way a user is blocked is an explicit, well-formed remote instruction.

import 'dart:async';

import 'package:firebase_remote_config/firebase_remote_config.dart';
import 'package:flutter/foundation.dart';
import 'package:package_info_plus/package_info_plus.dart';

/// What the app should do right now, given remote config + the running version.
enum AppGateStatus {
  /// Normal operation.
  ok,

  /// A newer version exists and is worth prompting for, but this one still works.
  updateAvailable,

  /// This version is below the minimum supported — block until updated.
  updateRequired,

  /// Backend maintenance — block with a message.
  maintenance,
}

class AppGate {
  const AppGate({
    required this.status,
    this.message,
    this.storeUrl,
  });

  final AppGateStatus status;
  final String? message;
  final String? storeUrl;

  bool get blocks =>
      status == AppGateStatus.updateRequired || status == AppGateStatus.maintenance;

  static const ok = AppGate(status: AppGateStatus.ok);
}

class RemoteConfigService {
  RemoteConfigService._();
  static final RemoteConfigService instance = RemoteConfigService._();

  FirebaseRemoteConfig? _rc;
  String _currentVersion = '0.0.0';

  // Remote Config parameter keys. Create these in the Firebase console; until
  // they exist the in-app defaults below apply and everything stays permissive.
  static const _kMinVersion       = 'minimum_supported_version';
  static const _kLatestVersion    = 'latest_version';
  static const _kMaintenance      = 'maintenance_mode';
  static const _kMaintenanceMsg   = 'maintenance_message';
  static const _kUpdateMsg        = 'update_message';
  static const _kKilledFeatures   = 'killed_features';
  static const _kAndroidStoreUrl  = 'android_store_url';
  static const _kIosStoreUrl      = 'ios_store_url';

  /// Permissive by design — see the FAIL OPEN rule above. If Firebase never
  /// initialises or the first fetch fails, these are what the app runs on.
  static final Map<String, dynamic> _defaults = {
    _kMinVersion: '0.0.0',        // no floor
    _kLatestVersion: '0.0.0',     // nothing newer
    _kMaintenance: false,
    _kMaintenanceMsg: 'Loyalty Nexus is briefly unavailable while we carry out '
        'maintenance. Please try again shortly.',
    _kUpdateMsg: 'A new version of Loyalty Nexus is available.',
    _kKilledFeatures: '[]',
    _kAndroidStoreUrl:
        'https://play.google.com/store/apps/details?id=ng.loyaltynexus.loyalty_nexus',
    _kIosStoreUrl: 'https://apps.apple.com/app/loyalty-nexus/id000000000',
  };

  bool get isEnabled => _rc != null;

  /// Wires up Remote Config. Called only when Firebase actually came up.
  ///
  /// Never throws and never blocks startup for long: the fetch timeout is short
  /// and a failure simply leaves the defaults in place.
  Future<void> init(FirebaseRemoteConfig rc) async {
    try {
      _currentVersion = (await PackageInfo.fromPlatform()).version;
    } catch (_) {
      // Version unknown -> every comparison below yields "no constraint".
      _currentVersion = '0.0.0';
    }

    try {
      await rc.setDefaults(_defaults);
      await rc.setConfigSettings(RemoteConfigSettings(
        // Short: this sits on the startup path and must not stall a cold launch
        // on a poor Nigerian connection.
        fetchTimeout: const Duration(seconds: 8),
        // A kill switch that takes 12 hours to propagate is not a kill switch.
        // 15 minutes is the practical floor Firebase allows without throttling.
        minimumFetchInterval:
            kDebugMode ? Duration.zero : const Duration(minutes: 15),
      ));
      _rc = rc;
      // Activate whatever was cached from a previous run FIRST, so the app has
      // usable values immediately even if this fetch fails.
      await rc.activate();
      unawaited(_fetchInBackground(rc));
    } catch (e) {
      debugPrint('[remote_config] init failed, running on defaults: $e');
      _rc = null;
    }
  }

  Future<void> _fetchInBackground(FirebaseRemoteConfig rc) async {
    try {
      await rc.fetchAndActivate();
      debugPrint('[remote_config] fetched; min=${rc.getString(_kMinVersion)} '
          'maintenance=${rc.getBool(_kMaintenance)}');
    } catch (e) {
      // Cached/default values remain in force. Not an error worth surfacing.
      debugPrint('[remote_config] fetch failed, using cached/defaults: $e');
    }
  }

  /// Re-checks now. Used by the retry button on the blocking screen so a user is
  /// never stranded waiting for the 15-minute interval.
  Future<void> refresh() async {
    final rc = _rc;
    if (rc == null) return;
    try {
      await rc.fetchAndActivate();
    } catch (_) {/* keep cached */}
  }

  /// The current gate decision. Cheap and synchronous — safe to call in build().
  AppGate evaluate() {
    final rc = _rc;
    if (rc == null) return AppGate.ok; // fail open

    try {
      if (rc.getBool(_kMaintenance)) {
        return AppGate(
          status: AppGateStatus.maintenance,
          message: _nonEmpty(rc.getString(_kMaintenanceMsg)) ??
              _defaults[_kMaintenanceMsg] as String,
        );
      }

      final storeUrl = defaultTargetPlatform == TargetPlatform.iOS
          ? _nonEmpty(rc.getString(_kIosStoreUrl))
          : _nonEmpty(rc.getString(_kAndroidStoreUrl));

      final min = _nonEmpty(rc.getString(_kMinVersion));
      if (min != null && _isOlder(_currentVersion, min)) {
        return AppGate(
          status: AppGateStatus.updateRequired,
          message: _nonEmpty(rc.getString(_kUpdateMsg)),
          storeUrl: storeUrl,
        );
      }

      final latest = _nonEmpty(rc.getString(_kLatestVersion));
      if (latest != null && _isOlder(_currentVersion, latest)) {
        return AppGate(
          status: AppGateStatus.updateAvailable,
          message: _nonEmpty(rc.getString(_kUpdateMsg)),
          storeUrl: storeUrl,
        );
      }
    } catch (e) {
      debugPrint('[remote_config] evaluate failed, failing open: $e');
    }
    return AppGate.ok;
  }

  /// Per-feature kill switch. Lets one broken flow be disabled without blocking
  /// the whole app — the proportionate response to most incidents.
  ///
  /// Remote value is a JSON array of slugs, e.g. ["spin","website_builder"].
  bool isFeatureKilled(String slug) {
    final rc = _rc;
    if (rc == null) return false; // fail open
    try {
      final raw = rc.getString(_kKilledFeatures);
      if (raw.isEmpty) return false;
      // Cheap containment check rather than a full JSON parse: the values are
      // simple slugs and a malformed payload must not throw here.
      return raw.contains('"$slug"');
    } catch (_) {
      return false;
    }
  }

  String get currentVersion => _currentVersion;

  static String? _nonEmpty(String s) => s.trim().isEmpty ? null : s.trim();

  /// True when [version] is strictly older than [other].
  ///
  /// Lenient on purpose: anything unparseable yields false ("no constraint"),
  /// because a typo in the console must never lock users out.
  @visibleForTesting
  static bool isOlder(String version, String other) => _isOlder(version, other);

  static bool _isOlder(String version, String other) {
    final a = _parse(version);
    final b = _parse(other);
    if (a == null || b == null) return false;
    for (var i = 0; i < 3; i++) {
      if (a[i] < b[i]) return true;
      if (a[i] > b[i]) return false;
    }
    return false;
  }

  static List<int>? _parse(String v) {
    // Tolerate "1.2.3", "1.2", "1.2.3+45", "v1.2.3".
    final cleaned = v.trim().replaceFirst(RegExp(r'^v'), '').split('+').first;
    final parts = cleaned.split('.');
    if (parts.isEmpty || parts.length > 3) return null;
    final out = <int>[0, 0, 0];
    for (var i = 0; i < parts.length; i++) {
      final n = int.tryParse(parts[i]);
      if (n == null || n < 0) return null;
      out[i] = n;
    }
    return out;
  }
}
