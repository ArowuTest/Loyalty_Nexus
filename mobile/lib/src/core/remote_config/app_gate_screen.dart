// The blocking screen shown when remote config says this build must not be used.
//
// Wrapped around the whole router (see main.dart) rather than shown as a dialog,
// because a dialog can be dismissed and a route can be deep-linked past. If the
// app is genuinely broken or the backend is down, there must be no way around it.

import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import 'remote_config_service.dart';

class AppGateScreen extends StatefulWidget {
  const AppGateScreen({super.key, required this.gate, required this.onRechecked});
  final AppGate gate;

  /// Rebuilds the parent gate wrapper after a manual re-check, so a cleared gate
  /// dismisses this screen instead of stranding the user here until relaunch.
  final VoidCallback onRechecked;

  @override
  State<AppGateScreen> createState() => _AppGateScreenState();
}

class _AppGateScreenState extends State<AppGateScreen> {
  bool _checking = false;

  bool get _isMaintenance => widget.gate.status == AppGateStatus.maintenance;

  Future<void> _recheck() async {
    setState(() => _checking = true);
    await RemoteConfigService.instance.refresh();
    if (!mounted) return;
    setState(() => _checking = false);
    // Ask the wrapper to re-evaluate; if the gate cleared, this screen goes away.
    widget.onRechecked();
  }

  Future<void> _openStore() async {
    final url = widget.gate.storeUrl;
    if (url == null) return;
    try {
      await launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication);
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Could not open the store — please update manually.')),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final g = widget.gate;
    return Scaffold(
      backgroundColor: const Color(0xFF0F1123),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  _isMaintenance
                      ? Icons.construction_rounded
                      : Icons.system_update_alt_rounded,
                  size: 56,
                  color: const Color(0xFF5F72F9),
                ),
                const SizedBox(height: 24),
                Text(
                  _isMaintenance ? 'Back shortly' : 'Update required',
                  textAlign: TextAlign.center,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 22,
                    fontWeight: FontWeight.w800,
                    fontFamily: 'Syne',
                  ),
                ),
                const SizedBox(height: 12),
                Text(
                  g.message ??
                      (_isMaintenance
                          ? 'Loyalty Nexus is briefly unavailable. Please try again shortly.'
                          : 'Please update to continue using Loyalty Nexus.'),
                  textAlign: TextAlign.center,
                  style: const TextStyle(
                      color: Color(0xFF9ca3af), fontSize: 14, height: 1.5),
                ),
                const SizedBox(height: 32),

                if (!_isMaintenance && g.storeUrl != null)
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton(
                      onPressed: _openStore,
                      style: FilledButton.styleFrom(
                        minimumSize: const Size(0, 50),
                        backgroundColor: const Color(0xFF5F72F9),
                      ),
                      child: const Text('Update now'),
                    ),
                  ),

                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton(
                    onPressed: _checking ? null : _recheck,
                    style: OutlinedButton.styleFrom(
                      minimumSize: const Size(0, 50),
                      foregroundColor: Colors.white70,
                      side: const BorderSide(color: Colors.white24),
                    ),
                    child: _checking
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2))
                        : const Text('Check again'),
                  ),
                ),

                const SizedBox(height: 20),
                Text(
                  'Version ${RemoteConfigService.instance.currentVersion}',
                  style: const TextStyle(color: Color(0xFF4b5563), fontSize: 11),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Non-blocking prompt for an optional update.
///
/// Separate from the blocking screen on purpose: "a newer version exists" must
/// never stop someone using the app they already have.
class UpdateAvailableBanner extends StatelessWidget {
  const UpdateAvailableBanner({super.key, required this.gate, required this.onDismiss});
  final AppGate gate;
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: const Color(0xFF1a1d35),
      child: SafeArea(
        bottom: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(14, 8, 6, 8),
          child: Row(
            children: [
              const Icon(Icons.arrow_circle_up_rounded,
                  size: 18, color: Color(0xFF5F72F9)),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  gate.message ?? 'A new version is available.',
                  style: const TextStyle(color: Colors.white70, fontSize: 12),
                ),
              ),
              if (gate.storeUrl != null)
                TextButton(
                  onPressed: () => launchUrl(Uri.parse(gate.storeUrl!),
                      mode: LaunchMode.externalApplication),
                  child: const Text('Update', style: TextStyle(fontSize: 12)),
                ),
              IconButton(
                onPressed: onDismiss,
                icon: const Icon(Icons.close_rounded, size: 16, color: Colors.white38),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
