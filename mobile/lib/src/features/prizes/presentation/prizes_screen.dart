import 'package:flutter/material.dart';
import '../../../core/theme/nexus_theme.dart';

class PrizesScreen extends StatelessWidget {
  const PrizesScreen({super.key});
  @override
  Widget build(BuildContext ctx) => Scaffold(
    appBar: AppBar(title: const Text('My Prizes')),
    body: const Center(
      child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
        Icon(Icons.card_giftcard, color: NexusColors.gold, size: 64),
        SizedBox(height: 16),
        Text('Prizes will appear here', style: TextStyle(
          color: NexusColors.textPrimary, fontSize: 18, fontWeight: FontWeight.w600)),
        SizedBox(height: 8),
        Text('Spin the wheel to win!', style: TextStyle(color: NexusColors.textSecondary)),
      ])),
  );
}