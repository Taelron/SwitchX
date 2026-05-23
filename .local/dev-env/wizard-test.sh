#!/usr/bin/env bash
# Launch the bootstrap wizard with a pre-filled seed config so the user
# can walk through the wizard UI for one of three scenarios:
#
#   make bootstrap-test-wizard-ok          — every value correct: validate succeeds,
#                                            config persisted, wizard exits cleanly.
#   make bootstrap-test-wizard-bad-azure   — bad subscription: validate fails with
#                                            "secret store (...)" in the banner.
#   make bootstrap-test-wizard-bad-db      — nonexistent DB: validate fails with
#                                            "database connect (...)" in the banner.
#
# Reads .local/.env (see .local/.env.example).
#
# Each invocation wipes the seed dir under .local/run/ and writes a
# fresh config.toml there so successive runs do not see leftover
# state from the previous scenario.
set -euo pipefail

scenario="${1:-}"
case "$scenario" in
    ok|bad-azure|bad-db) ;;
    *)
        echo "usage: $0 {ok|bad-azure|bad-db}" >&2
        exit 2
        ;;
esac

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$DIR/../.env"
if [[ ! -f "$ENV_FILE" ]]; then
    echo ".local/.env not found — copy from .local/.env.example to begin" >&2
    exit 1
fi
# shellcheck source=/dev/null
source "$ENV_FILE"

: "${AZURE_VAULT:?missing in .local/.env}"
KV_USER_REF="${KV_USER_REF:-SWITCHX-DB-USER}"
KV_PASSWORD_REF="${KV_PASSWORD_REF:-SWITCHX-DB-PASSWORD}"

SUBSCRIPTION="${AZURE_SUBSCRIPTION:-$(az account show --query name -o tsv)}"
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_DB="${PG_DB:-switchx}"
PG_SSLMODE="${PG_SSLMODE:-disable}"

case "$scenario" in
    bad-azure)
        SUBSCRIPTION="this-subscription-does-not-exist"
        ;;
    bad-db)
        PG_DB="switchx_does_not_exist_xyz"
        ;;
esac

REPO_ROOT="$(cd "$DIR/../.." && pwd)"
SEED_DIR="$REPO_ROOT/.local/run/wizard-test-seed"
rm -rf "$SEED_DIR"
mkdir -p "$SEED_DIR/switchx"

cat > "$SEED_DIR/switchx/config.toml" <<EOF
[database]
host = "$PG_HOST"
port = $PG_PORT
name = "$PG_DB"
sslmode = "$PG_SSLMODE"

[database.secret]
provider     = "azurekeyvault"
subscription = "$SUBSCRIPTION"
vault        = "$AZURE_VAULT"
user_ref     = "$KV_USER_REF"
password_ref = "$KV_PASSWORD_REF"

[user]
timezone = "Europe/Brussels"

[ui]
editor = "vi"
EOF
chmod 0600 "$SEED_DIR/switchx/config.toml"

echo "scenario    : $scenario"
echo "seed file   : $SEED_DIR/switchx/config.toml"
echo "subscription: $SUBSCRIPTION"
echo "database    : $PG_DB"
echo
echo "Press Enter through each step to reach the validate state."
echo

cd "$REPO_ROOT"
XDG_CONFIG_HOME="$SEED_DIR" \
SWITCHX_FORCE_WIZARD=1 \
    go run ./cmd/switchx
