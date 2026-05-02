package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ErrSaveFailed wraps any failure path inside Save (mkdir, encode, write,
// rename). Callers branch on this with errors.Is.
var ErrSaveFailed = errors.New("config: save failed")

// LoadPartial reads the config file if it exists and returns whatever
// fields it contains. Unlike Load it never returns an error: a missing
// file, a malformed file, and a file failing the 0o600 check all yield
// an empty *Config.
//
// The bootstrap wizard uses this to seed its form fields from any
// previously-written config so the user does not have to re-enter
// values that were already correct (e.g. after a connect-validate
// failure that was followed by Esc and a re-launch).
func LoadPartial() *Config {
	var cfg Config
	path, err := Path()
	if err != nil {
		return &cfg
	}
	_, _ = toml.DecodeFile(path, &cfg)
	return &cfg
}

// Save serialises cfg to TOML and writes it to the resolved config path
// (see Path) atomically: it writes to "<path>.tmp" with mode 0600, then
// renames into place. The parent directory is created with mode 0700 if
// it does not already exist.
//
// Atomicity matters because the bootstrap wizard runs Save after a
// successful connect-validate; a partially-written file would leave the
// next Load failing with ErrMalformed and the user unable to start the
// app without manual cleanup.
//
// Mode 0600 on the final file is enforced by Load's existing permission
// check (see ErrWrongMode); writing at 0600 keeps a fresh write
// compatible with that check on the next launch.
func Save(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: nil config", ErrSaveFailed)
	}
	path, err := Path()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSaveFailed, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("%w: create parent dir: %v", ErrSaveFailed, err)
	}

	tmp := path + ".tmp"
	// #nosec G304 -- tmp is derived from Path() (XDG_CONFIG_HOME or
	// $HOME/.config/switchx/config.toml), not from untrusted input.
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("%w: open tmp: %v", ErrSaveFailed, err)
	}
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("%w: encode: %v", ErrSaveFailed, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%w: close tmp: %v", ErrSaveFailed, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%w: rename: %v", ErrSaveFailed, err)
	}
	return nil
}
