#!/usr/bin/env bash
# Run the gated Postgres integration tests against the local DB. Pulls
# the password from Azure Key Vault and exports the env vars expected
# by the runIntegration() helper in the test files.
#
# Reads .local/.env (see .local/.env.example).
# Invoked by `make pg-integration`.
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
: "${KV_PASSWORD_REF:?missing in .local/.env}"
PG_USER="${PG_USER:-switchx_user}"

PW=$(az keyvault secret show --vault-name "$AZURE_VAULT" --name "$KV_PASSWORD_REF" --query value -o tsv)

REPO_ROOT="$(cd "$DIR/../.." && pwd)"
cd "$REPO_ROOT"

SWITCHX_PG_INTEGRATION=1 \
SWITCHX_PG_PASSWORD="$PW" \
SWITCHX_PG_USER="$PG_USER" \
    go test -run 'TestPGIntegration|TestMigrateIntegration|TestStartupOnce_HappyPath_RealPG' -v \
        ./internal/storage/postgres/ ./cmd/switchx/
