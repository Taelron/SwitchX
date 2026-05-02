# switchx

Terminal-native time tracker for consulting teams. Workspace-level rules and the collaboration model are in the parent directory's `CLAUDE.md` — this file only contains switchx-specific context.

## Current state

**Phase 1 — minimum viable implementation.**
Active milestone: **M1 — Foundation & Bootstrap** ([switchx project in Linear](https://linear.app/taelron/project/switchx-0b0069bd1c04)).

This file is updated when the active milestone changes (typically once per milestone closure).

## switchx-specific design (Linear — switchx project)

- @README ([Linear](https://linear.app/taelron/document/readme-3b00ab0b67b2)) — what it is, design principles, audience
- @Specification ([Linear](https://linear.app/taelron/document/specification-3645653a2170)) — Phase 1 behavior: use cases, views, validation, report format
- @Domain Model ([Linear](https://linear.app/taelron/document/domain-model-11af391c0064)) — entities, invariants, relationships, the at-most-one-open-session rule
- @ADR-0001 ([Linear](https://linear.app/taelron/document/adr-0001-at-most-one-open-session-per-user-d870088d6e6e)) — At-Most-One-Open-Session Per User
- @ADR-0002 ([Linear](https://linear.app/taelron/document/adr-0002-postgres-for-shared-team-storage-b6b0df46161c)) — Postgres for Shared Team Storage
- @ADR-0003 ([Linear](https://linear.app/taelron/document/adr-0003-adopt-hexagonal-architecture-and-domain-and-persistence-3392ec24d042)) — Adopt Hexagonal Architecture and Domain & Persistence Separation Baselines
- @ADR-0004 ([Linear](https://linear.app/taelron/document/adr-0004-configuration-and-bootstrap-e5000764e4c4)) — Configuration and Bootstrap
- @ADR-0005 ([Linear](https://linear.app/taelron/document/adr-0005-secret-handling-for-switchx-ea21d3f379f4)) — Secret Handling for switchx

## Active milestone — M1

**Goal:** A user can install the binary, run it, complete the bootstrap wizard, and reach a placeholder home screen connected to Postgres. No domain features yet.

**Issues:** TAE-5 through TAE-14, attached to the M1 milestone in Linear. Each issue's description carries its acceptance criteria, references, out-of-scope items, and dependencies. Start with TAE-5 (repo scaffold + CI); the dependency graph dictates the rest.

**Locked decisions for M1:**
- PostgreSQL minimum version: **16** (tested against 16 and 17).
- Migrations run on every launch; M1 ships with empty `migrations/`.
- Wizard re-runs until the database connection is validated.

## Quick reference

- **Module layout:** `internal/domain/`, `internal/app/`, `internal/storage/`, `internal/ui/tui/`, `migrations/`, `cmd/switchx/`
- **TUI stack:** Bubble Tea + Lipgloss
- **DB driver:** `pgx/v5` with `pgxpool` (max 4 connections)
- **Migrations:** `golang-migrate`
- **Secret provider (M1):** Azure Key Vault via `az` CLI

## Conventions specific to this repo

None yet beyond what the workspace baselines define. This section grows as the codebase reveals patterns worth codifying. New conventions land via a PR that updates both code and this file in the same change.

## When the active milestone closes

Update the **Current state** and **Active milestone** sections of this file in the same PR that ships the last issue of the closing milestone, *or* in a small follow-up PR opened by Web Claude when the next milestone is planned.
