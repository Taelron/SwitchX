package bootstrap

import (
	"testing"

	"github.com/Taelron/SwitchX/internal/config"
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

func TestDatabaseValidateAggregatesErrors(t *testing.T) {
	s := newDatabaseStep(config.Database{})
	s.inputs[fieldDBPort].SetValue("")
	s, ok := s.validate()
	if ok {
		t.Fatal("validate() = true with empty fields, want false")
	}
	for _, i := range []int{fieldDBHost, fieldDBPort, fieldDBName} {
		if s.errors[i] == "" {
			t.Errorf("errors[%d] empty, want a message", i)
		}
	}
	if s.errors[fieldDBSSLMode] != "" {
		t.Errorf("errors[fieldDBSSLMode] = %q, want empty (picker is always valid)", s.errors[fieldDBSSLMode])
	}
}

func TestDatabaseValidatePassesWithValidInputs(t *testing.T) {
	s := newDatabaseStep(config.Database{})
	s.inputs[fieldDBHost].SetValue("db.example.internal")
	s.inputs[fieldDBName].SetValue("switchx")
	if _, ok := s.validate(); !ok {
		t.Fatalf("validate() = false with valid inputs, errors=%v", s.errors)
	}
}

func TestSSLModePickerCyclesWhenFocused(t *testing.T) {
	s := newDatabaseStep(config.Database{})
	s.focus = fieldDBSSLMode
	initial := s.sslmodeIndex

	s, _, _, _ = s.Update(keyMsg("right"))
	if want := (initial + 1) % len(validSSLModes); s.sslmodeIndex != want {
		t.Fatalf("right: sslmodeIndex = %d, want %d", s.sslmodeIndex, want)
	}

	s, _, _, _ = s.Update(keyMsg("left"))
	if s.sslmodeIndex != initial {
		t.Fatalf("left after right: sslmodeIndex = %d, want %d", s.sslmodeIndex, initial)
	}

	s.sslmodeIndex = 0
	s, _, _, _ = s.Update(keyMsg("left"))
	if want := len(validSSLModes) - 1; s.sslmodeIndex != want {
		t.Fatalf("left wrap: sslmodeIndex = %d, want %d", s.sslmodeIndex, want)
	}
}

func TestSSLModePickerIgnoresArrowsWhenNotFocused(t *testing.T) {
	s := newDatabaseStep(config.Database{})
	if s.focus != fieldDBHost {
		t.Fatalf("precondition: focus = %d", s.focus)
	}
	initial := s.sslmodeIndex
	s, _, _, _ = s.Update(keyMsg("right"))
	s, _, _, _ = s.Update(keyMsg("left"))
	if s.sslmodeIndex != initial {
		t.Fatalf("picker cycled while focus on host: %d", s.sslmodeIndex)
	}
}

func TestDatabaseStepSeedPreFillsFields(t *testing.T) {
	seed := config.Database{
		Host:    "db.example.internal",
		Port:    6543,
		Name:    "switchx_seed",
		SSLMode: "verify-full",
	}
	s := newDatabaseStep(seed)
	if got := s.host(); got != seed.Host {
		t.Errorf("host = %q, want %q", got, seed.Host)
	}
	if got := s.port(); got != "6543" {
		t.Errorf("port = %q, want %q", got, "6543")
	}
	if got := s.name(); got != seed.Name {
		t.Errorf("name = %q, want %q", got, seed.Name)
	}
	if got := s.sslmode(); got != seed.SSLMode {
		t.Errorf("sslmode = %q, want %q", got, seed.SSLMode)
	}
}

func TestDatabaseStepSeedFallsBackToDefaultsWhenEmpty(t *testing.T) {
	s := newDatabaseStep(config.Database{})
	if got := s.port(); got != "5432" {
		t.Errorf("port = %q, want default 5432", got)
	}
	if got := s.sslmode(); got != "require" {
		t.Errorf("sslmode = %q, want default require", got)
	}
}

func TestEditingFocusedFieldClearsItsError(t *testing.T) {
	s := newDatabaseStep(config.Database{})
	s.inputs[fieldDBPort].SetValue("")
	s, _ = s.validate()
	if s.errors[fieldDBHost] == "" {
		t.Fatal("precondition: validate did not flag empty host")
	}
	s, _, _, _ = s.Update(keyMsg("a"))
	if s.errors[fieldDBHost] != "" {
		t.Errorf("editing host did not clear error: %q", s.errors[fieldDBHost])
	}
	if s.errors[fieldDBPort] == "" {
		t.Error("port error cleared by editing host (should be untouched)")
	}
}
