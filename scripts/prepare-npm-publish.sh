#!/usr/bin/env bash
# Resolves pnpm workspace:* protocol and stamps version for npm publishing.
# Usage: ./scripts/prepare-npm-publish.sh <version>
# Note: This script uses GNU sed syntax and is intended for CI (Linux).
set -euo pipefail

VERSION="${1:?Usage: prepare-npm-publish.sh <version>}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "Preparing npm packages for publish at version $VERSION..."

# Stamp version in all publishable packages
for pkg in ui/packages/core ui/packages/react ui/packages/vue ui/packages/components ui/packages/nextjs sdk/typescript; do
  cd "$ROOT/$pkg"
  npm version "$VERSION" --no-git-tag-version --allow-same-version
  echo "  Set $pkg version to $VERSION"
done

# Replace workspace:* with ^VERSION in all package.json files under ui/packages/
for pkg in ui/packages/react ui/packages/vue ui/packages/components ui/packages/nextjs; do
  cd "$ROOT/$pkg"
  sed -i "s/\"workspace:\*\"/\"^$VERSION\"/g" package.json
  echo "  Resolved workspace:* references in $pkg"
done

echo "Done. npm packages ready for publishing."
