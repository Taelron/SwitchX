#!/usr/bin/env bash
# Confirm that internal naming (vault names, subscription names,
# internal hostnames, etc.) does not appear in any tracked file.
# Per @Local Development Pattern 5.
#
# Invoked by `make verify-naming-leak`. Add new patterns below as new
# internal names enter the project; the goal is that this list grows
# over time and never finds a hit on a green branch.
set -euo pipefail

# Patterns to forbid in committed content. These are the real internal
# names that must live only in .local/.env or in Azure Key Vault.
PATTERNS=(
    "TAELRON-KV-SHA-DEV"
    "Visual Studio Enterprise"
)

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DIR/../.." && pwd)"
cd "$REPO_ROOT"

fail=0
for p in "${PATTERNS[@]}"; do
    # `git grep` searches only tracked content by default — exactly
    # what we want. An untracked .local/.env carrying these names is
    # the expected state. Exclude this script itself: the PATTERNS array
    # literal is the whole point of the file and obviously not a leak.
    if git grep -nF "$p" -- ':!:.local/dev-env/verify-naming-leak.sh' >&2; then
        echo "FAIL: '$p' found in tracked content above" >&2
        fail=1
    fi
done

if [[ $fail -eq 0 ]]; then
    echo "OK: no internal-naming leaks in tracked files (${#PATTERNS[@]} patterns checked)"
fi
exit $fail
