import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import 'recharge_models.dart';

// ─── Networks provider ────────────────────────────────────────────────────────

final rechargeNetworksProvider =
    FutureProvider.autoDispose<List<NetworkOperator>>((ref) async {
  final dio = ref.read(dioProvider);
  final raw = await dio.apiGet<List<dynamic>>('/recharge/networks');
  return raw
      .map((e) => NetworkOperator.fromJson(e as Map<String, dynamic>))
      .toList();
});

// ─── Bundles provider (keyed by network code) ─────────────────────────────────

final rechargeBundlesProvider = FutureProvider.autoDispose
    .family<List<DataBundle>, String>((ref, networkCode) async {
  final dio = ref.read(dioProvider);
  final raw =
      await dio.apiGet<List<dynamic>>('/recharge/networks/$networkCode/bundles');
  return raw
      .map((e) => DataBundle.fromJson(e as Map<String, dynamic>))
      .toList();
});

// ─── Recharge API service ─────────────────────────────────────────────────────

final rechargeApiProvider = Provider<RechargeApi>((ref) {
  return RechargeApi(ref.read(dioProvider));
});

class RechargeApi {
  final Dio _dio;
  RechargeApi(this._dio);

  Future<InitiateRechargeResponse> initiateAirtime({
    required String phone,
    required String networkCode,
    required int amount,
    String? userId,
  }) async {
    final body = <String, dynamic>{
      'phone':        phone,
      'network_code': networkCode,
      'amount':       amount,
      'type':         'airtime',
      if (userId != null) 'user_id': userId,
    };
    final raw = await _dio.apiPost<Map<String, dynamic>>(
      '/recharge/initiate',
      data: body,
    );
    return InitiateRechargeResponse.fromJson(raw);
  }

  Future<InitiateRechargeResponse> initiateData({
    required String phone,
    required String networkCode,
    required String variationCode,
    required int amount,
    String? userId,
  }) async {
    final body = <String, dynamic>{
      'phone':          phone,
      'network_code':   networkCode,
      'amount':         amount,
      'type':           'data',
      'variation_code': variationCode,
      if (userId != null) 'user_id': userId,
    };
    final raw = await _dio.apiPost<Map<String, dynamic>>(
      '/recharge/initiate',
      data: body,
    );
    return InitiateRechargeResponse.fromJson(raw);
  }
}
