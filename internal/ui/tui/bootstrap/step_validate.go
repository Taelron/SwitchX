package bootstrap

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taelron/SwitchX/internal/config"
	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// validateResultMsg carries the outcome of the connect-validate cmd.
// nil err means the secrets fetched and the database is reachable. The
// generation field lets the model drop stale results from a previous
// validate run that the user has since superseded by editing fields and
// pressing Enter again before the in-flight goroutine returned.
type validateResultMsg struct {
	generation int
	err        error
}

// validateStep is the transient state shown while the wizard runs the
// Validator: a spinner and a "Validating connection..." line. It has no
// fields the user can edit and no breadcrumb position of its own (the
// breadcrumb keeps reading "Step 4/4" through the validate).
type validateStep struct {
	spinner spinner.Model
}

func newValidateStep() validateStep {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(styles.ColorInfo)
	return validateStep{spinner: sp}
}

// startCmd returns the tea.Cmd that runs the Validator and posts
// validateResultMsg. The generation argument tags the result so the
// model can drop stale results from a superseded run; see
// validateResultMsg. The wizard-level context cancels the underlying
// KV / DB calls if the user Esc's out of the wizard while validation
// is in flight.
func (s validateStep) startCmd(ctx context.Context, validate Validator, cfg *config.Config, generation int) tea.Cmd {
	return func() tea.Msg {
		return validateResultMsg{generation: generation, err: validate(ctx, cfg)}
	}
}

// Update only handles spinner ticks; the validateResultMsg is consumed
// by the top-level model so it can drive the state transition + Save.
func (s validateStep) Update(msg tea.Msg) (validateStep, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

func (s validateStep) View() string {
	return s.spinner.View() + " Validating connection..."
}

// Init kicks off the spinner ticks. The validate cmd itself is started
// by the model when transitioning into stepValidating so it can pass the
// wizard context and the assembled *config.Config.
func (s validateStep) Init() tea.Cmd {
	return s.spinner.Tick
}
