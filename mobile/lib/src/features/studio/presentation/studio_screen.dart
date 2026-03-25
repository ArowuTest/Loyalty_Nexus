import 'package:flutter/material.dart';
import '../../../core/theme/nexus_theme.dart';

class StudioScreen extends StatelessWidget {
  const StudioScreen({super.key});
  @override
  Widget build(BuildContext ctx) => Scaffold(
    appBar: AppBar(title: const Text('AI Studio')),
    body: const Center(
      child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
        Icon(Icons.auto_awesome, color: NexusColors.primary, size: 64),
        SizedBox(height: 16),
        Text('Nexus AI Studio', style: TextStyle(
          color: NexusColors.textPrimary, fontSize: 22, fontWeight: FontWeight.bold)),
        SizedBox(height: 8),
        Text('Open the web app for full studio access',
          style: TextStyle(color: NexusColors.textSecondary)),
      ]),
    ),
  );
}