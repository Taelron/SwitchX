package azurekeyvault

import (
	"context"
	"os"
	"testing"

	"github.com/Taelron/SwitchX/internal/app/secrets"
)

// TestAzKVIntegration_RealVault hits a real Azure Key Vault. It is
// gated by the SWITCHX_AZURE_INTEGRATION env var so unit-test runs
// (CI, local `go test ./...`) skip it without needing `az` configured.
//
// To run:
//
//	SWITCHX_AZURE_INTEGRATION=1 \
//	SWITCHX_TEST_SUBSCRIPTION=<sub> \
//	SWITCHX_TEST_VAULT=<vault> \
//	SWITCHX_TEST_SECRET=<secret-name> \
//	go test -run TestAzKVIntegration -v ./internal/storage/secrets/azurekeyvault/
//
// The test does not log the fetched value, only its byte length.
func TestAzKVIntegration_RealVault(t *testing.T) {
	if os.Getenv("SWITCHX_AZURE_INTEGRATION") != "1" {
		t.Skip("set SWITCHX_AZURE_INTEGRATION=1 to run")
	}
	sub := os.Getenv("SWITCHX_TEST_SUBSCRIPTION")
	vault := os.Getenv("SWITCHX_TEST_VAULT")
	name := os.Getenv("SWITCHX_TEST_SECRET")
	if sub == "" || vault == "" || name == "" {
		t.Skip("missing SWITCHX_TEST_SUBSCRIPTION / VAULT / SECRET")
	}

	p := New()
	val, err := p.GetSecret(context.Background(), secrets.Ref{
		Subscription: sub, Vault: vault, Name: name,
	})
	if err != nil {
		t.Fatalf("integration GetSecret: %v", err)
	}
	defer zero(val)

	if len(val) == 0 {
		t.Fatal("integration: empty value")
	}
	t.Logf("integration: fetched %d bytes (value redacted)", len(val))
}
