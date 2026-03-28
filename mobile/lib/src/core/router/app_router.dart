import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

// ── Route constants \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\nclass AppRoutes {\n  static const home          = '/';\n  static const login         = '/login';\n  static const register      = '/register';\n  static const spin          = '/spin';\n  static const studio        = '/studio';\n  static const wars          = '/wars';\n  static const profile       = '/profile';\n  static const passport      = '/passport';\n  static const prizes        = '/prizes';\n  static const draws         = '/draws';\n  static const pulseAwards   = '/pulse-awards';\n  static const notifications = '/notifications';\n  static const settings      = '/settings';\n}
import '../../features/auth/presentation/login_screen.dart';
import '../../features/auth/presentation/register_screen.dart';
import '../../features/dashboard/presentation/dashboard_screen.dart';
import '../../features/spin/presentation/spin_screen.dart';
import '../../features/studio/presentation/studio_screen.dart';
import '../../features/wars/presentation/wars_screen.dart';
import '../../features/profile/presentation/profile_screen.dart';
import '../../features/profile/presentation/passport_screen.dart';
import '../../features/notifications/presentation/notifications_screen.dart';
import '../../features/prizes/presentation/prizes_screen.dart';
import '../../features/prizes/presentation/draws_screen.dart';
import '../../features/prizes/presentation/pulse_awards_screen.dart';
import '../../features/settings/presentation/settings_screen.dart';
import '../auth/auth_provider.dart';
import '../shell/main_shell.dart';

final appRouterProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authStateProvider);

  return GoRouter(
    initialLocation: '/',
    debugLogDiagnostics: false,
    refreshListenable: _AuthListenable(ref),

    redirect: (ctx, state) {
      final loc       = state.matchedLocation;
      final loading   = authState.isLoading;
      final loggedIn  = authState.isAuthenticated;
      final isNew     = authState.isNewUser;

      // Wait for auth init
      if (loading) return null;

      // Not logged in — only allow /
      if (!loggedIn) {
        return loc == '/' ? null : '/';
      }

      // New user — only allow /register
      if (isNew && loc != '/register') return '/register';

      // Already logged in — skip login screen
      if (loc == '/') return '/dashboard';

      return null;
    },

    routes: [
      // ── Auth ─────────────────────────────────────────────────────────────
      GoRoute(path: '/', builder: (_, __) => const LoginScreen()),
      GoRoute(path: '/register', builder: (_, __) => const RegisterScreen()),

      // ── Shell (bottom-nav) ────────────────────────────────────────────────
      StatefulShellRoute.indexedStack(
        builder: (_, __, shell) => MainShell(navigationShell: shell),
        branches: [
          StatefulShellBranch(routes: [
            GoRoute(path: '/dashboard', builder: (_, __) => const DashboardScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/spin',
              builder: (_, __) => const SpinScreen(),
              routes: [
                GoRoute(path: 'prizes', builder: (_, __) => const PrizesScreen()),
              ],
            ),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/studio', builder: (_, __) => const StudioScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/wars', builder: (_, __) => const WarsScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/profile', builder: (_, __) => const ProfileScreen()),
          ]),
        ],
      ),

      // ── Global push routes ────────────────────────────────────────────────
      GoRoute(path: '/passport',      builder: (_, __) => const PassportScreen()),
      GoRoute(path: '/prizes',        builder: (_, __) => const PrizesScreen()),
      GoRoute(path: '/draws',         builder: (_, __) => const DrawsScreen()),
      GoRoute(path: '/pulse-awards',  builder: (_, __) => const PulseAwardsScreen()),
      GoRoute(path: '/notifications', builder: (_, __) => const NotificationsScreen()),
      GoRoute(path: '/settings',      builder: (_, __) => const SettingsScreen()),
    ],
  );
});

// ─── Listenable for redirect refresh ─────────────────────────────────────────

class _AuthListenable extends ChangeNotifier {
  _AuthListenable(ProviderRef ref) {
    ref.listen(authStateProvider, (_, __) => notifyListeners());
  }
}
