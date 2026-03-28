# authsome_core

Framework-agnostic Dart SDK for [AuthSome](https://github.com/xraph/authsome) authentication.

## Features

- Complete API client for AuthSome server
- Token management and auto-refresh
- MFA support
- Social/OAuth provider support

## Getting Started

```dart
import 'package:authsome_core/authsome_core.dart';

final client = AuthSomeClient(
  config: AuthClientConfig(baseUrl: 'https://your-authsome-server.com'),
);

final session = await client.signIn(
  email: 'user@example.com',
  password: 'password',
);
```

## Documentation

See the [AuthSome documentation](https://github.com/xraph/authsome) for more details.
