package main

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Taelron/SwitchX/internal/app/secrets"
	"github.com/Taelron/SwitchX/internal/config"
	"github.com/Taelron/SwitchX/internal/storage/postgres"
	"github.com/Taelron/SwitchX/internal/ui/tui/errscreen"
)

// fakeProvider implements secrets.Provider with values keyed off the
// secret reference Name. Two seams matter for the retry tests:
// callCount tracks how often GetSecret was called (used to assert
// re-fetch on retry) and the err field can be flipped between
// iterations (used to drive the recovery path in
// TestRunStartupWith_RetryRecovers).
type fakeProvider struct {
	mu        sync.Mutex
	userRef   string
	userVal   []byte
	passRef   string
	passVal   []byte
	err       error
	callCount int
}

func (f *fakeProvider) GetSecret(_ context.Context, ref secrets.Ref) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callCount++
	if f.err != nil {
		return nil, f.err
	}
	switch ref.Name {
	case f.userRef:
		// Hand the caller a fresh copy so its defer-zero does not
		// clobber the provider's stored value across iterations.
		return append([]byte(nil), f.userVal...), nil
	case f.passRef:
		return append([]byte(nil), f.passVal...), nil
	}
	return nil, errors.New("fakeProvider: unknown ref " + ref.Name)
}

// fixtureCfg returns a minimal cfg pointing at well-known refs so the
// fakeProvider can be tied to it without per-test setup.
func fixtureCfg() *config.Config {
	return &config.Config{
		Database: config.Database{
			Host:    "localhost",
			Port:    5432,
			Name:    "switchx_test",
			SSLMode: "disable",
			Secret: config.Secret{
				Provider:     "azure_key_vault",
				Subscription: "TEST-SUB",
				Vault:        "TEST-VAULT",
				UserRef:      "SWITCHX-DB-USER",
				PasswordRef:  "SWITCHX-DB-PASSWORD",
			},
		},
	}
}

func goodProvider() *fakeProvider {
	return &fakeProvider{
		userRef: "SWITCHX-DB-USER",
		userVal: []byte("switchx_user"),
		passRef: "SWITCHX-DB-PASSWORD",
		passVal: []byte("hunter2"),
	}
}

// noopMigrate is a migrateFn that always succeeds. The migration step
// is independently tested in internal/storage/postgres; runStartup
// tests only need to confirm it is invoked at the right place.
func noopMigrate(_ context.Context, _ config.Database, _, _ []byte) error {
	return nil
}

// stubCloser is the closer returned by fake connectFns on success. It
// records how many times it was called so tests can verify pool
// lifecycle.
type stubCloser struct{ count int }

func (s *stubCloser) close() { s.count++ }

// --- translateErr ---------------------------------------------------

// TestTranslateErr_AllSentinels asserts every named sentinel maps to
// a non-empty, non-Go-flavoured user message. The fallthrough is
// covered separately to make sure novel errors don't panic the
// translator.
func TestTranslateErr_AllSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"ErrCLINotFound", secrets.ErrCLINotFound},
		{"ErrNotLoggedIn", secrets.ErrNotLoggedIn},
		{"ErrSubscriptionAccess", secrets.ErrSubscriptionAccess},
		{"ErrVaultNotFound", secrets.ErrVaultNotFound},
		{"ErrSecretNotFound", secrets.ErrSecretNotFound},
		{"ErrEmptyValue", secrets.ErrEmptyValue},
		{"ErrNetwork", secrets.ErrNetwork},
		{"ErrInvalidRef", secrets.ErrInvalidRef},
		{"ErrUnexpected", secrets.ErrUnexpected},
		{"ErrConnRefused", postgres.ErrConnRefused},
		{"ErrAuthFailed", postgres.ErrAuthFailed},
		{"ErrDBNotFound", postgres.ErrDBNotFound},
		{"ErrTLSHandshake", postgres.ErrTLSHandshake},
		{"ErrUnreachable", postgres.ErrUnreachable},
		{"ErrUnknown", postgres.ErrUnknown},
		{"ErrMigrationsFS", postgres.ErrMigrationsFS},
		{"ErrMigrationLockTimeout", postgres.ErrMigrationLockTimeout},
		{"ErrMigrationFailed", postgres.ErrMigrationFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := translateErr(tc.err)
			if got == "" {
				t.Fatal("translateErr returned empty string")
			}
			if strings.Contains(got, "storage:") || strings.Contains(got, "secrets:") {
				t.Errorf("translateErr leaked internal sentinel prefix: %q", got)
			}
		})
	}
}

