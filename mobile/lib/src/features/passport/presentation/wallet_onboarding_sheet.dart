import 'dart:io' show Platform;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── SharedPreferences keys ───────────────────────────────────────────────────
const _kWalletAdded        = 'nexus_wallet_added';          // bool: user added the pass
const _kWalletDismissCount = 'nexus_wallet_dismiss_count';  // int:  0–3 dismissals
const _kWalletCooldownEnd  = 'nexus_wallet_cooldown_end';   // int:  epoch ms

/// After [_kMaxDismissals] dismissals, wait this many days before prompting again.
const _kCoolingPeriodDays = 14;

/// Number of dismissals before a cooling period kicks in.
const _kMaxDismissals = 3;

// ─── Public API ───────────────────────────────────────────────────────────────

/// Returns true if the wallet onboarding prompt should be shown right now.
///
/// Cycle:
///   1. User has already added the pass → never show again.
///   2. Dismiss count < [_kMaxDismissals] → show.
///   3. Dismiss count >= [_kMaxDismissals] AND cooldown not expired → hide.
///   4. Dismiss count >= [_kMaxDismissals] AND cooldown expired → reset cycle, show again.
Future<bool> shouldShowWalletOnboarding() async {
  final prefs = await SharedPreferences.getInstance();

  // Pass already added — never prompt again.
  if (prefs.getBool(_kWalletAdded) ?? false) return false;

  final dismissCount = prefs.getInt(_kWalletDismissCount) ?? 0;

  if (dismissCount < _kMaxDismissals) {
    return true;
  }

  // Threshold reached — check cooling period.
  final cooldownEnd = prefs.getInt(_kWalletCooldownEnd) ?? 0;
  final now = DateTime.now().millisecondsSinceEpoch;

  if (now < cooldownEnd) {
    // Still cooling — do not show.
    return false;
  }

  // Cooling period expired — reset the cycle.
  await prefs.setInt(_kWalletDismissCount, 0);
  await prefs.remove(_kWalletCooldownEnd);
  return true;
}

/// Records a dismissal (user tapped "Maybe Later" or swiped down).
/// After [_kMaxDismissals] dismissals, starts a [_kCoolingPeriodDays]-day cooldown.
Future<void> recordWalletDismissed() async {
  final prefs = await SharedPreferences.getInstance();
  final count = (prefs.getInt(_kWalletDismissCount) ?? 0) + 1;
  await prefs.setInt(_kWalletDismissCount, count);

  if (count >= _kMaxDismissals) {
    final cooldownEnd = DateTime.now()
        .add(const Duration(days: _kCoolingPeriodDays))
        .millisecondsSinceEpoch;
    await prefs.setInt(_kWalletCooldownEnd, cooldownEnd);
  }
}

/// Records that the user successfully added the pass.
/// After this, the prompt will never appear again.
Future<void> recordWalletAdded() async {
  final prefs = await SharedPreferences.getInstance();
  await prefs.setBool(_kWalletAdded, true);
}

/// Shows the wallet onboarding bottom sheet.
/// Call this from the dashboard or OTP success screen after confirming
/// [shouldShowWalletOnboarding] returns true.
Future<void> showWalletOnboardingSheet(BuildContext context) async {
  if (!context.mounted) return;
  await showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    isDismissible: true,
    builder: (_) => const _WalletOnboardingSheet(),
  );
}

// ─── Sheet ────────────────────────────────────────────────────────────────────
class _WalletOnboardingSheet extends ConsumerStatefulWidget {
  const _WalletOnboardingSheet();
  @override
  ConsumerState<_WalletOnboardingSheet> createState() =>
      _WalletOnboardingSheetState();
}

