import 'package:flutter/material.dart';
import 'dart:math' as math;

class SpinWheelScreen extends StatefulWidget {
  const SpinWheelScreen({super.key});

  @override
  State<SpinWheelScreen> createState() => _SpinWheelScreenState();
}

class _SpinWheelScreenState extends State<SpinWheelScreen> with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<double> _animation;
  double _angle = 0;

  final List<String> _prizes = [
    '₦10,000', '5GB Data', '₦500', '100 PTS', 'TRY AGAIN', '2GB Data', '₦200', '50 PTS'
  ];

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 5),
    );

    _animation = CurvedAnimation(
      parent: _controller,
      curve: Curves.elasticOut,
    );
  }

  void _spin() {
    if (_controller.isAnimating) return;
    
    final random = math.Random();
    final double extraSpins = (random.nextInt(5) + 5) * 2 * math.pi;
    final double targetAngle = random.nextDouble() * 2 * math.pi;
    
    _controller.reset();
    setState(() {
      _angle = extraSpins + targetAngle;
    });
    _controller.forward();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('SPIN & WIN', style: TextStyle(fontWeight: FontWeight.black, fontStyle: FontStyle.italic)),
        backgroundColor: Colors.transparent,
        elevation: 0,
      ),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            AnimatedBuilder(
              animation: _animation,
              builder: (context, child) {
                return Transform.rotate(
                  angle: _animation.value * _angle,
                  child: _buildWheel(),
                );
              },
            ),
            const SizedBox(height: 64),
            ElevatedButton(
              onPressed: _spin,
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFFD4AF37),
                foregroundColor: Colors.black,
                padding: const EdgeInsets.symmetric(horizontal: 48, vertical: 20),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
              ),
              child: const Text('SPIN NOW', style: TextStyle(fontWeight: FontWeight.black, letterSpacing: 2)),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildWheel() {
    return Container(
      width: 300,
      height: 300,
      decoration: BoxDecoration(
        shape: BoxType.circle,
        border: Border.all(color: const Color(0xFFD4AF37), width: 8),
        boxShadow: [BoxShadow(color: const Color(0xFFD4AF37).withOpacity(0.2), blurRadius: 40)],
      ),
      child: Stack(
        children: List.generate(_prizes.length, (index) {
          return Transform.rotate(
            angle: (index * 2 * math.pi / _prizes.length),
            child: Align(
              alignment: Alignment.topCenter,
              child: Padding(
                padding: const EdgeInsets.all(20),
                child: Text(
                  _prizes[index],
                  style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12),
                ),
              ),
            ),
          );
        }),
      ),
    );
  }
}
