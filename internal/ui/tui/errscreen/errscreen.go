// Package errscreen renders the post-wizard startup error screen.
//
// switchx's startup flow (per TAE-13) goes config → wizard (if needed)
// → resolve → pool → health → migrate → home. When a failure happens
// after the wizard has produced a valid config — i.e. somewhere in
// resolve / pool / health / migrate — the wizard is the wrong place
// to send the user (its job is fixing config, not waiting for the
// network or for Postgres to come back). The error screen is what we
// show instead: a red message describing the failure, a hint bar of
// r/Retry q/Quit, and the same breadcrumb chrome as every other view.
//
// The screen does not own retry semantics itself — it returns a
// Decision and main.go re-runs steps 3+. The split keeps the screen
// inert and easy to test; the retry policy lives in one place.
//
// References (canonical in Linear):
//
//   - UI Patterns — "Operational UI states / Error state"
//     https://linear.app/taelron/document/ui-patterns-9c3982a46ef2
//   - Specification — "Lifecycle and concurrency notes / Database
//     connection lifecycle"
//     https://linear.app/taelron/document/specification-3645653a2170
package errscreen

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Decision is the user's choice on the error screen. The zero value is
// DecisionQuit so any abnormal exit (context cancellation, a tea
// runtime failure) defaults to "stop", which is the safe behaviour
// when the program is already in a degraded startup path.
type Decision int

// The Decision values returned by Run. DecisionQuit is the zero value
// so any abnormal exit collapses to "stop", which is the safe default
// on a degraded startup path.
const (
	DecisionQuit Decision = iota
	DecisionRetry
)

// Run displays the error screen and blocks until the user presses r or
// q (or the parent context is cancelled, which collapses to
// DecisionQuit). msg is the already-translated user-readable error
// string — errscreen does not consult any sentinel package.
//
// A tea runtime error is returned wrapped; the Decision is still
// usable (zero value = DecisionQuit) so the caller can branch cleanly
// either on Decision or on err.
func Run(ctx context.Context, msg string) (Decision, error) {
	prog := tea.NewProgram(
		newModel(msg),
		tea.WithContext(ctx),
		tea.WithAltScreen(),
	)
	final, err := prog.Run()
	if err != nil && !errors.Is(err, context.Canceled) {
		return DecisionQuit, fmt.Errorf("errscreen: %w", err)
	}
	fm, ok := final.(model)
	if !ok {
		return DecisionQuit, nil
	}
	return fm.decision, nil
}
