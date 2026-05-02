# switchx — developer Makefile.
#
# Run `make` (no args) to list targets. Per-machine dev-env scripts
# live in .local/dev-env/ and are not in Git.

.DEFAULT_GOAL := help

.PHONY: help build vet lint test verify run run-wizard dev-up dev-down pg-integration

help: ## list targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build:  ## go build ./...
	go build ./...

vet:    ## go vet ./...
	go vet ./...

lint:   ## golangci-lint run ./...
	golangci-lint run ./...

test:   ## go test ./...
	go test ./...

verify: build vet lint test ## run the full verification gate

run:    ## go run ./cmd/switchx
	go run ./cmd/switchx

run-wizard: ## smoke the bootstrap wizard against an empty XDG_CONFIG_HOME
	@rm -rf /tmp/switchx-wizard-smoke
	XDG_CONFIG_HOME=/tmp/switchx-wizard-smoke go run ./cmd/switchx

dev-up: ## bootstrap local PG: fetch KV creds + create role + database
	@bash .local/dev-env/dev-up.sh

dev-down: ## destructive: drop the local switchx database and role (KV untouched)
	@bash .local/dev-env/dev-down.sh

pg-integration: ## run the gated PG integration tests against the local DB
	@bash .local/dev-env/pg-integration.sh
