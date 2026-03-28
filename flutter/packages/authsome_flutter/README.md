# authsome_flutter

Flutter integration for [AuthSome](https://github.com/xraph/authsome) authentication.

## Features

- Secure token storage via `flutter_secure_storage`
- `AuthManager` with automatic token refresh
- Stream-based authentication state

## Getting Started

```dart
import 'package:authsome_flutter/authsome_flutter.dart';

final auth = AuthManager(
  baseUrl: 'https://your-authsome-server.com',
);

// Listen to auth state
auth.stateStream.listen((state) {
  print('Auth state: $state');
});

// Sign in
await auth.signIn(email: 'user@example.com', password: 'password');
```

## Documentation

See the [AuthSome documentation](https://github.com/xraph/authsome) for more details.
