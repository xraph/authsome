#!/usr/bin/env bash
set -euo pipefail

# Files this PR touched that must hit a higher security-critical bar.
declare -A HIGH_BAR=(
  ["plugins/social/sanitize.go"]=90
  ["plugins/social/allowlist.go"]=90
  ["dashboard/nonce.go"]=85
  ["dashboard/audit.go"]=85
  ["plugins/organization/service.go"]=80
  ["plugins/organization/dashboard.go"]=70
)

# Standard threshold for other touched files.
STD_BAR=70
declare -a STD_FILES=(
  "store/memory/store.go"
  "internal/secutil/secutil.go"
  "api/introspect_handler.go"
)

# Generate a single coverage profile across all packages with new tests.
go test \
  ./plugins/social/... \
  ./plugins/organization/... \
  ./dashboard/... \
  ./store/memory/... \
  ./internal/secutil/... \
  ./api/... \
  -coverprofile=cover.out \
  -coverpkg=./plugins/social/...,./plugins/organization/...,./dashboard/...,./store/memory/...,./internal/secutil/...,./api/... \
  > /dev/null

failures=0
check() {
  local file=$1
  local threshold=$2
  local pct
  pct=$(go tool cover -func=cover.out | awk -v f="$file" '$1 ~ f"$" || $1 ~ "/"f"$" {print $3}' | head -1 | tr -d '%')
  if [[ -z "$pct" ]]; then
    echo "WARN: no coverage data for $file"
    return
  fi
  if (( $(awk -v p="$pct" -v t="$threshold" 'BEGIN { print (p < t) }') )); then
    echo "FAIL: $file coverage ${pct}% < ${threshold}%"
    failures=$((failures+1))
  else
    echo "OK:   $file coverage ${pct}% >= ${threshold}%"
  fi
}

for file in "${!HIGH_BAR[@]}"; do check "$file" "${HIGH_BAR[$file]}"; done
for file in "${STD_FILES[@]}"; do check "$file" "$STD_BAR"; done

if (( failures > 0 )); then
  echo "Coverage gate FAILED: $failures file(s) below threshold"
  exit 1
fi
echo "Coverage gate: OK"