// TestTranslateErr_ConnRefusedAndAuthFailedCollapse verifies the
// ADR-0005 §Failure-modes collapse: both sentinels map to the same
// user-facing message. The plan's Linear comment documents the
// decision; this test pins it.
func TestTranslateErr_ConnRefusedAndAuthFailedCollapse(t *testing.T) {
	got1 := translateErr(postgres.ErrConnRefused)
	got2 := translateErr(postgres.ErrAuthFailed)
	if got1 != got2 {
		t.Errorf("ErrConnRefused and ErrAuthFailed should collapse to the same message.\nrefused=%q\nauth=%q",
			got1, got2)
	}
	if !strings.Contains(got1, "Database refused connection") {
		t.Errorf("message does not match ADR-0005 template: %q", got1)
	}
}

// TestTranslateErr_NamesWhichSecret confirms ADR-0005's "the error
// names which" requirement for secret-fetch failures: when the same
// sentinel triggers on user_ref vs password_ref, the rendered message
// distinguishes the two so the user knows which config field to fix.
func TestTranslateErr_NamesWhichSecret(t *testing.T) {
	userErr := &secretSourceError{which: "user_ref", err: secrets.ErrSecretNotFound}
	passErr := &secretSourceError{which: "password_ref", err: secrets.ErrSecretNotFound}

	userMsg := translateErr(userErr)
	passMsg := translateErr(passErr)

	if !strings.Contains(userMsg, "(user_ref)") {
		t.Errorf("user_ref source missing from message: %q", userMsg)
	}
	if !strings.Contains(passMsg, "(password_ref)") {
		t.Errorf("password_ref source missing from message: %q", passMsg)
	}
}

// TestTranslateErr_Fallthrough makes sure an unrecognised error does
// not panic the translator and that the underlying err.Error() text
// is NOT surfaced to the user. The fallthrough is defense-in-depth
// against an upstream contract drift (postgres.go promises that
// every escape is a named sentinel; pgconn's raw text could contain
// the password if that promise ever breaks).
func TestTranslateErr_Fallthrough(t *testing.T) {
	got := translateErr(errors.New("password=hunter2 leaked-via-pgconn"))
	if got == "" {
		t.Fatal("translateErr returned empty string")
	}
	if strings.Contains(got, "password") || strings.Contains(got, "hunter2") {
		t.Errorf("fallthrough leaked raw error text into user message: %q", got)
	}
}

// --- startupOnce ----------------------------------------------------

// TestStartupOnce_SecretFailure_UserRef checks that a secret failure
// on the FIRST GetSecret call tags the error with user_ref and never
// reaches connect or migrate.
func TestStartupOnce_SecretFailure_UserRef(t *testing.T) {
	prov := &fakeProvider{err: secrets.ErrVaultNotFound}
	connectCalled := false
	connect := connectFn(func(_ context.Context, _ config.Database, _, _ []byte) (func(), error) {
		connectCalled = true
		return nil, nil
	})

	closer, err := startupOnce(context.Background(), fixtureCfg(), prov, connect, noopMigrate)
	if closer != nil {
		t.Error("closer should be nil on secret failure")
	}
	if !errors.Is(err, secrets.ErrVaultNotFound) {
		t.Errorf("err = %v, want errors.Is(_, ErrVaultNotFound)", err)
	}
	var src *secretSourceError
	if !errors.As(err, &src) || src.which != "user_ref" {
		t.Errorf("err not tagged user_ref: %v", err)
	}
	if connectCalled {
		t.Error("connect was invoked despite secret failure")
	}
}

