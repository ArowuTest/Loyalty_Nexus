import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:mime/mime.dart';

// ─── Base URL ─────────────────────────────────────────────────────────────────
// Inject via: flutter run --dart-define=API_URL=https://loyalty-nexus-api.onrender.com/api/v1
const _baseUrl = String.fromEnvironment(
  'API_URL', defaultValue: 'https://loyalty-nexus-api.onrender.com/api/v1');

// ─── Dio Provider ─────────────────────────────────────────────────────────────

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
      if (token != null) opts.headers['Authorization'] = 'Bearer $token';
      handler.next(opts);
    },
    onError: (err, handler) {
      if (err.response?.statusCode == 401) storage.deleteAll();
      handler.next(err);
    },
  ));
  return dio;
});

// ─── Exception ────────────────────────────────────────────────────────────────

class ApiException implements Exception {
  final String message;
  final int? statusCode;
  ApiException(this.message, {this.statusCode});
  @override String toString() => message;
}

// ─── Extension helpers ────────────────────────────────────────────────────────

extension ApiCall on Dio {
  Future<T> apiGet<T>(String path, {Map<String, dynamic>? query}) async {
    try {
      final resp = await get<T>(path, queryParameters: query);
      return resp.data as T;
    } on DioException catch (e) {
      final errMsg = _extractError(e);
      throw ApiException(errMsg, statusCode: e.response?.statusCode);
    }
  }

  Future<T> apiPost<T>(String path, {Object? data}) async {
    try {
      final resp = await post<T>(path, data: data);
      return resp.data as T;
    } on DioException catch (e) {
      final errMsg = _extractError(e);
      throw ApiException(errMsg, statusCode: e.response?.statusCode);
    }
  }

  Future<T> apiPatch<T>(String path, {Object? data}) async {
    try {
      final resp = await patch<T>(path, data: data);
      return resp.data as T;
    } on DioException catch (e) {
      final errMsg = _extractError(e);
      throw ApiException(errMsg, statusCode: e.response?.statusCode);
    }
  }

  static String _extractError(DioException e) {
    final body = e.response?.data;
    if (body is Map) {
      return (body['error'] ?? body['message'] ?? e.message ?? 'Request failed').toString();
    }
    return e.message ?? 'Request failed';
  }
}

// ─── Auth API ─────────────────────────────────────────────────────────────────

class AuthApi {
  final Dio _dio;
  const AuthApi(this._dio);

  Future<void> sendOtp(String phone) =>
      _dio.apiPost('/auth/otp/send', data: {'phone_number': phone, 'purpose': 'login'});

