#!/usr/bin/env bash
# Aggregate Flutter package coverage into a single lcov report.
#
# Usage: ./flutter/scripts/coverage.sh
# Output: flutter/coverage/lcov.info (merged) + flutter/coverage/html/ (if lcov is installed)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FLUTTER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PACKAGES=(
  "authsome_core"
  "authsome_flutter"
  "authsome_flutter_ui"
)

mkdir -p "$FLUTTER_ROOT/coverage"
COMBINED="$FLUTTER_ROOT/coverage/lcov.info"
: >"$COMBINED"

for pkg in "${PACKAGES[@]}"; do
  PKG_DIR="$FLUTTER_ROOT/packages/$pkg"
  echo "==> $pkg"
  (cd "$PKG_DIR" && flutter test --coverage)

  if [[ -f "$PKG_DIR/coverage/lcov.info" ]]; then
    # Prefix paths so the merged report is unambiguous and exclude generated code.
    awk -v pkg="$pkg" '
      /^SF:/ {
        sub("^SF:", "SF:packages/" pkg "/");
        if ($0 ~ /lib\/src\/generated\//) { skip=1; next }
        skip=0
      }
      !skip { print }
      /^end_of_record/ { skip=0 }
    ' "$PKG_DIR/coverage/lcov.info" >>"$COMBINED"
  fi
done

echo
echo "Merged coverage written to: $COMBINED"

if command -v genhtml >/dev/null 2>&1; then
  genhtml -q -o "$FLUTTER_ROOT/coverage/html" "$COMBINED"
  echo "HTML report: $FLUTTER_ROOT/coverage/html/index.html"
else
  echo "(install lcov for an HTML report: brew install lcov)"
fi
