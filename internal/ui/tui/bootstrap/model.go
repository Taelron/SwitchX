package bootstrap

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taelron/SwitchX/internal/config"
	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// step identifies the wizard's current state. Values 1..4 are form
// steps shown in the breadcrumb counter ("Step N/4"); stepValidating is
// transient and not counted — the breadcrumb keeps reading "Step 4/4"
// through the validate roundtrip.
type step int

const (
	stepDatabase    step = 1
	stepSecret      step = 2
	stepIdentity    step = 3
	stepPreferences step = 4
	stepValidating  step = 5

	totalSteps int = 4
)

// model is the top-level Bubble Tea model. It owns the step state
// machine, the partially-built config, and the cross-step error banner.
//
// Field types stay value (not pointer) so Update returns a fresh model
// every call per the @TUI Go Conventions baseline; the *config.Config
// pointer is the one place we share mutable state and is replaced
// outright if a step rebuilds its slice of the config.
type model struct {
	current step

	database    databaseStep
	secret      secretStep
	identity    identityStep
	preferences preferencesStep
	validate    validateStep

	cfg         *config.Config
	finalCfg    *config.Config
	bannerError string

	// validateGen distinguishes results from successive validate runs.
	// If the user retries (validate fails → user fixes → presses Enter
	// again), an in-flight result from the previous run could otherwise
	// race the new one and surface a stale outcome. Each transition
	// into stepValidating bumps the counter; results carry the
	// generation that produced them and are dropped if the counter has
	// moved on.
	validateGen int

	width, height int

	ctx       context.Context
	validator Validator
	save      func(*config.Config) error
}

func newModel(ctx context.Context, validator Validator, save func(*config.Config) error, seed *config.Config) model {
	if seed == nil {
		seed = &config.Config{}
	}
	return model{
		current:     stepDatabase,
		database:    newDatabaseStep(seed.Database),
		secret:      newSecretStep(seed.Database.Secret),
		identity:    newIdentityStep(seed.User.Timezone),
		preferences: newPreferencesStep(seed.UI.Editor),
		validate:    newValidateStep(),
		cfg:         &config.Config{},
		ctx:         ctx,
		validator:   validator,
		save:        save,
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
		// Bubble Tea's runtime maps Ctrl+C → tea.Quit by default; we
		// only handle Esc here. Esc on any step (including validate)
		// quits with no file written.
		if msg.String() == "esc" {
			return m, tea.Quit
		}
	case validateResultMsg:
		return m.handleValidateResult(msg)
	}

	switch m.current {
	case stepDatabase:
		next, cmd, advance, _ := m.database.Update(msg)
		m.database = next
		if advance {
			m.cfg.Database.Host = m.database.host()
			m.cfg.Database.Port = mustAtoi(m.database.port())
			m.cfg.Database.Name = m.database.name()
			m.cfg.Database.SSLMode = m.database.sslmode()
			m.bannerError = ""
			m.current = stepSecret
		}
		return m, cmd
	case stepSecret:
		next, cmd, advance, retreat := m.secret.Update(msg)
		m.secret = next
		if retreat {
			m.current = stepDatabase
			m.database = m.database.focusLast()
			return m, nil
		}
		if advance {
			m.cfg.Database.Secret.Provider = providerTOMLValue
			m.cfg.Database.Secret.Subscription = m.secret.subscription()
			m.cfg.Database.Secret.Vault = m.secret.vault()
			m.cfg.Database.Secret.UserRef = m.secret.userRef()
			m.cfg.Database.Secret.PasswordRef = m.secret.passwordRef()
			m.bannerError = ""
			m.current = stepIdentity
		}
		return m, cmd
	case stepIdentity:
		next, cmd, advance, retreat := m.identity.Update(msg)
		m.identity = next
		if retreat {
			m.current = stepSecret
			m.secret = m.secret.focusLast()
			return m, nil
		}
		if advance {
			m.cfg.User.Timezone = m.identity.value()
			m.bannerError = ""
			m.current = stepPreferences
		}
		return m, cmd
	case stepPreferences:
		next, cmd, advance, retreat := m.preferences.Update(msg)
		m.preferences = next
		if retreat {
			m.current = stepIdentity
			m.identity = m.identity.focusLast()
			return m, nil
		}
		if advance {
			m.cfg.UI.Editor = m.preferences.value()
			m.bannerError = ""
			m.current = stepValidating
			m.validateGen++
			return m, tea.Batch(
				m.validate.Init(),
				m.validate.startCmd(m.ctx, m.validator, m.cfg, m.validateGen),
			)
		}
		return m, cmd
	case stepValidating:
		next, cmd := m.validate.Update(msg)
		m.validate = next
		return m, cmd
	}
	return m, nil
}