  Future<Map<String, dynamic>> verifyOtp(String phone, String code) async {
    final r = await _dio.apiPost<Map>('/auth/otp/verify',
        data: {'phone_number': phone, 'code': code, 'purpose': 'login'});
    return Map<String, dynamic>.from(r as Map);
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

  Future<List<dynamic>> getTransactions({int page = 1, int limit = 20}) async {
    final r = await _dio.apiGet<Map>('/user/transactions',
        query: {'page': page, 'limit': limit});
    return (r as Map)['transactions'] as List? ?? [];
  }

  Future<Map<String, dynamic>> getPassport() async {
    final r = await _dio.apiGet<Map>('/user/passport');
    return Map<String, dynamic>.from(r as Map);
  }

  /// Generate a QR code payload for the passport (5-min TTL)
  Future<Map<String, dynamic>> getPassportQR() async {
    final r = await _dio.apiGet<Map>('/passport/qr');
    return Map<String, dynamic>.from(r as Map);
  }

  /// Passport activity event log (tier upgrades, badge earns, QR scans)
  Future<List<dynamic>> getPassportEvents({int limit = 30}) async {
    try {
      final r = await _dio.apiGet<Map>('/passport/events', query: {'limit': limit});
      return (r as Map)['events'] as List? ?? [];
    } catch (_) { return []; }
  }

  Future<Map<String, dynamic>> getBonusPulseAwards() async {
    final r = await _dio.apiGet<Map>('/user/bonus-pulse');
    return Map<String, dynamic>.from(r as Map);
  }

  /// Update profile fields — only non-null fields are sent
  Future<Map<String, dynamic>> updateProfile({String? fullName, String? state}) async {
    final payload = <String, dynamic>{};
    if (fullName != null) payload['full_name'] = fullName;
    if (state != null)    payload['state'] = state;
    final r = await _dio.apiPatch<Map>('/user/profile', data: payload);
    return Map<String, dynamic>.from(r as Map);
  }

  /// Link MoMo wallet (initiates verification)
  Future<Map<String, dynamic>> requestMoMoLink(String msisdn) async {
    final r = await _dio.apiPost<Map>('/user/momo/request',
        data: {'msisdn': msisdn});
    return Map<String, dynamic>.from(r as Map);
  }

  Future<void> verifyMoMo(String code) =>
      _dio.apiPost('/user/momo/verify', data: {'code': code});

  /// Get current notification preferences from backend
  Future<Map<String, dynamic>> getNotificationPrefs() async {
    try {
      final r = await _dio.apiGet<Map>('/user/notification-preferences');
      return Map<String, dynamic>.from(r as Map);
    } catch (_) {
      return {'preferences': <String, dynamic>{}};
    }
  }

  /// Save notification preferences to backend
  Future<void> updateNotificationPrefs(Map<String, bool> prefs) =>
      _dio.apiPost('/user/notification-preferences', data: {'preferences': prefs});
}

final userApiProvider = Provider((ref) => UserApi(ref.read(dioProvider)));

// ─── Spin API ─────────────────────────────────────────────────────────────────

class SpinApi {
  final Dio _dio;
  const SpinApi(this._dio);

  /// Daily spin eligibility — credits available, recharge threshold, progress
  Future<Map<String, dynamic>> getEligibility() async {
    try {
      final r = await _dio.apiGet<Map>('/spin/eligibility');
      return Map<String, dynamic>.from(r as Map);
    } catch (_) { return {}; }
  }

  /// Returns wheel segments, cost, spin limit status — admin-configurable
  Future<Map<String, dynamic>> getWheelConfig() async {
    final r = await _dio.apiGet<Map>('/spin/wheel');
    return Map<String, dynamic>.from(r as Map);
  }

  /// Play one spin — returns result segment
  Future<Map<String, dynamic>> play() async {
    final r = await _dio.apiPost<Map>('/spin/play');
    return Map<String, dynamic>.from(r as Map);
  }

  /// Paginated spin history
  Future<List<dynamic>> getHistory({int page = 1}) async {
    final r = await _dio.apiGet<Map>('/spin/history', query: {'page': page});
    return (r as Map)['results'] as List? ?? [];
  }

  /// All wins for the authenticated user (prizes screen)
  Future<List<dynamic>> getMyWins() async {
    final r = await _dio.apiGet<Map>('/spin/wins');
    return (r as Map)['wins'] as List? ?? [];
  }

  /// Claim a prize; MoMo cash needs {momo_number}
  Future<Map<String, dynamic>> claimPrize(
      String winId, Map<String, dynamic> payload) async {
    final r = await _dio.apiPost<Map>('/spin/wins/$winId/claim', data: payload);
    return Map<String, dynamic>.from(r as Map);
  }
}

final spinApiProvider = Provider((ref) => SpinApi(ref.read(dioProvider)));

// ─── Studio API ───────────────────────────────────────────────────────────────

class StudioApi {
  final Dio _dio;
  const StudioApi(this._dio);

