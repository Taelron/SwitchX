package bootstrap

import (
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// commonTimezones is the curated list shown in the step-3 picker. The
// goal is "covers 95% of consulting team locations" rather than full
// IANA coverage — the inline chevron picker would be unusable scrolled
// across all 600 IANA zones. The system's resolved timezone is
// prepended at runtime if it falls outside this list, so the user
// always finds their own zone in the cycle.
var commonTimezones = []string{
	"UTC",
	"Europe/Brussels",
	"Europe/London",
	"Europe/Paris",
	"Europe/Berlin",
	"Europe/Madrid",
	"Europe/Rome",
	"Europe/Amsterdam",
	"Europe/Dublin",
	"Europe/Lisbon",
	"Europe/Zurich",
	"Europe/Vienna",
	"Europe/Stockholm",
	"Europe/Athens",
	"Europe/Warsaw",
	"America/New_York",
	"America/Chicago",
	"America/Denver",
	"America/Los_Angeles",
	"America/Toronto",
	"America/Vancouver",
	"America/Sao_Paulo",
	"Asia/Tokyo",
	"Asia/Shanghai",
	"Asia/Singapore",
	"Asia/Dubai",
	"Asia/Kolkata",
	"Asia/Seoul",
	"Australia/Sydney",
	"Australia/Melbourne",
	"Pacific/Auckland",
}

// identityStep is wizard step 3: pick the user's timezone via an inline
// chevron picker over commonTimezones. The system's resolved timezone
// is prepended (and pre-selected) when it isn't already in the list.
//
// A picker is the right widget here because the value set is closed
// (Go's time.LoadLocation either accepts a name or rejects it; there
// is nothing in between) and end users cannot be expected to recall
// IANA strings. The same primitive is used on step 1's sslmode field.
type identityStep struct {
	zones []string
	index int
}

// newIdentityStep builds the timezone picker. Selection priority:
//
//  1. seed value if present and loadable;
//  2. system timezone via time.Local;
//  3. first entry of commonTimezones.
//
// Values not already in the curated list are prepended so the user
// always finds the relevant option without scrolling.
func newIdentityStep(seedTimezone string) identityStep {
	zones := append([]string{}, commonTimezones...)
	idx := 0

	prepend := func(name string) {
		zones = append([]string{name}, zones...)
		idx = 0
	}
	pickOrPrepend := func(name string) {
		if name == "" {
			return
		}
		if i := slices.Index(zones, name); i >= 0 {
			idx = i
			return
		}
		prepend(name)
	}

	if seedTimezone != "" {
		pickOrPrepend(seedTimezone)
	} else {
		pickOrPrepend(defaultTimezone())
	}
	return identityStep{zones: zones, index: idx}
}

func (s identityStep) Init() tea.Cmd { return nil }

func (s identityStep) Update(msg tea.Msg) (identityStep, tea.Cmd, bool, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil, false, false
	}
	switch key.String() {
	case "shift+tab", "up":
		return s, nil, false, true
	case "enter", "tab", "down":
		// The picker is always valid (closed enum), so advancing on
		// any forward key never produces a validation error.
		return s, nil, true, false
	case "left":
		s.index = (s.index - 1 + len(s.zones)) % len(s.zones)
		return s, nil, false, false
	case "right":
		s.index = (s.index + 1) % len(s.zones)
		return s, nil, false, false
	}
	return s, nil, false, false
}

func (s identityStep) View() string {
	val := s.zones[s.index]
	widget := styles.FocusedMarker.Render("‹ ") + val + styles.FocusedMarker.Render(" ›")
	return renderRow(fieldRow{
		focused: true,
		label:   "Timezone",
		widget:  widget,
	})
}

// focusLast is a no-op for the picker — it has no internal focus
// state — but is implemented so the model's back-nav routing can call
// it uniformly with the other steps.
func (s identityStep) focusLast() identityStep { return s }

func (s identityStep) value() string { return s.zones[s.index] }

// defaultTimezone returns the OS-resolved timezone name, or "" when Go
// cannot determine one. time.Local.String() returns "Local" when no
// IANA name is resolvable; that string is useless for users so we
// suppress it.
func defaultTimezone() string {
	name := time.Local.String()
	if name == "" || name == "Local" {
		return ""
	}
	return name
}
