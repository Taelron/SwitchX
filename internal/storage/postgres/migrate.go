package postgres

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Taelron/SwitchX/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// embeddedMigrations contains every file under the package's
// migrations/ subtree. M1 ships with only a .gitkeep (no .sql files);
// the iofs source surfaces zero migrations to golang-migrate, which
// the runner treats as a no-op.
//
//go:embed all:migrations
var embeddedMigrations embed.FS

// Migrate runs all pending migrations against the database identified
// by cfg/user/password using the embedded migration set. Per @ADR-0002
// (revised 2026-05-02) this is the canonical migration path — no
// separate admin migrate binary in M1; every application start
// re-applies any new migrations.
//
// Migrate opens its own connection (the golang-migrate idiom), so it
// does not contend with our pgxpool. golang-migrate takes a Postgres
// advisory lock for the duration; concurrent startups serialize on it
// (default 30s wait, surfaced as ErrMigrationLockTimeout if exceeded).
//
// Idempotent: with no pending migrations the function returns nil
// silently. Empty migration set is treated identically.
func Migrate(ctx context.Context, cfg config.Database, user, password []byte) error {
	return migrateFromFS(ctx, cfg, user, password, embeddedMigrations)
}

// migrateFromFS is the test-friendly inner. Production calls Migrate
// with the embedded FS; tests pass `os.DirFS("testdata/...")` to
// substitute alternative migration sets.
func migrateFromFS(ctx context.Context, cfg config.Database, user, password []byte, source fs.FS) error {
	// Empty migrations/ (e.g., M1's only-.gitkeep) is a legitimate
	// no-op: iofs.New errors when zero files match the migration
	// naming convention, but the runner spec treats empty as success.
	if !hasMigrationFiles(source) {
		return nil
	}

	src, err := iofs.New(source, "migrations")
	if err != nil {
		return ErrMigrationsFS
	}

	dbURL := buildMigrateURL(cfg, user, password)
	defer zero(dbURL)

	m, err := migrate.NewWithSourceInstance("iofs", src, string(dbURL))
	if err != nil {
		return classifyMigrateInitError(err)
	}
	defer func() { _, _ = m.Close() }()

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			// Non-blocking send: m.GracefulStop is unbuffered, and
			// m.Up() may have already returned (ErrNoChange path, or
			// finished before ctx fired) without ever receiving. A
			// blocking send would leak this goroutine.
			select {
			case m.GracefulStop <- true:
			default:
			}
		case <-stop:
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return classifyMigrateRunError(err)
	}
	return nil
}

// buildMigrateURL renders cfg + credentials into the URL form that
// golang-migrate's pgx/v5 driver expects:
//
//	pgx5://<user>:<password>@<host>:<port>/<db>?sslmode=<mode>
//
// net/url percent-encodes any special characters in user/password.
// Returned as []byte so the caller can scrub it.
func buildMigrateURL(cfg config.Database, user, password []byte) []byte {
	u := url.URL{
		Scheme:   "pgx5",
		User:     url.UserPassword(string(user), string(password)),
		Host:     net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:     "/" + cfg.Name,
		RawQuery: "sslmode=" + url.QueryEscape(cfg.SSLMode),
	}
	return []byte(u.String())
}

// hasMigrationFiles returns true when the source's `migrations`
// subdirectory contains at least one .sql file. iofs.New rejects an
// empty migrations dir, but TAE-9 says empty is valid (M1 ships with
// no migrations); we short-circuit here so the caller stays simple.
func hasMigrationFiles(source fs.FS) bool {
	entries, err := fs.ReadDir(source, "migrations")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sql") {
			return true
		}
	}
	return false
}

// classifyMigrateInitError covers the NewWithSourceInstance phase:
// connecting to the DB, parsing the URL, etc. Most failures here are
// connection-level — reuse the connection-level classifier first.
func classifyMigrateInitError(err error) error {
	if c := classifyError(err); !errors.Is(c, ErrUnknown) {
		return c
	}
	return ErrMigrationFailed
}

// classifyMigrateRunError covers the m.Up() phase. Maps the three
// categories the wizard / error screen care about. Raw migrate text
// is never embedded in the returned error — same defense posture as
// classifyError.
func classifyMigrateRunError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "lock timeout"),
		strings.Contains(msg, "could not obtain lock"):
		return ErrMigrationLockTimeout

	case errors.Is(err, fs.ErrNotExist),
		strings.Contains(msg, "no such file"),
		strings.Contains(msg, "file does not exist"):
		return ErrMigrationsFS
	}

	return ErrMigrationFailed
}
