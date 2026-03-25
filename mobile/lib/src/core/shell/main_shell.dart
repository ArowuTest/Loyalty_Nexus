import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../theme/nexus_theme.dart';

class MainShell extends StatelessWidget {
  final Widget child;
  const MainShell({super.key, required this.child});

  static const _tabs = [
    (path: '/dashboard', icon: Icons.home_rounded,      label: 'Home'),
    (path: '/spin',      icon: Icons.casino_rounded,    label: 'Spin'),
    (path: '/studio',    icon: Icons.auto_awesome,      label: 'Studio'),
    (path: '/wars',      icon: Icons.public_rounded,    label: 'Wars'),
    (path: '/prizes',    icon: Icons.card_giftcard,     label: 'Prizes'),
  ];

  int _idx(BuildContext ctx) {
    final path = GoRouterState.of(ctx).matchedLocation;
    final i = _tabs.indexWhere((t) => t.path == path);
    return i < 0 ? 0 : i;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: Container(
        decoration: const BoxDecoration(
          color: NexusColors.surface,
          border: Border(top: BorderSide(color: NexusColors.border))),
        child: NavigationBar(
          backgroundColor: NexusColors.surface,
          indicatorColor: const Color(0x265F72F9),
          selectedIndex: _idx(context),
          onDestinationSelected: (i) => context.go(_tabs[i].path),
          destinations: [
            ..._tabs.map((t) =>
                NavigationDestination(icon: Icon(t.icon), label: t.label)),
          ],
        ),
      ),
      floatingActionButton: GoRouterState.of(context).matchedLocation != '/notifications'
          ? _NotifFab()
          : null,
    );
  }
}

/// Floating notification bell shown on all main screens
class _NotifFab extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 44,
      height: 44,
      child: FloatingActionButton(
        mini: true,
        backgroundColor: NexusColors.surface,
        elevation: 2,
        onPressed: () => context.push('/notifications'),
        child: const Icon(Icons.notifications_outlined,
            color: NexusColors.textPrimary, size: 22),
      ),
    );
  }
}
