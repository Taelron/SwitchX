// Command switchx is the terminal time tracker for consulting teams.
//
// This is the M1 placeholder; subsequent issues replace it with the full
// bootstrap flow and TUI. See the M1 milestone in Linear:
// https://linear.app/taelron/project/switchx-0b0069bd1c04
//
// TAE-6 wires the config loader; TAE-7 wires the secret resolver;
// TAE-8 wires the connection pool; TAE-9 wires the migrations runner.
// TAE-13 replaces this minimal flow with the full startup decision
// tree (config -> secret -> pool -> migrations -> home).
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Taelron/SwitchX/internal/app/secrets"
	"github.com/Taelron/SwitchX/internal/config"
	"github.com/Taelron/SwitchX/internal/storage/postgres"
	"github.com/Taelron/SwitchX/internal/storage/secrets/azurekeyvault"
	"github.com/Taelron/SwitchX/internal/ui/tui/bootstrap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run holds the bootstrap sequence in a function whose deferred calls
// run on every exit path. main() only translates errors into a process
// exit code, so secret-zero and pool-close defers never get skipped by
// os.Exit.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if errors.Is(err, config.ErrNoConfig) || errors.Is(err, config.ErrIncomplete) {
		// Bootstrap wizard collects the missing values. TAE-10 ships
		// the wizard shell and step 1 only; persistence and the
		// connect-validate round-trip arrive in TAE-11, after which
		// run() will reload the config and continue.
		return bootstrap.Run(ctx)
	}
	if err != nil {
		return err
	}

	provider := azurekeyvault.New()

	user, err := provider.GetSecret(ctx, secrets.Ref{
		Subscription: cfg.Database.Secret.Subscription,
		Vault:        cfg.Database.Secret.Vault,
		Name:         cfg.Database.Secret.UserRef,
	})
	if err != nil {
		return fmt.Errorf("user_ref: %w", err)
	}
	defer zero(user)

	password, err := provider.GetSecret(ctx, secrets.Ref{
		Subscription: cfg.Database.Secret.Subscription,
		Vault:        cfg.Database.Secret.Vault,
		Name:         cfg.Database.Secret.PasswordRef,
	})
	if err != nil {
		return fmt.Errorf("password_ref: %w", err)
	}
	defer zero(password)

	pool, err := postgres.Connect(ctx, cfg.Database, user, password)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	if err := postgres.HealthCheck(ctx, pool); err != nil {
		return fmt.Errorf("health: %w", err)
	}

	if err := postgres.Migrate(ctx, cfg.Database, user, password); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	fmt.Printf("switchx (placeholder) — DB ready, migrations applied (pool max %d, host %s, db %s)\n",
		pool.Config().MaxConns, cfg.Database.Host, cfg.Database.Name)
	return nil
}

// zero overwrites a byte slice with zeros. Used via defer to scrub
// secret material from memory once consumed.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
