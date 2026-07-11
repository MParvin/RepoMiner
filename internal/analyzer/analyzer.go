package analyzer

import (
	"context"
	"fmt"
	"sync"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/provider"
)

// LanguageAnalyzer abstracts source code analysis for a programming language.
type LanguageAnalyzer interface {
	Name() string
	Analyze(ctx context.Context, repoPath string) (*domain.AnalysisResult, error)
}

// Factory creates a LanguageAnalyzer from configuration.
type Factory func(cfg map[string]string) (LanguageAnalyzer, error)

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

// Register adds an analyzer factory to the global registry.
func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("analyzer %q already registered", name))
	}
	registry[name] = factory
}

// Get returns an analyzer instance by name.
func Get(name string, cfg map[string]string) (LanguageAnalyzer, error) {
	mu.RLock()
	factory, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown analyzer %q", name)
	}
	return factory(cfg)
}

// List returns all registered analyzer names.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// ErrNotImplemented re-exports the provider error for analyzer stubs.
var ErrNotImplemented = provider.ErrNotImplemented
