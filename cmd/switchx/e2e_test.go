package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Taelron/SwitchX/internal/config"
	"github.com/Taelron/SwitchX/internal/storage/postgres"
	"github.com/Taelron/SwitchX/internal/ui/tui/errscreen"
)

// TestM1E2E exercises the M1 milestone's nine-step exit criterion
// end-to-end against a live local Postgres. Gated by
// SWITCHX_PG_INTEGRATION=1 (same env-var skip convention as
// internal/storage/postgres/integration_test.go and the rest of
// startup_test.go); skipped in CI.
//
// Subtests map 1:1 to the numbered steps so a regression names which
// step broke. Steps deliberately omitted from this test:
//
//   - Step 1 (`go install`) is human-verified per docs/m1-smoke-test.md.
//     Automating the install adds nothing beyond `go build`, which
//     `make test` and `make verify` already cover.
//   - Steps 3, 4, 6 (wizard UI itself) are exhaustively covered by
//     internal/ui/tui/bootstrap/bootstrap_test.go::TestEndToEndPersist
//     and the model_test.go subtree, which drive every field, every
//     validation path, and the persist-to-disk-at-mode-0600 contract.
//     Re-testing the wizard internals here would duplicate work.
func TestM1E2E(t *testing.T) {
	dbCfg, user, pw := runIntegration(t)

	// XDG used by steps 5+; step 2 overrides with its own empty dir.
	xdgDefault := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDefault)

	t.Run("step2_no_config_triggers_wizard", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		_, err := config.Load()
		if !errors.Is(err, config.ErrNoConfig) {
			t.Fatalf("config.Load() = %v\nwant errors.Is(_, ErrNoConfig); this is the sentinel that drives run() into the wizard branch", err)
		}
	})

	cfg, prov := setupE2EDB(t, dbCfg, user, pw)

	t.Run("step5_startuponce_validates_db_and_runs_migrations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		closer, err := startupOnce(ctx, cfg, prov, realConnect, postgres.Migrate)
		if err != nil {
			t.Fatalf("startupOnce: %v\nwant nil; step 5 expects resolve→connect→health→migrate to succeed against the live DB", err)
		}
		if closer == nil {
			t.Fatal("startupOnce returned nil closer; the pool was not handed back")
		}
		closer()
	})

	t.Run("step7_runhome_receives_correct_dbname", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var captured string
		runHome := func(_ context.Context, dbname string) error {
			captured = dbname
			return nil
		}
		showError := showErrorFn(func(_ context.Context, msg string) (errscreen.Decision, error) {
			t.Fatalf("showError invoked on happy path: %q", msg)
			return errscreen.DecisionQuit, nil
		})
		if err := runStartupWith(ctx, cfg, prov, realConnect, postgres.Migrate, runHome, showError); err != nil {
			t.Fatalf("runStartupWith: %v", err)
		}
		if captured != cfg.Database.Name {
			t.Errorf("runHome dbname = %q, want %q\nhome screen would render \"Connected to %s\"; the wiring fed it %q",
				captured, cfg.Database.Name, cfg.Database.Name, captured)
		}
	})

	t.Run("step8_rerun_skips_wizard_path", func(t *testing.T) {
		loaded, err := config.Load()
		// Check the wizard-triggering sentinels first; on a successful
		// re-run neither must be present, because either one would
		// route run() back into the wizard branch.
		if errors.Is(err, config.ErrNoConfig) || errors.Is(err, config.ErrIncomplete) {
			t.Fatalf("re-run config.Load() returned a wizard-triggering sentinel: %v", err)
		}
		if err != nil {
			t.Fatalf("re-run config.Load() = %v\nshould succeed; setupE2EDB persisted a complete config to XDG_CONFIG_HOME=%s",
				err, os.Getenv("XDG_CONFIG_HOME"))
		}
		if loaded.Database.Name != cfg.Database.Name {
			t.Errorf("re-run loaded.Database.Name = %q, want %q", loaded.Database.Name, cfg.Database.Name)
		}

		// The full chain still works on a second invocation — proves
		// no wizard is needed and no state is carried over.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		closer, err := startupOnce(ctx, loaded, prov, realConnect, postgres.Migrate)
		if err != nil {
			t.Fatalf("re-run startupOnce: %v", err)
		}
		closer()
	})

	t.Run("step9_db_down_then_retry_recovers", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		calls := 0
		connect := connectFn(func(c context.Context, db config.Database, u, p []byte) (func(), error) {
			calls++
			if calls == 1 {
				return nil, postgres.ErrConnRefused
			}
			return realConnect(c, db, u, p)
		})
		showCalls := 0
		showError := showErrorFn(func(_ context.Context, _ string) (errscreen.Decision, error) {
			showCalls++
			return errscreen.DecisionRetry, nil
		})
		var captured string
		runHome := func(_ context.Context, dbname string) error {
			captured = dbname
			return nil
		}
		if err := runStartupWith(ctx, cfg, prov, connect, postgres.Migrate, runHome, showError); err != nil {
			t.Fatalf("runStartupWith: %v\nwant nil; retry path must recover and reach home", err)
		}
		if calls != 2 {
			t.Errorf("connect calls = %d, want 2 (first fails with ErrConnRefused, second succeeds)", calls)
		}
		if showCalls != 1 {
			t.Errorf("showError calls = %d, want 1 (one failure → one error screen)", showCalls)
		}
		if captured != cfg.Database.Name {
			t.Errorf("runHome dbname after retry = %q, want %q", captured, cfg.Database.Name)
		}
	})
}

