package bootstrap

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"non-numeric", "abc", true},
		{"zero", "0", true},
		{"negative", "-1", true},
		{"too-large", "65536", true},
		{"min-valid", "1", false},
		{"max-valid", "65535", false},
		{"default", "5432", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePort(tt.input)
			if (got != "") != tt.wantErr {
				t.Fatalf("validatePort(%q) = %q, wantErr=%v", tt.input, got, tt.wantErr)
			}
		})
	}
}

func TestDatabaseStepValidateAggregatesErrors(t *testing.T) {
	s := newDatabaseStep()
	// Wipe the seeded port default so it also fails. host and name
	// already start empty. The picker can never produce an invalid
	// sslmode so it has no error path to exercise.
	s.inputs[fieldPort].SetValue("")
	s, ok := s.validate()
	if ok {
		t.Fatal("validate() = true with empty host/port/name, want false")
	}
	for _, i := range []int{fieldHost, fieldPort, fieldName} {
		if s.errors[i] == "" {
			t.Errorf("errors[%d] is empty, want a validation message", i)
		}
	}
	if s.errors[fieldSSLMode] != "" {
		t.Errorf("errors[fieldSSLMode] = %q, want empty (picker is always valid)", s.errors[fieldSSLMode])
	}
}

func TestDatabaseStepValidatePassesWithDefaults(t *testing.T) {
	s := newDatabaseStep()
	s.inputs[fieldHost].SetValue("db.example.internal")
	s.inputs[fieldName].SetValue("switchx")
	s, ok := s.validate()
	if !ok {
		t.Fatalf("validate() = false with valid inputs, errors=%v", s.errors)
	}
	for i, msg := range s.errors {
		if msg != "" {
			t.Errorf("errors[%d] = %q, want empty", i, msg)
		}
	}
}

