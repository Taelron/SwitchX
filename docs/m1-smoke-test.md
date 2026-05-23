# M1 smoke test

End-to-end verification procedure for the M1 milestone — Foundation &
Bootstrap. Walk these nine steps on a clean Linux machine (or a fresh
`$XDG_CONFIG_HOME` directory) to confirm the operational spine:
config → secret → database → migrations → TUI shell.

Each step cites the exact `make` target that drives it where one
exists. Automated coverage of steps 2, 5, 7, 8, 9 lives in
`TestM1E2E` (run via `make smoke-integration`); steps 3, 4, 6 are
covered by unit tests in `internal/ui/tui/bootstrap/`. Step 1 is
manual only — see the note in that section.

## Prerequisites

Before starting:

1. A local Postgres reachable at `localhost:5432`, with a role
   matching your Linux username that holds `CREATEDB` and
   `CREATEROLE`.
2. `az login` has produced an Azure session that can read the
   developer Key Vault named by `AZURE_VAULT` in `.local/.env`
   (see `.local/.env.example` for the template).
3. `make dev-up` has run successfully — provisions the `switchx_user`
   role + `switchx` database against your local PG **and** writes
   `.local/dev-config/switchx/config.toml` (the dev XDG config the
   `make` targets use). The deployed binary's `~/.config/switchx/`
   is independent and not touched by this flow.

## Steps

### Step 1 — `go install` the binary

```
go install ./cmd/switchx
```

The command must exit 0. `which switchx` should resolve to a binary
under `$GOPATH/bin` (or `$GOBIN`). Run `switchx --help` to confirm
the binary itself works.

**Why manual-only:** automating `go install` adds nothing on top of
`go build`, which `make test` and `make verify` already cover. The
purpose of step 1 is to confirm that an end-user invocation produces
a usable binary on `$PATH` — an environment-level concern outside
the test harness.

### Step 2 — Run with no config: wizard appears

```
make startup-test-no-config
```

The script points `XDG_CONFIG_HOME` at a freshly-wiped temporary
directory and launches `switchx`. The bootstrap wizard must open
on step 1 (database).

Automated equivalent: `TestM1E2E/step2_no_config_triggers_wizard`
asserts `config.Load()` returns `errors.Is(_, ErrNoConfig)`, which
is the sentinel that drives `run()` into the wizard branch.

### Step 3 — Wizard step 1: enter database coordinates

In the wizard:

- Host: `localhost`
- Port: `5432`
- Name: `switchx`
- SSL mode: `disable`

Press Enter on the last field. The wizard advances to step 2.

Automated equivalent: `internal/ui/tui/bootstrap/model_test.go` —
`TestStep1AdvanceWritesDatabaseFields` and related step-1 tests
cover every field, validation path, and advance condition.

### Step 4 — Wizard step 2: secret references

- Provider: `azurekeyvault` (pre-filled)
- Subscription: the value `az account show --query name -o tsv` prints
- Vault: your developer KV name
- User ref: `SWITCHX-DB-USER`
- Password ref: `SWITCHX-DB-PASSWORD`

Press Enter on the last field. The wizard advances to step 3.

Automated equivalent: `bootstrap/model_test.go::TestStep2*` covers
the field-by-field advance and validation.

### Step 5 — Wizard step 3: timezone

Pick `Europe/Brussels` (or your IANA TZ) with the arrow keys.
Press Enter. The wizard advances to step 4.

There is no "display name" field — switchx uses your Linux username
directly for `Session.created_by` attribution.

Automated equivalent: `bootstrap/model_test.go::TestStep3*`.

### Step 6 — Wizard step 4 + validate: connection confirmed

Pick `vi` (or your editor) with the arrow keys. Press Enter.

The wizard transitions to the "Validating…" state. It performs the
full chain: resolve the secret references via `az`, open a Postgres
connection with the resolved credentials, health-check it, save the
config to `$XDG_CONFIG_HOME/switchx/config.toml` at mode `0600`, and
quit.

Automated equivalents:

- `bootstrap/bootstrap_test.go::TestEndToEndPersist` — exercises the
  validator + persist path including the on-disk mode-0600 check
  and round-trip through `config.Load`.
- `TestM1E2E/step5_startuponce_validates_db_and_runs_migrations` —
  exercises the same resolve→connect→health→migrate chain against
  a live database via `make smoke-integration`.

### Step 7 — Home screen appears

After the wizard quits, the binary continues into the main flow.
Migrations run cleanly (a no-op against M1's empty `migrations/`
directory). The placeholder home screen renders:

```
switchx › home

       Connected to switchx

q/Quit  ?/Help
```

Press `q` (or Ctrl+C) to exit.

Automated equivalents:

- `internal/ui/tui/home/model_test.go` — golden snapshots for
  every rendered variant of the home model.
- `TestM1E2E/step7_runhome_receives_correct_dbname` — asserts the
  database name from the persisted config threads through
  `runStartupWith` into `runHome`.

### Step 8 — Re-run: wizard skipped, home loads immediately

```
make bootstrap-test-no-wizard
```

With the config already on disk, run the binary again. The wizard
must NOT appear; the home screen renders directly.

Automated equivalent: `TestM1E2E/step8_rerun_skips_wizard_path`
asserts that a second `config.Load()` succeeds (no
wizard-triggering sentinel) and that the post-wizard chain runs
cleanly on the second invocation.

### Step 9 — Re-run with DB stopped: error screen

```
make startup-test-db-down
```

The script stops the local Postgres service, then launches `switchx`.
The error screen — not the wizard — must appear, showing:

> Database refused connection. The role name or password may be
> wrong, or the role may not have CONNECT privilege.
>
> r/Retry  q/Quit

Then verify the retry path recovers:

```
make startup-test-retry-recovers
```

The script tells you when to start Postgres in a separate terminal,
then press `r` on the error screen. The home screen appears.

Automated equivalent:
`TestM1E2E/step9_db_down_then_retry_recovers` exercises the same
`runStartupWith` retry loop using a `connectFn` that fails once and
succeeds on retry; asserts the error screen was shown exactly once
and the home screen received the correct database name.

## Make targets referenced

| Target | Purpose |
|---|---|
| `make dev-up` | Provision local PG role + database (prerequisite, one-time per machine) |
| `make startup-test-no-config` | Manual smoke for step 2 |
| `make bootstrap-test-no-wizard` | Manual smoke for steps 7, 8 |
| `make startup-test-db-down` | Manual smoke for step 9 (error screen) |
| `make startup-test-retry-recovers` | Manual smoke for step 9 (retry path) |
| `make pg-start` | Start the local Postgres service (used during step 9 retry scenario) |
| `make pg-stop` | Stop the local Postgres service (used internally by `startup-test-db-down`) |
| `make smoke-integration` | Automated end-to-end (`TestM1E2E`) — covers steps 2, 5, 7, 8, 9 against the live local PG |
| `make test` | Unit suite — covers steps 3, 4, 5 (wizard internals), 6 (validate), 7 (home rendering) |
