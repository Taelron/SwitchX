package bootstrap

import (
	"strings"
	"testing"

	"github.com/Taelron/SwitchX/internal/config"
)

func TestSecretValidateRejectsEmptyFields(t *testing.T) {
	s := newSecretStep(config.Secret{})
	s, ok := s.validate()
	if ok {
		t.Fatal("validate() = true with all fields empty, want false")
	}
	for i := range secretFieldCount {
		if s.errors[i] == "" {
			t.Errorf("errors[%d] empty, want a message", i)
		}
	}
}

func TestSecretValidatePassesWithAllFields(t *testing.T) {
	s := newSecretStep(config.Secret{})
	s.inputs[fieldSecretSubscription].SetValue("00000000-0000-0000-0000-000000000000")
	s.inputs[fieldSecretVault].SetValue("vault")
	s.inputs[fieldSecretUserRef].SetValue("user-ref")
	s.inputs[fieldSecretPasswordRef].SetValue("password-ref")
	if _, ok := s.validate(); !ok {
		t.Fatalf("validate() = false with valid inputs, errors=%v", s.errors)
	}
}

func TestSecretShiftTabOnFirstFieldRetreats(t *testing.T) {
	s := newSecretStep(config.Secret{})
	if s.focus != 0 {
		t.Fatalf("precondition: focus=%d", s.focus)
	}
	_, _, _, retreat := s.Update(keyMsg("shift+tab"))
	if !retreat {
		t.Error("Shift+Tab on field 0 did not signal retreat")
	}
}

func TestSecretShiftTabOnNonFirstFieldDoesNotRetreat(t *testing.T) {
	s := newSecretStep(config.Secret{})
	s.focus = fieldSecretVault
	s, _, _, retreat := s.Update(keyMsg("shift+tab"))
	if retreat {
		t.Error("Shift+Tab on field 1 should move focus, not retreat")
	}
	if s.focus != fieldSecretSubscription {
		t.Errorf("focus = %d, want %d", s.focus, fieldSecretSubscription)
	}
}

func TestSecretViewIncludesProviderLabel(t *testing.T) {
	s := newSecretStep(config.Secret{})
	view := s.View()
	if !strings.Contains(view, "Provider") {
		t.Errorf("view missing 'Provider' label")
	}
	if !strings.Contains(view, providerLabel) {
		t.Errorf("view missing provider value %q", providerLabel)
	}
}
