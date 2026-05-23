// Package home renders the M1 placeholder home screen — the view a
// user sees after the bootstrap wizard succeeds, or directly on launch
// when a valid config is already on disk.
//
// The screen is intentionally minimal: a breadcrumb, a centred
// "Connected to {dbname}" line, a hint bar, and a `?` help overlay.
// No domain features land here in M1 — the real home (session block,
// today's timeline, totals, switch palette) ships in M5+.
//
// Per @UI Patterns the home is a divergent view shape (status display,
// not a tabular list), explicitly allowed by the "Rendering structure"
// section. Breadcrumb at the top and hint bar at the bottom still
// apply; the body slot is filled by the centred status line.
//
// References (canonical in Linear):
//
//   - UI Patterns
//     https://linear.app/taelron/document/ui-patterns-9c3982a46ef2
//   - TUI Go Conventions
//     https://linear.app/taelron/document/tui-go-conventions-1aca4ef63a66
package home

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Run launches the placeholder home view and blocks until the user
// quits (`q` / Ctrl+C) or the parent context is cancelled (SIGINT /
// SIGTERM). Returns nil for a clean exit; a wrapped error only if the
// Bubble Tea runtime itself fails.
//
// dbname is rendered on screen and comes from cfg.Database.Name in
// cmd/switchx/main.go. It is the config value, not the result of a
// SELECT current_database() round-trip — the config value is already
// known and avoids extra I/O on a placeholder screen.
func Run(ctx context.Context, dbname string) error {
	prog := tea.NewProgram(
		newModel(dbname),
		tea.WithContext(ctx),
		tea.WithAltScreen(),
	)
	if _, err := prog.Run(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("home view: %w", err)
	}
	return nil
}
