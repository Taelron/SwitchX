// Package postgres is the storage adapter for switchx's Postgres
// backing store. It exposes the connection-pool lifecycle and a
// minimal health check; domain repositories will be added in M2+.
//
// Per @ADR-0002 the pool is sized to MaxConns=4 (the M1 cap). Per
// @ADR-0005 / @Security & Secret Handling the user and password
// arrive as []byte from the secrets resolver and are scrubbed
// before Connect returns; pgx owns its own internal copy from there.
package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Taelron/SwitchX/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxConns is the per-process Postgres connection cap for switchx M1.
const MaxConns = 4

// Connect builds a libpq connection string from cfg + user + password,
// hands it to pgxpool with MaxConns=4, and returns the live pool.
//
// The user and password []byte buffers are scrubbed before Connect
// returns — pgxpool has made its own internal copy of the connection
// string by then. The caller's defer-zero of the same buffers is still
// valuable as a second layer of defense.
func Connect(ctx context.Context, cfg config.Database, user, password []byte) (*pgxpool.Pool, error) {
	connStr := buildConnString(cfg, user, password)
	defer zero(connStr)

	poolCfg, err := pgxpool.ParseConfig(string(connStr))
	if err != nil {
		return nil, fmt.Errorf("storage: parse connection string: %w", err)
	}
	poolCfg.MaxConns = MaxConns

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, classifyError(err)
	}

	// pgxpool.NewWithConfig is lazy — it does not open a connection.
	// Force one round-trip so the caller learns about auth / TLS /
	// missing-database failures here, not on the first query.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%w (host=%s db=%s)",
			classifyError(err), cfg.Host, cfg.Name)
	}

	return pool, nil
}

// HealthCheck runs a trivial query against the pool to verify the
// connection is live. Used by the bootstrap flow (TAE-13) and any
// future readiness probes.
func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("storage: nil pool")
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&n); err != nil {
		return classifyError(err)
	}
	if n != 1 {
		return fmt.Errorf("storage: health check returned %d, want 1", n)
	}
	return nil
}

// buildConnString assembles the libpq key=value form. Returning a
// []byte (not string) lets the caller scrub the buffer before pgxpool
// makes its own copy.
//
// Per the libpq documentation, values containing whitespace, single
// quotes, or backslashes must be single-quoted with backslash-escaping
// of '\' and '\”. We always quote user and password defensively —
// they're under operator control via the KV secrets and may rotate to
// values with special characters.
func buildConnString(cfg config.Database, user, password []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(256)
	buf.WriteString("host=")
	buf.WriteString(cfg.Host)
	buf.WriteString(" port=")
	buf.WriteString(strconv.Itoa(cfg.Port))
	buf.WriteString(" dbname=")
	buf.WriteString(cfg.Name)
	buf.WriteString(" sslmode=")
	buf.WriteString(cfg.SSLMode)
	buf.WriteString(" user=")
	buf.WriteString(libpqQuote(user))
	buf.WriteString(" password=")
	buf.WriteString(libpqQuote(password))
	return buf.Bytes()
}

// libpqQuote wraps a value in single quotes per the libpq connection-
// string format, escaping embedded backslashes and single quotes.
func libpqQuote(b []byte) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(string(b)) + "'"
}

// zero overwrites a byte slice with zeros.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
