// Package bootstrap implements the modal Bubble Tea wizard that runs
// when switchx starts without a usable configuration. It collects the
// values needed to populate the config file, validates the database
// connection end-to-end, persists the file at mode 0600, and hands the
// resulting *config.Config back to main so the existing KV / connect /
// migrate flow continues in-process.
//
// References (canonical in Linear):
//
//   - UI Patterns
//     https://linear.app/taelron/document/ui-patterns-9c3982a46ef2
//   - TUI Go Conventions
//     https://linear.app/taelron/document/tui-go-conventions-1aca4ef63a66
//   - ADR-0004 — Configuration and Bootstrap
//     https://linear.app/taelron/document/adr-0004-configuration-and-bootstrap-e5000764e4c4
//   - ADR-0005 — Secret Handling for switchx
//     https://linear.app/taelron/document/adr-0005-secret-handling-for-switchx-ea21d3f379f4
package bootstrap

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taelron/SwitchX/internal/config"
)

// Validator runs the cross-step end-to-end check that the user's
// answers actually work: fetch both secrets via the configured Provider,
// open the Postgres pool, run the health-check round-trip. The wizard
// package never imports azurekeyvault or postgres directly — that
// wiring stays in cmd/switchx/main.go and tests inject a fake.
//
// Implementations MUST honour ctx cancellation: the wizard's Esc handler
// cancels the context so an in-flight validate (KV roundtrip + DB dial)
// returns promptly when the user aborts.
type Validator func(ctx context.Context, cfg *config.Config) error

// Run launches the bootstrap wizard.
//
//   - On success: returns the persisted *config.Config (already written
//     to disk at mode 0600) and a nil error.
//   - On Esc / clean abort: returns (nil, nil).
//   - On unrecoverable error inside the wizard runtime itself (not a
//     validate failure — those are surfaced via the in-app banner):
//     returns (nil, err).
//
// The seed *config.Config pre-fills every form field from any
// previously-written config so the user does not have to retype values
// that were already correct. Pass nil for an empty wizard.
func Run(ctx context.Context, validate Validator, seed *config.Config) (*config.Config, error) {
	wizardCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	prog := tea.NewProgram(
		newModel(wizardCtx, validate, config.Save, seed),
		tea.WithContext(ctx),
		tea.WithAltScreen(),
	)
	final, err := prog.Run()
	// tea.WithContext makes prog.Run return ctx.Err() on cancellation;
	// SIGINT / SIGTERM via signal.NotifyContext is the user asking to
	// quit cleanly, not a wizard failure.
	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, fmt.Errorf("bootstrap wizard: %w", err)
	}
	fm, ok := final.(model)
	if !ok {
		return nil, nil
	}
	return fm.finalCfg, nil
}
