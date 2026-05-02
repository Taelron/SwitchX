package bootstrap

import (
	"os"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// commonEditors is the curated list shown in the step-4 picker. The
// goal mirrors step 3's timezone picker — cover the editors a typical
// developer reaches for, and let $EDITOR override / extend the list at
// runtime so users with anything custom still find their choice.
var commonEditors = []string{
	"vi",
	"vim",
	"nvim",
	"nano",
	"emacs",
	"micro",
	"helix",
}

// preferencesStep is wizard step 4: pick the editor via an inline
// chevron picker. The user's $EDITOR is prepended (and pre-selected)
// when it isn't already in the list. Step 4 is the last form step;
// advancing from here triggers the connect-validate state in the
// top-level model.
type preferencesStep struct {
	editors []string
	index   int
}

// newPreferencesStep builds the editor picker. Selection priority
// mirrors step 3: seed first, then $EDITOR, then commonEditors[0].
// Values outside the curated list are prepended.
func newPreferencesStep(seedEditor string) preferencesStep {
	editors := append([]string{}, commonEditors...)
	idx := 0

	pickOrPrepend := func(name string) {
		if name == "" {
			return
		}
		if i := slices.Index(editors, name); i >= 0 {
			idx = i
			return
		}
		editors = append([]string{name}, editors...)
		idx = 0
	}

	if seedEditor != "" {
		pickOrPrepend(seedEditor)
	} else {
		pickOrPrepend(os.Getenv("EDITOR"))
	}
	return preferencesStep{editors: editors, index: idx}
}

func (s preferencesStep) Init() tea.Cmd { return nil }

func (s preferencesStep) Update(msg tea.Msg) (preferencesStep, tea.Cmd, bool, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil, false, false
	}
	switch key.String() {
	case "shift+tab", "up":
		return s, nil, false, true
	case "enter", "tab", "down":
		// Picker is always valid (closed enum); advancing on any
		// forward key never produces a validation error.
		return s, nil, true, false
	case "left":
		s.index = (s.index - 1 + len(s.editors)) % len(s.editors)
		return s, nil, false, false
	case "right":
		s.index = (s.index + 1) % len(s.editors)
		return s, nil, false, false
	}
	return s, nil, false, false
}

func (s preferencesStep) View() string {
	val := s.editors[s.index]
	widget := styles.FocusedMarker.Render("‹ ") + val + styles.FocusedMarker.Render(" ›")
	return renderRow(fieldRow{
		focused: true,
		label:   "Editor",
		widget:  widget,
	})
}

func (s preferencesStep) value() string { return s.editors[s.index] }
