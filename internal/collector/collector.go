package collector

import (
	"context"
	"fmt"

	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/provider"
	"github.com/mparvin/repo-miner/internal/storage"
)

// Collector orchestrates repository data collection via a provider plugin.
type Collector struct {
	Provider provider.RepositoryProvider
	Storage  storage.Storage
}

// New creates a Collector from application config and storage.
func New(cfg *config.Config, store storage.Storage) (*Collector, error) {
	prov, err := provider.Get(cfg.Source.Type, cfg.ProviderConfig())
	if err != nil {
		return nil, fmt.Errorf("resolve provider: %w", err)
	}
	return &Collector{Provider: prov, Storage: store}, nil
}

// Collect fetches and stores all data for a repository.
func (c *Collector) Collect(ctx context.Context, ref domain.RepositoryRef) error {
	repo, err := c.Provider.GetRepository(ctx, ref)
	if err != nil {
		return fmt.Errorf("get repository: %w", err)
	}
	if err := c.Storage.SaveRepository(ctx, repo); err != nil {
		return fmt.Errorf("save repository: %w", err)
	}

	commits, err := c.Provider.GetCommits(ctx, ref, domain.CommitListOptions{})
	if err != nil {
		return fmt.Errorf("get commits: %w", err)
	}
	if err := c.Storage.SaveCommits(ctx, ref, commits); err != nil {
		return fmt.Errorf("save commits: %w", err)
	}

	prs, err := c.Provider.GetPullRequests(ctx, ref, domain.ListOptions{})
	if err != nil {
		return fmt.Errorf("get pull requests: %w", err)
	}
	if err := c.Storage.SavePullRequests(ctx, ref, prs); err != nil {
		return fmt.Errorf("save pull requests: %w", err)
	}

	issues, err := c.Provider.GetIssues(ctx, ref, domain.ListOptions{})
	if err != nil {
		return fmt.Errorf("get issues: %w", err)
	}
	if err := c.Storage.SaveIssues(ctx, ref, issues); err != nil {
		return fmt.Errorf("save issues: %w", err)
	}

	return nil
}
