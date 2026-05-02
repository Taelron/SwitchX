package bootstrap

import (
	"slices"
	"testing"
	"time"
)

func TestIdentityPickerCyclesThroughZones(t *testing.T) {
	s := newIdentityStep("")
	if len(s.zones) == 0 {
		t.Fatal("identityStep zones list is empty")
	}
	initial := s.index

	s, _, _, _ = s.Update(keyMsg("right"))
	if want := (initial + 1) % len(s.zones); s.index != want {
		t.Errorf("right: index = %d, want %d", s.index, want)
	}

	s, _, _, _ = s.Update(keyMsg("left"))
	if s.index != initial {
		t.Errorf("left after right: index = %d, want %d", s.index, initial)
	}

	// Wrap-around: from index 0, left lands on the last entry.
	s.index = 0
	s, _, _, _ = s.Update(keyMsg("left"))
	if want := len(s.zones) - 1; s.index != want {
		t.Errorf("left wrap: index = %d, want %d", s.index, want)
	}
}

func TestIdentityEveryZoneIsLoadable(t *testing.T) {
	// Every entry in the curated list (and the system default if
	// prepended) must round-trip through time.LoadLocation. A typo
	// here would break a user's persisted config silently.
	s := newIdentityStep("")
	for _, zone := range s.zones {
		if _, err := time.LoadLocation(zone); err != nil {
			t.Errorf("zone %q is not loadable: %v", zone, err)
		}
	}
}

func TestIdentityShiftTabRetreats(t *testing.T) {
	s := newIdentityStep("")
	_, _, _, retreat := s.Update(keyMsg("shift+tab"))
	if !retreat {
		t.Error("Shift+Tab did not signal retreat")
	}
}

func TestIdentityEnterAdvances(t *testing.T) {
	s := newIdentityStep("")
	_, _, advance, retreat := s.Update(keyMsg("enter"))
	if !advance {
		t.Error("Enter did not signal advance from picker")
	}
	if retreat {
		t.Error("Enter unexpectedly signalled retreat")
	}
}

func TestCommonTimezonesContainsBrussels(t *testing.T) {
	// Sanity check on the curated list — the dev's home zone must be
	// present so the picker doesn't have to prepend a duplicate.
	if !slices.Contains(commonTimezones, "Europe/Brussels") {
		t.Error("commonTimezones missing Europe/Brussels")
	}
}
