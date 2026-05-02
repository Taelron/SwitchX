package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

// Each migration integration test gets its own throwaway database via
// setupTestDB(t) so the developer's working `switchx` DB is never
// touched. The DB is dropped on cleanup; no per-table reset needed.

func TestMigrateIntegration_EmptyDir(t *testing.T) {
	cfg, user, pw := setupTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	src := os.DirFS("testdata/migrations-empty")
	if err := migrateFromFS(ctx, cfg, user, pw, src); err != nil {
		t.Fatalf("empty migrations: %v", err)
	}
}

func TestMigrateIntegration_SingleMigration(t *testing.T) {
	cfg, user, pw := setupTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	src := os.DirFS("testdata/migrations-single")
	if err := migrateFromFS(ctx, cfg, user, pw, src); err != nil {
		t.Fatalf("apply: %v", err)
	}

	pool, err := Connect(ctx, cfg, user, pw)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var note string
	row := pool.QueryRow(ctx, "SELECT note FROM switchx_test_marker WHERE id = 1")
	if err := row.Scan(&note); err != nil {
		t.Fatalf("expected migration to have created the table: %v", err)
	}
	if note != "tae-9 single migration" {
		t.Errorf("note = %q", note)
	}
}

func TestMigrateIntegration_RerunIsNoop(t *testing.T) {
	cfg, user, pw := setupTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	src := os.DirFS("testdata/migrations-single")
	if err := migrateFromFS(ctx, cfg, user, pw, src); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := migrateFromFS(ctx, cfg, user, pw, src); err != nil {
		t.Fatalf("re-apply (should be no-op): %v", err)
	}
}

func TestMigrateIntegration_MalformedFails(t *testing.T) {
	cfg, user, pw := setupTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	src := os.DirFS("testdata/migrations-malformed")
	err := migrateFromFS(ctx, cfg, user, pw, src)
	if !errors.Is(err, ErrMigrationFailed) {
		t.Fatalf("malformed → %v, want ErrMigrationFailed", err)
	}
}

// Two concurrent migrate calls must serialize on the advisory lock.
// The second waits, then proceeds without error (golang-migrate's
// default behavior, surfaced unchanged per @ADR-0002 / TAE-9 spec).
func TestMigrateIntegration_ConcurrentLockRace(t *testing.T) {
	cfg, user, pw := setupTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	src := os.DirFS("testdata/migrations-slow")

	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	for i := range errs {
		go func(i int) {
			defer wg.Done()
			errs[i] = migrateFromFS(ctx, cfg, user, pw, src)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("runner %d: %v", i, err)
		}
	}
}
