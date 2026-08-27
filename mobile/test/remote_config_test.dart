// Version-comparison tests for the remote kill switch.
//
// This is the logic that decides whether to LOCK A USER OUT of the app, so the
// failure modes matter more than the happy path. The governing rule is FAIL
// OPEN: anything ambiguous or malformed must resolve to "no constraint", because
// a typo in the Firebase console must never brick every install.

import 'package:flutter_test/flutter_test.dart';
import 'package:loyalty_nexus/src/core/remote_config/remote_config_service.dart';

void main() {
  group('version comparison — ordering', () {
    test('older patch/minor/major are detected', () {
      expect(RemoteConfigService.isOlder('1.0.0', '1.0.1'), isTrue);
      expect(RemoteConfigService.isOlder('1.0.9', '1.1.0'), isTrue);
      expect(RemoteConfigService.isOlder('1.9.9', '2.0.0'), isTrue);
    });

    test('equal versions are NOT older — the boundary that locks people out', () {
      // If this were true, setting minimum_supported_version to the shipped
      // version would block every single user.
      expect(RemoteConfigService.isOlder('1.2.3', '1.2.3'), isFalse);
    });

    test('newer versions are not older', () {
      expect(RemoteConfigService.isOlder('2.0.0', '1.9.9'), isFalse);
      expect(RemoteConfigService.isOlder('1.1.0', '1.0.9'), isFalse);
    });

    test('numeric comparison, not lexicographic', () {
      // '10' < '9' as strings — a string compare would wrongly block v1.10.0.
      expect(RemoteConfigService.isOlder('1.10.0', '1.9.0'), isFalse);
      expect(RemoteConfigService.isOlder('1.9.0', '1.10.0'), isTrue);
      expect(RemoteConfigService.isOlder('2.0.0', '10.0.0'), isTrue);
    });
  });

  group('version comparison — tolerated formats', () {
    test('build metadata after + is ignored', () {
      // pubspec uses "1.0.0+1"; the build number must not affect the gate.
      expect(RemoteConfigService.isOlder('1.0.0+5', '1.0.1'), isTrue);
      expect(RemoteConfigService.isOlder('1.0.0+5', '1.0.0+9'), isFalse);
    });

    test('a leading v is tolerated (git tag style)', () {
      expect(RemoteConfigService.isOlder('v1.0.0', 'v1.0.1'), isTrue);
    });

    test('two-part versions are treated as x.y.0', () {
      expect(RemoteConfigService.isOlder('1.0', '1.0.1'), isTrue);
      expect(RemoteConfigService.isOlder('1.1', '1.0.9'), isFalse);
    });
  });

  group('FAIL OPEN — malformed input must never block a user', () {
    test('empty and junk strings yield no constraint', () {
      expect(RemoteConfigService.isOlder('', '1.0.0'), isFalse);
      expect(RemoteConfigService.isOlder('1.0.0', ''), isFalse);
      expect(RemoteConfigService.isOlder('not-a-version', '1.0.0'), isFalse);
      expect(RemoteConfigService.isOlder('1.0.0', 'latest'), isFalse);
    });

    test('too many segments yields no constraint', () {
      expect(RemoteConfigService.isOlder('1.0.0.0', '2.0.0'), isFalse);
    });

    test('negative and non-numeric segments yield no constraint', () {
      expect(RemoteConfigService.isOlder('1.-2.0', '2.0.0'), isFalse);
      expect(RemoteConfigService.isOlder('1.x.0', '2.0.0'), isFalse);
    });

    test('the permissive default 0.0.0 never blocks anything', () {
      // This is what ships as the in-app default, so it must be inert.
      expect(RemoteConfigService.isOlder('1.0.0', '0.0.0'), isFalse);
      expect(RemoteConfigService.isOlder('0.0.0', '0.0.0'), isFalse);
    });
  });

  group('service defaults before any fetch', () {
    test('an un-initialised service fails open', () {
      // Firebase down, or init() threw: the app must be fully usable.
      final gate = RemoteConfigService.instance.evaluate();
      expect(gate.status, AppGateStatus.ok);
      expect(gate.blocks, isFalse);
      expect(RemoteConfigService.instance.isFeatureKilled('spin'), isFalse);
    });
  });

  group('AppGate.blocks', () {
    test('only updateRequired and maintenance block', () {
      expect(const AppGate(status: AppGateStatus.ok).blocks, isFalse);
      expect(const AppGate(status: AppGateStatus.updateAvailable).blocks, isFalse);
      expect(const AppGate(status: AppGateStatus.updateRequired).blocks, isTrue);
      expect(const AppGate(status: AppGateStatus.maintenance).blocks, isTrue);
    });
  });
}
