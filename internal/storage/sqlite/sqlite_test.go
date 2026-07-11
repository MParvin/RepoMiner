package sqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/storage/sqlite"
)

func TestSQLiteMigrateAndPing(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Verify DB file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}
}

func TestSQLiteSaveAndGetRepository(t *testing.T) {
	dir := t.TempDir()
	store, err := sqlite.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ref := domain.RepositoryRef{
		Provider: "github",
		Owner:    "gin-gonic",
		Name:     "gin",
		FullName: "gin-gonic/gin",
	}
	repo := &domain.Repository{
		Ref:       ref,
		Language:  "Go",
		Stars:     75000,
		CreatedAt: time.Date(2014, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := store.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.GetRepository(ctx, ref)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Language != "Go" {
		t.Errorf("expected language Go, got %s", got.Language)
	}
	if got.Stars != 75000 {
		t.Errorf("expected 75000 stars, got %d", got.Stars)
	}
}
