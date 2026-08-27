// Analytics — thin, safe wrapper over Firebase Analytics.
//
// `firebase_analytics` was a declared dependency with ZERO imports anywhere in
// lib/. The native SDK still auto-collected sessions, but there was no screen
// tracking and no custom events — so a tester round would have produced crash
// data (Crashlytics) and nothing at all about where people actually drop off.
// For a loyalty product the funnel (login → recharge → points → spin) is the
// whole game.
//
// Two design rules here:
//  1. Analytics must NEVER break the app. Every call is wrapped and swallows its
//     own errors; a reporting backend is not worth a crash.
//  2. Never log PII. No phone numbers, no OTP codes, no tokens, no names.
//     Firebase's own policy forbids it and Nigerian users' MSISDNs are
//     identifying. Log shapes and outcomes, not people.

import 'package:firebase_analytics/firebase_analytics.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

class Analytics {
  Analytics._();
  static final Analytics instance = Analytics._();

  FirebaseAnalytics? _fa;

  /// Wires up the real backend. Called only when Firebase actually came up —
  /// see `firebaseReady` in main.dart. Until then every method is a no-op.
  void attach(FirebaseAnalytics analytics) => _fa = analytics;

  bool get isEnabled => _fa != null;

  Future<void> _safe(Future<void> Function(FirebaseAnalytics a) op) async {
    final fa = _fa;
    if (fa == null) return;
    try {
      await op(fa);
    } catch (e) {
      // Deliberately swallowed. Analytics is observability, not a feature —
      // it must never surface to the user or fail a flow.
      debugPrint('[analytics] dropped: $e');
    }
  }

  /// A screen the user landed on. Driven by the GoRouter observer below.
  Future<void> screen(String name) =>
      _safe((a) => a.logScreenView(screenName: name));

  /// Sign-in completed. `isNewUser` separates activation from return visits.
  Future<void> loginSucceeded({required bool isNewUser}) => _safe((a) async {
        await a.logLogin(loginMethod: 'otp');
        await a.logEvent(
          name: 'login_succeeded',
          parameters: {'is_new_user': isNewUser ? 1 : 0},
        );
      });

  /// Sign-in failed. `reason` is a short slug — never the raw server message,
  /// which can contain identifying detail.
  Future<void> loginFailed(String reason) => _safe((a) => a.logEvent(
        name: 'login_failed',
        parameters: {'reason': reason},
      ));

  /// A points-earning or points-spending action completed.
  Future<void> action(String name, {Map<String, Object>? params}) =>
      _safe((a) => a.logEvent(name: name, parameters: params));

  /// Ties events to an account WITHOUT storing a phone number. The caller must
  /// pass an opaque server-side id, never an MSISDN.
  Future<void> setUser(String? opaqueUserId) =>
      _safe((a) => a.setUserId(id: opaqueUserId));
}

/// Reports every route change to Analytics.
///
/// GoRouter had no observers at all, so no screen views were ever recorded.
/// Registered via `observers:` on the router.
class AnalyticsRouteObserver extends NavigatorObserver {
  void _report(Route<dynamic>? route) {
    final name = route?.settings.name;
    // GoRouter sets settings.name to the matched path for top-level routes.
    // Anonymous dialogs/sheets have none — skip rather than log noise.
    if (name == null || name.isEmpty) return;
    Analytics.instance.screen(name);
  }

  @override
  void didPush(Route<dynamic> route, Route<dynamic>? previousRoute) =>
      _report(route);

  @override
  void didReplace({Route<dynamic>? newRoute, Route<dynamic>? oldRoute}) =>
      _report(newRoute);

  @override
  void didPop(Route<dynamic> route, Route<dynamic>? previousRoute) =>
      _report(previousRoute);
}
