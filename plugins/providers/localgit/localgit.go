package localgit

import (
	"context"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/provider"
)

func init() {
	provider.Register("localgit", New)
}

// Provider is a stub local Git repository provider (Phase 1).
type Provider struct {
	path string
}

// New creates a new local Git provider from config.
func New(cfg map[string]string) (provider.RepositoryProvider, error) {
	return &Provider{path: cfg["path"]}, nil
}

func (p *Provider) Name() string { return "localgit" }

func (p *Provider) ListRepositories(_ context.Context, _ domain.ListOptions) ([]domain.Repository, error) {
	return nil, provider.ErrNotImplemented
}

func (p *Provider) GetRepository(_ context.Context, _ domain.RepositoryRef) (*domain.Repository, error) {
	return nil, provider.ErrNotImplemented
}

func (p *Provider) GetCommits(_ context.Context, _ domain.RepositoryRef, _ domain.CommitListOptions) ([]domain.Commit, error) {
	return nil, provider.ErrNotImplemented
}

func (p *Provider) GetIssues(_ context.Context, _ domain.RepositoryRef, _ domain.ListOptions) ([]domain.Issue, error) {
	return nil, provider.ErrNotImplemented
}

func (p *Provider) GetPullRequests(_ context.Context, _ domain.RepositoryRef, _ domain.ListOptions) ([]domain.PullRequest, error) {
	return nil, provider.ErrNotImplemented
}

func (p *Provider) CloneRepository(_ context.Context, _ domain.RepositoryRef, _ domain.CloneOptions) (string, error) {
	return "", provider.ErrNotImplemented
}
