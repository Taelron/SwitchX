#!/usr/bin/env bash
# Start or stop the local Postgres service so the startup-test scripts
# can exercise the error-screen flow against a real binary without typing
# the systemctl invocation each time.
#
# Reads .local/.env for PG_SERVICE (defaults to "postgresql").
# Invoked by `make pg-start` / `make pg-stop` and by startup-test.sh.
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Source .env only if present. PG_SERVICE has a safe default, so this
# script works on a fresh machine before the contributor has run the
# first-run setup (make startup-test-* may invoke pg.sh before .env exists).
# shellcheck source=/dev/null
[[ -f "$DIR/../.env" ]] && source "$DIR/../.env"

PG_SERVICE="${PG_SERVICE:-postgresql}"

case "${1:-}" in
    start) sudo systemctl start "$PG_SERVICE" ;;
    stop)  sudo systemctl stop  "$PG_SERVICE" ;;
    *)     echo "usage: $0 {start|stop}" >&2; exit 2 ;;
esac
