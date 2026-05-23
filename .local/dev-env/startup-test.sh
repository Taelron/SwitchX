#!/usr/bin/env bash
# Manual smoke tests for the M1 startup decision tree.
# Backs the Makefile targets startup-test-no-config /
# startup-test-db-down / startup-test-retry-recovers.
#
# Pre-requisites (one-time):
#   1. `make dev-up` has run (local PG + role + database).
#   2. ~/.config/switchx/config.toml is complete (for db-down and
#      retry-recovers scenarios; the no-config scenario uses a
#      temporary XDG_CONFIG_HOME so the real config is left alone).
set -euo pipefail

scenario="${1:-}"
case "$scenario" in
    no-config|db-down|retry-recovers) ;;
    *)
        echo "usage: $0 {no-config|db-down|retry-recovers}" >&2
        exit 2
        ;;
esac

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DIR/../.." && pwd)"
cd "$REPO_ROOT"

case "$scenario" in
    no-config)
        XDG="$REPO_ROOT/.local/run/startup-smoke-no-config"
        rm -rf "$XDG"
        mkdir -p "$XDG"
        echo "scenario : no-config"
        echo "XDG dir  : $XDG"
        echo
        echo "Expecting: bootstrap wizard opens on step 1 (database)."
        echo
        XDG_CONFIG_HOME="$XDG" go run ./cmd/switchx
        ;;

    db-down)
        echo "scenario : db-down"
        echo
        echo "Stopping local Postgres…"
        bash "$DIR/pg.sh" stop
        echo
        echo "Expecting: error screen with \"Database refused connection\"."
        echo "Press q to quit. Then run \`make pg-start\` to restart PG."
        echo
        go run ./cmd/switchx
        ;;

    retry-recovers)
        echo "scenario : retry-recovers"
        echo
        echo "Stopping local Postgres…"
        bash "$DIR/pg.sh" stop
        echo
        echo "Expecting: error screen appears."
        echo
        echo "When you see the error screen:"
        echo "  1. In another terminal, run: make pg-start"
        echo "  2. Press \`r\` on the error screen."
        echo "  3. The home screen should appear with \"Connected to switchx\"."
        echo
        go run ./cmd/switchx
        ;;
esac
