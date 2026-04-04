import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Providers ────────────────────────────────────────────────────────────────

final _profileProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getProfile();
});

final _notifPrefsProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getNotificationPrefs();
});

// ─── Settings Screen ──────────────────────────────────────────────────────────

class SettingsScreen extends ConsumerStatefulWidget {
  const SettingsScreen({super.key});
  @override ConsumerState<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends ConsumerState<SettingsScreen> {

  // ── MoMo ──
  final _momoCtrl = TextEditingController();
  bool _linkingMomo  = false;
  String? _momoError;
  bool _showMoMo     = false;

  // ── Notification prefs local mirror ──
  final Map<String, bool> _notifToggles = {
    'spin_results':    true,
    'draw_winners':    true,
    'point_updates':   true,
    'war_updates':     true,
    'bonus_awards':    true,
    'system_alerts':   true,
  };
  bool _savingNotif  = false;

  @override
  void dispose() { _momoCtrl.dispose(); super.dispose(); }

  // ─────────────────────────────────────────────────────────────────────────────

  Future<void> _linkMomo() async {
    final num = _momoCtrl.text.trim();
    if (num.length < 11) {
      setState(() => _momoError = 'Enter a valid 11-digit MoMo number');
      return;
    }
    setState(() { _linkingMomo = true; _momoError = null; });
    try {
      await ref.read(userApiProvider).requestMoMoLink(num);
      ref.invalidate(_profileProvider);
      if (mounted) {
        setState(() => _showMoMo = false);
        _showSnack('✅ MoMo number linked! Verification in progress.', NexusColors.green);
      }
    } on ApiException catch (e) {
      setState(() => _momoError = e.message);
    } finally {
      setState(() => _linkingMomo = false);
    }
  }

  Future<void> _saveNotifPrefs() async {
    setState(() => _savingNotif = true);
    try {
      await ref.read(userApiProvider).updateNotificationPrefs(_notifToggles);
      if (mounted) _showSnack('✅ Notification preferences saved', NexusColors.green);
    } on ApiException catch (e) {
      if (mounted) _showSnack('❌ ${e.message}', NexusColors.red);
    } finally {
      if (mounted) setState(() => _savingNotif = false);
    }
  }

  void _showSnack(String msg, Color bg) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg), backgroundColor: bg,
      behavior: SnackBarBehavior.floating,
    ));
  }

  Future<void> _signOut() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        backgroundColor: NexusColors.surface,
        title: const Text('Sign Out', style: TextStyle(color: NexusColors.textPrimary)),
        content: const Text('Are you sure you want to sign out?',
            style: TextStyle(color: NexusColors.textSecondary)),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel')),
          TextButton(onPressed: () => Navigator.pop(context, true),
              child: const Text('Sign Out', style: TextStyle(color: NexusColors.red))),
        ],
      ),
    );
    if (ok == true) {
      await ref.read(authStateProvider.notifier).logout();
      if (!mounted) return;
      context.go('/');
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth     = ref.watch(authStateProvider);
    final profile  = ref.watch(_profileProvider);
    final notifPrefs = ref.watch(_notifPrefsProvider);

    // Sync notification prefs when loaded
    notifPrefs.whenData((prefs) {
      if (mounted) {
        final map = prefs['preferences'] as Map<String, dynamic>? ?? {};
        for (final k in _notifToggles.keys) {
          if (map.containsKey(k)) {
            _notifToggles[k] = map[k] as bool? ?? true;
          }
        }
      }
    });

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.surface,
        title: const Text('Settings'),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [

          // ── Account section ──
          _SectionLabel(label: 'Account'),
          _SettingsCard(children: [
            // Phone number
            _Tile(
              icon: Icons.phone_android_rounded,
              title: 'Phone Number',
              subtitle: auth.phoneNumber ?? '—',
              trailing: null,
            ),
            const _Divider(),

            // Profile info (name / state)
            profile.when(
              loading: () => const _Tile(
                  icon: Icons.person_outline_rounded,
                  title: 'Profile', subtitle: 'Loading…', trailing: null),
              error: (_, __) => _Tile(
                  icon: Icons.person_outline_rounded,
                  title: 'Profile',
                  subtitle: 'Could not load profile',
                  trailing: null),
              data: (p) => _Tile(
                icon: Icons.person_outline_rounded,
                title: p['full_name'] as String? ?? 'Your Profile',
                subtitle: p['state'] as String? ?? 'Update your profile',
                onTap: () => _showEditProfileSheet(p),
                trailing: const Icon(Icons.chevron_right_rounded,
                    color: NexusColors.textSecondary, size: 18),
              ),
            ),
            const _Divider(),

            // MoMo wallet
            _Tile(
              icon: Icons.account_balance_wallet_rounded,
              iconColor: NexusColors.green,
              title: 'Link MoMo Wallet',
              subtitle: 'Required to receive cash prizes',
              onTap: () => setState(() { _showMoMo = true; _momoError = null; }),
              trailing: const Icon(Icons.chevron_right_rounded,
                  color: NexusColors.textSecondary, size: 18),
            ),
          ]),

          // ── Notifications section ──
          _SectionLabel(label: 'Notifications'),
          _SettingsCard(children: [
            ..._notifToggles.entries.map((e) {
              final label = _notifLabel(e.key);
              return Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  _ToggleTile(
                    icon: _notifIcon(e.key),
                    title: label.$1,
                    subtitle: label.$2,
                    value: e.value,
                    onChanged: (v) => setState(() => _notifToggles[e.key] = v),
                  ),
                  if (e.key != _notifToggles.keys.last) const _Divider(),
                ],
              );
            }),
            const Divider(color: NexusColors.border, height: 1),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
              child: ElevatedButton(
                onPressed: _savingNotif ? null : _saveNotifPrefs,
                style: ElevatedButton.styleFrom(
                  backgroundColor: NexusColors.primary,
                  minimumSize: const Size(double.infinity, 44),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                ),
                child: _savingNotif
                    ? const SizedBox(width: 18, height: 18,
                        child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
                    : const Text('Save Preferences',
                        style: TextStyle(fontWeight: FontWeight.w700)),
              ),
            ),
          ]),

          // ── Support section ──
          _SectionLabel(label: 'Support'),
          _SettingsCard(children: [
            _Tile(
              icon: Icons.privacy_tip_outlined,
              title: 'Privacy Policy',
              subtitle: 'How we use your data',
              trailing: const Icon(Icons.open_in_new_rounded,
                  color: NexusColors.textSecondary, size: 16),
              onTap: () => launchUrl(Uri.parse('https://loyaltynexus.ng/privacy'),
                  mode: LaunchMode.externalApplication),
            ),
            const _Divider(),
            _Tile(
              icon: Icons.article_outlined,
              title: 'Terms of Service',
              subtitle: 'Usage terms and conditions',
              trailing: const Icon(Icons.open_in_new_rounded,
                  color: NexusColors.textSecondary, size: 16),
              onTap: () => launchUrl(Uri.parse('https://loyaltynexus.ng/terms'),
                  mode: LaunchMode.externalApplication),
            ),
            const _Divider(),
            _Tile(
              icon: Icons.help_outline_rounded,
              title: 'Help & FAQs',
              subtitle: 'Get support and answers',
              trailing: const Icon(Icons.open_in_new_rounded,
                  color: NexusColors.textSecondary, size: 16),
              onTap: () => launchUrl(Uri.parse('https://loyaltynexus.ng/help'),
                  mode: LaunchMode.externalApplication),
            ),
          ]),

          const SizedBox(height: 16),

          // ── Sign Out ──
          GestureDetector(
            onTap: _signOut,
            child: Container(
              width: double.infinity,
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: NexusColors.surface,
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: NexusColors.red.withValues(alpha: 0.2)),
              ),
              child: Row(children: [
                Container(
                  width: 38, height: 38,
                  decoration: BoxDecoration(
                    color: NexusColors.red.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(11),
                  ),
                  child: const Icon(Icons.logout_rounded, color: NexusColors.red, size: 18),
                ),
                const SizedBox(width: 12),
                const Text('Sign Out', style: TextStyle(
                    color: NexusColors.red, fontSize: 14, fontWeight: FontWeight.w700)),
              ]),
            ),
          ),

          const SizedBox(height: 32),
        ],
      ),

      // ── MoMo bottom sheet ──
      bottomSheet: _showMoMo ? _MoMoSheet(
        ctrl:     _momoCtrl,
        error:    _momoError,
        loading:  _linkingMomo,
        onLink:   _linkMomo,
        onClose:  () => setState(() => _showMoMo = false),
        onChanged: (_) => setState(() => _momoError = null),
      ) : null,
    );
  }

  void _showEditProfileSheet(Map<String, dynamic> profile) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: NexusColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
      builder: (_) => _EditProfileSheet(profile: profile, ref: ref),
    );
  }
}