class _WalletOnboardingSheetState
    extends ConsumerState<_WalletOnboardingSheet> {
  bool _loading = false;

  bool get _isIOS {
    try {
      return Platform.isIOS;
    } catch (_) {
      return false;
    }
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      // Swipe-down or tap-outside counts as a dismissal.
      onPopInvoked: (_) => recordWalletDismissed(),
      child: Container(
        decoration: const BoxDecoration(
          color: NexusColors.surfaceCard,
          borderRadius: BorderRadius.vertical(top: Radius.circular(28)),
        ),
        padding: EdgeInsets.fromLTRB(
            24, 16, 24, MediaQuery.of(context).viewInsets.bottom + 32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Handle bar
            Container(
              width: 40,
              height: 4,
              decoration: BoxDecoration(
                color: NexusColors.textMuted.withOpacity(0.4),
                borderRadius: BorderRadius.circular(2),
              ),
            ),
            const SizedBox(height: 24),

            // Lock screen icon
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                gradient: const LinearGradient(
                  colors: [NexusColors.primary, NexusColors.primaryDark],
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                ),
                borderRadius: BorderRadius.circular(20),
                boxShadow: [
                  BoxShadow(
                    color: NexusColors.primary.withOpacity(0.4),
                    blurRadius: 20,
                    offset: const Offset(0, 8),
                  ),
                ],
              ),
              child: const Icon(
                Icons.lock_outline_rounded,
                color: Colors.white,
                size: 36,
              ),
            ),
            const SizedBox(height: 20),

            // Headline
            const Text(
              'Your Passport on Your Lock Screen',
              textAlign: TextAlign.center,
              style: TextStyle(
                color: NexusColors.textPrimary,
                fontSize: 20,
                fontWeight: FontWeight.w800,
                fontFamily: 'Syne',
                height: 1.25,
              ),
            ),
            const SizedBox(height: 10),

            // Sub-copy
            const Text(
              'Add your Loyalty Nexus Digital Passport to your phone\'s Wallet. '
              'It stays on your lock screen and receives live updates — '
              'streak alerts, spin reminders, tier upgrades, and prize wins.',
              textAlign: TextAlign.center,
              style: TextStyle(
                color: NexusColors.textSecondary,
                fontSize: 14,
                height: 1.55,
              ),
            ),
            const SizedBox(height: 24),

            // Feature pills
            Wrap(
              spacing: 8,
              runSpacing: 8,
              alignment: WrapAlignment.center,
              children: const [
                _FeaturePill(
                    icon: Icons.lock_outline, label: 'Lock Screen'),
                _FeaturePill(
                    icon: Icons.notifications_active_outlined,
                    label: 'Live Alerts'),
                _FeaturePill(
                    icon: Icons.trending_up_outlined,
                    label: 'Tier Updates'),
                _FeaturePill(
                    icon: Icons.emoji_events_outlined,
                    label: 'Prize Wins'),
                _FeaturePill(
                    icon: Icons.casino_outlined, label: 'Spin Ready'),
              ],
            ),
            const SizedBox(height: 28),

            // Primary CTA
            SizedBox(
              width: double.infinity,
              height: 52,
              child: ElevatedButton(
                onPressed: _loading ? null : _addToWallet,
                style: ElevatedButton.styleFrom(
                  backgroundColor: NexusColors.primary,
                  foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(14)),
                  elevation: 0,
                ),
                child: _loading
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                            strokeWidth: 2, color: Colors.white),
                      )
                    : Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Text(_isIOS ? '🍎' : '🤖',
                              style: const TextStyle(fontSize: 18)),
                          const SizedBox(width: 8),
                          Text(
                            _isIOS
                                ? 'Add to Apple Wallet'
                                : 'Add to Google Wallet',
                            style: const TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                        ],
                      ),
              ),
            ),
            const SizedBox(height: 12),

            // "Maybe Later" — records a dismissal
            SizedBox(
              width: double.infinity,
              height: 48,
              child: TextButton(
                onPressed: () async {
                  await recordWalletDismissed();
                  if (mounted) Navigator.of(context).pop();
                },
                child: const Text(
                  'Maybe Later',
                  style: TextStyle(
                    color: NexusColors.textSecondary,
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _addToWallet() async {
    setState(() => _loading = true);
    try {
      if (_isIOS) {
        final url =
            await ref.read(passportApiProvider).getApplePKPassURL();
        final uri = Uri.parse(url);
        if (await canLaunchUrl(uri)) {
          await launchUrl(uri, mode: LaunchMode.externalApplication);
          // Mark pass as added — prompt will never show again.
          await recordWalletAdded();
          if (mounted) Navigator.of(context).pop();
        } else {
          _showError('Apple Wallet is not available on this device');
        }
      } else {
        final urls =
            await ref.read(passportApiProvider).getWalletPassURLs();
        final googleUrl = urls['google_wallet_url']?.toString() ?? '';
        if (googleUrl.isEmpty) {
          _showError('Google Wallet is not available right now');
          return;
        }
        final uri = Uri.parse(googleUrl);
        if (await canLaunchUrl(uri)) {
          await launchUrl(uri, mode: LaunchMode.externalApplication);
          // Mark pass as added — prompt will never show again.
          await recordWalletAdded();
          if (mounted) Navigator.of(context).pop();
        } else {
          _showError('Could not open Google Wallet');
        }
      }
    } catch (e) {
      _showError('Something went wrong. Please try from your Passport page.');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _showError(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(message),
      backgroundColor: NexusColors.surfaceCard,
      behavior: SnackBarBehavior.floating,
      shape:
          RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
    ));
  }
}

// ─── Feature Pill ─────────────────────────────────────────────────────────────
class _FeaturePill extends StatelessWidget {
  final IconData icon;
  final String label;
  const _FeaturePill({required this.icon, required this.label});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: NexusColors.primary.withOpacity(0.12),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(
            color: NexusColors.primary.withOpacity(0.3), width: 1),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 14, color: NexusColors.primary),
          const SizedBox(width: 5),
          Text(
            label,
            style: const TextStyle(
              color: NexusColors.primary,
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}
