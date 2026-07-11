package generator

import (
	"context"
	"fmt"

	"github.com/mparvin/repo-miner/internal/analyzer"
	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/storage"
)

// Generator orchestrates dataset sample generation from collected data.
type Generator struct {
	Analyzer analyzer.LanguageAnalyzer
	Storage  storage.Storage
}

// New creates a Generator from application config and storage.
func New(cfg *config.Config, store storage.Storage) (*Generator, error) {
	ana, err := analyzer.Get(cfg.Analyzer.Language, cfg.AnalyzerConfigMap())
	if err != nil {
		return nil, fmt.Errorf("resolve analyzer: %w", err)
	}
	return &Generator{Analyzer: ana, Storage: store}, nil
}

// Generate produces dataset samples for a repository.
func (g *Generator) Generate(ctx context.Context, ref domain.RepositoryRef) ([]domain.DatasetSample, error) {
	_, err := g.Storage.GetRepository(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}

	// Phase 3 will implement actual sample generation from issues, PRs, and commits.
	return nil, fmt.Errorf("dataset generation not implemented (Phase 3)")
}
