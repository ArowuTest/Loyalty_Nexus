import 'dart:async';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_crashlytics/firebase_crashlytics.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'src/core/theme/nexus_theme.dart';
import 'src/core/router/app_router.dart';
import 'src/core/cache/cache_service.dart';
import 'src/core/notifications/push_notification_service.dart';

/// True when Firebase came up. When false the app still runs — it just has no
/// push and no crash reporting. Losing Firebase must never mean a white screen.
bool firebaseReady = false;

void main() {
  // runZonedGuarded catches async errors that escape the widget tree. Without it
  // an uncaught Future error is written to a console nobody can read on a
  // tester's device.
  runZonedGuarded<Future<void>>(() async {
    WidgetsFlutterBinding.ensureInitialized();

    // Fonts: Inter + Syne are BUNDLED (see pubspec assets/fonts) and applied via
    // `fontFamily`. google_fonts has been removed entirely — it fetched Inter from
    // fonts.gstatic.com while BUILDING the theme, so first launch on a poor
    // connection rendered a fallback font and re-laid-out when the download landed.
    // Nothing here touches the network.

    // ── Firebase: guarded, never fatal ──────────────────────────────────────
    // Previously an unguarded `await Firebase.initializeApp()` sat between
    // ensureInitialized() and runApp(). If it threw or hung — bad APNs config,
    // Play Services trouble, missing plist — runApp was never reached and the
    // user stared at a white screen forever with no message and no retry.
    try {
      await Firebase.initializeApp();
      firebaseReady = true;
    } catch (e, st) {
      debugPrint('[boot] Firebase init failed, continuing degraded: $e');
      debugPrintStack(stackTrace: st);
    }

    if (firebaseReady) {
      // Route Flutter framework errors into Crashlytics. Debug builds keep the
      // red error box; release builds report instead of silently dying.
      FlutterError.onError = (details) {
        FlutterError.presentError(details);
        FirebaseCrashlytics.instance.recordFlutterFatalError(details);
      };
      // Errors from the engine that never reach a Dart zone.
      PlatformDispatcher.instance.onError = (error, stack) {
        FirebaseCrashlytics.instance.recordError(error, stack, fatal: true);
        return true;
      };
      await FirebaseCrashlytics.instance
          .setCrashlyticsCollectionEnabled(!kDebugMode);
    }

    // A build-method exception should not show testers a raw red/grey box.
    ErrorWidget.builder = (details) {
      if (kDebugMode) return ErrorWidget(details.exception);
      return const _FriendlyErrorBox();
    };

    // SharedPreferences is injected as a ProviderScope override, so it genuinely
    // must resolve before runApp. It is cheap (~10-60ms).
    final prefs = await SharedPreferences.getInstance();

    // NOT awaited: nothing downstream depends on these, and awaiting them just
    // adds platform-channel round trips before first paint.
    unawaited(SystemChrome.setPreferredOrientations([DeviceOrientation.portraitUp]));
    SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    SystemChrome.setSystemUIOverlayStyle(const SystemUiOverlayStyle(
      statusBarColor:                    Colors.transparent,
      statusBarIconBrightness:           Brightness.light,
      systemNavigationBarColor:          Colors.transparent,
      systemNavigationBarContrastEnforced: false,
      systemNavigationBarIconBrightness: Brightness.light,
    ));

    runApp(
      ProviderScope(
        overrides: [
          cacheServiceProvider.overrideWithValue(CacheService(prefs)),
        ],
        child: const LoyaltyNexusApp(),
      ),
    );
  }, (error, stack) {
    debugPrint('[zone] uncaught: $error');
    if (firebaseReady) {
      FirebaseCrashlytics.instance.recordError(error, stack, fatal: true);
    }
  });
}

/// Shown instead of Flutter's red box when a widget fails to build in release.
class _FriendlyErrorBox extends StatelessWidget {
  const _FriendlyErrorBox();

  @override
  Widget build(BuildContext context) {
    return Container(
      color: const Color(0xFF0A0A0F),
      alignment: Alignment.center,
      padding: const EdgeInsets.all(24),
      child: const Text(
        'Something went wrong here.\nPlease go back and try again.',
        textAlign: TextAlign.center,
        style: TextStyle(color: Colors.white70, fontSize: 14),
      ),
    );
  }
}

// ─── App root ─────────────────────────────────────────────────────────────────

class LoyaltyNexusApp extends ConsumerStatefulWidget {
  const LoyaltyNexusApp({super.key});
  @override ConsumerState<LoyaltyNexusApp> createState() => _LoyaltyNexusAppState();
}

class _LoyaltyNexusAppState extends ConsumerState<LoyaltyNexusApp> {
  PushNotificationService? _push;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _initPush());
  }

  void _initPush() {
    // No Firebase means no messaging — attempting init would throw.
    if (!firebaseReady) return;
    try {
      final router    = ref.read(appRouterProvider);
      final container = ProviderScope.containerOf(context);
      _push = PushNotificationService(container: container, router: router);
      _push!.init();
    } catch (e, st) {
      debugPrint('[push] init failed: $e');
      if (firebaseReady) {
        FirebaseCrashlytics.instance.recordError(e, st, fatal: false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final router = ref.watch(appRouterProvider);
    return MaterialApp.router(
      title:                    'Loyalty Nexus',
      debugShowCheckedModeBanner: false,
      theme:                    NexusTheme.dark(),
      routerConfig:             router,
    );
  }
}
