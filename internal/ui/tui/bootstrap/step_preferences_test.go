package bootstrap

import (
	"slices"
	"testing"
)

func TestPreferencesPickerCyclesThroughEditors(t *testing.T) {
	s := newPreferencesStep("")
	if len(s.editors) == 0 {
		t.Fatal("preferencesStep editors list is empty")
	}
	initial := s.index

	s, _, _, _ = s.Update(keyMsg("right"))
	if want := (initial + 1) % len(s.editors); s.index != want {
		t.Errorf("right: index = %d, want %d", s.index, want)
	}

	s, _, _, _ = s.Update(keyMsg("left"))
	if s.index != initial {
		t.Errorf("left after right: index = %d, want %d", s.index, initial)
	}

	s.index = 0
	s, _, _, _ = s.Update(keyMsg("left"))
	if want := len(s.editors) - 1; s.index != want {
		t.Errorf("left wrap: index = %d, want %d", s.index, want)
	}
}

func TestPreferencesShiftTabRetreats(t *testing.T) {
	s := newPreferencesStep("")
	_, _, _, retreat := s.Update(keyMsg("shift+tab"))
	if !retreat {
		t.Error("Shift+Tab did not signal retreat")
	}
}

func TestPreferencesEnterAdvances(t *testing.T) {
	s := newPreferencesStep("")
	_, _, advance, _ := s.Update(keyMsg("enter"))
	if !advance {
		t.Error("Enter did not signal advance from picker")
	}
}

func TestPreferencesEditorEnvPrependedWhenUnknown(t *testing.T) {
	t.Setenv("EDITOR", "kakoune")
	s := newPreferencesStep("")
	if got := s.editors[0]; got != "kakoune" {
		t.Errorf("first editor = %q, want kakoune (prepended from $EDITOR)", got)
	}
	if s.index != 0 {
		t.Errorf("index = %d, want 0 (pre-selected to prepended editor)", s.index)
	}
	if !slices.Contains(s.editors, "vim") {
		t.Error("editors list lost vim after prepend")
	}
}

func TestPreferencesEditorEnvSelectsExisting(t *testing.T) {
	t.Setenv("EDITOR", "nvim")
	s := newPreferencesStep("")
	if got := s.editors[s.index]; got != "nvim" {
		t.Errorf("selected editor = %q, want nvim", got)
	}
}

func TestCommonEditorsContainsVi(t *testing.T) {
	for _, want := range []string{"vi", "vim", "nvim", "nano"} {
		if !slices.Contains(commonEditors, want) {
			t.Errorf("commonEditors missing %q", want)
		}
	}
}
