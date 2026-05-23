#!/usr/bin/env bash
# Boot smoke test against the real, on-disk dev config. Whatever lives
# at ~/.config/switchx/config.toml IS the input — no temp seed, no XDG
# override. If that config is valid and the secrets/DB it points at
# are reachable, switchx must boot the full chain (load → secrets →
# connect → health → migrate → placeholder banner) and exit 0 without
# ever showing the wizard.
#
# Distinct from wizard-test-* on purpose: those scripts force the
# wizard via SWITCHX_FORCE_WIZARD=1 to exercise the validate state.
# This one leaves SWITCHX_FORCE_WIZARD unset so config.Load succeeds
# and main takes the no-wizard path.
#
# Pre-requisites (one-time):
#   1. `make dev-up` has run (local PG + role + database).
#   2. `az login` has produced a session that can read the dev KV
#      named in .local/.env (AZURE_VAULT).
#   3. ~/.config/switchx/config.toml exists and is complete (run the
#      wizard once, or hand-write it).
set -euo pipefail

CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}/switchx/config.toml"
if [[ ! -f "$CONFIG" ]]; then
    echo "no config at $CONFIG" >&2
    echo "run the wizard once (e.g. \`make run\`) or hand-write the file." >&2
    exit 2
fi

echo "config: $CONFIG"
echo
echo "Expecting: no wizard UI, just the placeholder banner."
echo

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR/../.."
go run ./cmd/switchx
