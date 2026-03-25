import 'dart:developer';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../api/api_client.dart';

class AuthState {
  final String? token;
  final String? phoneNumber;
  const AuthState({this.token, this.phoneNumber});
  AuthState copyWith({String? token, String? phoneNumber}) =>
      AuthState(token: token ?? this.token, phoneNumber: phoneNumber ?? this.phoneNumber);
}

class AuthNotifier extends StateNotifier<AuthState> {
  static const _storage = FlutterSecureStorage();
  final Ref _ref;

  AuthNotifier(this._ref) : super(const AuthState()) { _init(); }

  Future<void> _init() async {
    final token = await _storage.read(key: 'nexus_token');
    final phone = await _storage.read(key: 'nexus_phone');
    if (token != null) {
      state = state.copyWith(token: token, phoneNumber: phone);
      // Re-register FCM token on app restart
      await _registerFcmToken();
    }
  }

  Future<void> setAuth(String token, String phone) async {
    await _storage.write(key: 'nexus_token', value: token);
    await _storage.write(key: 'nexus_phone', value: phone);
    state = state.copyWith(token: token, phoneNumber: phone);
    // Register FCM token immediately after login
    await _registerFcmToken();
  }

  Future<void> logout() async {
    await _storage.deleteAll();
    state = const AuthState();
  }

  Future<void> _registerFcmToken() async {
    try {
      await FirebaseMessaging.instance.requestPermission(alert: true, badge: true, sound: true);
      final fcmToken = await FirebaseMessaging.instance.getToken();
      if (fcmToken == null) return;
      log('[AUTH] Registering FCM token: ${fcmToken.substring(0, 20)}...');
      await _ref.read(notificationsApiProvider).registerPushToken(fcmToken, 'android');
    } catch (e) {
      log('[AUTH] FCM token registration skipped: $e');
    }
  }
}

final authStateProvider = StateNotifierProvider<AuthNotifier, AuthState>(
    (ref) => AuthNotifier(ref));
