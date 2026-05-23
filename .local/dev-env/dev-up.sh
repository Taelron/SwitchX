#!/usr/bin/env bash
# Fetch the switchx DB credentials from Azure Key Vault and bootstrap
# the local Postgres role + database. Idempotent.
#
# Reads .local/.env (see .local/.env.example).
# Invoked by `make dev-up`.
set -euo pipefail

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
PG_DB="${PG_DB:-switchx}"
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_SSLMODE="${PG_SSLMODE:-disable}"

ROLE=$(az keyvault secret show --vault-name "$AZURE_VAULT" --name "$KV_USER_REF"     --query value -o tsv)
PW=$(  az keyvault secret show --vault-name "$AZURE_VAULT" --name "$KV_PASSWORD_REF" --query value -o tsv)

psql -v ON_ERROR_STOP=1 \
    -v role="$ROLE" -v pw="$PW" -v db="$PG_DB" -d postgres <<'EOSQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'role', :'pw')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'role')
\gexec

-- Grant LOGIN, the rotated password, and CREATEDB. CREATEDB lets the
-- integration tests spin up uniquely-named throwaway databases so they
-- never touch the developer's working `switchx` DB.
SELECT format('ALTER ROLE %I WITH LOGIN CREATEDB PASSWORD %L', :'role', :'pw')
\gexec

-- The current OS-user role (e.g. gaetan) must be a member of the app role
-- to create / own a database on its behalf. Idempotent.
SELECT format('GRANT %I TO CURRENT_USER', :'role')
\gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'db', :'role')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'db')
\gexec

SELECT format('ALTER DATABASE %I OWNER TO %I', :'db', :'role')
\gexec
EOSQL

# Write the dev XDG config so `make run` and the bootstrap/startup test
# targets read from .local/dev-config/ instead of ~/.config/switchx/
# (which is reserved for the deployed binary's real user data).
# Subscription is resolved from .env first, then `az account show`, matching
# wizard-test.sh's pattern.
SUBSCRIPTION="${AZURE_SUBSCRIPTION:-$(az account show --query name -o tsv)}"
REPO_ROOT="$(cd "$DIR/../.." && pwd)"
DEV_XDG="$REPO_ROOT/.local/dev-config"
mkdir -p "$DEV_XDG/switchx"
cat > "$DEV_XDG/switchx/config.toml" <<EOF
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
chmod 0600 "$DEV_XDG/switchx/config.toml"

echo "ready: database '$PG_DB' owned by role '$ROLE' on $PG_HOST:$PG_PORT"
echo "dev config: $DEV_XDG/switchx/config.toml (XDG_CONFIG_HOME for make targets)"
