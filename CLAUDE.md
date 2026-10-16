# switchx

Terminal-native time tracker for consulting teams. Workspace-level rules and the collaboration model are in the parent directory's `CLAUDE.md` — this file only contains switchx-specific context.

## Current state

**Phase 1 — minimum viable implementation.**
**M1 — Foundation & Bootstrap: closed.** Verification procedure in [`docs/m1-smoke-test.md`](docs/m1-smoke-test.md); release notes drafted in [`docs/releases/v0.1.md`](docs/releases/v0.1.md).
Active milestone: TBD — next milestone planned by the Systems Analyst in [the switchx project](https://linear.app/taelron/project/switchx-0b0069bd1c04).

This file is updated when the active milestone changes (typically once per milestone closure).

<!-- taelron:workflow:start -->
## Taelron AI Build Workflow

Every Taelron product runs the
[AI Build Workflow](https://linear.app/taelron/document/ai-build-workflow-bedca8de2907),
version 2. It is canonical; this section is a summary. Re-read it at session start,
and where this section and it disagree, it wins. Your role prompt says which steps
are yours.

### Roles

| # | Name            | Runs on                  | Model    | Role                            |
|---|-----------------|--------------------------|----------|---------------------------------|
| 1 | PO              | Human                    | —        | Product decisions, `go`, merge  |
| 2 | Systems Analyst | Claude Desktop           | Opus 5   | Design, Milestone Plan, close   |
| 3 | Orchestrator    | Claude Code, own clone   | Opus 5   | Runs the milestone              |
| 4 | Architect       | Claude Code              | Fable 5  | Issue check, Implementation Plan |
| 5 | Tester          | Claude Code              | Sonnet 5 | Acceptance tests, from the issue |
| 6 | Dev             | Claude Code              | Sonnet 5 | Code, push, draft PR, brief     |
| 7 | Reviewer        | Claude Code              | Opus 5   | Fresh-context review of a diff  |
| 8 | Verifier        | Claude Code              | Sonnet 5 | Drives the built binary         |

2. Owns design, ADRs, the Milestone Plan and its issues, amendments that change
   what the product promises, and the milestone close. Writes Linear.
3. Runs outside the product repository, from Linear and a read-only clone. Starts
   each role, reviews the Implementation Plan, posts the APPROVED PLAN, settles
   technical disputes and conflicts, routes open items with the PO, writes the
   milestone report. Never runs the Architect's model.
8. Fixtures per fleet type. Reports; never fixes.

### Rules for every phase

- **The PO links the two sides only at the boundary of a milestone.** One paste in
  (Systems Analyst → Orchestrator), one paste out (Orchestrator → Systems Analyst).
  Inside a milestone the Orchestrator carries every handoff, and Claude Code
  sessions message each other directly with `SendMessage`, addressed by session
  name (`<product>-<role>`).
- **The PO answers in one pane, the Orchestrator's.** Every question, the plan
  approval, the merge and the routing reach the PO there; the PO types the answer.
- **Promises and design go to the Systems Analyst.** An amendment that adds, removes
  or weakens an acceptance criterion, or a question about a design decision, leaves
  the milestone through the PO: one paste out, one paste back.
- **A handover across the boundary ends with a prompt.** Details go to Linear first,
  then a short, ready-to-paste prompt: what changed, and where to read it.
- **Decisions for the PO are numbered, each with a recommendation and why.** The PO
  answers by number, or `ok go` to apply every recommendation.
- **The PO is asked product questions only:** what the user sees, what the product
  promises, scope, licence, money, anything irreversible. Anything derivable from a
  baseline, a document or the machine is decided by the session that meets it, and
  recorded where the next reader finds it. A disagreement on a fact is settled by
  measuring.
- **If it changes the issue or the plan, it goes on the issue.** The terminal is for
  conversation; Linear is the record.
- **Comments post directly.** Issue checks, decisions, plans, reviews, the APPROVED
  PLAN, briefs, routes, the milestone report, and an amendment that corrects a fact
  or names an interface. A change to what the product promises, a product
  document, a baseline, or a milestone's scope or exit criterion is shown to the PO
  first, by the Systems Analyst.
- **Two rounds, then escalate.** Every loop stops after two rounds without agreement
  and goes to the PO, through the Orchestrator.
- **One ball.** A session starts a step only when it holds the ball; no two
  sessions hold it at once.
- **No PR merges on red CI.** The Orchestrator checks the PR's CI before it asks
  the PO to merge, and `main`'s CI after the merge.
- **Fact or promise: when in doubt, it is a promise.** An amendment whose effect on
  what the product promises is unclear goes to the Systems Analyst.

### Phase rules

- **A · Planning** (PO, Systems Analyst). One issue per milestone unless a named
  reason forces a split. The plan says what, never how. Nothing is written to Linear
  before the PO validates the plan. The milestone is locked once created. The issue
  names the interface its tests rely on, the repository, the issue branch and the
  source of the toolchain versions.
- **B · Implementation planning** (Orchestrator, Architect). The Architect session
  is fresh for each milestone, when the PO starts the layout; between issues the
  Orchestrator's start prompt names the new issue and its sources. The issue check
  is posted before any amendment it argues for; the decisions before the plan. The
  Architect asks product questions and decides technical ones. The plan is a comment
  titled `Implementation Plan, revision <n>`. A promise change goes to the Systems
  Analyst only once the plan review loop has closed. After the PO's `go`, the
  Orchestrator posts `APPROVED PLAN` with the plan in full. No code is written in
  this phase.
- **C · Build** (Orchestrator, Tester, Dev, Reviewer, Verifier). The Tester and Dev
  share one folder and one branch; the Verifier builds from a clean clone; the
  Orchestrator reads its own clone and writes none. The Tester writes tests from the
  issue alone and commits the red run, taken from a clean copy of its commit; it
  passes the APPROVED PLAN link to Dev with "tests are red", unread. A Tester that
  finds a weakness in its own tests later tells the Orchestrator, which holds Dev.
  Dev never modifies an acceptance test: a dispute goes to the Orchestrator. The
  Reviewer gets the branch's diff against `main`, the Tester's commits included,
  and the issue, never the PR. The Verifier gets the branch, the PR URL, the issue
  and the plan, never the PR body; it reports and never fixes. On disagreement the
  acceptance test wins. Nothing is pushed before the tests and the code; the PR is
  a draft carrying the verification handoff (ordered Make targets, purpose,
  expected result). The brief is a comment on the issue and gates the merge. The
  PO verifies, marks the PR ready and merges, once its CI is green.
- **A post-merge fix** is always a new issue in the milestone: the Orchestrator
  creates it once the PO has routed it, the role that owns the files makes the
  change (the Tester for an acceptance test, Dev otherwise), and it is traced
  through Phase C steps 13 to 21.

### Tracing: pick-up first, then the listed events

- **Pick-up rule.** The moment you receive work, from the PO or from another
  session, your first action, before reading or doing anything else, is the `R`
  event, with `--ball` set to yourself:
  `~/.local/bin/taelron-trace --product <product> --milestone <milestone> --issue <issue> --phase <phase> --event R --step <step> --from <you> --ball <you> --next '<what you do now>'`.
  This covers every handoff, including each round of a loop.
- **A started event is the pick-up for its step.** Where your step opens with a
  started event (A·T1, B·T1, C·T1), send it instead of `R`.
- **Listed events.** Send the events your role prompt lists, only once their step
  is complete. `taelron-trace --help` lists every event; it refuses any other.
  Never invent one. `--link` is optional: the PR or Linear URL the event points to.
- **The `next` rule.** `--next` is what the ball-holder does next, in a few words.
- **A refused event is never silent.** When `taelron-trace` cannot send, it prints
  `taelron-trace: event NOT sent: <reason>` on stderr and exits non-zero. Fix the
  cause and send the event again.

### Linear access

Claude Code reads and writes Linear with curl against
`https://api.linear.app/graphql`, never the Linear MCP. Every role that reads
Linear takes the key from the `pass` entry `taelron/linear/api-key` at each call
and passes the header on a file descriptor:
`-H @<(printf 'Authorization: %s\n' "$(pass show taelron/linear/api-key | head -n1)")`.
The key never goes in a file, an exported variable, argv or shell history.
Fetch documents in full with `document(id:)`, which accepts the URL slug; a list
response is truncated. Writes are issue comments and review-gated `spec-drift`
issues; the Orchestrator may also apply an amendment that corrects a fact or names
an interface to the issue itself, and creates the issue of a routed post-merge
fix. The Reviewer has no Linear access, by design.

### Session read order

1. [AI Build Workflow](https://linear.app/taelron/document/ai-build-workflow-bedca8de2907)
   — participants, models, the three phases.
2. [AI Workflow](https://linear.app/taelron/document/ai-workflow-805fb2002fce)
   — surfaces, author capability, the Linear write gate, phased delivery.
3. [Delivery Workflow](https://linear.app/taelron/document/delivery-workflow-edab9d0993e8)
   and whichever other baselines apply to the task.
4. The product documents: Domain Model, ADRs, Specification, Technical Design or
   Design Notes.
5. The issue you were given, its milestone and its Milestone Plan.

Before offering a decision, open the document that would record it.
<!-- taelron:workflow:end -->

## switchx-specific design (Linear — switchx project)

- @README ([Linear](https://linear.app/taelron/document/readme-3b00ab0b67b2)) — what it is, design principles, audience
- @Specification ([Linear](https://linear.app/taelron/document/specification-3645653a2170)) — Phase 1 behavior: use cases, views, validation, report format
- @Domain Model ([Linear](https://linear.app/taelron/document/domain-model-11af391c0064)) — entities, invariants, relationships, the at-most-one-open-session rule
- @ADR-0001 ([Linear](https://linear.app/taelron/document/adr-0001-at-most-one-open-session-per-user-d870088d6e6e)) — At-Most-One-Open-Session Per User
- @ADR-0002 ([Linear](https://linear.app/taelron/document/adr-0002-postgres-for-shared-team-storage-b6b0df46161c)) — Postgres for Shared Team Storage
- @ADR-0003 ([Linear](https://linear.app/taelron/document/adr-0003-adopt-hexagonal-architecture-and-domain-and-persistence-3392ec24d042)) — Adopt Hexagonal Architecture and Domain & Persistence Separation Baselines
- @ADR-0004 ([Linear](https://linear.app/taelron/document/adr-0004-configuration-and-bootstrap-e5000764e4c4)) — Configuration and Bootstrap
- @ADR-0005 ([Linear](https://linear.app/taelron/document/adr-0005-secret-handling-for-switchx-ea21d3f379f4)) — Secret Handling for switchx

## M1 (closed)

**Goal achieved:** A user can install the binary, run it, complete the bootstrap wizard, and reach a placeholder home screen connected to Postgres. No domain features — that's M2+.

**Issues:** TAE-5 through TAE-14, all merged. End-to-end verification: [`docs/m1-smoke-test.md`](docs/m1-smoke-test.md) (manual walkthrough) and `make smoke-integration` (automated against local PG).

**Locked decisions from M1, still in force:**
- PostgreSQL minimum version: **16** (tested against 16 and 17).
- Migrations run on every launch; M1 ships with an empty `internal/storage/postgres/migrations/`.
- Wizard re-runs until the database connection is validated.

## Quick reference

- **Module layout:** `internal/domain/`, `internal/app/`, `internal/storage/`, `internal/ui/tui/`, `internal/storage/postgres/migrations/`, `cmd/switchx/`
- **TUI stack:** Bubble Tea + Lipgloss
- **DB driver:** `pgx/v5` with `pgxpool` (max 4 connections)
- **Migrations:** `golang-migrate`
- **Secret provider (M1):** Azure Key Vault via `az` CLI

## Conventions specific to this repo

None yet beyond what the workspace baselines define. This section grows as the codebase reveals patterns worth codifying. New conventions land via a PR that updates both code and this file in the same change.

## When the active milestone closes

Update the **Current state** and **Active milestone** sections of this file in the same PR that ships the last issue of the closing milestone, *or* in a small follow-up PR opened by the Systems Analyst when the next milestone is planned.
