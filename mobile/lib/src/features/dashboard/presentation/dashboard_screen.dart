import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';

class DashboardScreen extends StatelessWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _buildHeader(),
              const SizedBox(height: 32),
              _buildBalanceGrid(),
              const SizedBox(height: 32),
              _buildPassportCard(),
              const SizedBox(height: 32),
              _buildTournamentSection(),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildHeader() {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('NEXUS', style: TextStyle(fontSize: 24, fontWeight: FontWeight.black, fontStyle: FontStyle.italic, color: Color(0xFFD4AF37))),
            Text('Lagos, Nigeria', style: TextStyle(fontSize: 12, color: Colors.grey[600], fontWeight: FontWeight.bold)),
          ],
        ),
        const CircleAvatar(radius: 24, backgroundColor: Color(0xFFD4AF37)),
      ],
    );
  }

  Widget _buildBalanceGrid() {
    return GridView.count(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      crossAxisCount: 2,
      crossAxisSpacing: 16,
      mainAxisSpacing: 16,
      children: [
        _buildStatCard('Airtime', '₦4,250', LucideIcons.phone),
        _buildStatCard('Data', '12.5 GB', LucideIcons.wifi),
      ],
    );
  }

  Widget _buildStatCard(String title, String value, IconData icon) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.05),
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: Colors.white.withOpacity(0.1)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, color: const Color(0xFFD4AF37), size: 20),
          const SizedBox(height: 12),
          Text(title, style: const TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: Colors.grey)),
          Text(value, style: const TextStyle(fontSize: 20, fontWeight: FontWeight.black, color: Colors.white)),
        ],
      ),
    );
  }

  Widget _buildPassportCard() {
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(32),
        border: Border.all(color: const Color(0xFFD4AF37).withOpacity(0.3)),
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [Colors.white.withOpacity(0.05), Colors.white.withOpacity(0.01)],
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                decoration: BoxDecoration(color: Colors.green.withOpacity(0.1), borderRadius: BorderRadius.circular(20)),
                child: const Text('PASSPORT ACTIVE', style: TextStyle(color: Colors.green, fontSize: 8, fontWeight: FontWeight.black)),
              ),
              const Icon(LucideIcons.smartphone, color: Color(0xFFD4AF37)),
            ],
          ),
          const SizedBox(height: 24),
          const Text('Digital Passport', style: TextStyle(fontSize: 20, fontWeight: FontWeight.black, fontStyle: FontStyle.italic)),
          const Text('Persistent Lock-screen Card', style: TextStyle(color: Colors.grey, fontSize: 10, fontWeight: FontWeight.bold)),
        ],
      ),
    );
  }

  Widget _buildTournamentSection() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('REGIONAL WARS', style: TextStyle(fontSize: 18, fontWeight: FontWeight.black, fontStyle: FontStyle.italic)),
            Text('LIVE', style: TextStyle(color: Color(0xFFD4AF37), fontSize: 10, fontWeight: FontWeight.black)),
          ],
        ),
        const SizedBox(height: 16),
        Container(
          height: 100,
          decoration: BoxDecoration(color: Colors.white.withOpacity(0.05), borderRadius: BorderRadius.circular(24)),
          child: const Center(child: Text('Tournament Feed...', style: TextStyle(color: Colors.grey, fontSize: 12))),
        ),
      ],
    );
  }
}
