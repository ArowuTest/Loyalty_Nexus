import 'dart:io';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:gap/gap.dart';
import 'package:go_router/go_router.dart';
import '../theme/nexus_theme.dart';
import '../api/api_client.dart';

// ── Connectivity provider ───────────────────────────────────────────────────────────

final _connectivityProvider = StreamProvider.autoDispose<bool>((ref) async* {
  yield* Connectivity().onConnectivityChanged.map(
    (results) => !results.contains(ConnectivityResult.none));
});

// ─── Unread count provider ────────────────────────────────────────────────────

final _unreadCountProvider = FutureProvider.autoDispose<int>((ref) async {
  try {
    final r = await ref.read(notificationsApiProvider).list(limit: 1);
    return (r['unread_count'] as int?) ?? 0;
  } catch (_) { return 0; }
});

// ─── Shell ────────────────────────────────────────────────────────────────────

class MainShell extends ConsumerWidget {
  final StatefulNavigationShell navigationShell;
  const MainShell({super.key, required this.navigationShell});

  static const _tabs = [
    _Tab(icon: Icons.home_outlined,       activeIcon: Icons.home_rounded,         label: 'Home'),
    _Tab(icon: Icons.casino_outlined,      activeIcon: Icons.casino_rounded,       label: 'Spin'),
    _Tab(icon: Icons.auto_awesome_outlined,activeIcon: Icons.auto_awesome,         label: 'Studio'),
    _Tab(icon: Icons.public_outlined,      activeIcon: Icons.public_rounded,       label: 'Wars'),
    _Tab(icon: Icons.sports_esports_outlined, activeIcon: Icons.sports_esports_rounded, label: 'Arcade'),
    _Tab(icon: Icons.person_outline,       activeIcon: Icons.person_rounded,       label: 'Profile'),
  ];

  void _onTap(int i) {
    HapticFeedback.selectionClick();
    navigationShell.goBranch(i,
        initialLocation: i == navigationShell.currentIndex);
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final current    = navigationShell.currentIndex;
    final unreadAsync = ref.watch(_unreadCountProvider);
    final unread     = unreadAsync.valueOrNull ?? 0;
    final isOnline   = ref.watch(_connectivityProvider).valueOrNull ?? true;

    return Scaffold(
      body: Column(children: [
        // Offline banner — slides in from top when connectivity lost
        AnimatedSwitcher(
          duration: const Duration(milliseconds: 350),
          child: isOnline ? const SizedBox.shrink() : _OfflineBanner(),
        ),
        Expanded(child: navigationShell),
      ]),
      extendBody: true,
      bottomNavigationBar: _BottomBar(
        tabs:    _tabs,
        current: current,
        unread:  unread,
        onTap:   _onTap,
      ),
    );
  }
}

// ─── Offline banner ───────────────────────────────────────────────────────────

class _OfflineBanner extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Container(
    width: double.infinity,
    color: const Color(0xFF7f1d1d),
    padding: const EdgeInsets.symmetric(vertical: 6),
    child: const Row(mainAxisAlignment: MainAxisAlignment.center, children: [
      Icon(Icons.wifi_off_rounded, size: 14, color: Colors.white),
      Gap(8),
      Text('No internet connection',
          style: TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.w600)),
    ]),
  ).animate().slideY(begin: -1, end: 0, duration: 300.ms, curve: Curves.easeOut);
}

// ─── Bottom bar ───────────────────────────────────────────────────────────────

class _BottomBar extends StatelessWidget {
  final List<_Tab> tabs;
  final int current, unread;
  final ValueChanged<int> onTap;
  const _BottomBar({
    required this.tabs, required this.current,
    required this.unread, required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: NexusColors.surface,
        border: const Border(top: BorderSide(color: NexusColors.border)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.4),
            blurRadius: 24, offset: const Offset(0, -4),
          ),
        ],
      ),
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 6),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceAround,
            children: List.generate(tabs.length, (i) => _TabItem(
              tab:     tabs[i],
              active:  i == current,
              // Show badge on Profile tab (index 5) — notifications
              badge:   i == 5 ? unread : 0,
              // Arcade tab (index 4) gets a special "NEW" badge
              isNew:   i == 4,
              onTap:   () => onTap(i),
            )),
          ),
        ),
      ),
    );
  }
}

