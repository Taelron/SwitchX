package bootstrap

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taelron/SwitchX/internal/config"
)

// providerLabel is the static text shown for the provider row on step 2.
// M1 ships with Azure Key Vault as the only provider; the row is
// non-focusable because there is nothing to choose. If a second provider
// is added later, this becomes a 2-option picker reusing the inline
// picker primitive from step 1.
const providerLabel = "Azure Key Vault"

// providerTOMLValue is what gets emitted to the persisted config under
// database.secret.provider. Matches the value Load expects.
const providerTOMLValue = "azurekeyvault"

// Field indices on step 2. Order is the on-screen focus order. The
// static "Provider" row is rendered between the breadcrumb and the
// subscription field but is not in the focus list.
const (
	fieldSecretSubscription = iota
	fieldSecretVault
	fieldSecretUserRef
	fieldSecretPasswordRef

	secretFieldCount
)

// secretStep is wizard step 2: gather the Azure Key Vault coordinates
// (subscription, vault, user_ref, password_ref) plus a static provider
// label.
//
// Field-level validation here is non-empty only; the actual KV roundtrip
// runs at the end of step 4 in the validate state and surfaces errors
// via the full-width banner — long error messages do not fit the
// per-field error column.
type secretStep struct {
	inputs []textinput.Model
	errors []string
	focus  int
}

// newSecretStep builds step 2 with each field pre-filled from the seed
// when present. The seed.Provider is ignored — provider is a static
// label in M1 and the persisted TOML key is hard-coded to
// "azurekeyvault" on advance.
func newSecretStep(seed config.Secret) secretStep {
	subscription := textinput.New()
	subscription.Placeholder = "my-subscription"
	subscription.Prompt = ""
	subscription.Width = textInputWidth
	subscription.SetValue(seed.Subscription)
	subscription.Focus()

	vault := textinput.New()
	vault.Placeholder = "switchx-vault"
	vault.Prompt = ""
	vault.Width = textInputWidth
	vault.SetValue(seed.Vault)

	userRef := textinput.New()
	userRef.Placeholder = "switchx-pg-user"
	userRef.Prompt = ""
	userRef.Width = textInputWidth
	userRef.SetValue(seed.UserRef)

	passwordRef := textinput.New()
	passwordRef.Placeholder = "switchx-pg-password"
	passwordRef.Prompt = ""
	passwordRef.Width = textInputWidth
	passwordRef.SetValue(seed.PasswordRef)

	return secretStep{
		inputs: []textinput.Model{subscription, vault, userRef, passwordRef},
		errors: make([]string, secretFieldCount),
	}
}

func (s secretStep) Init() tea.Cmd { return textinput.Blink }

// Update returns (next, cmd, advance, retreat). Shift+Tab on the first
// focusable field signals a retreat to the previous step (database).
func (s secretStep) Update(msg tea.Msg) (secretStep, tea.Cmd, bool, bool) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "down":
			s, cmd := s.moveFocus(1)
			return s, cmd, false, false
		case "shift+tab", "up":
			if s.focus == 0 {
				return s, nil, false, true
			}
			s, cmd := s.moveFocus(-1)
			return s, cmd, false, false
		case "enter":
			if s.focus < secretFieldCount-1 {
				s, cmd := s.moveFocus(1)
				return s, cmd, false, false
			}
			s, ok := s.validate()
			return s, nil, ok, false
		}
	}

	prev := s.inputs[s.focus].Value()
	var cmd tea.Cmd
	s.inputs[s.focus], cmd = s.inputs[s.focus].Update(msg)
	if s.inputs[s.focus].Value() != prev {
		s.errors[s.focus] = ""
	}
	return s, cmd, false, false
}

func (s secretStep) View() string {
	labels := []string{"Subscription", "Vault", "User ref", "Password ref"}

	rows := []string{
		// Static provider row — always rendered first, never focusable.
		renderRow(fieldRow{label: "Provider", widget: providerLabel}),
	}
	for i := range secretFieldCount {
		rows = append(rows, renderRow(fieldRow{
			focused: i == s.focus,
			label:   labels[i],
			widget:  s.inputs[i].View(),
			err:     s.errors[i],
		}))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (s secretStep) focusLast() secretStep {
	s.inputs[s.focus].Blur()
	s.focus = secretFieldCount - 1
	s.inputs[s.focus].Focus()
	return s
}

func (s secretStep) moveFocus(delta int) (secretStep, tea.Cmd) {
	s.inputs[s.focus].Blur()
	s.focus = (s.focus + delta + secretFieldCount) % secretFieldCount
	return s, s.inputs[s.focus].Focus()
}

func (s secretStep) validate() (secretStep, bool) {
	for i := range s.errors {
		s.errors[i] = ""
	}
	type check struct {
		idx     int
		val     string
		message string
	}
	checks := []check{
		{fieldSecretSubscription, s.subscription(), "subscription is required"},
		{fieldSecretVault, s.vault(), "vault is required"},
		{fieldSecretUserRef, s.userRef(), "user_ref is required"},
		{fieldSecretPasswordRef, s.passwordRef(), "password_ref is required"},
	}
	ok := true
	for _, c := range checks {
		if c.val == "" {
			s.errors[c.idx] = c.message
			ok = false
		}
	}
	return s, ok
}

func (s secretStep) subscription() string { return trimmed(s.inputs[fieldSecretSubscription]) }
func (s secretStep) vault() string        { return trimmed(s.inputs[fieldSecretVault]) }
func (s secretStep) userRef() string      { return trimmed(s.inputs[fieldSecretUserRef]) }
func (s secretStep) passwordRef() string  { return trimmed(s.inputs[fieldSecretPasswordRef]) }