// ─── MoMo Sheet ───────────────────────────────────────────────────────────────

class _MoMoSheet extends StatelessWidget {
  final TextEditingController ctrl;
  final String? error;
  final bool loading;
  final VoidCallback onLink, onClose;
  final ValueChanged<String> onChanged;
  const _MoMoSheet({
    required this.ctrl, required this.error, required this.loading,
    required this.onLink, required this.onClose, required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      padding: EdgeInsets.only(
        left: 24, right: 24, top: 20,
        bottom: MediaQuery.of(context).viewInsets.bottom + 24,
      ),
      child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          const Text('Link MoMo Wallet',
              style: TextStyle(color: NexusColors.textPrimary, fontSize: 18, fontWeight: FontWeight.w800)),
          const Spacer(),
          GestureDetector(onTap: onClose,
              child: const Icon(Icons.close_rounded, color: NexusColors.textSecondary)),
        ]),
        const SizedBox(height: 6),
        const Text('Enter your MTN MoMo number to receive cash prizes directly.',
            style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
        const SizedBox(height: 20),
        TextField(
          controller: ctrl,
          keyboardType: TextInputType.phone,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly,
              LengthLimitingTextInputFormatter(11)],
          style: const TextStyle(color: NexusColors.textPrimary, fontSize: 16),
          onChanged: onChanged,
          decoration: InputDecoration(
            hintText: '08031234567',
            prefixIcon: const Icon(Icons.phone_android_rounded, color: NexusColors.primary),
            filled: true, fillColor: NexusColors.background,
            border: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
                borderSide: const BorderSide(color: NexusColors.border)),
            enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
                borderSide: const BorderSide(color: NexusColors.border)),
            errorText: error,
          ),
        ),
        const SizedBox(height: 8),
        const Text("Don't have MoMo? Dial *671# on your MTN line.",
            style: TextStyle(color: NexusColors.gold, fontSize: 11)),
        const SizedBox(height: 20),
        Row(children: [
          Expanded(child: OutlinedButton(
            onPressed: onClose,
            style: OutlinedButton.styleFrom(
              foregroundColor: NexusColors.textSecondary,
              side: const BorderSide(color: NexusColors.border),
              padding: const EdgeInsets.symmetric(vertical: 14),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            ),
            child: const Text('Cancel'),
          )),
          const SizedBox(width: 10),
          Expanded(child: ElevatedButton(
            onPressed: loading ? null : onLink,
            style: ElevatedButton.styleFrom(
              backgroundColor: NexusColors.green,
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(vertical: 14),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            ),
            child: loading
                ? const SizedBox(width: 20, height: 20,
                    child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
                : const Text('Link Wallet', style: TextStyle(fontWeight: FontWeight.w800)),
          )),
        ]),
      ]),
    );
  }
}

