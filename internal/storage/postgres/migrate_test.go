package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/Taelron/SwitchX/internal/config"
)

func TestBuildMigrateURL_Shape(t *testing.T) {
	cfg := config.Database{
		Host:    "db.example",
		Port:    5432,
		Name:    "switchx",
		SSLMode: "require",
	}
	got := string(buildMigrateURL(cfg, []byte("alice"), []byte("p@ss")))
	for _, want := range []string{
		"pgx5://",
		"alice:p%40ss@db.example:5432",
		"/switchx",
		"sslmode=require",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("URL missing %q; got: %s", want, got)
		}
	}
}

// Special characters that net/url percent-encodes correctly. The libpq
// quoting test in postgres_test.go covers the key=value format; this
// covers the URL form used by golang-migrate's pgx/v5 driver.
func TestBuildMigrateURL_SpecialChars(t *testing.T) {
	cfg := config.Database{Host: "h", Port: 1, Name: "n", SSLMode: "disable"}
	got := string(buildMigrateURL(cfg, []byte("bob"), []byte(`p ' \ s`)))
	// Single quote → %27, space → %20, backslash → %5C
	if !strings.Contains(got, "bob:") {
		t.Errorf("user not in URL; got: %s", got)
	}
	if !strings.Contains(got, "%27") || !strings.Contains(got, "%20") || !strings.Contains(got, "%5C") {
		t.Errorf("password not percent-encoded; got: %s", got)
	}
}

func TestClassifyMigrateRunError_LockTimeout(t *testing.T) {
	cases := []string{
		"timeout: lock timeout exceeded",
		"could not obtain lock on database",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			err := errors.New(msg)
			if got := classifyMigrateRunError(err); !errors.Is(got, ErrMigrationLockTimeout) {
				t.Errorf("got %v, want ErrMigrationLockTimeout", got)
			}
		})
	}
}

func TestClassifyMigrateRunError_FS(t *testing.T) {
	cases := []string{
		"open migrations/0001.up.sql: no such file or directory",
		"file does not exist: 0002.up.sql",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			err := errors.New(msg)
			if got := classifyMigrateRunError(err); !errors.Is(got, ErrMigrationsFS) {
				t.Errorf("got %v, want ErrMigrationsFS", got)
			}
		})
	}
}

func TestClassifyMigrateRunError_Generic(t *testing.T) {
	err := errors.New("syntax error at or near \"BANANAS\"")
	if got := classifyMigrateRunError(err); !errors.Is(got, ErrMigrationFailed) {
		t.Errorf("got %v, want ErrMigrationFailed", got)
	}
}

// The same defense posture as TAE-7: an unclassified migration error
// must not leak its raw text in the returned error, since pgconn /
// migrate strings can include the connection URL (with password).
func TestClassifyMigrateRunError_NoLeakage(t *testing.T) {
	const leak = "pgx5://switchx_user:hunter2-DO-NOT-LOG@localhost"
	err := errors.New("driver hit an unexpected error: " + leak)
	got := classifyMigrateRunError(err)
	if strings.Contains(got.Error(), leak) {
		t.Errorf("error leaked driver URL: %q", got.Error())
	}
}
