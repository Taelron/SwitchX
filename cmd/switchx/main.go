// Command switchx is the terminal time tracker for consulting teams.
//
// main owns process glue only: flag parsing, the signal-driven
// context, and the wizard branch. The post-wizard startup chain
// (resolve → pool → health → migrate → home) and its retry loop live
// in startup.go (TAE-13).
//
// See the M1 milestone in Linear:
// https://linear.app/taelron/project/switchx-0b0069bd1c04
package main

import (
	"context"
	"errors"
	"flag"
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

// run holds the program's outermost orchestration. It returns the
// error main translates into a non-zero exit; everything else
// (success path, clean cancellation, user-driven quit) returns nil.
func run() error {
	editFlag := flag.Bool("edit", false,
		"Re-run the bootstrap wizard against the existing config to edit it.")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	provider := azurekeyvault.New()

	// SWITCHX_FORCE_WIZARD is a dev affordance used by the make
	// wizard-test-* targets so that the wizard UI can be smoke-tested
	// with seeded configs even when config.Load would otherwise
	// succeed. Production users use --edit instead.
	forceWizard := *editFlag || os.Getenv("SWITCHX_FORCE_WIZARD") == "1"

	cfg, err := config.Load()
	needsWizard := forceWizard ||
		errors.Is(err, config.ErrNoConfig) ||
		errors.Is(err, config.ErrIncomplete)
	if needsWizard {
		seed := config.LoadPartial()
		wizCfg, werr := bootstrap.Run(ctx, validateConnection(provider), seed)
		if werr != nil {
			return werr
		}
		if wizCfg == nil {
			// User aborted with Esc — exit cleanly. Wizard re-runs on
			// next launch because config is still missing/incomplete.
			return nil
		}
		cfg = wizCfg
	} else if err != nil {
		return err
	}

	return runStartup(ctx, cfg, provider)
}

// validateConnection returns a bootstrap.Validator that exercises the
// full secret-fetch → pool-connect → health-check chain. The wizard
// invokes it at the end of step 4; the pool opened here is closed
// immediately after the health check succeeds and the runStartup loop
// reopens its own pool with the persisted config.
func validateConnection(provider secrets.Provider) bootstrap.Validator {
	return func(ctx context.Context, cfg *config.Config) error {
		// Categorising the error source ("secret store" vs "database")
		// is the part of the message users care about — they need to
		// know whether to fix step 2 (KV coords) or step 1 (DB host /
		// port / name / sslmode). Keep the field name in parentheses
		// so the underlying sentinel still tells operators which
		// specific input was wrong.
		user, err := provider.GetSecret(ctx, secrets.Ref{
			Subscription: cfg.Database.Secret.Subscription,
			Vault:        cfg.Database.Secret.Vault,
			Name:         cfg.Database.Secret.UserRef,
		})
		if err != nil {
			return fmt.Errorf("secret store (user_ref): %w", err)
		}
		defer zero(user)

		password, err := provider.GetSecret(ctx, secrets.Ref{
			Subscription: cfg.Database.Secret.Subscription,
			Vault:        cfg.Database.Secret.Vault,
			Name:         cfg.Database.Secret.PasswordRef,
		})
		if err != nil {
			return fmt.Errorf("secret store (password_ref): %w", err)
		}
		defer zero(password)

		pool, err := postgres.Connect(ctx, cfg.Database, user, password)
		if err != nil {
			return fmt.Errorf("database connect: %w", err)
		}
		defer pool.Close()
		if err := postgres.HealthCheck(ctx, pool); err != nil {
			return fmt.Errorf("database health check: %w", err)
		}
		return nil
	}
}

// zero overwrites a byte slice with zeros. Used via defer to scrub
// secret material from memory once consumed.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