// ─── Edit Profile Sheet ───────────────────────────────────────────────────────

const _nigerianStates = [
  'Abia','Adamawa','Akwa Ibom','Anambra','Bauchi','Bayelsa','Benue','Borno',
  'Cross River','Delta','Ebonyi','Edo','Ekiti','Enugu','FCT Abuja','Gombe',
  'Imo','Jigawa','Kaduna','Kano','Katsina','Kebbi','Kogi','Kwara','Lagos',
  'Nasarawa','Niger','Ogun','Ondo','Osun','Oyo','Plateau','Rivers','Sokoto',
  'Taraba','Yobe','Zamfara',
];

class _EditProfileSheet extends StatefulWidget {
  final Map<String, dynamic> profile;
  final WidgetRef ref;
  const _EditProfileSheet({required this.profile, required this.ref});
  @override State<_EditProfileSheet> createState() => _EditProfileSheetState();
}

class _EditProfileSheetState extends State<_EditProfileSheet> {
  late final TextEditingController _nameCtrl;
  String? _selectedState;
  bool _saving = false;
  String? _err;

  @override
  void initState() {
    super.initState();
    _nameCtrl = TextEditingController(text: widget.profile['full_name'] as String? ?? '');
    _selectedState = widget.profile['state'] as String?;
  }
  @override void dispose() { _nameCtrl.dispose(); super.dispose(); }