// setupE2EDB creates a uniquely-named throwaway database, writes a
// valid config.toml to XDG_CONFIG_HOME pointing at that database, and
// returns the *config.Config plus a fakeProvider wired to the live
// user/password obtained from runIntegration. The database is dropped
// on t.Cleanup.
//
// Mirrors the setupTestDB pattern in
// internal/storage/postgres/integration_test.go but is duplicated here
// because (a) the postgres helper lives in a different package and
// (b) the E2E test needs the full *Config plus a configured fakeProvider,
// not just a config.Database.
func setupE2EDB(t *testing.T, base config.Database, user, password []byte) (*config.Config, *fakeProvider) {
	t.Helper()

	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("rand: %v", err)
	}
	testDB := "switchx_e2e_" + hex.EncodeToString(b[:])

	admin := base
	admin.Name = "postgres"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := postgres.Connect(ctx, admin, user, password)
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %q", testDB)); err != nil {
		pool.Close()
		t.Fatalf("create test db %s: %v", testDB, err)
	}
	pool.Close()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		admin := base
		admin.Name = "postgres"
		pool, err := postgres.Connect(ctx, admin, user, password)
		if err != nil {
			return // best-effort
		}
		defer pool.Close()
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %q WITH (FORCE)", testDB))
	})

	const (
		userRef     = "SWITCHX-DB-USER"
		passwordRef = "SWITCHX-DB-PASSWORD"
	)
	cfg := &config.Config{
		Database: config.Database{
			Host:    base.Host,
			Port:    base.Port,
			Name:    testDB,
			SSLMode: base.SSLMode,
			Secret: config.Secret{
				Provider:     "azurekeyvault",
				Subscription: "ignored-for-fake",
				Vault:        "ignored-for-fake",
				UserRef:      userRef,
				PasswordRef:  passwordRef,
			},
		},
		User: config.User{Timezone: "Europe/Brussels"},
		UI:   config.UI{Editor: "vi"},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save: %v (XDG_CONFIG_HOME=%s)", err, os.Getenv("XDG_CONFIG_HOME"))
	}

	prov := &fakeProvider{
		userRef: userRef,
		userVal: append([]byte(nil), user...),
		passRef: passwordRef,
		passVal: append([]byte(nil), password...),
	}
	return cfg, prov
}
