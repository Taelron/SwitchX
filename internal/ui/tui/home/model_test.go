package home

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
// snapshots when the rendering changes intentionally. Run, eyeball the
// diff, commit.
var updateGolden = flag.Bool("update", false, "regenerate testdata/*.golden files")

// TestMain forces Lipgloss to plain ASCII for this package so the
// golden snapshots stay terminal-independent — otherwise the bytes
// vary with $TERM, $COLORTERM, and CI vs local runs.
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

// render builds a sized model and returns its View(). Centralised so
// every golden test exercises the same WindowSizeMsg path the real
// program takes on launch.
func render(t *testing.T, dbname string, w, h int, keys ...string) string {
	t.Helper()
	var m tea.Model = newModel(dbname)
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

func TestHomeViewGolden(t *testing.T) {
	got := render(t, "switchx_dev", 80, 24)
	compareGolden(t, "home.golden", got)
}

func TestHelpOverlayGolden(t *testing.T) {
	got := render(t, "switchx_dev", 80, 24, "?")
	compareGolden(t, "help.golden", got)
}

// TestHelpToggle exercises the two interactions the View() golden
// tests do not cover on their own: closing via `?` and closing via
// Esc.
func TestHelpToggle(t *testing.T) {
	var m tea.Model = newModel("dev")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m, _ = m.Update(keyMsg("?"))
	if !m.(model).showHelp {
		t.Fatal("? did not open help overlay")
	}

	m, _ = m.Update(keyMsg("?"))
	if m.(model).showHelp {
		t.Fatal("? did not close help overlay")
	}

	m, _ = m.Update(keyMsg("?"))
	m, _ = m.Update(keyMsg("esc"))
	if m.(model).showHelp {
		t.Fatal("Esc did not close help overlay")
	}
}

// TestEscOnRootIsNoOp verifies the design call in the architect plan:
// Esc on the bare home view does not quit (only q / Ctrl+C do).
func TestEscOnRootIsNoOp(t *testing.T) {
	var m tea.Model = newModel("dev")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	out, cmd := m.Update(keyMsg("esc"))
	if cmd != nil {
		t.Errorf("Esc on root returned cmd=%v, want nil", cmd)
	}
	if out.(model).showHelp {
		t.Errorf("Esc on root toggled showHelp; should be inert when overlay closed")
	}
}

// TestQuitKeys verifies q and Ctrl+C both return tea.Quit.
func TestQuitKeys(t *testing.T) {
	for _, key := range []string{"q", "ctrl+c"} {
		var m tea.Model = newModel("dev")
		_, cmd := m.Update(keyMsg(key))
		if cmd == nil {
			t.Errorf("%s returned nil cmd, want tea.Quit", key)
			continue
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s cmd produced %T, want tea.QuitMsg", key, cmd())
		}
	}
}
