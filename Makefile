# switchx — developer Makefile.
#
# Run `make` (no args) to list targets. Per-machine dev-env scripts
# live in .local/dev-env/ and are not in Git.

.DEFAULT_GOAL := help

.PHONY: help build vet lint test verify run run-wizard dev-up dev-down pg-integration \
        bootstrap-test-no-wizard bootstrap-test-edit bootstrap-test-wizard-ok \
        bootstrap-test-wizard-bad-azure bootstrap-test-wizard-bad-db

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
