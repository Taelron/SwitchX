package postgres

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/Taelron/SwitchX/internal/config"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBuildConnString_Shape(t *testing.T) {
	cfg := config.Database{
		Host:    "db.example",
		Port:    5432,
		Name:    "switchx",
		SSLMode: "require",
	}
	got := string(buildConnString(cfg, []byte("alice"), []byte("p@ss")))
	for _, want := range []string{
		"host=db.example",
		"port=5432",
		"dbname=switchx",
		"sslmode=require",
		"user='alice'",
		"password='p@ss'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("connstring missing %q; got: %s", want, got)
		}
	}
}

// libpq requires values containing whitespace, single quotes, or
// backslashes to be single-quoted with embedded ' and \ escaped.
// Verify that user/password go through the quoter, so a future password
// rotation to a value with special characters does not break Connect.
func TestBuildConnString_SpecialCharsQuoted(t *testing.T) {
	cfg := config.Database{Host: "h", Port: 1, Name: "n", SSLMode: "disable"}
	got := string(buildConnString(cfg, []byte(`bob`), []byte(`p ' \ s`)))
	if !strings.Contains(got, `user='bob'`) {
		t.Errorf("user not quoted; got: %s", got)
	}
	// Single quote → \' ; backslash → \\
	if !strings.Contains(got, `password='p \' \\ s'`) {
		t.Errorf("password not properly escaped; got: %s", got)
	}
	// Double-check the resulting string parses with pgx itself.
	if _, err := pgxpool.ParseConfig(got); err != nil {
		t.Errorf("pgxpool rejected our connstring: %v", err)
	}
}

func TestBuildConnString_OrderingIsStable(t *testing.T) {
	cfg := config.Database{Host: "h", Port: 1, Name: "n", SSLMode: "disable"}
	a := string(buildConnString(cfg, []byte("u"), []byte("p")))
	b := string(buildConnString(cfg, []byte("u"), []byte("p")))
	if a != b {
		t.Errorf("non-deterministic connstring:\n a=%s\n b=%s", a, b)
	}
}

func TestClassifyError_Nil(t *testing.T) {
	if err := classifyError(nil); err != nil {
		t.Errorf("nil → %v, want nil", err)
	}
}

func TestClassifyError_PgErrors(t *testing.T) {
	cases := []struct {
		code string
		want error
	}{
		{"28P01", ErrAuthFailed},
		{"28000", ErrAuthFailed},
		{"3D000", ErrDBNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			err := &pgconn.PgError{Code: tc.code, Message: "from PG"}
			if got := classifyError(err); !errors.Is(got, tc.want) {
				t.Errorf("code %s → %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}

func TestClassifyError_NetOpRefused(t *testing.T) {
	err := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("connection refused"),
	}
	if got := classifyError(err); !errors.Is(got, ErrConnRefused) {
		t.Errorf("got %v, want ErrConnRefused", got)
	}
}

func TestClassifyError_NetOpOther(t *testing.T) {
	err := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("permission denied"),
	}
	if got := classifyError(err); !errors.Is(got, ErrUnreachable) {
		t.Errorf("got %v, want ErrUnreachable", got)
	}
}

func TestClassifyError_TextHeuristics(t *testing.T) {
	cases := []struct {
		name string
		text string
		want error
	}{
		{"refused_text", "dial tcp 1.2.3.4:5432: connection refused", ErrConnRefused},
		{"no_such_host", "lookup nope.invalid: no such host", ErrUnreachable},
		{"no_route", "dial tcp: no route to host", ErrUnreachable},
		{"tls_handshake", "tls: handshake failure", ErrTLSHandshake},
		{"tls_certificate", "x509: certificate signed by unknown authority", ErrTLSHandshake},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := errors.New(tc.text)
			if got := classifyError(err); !errors.Is(got, tc.want) {
				t.Errorf("%q → %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

// Critical security guarantee: an unclassified driver error must NOT
// be returned verbatim — pgconn errors can embed the connection config
// (including the password) in their .Error() string. classifyError
// must collapse the unknown case to ErrUnknown so callers cannot
// inadvertently propagate raw driver text.
func TestClassifyError_UnknownIsSwallowed(t *testing.T) {
	const leak = "password=hunter2-DO-NOT-LOG"
	orig := errors.New("some pgconn error: " + leak)
	got := classifyError(orig)
	if !errors.Is(got, ErrUnknown) {
		t.Fatalf("got %v, want ErrUnknown", got)
	}
	if strings.Contains(got.Error(), leak) {
		t.Errorf("ErrUnknown leaked driver text: %q", got.Error())
	}
}
