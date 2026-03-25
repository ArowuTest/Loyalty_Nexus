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