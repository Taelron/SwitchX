package errscreen

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// model is the error screen's Bubble Tea model. State is intentionally
// flat: msg is set once at construction, width/height update on every
// WindowSizeMsg, decision flips to DecisionRetry only on `r` and is
// read by Run after the program exits.
//
// No help overlay: the hint bar (`r/Retry  q/Quit`) is the full
// keymap. Hiding additional keys behind `?` would surprise users on a
// recovery surface where predictability matters most — the wizard
// makes the same call for the same reason.
type model struct {
	msg      string
	decision Decision
	width    int
	height   int
}

func newModel(msg string) model {
	return model{msg: msg}
}

func (m model) Init() tea.Cmd { return nil }

// Update handles the two key bindings the screen supports (r and
// q/Ctrl+C). All other input is silently ignored per @UI Patterns.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.decision = DecisionRetry
			return m, tea.Quit
		case "q", "ctrl+c":
			m.decision = DecisionQuit
			return m, tea.Quit
		}
	}
	return m, nil
}

// errorMessageStyle renders the failure description in red. UI
// Patterns specifies red for errors and a "clear, plain-language
// description"; no border, no decorative chrome. The style's Width is
// set at render time so long messages wrap inside the terminal rather
// than spilling past the right edge on the 80×24 minimum.
var errorMessageStyle = lipgloss.NewStyle().
	Foreground(styles.ColorError).
	Bold(true)

// maxBodyWidth caps the message render width so an over-eager
// translateErr line still wraps inside a typical terminal. The cap
// keeps centred text from spanning the full width of a wide terminal,
// which makes the message harder to scan.
const maxBodyWidth = 76

func (m model) View() string {
	breadcrumb := styles.Breadcrumb.Render("switchx › startup")
	bodyWidth := maxBodyWidth
	if m.width > 0 && m.width-4 < bodyWidth {
		bodyWidth = max(20, m.width-4)
	}
	body := errorMessageStyle.Width(bodyWidth).Render(m.msg)
	hintbar := styles.HintBar.Render("r/Retry  q/Quit")

	// Centre the error message vertically between the breadcrumb and
	// the hint bar so the failure reads as the screen's focus, mirroring
	// the home view's centred status line.
	bodyHeight := max(1, m.height-lipgloss.Height(breadcrumb)-lipgloss.Height(hintbar))
	centred := lipgloss.Place(m.width, bodyHeight,
		lipgloss.Center, lipgloss.Center, body)

	return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, centred, hintbar)
}
