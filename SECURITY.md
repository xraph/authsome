# Security policy

## Reporting a vulnerability

Email security@xraph.com with what you found, the version or commit you tested, and the steps to reproduce it. Do not open a public issue for a vulnerability. You will get an acknowledgement within two working days and a fix or a mitigation plan within thirty days for anything rated high or critical. We credit reporters in the release notes unless you ask us not to.

## Scope

In scope: this module, its plugins, the SDKs under `sdk/`, the Flutter packages under `flutter/`, and the release and CI workflows. Out of scope: the Forge, Relay, Chronicle, Warden and Keysmith modules, which have their own policies, and any deployment configuration you own.

## Supported versions

The latest minor release receives security fixes. Older releases get a fix only when the issue is critical and the fix is small.

## What we do on our side

Every commit runs `govulncheck`, `gosec` through golangci-lint, a secret scan and, on pull requests, a dependency review. Actions are pinned to commits. Releases are signed and published through trusted publishing rather than long-lived tokens. See `docs/content/docs/security/` for how the controls in the code work.