  /// Upload a file (image, video, audio, document) to the studio CDN.
  /// Returns the CDN URL string to pass as image_url / document_url etc.
  Future<String> uploadAsset(File file) async {
    final mimeType = lookupMimeType(file.path) ?? 'application/octet-stream';
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(
        file.path,
        filename: file.path.split('/').last,
        contentType: DioMediaType.parse(mimeType),
      ),
    });
    try {
      final res = await _dio.post<Map>('/studio/upload', data: formData);
      final data = res.data as Map;
      return data['url'] as String;
    } on DioException catch (e) {
      final msg = (e.response?.data as Map?)?['error'] as String? ?? 'Upload failed';
      throw ApiException(msg, statusCode: e.response?.statusCode);
    }
  }

  /// Full tool list — includes ui_template and ui_config for each tool
  /// so the mobile app never hardcodes tool behaviour — admin configures it.
  Future<List<dynamic>> listTools() async {
    final r = await _dio.apiGet<Map>('/studio/tools');
    return (r as Map)['tools'] as List? ?? [];
  }

  /// Categories from server (admin-managed)
  Future<List<dynamic>> listCategories() async {
    try {
      final r = await _dio.apiGet<Map>('/studio/categories');
      return (r as Map)['categories'] as List? ?? [];
    } catch (_) { return []; }
  }

  /// Start an async generation job
  Future<Map<String, dynamic>> startGeneration(
      String toolSlug, Map<String, dynamic> params) async {
    final r = await _dio.apiPost<Map>('/studio/generate',
        data: {'tool': toolSlug, 'params': params});
    return Map<String, dynamic>.from(r as Map);
  }

  /// Poll generation status (pending → processing → completed/failed)
  Future<Map<String, dynamic>> getGenerationStatus(String genId) async {
    final r = await _dio.apiGet<Map>('/studio/generate/$genId/status');
    return Map<String, dynamic>.from(r as Map);
  }

  /// Delete a gallery item
  Future<void> deleteGeneration(String genId) =>
      _dio.apiPost('/studio/generate/$genId/delete');

  /// User's gallery with pagination + optional tool filter
  Future<List<dynamic>> getGallery({int page = 1, String? toolSlug}) async {
    final r = await _dio.apiGet<Map>('/studio/gallery', query: {
      'page': page,
      if (toolSlug != null) 'tool': toolSlug,
    });
    return (r as Map)['generations'] as List? ?? [];
  }

  /// Chat with AI (session-aware, supports tool-scoped context)
  Future<Map<String, dynamic>> sendChat(
      String message, String? toolSlug, {String? sessionId}) async {
    final r = await _dio.apiPost<Map>('/studio/chat', data: {
      'message': message,
      if (sessionId != null) 'session_id': sessionId,
      if (toolSlug != null)  'tool_slug':  toolSlug,
    });
    return Map<String, dynamic>.from(r as Map);
  }

  /// Chat history for a given mode (general / web-search-ai / code-helper)
  Future<List<dynamic>> getChatHistory(String mode) async {
    try {
      final r = await _dio.apiGet<Map>('/studio/chat/history', query: {'mode': mode});
      return (r as Map)['messages'] as List? ?? [];
    } catch (_) { return []; }
  }

  /// Chat usage / quota (messages remaining, resets, plan)
  Future<Map<String, dynamic>> getChatUsage() async {
    try {
      final r = await _dio.apiGet<Map>('/studio/chat/usage');
      return Map<String, dynamic>.from(r as Map);
    } catch (_) { return {}; }
  }

  /// Current session usage (points spent, generations count, session start)
  Future<Map<String, dynamic>> getSessionUsage() async {
    try {
      final r = await _dio.apiGet<Map>('/studio/session');
      return Map<String, dynamic>.from(r as Map);
    } catch (_) { return {}; }
  }
}

final studioApiProvider = Provider((ref) => StudioApi(ref.read(dioProvider)));

// ─── Draws API ────────────────────────────────────────────────────────────────

class DrawsApi {
  final Dio _dio;
  const DrawsApi(this._dio);

  /// All draws — active + completed — from backend
  Future<List<dynamic>> getDraws() async {
    final r = await _dio.apiGet<Map>('/draws');
    return (r as Map)['draws'] as List? ?? [];
  }

  /// Upcoming draws only (legacy alias kept for dashboard)
  Future<List<dynamic>> listUpcoming() => getDraws();

  /// Winners for a specific draw
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

  /// Full state leaderboard for the active war period
  Future<List<dynamic>> getLeaderboard() async {
    final r = await _dio.apiGet<Map>('/wars/leaderboard');
    return (r as Map)['leaderboard'] as List? ?? [];
  }

  /// Authenticated user's personal rank + entry in current war
  Future<Map<String, dynamic>> getMyRank() async {
    final r = await _dio.apiGet<Map>('/wars/my-rank');
    return Map<String, dynamic>.from(r as Map);
  }

