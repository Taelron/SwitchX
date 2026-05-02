// Package azurekeyvault implements secrets.Provider by shelling out to
// the Azure `az` CLI. Per @ADR-0005, every invocation passes
// --subscription explicitly to defeat the multi-subscription "wrong
// active subscription" failure mode.
//
// The adapter never embeds raw stderr text into returned errors. It
// pattern-matches stderr to classify failures into sentinel errors and
// reports only structured context (subscription, vault, secret name).
// This is a defense-in-depth measure: today `az` does not echo secret
// values in stderr, but a future change could, and the @Security &
// Secret Handling baseline forbids any path that could leak secret
// material to logs or user-facing errors.
package azurekeyvault

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Taelron/SwitchX/internal/app/secrets"
)

// Runner runs an external command and returns its stdout, stderr, and
// any error. Production uses runExec (exec.CommandContext); tests
// inject a fake to avoid needing a real `az`.
type Runner func(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)

// Provider implements secrets.Provider via the `az` CLI.
type Provider struct {
	runner Runner
}

// New constructs a Provider that uses the real `az` binary on PATH.
func New() *Provider {
	return &Provider{runner: runExec}
}

// NewWithRunner constructs a Provider with a caller-supplied Runner.
// Tests use this to substitute a fake `az`.
func NewWithRunner(r Runner) *Provider {
	return &Provider{runner: r}
}

// GetSecret fetches one secret from Azure Key Vault. The exact command
// invoked is:
//
//	az keyvault secret show --subscription <sub> --vault-name <vault>
//	                        --name <name> --query value -o tsv
//
// On success the secret value is returned as []byte; the caller is
// responsible for zeroing it after use.
func (p *Provider) GetSecret(ctx context.Context, ref secrets.Ref) ([]byte, error) {
	if ref.Subscription == "" || ref.Vault == "" || ref.Name == "" {
		return nil, fmt.Errorf("%w: subscription, vault, and name are all required", secrets.ErrInvalidRef)
	}

	stdout, stderr, err := p.runner(ctx, "az",
		"keyvault", "secret", "show",
		"--subscription", ref.Subscription,
		"--vault-name", ref.Vault,
		"--name", ref.Name,
		"--query", "value",
		"-o", "tsv",
	)
	// Scrub the runner's stdout buffer before this function returns, no
	// matter the path taken. The returned []byte below is a fresh copy
	// the caller owns; this defer zeros the original so two slices into
	// the secret never coexist longer than necessary.
	defer zero(stdout)

	if err != nil {
		return nil, classify(err, stderr, ref)
	}

	// `-o tsv` appends a trailing newline. Trim, then copy into an
	// exact-size slice so the caller's defer-zero covers every byte
	// (the buffer's backing array may extend past len).
	trimmed := bytes.TrimRight(stdout, "\n")
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%w: secret %q in vault %q",
			secrets.ErrEmptyValue, ref.Name, ref.Vault)
	}
	out := make([]byte, len(trimmed))
	copy(out, trimmed)
	return out, nil
}

// zero overwrites a byte slice with zeros.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// runExec is the production Runner. The command name is hardcoded to
// "az" (the package's only legitimate target) so that gosec G204 is not
// triggered and so that no caller can redirect the production path to
// an arbitrary binary; tests still get full control via NewWithRunner.
// The name parameter is accepted for Runner symmetry and ignored here.
func runExec(ctx context.Context, _ string, args ...string) ([]byte, []byte, error) {
	// #nosec G204 -- command is the literal "az"; args are constructed in
	// GetSecret from the user's mode-0600 config (subscription, vault,
	// secret name) — never from untrusted input.
	cmd := exec.CommandContext(ctx, "az", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// classify maps an exec error and stderr text to a sentinel-wrapped
// error. The stderr is consulted only for pattern matching; its content
// is never embedded in the returned message.
func classify(execErr error, stderr []byte, ref secrets.Ref) error {
	var execErrPtr *exec.Error
	if errors.As(execErr, &execErrPtr) {
		return fmt.Errorf("%w: install Azure CLI and ensure 'az' is on PATH",
			secrets.ErrCLINotFound)
	}

	msg := strings.ToLower(string(stderr))
	switch {
	case strings.Contains(msg, "az login"),
		strings.Contains(msg, "please run 'az login'"),
		strings.Contains(msg, "not logged in"):
		return fmt.Errorf("%w: run 'az login'", secrets.ErrNotLoggedIn)

	case strings.Contains(msg, "subscription") &&
		(strings.Contains(msg, "not found") ||
			strings.Contains(msg, "could not find") ||
			strings.Contains(msg, "could not be found") ||
			strings.Contains(msg, "does not have access")):
		return fmt.Errorf("%w: subscription %q (run 'az account list' to confirm)",
			secrets.ErrSubscriptionAccess, ref.Subscription)

	// Ordering: secret-not-found MUST precede vault-not-found because
	// az's "secret X not found in vault Y" stderr satisfies both
	// branches. The lead noun is what actually failed.
	case strings.Contains(msg, "secret") && strings.Contains(msg, "not found"):
		return fmt.Errorf("%w: secret %q in vault %q",
			secrets.ErrSecretNotFound, ref.Name, ref.Vault)

	// Reached only when stderr names the vault but not "secret" — i.e.
	// the vault itself is missing. Do not reorder above the secret case.
	case strings.Contains(msg, "vault") && strings.Contains(msg, "not found"):
		return fmt.Errorf("%w: vault %q in subscription %q",
			secrets.ErrVaultNotFound, ref.Vault, ref.Subscription)

	case strings.Contains(msg, "could not resolve"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "network is unreachable"),
		strings.Contains(msg, "timeout"):
		return fmt.Errorf("%w: while fetching secret %q from vault %q",
			secrets.ErrNetwork, ref.Name, ref.Vault)

	default:
		return fmt.Errorf("%w: az exited unexpectedly while fetching secret %q from vault %q",
			secrets.ErrUnexpected, ref.Name, ref.Vault)
	}
}