  Future<void> _save() async {
    setState(() { _saving = true; _err = null; });
    try {
      await widget.ref.read(userApiProvider).updateProfile(
        fullName: _nameCtrl.text.trim().isNotEmpty ? _nameCtrl.text.trim() : null,
        state:    _selectedState,
      );
      widget.ref.invalidate(_profileProvider);
      if (mounted) Navigator.pop(context);
    } on ApiException catch (e) {
      setState(() => _err = e.message);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) => Padding(
    padding: EdgeInsets.only(
      left: 24, right: 24, top: 24,
      bottom: MediaQuery.of(context).viewInsets.bottom + 24,
    ),
    child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
      const Text('Edit Profile', style: TextStyle(
          color: NexusColors.textPrimary, fontSize: 18, fontWeight: FontWeight.w800)),
      const SizedBox(height: 20),

      const Text('FULL NAME', style: TextStyle(color: NexusColors.textSecondary,
          fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 0.8)),
      const SizedBox(height: 8),
      TextField(
        controller: _nameCtrl,
        style: const TextStyle(color: NexusColors.textPrimary),
        decoration: InputDecoration(
          hintText: 'Enter your full name',
          hintStyle: const TextStyle(color: NexusColors.textSecondary),
          filled: true, fillColor: NexusColors.background,
          border: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
              borderSide: const BorderSide(color: NexusColors.border)),
          enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
              borderSide: const BorderSide(color: NexusColors.border)),
        ),
      ),
      const SizedBox(height: 16),

      const Text('STATE', style: TextStyle(color: NexusColors.textSecondary,
          fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 0.8)),
      const SizedBox(height: 8),
      DropdownButtonFormField<String>(
        value: _selectedState,
        hint: const Text('Select your state', style: TextStyle(color: NexusColors.textSecondary)),
        onChanged: (v) => setState(() => _selectedState = v),
        dropdownColor: NexusColors.surface,
        style: const TextStyle(color: NexusColors.textPrimary, fontSize: 14),
        items: _nigerianStates.map((s) =>
            DropdownMenuItem(value: s, child: Text(s))).toList(),
        decoration: InputDecoration(
          filled: true, fillColor: NexusColors.background,
          border: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
              borderSide: const BorderSide(color: NexusColors.border)),
          enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
              borderSide: const BorderSide(color: NexusColors.border)),
        ),
      ),

      if (_err != null) ...[
        const SizedBox(height: 12),
        Text(_err!, style: const TextStyle(color: NexusColors.red, fontSize: 12)),
      ],

      const SizedBox(height: 20),
      ElevatedButton(
        onPressed: _saving ? null : _save,
        style: ElevatedButton.styleFrom(
          backgroundColor: NexusColors.primary,
          minimumSize: const Size(double.infinity, 50),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
        ),
        child: _saving
            ? const SizedBox(width: 20, height: 20,
                child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
            : const Text('Save Changes', style: TextStyle(fontWeight: FontWeight.w800)),
      ),
    ]),
  );
}

// ─── Notification helpers ─────────────────────────────────────────────────────

