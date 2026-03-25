import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

const _baseUrl = String.fromEnvironment(
  'API_URL', defaultValue: 'http://10.0.2.2:8080/api/v1');

final dioProvider = Provider<Dio>((ref) {
  const storage = FlutterSecureStorage();
  final dio = Dio(BaseOptions(
    baseUrl: _baseUrl,
    connectTimeout: const Duration(seconds: 15),
    receiveTimeout: const Duration(seconds: 30),
    headers: {'Content-Type': 'application/json'},
  ));
  dio.interceptors.add(InterceptorsWrapper(
    onRequest: (opts, handler) async {
      final token = await storage.read(key: 'nexus_token');
      if (token != null) opts.headers['Authorization'] = 'Bearer \$token';
      handler.next(opts);
    },
    onError: (err, handler) {
      if (err.response?.statusCode == 401) storage.deleteAll();
      handler.next(err);
    },
  ));
  return dio;
});

class ApiException implements Exception {
  final String message;
  final int? statusCode;
  ApiException(this.message, {this.statusCode});
  @override String toString() => message;
}

extension ApiCall on Dio {
  Future<T> apiGet<T>(String path, {Map<String, dynamic>? query}) async {
    try {
      final resp = await get<T>(path, queryParameters: query);
      return resp.data as T;
    } on DioException catch (e) {
      throw ApiException(
        (e.response?.data as Map?)?['error'] ?? e.message ?? 'Request failed',
        statusCode: e.response?.statusCode);
    }
  }
  Future<T> apiPost<T>(String path, {Object? data}) async {
    try {
      final resp = await post<T>(path, data: data);
      return resp.data as T;
    } on DioException catch (e) {
      throw ApiException(
        (e.response?.data as Map?)?['error'] ?? e.message ?? 'Request failed',
        statusCode: e.response?.statusCode);
    }
  }
}
// ─── Auth API ─────────────────────────────────────────────────────────────────
class AuthApi {
  final Dio _dio;
  const AuthApi(this._dio);

  Future<void> sendOtp(String phone) =>
      _dio.apiPost('/auth/otp/send', data: {'phone': phone, 'purpose': 'login'});

  Future<Map<String, dynamic>> verifyOtp(String phone, String code) async {
    final resp = await _dio.apiPost<Map>('/auth/otp/verify',
        data: {'phone': phone, 'code': code});
    return Map<String, dynamic>.from(resp as Map);
  }
}

final authApiProvider = Provider((ref) => AuthApi(ref.read(dioProvider)));

// ─── User / Wallet API ────────────────────────────────────────────────────────
class UserApi {
  final Dio _dio;
  const UserApi(this._dio);

  Future<Map<String, dynamic>> getProfile() async {
    final r = await _dio.apiGet<Map>('/user/profile');
    return Map<String, dynamic>.from(r as Map);
  }

  Future<Map<String, dynamic>> getWallet() async {
    final r = await _dio.apiGet<Map>('/user/wallet');
    return Map<String, dynamic>.from(r as Map);
  }

  Future<List<dynamic>> getTransactions({int page = 1}) async {
    final r = await _dio.apiGet<Map>('/user/transactions', query: {'page': page});
    return (r as Map)['transactions'] as List? ?? [];
  }

  Future<Map<String, dynamic>> getPassport() async {
    final r = await _dio.apiGet<Map>('/user/passport');
    return Map<String, dynamic>.from(r as Map);
  }

  Future<void> requestMoMoLink(String msisdn) =>
      _dio.apiPost('/user/momo/request', data: {'msisdn': msisdn});

  Future<void> verifyMoMo(String code) =>
      _dio.apiPost('/user/momo/verify', data: {'code': code});
}

final userApiProvider = Provider((ref) => UserApi(ref.read(dioProvider)));

// ─── Spin API ─────────────────────────────────────────────────────────────────
class SpinApi {
  final Dio _dio;
  const SpinApi(this._dio);

  Future<Map<String, dynamic>> getWheelConfig() async {
    final r = await _dio.apiGet<Map>('/spin/wheel');
    return Map<String, dynamic>.from(r as Map);
  }

