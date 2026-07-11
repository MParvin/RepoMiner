package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/mparvin/repo-miner/internal/core/domain"
)

// ErrNotImplemented is returned by stub providers that have not yet been implemented.
var ErrNotImplemented = errors.New("not implemented")

// RepositoryProvider abstracts access to a source control platform.
type RepositoryProvider interface {
	Name() string
	ListRepositories(ctx context.Context, opts domain.ListOptions) ([]domain.Repository, error)
	GetRepository(ctx context.Context, ref domain.RepositoryRef) (*domain.Repository, error)
	GetCommits(ctx context.Context, ref domain.RepositoryRef, opts domain.CommitListOptions) ([]domain.Commit, error)
	GetIssues(ctx context.Context, ref domain.RepositoryRef, opts domain.ListOptions) ([]domain.Issue, error)
	GetPullRequests(ctx context.Context, ref domain.RepositoryRef, opts domain.ListOptions) ([]domain.PullRequest, error)
	CloneRepository(ctx context.Context, ref domain.RepositoryRef, opts domain.CloneOptions) (string, error)
}

// Factory creates a RepositoryProvider from configuration.
type Factory func(cfg map[string]string) (RepositoryProvider, error)

var (
	mu        sync.RWMutex
	registry  = make(map[string]Factory)
)

// Register adds a provider factory to the global registry.
func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("provider %q already registered", name))
	}
	registry[name] = factory
}

// Get returns a provider instance by name.
func Get(name string, cfg map[string]string) (RepositoryProvider, error) {
	mu.RLock()
	factory, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}
	return factory(cfg)
}

// List returns all registered provider names.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
