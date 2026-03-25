import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

final _profileProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getProfile();
});

final _walletProvider = FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getWallet();
});

// Nigerian states for REQ-1.5
const _nigerianStates = [
  'Abia','Adamawa','Akwa Ibom','Anambra','Bauchi','Bayelsa','Benue','Borno',
  'Cross River','Delta','Ebonyi','Edo','Ekiti','Enugu','FCT Abuja','Gombe',
  'Imo','Jigawa','Kaduna','Kano','Katsina','Kebbi','Kogi','Kwara','Lagos',
  'Nasarawa','Niger','Ogun','Ondo','Osun','Oyo','Plateau','Rivers','Sokoto',
  'Taraba','Yobe','Zamfara',
];

class ProfileScreen extends ConsumerStatefulWidget {
  const ProfileScreen({super.key});
  @override ConsumerState<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends ConsumerState<ProfileScreen> {
  bool _linkingMomo   = false;
  bool _savingState   = false;
  String? _momoError;
  String? _stateError;
  final _momoCtrl     = TextEditingController();

  @override
  void dispose() { _momoCtrl.dispose(); super.dispose(); }

  Future<void> _linkMomo() async {
    final num = _momoCtrl.text.trim();
    if (num.length < 11) {
      setState(() => _momoError = 'Enter a valid 11-digit MTN number');
      return;
    }
    setState(() { _linkingMomo = true; _momoError = null; });
    try {
      await ref.read(userApiProvider).requestMoMoLink(num);
      ref.invalidate(_profileProvider);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
          content: Text('MoMo number linked! Verification in progress.'),
          backgroundColor: NexusColors.green));
        Navigator.pop(context);
      }
    } on ApiException catch (e) {
      setState(() => _momoError = e.message);
    } finally {
      setState(() => _linkingMomo = false);
    }
  }

  void _showMoMoSheet(Map profile) {
    _momoCtrl.text = profile['momo_number']?.toString() ?? '';
    _momoError = null;
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: NexusColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
      builder: (_) => StatefulBuilder(builder: (ctx, setBS) => Padding(
        padding: EdgeInsets.only(
            left: 24, right: 24, top: 24,
            bottom: MediaQuery.of(ctx).viewInsets.bottom + 24),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Text('Link MoMo Wallet', style: TextStyle(
              color: NexusColors.textPrimary, fontSize: 20, fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          const Text('Enter your MTN MoMo number to receive cash prizes directly.',
              style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
          const SizedBox(height: 20),
          TextField(
            controller: _momoCtrl,
            keyboardType: TextInputType.phone,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly, LengthLimitingTextInputFormatter(11)],
            style: const TextStyle(color: NexusColors.textPrimary),
            decoration: InputDecoration(
              hintText: '08031234567',
              prefixIcon: const Icon(Icons.phone_android, color: NexusColors.primary),
              filled: true, fillColor: NexusColors.background,
              border: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
                  borderSide: const BorderSide(color: NexusColors.border)),
              errorText: _momoError,
            ),
          ),
          const SizedBox(height: 8),
          const Text("Don't have MoMo? Dial *671# on your MTN line to open one in 2 minutes.",
              style: TextStyle(color: NexusColors.gold, fontSize: 12)),
          const SizedBox(height: 20),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton(
              onPressed: _linkingMomo ? null : () async {
                await _linkMomo();
                if (mounted) setBS(() {});
              },
              child: _linkingMomo
                  ? const SizedBox(width: 20, height: 20,
                      child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
                  : const Text('Link MoMo Number'),
            ),
          ),
        ]),
      )),
    );
  }

  void _showStateSheet(Map profile) {
    String? selected = profile['state']?.toString();
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: NexusColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
      builder: (_) => DraggableScrollableSheet(
        initialChildSize: 0.6, maxChildSize: 0.9, minChildSize: 0.4,
        expand: false,
        builder: (ctx, sc) => Column(children: [
          const SizedBox(height: 16),
          Container(width: 40, height: 4,
              decoration: BoxDecoration(color: NexusColors.border,
                  borderRadius: BorderRadius.circular(2))),
          const SizedBox(height: 16),
          const Text('Your State', style: TextStyle(color: NexusColors.textPrimary,
              fontSize: 18, fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          const Text('Used for Regional Wars team assignment',
              style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
          const Divider(color: NexusColors.border),
          Expanded(child: ListView.builder(
            controller: sc,
            itemCount: _nigerianStates.length,
            itemBuilder: (_, i) => ListTile(
              title: Text(_nigerianStates[i],
                  style: const TextStyle(color: NexusColors.textPrimary)),
              trailing: selected == _nigerianStates[i]
                  ? const Icon(Icons.check_circle, color: NexusColors.primary)
                  : null,
              onTap: () async {
                selected = _nigerianStates[i];
                setState(() => _savingState = true);
                try {
                  // PATCH /user/profile with state
                  await ref.read(dioProvider).apiPost('/user/profile/state',
                      data: {'state': selected});
                  ref.invalidate(_profileProvider);
                  if (mounted) Navigator.pop(ctx);
                } catch (_) {} finally {
                  if (mounted) setState(() => _savingState = false);
                }
              },
            ),
          )),
        ]),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final profileAsync = ref.watch(_profileProvider);
    final walletAsync  = ref.watch(_walletProvider);
    final phone        = ref.watch(authStateProvider).phoneNumber ?? '—';

    return Scaffold(
      appBar: AppBar(
        title: const Text('My Profile'),
        actions: [
          IconButton(
            icon: const Icon(Icons.notifications_outlined),
            onPressed: () => context.push('/notifications'),
          ),
        ],
      ),
      body: profileAsync.when(
        loading: () => const Center(child: CircularProgressIndicator(color: NexusColors.primary)),
        error:   (_, __) => Center(child: _retryBtn(() => ref.invalidate(_profileProvider))),
        data: (profile) {
          final tier        = profile['tier']?.toString() ?? 'BRONZE';
          final streak      = profile['streak_count'] as int? ?? 0;
          final momoVerified = profile['momo_verified'] as bool? ?? false;
          final momoNumber  = profile['momo_number']?.toString();
          final state       = profile['state']?.toString() ?? 'Not set';
          final memberSince = profile['created_at']?.toString() ?? '';

          return RefreshIndicator(
            onRefresh: () async {
              ref.invalidate(_profileProvider);
              ref.invalidate(_walletProvider);
            },
            color: NexusColors.primary,
            child: ListView(
              padding: const EdgeInsets.all(20),
              children: [
                // ── Avatar card ──────────────────────────────────
                Container(
                  padding: const EdgeInsets.all(20),
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                        colors: [Color(0xFF4A56EE), Color(0xFF8B5CF6)],
                        begin: Alignment.topLeft, end: Alignment.bottomRight),
                    borderRadius: BorderRadius.circular(20)),
                  child: Row(children: [
                    CircleAvatar(radius: 32,
                        backgroundColor: Colors.white.withOpacity(0.2),
                        child: Text(phone.length >= 4 ? phone.substring(phone.length - 4) : phone,
                            style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold))),
                    const SizedBox(width: 16),
                    Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Text(phone, style: const TextStyle(
                          color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold)),
                      const SizedBox(height: 4),
                      Row(children: [
                        _tierBadge(tier),
                        const SizedBox(width: 8),
                        Text('Member since ${_fmtDate(memberSince)}',
                            style: const TextStyle(color: Colors.white60, fontSize: 11)),
                      ]),
                    ])),
                  ]),
                ),
                const SizedBox(height: 20),

                // ── Wallet summary ────────────────────────────────
                walletAsync.when(
                  loading: () => const SizedBox(height: 60,
                      child: Center(child: CircularProgressIndicator(strokeWidth: 2))),
                  error: (_, __) => const SizedBox.shrink(),
                  data: (wallet) => Row(children: [
                    Expanded(child: _statCard('💎 Pulse Points',
                        '${wallet['pulse_points'] ?? 0}', 'pts')),
                    const SizedBox(width: 12),
                    Expanded(child: _statCard('🎰 Spin Credits',
                        '${wallet['spin_credits'] ?? 0}', 'left')),
                    const SizedBox(width: 12),
                    Expanded(child: _statCard('🔥 Streak',
                        '$streak', streak == 1 ? 'day' : 'days')),
                  ]),
                ),
                const SizedBox(height: 24),

                // ── Section: Account ──────────────────────────────
                _sectionHeader('Account'),
                _tile(
                  icon: Icons.map_outlined,
                  title: 'My State',
                  subtitle: state,
                  trailing: const Icon(Icons.chevron_right, color: NexusColors.textSecondary),
                  onTap: () => _showStateSheet(profile),
                ),
                _tile(
                  icon: Icons.account_balance_wallet_outlined,
                  title: 'MoMo Wallet',
                  subtitle: momoVerified
                      ? '✅ Verified: $momoNumber'
                      : momoNumber != null
                          ? '⏳ Pending: $momoNumber'
                          : 'Tap to link for cash prizes',
                  trailing: momoVerified
                      ? const Icon(Icons.verified, color: NexusColors.green, size: 18)
                      : const Icon(Icons.chevron_right, color: NexusColors.textSecondary),
                  onTap: () => _showMoMoSheet(profile),
                ),
                const SizedBox(height: 16),

                // ── Section: Digital Passport ─────────────────────
                _sectionHeader('Digital Passport'),
                Container(
                  margin: const EdgeInsets.only(bottom: 8),
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: NexusColors.surface,
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(color: NexusColors.border)),
                  child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    const Text('Your lock-screen loyalty card',
                        style: TextStyle(color: NexusColors.textPrimary,
                            fontWeight: FontWeight.bold)),
                    const SizedBox(height: 4),
                    const Text('Always shows your Pulse Points & streak — even when your Loyalty Nexus SIM is not active.',
                        style: TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
                    const SizedBox(height: 12),
                    Row(children: [
                      Expanded(child: _passportBtn(
                        icon: Icons.apple,
                        label: 'Add to Apple Wallet',
                        onTap: () => _downloadPassport('apple'),
                      )),
                      const SizedBox(width: 10),
                      Expanded(child: _passportBtn(
                        icon: Icons.wallet,
                        label: 'Add to Google Wallet',
                        onTap: () => _downloadPassport('google'),
                      )),
                    ]),
                  ]),
                ),
                const SizedBox(height: 16),

                // ── Section: Notifications ────────────────────────
                _sectionHeader('Notifications'),
                _tile(
                  icon: Icons.notifications_outlined,
                  title: 'Notification Inbox',
                  subtitle: 'View all your alerts',
                  trailing: const Icon(Icons.chevron_right, color: NexusColors.textSecondary),
                  onTap: () => context.push('/notifications'),
                ),
                const SizedBox(height: 16),

                // ── Section: Support ──────────────────────────────
                _sectionHeader('Support'),
                _tile(
                  icon: Icons.help_outline_rounded,
                  title: 'How to Earn Points',
                  subtitle: 'Recharge ₦1,000 = 4 pts + 1 spin',
                  onTap: () => _showEarningInfo(),
                ),
                _tile(
                  icon: Icons.privacy_tip_outlined,
                  title: 'Privacy Policy',
                  subtitle: 'How we use your data',
                  onTap: () {},
                ),
                _tile(
                  icon: Icons.ussd,
                  title: 'USSD Access',
                  subtitle: 'Dial *789*NEXUS# on any phone',
                  onTap: () {},
                ),
                const SizedBox(height: 24),

                // ── Sign out ──────────────────────────────────────
                OutlinedButton.icon(
                  onPressed: () async {
                    await ref.read(authStateProvider.notifier).logout();
                    if (context.mounted) context.go('/');
                  },
                  icon: const Icon(Icons.logout, color: NexusColors.red),
                  label: const Text('Sign Out',
                      style: TextStyle(color: NexusColors.red, fontWeight: FontWeight.bold)),
                  style: OutlinedButton.styleFrom(
                    side: const BorderSide(color: NexusColors.red),
                    minimumSize: const Size(double.infinity, 52),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                  ),
                ),
                const SizedBox(height: 32),
              ],
            ),
          );
        },
      ),
    );
  }

  void _downloadPassport(String platform) async {
    try {
      final data = await ref.read(userApiProvider).getPassport();
      final url = platform == 'apple'
          ? data['pkpass_url']?.toString()
          : data['google_wallet_url']?.toString();
      if (url == null || url.isEmpty) {
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Passport not available yet')));
        return;
      }
      // In a real app: launch url_launcher to open the .pkpass / google wallet link
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Opening $platform Wallet…')));
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: ${(e as ApiException).message}')));
    }
  }

  void _showEarningInfo() {
    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        backgroundColor: NexusColors.surface,
        title: const Text('How to Earn', style: TextStyle(color: NexusColors.textPrimary)),
        content: const Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text('💎 Pulse Points', style: TextStyle(color: NexusColors.primary, fontWeight: FontWeight.bold)),
          Text('• 1 point per ₦250 recharged\n• Used for AI Studio tools', style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
          SizedBox(height: 12),
          Text('🎰 Spin Credits', style: TextStyle(color: NexusColors.gold, fontWeight: FontWeight.bold)),
          Text('• 1 credit per ₦1,000 recharged\n• Used for the Spin & Win wheel', style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
          SizedBox(height: 12),
          Text('🔥 Recharge Streak', style: TextStyle(color: NexusColors.red, fontWeight: FontWeight.bold)),
          Text('• Recharge within 36 hours to keep streak\n• Milestone bonuses at 7, 14, 30 days', style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
        ]),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Got it', style: TextStyle(color: NexusColors.primary)),
          ),
        ],
      ),
    );
  }

  Widget _statCard(String label, String value, String unit) => Container(
    padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 10),
    decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: NexusColors.border)),
    child: Column(children: [
      Text(label, style: const TextStyle(fontSize: 11, color: NexusColors.textSecondary),
          textAlign: TextAlign.center),
      const SizedBox(height: 4),
      Text(value, style: const TextStyle(
          fontSize: 22, fontWeight: FontWeight.bold, color: NexusColors.textPrimary)),
      Text(unit, style: const TextStyle(fontSize: 10, color: NexusColors.textSecondary)),
    ]),
  );

  Widget _sectionHeader(String t) => Padding(
    padding: const EdgeInsets.only(bottom: 10),
    child: Text(t, style: const TextStyle(
        color: NexusColors.textSecondary, fontSize: 11,
        fontWeight: FontWeight.bold, letterSpacing: 1.2)),
  );

  Widget _tile({required IconData icon, required String title,
      required String subtitle, Widget? trailing, VoidCallback? onTap}) =>
    Container(
      margin: const EdgeInsets.only(bottom: 6),
      decoration: BoxDecoration(
          color: NexusColors.surface, borderRadius: BorderRadius.circular(14),
          border: Border.all(color: NexusColors.border)),
      child: ListTile(
        leading: Container(
          width: 40, height: 40,
          decoration: BoxDecoration(
              color: NexusColors.primary.withOpacity(0.1),
              borderRadius: BorderRadius.circular(10)),
          child: Icon(icon, color: NexusColors.primary, size: 20)),
        title: Text(title, style: const TextStyle(
            color: NexusColors.textPrimary, fontWeight: FontWeight.w500, fontSize: 14)),
        subtitle: Text(subtitle,
            style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12),
            maxLines: 2),
        trailing: trailing,
        onTap: onTap,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      ),
    );

  Widget _passportBtn({required IconData icon, required String label, required VoidCallback onTap}) =>
    OutlinedButton.icon(
      onPressed: onTap,
      icon: Icon(icon, size: 16),
      label: Text(label, style: const TextStyle(fontSize: 11)),
      style: OutlinedButton.styleFrom(
        foregroundColor: NexusColors.textPrimary,
        side: const BorderSide(color: NexusColors.border),
        padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 8),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
      ),
    );

  Widget _tierBadge(String tier) {
    final colors = {
      'BRONZE':   const Color(0xFFCD7F32),
      'SILVER':   const Color(0xFFC0C0C0),
      'GOLD':     const Color(0xFFFFD700),
      'PLATINUM': const Color(0xFFE5E4E2),
    };
    final c = colors[tier] ?? NexusColors.primary;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
          color: c.withOpacity(0.2),
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: c.withOpacity(0.5))),
      child: Text(tier,
          style: TextStyle(color: c, fontSize: 10, fontWeight: FontWeight.bold)),
    );
  }

  Widget _retryBtn(VoidCallback onTap) => ElevatedButton.icon(
      onPressed: onTap,
      icon: const Icon(Icons.refresh),
      label: const Text('Retry'));

  String _fmtDate(String iso) {
    try {
      final d = DateTime.parse(iso);
      const months = ['Jan','Feb','Mar','Apr','May','Jun',
                      'Jul','Aug','Sep','Oct','Nov','Dec'];
      return '${months[d.month - 1]} ${d.year}';
    } catch (_) { return ''; }
  }
}
