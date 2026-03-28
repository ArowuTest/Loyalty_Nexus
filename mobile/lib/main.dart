import 'package:firebase_core/firebase_core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'src/core/theme/nexus_theme.dart';
import 'src/core/router/app_router.dart';
import 'src/core/cache/cache_service.dart';
import 'src/core/notifications/push_notification_service.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Firebase init — must precede FCM background handler registration
  await Firebase.initializeApp();

  // SharedPreferences — initialised once, injected as provider override
  final prefs = await SharedPreferences.getInstance();

  // Lock to portrait only (landscape adds no value for this app)
  await SystemChrome.setPreferredOrientations([DeviceOrientation.portraitUp]);

  // Edge-to-edge: content draws behind status bar & nav bar
  SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);

  // Transparent system bars — theme handles colouring via NavigationBarTheme
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
        // Inject SharedPreferences-backed cache service
        cacheServiceProvider.overrideWithValue(CacheService(prefs)),
      ],
      child: const LoyaltyNexusApp(),
    ),
  );
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
    final router    = ref.read(appRouterProvider);
    final container = ProviderScope.containerOf(context);
    _push = PushNotificationService(container: container, router: router);
    _push!.init();
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