// TestStartupOnce_DBDown asserts that a connect failure surfaces the
// raw sentinel (no source wrapping — the failure is post-secrets)
// and that migrate is not invoked.
func TestStartupOnce_DBDown(t *testing.T) {
	prov := goodProvider()
	migrateCalled := false
	connect := connectFn(func(_ context.Context, _ config.Database, _, _ []byte) (func(), error) {
		return nil, postgres.ErrConnRefused
	})
	migrate := migrateFn(func(_ context.Context, _ config.Database, _, _ []byte) error {
		migrateCalled = true
		return nil
	})

	closer, err := startupOnce(context.Background(), fixtureCfg(), prov, connect, migrate)
	if closer != nil {
		t.Error("closer should be nil on connect failure")
	}
	if !errors.Is(err, postgres.ErrConnRefused) {
		t.Errorf("err = %v, want errors.Is(_, ErrConnRefused)", err)
	}
	if migrateCalled {
		t.Error("migrate was invoked despite connect failure")
	}
}

// TestStartupOnce_MigrateFailureClosesPool ensures the pool closer
// fires on a migration failure so we do not leak a live pool on a
// post-connect error path.
func TestStartupOnce_MigrateFailureClosesPool(t *testing.T) {
	prov := goodProvider()
	stub := &stubCloser{}
	connect := connectFn(func(_ context.Context, _ config.Database, _, _ []byte) (func(), error) {
		return stub.close, nil
	})
	migrate := migrateFn(func(_ context.Context, _ config.Database, _, _ []byte) error {
		return postgres.ErrMigrationFailed
	})

	closer, err := startupOnce(context.Background(), fixtureCfg(), prov, connect, migrate)
	if closer != nil {
		t.Error("closer should be nil on migrate failure")
	}
	if !errors.Is(err, postgres.ErrMigrationFailed) {
		t.Errorf("err = %v, want errors.Is(_, ErrMigrationFailed)", err)
	}
	if stub.count != 1 {
		t.Errorf("pool closer fired %d times, want 1", stub.count)
	}
}

// TestStartupOnce_Success returns a closer and never touches the
// stderr path.
func TestStartupOnce_Success(t *testing.T) {
	prov := goodProvider()
	stub := &stubCloser{}
	connect := connectFn(func(_ context.Context, _ config.Database, _, _ []byte) (func(), error) {
		return stub.close, nil
	})

	closer, err := startupOnce(context.Background(), fixtureCfg(), prov, connect, noopMigrate)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if closer == nil {
		t.Fatal("closer = nil, want stubCloser.close")
	}
	closer()
	if stub.count != 1 {
		t.Errorf("closer not wired to stubCloser; count = %d", stub.count)
	}
	if prov.callCount != 2 {
		t.Errorf("provider.GetSecret called %d times, want 2 (user_ref + password_ref)", prov.callCount)
	}
}

// --- gated integration -------------------------------------------

// runIntegration mirrors the gating helper in
// internal/storage/postgres/integration_test.go: tests requiring a
// live database read connection details from SWITCHX_PG_* env vars
// and skip if SWITCHX_PG_INTEGRATION is not set to 1. The make target
// pg-integration delegates to a .local/dev-env script that exports
// these from the local dev environment.
func runIntegration(t *testing.T) (config.Database, []byte, []byte) {
	t.Helper()
	if os.Getenv("SWITCHX_PG_INTEGRATION") != "1" {
		t.Skip("set SWITCHX_PG_INTEGRATION=1 to run")
	}
	pw := os.Getenv("SWITCHX_PG_PASSWORD")
	if pw == "" {
		t.Skip("missing SWITCHX_PG_PASSWORD")
	}
	port := 5432
	if v := os.Getenv("SWITCHX_PG_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("bad SWITCHX_PG_PORT=%q: %v", v, err)
		}
		port = p
	}
	envOr := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}
	cfg := config.Database{
		Host:    envOr("SWITCHX_PG_HOST", "localhost"),
		Port:    port,
		Name:    envOr("SWITCHX_PG_DB", "switchx"),
		SSLMode: envOr("SWITCHX_PG_SSLMODE", "disable"),
	}
	user := []byte(envOr("SWITCHX_PG_USER", "switchx_user"))
	return cfg, user, []byte(pw)
}

