// Startup decision tree (TAE-13).
//
// `run()` (in main.go) calls runStartup once the config is available,
// either freshly loaded from disk or produced by the bootstrap wizard.
// runStartup owns the resolve → pool → health → migrate → home chain
// and the user-driven retry loop:
//
//   loop:
//     startupOnce ──┬─► nil   ─► home.Run        (defer closer)
//                   │
//                   └─► err   ─► errscreen.Run
//                                  │
//                                  ├─► DecisionRetry ─► loop again
//                                  └─► DecisionQuit  ─► return
//
// startupOnce takes the post-wizard work as two injectable function
// types (connectFn, migrateFn). Production wires postgres.Connect +
// postgres.HealthCheck + postgres.Migrate; tests inject fakes so the
// retry path is exercisable without a live database.
//
// Error translation lives in this file because the consumer (the
// error screen) cannot import postgres or secrets without dragging
// storage/port packages into the TUI tree. Keeping translateErr in
// package main preserves the import graph the layout enforces.

package main

import (
	"context"
	"errors"

	"github.com/Taelron/SwitchX/internal/app/secrets"
	"github.com/Taelron/SwitchX/internal/config"
	"github.com/Taelron/SwitchX/internal/storage/postgres"
	"github.com/Taelron/SwitchX/internal/ui/tui/errscreen"
	"github.com/Taelron/SwitchX/internal/ui/tui/home"
)

// connectFn opens a Postgres connection, verifies it with a round-trip
// health check, and returns a closer the caller can defer. Collapsing
// open + health-check behind one seam lets tests fake the whole
// "connect to a live database" step with a stub closer instead of
// constructing a real *pgxpool.Pool.
type connectFn func(ctx context.Context, cfg config.Database, user, password []byte) (close func(), err error)

// migrateFn matches postgres.Migrate's signature so the production
// wiring is a direct function-value assignment with no adapter.
type migrateFn func(ctx context.Context, cfg config.Database, user, password []byte) error

