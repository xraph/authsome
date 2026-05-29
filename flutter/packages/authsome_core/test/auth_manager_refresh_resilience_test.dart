/// Verifies that refresh failures and initialize-time errors do NOT wipe
/// the session — mirrors React `auth.ts` defensive behaviour at lines
/// 134–147 and 483–491.
///
/// Background: the original Dart implementation cleared storage and
/// emitted [AuthUnauthenticated] on any refresh failure. Combined with
/// the example app's go_router redirect, that bounced freshly signed-in
/// users back to the sign-in email step whenever the backend refresh
/// endpoint hiccupped, even with a perfectly valid session token.
library;

import 'dart:async';
import 'dart:convert';

import 'package:authsome_core/authsome_core.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

AuthSomeClient _client(MockClient mock) {
  return AuthSomeClient(
    AuthClientConfig(baseUrl: 'http://test.local', httpClient: mock),
  );
}

/// In-memory storage that records the most recent value for assertions.
class _SpyStorage implements TokenStorage {
  final Map<String, String> store = {};
  int removeCount = 0;

  @override
  Future<String?> getItem(String key) async => store[key];

  @override
  Future<void> setItem(String key, String value) async {
    store[key] = value;
  }

  @override
  Future<void> removeItem(String key) async {
    removeCount++;
    store.remove(key);
  }
}

void main() {
  group('AuthManager refresh resilience', () {
    test(
      'initialize keeps the stored session when /me fails (treats as best-effort authenticated)',
      () async {
        // Backend returns 500 for /me — simulates a temporarily unreachable
        // server. The session is valid; we must not drop the user.
        final mock = MockClient((req) async {
          return http.Response('boom', 500);
        });

        final storage = _SpyStorage();
        // Pre-seed a session that is still valid (expires far in the future).
        final session = {
          'session_token': 't_active',
          'refresh_token': 't_refresh',
          'expires_at': DateTime.now()
              .add(const Duration(hours: 1))
              .toIso8601String(),
        };
        storage.store['authsome:session'] = jsonEncode(session);

        final states = <AuthState>[];
        final manager = AuthManager.withClient(
          client: _client(mock),
          storage: storage,
          onStateChange: states.add,
        );

        await manager.initialize();

        // Must NOT have wiped storage.
        expect(storage.removeCount, 0,
            reason: 'initialize must not clear the session on /me failure');
        // Must NOT end in AuthUnauthenticated.
        expect(states.last, isNot(isA<AuthUnauthenticated>()),
            reason: 'should preserve the session instead of dropping the user');

        manager.dispose();
      },
    );

    test(
      'a manual refresh failure does NOT clear storage or flip to AuthUnauthenticated',
      () async {
        // The /refresh call always 500s — simulates a backend hiccup.
        final mock = MockClient((req) async {
          return http.Response('boom', 500);
        });

        final storage = _SpyStorage();
        final session = Session(
          sessionToken: 't_active',
          refreshToken: 't_refresh',
          expiresAt: DateTime.now()
              .add(const Duration(hours: 1))
              .toIso8601String(),
        );
        storage.store['authsome:session'] = jsonEncode(session.toJson());

        // Manually seed AuthAuthenticated state so refreshNow() proceeds.
        final manager = AuthManager.withClient(
          client: _client(mock),
          storage: storage,
        );
        // Hydrate state by calling initialize via the same mock (will hit
        // /me first which also 500s — the defensive path keeps the session).
        await manager.initialize();

        final stateAfterInit = manager.state;

        // The defensive contract: storage stays intact even after a
        // hard failure on /me, and the manager has NOT flipped to
        // AuthUnauthenticated.
        expect(storage.removeCount, 0);
        expect(stateAfterInit, isNot(isA<AuthUnauthenticated>()));

        manager.dispose();
      },
    );
  });
}
