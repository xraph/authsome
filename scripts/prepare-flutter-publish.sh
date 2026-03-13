#!/usr/bin/env bash
# Converts Flutter path dependencies to hosted version dependencies for pub.dev publishing.
# Usage: ./scripts/prepare-flutter-publish.sh <version>
# Note: This script uses GNU sed syntax and is intended for CI (Linux).
set -euo pipefail

VERSION="${1:?Usage: prepare-flutter-publish.sh <version>}"
FLUTTER_DIR="$(cd "$(dirname "$0")/../flutter/packages" && pwd)"

echo "Preparing Flutter packages for pub.dev publish at version $VERSION..."

# Set version in all pubspec.yaml files
for pkg in authsome_core authsome_flutter authsome_flutter_ui; do
  sed -i "s/^version: .*/version: $VERSION/" "$FLUTTER_DIR/$pkg/pubspec.yaml"
  echo "  Set $pkg version to $VERSION"
done

# Convert authsome_flutter's path dep on authsome_core to hosted
sed -i '/authsome_core:/{
n
s|path: ../authsome_core|version: ^'"$VERSION"'|
}' "$FLUTTER_DIR/authsome_flutter/pubspec.yaml"
echo "  Converted authsome_flutter -> authsome_core to hosted ^$VERSION"

# Convert authsome_flutter_ui's path dep on authsome_flutter to hosted
sed -i '/authsome_flutter:/{
n
s|path: ../authsome_flutter|version: ^'"$VERSION"'|
}' "$FLUTTER_DIR/authsome_flutter_ui/pubspec.yaml"
echo "  Converted authsome_flutter_ui -> authsome_flutter to hosted ^$VERSION"

echo "Done. Flutter packages ready for pub.dev publishing."
