package home

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// model is the home view's Bubble Tea model. State is intentionally
// flat: dbname is set at construction, width/height update on every
// WindowSizeMsg, showHelp toggles when the user presses `?`.
//
// Methods use value receivers per @TUI Go Conventions — Bubble Tea's
// update loop relies on copy semantics.
type model struct {
	dbname   string
	width    int
	height   int
	showHelp bool
}

func newModel(dbname string) model {
	return model{dbname: dbname}
}

func (m model) Init() tea.Cmd { return nil }

// Update handles the four key bindings the placeholder home supports
// (q / Ctrl+C / ? / Esc) plus window resizes. All other input is
// silently ignored per @UI Patterns "Unknown keys are silently
// ignored".
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "esc":
			// Esc on the root view is a no-op (no parent to return to);
			// it only closes the help overlay when one is open.
			if m.showHelp {
				m.showHelp = false
			}
			return m, nil
		}
	}
	return m, nil
}

// View renders either the bare home screen or, when the help overlay
// is open, the help modal centred on a blank background. @UI Patterns
// also calls for dimming the underlying view behind a modal; that
// requires line-level compositing Lipgloss does not provide natively
// and is deferred until a generic modal-overlay helper exists.
func (m model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}
	return m.renderHome()
}

func (m model) renderHome() string {
	breadcrumb := styles.Breadcrumb.Render("switchx › home")
	body := fmt.Sprintf("Connected to %s", m.dbname)
	hintbar := styles.HintBar.Render("q/Quit  ?/Help")

	// Vertically centre the status line in the space between the
	// breadcrumb and the hint bar so the home reads as a calm status
	// display rather than a top-heavy list.
	bodyHeight := max(1, m.height-lipgloss.Height(breadcrumb)-lipgloss.Height(hintbar))
	centred := lipgloss.Place(m.width, bodyHeight,
		lipgloss.Center, lipgloss.Center, body)

	return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, centred, hintbar)
}

// helpKeys is the context-sensitive key list the help overlay shows.
// Order mirrors the hint bar (quit before help-toggle) and Esc/Ctrl+C
// follow because they are dismiss/exit affordances rather than primary
// actions.
var helpKeys = []struct{ key, desc string }{
	{"q", "Quit"},
	{"?", "Toggle this help"},
	{"Esc", "Close this help"},
	{"Ctrl+C", "Quit"},
}

// helpOverlayWidth sizes the help modal to its content per @UI
// Patterns ("sized to content, capped at ~60% of the terminal width").
// The wizard's fixed ModalWidth=88 is for its own multi-step content
// stability; the help overlay is single-screen and tighter.
const helpOverlayWidth = 30

func (m model) renderHelp() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorEmphasis).
		Render("Help")

	rows := make([]string, 0, len(helpKeys))
	for _, k := range helpKeys {
		rows = append(rows, fmt.Sprintf("%-8s %s", k.key, k.desc))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, rows...)

	hint := styles.HintBar.Render("?/Close  Esc/Close")
	inner := lipgloss.JoinVertical(lipgloss.Left, title, "", body, hint)
	overlay := styles.Modal.Width(helpOverlayWidth).Render(inner)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, overlay)
}
