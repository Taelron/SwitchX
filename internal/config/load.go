package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Path returns the resolved config file path:
// $XDG_CONFIG_HOME/switchx/config.toml, falling back to
// $HOME/.config/switchx/config.toml when XDG_CONFIG_HOME is unset.
//
// The "no environment variables" rule from @Security & Secret Handling
// applies to switchx config field values, not to OS-level path
// resolution. Reading XDG_CONFIG_HOME and HOME for path discovery is
// expected.
func Path() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "switchx", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: cannot resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "switchx", "config.toml"), nil
}

// Load reads, validates, and returns the configuration. Errors wrap the
// package-level sentinels (ErrNoConfig, ErrWrongMode, ErrMalformed,
// ErrIncomplete) so callers can branch with errors.Is.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("%w at %s", ErrNoConfig, path)
	case err != nil:
		return nil, fmt.Errorf("config: stat %s: %w", path, err)
	}

	// File-mode check per @Security & Secret Handling baseline:
	// reject any file with group or world permission bits set.
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: %s has mode %v (run: chmod 0600 %s)",
			ErrWrongMode, path, info.Mode().Perm(), path)
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformed, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate walks the required-fields list in declaration order and
// returns an error wrapping ErrIncomplete identifying the first missing
// field. Port is considered missing when outside 1..65535.
func (c *Config) validate() error {
	type check struct {
		name string
		ok   bool
	}
	checks := []check{
		{"database.host", c.Database.Host != ""},
		{"database.port", c.Database.Port >= 1 && c.Database.Port <= 65535},
		{"database.name", c.Database.Name != ""},
		{"database.sslmode", c.Database.SSLMode != ""},
		{"database.secret.provider", c.Database.Secret.Provider != ""},
		{"database.secret.subscription", c.Database.Secret.Subscription != ""},
		{"database.secret.vault", c.Database.Secret.Vault != ""},
		{"database.secret.user_ref", c.Database.Secret.UserRef != ""},
		{"database.secret.password_ref", c.Database.Secret.PasswordRef != ""},
		{"user.timezone", c.User.Timezone != ""},
		{"ui.editor", c.UI.Editor != ""},
	}
	for _, c := range checks {
		if !c.ok {
			return fmt.Errorf("%w: %s", ErrIncomplete, c.name)
		}
	}
	return nil
}
