// Package bootstrap implements the modal Bubble Tea wizard that runs
// when switchx starts without a usable configuration. It collects the
// values needed to populate the config file and then hands control back
// to main.
//
// TAE-10 ships the wizard shell and step 1 (database connection
// details). Steps 2–4, the connection-validation round-trip, and the
// config-file write are TAE-11.
//
// References (canonical in Linear):
//
//   - UI Patterns
//     https://linear.app/taelron/document/ui-patterns-9c3982a46ef2
//   - TUI Go Conventions
//     https://linear.app/taelron/document/tui-go-conventions-1aca4ef63a66
//   - ADR-0004 — Configuration and Bootstrap
//     https://linear.app/taelron/document/adr-0004-configuration-and-bootstrap-e5000764e4c4
package bootstrap

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Run launches the bootstrap wizard. It blocks until the user exits
// (Esc) or reaches the terminal step. For TAE-10 the terminal step is
// the placeholder for step 2; TAE-11 replaces it with the remaining
// steps and a config-file write.
//
// The context is observed for cancellation: SIGINT/SIGTERM at the OS
// level cancel ctx, which causes the program to quit cleanly.
func Run(ctx context.Context) error {
	prog := tea.NewProgram(newModel(),
		tea.WithContext(ctx),
		tea.WithAltScreen(),
	)
	// tea.WithContext makes prog.Run return ctx.Err() on cancellation;
	// SIGINT/SIGTERM via signal.NotifyContext is the user asking to
	// quit cleanly, not a wizard failure.
	if _, err := prog.Run(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("bootstrap wizard: %w", err)
	}
	return nil
}