class _TabItem extends StatefulWidget {
  final _Tab tab;
  final bool active;
  final int badge;
  final bool isNew;
  final VoidCallback onTap;
  const _TabItem({
    required this.tab,
    required this.active,
    required this.badge,
    required this.onTap,
    this.isNew = false,
  });
  @override State<_TabItem> createState() => _TabItemState();
}

class _TabItemState extends State<_TabItem> with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late final Animation<double>   _scale;

  @override
  void initState() {
    super.initState();
    _ctrl  = AnimationController(vsync: this, duration: const Duration(milliseconds: 180));
    _scale = CurvedAnimation(parent: _ctrl, curve: Curves.easeOut);
    if (widget.active) _ctrl.forward();
  }

  @override
  void didUpdateWidget(_TabItem old) {
    super.didUpdateWidget(old);
    if (widget.active != old.active) {
      widget.active ? _ctrl.forward() : _ctrl.reverse();
    }
  }

  @override
  void dispose() { _ctrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: GestureDetector(
        onTap: widget.onTap,
        behavior: HitTestBehavior.opaque,
        child: Column(mainAxisSize: MainAxisSize.min, children: [

          // ── Pill indicator + icon ──
          AnimatedBuilder(
            animation: _scale,
            builder: (_, child) => AnimatedContainer(
              duration: const Duration(milliseconds: 200),
              curve: Curves.easeOut,
              width: widget.active ? 52 : 40,
              height: 32,
              decoration: widget.active
                  ? BoxDecoration(
                      color: NexusColors.primaryGlow,
                      borderRadius: NexusRadius.pill,
                    )
                  : null,
              child: Stack(alignment: Alignment.center, children: [
                Icon(
                  widget.active ? widget.tab.activeIcon : widget.tab.icon,
                  size: 20,
                  color: widget.active
                      ? NexusColors.primary
                      : NexusColors.textSecondary,
                ),
                // Notification badge
                if (widget.badge > 0)
                  Positioned(
                    top: 3, right: widget.active ? 5 : 3,
                    child: Container(
                      width: 15, height: 15,
                      decoration: const BoxDecoration(
                        color: NexusColors.red, shape: BoxShape.circle),
                      child: Center(child: Text(
                        widget.badge > 9 ? '9+' : '${widget.badge}',
                        style: const TextStyle(color: Colors.white, fontSize: 8,
                            fontWeight: FontWeight.w800),
                      )),
                    ),
                  ),
                // "NEW" badge for Arcade tab
                if (widget.isNew && !widget.active)
                  Positioned(
                    top: 0, right: 0,
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 3, vertical: 1),
                      decoration: BoxDecoration(
                        gradient: const LinearGradient(
                          colors: [NexusColors.purple, NexusColors.primary],
                        ),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: const Text(
                        'NEW',
                        style: TextStyle(
                          color: Colors.white,
                          fontSize: 6,
                          fontWeight: FontWeight.w900,
                          letterSpacing: 0.3,
                        ),
                      ),
                    ),
                  ),
              ]),
            ),
          ),

          const SizedBox(height: 2),

          // ── Label ──
          AnimatedDefaultTextStyle(
            duration: const Duration(milliseconds: 200),
            style: TextStyle(
              fontSize: 9,
              fontWeight: widget.active ? FontWeight.w700 : FontWeight.w500,
              color: widget.active
                  ? NexusColors.primary
                  : NexusColors.textSecondary,
            ),
            child: Text(widget.tab.label),
          ),
        ]),
      ),
    );
  }
}

class _Tab {
  final IconData icon, activeIcon;
  final String label;
  const _Tab({required this.icon, required this.activeIcon, required this.label});
}
