package bootstrap

import (
	"slices"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taelron/SwitchX/internal/config"
	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// Field indices on step 1. Order is the on-screen order. fieldDBHost,
// fieldDBPort, and fieldDBName map 1:1 to the inputs slice; fieldDBSSLMode
// is rendered as a cyclable picker (no textinput) because the set of
// valid values is closed and end users cannot be expected to recall the
// libpq names.
const (
	fieldDBHost = iota
	fieldDBPort
	fieldDBName
	fieldDBSSLMode

	dbFieldCount
	dbTextInputCount = 3
)

// validSSLModes mirrors libpq's accepted values. The picker cycles
// through these in order; the default focus value is "require".
var validSSLModes = []string{
	"disable",
	"allow",
	"prefer",
	"require",
	"verify-ca",
	"verify-full",
}

const defaultSSLModeIndex = 3 // "require"

// databaseStep is wizard step 1: gather host/port/name/sslmode.
//
// Step 1 is the first step of the wizard, so Shift+Tab on the first
// field wraps to the last field rather than retreating — there is no
// previous step to fall back to.
type databaseStep struct {
	inputs       []textinput.Model // host, port, name (length dbTextInputCount)
	sslmodeIndex int               // index into validSSLModes
	errors       []string
	focus        int
}

// newDatabaseStep builds step 1 with each field pre-filled from the
// seed config when present. Empty seed values fall back to placeholders
// (host/name) or sensible defaults (port=5432, sslmode=require). The
// sslmode picker maps the seed value to its index in validSSLModes;
// values outside the curated list silently fall back to "require".
func newDatabaseStep(seed config.Database) databaseStep {
	host := textinput.New()
	host.Placeholder = "db.example.internal"
	host.Prompt = ""
	host.Width = textInputWidth
	host.SetValue(seed.Host)
	host.Focus()

	port := textinput.New()
	port.Placeholder = "5432"
	port.Prompt = ""
	port.Width = textInputWidth
	if seed.Port > 0 {
		port.SetValue(strconv.Itoa(seed.Port))
	} else {
		port.SetValue("5432")
	}

	name := textinput.New()
	name.Placeholder = "switchx"
	name.Prompt = ""
	name.Width = textInputWidth
	name.SetValue(seed.Name)

	sslmodeIndex := defaultSSLModeIndex
	if seed.SSLMode != "" {
		if i := slices.Index(validSSLModes, seed.SSLMode); i >= 0 {
			sslmodeIndex = i
		}
	}

	return databaseStep{
		inputs:       []textinput.Model{host, port, name},
		sslmodeIndex: sslmodeIndex,
		errors:       make([]string, dbFieldCount),
	}
}

func (s databaseStep) Init() tea.Cmd { return textinput.Blink }

// Update advances the database step. The third return value is true
// when the user pressed Enter on the last field with all fields valid;
// the fourth (retreat) is always false because step 1 has no previous
// step — Shift+Tab on field 0 wraps internally.
func (s databaseStep) Update(msg tea.Msg) (databaseStep, tea.Cmd, bool, bool) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "down":
			s, cmd := s.moveFocus(1)
			return s, cmd, false, false
		case "shift+tab", "up":
			s, cmd := s.moveFocus(-1)
			return s, cmd, false, false
		case "enter":
			if s.focus < dbFieldCount-1 {
				s, cmd := s.moveFocus(1)
				return s, cmd, false, false
			}
			s, ok := s.validate()
			return s, nil, ok, false
		case "left":
			if s.focus == fieldDBSSLMode {
				s.sslmodeIndex = (s.sslmodeIndex - 1 + len(validSSLModes)) % len(validSSLModes)
				return s, nil, false, false
			}
		case "right":
			if s.focus == fieldDBSSLMode {
				s.sslmodeIndex = (s.sslmodeIndex + 1) % len(validSSLModes)
				return s, nil, false, false
			}
		}
	}

	if s.focus < dbTextInputCount {
		prev := s.inputs[s.focus].Value()
		var cmd tea.Cmd
		s.inputs[s.focus], cmd = s.inputs[s.focus].Update(msg)
		if s.inputs[s.focus].Value() != prev {
			s.errors[s.focus] = ""
		}
		return s, cmd, false, false
	}
	return s, nil, false, false
}

func (s databaseStep) View() string {
	labels := []string{"Host", "Port", "Database", "SSL mode"}

	var lines []string
	for i := range dbFieldCount {
		var widget string
		if i < dbTextInputCount {
			widget = s.inputs[i].View()
		} else {
			widget = s.sslmodePickerView()
		}
		lines = append(lines, renderRow(fieldRow{
			focused: i == s.focus,
			label:   labels[i],
			widget:  widget,
			err:     s.errors[i],
		}))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// sslmodePickerView renders the current sslmode value with chevrons that
// hint at the cyclable nature of the field when focused.
func (s databaseStep) sslmodePickerView() string {
	val := validSSLModes[s.sslmodeIndex]
	if s.focus == fieldDBSSLMode {
		return styles.FocusedMarker.Render("‹ ") + val + styles.FocusedMarker.Render(" ›")
	}
	return "  " + val + "  "
}

// focusLast positions focus on the final field of the step, used when
// the wizard returns from a later step via Shift+Tab back-navigation.
func (s databaseStep) focusLast() databaseStep {
	if s.focus < dbTextInputCount {
		s.inputs[s.focus].Blur()
	}
	s.focus = dbFieldCount - 1
	return s
}

// moveFocus blurs the current input and focuses the next one delta away
// (wrapping around). The returned tea.Cmd starts the new input's cursor
// blink — dropping it leaves the cursor frozen. The picker has no blink
// of its own, so leaving / arriving at fieldDBSSLMode produces a nil cmd.
func (s databaseStep) moveFocus(delta int) (databaseStep, tea.Cmd) {
	if s.focus < dbTextInputCount {
		s.inputs[s.focus].Blur()
	}
	s.focus = (s.focus + delta + dbFieldCount) % dbFieldCount
	if s.focus < dbTextInputCount {
		return s, s.inputs[s.focus].Focus()
	}
	return s, nil
}

// validate fills s.errors and returns the updated step plus a bool that
// is true when all fields pass. The picker can never produce an invalid
// sslmode, so it has no validation entry.
func (s databaseStep) validate() (databaseStep, bool) {
	for i := range s.errors {
		s.errors[i] = ""
	}
	ok := true
	if s.host() == "" {
		s.errors[fieldDBHost] = "host is required"
		ok = false
	}
	if msg := validatePort(s.port()); msg != "" {
		s.errors[fieldDBPort] = msg
		ok = false
	}
	if s.name() == "" {
		s.errors[fieldDBName] = "database name is required"
		ok = false
	}
	return s, ok
}

func (s databaseStep) host() string    { return trimmed(s.inputs[fieldDBHost]) }
func (s databaseStep) port() string    { return trimmed(s.inputs[fieldDBPort]) }
func (s databaseStep) name() string    { return trimmed(s.inputs[fieldDBName]) }
func (s databaseStep) sslmode() string { return validSSLModes[s.sslmodeIndex] }

func (s databaseStep) hintBarExtras() []string {
	if s.focus == fieldDBSSLMode {
		return []string{"← → cycle"}
	}
	return nil
}

// validatePort returns an empty string when the value is a valid TCP
// port and an error message otherwise.
func validatePort(v string) string {
	if v == "" {
		return "port is required"
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return "port must be a number"
	}
	if n < 1 || n > 65535 {
		return "port must be between 1 and 65535"
	}
	return ""
}
