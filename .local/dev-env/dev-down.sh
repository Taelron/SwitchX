#!/usr/bin/env bash
# Drop the local switchx database (and the per-app role). Use to fully
# wipe state between test runs or before reinstalling.
#
# Reads .local/.env (see .local/.env.example).
# Invoked by `make dev-down`.
#
# This is destructive: every table in the switchx DB is dropped along
# with the DB itself. The KV secrets are NOT touched; the next
# `make dev-up` will recreate the DB and ALTER the role's password
# back to whatever KV currently holds.
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$DIR/../.env"
if [[ ! -f "$ENV_FILE" ]]; then
    echo ".local/.env not found — copy from .local/.env.example to begin" >&2
    exit 1
fi
# shellcheck source=/dev/null
source "$ENV_FILE"

PG_DB="${PG_DB:-switchx}"
PG_USER="${PG_USER:-switchx_user}"

psql -v ON_ERROR_STOP=1 -d postgres <<EOSQL
-- Drop the database (closes any open connections via FORCE).
DROP DATABASE IF EXISTS ${PG_DB} WITH (FORCE);

-- Drop the per-app role too so dev-up recreates it cleanly.
DROP ROLE IF EXISTS ${PG_USER};
EOSQL

echo "dropped database '${PG_DB}' and role '${PG_USER}'"

# Symmetric to dev-up: remove the synthetic dev config too. ~/.config/
# is untouched on principle (it belongs to the deployed binary).
REPO_ROOT="$(cd "$DIR/../.." && pwd)"
DEV_XDG="$REPO_ROOT/.local/dev-config"
if [[ -d "$DEV_XDG" ]]; then
    rm -rf "$DEV_XDG"
    echo "removed dev XDG: $DEV_XDG"
fi
