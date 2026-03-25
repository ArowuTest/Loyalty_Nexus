import 'package:firebase_core/firebase_core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'src/core/theme/nexus_theme.dart';
import 'src/core/router/app_router.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Firebase initialisation — options loaded from google-services.json / GoogleService-Info.plist
  await Firebase.initializeApp();

  await SystemChrome.setPreferredOrientations([DeviceOrientation.portraitUp]);
  SystemChrome.setSystemUIOverlayStyle(const SystemUiOverlayStyle(
    statusBarColor: Colors.transparent,
    statusBarIconBrightness: Brightness.light,
  ));
  runApp(const ProviderScope(child: LoyaltyNexusApp()));
}

class LoyaltyNexusApp extends ConsumerStatefulWidget {
  const LoyaltyNexusApp({super.key});
  @override ConsumerState<LoyaltyNexusApp> createState() => _LoyaltyNexusAppState();
}

class _LoyaltyNexusAppState extends ConsumerState<LoyaltyNexusApp> {
  @override
  void initState() {
    super.initState();
    // Push notifications initialised after first build so ref is available
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      // Only initialise when user is authenticated (token will be skipped if not)
      try {
        from(context, ref);
      } catch (_) {}
    });
  }

  // Trigger FCM setup lazily — the service ignores the call if no auth token exists
  static void from(BuildContext ctx, WidgetRef ref) {
    // Push init is handled inside auth flow after login
    // See auth_provider.dart → on login success
  }

  @override
  Widget build(BuildContext context) {
    final router = ref.watch(appRouterProvider);
    return MaterialApp.router(
      title: 'Loyalty Nexus',
      debugShowCheckedModeBanner: false,
      theme: NexusTheme.dark(),
      routerConfig: router,
    );
  }
}
