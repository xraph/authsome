#!/usr/bin/env bash
# Bumps every Dart/Flutter package to a release version in the working tree.
#
# RUN THIS BEFORE TAGGING AND COMMIT THE RESULT. pub.dev's automated publishing
# matches the pushed tag against each package's `version:` in pubspec.yaml and
# refuses the upload when they disagree, so the version has to be in the commit
# the tag points at. CI cannot stamp it during the run -- that is the difference
# from the old prepare-flutter-publish.sh, which mutated pubspecs mid-workflow.
#
# Usage: ./scripts/bump-flutter-version.sh <version>   # e.g. 1.6.0
set -euo pipefail

VERSION="${1:?Usage: bump-flutter-version.sh <version> (e.g. 1.6.0)}"

if ! printf '%s' "$VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([-+].+)?$'; then
  echo "error: version must be semver without a leading v, got '$VERSION'" >&2
  exit 1
fi

PKG_DIR="$(cd "$(dirname "$0")/../flutter/packages" && pwd)"
PACKAGES=(authsome_core authsome_flutter authsome_flutter_ui)

echo "Bumping Flutter packages to $VERSION..."

# perl rather than `sed -i`, whose in-place flag takes a mandatory argument on
# BSD/macOS but not GNU. This runs on a maintainer's machine, not only on CI.
for pkg in "${PACKAGES[@]}"; do
  perl -pi -e "s{^version: .*}{version: $VERSION}" "$PKG_DIR/$pkg/pubspec.yaml"
  echo "  $pkg -> $VERSION"
done

# Inter-package constraints track the release version: the three are published as
# a set, so a dependent never wants an older sibling. Anchored on leading
# whitespace so the `name:` key at column 0 can't match.
perl -pi -e "s{^(\s+authsome_core:).*}{\$1 ^$VERSION}" "$PKG_DIR/authsome_flutter/pubspec.yaml"
perl -pi -e "s{^(\s+authsome_flutter:).*}{\$1 ^$VERSION}" "$PKG_DIR/authsome_flutter_ui/pubspec.yaml"
echo "  inter-package constraints -> ^$VERSION"

echo
echo "Done. Review, commit, then tag v$VERSION:"
echo "  git commit -am 'chore(flutter): release $VERSION' && git tag v$VERSION && git push --follow-tags"
