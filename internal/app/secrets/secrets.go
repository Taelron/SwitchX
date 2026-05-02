// Package secrets defines the port for fetching secret material from
// an external store. Adapters live in internal/storage/secrets/.
//
// Per @ADR-0005, switchx fetches two secrets at startup (the Postgres
// role name and its password). Both flow through this port. The port is
// provider-agnostic; the Azure Key Vault adapter is the only concrete
// implementation in M1.
//
// The returned []byte is the caller's responsibility to zero once the
// secret has been consumed (typically via defer) per the @Security &
// Secret Handling baseline.
package secrets

import (
	"context"
	"errors"
)

// Provider resolves a single secret value by reference.
type Provider interface {
	GetSecret(ctx context.Context, ref Ref) ([]byte, error)
}

// Ref names a single secret. Subscription is required by the Azure Key
// Vault adapter and may be ignored by other providers.
type Ref struct {
	Subscription string
	Vault        string
	Name         string
}

// Sentinel errors returned by Provider implementations. Callers branch
// on these with errors.Is.
var (
	ErrInvalidRef         = errors.New("secrets: invalid reference")
	ErrCLINotFound        = errors.New("secrets: az CLI not on PATH")
	ErrNotLoggedIn        = errors.New("secrets: az not logged in")
	ErrSubscriptionAccess = errors.New("secrets: subscription not accessible")
	ErrVaultNotFound      = errors.New("secrets: vault not found")
	ErrSecretNotFound     = errors.New("secrets: secret not found")
	ErrNetwork            = errors.New("secrets: network failure")
	ErrEmptyValue         = errors.New("secrets: empty secret value")
	ErrUnexpected         = errors.New("secrets: unexpected error")
)
