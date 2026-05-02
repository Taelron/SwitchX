package bootstrap

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// step identifies the current wizard step. Values are 1-based so they
// match the breadcrumb shown to the user.
type step int

const (
	stepDatabase    step = 1
	stepPlaceholder step = 2
	totalSteps      int  = 2
)

// accumulator holds the values gathered across wizard steps. TAE-10 only
// fills the database fields; TAE-11 grows the type to cover steps 2–4
// and persists it via the config package.
type accumulator struct {
	host    string
	port    string
	name    string
	sslmode string
}

// model is the top-level Bubble Tea model. It owns the step state
// machine and delegates field-level interaction to the active step.
type model struct {
	current  step
	database databaseStep
	acc      accumulator
	width    int
	height   int
}

func newModel() model {
	return model{
		current:  stepDatabase,
		database: newDatabaseStep(),
	}
}

func (m model) Init() tea.Cmd {
	return m.database.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		// Bubble Tea's runtime maps Ctrl+C → tea.Quit by default, so
		// we only need to handle Esc here.
		if msg.String() == "esc" {
			return m, tea.Quit
		}
	}

	switch m.current {
	case stepDatabase:
		next, cmd, advance := m.database.Update(msg)
		m.database = next
		if advance {
			m.acc.host = m.database.host()
			m.acc.port = m.database.port()
			m.acc.name = m.database.name()
			m.acc.sslmode = m.database.sslmode()
			m.current = stepPlaceholder
		}
		return m, cmd
	case stepPlaceholder:
		// No interactive elements on the placeholder; only Esc/Ctrl+C
		// exit, handled above.
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	var body string
	switch m.current {
	case stepDatabase:
		body = m.database.View()
	case stepPlaceholder:
		body = renderPlaceholder()
	}

	header := styles.Breadcrumb.Render(m.breadcrumb())
	hint := styles.HintBar.Render(m.hints())

	content := lipgloss.JoinVertical(lipgloss.Left, header, body, hint)
	modal := styles.Modal.Render(content)

	// Before the first WindowSizeMsg arrives we don't know the
	// terminal size; render the modal at its natural position.
	if m.width == 0 || m.height == 0 {
		return modal
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m model) breadcrumb() string {
	switch m.current {
	case stepDatabase:
		return "Bootstrap wizard › Step 1/2 — Database"
	case stepPlaceholder:
		return "Bootstrap wizard › Step 2/2 — (placeholder)"
	}
	return "Bootstrap wizard"
}

func (m model) hints() string {
	switch m.current {
	case stepDatabase:
		parts := []string{
			"Tab next field",
			"Shift+Tab previous",
		}
		if m.database.focus == fieldSSLMode {
			parts = append(parts, "← → cycle")
		}
		parts = append(parts, "Enter advance", "Esc quit")
		return strings.Join(parts, " · ")
	case stepPlaceholder:
		return "Esc quit"
	}
	return "Esc quit"
}

func renderPlaceholder() string {
	return "Step 2 — bootstrap wizard continues in TAE-11.\n" +
		"Press Esc to exit."
}
