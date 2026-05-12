import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:gap/gap.dart';
import 'package:go_router/go_router.dart';
import '../theme/nexus_theme.dart';
import '../widgets/nexus_gamification.dart';
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

  // 5 tabs: Home | Earn | Studio | Rewards | Profile
  // Recharge is a gold ⚡ button between Studio and Rewards (not a shell branch)
  static const _tabs = [
    _Tab(icon: Icons.home_outlined,          activeIcon: Icons.home_rounded,          label: 'Home'),
    _Tab(icon: Icons.trending_up_outlined,   activeIcon: Icons.trending_up_rounded,   label: 'Earn'),
    _Tab(icon: Icons.auto_awesome_outlined,  activeIcon: Icons.auto_awesome,          label: 'Studio'),
    _Tab(icon: Icons.card_giftcard_outlined, activeIcon: Icons.card_giftcard_rounded, label: 'Rewards'),
    _Tab(icon: Icons.person_outline,         activeIcon: Icons.person_rounded,        label: 'Profile'),
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
    // Split tabs into left (0–2) and right (3–5) groups with a Recharge button in the middle
    final leftTabs  = tabs.sublist(0, 3);  // Home, Earn, Studio
    final rightTabs = tabs.sublist(3, 5);  // Rewards, Profile

    return Container(
      decoration: BoxDecoration(
        color: NexusColors.surface,
        border: const Border(top: BorderSide(color: NexusColors.border)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.4),
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
            children: [
              // Left tabs: Home (0), Spin (1), Studio (2)
              ...List.generate(leftTabs.length, (i) => _TabItem(
                tab:    leftTabs[i],
                active: i == current,
                badge:  0,
                isNew:  false,
                onTap:  () => onTap(i),
              )),

              // ── Centre Recharge button ──
              _RechargeNavButton(),

              // Right tabs: Wars (3), Arcade (4), Profile (5)
              ...List.generate(rightTabs.length, (i) => _TabItem(
                tab:    rightTabs[i],
                active: (i + 3) == current,
                badge:  (i + 3) == 4 ? unread : 0,
                isNew:  (i + 3) == 4,
                onTap:  () => onTap(i + 3),
              )),
            ],
          ),
        ),
      ),
    );
  }
}

// ─── Recharge centre-nav button ───────────────────────────────────────────────

class _RechargeNavButton extends StatelessWidget {
  const _RechargeNavButton();

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () {
        HapticFeedback.lightImpact();
        context.push('/recharge');
      },
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width:  44,
            height: 34,
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                colors: [Color(0xFFF9C74F), Color(0xFFFFB703)],
                begin: Alignment.topLeft,
                end:   Alignment.bottomRight,
              ),
              borderRadius: BorderRadius.circular(12),
              boxShadow: [
                BoxShadow(
                  color:       NexusColors.gold.withValues(alpha: 0.45),
                  blurRadius:  10,
                  spreadRadius: 0,
                  offset:      const Offset(0, 2),
                ),
              ],
            ),
            child: const Icon(
              Icons.bolt_rounded,
              color: Color(0xFF1A1200),
              size:  22,
            ),
          ),
          const SizedBox(height: 3),
          const Text(
            'Recharge',
            style: TextStyle(
              color:      NexusColors.gold,
              fontSize:   10,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
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
