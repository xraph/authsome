# authsome

Dart client SDK for the [Authsome](https://github.com/xraph/authsome) authentication platform.

This package provides a typed HTTP client and request/response models for
interacting with an Authsome server from Dart and Flutter applications. It is
the underlying transport used by the `authsome_core`, `authsome_flutter`, and
`authsome_flutter_ui` packages.

## Installation

Add the package to your `pubspec.yaml`:

```yaml
dependencies:
  authsome: ^1.5.0
```

Then run:

```sh
dart pub get
```

## Usage

```dart
import 'package:authsome/authsome.dart';

void main() async {
  final client = AuthsomeClient(
    baseUrl: 'https://auth.example.com',
    publishableKey: 'pk_live_...',
  );

  final session = await client.signIn(
    SignInRequest(identifier: 'user@example.com', password: 'hunter2'),
  );

  print(session.user?.id);
}
```

See the [Authsome documentation](https://github.com/xraph/authsome) for the
full set of endpoints, flows, and configuration options.

## License

MIT — see [LICENSE](LICENSE).
