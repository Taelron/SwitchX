package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taelron/SwitchX/internal/config"
)

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

// modelWith builds a model with no-op validator/save by default.
// Individual tests override either field via the returned model.
func modelWith(t *testing.T) model {
	t.Helper()
	return newModel(
		context.Background(),
		func(context.Context, *config.Config) error { return nil },
		func(*config.Config) error { return nil }, nil,
	)
}

// fillStep1 puts step 1 into a state where validate passes and the
// wizard advances to step 2.
func fillStep1(m model) model {
	m.database.inputs[fieldDBHost].SetValue("db.example.internal")
	m.database.inputs[fieldDBName].SetValue("switchx")
	m.database.focus = fieldDBSSLMode
	next, _ := m.Update(keyMsg("enter"))
	return next.(model)
}

// fillStep2 fills step 2's four textinputs and Enters on the last
// field. The test must already be on stepSecret.
func fillStep2(m model) model {
	m.secret.inputs[fieldSecretSubscription].SetValue("00000000-0000-0000-0000-000000000000")
	m.secret.inputs[fieldSecretVault].SetValue("vault")
	m.secret.inputs[fieldSecretUserRef].SetValue("user-ref")
	m.secret.inputs[fieldSecretPasswordRef].SetValue("password-ref")
	m.secret.focus = fieldSecretPasswordRef
	next, _ := m.Update(keyMsg("enter"))
	return next.(model)
}

// pickEditor sets the step-4 picker to the given editor name. The
// curated list always contains "vi"; if the helper is asked for an
// editor that is not in the list, the helper just keeps the default.
func pickEditor(m model, name string) model {
	for i, e := range m.preferences.editors {
		if e == name {
			m.preferences.index = i
			return m
		}
	}
	return m
}

// fillStep3 cycles step 3's picker to Europe/Brussels and Enters. The
// curated list always contains Brussels; cycling forward until found
// keeps the helper resilient to whether the system TZ was prepended.
func fillStep3(m model) model {
	for range len(m.identity.zones) {
		if m.identity.zones[m.identity.index] == "Europe/Brussels" {
			break
		}
		s, _, _, _ := m.identity.Update(keyMsg("right"))
		m.identity = s
	}
	next, _ := m.Update(keyMsg("enter"))
	return next.(model)
}

func TestEscQuitsFromAnyStep(t *testing.T) {
	for _, s := range []step{stepDatabase, stepSecret, stepIdentity, stepPreferences, stepValidating} {
		m := modelWith(t)
		m.current = s
		_, cmd := m.Update(keyMsg("esc"))
		if cmd == nil {
			t.Errorf("step %d: Esc returned nil cmd", s)
			continue
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("step %d: Esc cmd produced %T, want tea.QuitMsg", s, cmd())
		}
	}
}

func TestStep1AdvanceWritesDatabaseFields(t *testing.T) {
	m := modelWith(t)
	m = fillStep1(m)
	if m.current != stepSecret {
		t.Fatalf("after step-1 enter: current = %d, want %d", m.current, stepSecret)
	}
	if m.cfg.Database.Host != "db.example.internal" {
		t.Errorf("cfg.Database.Host = %q", m.cfg.Database.Host)
	}
	if m.cfg.Database.Port != 5432 {
		t.Errorf("cfg.Database.Port = %d", m.cfg.Database.Port)
	}
	if m.cfg.Database.Name != "switchx" {
		t.Errorf("cfg.Database.Name = %q", m.cfg.Database.Name)
	}
	if m.cfg.Database.SSLMode != "require" {
		t.Errorf("cfg.Database.SSLMode = %q", m.cfg.Database.SSLMode)
	}
}

func TestStep2BackNavReturnsToStep1LastField(t *testing.T) {
	m := modelWith(t)
	m = fillStep1(m)
	if m.current != stepSecret {
		t.Fatalf("precondition: not on step 2 (current=%d)", m.current)
	}
	if m.secret.focus != 0 {
		t.Fatalf("precondition: step 2 focus=%d, want 0", m.secret.focus)
	}
	next, _ := m.Update(keyMsg("shift+tab"))
	m = next.(model)
	if m.current != stepDatabase {
		t.Fatalf("after shift+tab on step-2 field 0: current = %d, want %d", m.current, stepDatabase)
	}
	if m.database.focus != dbFieldCount-1 {
		t.Errorf("step 1 focus on return = %d, want %d (last field)", m.database.focus, dbFieldCount-1)
	}
}

