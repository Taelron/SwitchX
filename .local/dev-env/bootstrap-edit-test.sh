#!/usr/bin/env bash
# Edit-mode smoke: switchx --edit against the real on-disk config.
# Forces the wizard pre-filled with the existing values; on success
# the wizard saves back to the same path. Useful for verifying that
# round-tripping through the wizard does not corrupt the config.
#
#   make bootstrap-test-edit
#
# Pre-requisites (one-time):
#   1. `make dev-up` has run (local PG + role + database).
#   2. `az login` has produced a session that can read the dev KV
#      named in .local/.env (AZURE_VAULT).
#   3. ~/.config/switchx/config.toml exists and is complete.
set -euo pipefail

CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}/switchx/config.toml"
if [[ ! -f "$CONFIG" ]]; then
    echo "no config at $CONFIG" >&2
    echo "run \`make bootstrap-test-wizard-ok\` once to create one." >&2
    exit 2
fi

echo "config: $CONFIG"
echo
echo "Expecting: wizard opens pre-filled with current values; saves back on success."
echo

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR/../.."
go run ./cmd/switchx --edit
