package gitlab

import (
	"context"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/provider"
)

func init() {
	provider.Register("gitlab", New)
}

// Provider is a stub GitLab repository provider (Phase 1).
// Supports both GitLab Cloud and self-hosted instances via the url config.
type Provider struct {
	url   string
	token string
}

// New creates a new GitLab provider from config.
func New(cfg map[string]string) (provider.RepositoryProvider, error) {
	return &Provider{
		url:   cfg["url"],
		token: cfg["token"],
	}, nil
}

func (p *Provider) Name() string { return "gitlab" }

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
