package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// withXDGConfigHome points config.Path at a t.TempDir() and restores the
// previous environment on cleanup. It mirrors the real-world resolution
// (XDG_CONFIG_HOME → "switchx/config.toml") so tests exercise the same
// path code that production uses.
func withXDGConfigHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, "switchx", "config.toml")
}

func validConfig() *Config {
	return &Config{
		Database: Database{
			Host:    "db.example.internal",
			Port:    5432,
			Name:    "switchx",
			SSLMode: "require",
			Secret: Secret{
				Provider:     "azurekeyvault",
				Subscription: "00000000-0000-0000-0000-000000000000",
				Vault:        "switchx-vault",
				UserRef:      "switchx-user",
				PasswordRef:  "switchx-password",
			},
		},
		User: User{Timezone: "Europe/Brussels"},
		UI:   UI{Editor: "vi"},
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := withXDGConfigHome(t)
	cfg := validConfig()

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() = %v, want nil", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() = %v, want nil", err)
	}
	if *loaded != *cfg {
		t.Errorf("round-trip mismatch:\n got: %+v\nwant: %+v", loaded, cfg)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
}

func TestSaveEnforcesMode0600(t *testing.T) {
	path := withXDGConfigHome(t)
	if err := Save(validConfig()); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) = %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %v, want 0600", got)
	}
}

func TestSaveCreatesParentDir(t *testing.T) {
	path := withXDGConfigHome(t)
	parent := filepath.Dir(path)
	// The temp dir exists but the "switchx" subdirectory does not yet —
	// Save must create it.
	if _, err := os.Stat(parent); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("precondition: parent %s should not yet exist, got err=%v", parent, err)
	}
	if err := Save(validConfig()); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	info, err := os.Stat(parent)
	if err != nil {
		t.Fatalf("Stat(parent) = %v after Save", err)
	}
	if !info.IsDir() {
		t.Errorf("parent %s is not a directory", parent)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("parent dir mode = %v, want 0700", got)
	}
}

func TestSaveOverwritesExistingFile(t *testing.T) {
	path := withXDGConfigHome(t)
	if err := Save(validConfig()); err != nil {
		t.Fatalf("Save() #1 = %v", err)
	}

	updated := validConfig()
	updated.UI.Editor = "nvim"
	if err := Save(updated); err != nil {
		t.Fatalf("Save() #2 = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after second Save = %v", err)
	}
	if loaded.UI.Editor != "nvim" {
		t.Errorf("editor = %q, want nvim", loaded.UI.Editor)
	}
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("tmp file leaked: %v", err)
	}
}

func TestLoadPartialReturnsEmptyWhenFileMissing(t *testing.T) {
	withXDGConfigHome(t)
	cfg := LoadPartial()
	if cfg == nil {
		t.Fatal("LoadPartial returned nil; expected empty *Config")
	}
	if *cfg != (Config{}) {
		t.Errorf("LoadPartial on missing file = %+v, want zero", *cfg)
	}
}

func TestLoadPartialReturnsWhateverIsPresent(t *testing.T) {
	path := withXDGConfigHome(t)
	cfg := validConfig()
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	got := LoadPartial()
	if got.Database.Host != cfg.Database.Host {
		t.Errorf("LoadPartial host = %q, want %q", got.Database.Host, cfg.Database.Host)
	}
	if got.UI.Editor != cfg.UI.Editor {
		t.Errorf("LoadPartial editor = %q, want %q", got.UI.Editor, cfg.UI.Editor)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
}

func TestSaveNilConfig(t *testing.T) {
	withXDGConfigHome(t)
	err := Save(nil)
	if !errors.Is(err, ErrSaveFailed) {
		t.Errorf("Save(nil) error = %v, want wrapping ErrSaveFailed", err)
	}
}
