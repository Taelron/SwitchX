package config

import "errors"

// Sentinel errors returned by Load. Callers branch on these with
// errors.Is; the wrapped messages carry the specific path or field.
var (
	ErrNoConfig   = errors.New("config: no config file")
	ErrWrongMode  = errors.New("config: file mode is too permissive")
	ErrMalformed  = errors.New("config: malformed TOML")
	ErrIncomplete = errors.New("config: missing required field")
)

// Config is the parsed switchx configuration. Loaded by Load from the
// TOML file at the XDG config path.
type Config struct {
	Database Database `toml:"database"`
	User     User     `toml:"user"`
	UI       UI       `toml:"ui"`
}

// Database holds the Postgres connection parameters and the secret
// reference used to fetch the password at startup.
type Database struct {
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
	Name    string `toml:"name"`
	SSLMode string `toml:"sslmode"`
	Secret  Secret `toml:"secret"`
}

// Secret identifies where the database credentials live in the secret
// store. The user and password are stored as two separate secrets in
// the same vault under the same Azure subscription. The provider adapter
// resolves both values at runtime (see ADR-0005).
type Secret struct {
	Provider     string `toml:"provider"`
	Subscription string `toml:"subscription"`
	Vault        string `toml:"vault"`
	UserRef      string `toml:"user_ref"`
	PasswordRef  string `toml:"password_ref"`
}

// User holds the consultant's locale. The Linux username is used directly
// for display and for Session.created_by attribution; no separate display
// name is captured.
type User struct {
	Timezone string `toml:"timezone"`
}

// UI holds preference fields for the terminal user interface.
type UI struct {
	Editor string `toml:"editor"`
}