// realConnect is the production connectFn: open the pool, ping it,
// return pool.Close as the closer. On any failure the pool is closed
// before returning so the caller never owns a half-open pool.
func realConnect(ctx context.Context, cfg config.Database, user, password []byte) (func(), error) {
	pool, err := postgres.Connect(ctx, cfg, user, password)
	if err != nil {
		return nil, err
	}
	if err := postgres.HealthCheck(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool.Close, nil
}

// runStartup drives the retry loop. It returns nil for any clean exit
// (success path that home.Run quit cleanly, user pressing q on the
// error screen, SIGINT during any phase), and a wrapped error only
// for an unrecoverable tea runtime or home runtime failure.
func runStartup(ctx context.Context, cfg *config.Config, provider secrets.Provider) error {
	return runStartupWith(ctx, cfg, provider, realConnect, postgres.Migrate, home.Run, errscreen.Run)
}

// showErrorFn is the seam errscreen.Run plugs into. Tests inject a
// closure that returns a Decision synchronously so the retry loop's
// wiring is exercisable without booting a real tea.Program.
type showErrorFn func(ctx context.Context, msg string) (errscreen.Decision, error)

// runStartupWith is the test-injectable inner. Production calls it via
// runStartup with the real wiring; tests pass fakes for connect,
// migrate, runHome, and showError so every branch of the retry loop
// is reachable without a TTY or a live database.
func runStartupWith(
	ctx context.Context,
	cfg *config.Config,
	provider secrets.Provider,
	connect connectFn,
	migrate migrateFn,
	runHome func(ctx context.Context, dbname string) error,
	showError showErrorFn,
) error {
	for {
		closer, err := startupOnce(ctx, cfg, provider, connect, migrate)
		if err == nil {
			// defer is intentional inside the loop body: the success
			// branch returns immediately after this line, so the defer
			// is effectively function-scoped and fires exactly once
			// after runHome unblocks. We keep defer (instead of an
			// explicit closer() call after runHome) for panic safety
			// — a panic inside runHome still releases the pool.
			defer closer()
			return runHome(ctx, cfg.Database.Name)
		}
		if errors.Is(err, context.Canceled) {
			return nil
		}
		decision, rerr := showError(ctx, translateErr(err))
		if rerr != nil {
			return rerr
		}
		if decision == errscreen.DecisionQuit {
			return nil
		}
		// DecisionRetry: loop. cfg, provider, connect, migrate are
		// reused as-is; each iteration re-fetches the secrets.
	}
}

// startupOnce runs a single pass through the post-wizard chain:
// resolve user → resolve password → connect+health → migrate. On
// success it returns a closer the caller defers (typically the live
// pool's Close). On failure it scrubs any secret buffer it allocated
// and returns the error tagged with the secret source where
// applicable, so translateErr can name which ref the user should fix.
func startupOnce(
	ctx context.Context,
	cfg *config.Config,
	provider secrets.Provider,
	connect connectFn,
	migrate migrateFn,
) (func(), error) {
	user, err := provider.GetSecret(ctx, secrets.Ref{
		Subscription: cfg.Database.Secret.Subscription,
		Vault:        cfg.Database.Secret.Vault,
		Name:         cfg.Database.Secret.UserRef,
	})
	if err != nil {
		return nil, &secretSourceError{which: "user_ref", err: err}
	}
	defer zero(user)

	password, err := provider.GetSecret(ctx, secrets.Ref{
		Subscription: cfg.Database.Secret.Subscription,
		Vault:        cfg.Database.Secret.Vault,
		Name:         cfg.Database.Secret.PasswordRef,
	})
	if err != nil {
		return nil, &secretSourceError{which: "password_ref", err: err}
	}
	defer zero(password)

	closer, err := connect(ctx, cfg.Database, user, password)
	if err != nil {
		return nil, err
	}

	if err := migrate(ctx, cfg.Database, user, password); err != nil {
		closer()
		return nil, err
	}

	return closer, nil
}

// secretSourceError marks a secret-store sentinel with which of the
// two secrets (user_ref or password_ref) triggered it. ADR-0005's
// "Failure modes" requires the error to name which; this wrapper
// preserves the sentinel for errors.Is while carrying the source for
// translateErr to render.
type secretSourceError struct {
	which string
	err   error
}

func (e *secretSourceError) Error() string { return e.which + ": " + e.err.Error() }
func (e *secretSourceError) Unwrap() error { return e.err }

// translateErr maps a startup sentinel to the user-readable message
// rendered on the error screen. Where ADR-0005 §"Failure modes"
// specifies a template, the wording is adopted verbatim; the
// `<pg error>` interpolation from the ADR is omitted because TAE-8
// strips raw pg text from sentinels to avoid leaking connection-string
// material. That divergence is documented in the TAE-13 plan; the ADR
// will be updated post-merge.
//
// The fallthrough deliberately does NOT embed err.Error(): the
// upstream contract (postgres.classifyError + classifyMigrateInitError
// + classifyMigrateRunError + the secrets adapter) ensures every
// error reaching this point is a named sentinel, but pgconn's raw
// errors can contain the password via their Config — so an
// unrecognised error is by definition outside that guarantee and
// unsafe to render verbatim. New sentinels must be added to the
// switch above; the fallthrough is a defense-in-depth catch only.
func translateErr(err error) string {
	which := ""
	var src *secretSourceError
	if errors.As(err, &src) {
		which = " (" + src.which + ")"
	}

	switch {
	// Secret-store sentinels.
	case errors.Is(err, secrets.ErrCLINotFound):
		return "Secret store: az CLI not found. Install the Azure CLI and try again."
	case errors.Is(err, secrets.ErrNotLoggedIn):
		return "Secret store: not logged in to Azure. Run `az login` and retry."
	case errors.Is(err, secrets.ErrSubscriptionAccess):
		return "Cannot access subscription. Confirm `az account list` includes it."
	case errors.Is(err, secrets.ErrVaultNotFound):
		return "Secret store: vault not found. Check the vault name in your config."
	case errors.Is(err, secrets.ErrSecretNotFound):
		return "Secret store: secret not found" + which + ". Check the reference name in your config."
	case errors.Is(err, secrets.ErrEmptyValue):
		return "Secret store: secret has an empty value" + which + ". Update the secret in Key Vault."
	case errors.Is(err, secrets.ErrNetwork):
		return "Secret store: network failure reaching Key Vault. Check connectivity and retry."
	case errors.Is(err, secrets.ErrInvalidRef):
		return "Secret store: invalid secret reference" + which + ". Re-run the wizard to correct it."
	case errors.Is(err, secrets.ErrUnexpected):
		return "Secret store: unexpected error. Check `az` CLI output for details."

	// Database connect / health sentinels. ConnRefused and AuthFailed
	// collapse into the ADR-0005 sentence; the underlying cause is
	// indistinguishable to the user and the same recovery hint applies.
	case errors.Is(err, postgres.ErrConnRefused),
		errors.Is(err, postgres.ErrAuthFailed):
		return "Database refused connection. The role name or password may be wrong, or the role may not have CONNECT privilege."
	case errors.Is(err, postgres.ErrDBNotFound):
		return "Database not found. Check the database name in your config."
	case errors.Is(err, postgres.ErrTLSHandshake):
		return "Database TLS handshake failed. Check sslmode and the server's certificate."
	case errors.Is(err, postgres.ErrUnreachable):
		return "Database host unreachable. Check the host and port in your config and your network."
	case errors.Is(err, postgres.ErrUnknown):
		return "Database connection failed. Check Postgres logs for details."

	// Migration sentinels.
	case errors.Is(err, postgres.ErrMigrationsFS):
		return "Migrations: embedded files are missing. This is a build error — reinstall switchx."
	case errors.Is(err, postgres.ErrMigrationLockTimeout):
		return "Migrations: could not acquire lock. Another switchx process may be running."
	case errors.Is(err, postgres.ErrMigrationFailed):
		return "Migrations: a migration failed. Check the database permissions and retry."
	}

	return "Startup failed with an unrecognised error. Please file an issue."
}
