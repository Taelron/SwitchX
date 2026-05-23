#!/usr/bin/env bash
# Run the M1 end-to-end integration test (TestM1E2E) against the local
# Postgres. Pulls the PG password from Azure Key Vault and exports the
# env vars expected by runIntegration() in cmd/switchx/startup_test.go.
#
# Reads .local/.env (see .local/.env.example).
# Invoked by `make smoke-integration`.
#
# Sibling of pg-integration.sh; kept separate because pg-integration.sh
# runs the per-package integration suites while this script runs only
# the M1 closure E2E test.
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
    go test -run TestM1E2E -v ./cmd/switchx/