IconData _notifIcon(String key) => switch (key) {
  'spin_results'   => Icons.casino_rounded,
  'draw_winners'   => Icons.emoji_events_rounded,
  'point_updates'  => Icons.bolt_rounded,
  'war_updates'    => Icons.flag_rounded,
  'bonus_awards'   => Icons.card_giftcard_rounded,
  'system_alerts'  => Icons.campaign_rounded,
  _                => Icons.notifications_outlined,
};

(String, String) _notifLabel(String key) => switch (key) {
  'spin_results'  => ('Spin Results',   'Notify when your spin lands'),
  'draw_winners'  => ('Draw Winners',   'Notify when draws are announced'),
  'point_updates' => ('Point Updates',  'Earn, spend, and bonus updates'),
  'war_updates'   => ('Regional Wars',  'Rank changes and war milestones'),
  'bonus_awards'  => ('Bonus Awards',   'Campaign and promotion bonuses'),
  'system_alerts' => ('System Alerts',  'Important announcements'),
  _               => (key,              ''),
};

// ─── Reusable widgets ─────────────────────────────────────────────────────────

class _SectionLabel extends StatelessWidget {
  final String label;
  const _SectionLabel({required this.label});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: 20, bottom: 8, left: 4),
    child: Text(label.toUpperCase(),
        style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10,
            fontWeight: FontWeight.w700, letterSpacing: 0.8)),
  );
}

class _SettingsCard extends StatelessWidget {
  final List<Widget> children;
  const _SettingsCard({required this.children});
  @override
  Widget build(BuildContext context) => Container(
    decoration: BoxDecoration(
      color: NexusColors.surface,
      borderRadius: BorderRadius.circular(16),
      border: Border.all(color: Colors.white.withValues(alpha: 0.06)),
    ),
    clipBehavior: Clip.hardEdge,
    child: Column(mainAxisSize: MainAxisSize.min, children: children),
  );
}

class _Tile extends StatelessWidget {
  final IconData icon;
  final Color? iconColor;
  final String title, subtitle;
  final Widget? trailing;
  final VoidCallback? onTap;
  const _Tile({
    required this.icon, required this.title, required this.subtitle,
    required this.trailing, this.onTap, this.iconColor,
  });
  @override
  Widget build(BuildContext context) => ListTile(
    onTap: onTap,
    contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
    leading: Container(
      width: 38, height: 38,
      decoration: BoxDecoration(
        color: (iconColor ?? NexusColors.primary).withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(11),
      ),
      child: Icon(icon, color: iconColor ?? NexusColors.primary, size: 18),
    ),
    title: Text(title, style: const TextStyle(
        color: NexusColors.textPrimary, fontSize: 13, fontWeight: FontWeight.w600)),
    subtitle: Text(subtitle, style: const TextStyle(
        color: NexusColors.textSecondary, fontSize: 11)),
    trailing: trailing,
  );
}

class _ToggleTile extends StatelessWidget {
  final IconData icon;
  final String title, subtitle;
  final bool value;
  final ValueChanged<bool> onChanged;
  const _ToggleTile({
    required this.icon, required this.title, required this.subtitle,
    required this.value, required this.onChanged,
  });
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    child: Row(children: [
      Container(
        width: 38, height: 38,
        decoration: BoxDecoration(
          color: NexusColors.primary.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(11),
        ),
        child: Icon(icon, color: NexusColors.primary, size: 18),
      ),
      const SizedBox(width: 12),
      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title, style: const TextStyle(
            color: NexusColors.textPrimary, fontSize: 13, fontWeight: FontWeight.w600)),
        Text(subtitle, style: const TextStyle(
            color: NexusColors.textSecondary, fontSize: 11)),
      ])),
      Switch(
        value: value,
        onChanged: onChanged,
        activeColor: NexusColors.primary,
      ),
    ]),
  );
}

class _Divider extends StatelessWidget {
  const _Divider();
  @override
  Widget build(BuildContext context) =>
      const Divider(color: NexusColors.border, height: 1, indent: 16, endIndent: 16);
}
