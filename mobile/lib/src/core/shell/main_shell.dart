import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../theme/nexus_theme.dart';

/// Main shell with 5 tabs:
/// Home · Spin · Studio · Wars · Profile
class MainShell extends StatelessWidget {
  final StatefulNavigationShell navigationShell;
  const MainShell({super.key, required this.navigationShell});

  static const _tabs = [
    (icon: Icons.home_rounded,    label: 'Home'),
    (icon: Icons.casino_rounded,  label: 'Spin'),
    (icon: Icons.auto_awesome,    label: 'Studio'),
    (icon: Icons.public_rounded,  label: 'Wars'),
    (icon: Icons.person_rounded,  label: 'Profile'),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: Container(
        decoration: BoxDecoration(
          color: NexusColors.surface,
          border: const Border(top: BorderSide(color: NexusColors.border)),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.3),
              blurRadius: 20, offset: const Offset(0, -4)),
          ],
        ),
        child: SafeArea(
          child: NavigationBar(
            backgroundColor: NexusColors.surface,
            indicatorColor: NexusColors.primary.withOpacity(0.15),
            indicatorShape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            selectedIndex: navigationShell.currentIndex,
            onDestinationSelected: (i) => navigationShell.goBranch(
              i,
              // Re-tap active tab → pop to root of that tab stack
              initialLocation: i == navigationShell.currentIndex,
            ),
            destinations: _tabs.map((t) => NavigationDestination(
              icon: Icon(t.icon),
              selectedIcon: Icon(t.icon, color: NexusColors.primary),
              label: t.label,
            )).toList(),
          ),
        ),
      ),
    );
  }
}
