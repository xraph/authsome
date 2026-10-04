# Changelog

## Unreleased

- Sign-in now hides the passkey option, spacing and divider when the selected
  authenticator is unavailable. Social login retains its divider, and you can
  still supply a supported native authenticator through `passkeyAuthenticator`.

## 1.6.1

- Release versioning is now unified across `authsome_core`, `authsome_flutter`
  and `authsome_flutter_ui`. All three publish from a single `v{{version}}` tag
  via pub.dev automated publishing.
- Fixed the `authsome_flutter` dependency constraint, which was `^0.1.0` and so
  could never resolve against a 1.x release. It is now `^1.6.1`.

## 1.4.0

- See the [GitHub releases](https://github.com/xraph/authsome/releases) for the
  changes in this and earlier versions.
