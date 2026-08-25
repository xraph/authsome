import 'package:authsome/authsome.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

/// Builds a client whose transport records the body instead of sending it, so
/// the assertions below read what actually went on the wire rather than what
/// the encoder was asked to produce.
({AuthClient client, String Function() body}) clientRecordingBody() {
  var sent = '';

  final client = AuthClient(AuthClientConfig(
    baseUrl: 'https://auth.example.com',
    httpClient: MockClient((request) async {
      sent = request.body;
      return http.Response(
        '{"access_token":"at","token_type":"Bearer","expires_in":3600}',
        200,
        headers: {'content-type': 'application/json'},
      );
    }),
  ));

  return (client: client, body: () => sent);
}

void main() {
  group('form-encoded request bodies', () {
    // RFC 8707 makes `resource` repeatable, and the server reads every value
    // it is given. One value per field is the only shape that survives that,
    // so these assert on queryParametersAll rather than the joined string.
    test('sends a one-element array as a single field', () async {
      final recorder = clientRecordingBody();

      await recorder.client.oauth2Token(
        body: Oauth2TokenRequest(
          grantType: 'client_credentials',
          resource: ['https://api.example.com'],
        ),
      );

      expect(
        Uri(query: recorder.body()).queryParametersAll['resource'],
        ['https://api.example.com'],
      );
    });

    test('repeats the field once per element for a two-element array',
        () async {
      final recorder = clientRecordingBody();

      await recorder.client.oauth2Token(
        body: Oauth2TokenRequest(
          grantType: 'client_credentials',
          resource: ['https://a.example.com', 'https://b.example.com'],
        ),
      );

      expect(
        Uri(query: recorder.body()).queryParametersAll['resource'],
        ['https://a.example.com', 'https://b.example.com'],
      );
    });

    test('keeps an empty array off the wire entirely', () async {
      final recorder = clientRecordingBody();

      await recorder.client.oauth2Token(
        body: Oauth2TokenRequest(
          grantType: 'client_credentials',
          resource: const [],
        ),
      );

      expect(
        Uri(query: recorder.body()).queryParametersAll.containsKey('resource'),
        isFalse,
      );
    });

    test('still writes a scalar field as one value', () async {
      final recorder = clientRecordingBody();

      await recorder.client.oauth2Token(
        body: Oauth2TokenRequest(
          grantType: 'authorization_code',
          code: 'the-code',
          resource: ['https://api.example.com'],
        ),
      );

      expect(
        Uri(query: recorder.body()).queryParameters['grant_type'],
        'authorization_code',
      );
    });

    test('omits an absent optional field', () async {
      final recorder = clientRecordingBody();

      await recorder.client.oauth2Token(
        body: Oauth2TokenRequest(grantType: 'client_credentials'),
      );

      expect(
        Uri(query: recorder.body()).queryParametersAll.containsKey('resource'),
        isFalse,
      );
    });
  });

  group('query-string parameters', () {
    // A query string carries repeated keys, the same as a form body does. The
    // authorization endpoint reads every `resource` it is given, so a list has
    // to be written one element at a time rather than stringified.
    ({AuthClient client, Uri Function() url}) clientRecordingUrl() {
      var seen = Uri.parse('https://auth.example.com');

      final client = AuthClient(AuthClientConfig(
        baseUrl: 'https://auth.example.com',
        httpClient: MockClient((request) async {
          seen = request.url;
          return http.Response('', 204);
        }),
      ));

      return (client: client, url: () => seen);
    }

    test('sends a one-element list as a single parameter', () async {
      final recorder = clientRecordingUrl();

      await recorder.client.oauth2Authorize(
        responseType: 'code',
        clientId: 'the-client',
        resource: ['https://api.example.com'],
      );

      expect(recorder.url().queryParametersAll['resource'],
          ['https://api.example.com']);
    });

    test('repeats the parameter once per element for a two-element list',
        () async {
      final recorder = clientRecordingUrl();

      await recorder.client.oauth2Authorize(
        responseType: 'code',
        clientId: 'the-client',
        resource: ['https://a.example.com', 'https://b.example.com'],
      );

      expect(recorder.url().queryParametersAll['resource'],
          ['https://a.example.com', 'https://b.example.com']);
    });

    test('keeps an absent list off the query string', () async {
      final recorder = clientRecordingUrl();

      await recorder.client
          .oauth2Authorize(responseType: 'code', clientId: 'the-client');

      expect(recorder.url().queryParametersAll.containsKey('resource'),
          isFalse);
    });
  });
}