func TestStep3BackNavReturnsToStep2LastField(t *testing.T) {
	m := modelWith(t)
	m = fillStep1(m)
	m = fillStep2(m)
	if m.current != stepIdentity {
		t.Fatalf("precondition: not on step 3 (current=%d)", m.current)
	}
	next, _ := m.Update(keyMsg("shift+tab"))
	m = next.(model)
	if m.current != stepSecret {
		t.Fatalf("after shift+tab on step-3 field 0: current = %d, want %d", m.current, stepSecret)
	}
	if m.secret.focus != secretFieldCount-1 {
		t.Errorf("step 2 focus on return = %d, want %d", m.secret.focus, secretFieldCount-1)
	}
}

func TestStep4BackNavReturnsToStep3(t *testing.T) {
	m := modelWith(t)
	m = fillStep1(m)
	m = fillStep2(m)
	m = fillStep3(m)
	if m.current != stepPreferences {
		t.Fatalf("precondition: not on step 4 (current=%d)", m.current)
	}
	next, _ := m.Update(keyMsg("shift+tab"))
	m = next.(model)
	if m.current != stepIdentity {
		t.Fatalf("after shift+tab on step-4 field 0: current = %d, want %d", m.current, stepIdentity)
	}
}

func TestStep1ShiftTabWrapsNoRetreat(t *testing.T) {
	// Step 1 has no previous step; Shift+Tab on field 0 wraps to the
	// last field, not retreats.
	m := modelWith(t)
	if m.database.focus != fieldDBHost {
		t.Fatalf("precondition: focus=%d, want %d", m.database.focus, fieldDBHost)
	}
	next, _ := m.Update(keyMsg("shift+tab"))
	m = next.(model)
	if m.current != stepDatabase {
		t.Fatalf("step 1 retreated unexpectedly: current=%d", m.current)
	}
	if m.database.focus != dbFieldCount-1 {
		t.Errorf("focus after wrap = %d, want %d", m.database.focus, dbFieldCount-1)
	}
}

func TestStep4AdvanceTriggersValidate(t *testing.T) {
	m := modelWith(t)
	m = fillStep1(m)
	m = fillStep2(m)
	m = fillStep3(m)
	// Editor default may already be set; ensure non-empty.
	m = pickEditor(m, "vi")
	next, _ := m.Update(keyMsg("enter"))
	m = next.(model)
	if m.current != stepValidating {
		t.Fatalf("after step-4 enter: current = %d, want %d", m.current, stepValidating)
	}
	if m.cfg.UI.Editor != "vi" {
		t.Errorf("cfg.UI.Editor = %q, want vi", m.cfg.UI.Editor)
	}
}

