// Command switchx is the terminal time tracker for consulting teams.
//
// This is the M1 placeholder; subsequent issues replace it with the full
// bootstrap flow and TUI. See the M1 milestone in Linear:
// https://linear.app/taelron/project/switchx-0b0069bd1c04
//
// TAE-6 wires the config loader; TAE-7 wires the secret resolver. TAE-13
// replaces this minimal wiring with the full startup decision tree
// (config -> secret -> pool -> migrations -> home).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Taelron/SwitchX/internal/app/secrets"
	"github.com/Taelron/SwitchX/internal/config"
	"github.com/Taelron/SwitchX/internal/storage/secrets/azurekeyvault"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run holds the bootstrap sequence in a function whose deferred calls
// run on every exit path. main() only translates errors into a process
// exit code, so secret-zero defers never get skipped by os.Exit.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	provider := azurekeyvault.New()
	ctx := context.Background()

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

	fmt.Printf("switchx (placeholder) — config OK; user (%d bytes) + password (%d bytes) fetched\n",
		len(user), len(password))
	return nil
}

// zero overwrites a byte slice with zeros. Used via defer to scrub
// secret material from memory once consumed.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
