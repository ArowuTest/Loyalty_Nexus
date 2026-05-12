// ─── Recharge Domain Models ───────────────────────────────────────────────────

class NetworkOperator {
  final String code;
  final String name;
  final bool isActive;
  final bool airtimeEnabled;
  final bool dataEnabled;

  const NetworkOperator({
    required this.code,
    required this.name,
    required this.isActive,
    required this.airtimeEnabled,
    required this.dataEnabled,
  });

  factory NetworkOperator.fromJson(Map<String, dynamic> j) => NetworkOperator(
        code:           j['code']            as String? ?? '',
        name:           j['name']            as String? ?? '',
        isActive:       j['is_active']       as bool?   ?? false,
        airtimeEnabled: j['airtime_enabled'] as bool?   ?? false,
        dataEnabled:    j['data_enabled']    as bool?   ?? false,
      );
}

class DataBundle {
  final String id; // variation_code
  final String name;
  final int price;
  final String validity;

  const DataBundle({
    required this.id,
    required this.name,
    required this.price,
    required this.validity,
  });

  factory DataBundle.fromJson(Map<String, dynamic> j) => DataBundle(
        id:       j['id']       as String? ?? '',
        name:     j['name']     as String? ?? '',
        price:    (j['price']   as num?)?.toInt() ?? 0,
        validity: j['validity'] as String? ?? '',
      );
}

class InitiateRechargeResponse {
  final String paymentUrl;
  final String reference;

  const InitiateRechargeResponse({
    required this.paymentUrl,
    required this.reference,
  });

  factory InitiateRechargeResponse.fromJson(Map<String, dynamic> j) =>
      InitiateRechargeResponse(
        paymentUrl: j['payment_url'] as String? ?? '',
        reference:  j['reference']  as String? ?? '',
      );
}