  /// Winners for a past war period
  Future<List<dynamic>> getWarWinners({String? period}) async {
    final r = await _dio.apiGet<Map>(
        period != null ? '/wars/$period/winners' : '/wars/winners');
    return (r as Map)['winners'] as List? ?? [];
  }

  /// Past war periods with winners
  Future<List<dynamic>> getHistory({int limit = 12}) async {
    try {
      final r = await _dio.apiGet<Map>('/wars/history', query: {'limit': limit});
      return (r as Map)['history'] as List? ?? [];
    } catch (_) { return []; }
  }

  /// Full war config — period, prize pool, rules (admin-set)
  Future<Map<String, dynamic>> getWarConfig() async {
    try {
      final r = await _dio.apiGet<Map>('/wars/config');
      return Map<String, dynamic>.from(r as Map);
    } catch (_) { return {}; }
  }
}

final warsApiProvider = Provider((ref) => WarsApi(ref.read(dioProvider)));

// ─── Notifications API ────────────────────────────────────────────────────────

class NotificationsApi {
  final Dio _dio;
  const NotificationsApi(this._dio);

  /// Paginated notification list with optional cursor
  Future<Map<String, dynamic>> list({String? cursor, int limit = 30}) async {
    final r = await _dio.apiGet<Map>('/notifications', query: {
      'limit': limit,
      if (cursor != null) 'cursor': cursor,
    });
    return Map<String, dynamic>.from(r as Map);
  }

  Future<void> markRead(String id) =>
      _dio.apiPost('/notifications/$id/read');

  Future<void> markAllRead() =>
      _dio.apiPost('/notifications/read-all');

  /// Register FCM / APNs push token with backend
  Future<void> registerPushToken(String token, String platform) =>
      _dio.apiPost('/notifications/push-token',
          data: {'token': token, 'platform': platform});
}

final notificationsApiProvider =
    Provider((ref) => NotificationsApi(ref.read(dioProvider)));

// ─── Passport API ─────────────────────────────────────────────────────────────
class PassportApi {
  final Dio _dio;
  const PassportApi(this._dio);

  /// Full passport profile: tier, badges, streak, lifetime points, wallet URLs
  Future<Map<String, dynamic>> getPassport() async {
    final r = await _dio.apiGet<Map>('/passport');
    return Map<String, dynamic>.from(r as Map);
  }

  /// HMAC-signed QR payload string (expires in 5 minutes)
  Future<Map<String, dynamic>> getPassportQR() async {
    final r = await _dio.apiGet<Map>('/passport/qr');
    return Map<String, dynamic>.from(r as Map);
  }

  /// Passport activity event log (tier upgrades, badge earns, QR scans, streaks)
  Future<List<dynamic>> getPassportEvents({int limit = 30}) async {
    try {
      final r = await _dio.apiGet<Map>('/passport/events', query: {'limit': limit});
      return (r as Map)['events'] as List? ?? [];
    } catch (_) { return []; }
  }

  /// Returns both the Apple .pkpass download URL and the Google Wallet save URL.
  /// apple_pkpass_url  → direct download; iOS intercepts and opens Wallet
  /// google_wallet_url → "Add to Google Wallet" deep-link
  Future<Map<String, dynamic>> getWalletPassURLs() async {
    final r = await _dio.apiGet<Map>('/passport/wallet-urls');
    return Map<String, dynamic>.from(r as Map);
  }

  /// Constructs the direct Apple .pkpass download URL including the Bearer token
  /// so iOS can authenticate the download without a separate header.
  /// The token is appended as a query param because iOS Wallet does not send
  /// custom headers when downloading a .pkpass file.
  Future<String> getApplePKPassURL() async {
    const storage = FlutterSecureStorage();
    final token = await storage.read(key: 'nexus_token');
    final base = _baseUrl.replaceFirst('/api/v1', '');
    return token != null
        ? '$base/api/v1/passport/pkpass?token=$token'
        : '$base/api/v1/passport/pkpass';
  }
}

final passportApiProvider =
    Provider((ref) => PassportApi(ref.read(dioProvider)));