// keyMsg builds a tea.KeyMsg matching what Bubble Tea would produce for
// the given key string. Special keys (tab, enter, esc, arrows) use their
// typed constants; everything else is treated as runes.
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestModelEscQuits(t *testing.T) {
	m := newModel()
	_, cmd := m.Update(keyMsg("esc"))
	if cmd == nil {
		t.Fatal("Update(esc) returned nil cmd, want tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("Update(esc) cmd produced %T, want tea.QuitMsg", msg)
	}
}

func TestModelTabCyclesFocus(t *testing.T) {
	m := newModel()
	if got := m.database.focus; got != fieldHost {
		t.Fatalf("initial focus = %d, want %d", got, fieldHost)
	}
	for i := 1; i < fieldCount; i++ {
		next, _ := m.Update(keyMsg("tab"))
		m = next.(model)
		if m.database.focus != i {
			t.Fatalf("after %d tab(s) focus = %d, want %d", i, m.database.focus, i)
		}
	}
	// One more tab wraps to 0.
	next, _ := m.Update(keyMsg("tab"))
	m = next.(model)
	if m.database.focus != 0 {
		t.Fatalf("focus after wrap = %d, want 0", m.database.focus)
	}
}

func TestModelEnterOnNonLastFieldAdvancesFocus(t *testing.T) {
	m := newModel()
	next, _ := m.Update(keyMsg("enter"))
	m = next.(model)
	if m.database.focus != fieldPort {
		t.Fatalf("focus after enter on host = %d, want %d", m.database.focus, fieldPort)
	}
	if m.current != stepDatabase {
		t.Fatalf("current step = %d, want %d (still on step 1)", m.current, stepDatabase)
	}
}

func TestModelEnterOnLastFieldWithInvalidStays(t *testing.T) {
	m := newModel()
	// Leave host blank — validate must reject and stay on step 1.
	m.database.focus = fieldSSLMode
	next, _ := m.Update(keyMsg("enter"))
	m = next.(model)
	if m.current != stepDatabase {
		t.Fatalf("step advanced despite empty host (step=%d)", m.current)
	}
	if m.database.errors[fieldHost] == "" {
		t.Fatal("expected an inline error on the host field")
	}
}

func TestModelEnterOnLastFieldWithValidAdvances(t *testing.T) {
	m := newModel()
	m.database.inputs[fieldHost].SetValue("db.example.internal")
	m.database.inputs[fieldName].SetValue("switchx")
	m.database.focus = fieldSSLMode
	next, _ := m.Update(keyMsg("enter"))
	m = next.(model)
	if m.current != stepPlaceholder {
		t.Fatalf("step = %d, want %d (placeholder)", m.current, stepPlaceholder)
	}
	// Accumulator should carry the entered values forward.
	if m.acc.host != "db.example.internal" || m.acc.name != "switchx" ||
		m.acc.port != "5432" || m.acc.sslmode != "require" {
		t.Fatalf("accumulator did not capture step-1 values: %+v", m.acc)
	}
}

func TestSSLModePickerCyclesWhenFocused(t *testing.T) {
	s := newDatabaseStep()
	s.focus = fieldSSLMode
	initial := s.sslmodeIndex

	s, _, _ = s.Update(keyMsg("right"))
	if want := (initial + 1) % len(validSSLModes); s.sslmodeIndex != want {
		t.Fatalf("right: sslmodeIndex = %d, want %d", s.sslmodeIndex, want)
	}

	s, _, _ = s.Update(keyMsg("left"))
	if s.sslmodeIndex != initial {
		t.Fatalf("left after right: sslmodeIndex = %d, want %d", s.sslmodeIndex, initial)
	}

	// Wrap-around: from index 0, left should land on the last entry.
	s.sslmodeIndex = 0
	s, _, _ = s.Update(keyMsg("left"))
	if want := len(validSSLModes) - 1; s.sslmodeIndex != want {
		t.Fatalf("left wrap: sslmodeIndex = %d, want %d", s.sslmodeIndex, want)
	}
}

func TestSSLModePickerIgnoresArrowsWhenNotFocused(t *testing.T) {
	// When focus is on a textinput, left/right are part of the
	// textinput's normal cursor handling and must not cycle the picker.
	s := newDatabaseStep()
	if s.focus != fieldHost {
		t.Fatalf("precondition: initial focus = %d, want %d", s.focus, fieldHost)
	}
	initial := s.sslmodeIndex
	s, _, _ = s.Update(keyMsg("right"))
	s, _, _ = s.Update(keyMsg("left"))
	if s.sslmodeIndex != initial {
		t.Fatalf("picker cycled while focus on host: sslmodeIndex = %d, want %d", s.sslmodeIndex, initial)
	}
}

func TestEditingFocusedFieldClearsItsError(t *testing.T) {
	s := newDatabaseStep()
	// Force errors by validating with empty host/name and bad port.
	s.inputs[fieldPort].SetValue("")
	s, _ = s.validate()
	if s.errors[fieldHost] == "" || s.errors[fieldPort] == "" || s.errors[fieldName] == "" {
		t.Fatalf("precondition: validate() did not flag empty fields: %v", s.errors)
	}

	// Type into the focused (host) field. The host error must clear,
	// but errors on other fields must remain until those are edited.
	s, _, _ = s.Update(keyMsg("a"))
	if s.errors[fieldHost] != "" {
		t.Errorf("host error not cleared after edit: %q", s.errors[fieldHost])
	}
	if s.errors[fieldPort] == "" {
		t.Error("port error cleared by editing host (should be untouched)")
	}
	if s.errors[fieldName] == "" {
		t.Error("name error cleared by editing host (should be untouched)")
	}
}

func TestPlaceholderViewMentionsTAE11(t *testing.T) {
	m := newModel()
	m.current = stepPlaceholder
	if got := m.View(); !strings.Contains(got, "TAE-11") {
		t.Fatalf("placeholder view does not mention TAE-11: %q", got)
	}
}
