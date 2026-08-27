// Regression tests for the two classes of bug that actually shipped.
//
// These are deliberately not "coverage" tests. Each one exists because a real
// defect got all the way to a green CI run without being caught:
//
//  1. Theme construction — Flutter renamed CardTheme/TabBarTheme/DialogTheme to
//     *ThemeData. The app stopped compiling entirely and nothing noticed,
//     because CI ran no tests at all.
//  2. Push route mapping — 'spin_result' pointed at '/spin/prizes', which was
//     never a registered route, so the most common notification in a
//     spin-to-win app dropped testers on go_router's raw error page.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:loyalty_nexus/src/core/theme/nexus_theme.dart';
import 'package:loyalty_nexus/src/core/notifications/push_notification_service.dart';

/// Every route registered in app_router.dart. Kept as a literal list on purpose:
/// building the real GoRouter here would drag in Firebase and secure storage.
/// If you add a route, add it here too.
const _registeredRoutes = <String>{
  '/',
  '/arcade',
  '/dashboard',
  '/draws',
  '/how-it-works',
  '/notifications',
  '/passport',
  '/prizes',
  '/profile',
  '/pulse-awards',
  '/recharge',
  '/recharge/success',
  '/register',
  '/settings',
  '/spin',
  '/studio',
  '/transactions',
  '/wars',
};

void main() {
  group('NexusTheme', () {
    test('dark theme builds without throwing', () {
      // Guards against Flutter theme-API renames (the *ThemeData break).
      expect(NexusTheme.dark, returnsNormally);
    });

    test('dark theme is actually dark and uses Material 3', () {
      final theme = NexusTheme.dark();
      expect(theme.brightness, Brightness.dark);
      expect(theme.useMaterial3, isTrue);
    });

    testWidgets('theme renders in a real MaterialApp', (tester) async {
      // Catches theme values that only fail once a widget tree is built.
      await tester.pumpWidget(MaterialApp(
        theme: NexusTheme.dark(),
        home: const Scaffold(body: Text('ok')),
      ));
      expect(find.text('ok'), findsOneWidget);
    });
  });

  group('push notification routing', () {
    test('every mapped notification type resolves to a REGISTERED route', () {
      // The actual regression: '/spin/prizes' was not a route.
      const types = <String?>[
        'spin_result',
        'draw_winner',
        'draw_result',
        'point_credit',
        'bonus_award',
        'war_update',
        'war_rank',
        'passport_update',
        'prize_pending',
        null,
        'some_unknown_future_type',
      ];

      for (final t in types) {
        final route = PushNotificationService.typeToRoute(t);
        expect(
          _registeredRoutes.contains(route),
          isTrue,
          reason: "type '$t' maps to '$route', which is not a registered route "
              '— testers tapping this notification land on an error page',
        );
      }
    });

    test('spin_result specifically maps to /prizes', () {
      expect(PushNotificationService.typeToRoute('spin_result'), '/prizes');
    });

    test('unknown and null types fall back to a safe route', () {
      expect(_registeredRoutes.contains(PushNotificationService.typeToRoute(null)), isTrue);
      expect(_registeredRoutes.contains(PushNotificationService.typeToRoute('nonsense')), isTrue);
    });
  });
}
