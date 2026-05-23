#!/usr/bin/env bash
# Manual smoke tests for the M1 startup decision tree.
# Backs the Makefile targets startup-test-no-config /
# startup-test-db-down / startup-test-retry-recovers.
#
# Pre-requisites (one-time):
#   1. `make dev-up` has run (local PG + role + database + dev config).
#   2. The db-down and retry-recovers scenarios use the dev config that
#      `make dev-up` writes to .local/dev-config/switchx/config.toml.
#      The Makefile sets XDG_CONFIG_HOME for those scenarios; this
#      script honors it (and falls back to ~/.config/ if you run the
#      script directly — though that mode is not the supported path).
#   3. The no-config scenario uses a fresh temporary XDG_CONFIG_HOME
#      regardless, so neither the dev nor the deployed config is touched.
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

    db-down|retry-recovers)
        # Without a valid config the binary would hit the wizard branch
        # instead of the error-screen branch we want to exercise. Fail
        # fast with a clear message rather than a misleading wizard launch.
        CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}/switchx/config.toml"
        if [[ ! -f "$CONFIG" ]]; then
            echo "no config at $CONFIG" >&2
            echo "run \`make dev-up\` to generate the dev config." >&2
            exit 2
        fi

        echo "scenario : $scenario"
        echo "config   : $CONFIG"
        echo
        echo "Stopping local Postgres…"
        bash "$DIR/pg.sh" stop
        echo
        if [[ "$scenario" == "db-down" ]]; then
            echo "Expecting: error screen with \"Database refused connection\"."
            echo "Press q to quit. Then run \`make pg-start\` to restart PG."
        else
            echo "Expecting: error screen appears."
            echo
            echo "When you see the error screen:"
            echo "  1. In another terminal, run: make pg-start"
            echo "  2. Press \`r\` on the error screen."
            echo "  3. The home screen should appear with \"Connected to switchx\"."
        fi
        echo
        go run ./cmd/switchx
        ;;
esac