// TestStartupOnce_HappyPath_RealPG exercises the production wiring
// against a live database: realConnect (Connect + HealthCheck) and
// postgres.Migrate. Skipped when SWITCHX_PG_INTEGRATION is not set.
// The empty migrations/ embedded set in M1 makes Migrate a no-op;
// the test still confirms the call doesn't surface an error.
func TestStartupOnce_HappyPath_RealPG(t *testing.T) {
	dbCfg, user, password := runIntegration(t)

	prov := &fakeProvider{
		userRef: "SWITCHX-DB-USER",
		userVal: user,
		passRef: "SWITCHX-DB-PASSWORD",
		passVal: password,
	}
	cfg := &config.Config{
		Database: config.Database{
			Host:    dbCfg.Host,
			Port:    dbCfg.Port,
			Name:    dbCfg.Name,
			SSLMode: dbCfg.SSLMode,
			Secret: config.Secret{
				Provider:     "azure_key_vault",
				Subscription: "ignored-for-fake",
				Vault:        "ignored-for-fake",
				UserRef:      "SWITCHX-DB-USER",
				PasswordRef:  "SWITCHX-DB-PASSWORD",
			},
		},
	}

	closer, err := startupOnce(context.Background(), cfg, prov, realConnect, postgres.Migrate)
	if err != nil {
		t.Fatalf("startupOnce: %v", err)
	}
	if closer == nil {
		t.Fatal("closer is nil; pool was not handed back")
	}
	closer()
}

// --- retry loop ----------------------------------------------------

// TestRunStartupWith_RetryRecovers exercises the AC's "retry recovers"
// path through runStartupWith's loop: iteration 1 fails
// (ErrConnRefused), the injected showError returns DecisionRetry,
// iteration 2 succeeds and the runHome substitute is called. Without
// the showError injection the test would need a TTY for errscreen.Run.
func TestRunStartupWith_RetryRecovers(t *testing.T) {
	prov := goodProvider()
	stub := &stubCloser{}
	calls := 0
	connect := connectFn(func(_ context.Context, _ config.Database, _, _ []byte) (func(), error) {
		calls++
		if calls == 1 {
			return nil, postgres.ErrConnRefused
		}
		return stub.close, nil
	})
	showCalls := 0
	showError := showErrorFn(func(_ context.Context, msg string) (errscreen.Decision, error) {
		showCalls++
		if !strings.Contains(msg, "Database refused connection") {
			t.Errorf("error msg = %q, want ADR-0005 connection-refused template", msg)
		}
		return errscreen.DecisionRetry, nil
	})
	homeCalls := 0
	runHome := func(_ context.Context, dbname string) error {
		homeCalls++
		if dbname != "switchx_test" {
			t.Errorf("dbname = %q, want switchx_test", dbname)
		}
		return nil
	}

	err := runStartupWith(context.Background(), fixtureCfg(), prov,
		connect, noopMigrate, runHome, showError)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if calls != 2 {
		t.Errorf("connect calls = %d, want 2 (fail, then succeed)", calls)
	}
	if showCalls != 1 {
		t.Errorf("showError calls = %d, want 1 (only the first failure)", showCalls)
	}
	if homeCalls != 1 {
		t.Errorf("runHome calls = %d, want 1 (called after retry succeeds)", homeCalls)
	}
	if stub.count != 1 {
		t.Errorf("pool closer fired %d times, want 1 (deferred until runHome returns)", stub.count)
	}
	if prov.callCount != 4 {
		t.Errorf("provider calls = %d, want 4 (user+pass on each of two iterations)",
			prov.callCount)
	}
}

