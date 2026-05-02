package azurekeyvault

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/Taelron/SwitchX/internal/app/secrets"
)

// fakeRunner builds a Runner that returns a fixed (stdout, stderr, err)
// triple and records the args it was called with.
type fakeRunner struct {
	stdout []byte
	stderr []byte
	err    error
	gotCmd string
	gotArg []string
	calls  int
}

func (f *fakeRunner) run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	f.calls++
	f.gotCmd = name
	f.gotArg = args
	return f.stdout, f.stderr, f.err
}

func validRef() secrets.Ref {
	return secrets.Ref{
		Subscription: "sub-test",
		Vault:        "vault-test",
		Name:         "secret-test",
	}
}

func TestGetSecret_Success(t *testing.T) {
	f := &fakeRunner{stdout: []byte("the-value\n")}
	p := NewWithRunner(f.run)
	val, err := p.GetSecret(context.Background(), validRef())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if string(val) != "the-value" {
		t.Errorf("val = %q, want %q", val, "the-value")
	}
}

func TestGetSecret_PassesSubscriptionFlag(t *testing.T) {
	f := &fakeRunner{stdout: []byte("v\n")}
	p := NewWithRunner(f.run)
	if _, err := p.GetSecret(context.Background(), validRef()); err != nil {
		t.Fatalf("err = %v", err)
	}
	if f.gotCmd != "az" {
		t.Errorf("cmd = %q, want %q", f.gotCmd, "az")
	}
	args := strings.Join(f.gotArg, " ")
	for _, want := range []string{
		"keyvault secret show",
		"--subscription sub-test",
		"--vault-name vault-test",
		"--name secret-test",
		"--query value",
		"-o tsv",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q; got: %s", want, args)
		}
	}
}

func TestGetSecret_TwoIndependentCalls(t *testing.T) {
	f := &fakeRunner{stdout: []byte("v\n")}
	p := NewWithRunner(f.run)
	r1 := secrets.Ref{Subscription: "s", Vault: "v", Name: "user"}
	r2 := secrets.Ref{Subscription: "s", Vault: "v", Name: "password"}
	if _, err := p.GetSecret(context.Background(), r1); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetSecret(context.Background(), r2); err != nil {
		t.Fatal(err)
	}
	if f.calls != 2 {
		t.Errorf("runner calls = %d, want 2 (no caching)", f.calls)
	}
	if !strings.Contains(strings.Join(f.gotArg, " "), "--name password") {
		t.Errorf("second call did not pass --name password")
	}
}

func TestGetSecret_InvalidRef(t *testing.T) {
	cases := []secrets.Ref{
		{Subscription: "", Vault: "v", Name: "n"},
		{Subscription: "s", Vault: "", Name: "n"},
		{Subscription: "s", Vault: "v", Name: ""},
	}
	for _, ref := range cases {
		t.Run(ref.Subscription+"|"+ref.Vault+"|"+ref.Name, func(t *testing.T) {
			f := &fakeRunner{}
			p := NewWithRunner(f.run)
			_, err := p.GetSecret(context.Background(), ref)
			if !errors.Is(err, secrets.ErrInvalidRef) {
				t.Errorf("want ErrInvalidRef, got %v", err)
			}
			if f.calls != 0 {
				t.Errorf("runner should not be called for invalid ref")
			}
		})
	}
}

func TestGetSecret_AzNotOnPATH(t *testing.T) {
	f := &fakeRunner{err: &exec.Error{Name: "az", Err: exec.ErrNotFound}}
	p := NewWithRunner(f.run)
	_, err := p.GetSecret(context.Background(), validRef())
	if !errors.Is(err, secrets.ErrCLINotFound) {
		t.Fatalf("want ErrCLINotFound, got %v", err)
	}
}

func TestGetSecret_StderrClassification(t *testing.T) {
	cases := []struct {
		name   string
		stderr string
		want   error
	}{
		{"not_logged_in", "Please run 'az login' to setup account.", secrets.ErrNotLoggedIn},
		{"subscription_not_found", "The subscription 'foo' could not be found.", secrets.ErrSubscriptionAccess},
		{"subscription_no_access", "The user does not have access to subscription 'bar'.", secrets.ErrSubscriptionAccess},
		{"vault_not_found", "Vault 'kv-x' not found.", secrets.ErrVaultNotFound},
		{"secret_not_found", "Secret 'switchx-db-x-user' not found in vault.", secrets.ErrSecretNotFound},
		{"network_dns", "Could not resolve host: management.azure.com", secrets.ErrNetwork},
		{"network_refused", "connection refused", secrets.ErrNetwork},
		{"network_timeout", "request timeout", secrets.ErrNetwork},
		{"unexpected", "some other failure mode", secrets.ErrUnexpected},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeRunner{
				stderr: []byte(tc.stderr),
				err:    &exec.ExitError{},
			}
			p := NewWithRunner(f.run)
			_, err := p.GetSecret(context.Background(), validRef())
			if !errors.Is(err, tc.want) {
				t.Errorf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestGetSecret_EmptyOutput(t *testing.T) {
	f := &fakeRunner{stdout: []byte("\n")}
	p := NewWithRunner(f.run)
	_, err := p.GetSecret(context.Background(), validRef())
	if !errors.Is(err, secrets.ErrEmptyValue) {
		t.Fatalf("want ErrEmptyValue, got %v", err)
	}
}

// Critical security guarantee: a future `az` regression that echoes
// secret values into stderr must not cause those values to end up in
// the returned error message. The adapter classifies stderr by pattern
// but never embeds the raw text.
func TestGetSecret_NoSecretLeakage(t *testing.T) {
	const leak = "SuPeR-SeCrEt-PASSWORD-DO-NOT-LOG"
	f := &fakeRunner{
		stderr: []byte("Vault not found. (debug: value=" + leak + ")"),
		err:    &exec.ExitError{},
	}
	p := NewWithRunner(f.run)
	_, err := p.GetSecret(context.Background(), validRef())
	if err == nil {
		t.Fatal("want error")
	}
	if strings.Contains(err.Error(), leak) {
		t.Errorf("error message leaked secret material: %q", err.Error())
	}
}