func TestValidateSuccessSavesAndQuits(t *testing.T) {
	saved := false
	var savedCfg *config.Config
	m := newModel(
		context.Background(),
		func(context.Context, *config.Config) error { return nil },
		func(c *config.Config) error { saved = true; savedCfg = c; return nil }, nil,
	)
	m = fillStep1(m)
	m = fillStep2(m)
	m = fillStep3(m)
	m = pickEditor(m, "vi")
	next, _ := m.Update(keyMsg("enter"))
	m = next.(model)
	// Now post the success result directly; in production this would
	// be produced by the Validator goroutine.
	out, cmd := m.Update(validateResultMsg{generation: 1, err: nil})
	m = out.(model)

	if !saved {
		t.Fatal("save function was not called on validate success")
	}
	if savedCfg == nil || savedCfg.Database.Host != "db.example.internal" {
		t.Errorf("save received wrong config: %+v", savedCfg)
	}
	if m.finalCfg == nil {
		t.Error("model.finalCfg is nil after success; bootstrap.Run will return nil")
	}
	if cmd == nil {
		t.Fatal("success path returned nil cmd; expected tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("success cmd produced %T, want tea.QuitMsg", cmd())
	}
}

func TestValidateFailureShowsBannerAndStaysOnStep4(t *testing.T) {
	m := modelWith(t)
	m = fillStep1(m)
	m = fillStep2(m)
	m = fillStep3(m)
	m = pickEditor(m, "vi")
	next, _ := m.Update(keyMsg("enter"))
	m = next.(model)

	failure := errors.New("user_ref: secrets: secret not found")
	out, _ := m.Update(validateResultMsg{generation: 1, err: failure})
	m = out.(model)

	if m.current != stepPreferences {
		t.Fatalf("after validate failure: current = %d, want %d", m.current, stepPreferences)
	}
	if !strings.Contains(m.bannerError, "user_ref") {
		t.Errorf("bannerError = %q, expected to contain 'user_ref'", m.bannerError)
	}
	if m.finalCfg != nil {
		t.Error("finalCfg should be nil after a failed validate")
	}
}

func TestSaveFailureSurfacesBannerNotProcessExit(t *testing.T) {
	m := newModel(
		context.Background(),
		func(context.Context, *config.Config) error { return nil },
		func(*config.Config) error { return errors.New("permission denied") }, nil,
	)
	m = fillStep1(m)
	m = fillStep2(m)
	m = fillStep3(m)
	m = pickEditor(m, "vi")
	next, _ := m.Update(keyMsg("enter"))
	m = next.(model)

	out, cmd := m.Update(validateResultMsg{generation: 1, err: nil})
	m = out.(model)
	if m.current != stepPreferences {
		t.Fatalf("save failure should return user to step 4, got step %d", m.current)
	}
	if !strings.Contains(m.bannerError, "save:") {
		t.Errorf("bannerError = %q, expected save error prefix", m.bannerError)
	}
	if cmd != nil {
		// Save failure must not terminate the program — user can retry.
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("save failure unexpectedly produced tea.Quit")
		}
	}
}

func TestStaleValidateResultIsDropped(t *testing.T) {
	saved := false
	m := newModel(
		context.Background(),
		func(context.Context, *config.Config) error { return nil },
		func(*config.Config) error { saved = true; return nil }, nil,
	)
	m = fillStep1(m)
	m = fillStep2(m)
	m = fillStep3(m)
	m = pickEditor(m, "vi")
	// First validate run: bumps gen to 1.
	next, _ := m.Update(keyMsg("enter"))
	m = next.(model)
	// First run fails — back to step 4.
	out, _ := m.Update(validateResultMsg{generation: 1, err: errors.New("kv down")})
	m = out.(model)
	// User retries — bumps gen to 2.
	next2, _ := m.Update(keyMsg("enter"))
	m = next2.(model)
	// A late result from the first (gen=1) run arrives. It must be
	// silently dropped: no save, no transition, banner unchanged.
	prevBanner := m.bannerError
	out2, cmd := m.Update(validateResultMsg{generation: 1, err: nil})
	m = out2.(model)

	if saved {
		t.Error("stale validateResultMsg triggered Save")
	}
	if m.current != stepValidating {
		t.Errorf("current = %d, want stepValidating (%d) — stale result moved the state machine", m.current, stepValidating)
	}
	if m.bannerError != prevBanner {
		t.Errorf("stale result changed bannerError from %q to %q", prevBanner, m.bannerError)
	}
	if cmd != nil {
		// A stale result must not produce tea.Quit.
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("stale result produced tea.Quit")
		}
	}
}

func TestProviderRowNotInFocusList(t *testing.T) {
	m := modelWith(t)
	m = fillStep1(m)
	if m.current != stepSecret {
		t.Fatalf("not on step 2: %d", m.current)
	}
	// Tab three times: subscription → vault → user_ref → password_ref.
	// The provider row must never receive focus.
	for i := 1; i <= secretFieldCount-1; i++ {
		next, _ := m.Update(keyMsg("tab"))
		m = next.(model)
		if m.secret.focus != i {
			t.Fatalf("after %d tabs: focus = %d, want %d", i, m.secret.focus, i)
		}
	}
	// One more tab wraps to subscription — not to a "provider" index.
	next, _ := m.Update(keyMsg("tab"))
	m = next.(model)
	if m.secret.focus != fieldSecretSubscription {
		t.Errorf("tab wrap: focus = %d, want %d", m.secret.focus, fieldSecretSubscription)
	}
}
