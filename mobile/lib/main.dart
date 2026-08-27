import 'dart:async';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_analytics/firebase_analytics.dart';
import 'package:firebase_crashlytics/firebase_crashlytics.dart';
import 'package:firebase_remote_config/firebase_remote_config.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'src/core/theme/nexus_theme.dart';
import 'src/core/router/app_router.dart';
import 'src/core/analytics/analytics.dart';
import 'src/core/cache/cache_service.dart';
import 'src/core/remote_config/app_gate_screen.dart';
import 'src/core/remote_config/remote_config_service.dart';
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

      // firebase_analytics was a dependency with zero imports — sessions were
      // auto-collected but no screens and no events. Attach it so the funnel is
      // visible during the tester round, not just crashes.
      Analytics.instance.attach(FirebaseAnalytics.instance);

      // Remote kill switch / force-update. Awaited so a already-cached "block"
      // decision is known BEFORE first paint — otherwise a blocked user would
      // briefly see the real app. It cannot stall startup: init() has an 8s
      // fetch timeout, activates cached values first, and fails open.
      await RemoteConfigService.instance.init(FirebaseRemoteConfig.instance);
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
      // Publish it so screens can request notification permission at a sensible
      // moment (after login) rather than on the first frame.
      ref.read(pushNotificationServiceProvider.notifier).state = _push;
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
      // Wraps EVERY route. A blocking gate must not be a dialog (dismissible) or
      // a route (deep-linkable past) — it has to sit above the router entirely.
      builder: (context, child) => _RemoteGate(child: child ?? const SizedBox.shrink()),
    );
  }
}


/// Enforces the remote kill switch / force-update above the whole router.
class _RemoteGate extends StatefulWidget {
  const _RemoteGate({required this.child});
  final Widget child;

  @override
  State<_RemoteGate> createState() => _RemoteGateState();
}

class _RemoteGateState extends State<_RemoteGate> with WidgetsBindingObserver {
  bool _bannerDismissed = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // Re-evaluate on resume so a kill switch flipped while the app was
    // backgrounded takes effect without the user relaunching.
    if (state == AppLifecycleState.resumed) {
      RemoteConfigService.instance.refresh().then((_) {
        if (mounted) setState(() {}); // build() re-evaluates the gate
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    // Re-read on every build so the "Check again" button can clear the screen.
    final gate = RemoteConfigService.instance.evaluate();
    // onRechecked lets the blocking screen's "Check again" button rebuild THIS
    // widget — otherwise a cleared gate would leave the user stuck on the block
    // screen until they relaunched.
    if (gate.blocks) {
      return AppGateScreen(
        gate: gate,
        onRechecked: () { if (mounted) setState(() {}); },
      );
    }

    if (gate.status == AppGateStatus.updateAvailable && !_bannerDismissed) {
      return Column(
        children: [
          UpdateAvailableBanner(
            gate: gate,
            onDismiss: () => setState(() => _bannerDismissed = true),
          ),
          Expanded(child: widget.child),
        ],
      );
    }
    return widget.child;
  }
}
