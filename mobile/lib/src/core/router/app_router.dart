import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
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
import '../../features/transactions/presentation/transactions_screen.dart';
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
  static const rechargeSuccess  = '/recharge/success';
  static const transactions    = '/transactions';
}

final appRouterProvider = Provider<GoRouter>((ref) {
  // Deliberately NOT ref.watch(authStateProvider): watching would return a
  // brand-new GoRouter on every auth change, discarding the navigation stack and
  // all StatefulShellRoute branch state. refreshListenable below already re-runs
  // `redirect` on auth changes, and `redirect` reads live state via ref.read.

  return GoRouter(
    initialLocation: '/',
    debugLogDiagnostics: false,
    refreshListenable: _AuthListenable(ref),

    // Without this an unknown deep link or a stale push payload drops the
    // tester on go_router's default exception page.
    errorBuilder: (ctx, state) => _RouteNotFound(location: state.uri.toString()),

    redirect: (ctx, state) {
      final loc       = state.matchedLocation;
      final live      = ref.read(authStateProvider);
      final loading   = live.isLoading;
      final loggedIn  = live.isAuthenticated;
      final isNew     = live.isNewUser;

      // Wait for auth init
      if (loading) return null;

      // Not logged in — allow / and public recharge routes
      if (!loggedIn) {
        if (loc == '/' ||
            loc == '/recharge' ||
            loc == '/recharge/success') {
          return null;
        }
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
          // 3 — Spin
          StatefulShellBranch(routes: [
            GoRoute(path: '/spin', builder: (_, __) => const SpinScreen()),
          ]),
          // 4 — Prizes
          StatefulShellBranch(routes: [
            GoRoute(path: '/prizes', builder: (_, __) => const PrizesScreen()),
          ]),
          // 5 — Profile
          StatefulShellBranch(routes: [
            GoRoute(path: '/profile', builder: (_, __) => const ProfileScreen()),
          ]),
        ],
      ),

      // ── Global push routes ────────────────────────────────────────────────
      GoRoute(path: '/passport',      builder: (_, __) => const PassportScreen()),
      GoRoute(path: '/draws',         builder: (_, __) => const DrawsScreen()),
      GoRoute(path: '/pulse-awards',  builder: (_, __) => const PulseAwardsScreen()),
      GoRoute(path: '/notifications', builder: (_, __) => const NotificationsScreen()),
      GoRoute(path: '/settings',      builder: (_, __) => const SettingsScreen()),
      GoRoute(path: '/how-it-works',   builder: (_, __) => const HowItWorksScreen()),
      GoRoute(path: '/transactions',   builder: (_, __) => const TransactionsScreen()),
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

/// Shown when a deep link or push payload points at a route that does not exist.
/// Replaces go_router's raw exception page, which is not something a tester
/// should ever see.
class _RouteNotFound extends StatelessWidget {
  const _RouteNotFound({required this.location});
  final String location;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0A0A0F),
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(28),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.explore_off_outlined, size: 44, color: Colors.white38),
                const SizedBox(height: 16),
                const Text('We could not open that link',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: Colors.white, fontSize: 17, fontWeight: FontWeight.w700)),
                const SizedBox(height: 8),
                Text(location,
                    textAlign: TextAlign.center,
                    style: const TextStyle(color: Colors.white30, fontSize: 11)),
                const SizedBox(height: 20),
                FilledButton(
                  onPressed: () => context.go('/dashboard'),
                  child: const Text('Go to Home'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
