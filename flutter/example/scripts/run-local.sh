#!/usr/bin/env bash
# Launch the example app against a locally running authsome backend.
#
# Usage: ./flutter/example/scripts/run-local.sh [device]
#   device: macos | chrome | ios | android  (defaults to chrome)
set -euo pipefail

DEVICE="${1:-chrome}"
BASE_URL="${AUTHSOME_BASE_URL:-http://localhost:8080}"
PUBLISHABLE_KEY="${AUTHSOME_PUBLISHABLE_KEY:-}"

cd "$(dirname "${BASH_SOURCE[0]}")/.."

flutter run \
  -d "$DEVICE" \
  --dart-define=AUTHSOME_BASE_URL="$BASE_URL" \
  --dart-define=AUTHSOME_PUBLISHABLE_KEY="$PUBLISHABLE_KEY"
