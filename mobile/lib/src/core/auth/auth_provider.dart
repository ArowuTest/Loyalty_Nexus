import 'dart:developer';
import 'dart:io';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../api/api_client.dart';

// ─── State ────────────────────────────────────────────────────────────────────

class AuthState {
  final String? token;
  final String? phoneNumber;
  final bool    isNewUser;
  final bool    isLoading;

  const AuthState({
    this.token,
    this.phoneNumber,
    this.isNewUser  = false,
    this.isLoading  = true,
  });

  bool get isAuthenticated => token != null;

  AuthState copyWith({
    String? token,
    String? phoneNumber,
    bool?   isNewUser,
    bool?   isLoading,
  }) => AuthState(
    token:       token       ?? this.token,
    phoneNumber: phoneNumber ?? this.phoneNumber,
    isNewUser:   isNewUser   ?? this.isNewUser,
    isLoading:   isLoading   ?? this.isLoading,
  );
}

// ─── Notifier ─────────────────────────────────────────────────────────────────

class AuthNotifier extends StateNotifier<AuthState> {
  static const _storage = FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );
  final Ref _ref;

  AuthNotifier(this._ref) : super(const AuthState()) { _init(); }

  // ── Bootstrap ──────────────────────────────────────────────────────────────

  Future<void> _init() async {
    try {
      final token = await _storage.read(key: 'nexus_token');
      final phone = await _storage.read(key: 'nexus_phone');
      if (token != null && token.isNotEmpty) {
        state = state.copyWith(
          token:       token,
          phoneNumber: phone,
          isLoading:   false,
        );
        // Re-register FCM silently — token may have rotated
        _registerFcmToken().ignore();
        return;
      }
    } catch (e) {
      log('[AUTH] Init error: $e');
    }
    state = state.copyWith(isLoading: false);
  }

  // ── Public API ─────────────────────────────────────────────────────────────

  /// Called after OTP verify — stores token and marks isNewUser for routing
  Future<void> setAuth({
    required String token,
    required String phone,
    bool isNewUser = false,
  }) async {
    await _storage.write(key: 'nexus_token', value: token);
    await _storage.write(key: 'nexus_phone', value: phone);
    state = state.copyWith(
      token:       token,
      phoneNumber: phone,
      isNewUser:   isNewUser,
      isLoading:   false,
    );
    _registerFcmToken().ignore();
  }

  /// Called after new-user registration completes — clear isNewUser flag
  void markOnboarded() {
    state = state.copyWith(isNewUser: false);
  }

  Future<void> logout() async {
    try {
      // Best-effort: unregister FCM token
      final fcmToken = await FirebaseMessaging.instance.getToken();
      if (fcmToken != null) {
        await _ref.read(notificationsApiProvider)
            .registerPushToken('', Platform.isAndroid ? 'android' : 'ios');
      }
    } catch (_) {}
    await _storage.deleteAll();
    state = const AuthState(isLoading: false);
  }

  // ── FCM ────────────────────────────────────────────────────────────────────

  Future<void> _registerFcmToken() async {
    try {
      if (Platform.isIOS) {
        await FirebaseMessaging.instance.getAPNSToken();
      }
      final fcmToken = await FirebaseMessaging.instance.getToken();
      if (fcmToken == null) return;
      final platform = Platform.isAndroid ? 'android' : 'ios';
      await _ref.read(notificationsApiProvider)
          .registerPushToken(fcmToken, platform);
      log('[AUTH] FCM token registered');
    } catch (e) {
      log('[AUTH] FCM registration skipped: $e');
    }
  }
}

// ─── Providers ────────────────────────────────────────────────────────────────

final authStateProvider = StateNotifierProvider<AuthNotifier, AuthState>(
    (ref) => AuthNotifier(ref));
