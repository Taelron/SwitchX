package postgres

import (
	"errors"
	"net"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// Sentinel errors returned by Connect and HealthCheck. The wizard
// (TAE-11) and the startup error screen (TAE-13) consume these via
// errors.Is to decide which user-readable message to show.
//
// Raw driver text is never embedded in the returned error — only
// sentinel + non-sensitive context (host, database name). This is the
// same defense-in-depth posture used by the TAE-7 secret resolver.
var (
	ErrConnRefused  = errors.New("storage: connection refused")
	ErrAuthFailed   = errors.New("storage: authentication failed")
	ErrDBNotFound   = errors.New("storage: database not found")
	ErrTLSHandshake = errors.New("storage: TLS handshake failed")
	ErrUnreachable  = errors.New("storage: host unreachable")
	// ErrUnknown wraps any driver error that does not match the named
	// categories above. The wizard / error screen treats it as a
	// generic "could not connect" condition. The unclassified driver
	// error is intentionally NOT embedded — pgconn errors can include
	// password material via their Config, and the @Security & Secret
	// Handling baseline forbids any path that could leak it.
	ErrUnknown = errors.New("storage: connect failed")

	// Migration sentinels (TAE-9). Surface only the category — the
	// wrapped golang-migrate text is not embedded for the same
	// secret-leakage reason as ErrUnknown.
	ErrMigrationsFS         = errors.New("storage: migrations source unavailable")
	ErrMigrationLockTimeout = errors.New("storage: migration lock timeout")
	ErrMigrationFailed      = errors.New("storage: migration failed")
)

// classifyError maps a driver-level error into one of the storage
// sentinels. Anything unclassified collapses to ErrUnknown; the raw
// driver text is never returned because pgconn errors can embed the
// connection config (including the password) in their .Error() string.
func classifyError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		// 28P01 invalid_password, 28000 invalid_authorization_specification.
		case "28P01", "28000":
			return ErrAuthFailed
		// 3D000 invalid_catalog_name (= database does not exist).
		case "3D000":
			return ErrDBNotFound
		}
	}

	// pgconn surfaces TLS failures wrapped in its connect-error type;
	// the underlying type is from crypto/tls. Sniff the error chain.
	if isTLSError(err) {
		return ErrTLSHandshake
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// "connection refused" → server is down or wrong port.
		if strings.Contains(opErr.Error(), "connection refused") {
			return ErrConnRefused
		}
		// Anything else at the net layer (DNS, no route, EHOSTUNREACH).
		return ErrUnreachable
	}

	// pgconn wraps DNS lookup failures in its own type; match on text.
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection refused"):
		return ErrConnRefused
	case strings.Contains(msg, "no such host"),
		strings.Contains(msg, "host is unreachable"),
		strings.Contains(msg, "no route to host"):
		return ErrUnreachable
	}

	return ErrUnknown
}

// isTLSError walks the error chain looking for a crypto/tls or x509
// failure. The "tls:" and "x509:" prefixes are crypto/tls/x509's own
// formatting; matching the prefix avoids false positives from words
// like "certificate" appearing in unrelated messages.
func isTLSError(err error) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		s := strings.ToLower(e.Error())
		if strings.Contains(s, "tls:") || strings.Contains(s, "x509:") {
			return true
		}
	}
	return false
}
