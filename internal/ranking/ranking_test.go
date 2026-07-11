package ranking_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/ranking"
	"github.com/mparvin/repo-miner/internal/storage/sqlite"
)

func TestScoreRepository(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	ref := domain.RepositoryRef{Provider: "github", Owner: "test", Name: "proj", FullName: "test/proj"}
	repo := &domain.Repository{
		Ref: ref, Stars: 5000, UpdatedAt: time.Now(),
	}
	if err := store.SaveRepository(ctx, repo); err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Join(dir, "repos", "proj")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# test"), 0o644)
	_ = os.WriteFile(filepath.Join(repoDir, "AGENTS.md"), []byte("# agents"), 0o644)
	_ = os.MkdirAll(filepath.Join(repoDir, ".github", "workflows"), 0o755)
	_ = os.WriteFile(filepath.Join(repoDir, "main_test.go"), []byte("package main"), 0o644)

	cfg := &config.Config{
		Workspace: config.WorkspaceConfig{ReposDir: filepath.Join(dir, "repos")},
		Ranking:   config.RankingConfig{Weights: config.DefaultRankingWeights()},
	}
	engine := ranking.New(cfg, store)
	result, err := engine.Score(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalScore <= 0 {
		t.Errorf("expected positive score, got %d", result.TotalScore)
	}
	if result.AIScore <= 0 {
		t.Error("expected AI score from AGENTS.md")
	}
}
