import 'package:firebase_core/firebase_core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'src/core/theme/nexus_theme.dart';
import 'src/core/router/app_router.dart';
import 'src/core/notifications/push_notification_service.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Firebase must be initialised before FCM background handler runs
  await Firebase.initializeApp();

  // Lock orientation to portrait
  await SystemChrome.setPreferredOrientations([DeviceOrientation.portraitUp]);

  // Dark status bar
  SystemChrome.setSystemUIOverlayStyle(const SystemUiOverlayStyle(
    statusBarColor:             Colors.transparent,
    statusBarIconBrightness:    Brightness.light,
    systemNavigationBarColor:   Color(0xFF0D0F1E),
    systemNavigationBarIconBrightness: Brightness.light,
  ));

  runApp(const ProviderScope(child: LoyaltyNexusApp()));
}

// ─── App root ─────────────────────────────────────────────────────────────────

class LoyaltyNexusApp extends ConsumerStatefulWidget {
  const LoyaltyNexusApp({super.key});
  @override ConsumerState<LoyaltyNexusApp> createState() => _LoyaltyNexusAppState();
}

class _LoyaltyNexusAppState extends ConsumerState<LoyaltyNexusApp> {
  PushNotificationService? _pushService;

  @override
  void initState() {
    super.initState();
    // Initialise push notifications on the first frame
    // (router must be available first)
    WidgetsBinding.instance.addPostFrameCallback((_) => _initPush());
  }

  void _initPush() {
    final router    = ref.read(appRouterProvider);
    final container = ProviderScope.containerOf(context);

    _pushService = PushNotificationService(
      container: container,
      router:    router,
    );
    _pushService!.init();
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