// TestRunStartupWith_QuitFromErrorScreen confirms DecisionQuit returns
// runStartupWith cleanly (nil error) without invoking runHome.
func TestRunStartupWith_QuitFromErrorScreen(t *testing.T) {
	prov := goodProvider()
	connect := connectFn(func(_ context.Context, _ config.Database, _, _ []byte) (func(), error) {
		return nil, postgres.ErrConnRefused
	})
	showError := showErrorFn(func(_ context.Context, _ string) (errscreen.Decision, error) {
		return errscreen.DecisionQuit, nil
	})
	homeCalled := false
	runHome := func(_ context.Context, _ string) error {
		homeCalled = true
		return nil
	}

	err := runStartupWith(context.Background(), fixtureCfg(), prov,
		connect, noopMigrate, runHome, showError)
	if err != nil {
		t.Errorf("err = %v, want nil on user-driven quit", err)
	}
	if homeCalled {
		t.Error("runHome should not be called after DecisionQuit")
	}
}

// TestRunStartupWith_CanceledContext confirms context.Canceled from
// startupOnce short-circuits the loop and returns nil (clean SIGINT).
func TestRunStartupWith_CanceledContext(t *testing.T) {
	prov := &fakeProvider{err: context.Canceled}
	connect := connectFn(func(_ context.Context, _ config.Database, _, _ []byte) (func(), error) {
		t.Fatal("connect must not be called when provider returns context.Canceled")
		return nil, nil
	})
	showCalled := false
	showError := showErrorFn(func(_ context.Context, _ string) (errscreen.Decision, error) {
		showCalled = true
		return errscreen.DecisionQuit, nil
	})
	runHome := func(_ context.Context, _ string) error { return nil }

	err := runStartupWith(context.Background(), fixtureCfg(), prov,
		connect, noopMigrate, runHome, showError)
	if err != nil {
		t.Errorf("err = %v, want nil for context.Canceled", err)
	}
	if showCalled {
		t.Error("showError should not be called when context is cancelled")
	}
}

// TestStartupOnce_RetryIdempotence covers startupOnce's reentrance
// independently of the loop: calling it twice with the same provider
// and cfg, where iteration 1 fails and iteration 2 succeeds, mirrors
// what runStartupWith does without injecting any retry-loop seam.
// Kept alongside TestRunStartupWith_RetryRecovers because the two
// tests cover different layers (function-level vs loop-wiring).
func TestStartupOnce_RetryIdempotence(t *testing.T) {
	prov := goodProvider()
	calls := 0
	stub := &stubCloser{}
	connect := connectFn(func(_ context.Context, _ config.Database, _, _ []byte) (func(), error) {
		calls++
		if calls == 1 {
			return nil, postgres.ErrConnRefused
		}
		return stub.close, nil
	})

	ctx := context.Background()
	cfg := fixtureCfg()

	// Iteration 1: connect fails.
	closer1, err1 := startupOnce(ctx, cfg, prov, connect, noopMigrate)
	if closer1 != nil {
		t.Error("iter 1 closer should be nil")
	}
	if !errors.Is(err1, postgres.ErrConnRefused) {
		t.Errorf("iter 1 err = %v, want ErrConnRefused", err1)
	}
	if prov.callCount != 2 {
		t.Errorf("iter 1 provider calls = %d, want 2", prov.callCount)
	}

	// Iteration 2: connect succeeds. Same cfg, same provider, same
	// connect closure (toggled by the calls counter) — proving the
	// function is reentrant after failure with no carried state.
	closer2, err2 := startupOnce(ctx, cfg, prov, connect, noopMigrate)
	if err2 != nil {
		t.Fatalf("iter 2 err = %v, want nil", err2)
	}
	if closer2 == nil {
		t.Fatal("iter 2 closer = nil, want stub.close")
	}
	closer2()
	if stub.count != 1 {
		t.Errorf("stub closer fired %d times, want 1", stub.count)
	}
	if prov.callCount != 4 {
		t.Errorf("iter 2 cumulative provider calls = %d, want 4", prov.callCount)
	}
}
