package golang

import (
	"context"

	"github.com/mparvin/repo-miner/internal/analyzer"
	"github.com/mparvin/repo-miner/internal/core/domain"
)

func init() {
	analyzer.Register("golang", New)
}

// Analyzer is a stub Go language analyzer (Phase 2).
type Analyzer struct{}

// New creates a new Go analyzer from config.
func New(_ map[string]string) (analyzer.LanguageAnalyzer, error) {
	return &Analyzer{}, nil
}

func (a *Analyzer) Name() string { return "golang" }

func (a *Analyzer) Analyze(_ context.Context, _ string) (*domain.AnalysisResult, error) {
	return nil, analyzer.ErrNotImplemented
}
