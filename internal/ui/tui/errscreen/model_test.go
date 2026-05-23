package errscreen

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// updateGolden lets `go test ./... -update` regenerate the testdata
// snapshots when the rendering changes intentionally. Mirrors the
// home package's pattern so test ergonomics stay consistent.
var updateGolden = flag.Bool("update", false, "regenerate testdata/*.golden files")

// TestMain forces Lipgloss to plain ASCII so the golden snapshots
// stay terminal-independent across local runs and CI.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func render(t *testing.T, msg string, w, h int, keys ...string) string {
	t.Helper()
	var m tea.Model = newModel(msg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	for _, k := range keys {
		m, _ = m.Update(keyMsg(k))
	}
	return m.View()
}

func compareGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *updateGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	if got != string(want) {
		t.Errorf("%s mismatch.\n--- got ---\n%s\n--- want ---\n%s",
			name, got, string(want))
	}
}

func TestErrscreenViewGolden(t *testing.T) {
	got := render(t, "Database refused connection. The role name or password may be wrong, or the role may not have CONNECT privilege.", 80, 24)
	compareGolden(t, "errscreen.golden", got)
}

// TestRetryKey confirms `r` sets DecisionRetry and emits tea.Quit so
// the caller's loop can re-run step 3+.
func TestRetryKey(t *testing.T) {
	var m tea.Model = newModel("anything")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	out, cmd := m.Update(keyMsg("r"))
	if got := out.(model).decision; got != DecisionRetry {
		t.Errorf("decision = %v, want DecisionRetry", got)
	}
	if cmd == nil {
		t.Fatal("r returned nil cmd, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("r cmd produced %T, want tea.QuitMsg", cmd())
	}
}

// TestQuitKeys verifies q and Ctrl+C both quit with DecisionQuit. The
// zero value of Decision is DecisionQuit so the test also catches an
// accidental flip in the constant ordering.
func TestQuitKeys(t *testing.T) {
	for _, key := range []string{"q", "ctrl+c"} {
		var m tea.Model = newModel("anything")
		out, cmd := m.Update(keyMsg(key))
		if got := out.(model).decision; got != DecisionQuit {
			t.Errorf("%s: decision = %v, want DecisionQuit", key, got)
		}
		if cmd == nil {
			t.Errorf("%s returned nil cmd, want tea.Quit", key)
			continue
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s cmd produced %T, want tea.QuitMsg", key, cmd())
		}
	}
}

// TestUnknownKeyIgnored asserts that keys outside the documented
// keymap do not flip the decision and do not emit tea.Quit, per @UI
// Patterns "Unknown keys are silently ignored".
func TestUnknownKeyIgnored(t *testing.T) {
	for _, key := range []string{"x", "?", "esc", "enter"} {
		var m tea.Model = newModel("anything")
		out, cmd := m.Update(keyMsg(key))
		if got := out.(model).decision; got != DecisionQuit {
			t.Errorf("%s: decision = %v, want DecisionQuit (zero value)", key, got)
		}
		if cmd != nil {
			t.Errorf("%s emitted cmd=%v, want nil", key, cmd)
		}
	}
}
