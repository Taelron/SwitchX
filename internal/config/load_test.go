package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// installFile copies the named testdata file into the per-test XDG
// config dir and returns the resolved config path.
func installFile(t *testing.T, xdgDir, fixture string, mode os.FileMode) string {
	t.Helper()
	cfgDir := filepath.Join(xdgDir, "switchx")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("mkdir cfgDir: %v", err)
	}
	src, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixture, err)
	}
	path := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(path, src, mode); err != nil {
		t.Fatalf("write config: %v", err)
	}
	// Defensive chmod against the process umask.
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	return path
}

func TestPath_WithXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	want := "/tmp/xdg-test/switchx/config.toml"
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestPath_NoXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/home-test")
	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	want := "/tmp/home-test/.config/switchx/config.toml"
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := Load()
	if !errors.Is(err, ErrNoConfig) {
		t.Fatalf("want ErrNoConfig, got %v", err)
	}
}

func TestLoad_WrongMode_0644(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	installFile(t, dir, "valid.toml", 0o644)
	_, err := Load()
	if !errors.Is(err, ErrWrongMode) {
		t.Fatalf("want ErrWrongMode, got %v", err)
	}
}

func TestLoad_WrongMode_0660(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	installFile(t, dir, "valid.toml", 0o660)
	_, err := Load()
	if !errors.Is(err, ErrWrongMode) {
		t.Fatalf("want ErrWrongMode, got %v", err)
	}
}

func TestLoad_ModeAcceptsOwnerOnly(t *testing.T) {
	// 0600, 0500, 0400, 0700 should all pass per the baseline rule.
	for _, mode := range []os.FileMode{0o600, 0o500, 0o400, 0o700} {
		t.Run(mode.String(), func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			installFile(t, dir, "valid.toml", mode)
			if _, err := Load(); err != nil {
				t.Errorf("mode %v: want success, got %v", mode, err)
			}
		})
	}
}

func TestLoad_MalformedTOML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	installFile(t, dir, "malformed.toml", 0o600)
	_, err := Load()
	if !errors.Is(err, ErrMalformed) {
		t.Fatalf("want ErrMalformed, got %v", err)
	}
}

func TestLoad_IncompleteConfig_PerField(t *testing.T) {
	// Each subtest installs valid.toml and then deletes one required
	// field's line, asserting Load reports that specific field.
	cases := []struct {
		field      string
		removeLine string
	}{
		{"database.host", `host    = "test-db.example.com"`},
		{"database.port", `port    = 5432`},
		{"database.name", `name    = "switchx"`},
		{"database.sslmode", `sslmode = "require"`},
		{"database.secret.provider", `provider     = "azure-keyvault"`},
		{"database.secret.subscription", `subscription = "test-subscription"`},
		{"database.secret.vault", `vault        = "test-vault"`},
		{"database.secret.user_ref", `user_ref     = "switchx-db-test-user"`},
		{"database.secret.password_ref", `password_ref = "switchx-db-test-password"`},
		{"user.timezone", `timezone = "Europe/Brussels"`},
		{"ui.editor", `editor = "nvim"`},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			path := installFile(t, dir, "valid.toml", 0o600)
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			modified := strings.Replace(string(src), tc.removeLine, "", 1)
			if modified == string(src) {
				t.Fatalf("removeLine %q not found in fixture (test setup bug)", tc.removeLine)
			}
			if err := os.WriteFile(path, []byte(modified), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err = Load()
			if !errors.Is(err, ErrIncomplete) {
				t.Fatalf("want ErrIncomplete, got %v", err)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Errorf("error should mention %q, got: %v", tc.field, err)
			}
		})
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	installFile(t, dir, "valid.toml", 0o600)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Host != "test-db.example.com" {
		t.Errorf("Host = %q", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Port = %d", cfg.Database.Port)
	}
	if cfg.Database.Secret.Subscription != "test-subscription" {
		t.Errorf("Subscription = %q", cfg.Database.Secret.Subscription)
	}
	if cfg.Database.Secret.UserRef != "switchx-db-test-user" {
		t.Errorf("UserRef = %q", cfg.Database.Secret.UserRef)
	}
	if cfg.Database.Secret.PasswordRef != "switchx-db-test-password" {
		t.Errorf("PasswordRef = %q", cfg.Database.Secret.PasswordRef)
	}
	if cfg.User.Timezone != "Europe/Brussels" {
		t.Errorf("Timezone = %q", cfg.User.Timezone)
	}
	if cfg.UI.Editor != "nvim" {
		t.Errorf("Editor = %q", cfg.UI.Editor)
	}
}

func TestLoad_XDGOverride(t *testing.T) {
	// Confirms Load reads from XDG_CONFIG_HOME, not from $HOME/.config.
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", "/nonexistent-home-should-not-be-read")
	installFile(t, dir, "valid.toml", 0o600)
	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}
