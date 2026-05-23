# switchx — developer Makefile.
#
# Run `make` (no args) to list targets. Dev-env scripts live in
# .local/dev-env/ (tracked); per-machine values live in .local/.env
# (gitignored — see .local/.env.example for the template).

.DEFAULT_GOAL := help

.PHONY: help build vet lint test verify run run-wizard dev-up dev-down \
        pg-integration smoke-integration pg-start pg-stop \
        bootstrap-test-no-wizard bootstrap-test-edit bootstrap-test-wizard-ok \
        bootstrap-test-wizard-bad-azure bootstrap-test-wizard-bad-db \
        startup-test-no-config startup-test-db-down startup-test-retry-recovers \
        verify-naming-leak verify-env-failfast verify-local-env

help: ## list targets
	@awk 'BEGIN {FS = ":.*?## "} \
		/^##@/ {printf "\n%s\n", substr($$0, 5)} \
		/^[a-zA-Z_-]+:.*?## / {printf "  %-32s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Build & verify

build:  ## go build ./...
	go build ./...

vet:    ## go vet ./...
	go vet ./...

lint:   ## golangci-lint run ./...
	golangci-lint run ./...

test:   ## go test ./...
	go test ./...

verify: build vet lint test ## run the full verification gate

##@ Run

run:    ## go run ./cmd/switchx
	go run ./cmd/switchx

run-wizard: ## smoke the bootstrap wizard against an empty XDG_CONFIG_HOME
	@rm -rf .local/run/wizard-smoke
	XDG_CONFIG_HOME=.local/run/wizard-smoke go run ./cmd/switchx

##@ Local dev environment

dev-up: ## bootstrap local PG: fetch KV creds + create role + database
	@bash .local/dev-env/dev-up.sh

dev-down: ## destructive: drop the local switchx database and role (KV untouched)
	@bash .local/dev-env/dev-down.sh

pg-integration: ## run the gated PG integration tests against the local DB
	@bash .local/dev-env/pg-integration.sh

smoke-integration: ## run the M1 end-to-end smoke test (TestM1E2E) against the local DB
	@bash .local/dev-env/smoke-integration.sh

pg-start: ## start the local Postgres service (for manual error-screen testing)
	@bash .local/dev-env/pg.sh start

pg-stop: ## stop the local Postgres service (for manual error-screen testing)
	@bash .local/dev-env/pg.sh stop

##@ Bootstrap tests (interactive)

bootstrap-test-no-wizard: ## valid config exists → no wizard, placeholder banner
	@bash .local/dev-env/bootstrap-test.sh

bootstrap-test-edit: ## --edit against real config → wizard pre-filled, saves back
	@bash .local/dev-env/bootstrap-edit-test.sh

bootstrap-test-wizard-ok: ## force wizard + valid seed → validate succeeds, persist
	@bash .local/dev-env/wizard-test.sh ok

bootstrap-test-wizard-bad-azure: ## force wizard + bad subscription → "secret store" error
	@bash .local/dev-env/wizard-test.sh bad-azure

bootstrap-test-wizard-bad-db: ## force wizard + nonexistent database → "database" error
	@bash .local/dev-env/wizard-test.sh bad-db

##@ Startup decision tree tests (interactive)

startup-test-no-config: ## no config present → wizard launches (path 1 smoke)
	@bash .local/dev-env/startup-test.sh no-config

startup-test-db-down: ## valid config + DB stopped → error screen (path 3 smoke)
	@bash .local/dev-env/startup-test.sh db-down

startup-test-retry-recovers: ## error screen → start DB → `r` → home (path 4 smoke)
	@bash .local/dev-env/startup-test.sh retry-recovers

##@ Local environment verification

verify-naming-leak: ## confirm no internal naming (vault, subscription) leaks into tracked files
	@bash .local/dev-env/verify-naming-leak.sh

verify-env-failfast: ## confirm scripts fail with a clear message when .local/.env is missing
	@bash .local/dev-env/verify-env-failfast.sh

verify-local-env: verify-naming-leak verify-env-failfast ## run all local-env verifications
	@echo "verify-local-env: all checks passed"
