package bootstrap

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// Field indices. Order is the on-screen order. fieldHost, fieldPort and
// fieldName map 1:1 to the textinputs in the inputs slice; fieldSSLMode
// is rendered as a cyclable picker (no textinput) because the set of
// valid values is closed and end users cannot be expected to recall the
// libpq names.
const (
	fieldHost = iota
	fieldPort
	fieldName
	fieldSSLMode

	fieldCount
)

// textInputCount is the number of fields backed by a textinput.Model.
// The remaining fields up to fieldCount are custom widgets.
const textInputCount = 3

// Column widths for the field rows. They sum to ModalWidth minus the
// modal's horizontal padding so each row fills the content area exactly
// and the right-hand error column stays at a fixed offset regardless of
// which field is focused.
const (
	markerColumnWidth = 2
	labelColumnWidth  = 12
	widgetColumnWidth = 26
	errorColumnWidth  = styles.ModalWidth - 4 - markerColumnWidth - labelColumnWidth - widgetColumnWidth
	textInputWidth    = widgetColumnWidth - 2
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
type databaseStep struct {
	inputs       []textinput.Model // host, port, name (length textInputCount)
	sslmodeIndex int               // index into validSSLModes
	errors       []string
	focus        int
}

func newDatabaseStep() databaseStep {
	host := textinput.New()
	host.Placeholder = "db.example.internal"
	host.Prompt = ""
	host.Width = textInputWidth
	host.Focus()

	port := textinput.New()
	port.Placeholder = "5432"
	port.Prompt = ""
	port.Width = textInputWidth
	port.SetValue("5432")

	name := textinput.New()
	name.Placeholder = "switchx"
	name.Prompt = ""
	name.Width = textInputWidth

	return databaseStep{
		inputs:       []textinput.Model{host, port, name},
		sslmodeIndex: defaultSSLModeIndex,
		errors:       make([]string, fieldCount),
	}
}

func (s databaseStep) Init() tea.Cmd { return textinput.Blink }

// Update advances the database step. The third return value indicates
// whether the wizard should advance to the next step. It is true only
// when the user pressed Enter on the last field with all fields valid.
func (s databaseStep) Update(msg tea.Msg) (databaseStep, tea.Cmd, bool) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "down":
			s, cmd := s.moveFocus(1)
			return s, cmd, false
		case "shift+tab", "up":
			s, cmd := s.moveFocus(-1)
			return s, cmd, false
		case "enter":
			if s.focus < fieldCount-1 {
				s, cmd := s.moveFocus(1)
				return s, cmd, false
			}
			s, ok := s.validate()
			return s, nil, ok
		case "left":
			if s.focus == fieldSSLMode {
				s.sslmodeIndex = (s.sslmodeIndex - 1 + len(validSSLModes)) % len(validSSLModes)
				return s, nil, false
			}
		case "right":
			if s.focus == fieldSSLMode {
				s.sslmodeIndex = (s.sslmodeIndex + 1) % len(validSSLModes)
				return s, nil, false
			}
		}
	}

	// Forward the message to the focused widget. The picker has no
	// further key handling beyond the cases above, so messages aimed at
	// it (mouse, paste, etc.) are simply ignored.
	//
	// When the user actually changes the focused input's value, clear
	// any stale validation error on that field so feedback stays in
	// sync with what's typed.
	if s.focus < textInputCount {
		prev := s.inputs[s.focus].Value()
		var cmd tea.Cmd
		s.inputs[s.focus], cmd = s.inputs[s.focus].Update(msg)
		if s.inputs[s.focus].Value() != prev {
			s.errors[s.focus] = ""
		}
		return s, cmd, false
	}
	return s, nil, false
}

func (s databaseStep) View() string {
	labels := []string{"Host", "Port", "Database", "SSL mode"}

	labelStyle := styles.FieldLabel.Width(labelColumnWidth)
	widgetCol := lipgloss.NewStyle().Width(widgetColumnWidth)
	errorStyle := styles.FieldError.Width(errorColumnWidth).PaddingLeft(2)

	var lines []string
	for i := range fieldCount {
		marker := "  "
		if i == s.focus {
			marker = styles.FocusedMarker.Render("▸ ")
		}
		var widget string
		if i < textInputCount {
			widget = s.inputs[i].View()
		} else {
			widget = s.sslmodePickerView()
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			marker,
			labelStyle.Render(labels[i]+":"),
			widgetCol.Render(widget),
			errorStyle.Render(s.errors[i]),
		)
		lines = append(lines, row)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// sslmodePickerView renders the current sslmode value with chevrons that
// hint at the cyclable nature of the field when focused.
func (s databaseStep) sslmodePickerView() string {
	val := validSSLModes[s.sslmodeIndex]
	if s.focus == fieldSSLMode {
		return styles.FocusedMarker.Render("‹ ") + val + styles.FocusedMarker.Render(" ›")
	}
	return "  " + val + "  "
}

func (s databaseStep) host() string    { return strings.TrimSpace(s.inputs[fieldHost].Value()) }
func (s databaseStep) port() string    { return strings.TrimSpace(s.inputs[fieldPort].Value()) }
func (s databaseStep) name() string    { return strings.TrimSpace(s.inputs[fieldName].Value()) }
func (s databaseStep) sslmode() string { return validSSLModes[s.sslmodeIndex] }

// moveFocus blurs the current input and focuses the next one delta away
// (wrapping around). The returned tea.Cmd starts the new input's cursor
// blink — dropping it leaves the cursor frozen. The picker has no blink
// of its own, so leaving / arriving at fieldSSLMode produces a nil cmd.
func (s databaseStep) moveFocus(delta int) (databaseStep, tea.Cmd) {
	if s.focus < textInputCount {
		s.inputs[s.focus].Blur()
	}
	s.focus = (s.focus + delta + fieldCount) % fieldCount
	if s.focus < textInputCount {
		return s, s.inputs[s.focus].Focus()
	}
	return s, nil
}

// validate fills s.errors and returns the updated step plus a bool that
// is true when all fields pass. Field order matches the on-screen order
// so the first error the user sees is the topmost one. The picker can
// never produce an invalid sslmode, so it has no validation entry.
func (s databaseStep) validate() (databaseStep, bool) {
	for i := range s.errors {
		s.errors[i] = ""
	}
	ok := true
	if s.host() == "" {
		s.errors[fieldHost] = "host is required"
		ok = false
	}
	if msg := validatePort(s.port()); msg != "" {
		s.errors[fieldPort] = msg
		ok = false
	}
	if s.name() == "" {
		s.errors[fieldName] = "database name is required"
		ok = false
	}
	return s, ok
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
