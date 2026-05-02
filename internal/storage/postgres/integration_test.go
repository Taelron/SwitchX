package postgres

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Taelron/SwitchX/internal/config"
)

// runIntegration gates Postgres-touching tests behind a single env
// var so `go test ./...` skips them in CI / cold-clone environments.
//
// Set up the local DEV harness first (`make dev-up`), then export:
//
//	SWITCHX_PG_INTEGRATION=1
//	SWITCHX_PG_HOST       (default: localhost)
//	SWITCHX_PG_PORT       (default: 5432)
//	SWITCHX_PG_DB         (default: switchx)
//	SWITCHX_PG_USER       (default: switchx_user)
//	SWITCHX_PG_PASSWORD   (no default — required when gated on)
//	SWITCHX_PG_SSLMODE    (default: disable for local)
func runIntegration(t *testing.T) (config.Database, []byte, []byte) {
	t.Helper()
	if os.Getenv("SWITCHX_PG_INTEGRATION") != "1" {
		t.Skip("set SWITCHX_PG_INTEGRATION=1 to run")
	}
	pw := os.Getenv("SWITCHX_PG_PASSWORD")
	if pw == "" {
		t.Skip("missing SWITCHX_PG_PASSWORD")
	}
	port := 5432
	if v := os.Getenv("SWITCHX_PG_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("bad SWITCHX_PG_PORT=%q: %v", v, err)
		}
		port = p
	}
	cfg := config.Database{
		Host:    envOr("SWITCHX_PG_HOST", "localhost"),
		Port:    port,
		Name:    envOr("SWITCHX_PG_DB", "switchx"),
		SSLMode: envOr("SWITCHX_PG_SSLMODE", "disable"),
	}
	user := []byte(envOr("SWITCHX_PG_USER", "switchx_user"))
	return cfg, user, []byte(pw)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestPGIntegration_Connect_Healthy(t *testing.T) {
	cfg, user, pw := runIntegration(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := Connect(ctx, cfg, user, pw)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pool.Close()

	if err := HealthCheck(ctx, pool); err != nil {
		t.Errorf("HealthCheck: %v", err)
	}
	if got := pool.Config().MaxConns; got != MaxConns {
		t.Errorf("MaxConns = %d, want %d", got, MaxConns)
	}
}

func TestPGIntegration_WrongPassword(t *testing.T) {
	cfg, user, _ := runIntegration(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Connect(ctx, cfg, user, []byte("definitely-wrong"))
	if !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("wrong password → %v, want ErrAuthFailed", err)
	}
}

func TestPGIntegration_WrongDB(t *testing.T) {
	cfg, user, pw := runIntegration(t)
	cfg.Name = "no-such-db-12345"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Connect(ctx, cfg, user, pw)
	if !errors.Is(err, ErrDBNotFound) {
		t.Fatalf("wrong db → %v, want ErrDBNotFound", err)
	}
}

func TestPGIntegration_ClosedPoolRejects(t *testing.T) {
	cfg, user, pw := runIntegration(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := Connect(ctx, cfg, user, pw)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	pool.Close()

	if err := HealthCheck(ctx, pool); err == nil {
		t.Error("HealthCheck on closed pool returned nil; want error")
	}
}
