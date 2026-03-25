import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class AuthState {
  final String? token;
  final String? phoneNumber;
  const AuthState({this.token, this.phoneNumber});
  AuthState copyWith({String? token, String? phoneNumber}) =>
      AuthState(token: token ?? this.token, phoneNumber: phoneNumber ?? this.phoneNumber);
}

class AuthNotifier extends StateNotifier<AuthState> {
  static const _storage = FlutterSecureStorage();
  AuthNotifier() : super(const AuthState()) { _init(); }
  Future<void> _init() async {
    final token = await _storage.read(key: 'nexus_token');
    final phone = await _storage.read(key: 'nexus_phone');
    if (token != null) state = state.copyWith(token: token, phoneNumber: phone);
  }
  Future<void> setAuth(String token, String phone) async {
    await _storage.write(key: 'nexus_token', value: token);
    await _storage.write(key: 'nexus_phone', value: phone);
    state = state.copyWith(token: token, phoneNumber: phone);
  }
  Future<void> logout() async {
    await _storage.deleteAll();
    state = const AuthState();
  }
}

final authStateProvider = StateNotifierProvider<AuthNotifier, AuthState>(
  (ref) => AuthNotifier());