  Future<Map<String, dynamic>> play() async {
    final r = await _dio.apiPost<Map>('/spin/play');
    return Map<String, dynamic>.from(r as Map);
  }

  Future<List<dynamic>> getHistory({int page = 1}) async {
    final r = await _dio.apiGet<Map>('/spin/history', query: {'page': page});
    return (r as Map)['results'] as List? ?? [];
  }
}

final spinApiProvider = Provider((ref) => SpinApi(ref.read(dioProvider)));

// ─── Studio API ───────────────────────────────────────────────────────────────
class StudioApi {
  final Dio _dio;
  const StudioApi(this._dio);

  Future<List<dynamic>> listTools() async {
    final r = await _dio.apiGet<Map>('/studio/tools');
    return (r as Map)['tools'] as List? ?? [];
  }

  Future<Map<String, dynamic>> startGeneration(
      String toolSlug, Map<String, dynamic> params) async {
    final r = await _dio.apiPost<Map>('/studio/generate',
        data: {'tool': toolSlug, 'params': params});
    return Map<String, dynamic>.from(r as Map);
  }

  Future<Map<String, dynamic>> getGenerationStatus(String genId) async {
    final r = await _dio.apiGet<Map>('/studio/generate/$genId/status');
    return Map<String, dynamic>.from(r as Map);
  }

  Future<List<dynamic>> getGallery({int page = 1}) async {
    final r = await _dio.apiGet<Map>('/studio/gallery', query: {'page': page});
    return (r as Map)['generations'] as List? ?? [];
  }

  Future<Map<String, dynamic>> sendChat(String message, String? sessionId) async {
    final r = await _dio.apiPost<Map>('/studio/chat',
        data: {'message': message, if (sessionId != null) 'session_id': sessionId});
    return Map<String, dynamic>.from(r as Map);
  }
}

final studioApiProvider = Provider((ref) => StudioApi(ref.read(dioProvider)));

// ─── Draws API ────────────────────────────────────────────────────────────────
class DrawsApi {
  final Dio _dio;
  const DrawsApi(this._dio);

  Future<List<dynamic>> listUpcoming() async {
    final r = await _dio.apiGet<Map>('/draws');
    return (r as Map)['draws'] as List? ?? [];
  }

  Future<List<dynamic>> getWinners(String drawId) async {
    final r = await _dio.apiGet<Map>('/draws/$drawId/winners');
    return (r as Map)['winners'] as List? ?? [];
  }
}

final drawsApiProvider = Provider((ref) => DrawsApi(ref.read(dioProvider)));

// ─── Regional Wars API ────────────────────────────────────────────────────────
class WarsApi {
  final Dio _dio;
  const WarsApi(this._dio);

  Future<List<dynamic>> getLeaderboard() async {
    final r = await _dio.apiGet<Map>('/wars/leaderboard');
    return (r as Map)['leaderboard'] as List? ?? [];
  }

  Future<Map<String, dynamic>> getMyRank() async {
    final r = await _dio.apiGet<Map>('/wars/my-rank');
    return Map<String, dynamic>.from(r as Map);
  }
}

final warsApiProvider = Provider((ref) => WarsApi(ref.read(dioProvider)));

// ─── Notifications API ────────────────────────────────────────────────────────
class NotificationsApi {
  final Dio _dio;
  const NotificationsApi(this._dio);

  Future<Map<String, dynamic>> list({String? cursor}) async {
    final r = await _dio.apiGet<Map>('/notifications',
        query: cursor != null ? {'cursor': cursor} : null);
    return Map<String, dynamic>.from(r as Map);
  }

  Future<void> markRead(String id) =>
      _dio.apiPost('/notifications/$id/read');

  Future<void> markAllRead() =>
      _dio.apiPost('/notifications/read-all');

  Future<void> registerPushToken(String token, String platform) =>
      _dio.apiPost('/notifications/push-token',
          data: {'token': token, 'platform': platform});
}

final notificationsApiProvider =
    Provider((ref) => NotificationsApi(ref.read(dioProvider)));
