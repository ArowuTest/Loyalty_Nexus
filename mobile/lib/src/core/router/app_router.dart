import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../features/auth/presentation/login_screen.dart';
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
  final auth = ref.watch(authStateProvider);

  return GoRouter(
    initialLocation: '/',
    debugLogDiagnostics: false,

    // ── Auth guard ────────────────────────────────────────────────────────────
    redirect: (ctx, state) {
      final loggedIn = auth.token != null;
      final loc      = state.matchedLocation;
      // Public routes
      if (!loggedIn && loc != '/') return '/';
      // Already logged in → skip login screen
      if (loggedIn  && loc == '/') return '/dashboard';
      return null;
    },

    routes: [
      // ── Login / OTP ───────────────────────────────────────────────────────
      GoRoute(
        path: '/',
        builder: (_, __) => const LoginScreen(),
      ),

      // ── Shell (bottom-nav tabs) ────────────────────────────────────────────
      StatefulShellRoute.indexedStack(
        builder: (ctx, state, shell) => MainShell(navigationShell: shell),
        branches: [
          // Tab 0 — Dashboard
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/dashboard',
              builder: (_, __) => const DashboardScreen(),
            ),
          ]),

          // Tab 1 — Spin
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/spin',
              builder: (_, __) => const SpinScreen(),
              routes: [
                // /spin/prizes  — My Wins (accessible within spin tab)
                GoRoute(
                  path: 'prizes',
                  builder: (_, __) => const PrizesScreen(),
                ),
              ],
            ),
          ]),

          // Tab 2 — Studio
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/studio',
              builder: (_, __) => const StudioScreen(),
            ),
          ]),

          // Tab 3 — Wars
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/wars',
              builder: (_, __) => const WarsScreen(),
            ),
          ]),

          // Tab 4 — Profile
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/profile',
              builder: (_, __) => const ProfileScreen(),
            ),
          ]),
        ],
      ),

      // ── Global routes (accessible from any tab via push / deep-link) ───────

      GoRoute(
        path: '/passport',
        builder: (_, __) => const PassportScreen(),
      ),

      GoRoute(
        path: '/prizes',
        builder: (_, __) => const PrizesScreen(),
      ),

      GoRoute(
        path: '/draws',
        builder: (_, __) => const DrawsScreen(),
      ),

      GoRoute(
        path: '/pulse-awards',
        builder: (_, __) => const PulseAwardsScreen(),
      ),

      GoRoute(
        path: '/notifications',
        builder: (_, __) => const NotificationsScreen(),
      ),

      GoRoute(
        path: '/settings',
        builder: (_, __) => const SettingsScreen(),
      ),
    ],
  );
});