// handleValidateResult dispatches the outcome of the Validator. Stale
// results (from a previous run the user has since superseded) are
// dropped silently. Failure puts the user back on step 4 (the closest
// form step) with the error in the full-width banner; success persists
// the config and quits.
func (m model) handleValidateResult(msg validateResultMsg) (tea.Model, tea.Cmd) {
	if msg.generation != m.validateGen {
		return m, nil
	}
	if msg.err != nil {
		m.bannerError = msg.err.Error()
		m.current = stepPreferences
		return m, nil
	}
	if err := m.save(m.cfg); err != nil {
		// Save failure after a successful validate is recoverable —
		// the user can usually fix file permissions externally and
		// retry by pressing Enter on step 4 again.
		m.bannerError = "save: " + err.Error()
		m.current = stepPreferences
		return m, nil
	}
	m.finalCfg = m.cfg
	return m, tea.Quit
}

func (m model) View() string {
	body := m.body()

	header := styles.Breadcrumb.Render(m.breadcrumb())
	parts := []string{header, body}
	if m.bannerError != "" {
		parts = append(parts, styles.ErrorBanner.Render(m.bannerError))
	}
	parts = append(parts, styles.HintBar.Render(m.hints()))

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	modal := styles.Modal.Render(content)

	if m.width == 0 || m.height == 0 {
		return modal
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m model) body() string {
	switch m.current {
	case stepDatabase:
		return m.database.View()
	case stepSecret:
		return m.secret.View()
	case stepIdentity:
		return m.identity.View()
	case stepPreferences:
		return m.preferences.View()
	case stepValidating:
		return m.validate.View()
	}
	return ""
}

func (m model) breadcrumb() string {
	titles := map[step]string{
		stepDatabase:    "Database",
		stepSecret:      "Secret store",
		stepIdentity:    "Identity",
		stepPreferences: "Preferences",
	}
	// Validating reuses the step-4 label; user knows the wizard is
	// finalising step 4 rather than entering a new step.
	displayed := m.current
	if m.current == stepValidating {
		displayed = stepPreferences
	}
	return "Bootstrap wizard › Step " + itoa(int(displayed)) + "/" + itoa(totalSteps) +
		" — " + titles[displayed]
}

func (m model) hints() string {
	if m.current == stepValidating {
		return "Esc cancel"
	}
	// When a connect-validate failure has been surfaced, change the
	// hint bar so the user immediately sees how to recover: go back to
	// fix something or retry the same values. The default
	// "Tab/Shift+Tab fields" hint is misleading at that point because
	// the user's question is "what do I do now?" not "how do I move
	// between fields?".
	if m.bannerError != "" && m.current == stepPreferences {
		return "Shift+Tab back · Enter retry · Esc quit"
	}
	parts := []string{"Tab next field", "Shift+Tab previous"}
	if m.current == stepDatabase {
		parts = append(parts, m.database.hintBarExtras()...)
	}
	if m.current == stepIdentity || m.current == stepPreferences {
		parts = append(parts, "← → cycle")
	}
	parts = append(parts, "Enter advance", "Esc quit")
	return strings.Join(parts, " · ")
}

// mustAtoi parses an int that has already been validated by the step;
// any error here is a programmer mistake (validation skipped before
// advance) and should panic loud rather than silently zero the port.
func mustAtoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			panic("bootstrap: non-numeric port reached model after validate: " + s)
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// itoa is a tiny inline replacement for strconv.Itoa to keep the
// breadcrumb builder allocation-free and to avoid pulling strconv into
// model.go just for one digit.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [4]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
