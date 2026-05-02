package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taelron/SwitchX/internal/config"
)

// TestEndToEndPersist drives the model from step 1 through a successful
// validate to confirm the file actually lands on disk at the resolved
// XDG path with mode 0600 — exercising the real config.Save closure
// rather than an injected stub.
func TestEndToEndPersist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("EDITOR", "")

	m := newModel(
		context.Background(),
		func(context.Context, *config.Config) error { return nil },
		config.Save,
		nil,
	)
	m = fillStep1(m)
	m = fillStep2(m)
	m = fillStep3(m)
	m = pickEditor(m, "vi")
	next, _ := m.Update(keyMsg("enter"))
	m = next.(model)

	out, cmd := m.Update(validateResultMsg{generation: 1, err: nil})
	m = out.(model)

	if m.finalCfg == nil {
		t.Fatal("finalCfg is nil after success path")
	}
	if cmd == nil {
		t.Fatal("success path returned nil cmd, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("success cmd produced %T, want tea.QuitMsg", cmd())
	}

	// File on disk: exists, mode 0600, round-trips through Load.
	path := filepath.Join(dir, "switchx", "config.toml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) = %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %v, want 0600", got)
	}
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() of persisted file = %v", err)
	}
	if loaded.Database.Host != "db.example.internal" ||
		loaded.User.Timezone != "Europe/Brussels" ||
		loaded.UI.Editor != "vi" {
		t.Errorf("persisted config did not round-trip: %+v", loaded)
	}
}
