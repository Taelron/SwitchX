#!/usr/bin/env bash
# Confirm that dev-up.sh fails fast with a clear message when
# .local/.env is missing entirely. Required by TAE-15's acceptance
# criterion ("Scripts fail fast on missing variables").
#
# Invoked by `make verify-env-failfast`. Temporarily moves .local/.env
# aside, runs dev-up, asserts the failure message, restores the file.
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$DIR/../.env"
BAK="$ENV_FILE.verify-bak"

# Restore .env on any exit path — even Ctrl+C — so this verification
# is safe to run on a machine where .env carries real values.
cleanup() {
    if [[ -f "$BAK" ]]; then
        mv "$BAK" "$ENV_FILE"
    fi
}
trap cleanup EXIT INT TERM

# Move .env aside (if present) so dev-up sees no .env.
if [[ -f "$ENV_FILE" ]]; then
    mv "$ENV_FILE" "$BAK"
fi

# Run dev-up; expect a non-zero exit and the documented friendly
# message. Disable `set -e` for this call because we WANT it to fail.
set +e
output=$(bash "$DIR/dev-up.sh" 2>&1)
status=$?
set -e

if [[ $status -eq 0 ]]; then
    echo "FAIL: dev-up.sh exited 0 with no .env present" >&2
    echo "output was:" >&2
    echo "$output" >&2
    exit 1
fi

if ! grep -q ".local/.env not found" <<<"$output"; then
    echo "FAIL: dev-up.sh did not print the documented 'not found' message" >&2
    echo "got output:" >&2
    echo "$output" >&2
    exit 1
fi

echo "OK: dev-up.sh fails fast with a clear message when .local/.env is absent"
