#!/usr/bin/env bash
# Prepares Flutter packages for pub.dev publishing.
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

# Remove resolution: workspace from all member packages (not valid for pub.dev)
for pkg in authsome_core authsome_flutter authsome_flutter_ui; do
  sed -i '/^resolution: workspace$/d' "$FLUTTER_DIR/$pkg/pubspec.yaml"
  echo "  Removed resolution: workspace from $pkg"
done

# Update inter-package version constraints to match release version
sed -i "s/authsome_core: ^0.1.0/authsome_core: ^$VERSION/" "$FLUTTER_DIR/authsome_flutter/pubspec.yaml"
echo "  Updated authsome_flutter -> authsome_core constraint to ^$VERSION"

sed -i "s/authsome_flutter: ^0.1.0/authsome_flutter: ^$VERSION/" "$FLUTTER_DIR/authsome_flutter_ui/pubspec.yaml"
echo "  Updated authsome_flutter_ui -> authsome_flutter constraint to ^$VERSION"

echo "Done. Flutter packages ready for pub.dev publishing."
