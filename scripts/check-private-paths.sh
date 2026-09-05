#!/usr/bin/env bash
set -euo pipefail

# Paths that must never be tracked in this repository.
#
# This is not a style rule. Go's module proxy archives the ENTIRE module tree
# at every tagged version and serves it forever, whether or not the files are
# importable Go. Anything committed here and then tagged is public
# permanently, and deleting the tag afterwards does not take it back.
#
# That already happened once. cloud/ shipped in v0.0.1 through v0.0.16: ten
# markdown files covering the control plane's API, architecture, billing model
# and deployment topology. proxy.golang.org still serves every one of those
# versions today, and it is not removable. No credentials were in them, which
# is the only reason that was survivable.
#
# .gitignore stops the accident. This check stops .gitignore being bypassed by
# `git add -f`, by a stale clone whose .gitignore predates the entry, or by a
# merge that reintroduces a path from an older branch.
FORBIDDEN=(
  "cloud"
  "_project_files"
  ".superpowers"
  "docs/superpowers"
)

fail=0
for prefix in "${FORBIDDEN[@]}"; do
  # ls-files lists what is TRACKED, which is the thing that matters. A file
  # merely sitting on disk is fine and expected: the whole point is that you
  # can keep working on these locally.
  matches=$(git ls-files -- "$prefix" "$prefix/**" 2>/dev/null || true)
  if [ -n "$matches" ]; then
    fail=1
    echo "FORBIDDEN PATH TRACKED: $prefix/"
    echo "$matches" | sed 's/^/    /'
    echo
  fi
done

if [ "$fail" -ne 0 ]; then
  cat <<'EOF'
These paths are tracked and must not be.

Once a version is tagged, Go's module proxy archives the whole tree and serves
it indefinitely. Removing the file later, deleting the tag, or making the repo
private will not un-publish it.

To fix, keeping your local copy:

    git rm -r --cached <path>
    git commit -m "chore: untrack <path>"

Note that `git rm --cached` produces a commit that looks like an ordinary
deletion, so a later rebase or cherry-pick of it WILL remove the files from
your working tree. Back them up outside the repo first if you need them.
EOF
  exit 1
fi

echo "No forbidden paths tracked."
