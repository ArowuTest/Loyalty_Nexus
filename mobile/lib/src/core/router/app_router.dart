import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../features/auth/presentation/login_screen.dart';
import '../../features/dashboard/presentation/dashboard_screen.dart';
import '../../features/spin/presentation/spin_screen.dart';
import '../../features/studio/presentation/studio_screen.dart';
import '../../features/wars/presentation/wars_screen.dart';
import '../../features/prizes/presentation/prizes_screen.dart';
import '../../features/settings/presentation/settings_screen.dart';
import '../../features/notifications/presentation/notifications_screen.dart';
import '../auth/auth_provider.dart';
import '../shell/main_shell.dart';

final appRouterProvider = Provider<GoRouter>((ref) {
  final auth = ref.watch(authStateProvider);
  return GoRouter(
    initialLocation: '/',
    redirect: (ctx, state) {
      final ok = auth.token != null;
      if (!ok && state.matchedLocation != '/') return '/';
      if (ok && state.matchedLocation == '/') return '/dashboard';
      return null;
    },
    routes: [
      GoRoute(path: '/', builder: (_, __) => const LoginScreen()),
      ShellRoute(
        builder: (ctx, state, child) => MainShell(child: child),
        routes: [
          GoRoute(path: '/dashboard',      builder: (_, __) => const DashboardScreen()),
          GoRoute(path: '/spin',           builder: (_, __) => const SpinScreen()),
          GoRoute(path: '/studio',         builder: (_, __) => const StudioScreen()),
          GoRoute(path: '/wars',           builder: (_, __) => const WarsScreen()),
          GoRoute(path: '/prizes',         builder: (_, __) => const PrizesScreen()),
          GoRoute(path: '/settings',       builder: (_, __) => const SettingsScreen()),
          GoRoute(path: '/notifications',  builder: (_, __) => const NotificationsScreen()),
        ],
      ),
    ],
  );
});
