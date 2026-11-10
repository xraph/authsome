#!/usr/bin/env bash
# Runs govulncheck and fails on any reachable vulnerability that isn't listed
# in .govulncheck-allow. The plain text report is printed either way.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

version="${GOVULNCHECK_VERSION:-v1.6.0}"
cmd=(go run "golang.org/x/vuln/cmd/govulncheck@${version}")

"${cmd[@]}" ./... || true

json="$(mktemp)"
trap 'rm -f "$json"' EXIT
"${cmd[@]}" -format json ./... >"$json"

# A finding whose first trace frame names a function is one your code calls.
found="$(jq -r 'select(.finding != null and .finding.trace[0].function != null) | .finding.osv' "$json" | sort -u)"
allowed="$(grep -Ev '^\s*(#|$)' .govulncheck-allow 2>/dev/null | sort -u || true)"

blocking="$(comm -23 <(printf '%s\n' "$found" | sed '/^$/d') <(printf '%s\n' "$allowed" | sed '/^$/d'))"
stale="$(comm -13 <(printf '%s\n' "$found" | sed '/^$/d') <(printf '%s\n' "$allowed" | sed '/^$/d'))"

if [ -n "$stale" ]; then
  echo "These allowlisted advisories no longer show up; remove them from .govulncheck-allow:"
  printf '%s\n' "$stale" | sed 's/^/  /'
fi
if [ -n "$blocking" ]; then
  echo "Reachable vulnerabilities not in .govulncheck-allow:"
  printf '%s\n' "$blocking" | sed 's/^/  /'
  exit 1
fi
echo "govulncheck: no reachable vulnerabilities outside the allowlist."
