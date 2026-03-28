import 'dart:io';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../api/api_client.dart';

// ─── Background handler (top-level — required by FCM) ────────────────────────
@pragma('vm:entry-point')
Future<void> firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  // Firebase is already initialised in main.dart before this runs.
  debugPrint('[FCM] Background message: ${message.messageId}');
  // No navigation here — app may not be running.
}

// ─── Channel config ───────────────────────────────────────────────────────────

const _androidChannel = AndroidNotificationChannel(
  'nexus_high_importance',            // id
  'Loyalty Nexus Alerts',             // name
  description: 'Spin results, draws, points and war updates',
  importance: Importance.high,
  enableVibration: true,
  playSound: true,
);

// ─── Service ──────────────────────────────────────────────────────────────────

class PushNotificationService {
  final ProviderContainer _container;
  final GoRouter _router;

  final _localNotif = FlutterLocalNotificationsPlugin();
  bool _initialised = false;

  PushNotificationService({
    required ProviderContainer container,
    required GoRouter router,
  })  : _container = container,
        _router    = router;

  // ── Init ────────────────────────────────────────────────────────────────────

  Future<void> init() async {
    if (_initialised) return;
    _initialised = true;

    // 1 — Register background handler
    FirebaseMessaging.onBackgroundMessage(firebaseMessagingBackgroundHandler);

    // 2 — Request permission (iOS / Android 13+)
    await _requestPermission();

    // 3 — Set up local notifications plugin (for foreground display)
    await _setupLocalNotifications();

    // 4 — FCM token
    await _registerToken();

    // 5 — Foreground messages
    FirebaseMessaging.onMessage.listen(_handleForeground);

    // 6 — Background → foreground tap
    FirebaseMessaging.onMessageOpenedApp.listen(_handleTap);

    // 7 — Notification that launched app from terminated state
    final initial = await FirebaseMessaging.instance.getInitialMessage();
    if (initial != null) _handleTap(initial);

    // 8 — Token refresh
    FirebaseMessaging.instance.onTokenRefresh.listen(_onTokenRefresh);

    debugPrint('[FCM] Push notification service initialised');
  }

  // ── Permission ──────────────────────────────────────────────────────────────

  Future<void> _requestPermission() async {
    final settings = await FirebaseMessaging.instance.requestPermission(
      alert:         true,
      announcement:  false,
      badge:         true,
      carPlay:       false,
      criticalAlert: false,
      provisional:   false,
      sound:         true,
    );
    debugPrint('[FCM] Auth status: ${settings.authorizationStatus}');
  }

  // ── Local notification plugin setup ─────────────────────────────────────────

  Future<void> _setupLocalNotifications() async {
    const androidInit = AndroidInitializationSettings('@mipmap/ic_launcher');
    const iosInit     = DarwinInitializationSettings(
      requestAlertPermission: false, // already requested above
      requestBadgePermission: true,
      requestSoundPermission: true,
    );

    await _localNotif.initialize(
      const InitializationSettings(android: androidInit, iOS: iosInit),
      onDidReceiveNotificationResponse: (details) {
        // User tapped a local notification while app was in foreground
        _routeFromPayload(details.payload);
      },
    );

    // Create high-importance channel on Android
    if (Platform.isAndroid) {
      final androidPlugin = _localNotif
          .resolvePlatformSpecificImplementation<
              AndroidFlutterLocalNotificationsPlugin>();
      await androidPlugin?.createNotificationChannel(_androidChannel);
    }
  }

  // ── Token management ────────────────────────────────────────────────────────

  Future<void> _registerToken() async {
    try {
      // For iOS, get APNS token first
      if (Platform.isIOS) {
        await FirebaseMessaging.instance.getAPNSToken();
      }
      final token = await FirebaseMessaging.instance.getToken();
      if (token != null) {
        await _uploadToken(token);
      }
    } catch (e) {
      debugPrint('[FCM] Token fetch failed: $e');
    }
  }

  Future<void> _onTokenRefresh(String token) async {
    debugPrint('[FCM] Token refreshed');
    await _uploadToken(token);
  }

  Future<void> _uploadToken(String token) async {
    try {
      final platform = Platform.isAndroid ? 'android' : 'ios';
      await _container.read(notificationsApiProvider)
          .registerPushToken(token, platform);
      debugPrint('[FCM] Token registered: ${token.substring(0, 10)}…');
    } catch (e) {
      debugPrint('[FCM] Token upload failed: $e');
    }
  }

  // ── Message handlers ────────────────────────────────────────────────────────

  void _handleForeground(RemoteMessage message) {
    debugPrint('[FCM] Foreground: ${message.notification?.title}');
    final notif = message.notification;
    if (notif == null) return;

    // Show a local notification for foreground messages
    _localNotif.show(
      message.hashCode,
      notif.title,
      notif.body,
      NotificationDetails(
        android: AndroidNotificationDetails(
          _androidChannel.id,
          _androidChannel.name,
          channelDescription: _androidChannel.description,
          importance: Importance.high,
          priority: Priority.high,
          icon: '@mipmap/ic_launcher',
        ),
        iOS: const DarwinNotificationDetails(
          presentAlert: true,
          presentBadge: true,
          presentSound: true,
        ),
      ),
      payload: _buildPayload(message.data),
    );
  }

  void _handleTap(RemoteMessage message) {
    debugPrint('[FCM] Tapped: ${message.data}');
    _routeFromPayload(_buildPayload(message.data));
  }

  // ── Deep-link routing ───────────────────────────────────────────────────────

  String _buildPayload(Map<String, dynamic> data) {
    // Backend sends 'route' or 'type' in FCM data payload
    return data['route'] as String? ?? _typeToRoute(data['type'] as String?);
  }

  static String _typeToRoute(String? type) {
    switch (type) {
      case 'spin_result':     return '/spin/prizes';
      case 'draw_winner':
      case 'draw_result':     return '/draws';
      case 'point_credit':
      case 'bonus_award':     return '/pulse-awards';
      case 'war_update':
      case 'war_rank':        return '/wars';
      case 'passport_update': return '/passport';
      case 'prize_pending':   return '/prizes';
      default:                return '/notifications';
    }
  }

  void _routeFromPayload(String? payload) {
    if (payload == null || payload.isEmpty) return;
    // Wait one frame so the router context is ready
    Future.microtask(() {
      try {
        _router.push(payload);
      } catch (e) {
        debugPrint('[FCM] Route error: $e');
      }
    });
  }
}

// ─── Provider ─────────────────────────────────────────────────────────────────

final pushNotificationServiceProvider = Provider<PushNotificationService>((ref) {
  throw UnimplementedError(
      'pushNotificationServiceProvider must be overridden in main.dart '
      'with a ProviderContainer and GoRouter instance');
});
