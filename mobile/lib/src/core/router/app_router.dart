import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/presentation/login_screen.dart';
import '../../features/auth/presentation/register_screen.dart';
import '../../features/dashboard/presentation/dashboard_screen.dart';
import '../../features/spin/presentation/spin_screen.dart';
import '../../features/studio/presentation/studio_screen.dart';
import '../../features/wars/presentation/wars_screen.dart';
import '../../features/arcade/presentation/arcade_screen.dart';
import '../../features/profile/presentation/profile_screen.dart';
import '../../features/passport/presentation/passport_screen.dart';
import '../../features/notifications/presentation/notifications_screen.dart';
import '../../features/prizes/presentation/prizes_screen.dart';
import '../../features/prizes/presentation/draws_screen.dart';
import '../../features/prizes/presentation/pulse_awards_screen.dart';
import '../../features/settings/presentation/settings_screen.dart';
import '../../features/how_it_works/presentation/how_it_works_screen.dart';
import '../../features/recharge/presentation/recharge_screen.dart';
import '../../features/recharge/presentation/recharge_success_screen.dart';
import '../auth/auth_provider.dart';
import '../shell/main_shell.dart';

// ── Route constants ───────────────────────────────────────────────────────────
class AppRoutes {
  static const home          = '/';
  static const login         = '/login';
  static const register      = '/register';
  static const dashboard     = '/dashboard';
  static const spin          = '/spin';
  static const studio        = '/studio';
  static const wars          = '/wars';
  static const arcade        = '/arcade';
  static const profile       = '/profile';
  static const passport      = '/passport';
  static const prizes        = '/prizes';
  static const draws         = '/draws';
  static const pulseAwards   = '/pulse-awards';
  static const notifications = '/notifications';
  static const settings      = '/settings';
  static const howItWorks    = '/how-it-works';
  static const recharge       = '/recharge';
  static const rechargeSuccess = '/recharge/success';
}

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

      // Not logged in — allow / and public recharge routes
      if (!loggedIn) {
        if (loc == '/' ||
            loc == '/recharge' ||
            loc == '/recharge/success') return null;
        return '/';
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
          // 0 — Home / Dashboard
          StatefulShellBranch(routes: [
            GoRoute(path: '/dashboard', builder: (_, __) => const DashboardScreen()),
          ]),
          // 1 — Earn (Wars + Arcade)
          StatefulShellBranch(routes: [
            GoRoute(path: '/wars', builder: (_, __) => const WarsScreen()),
            GoRoute(path: '/arcade', builder: (_, __) => const ArcadeScreen()),
          ]),
          // 2 — AI Studio
          StatefulShellBranch(routes: [
            GoRoute(path: '/studio', builder: (_, __) => const StudioScreen()),
          ]),
          // 3 — Rewards (Spin + Prizes)
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/spin',
              builder: (_, __) => const SpinScreen(),
              routes: [
                GoRoute(path: 'prizes', builder: (_, __) => const PrizesScreen()),
              ],
            ),
          ]),
          // 4 — Profile
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
      GoRoute(path: '/how-it-works',  builder: (_, __) => const HowItWorksScreen()),
      // ── Public recharge ───────────────────────────────────────────────────
      GoRoute(path: '/recharge',         builder: (_, __) => const RechargeScreen()),
      GoRoute(
        path: '/recharge/success',
        builder: (_, state) {
          final ref = state.extra as String?;
          return RechargeSuccessScreen(reference: ref);
        },
      ),
    ],
  );
});

// ─── Listenable for redirect refresh ─────────────────────────────────────────

class _AuthListenable extends ChangeNotifier {
  _AuthListenable(Ref ref) {
    ref.listen(authStateProvider, (_, __) => notifyListeners());
  }
}
