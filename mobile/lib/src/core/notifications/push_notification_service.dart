import 'dart:developer';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../api/api_client.dart';

/// Background message handler — must be a top-level function
@pragma('vm:entry-point')
Future<void> _firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  await Firebase.initializeApp();
  log('[FCM] Background message: ${message.messageId}');
}

/// Singleton push notification service
class PushNotificationService {
  PushNotificationService._();
  static final instance = PushNotificationService._();

  final _fcm = FirebaseMessaging.instance;
  final _localNotifs = FlutterLocalNotificationsPlugin();

  static const _androidChannel = AndroidNotificationChannel(
    'nexus_high_importance',
    'Loyalty Nexus',
    description: 'Spin wins, draw results, and important updates',
    importance: Importance.max,
    enableVibration: true,
    playSound: true,
  );

  /// Call once from main() after Firebase.initializeApp()
  Future<void> initialize(WidgetRef ref) async {
    // Register background handler
    FirebaseMessaging.onBackgroundMessage(_firebaseMessagingBackgroundHandler);

    // Request permissions (iOS/Android 13+)
    await _fcm.requestPermission(
      alert: true, badge: true, sound: true,
      provisional: false,
    );

    // Android notification channel
    await _localNotifs
        .resolvePlatformSpecificImplementation<AndroidFlutterLocalNotificationsPlugin>()
        ?.createNotificationChannel(_androidChannel);

    // Init local notifications
    const initSettings = InitializationSettings(
      android: AndroidInitializationSettings('@mipmap/ic_launcher'),
      iOS: DarwinInitializationSettings(),
    );
    await _localNotifs.initialize(initSettings,
        onDidReceiveNotificationResponse: (details) {
      _handleDeepLink(details.payload);
    });

    // Foreground messages → show local notification
    FirebaseMessaging.onMessage.listen((RemoteMessage message) {
      final notification = message.notification;
      final android = message.notification?.android;
      if (notification != null && android != null) {
        _localNotifs.show(
          notification.hashCode,
          notification.title,
          notification.body,
          NotificationDetails(
            android: AndroidNotificationDetails(
              _androidChannel.id, _androidChannel.name,
              channelDescription: _androidChannel.description,
              importance: Importance.max, priority: Priority.high,
              icon: '@mipmap/ic_launcher',
            ),
          ),
          payload: message.data['deep_link'],
        );
      }
    });

    // Background message opened app
    FirebaseMessaging.onMessageOpenedApp.listen((message) {
      _handleDeepLink(message.data['deep_link']);
    });

    // App opened from terminated via notification
    final initialMessage = await _fcm.getInitialMessage();
    if (initialMessage != null) {
      _handleDeepLink(initialMessage.data['deep_link']);
    }

    // Register token with backend
    await _registerToken(ref);

    // Listen for token refresh
    _fcm.onTokenRefresh.listen((_) => _registerToken(ref));
  }

  Future<void> _registerToken(WidgetRef ref) async {
    try {
      final token = await _fcm.getToken();
      if (token == null) return;
      log('[FCM] Token: $token');
      await ref.read(notificationsApiProvider).registerPushToken(token, 'android');
    } catch (e) {
      log('[FCM] Token registration failed: $e');
    }
  }

  void _handleDeepLink(String? deepLink) {
    if (deepLink == null || deepLink.isEmpty) return;
    // Deep link routing handled by app router — store for later consumption
    log('[FCM] Deep link: $deepLink');
  }

  /// Get the current FCM token (for debugging)
  Future<String?> getToken() => _fcm.getToken();
}

final pushNotificationServiceProvider = Provider((_) => PushNotificationService.instance);
