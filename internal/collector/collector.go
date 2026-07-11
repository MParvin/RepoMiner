package collector

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/httpclient"
	"github.com/mparvin/repo-miner/internal/core/provider"
	"github.com/mparvin/repo-miner/internal/storage"
)

// RawSaverSetter allows providers to receive a raw response saver.
type RawSaverSetter interface {
	SetRawSaver(saver httpclient.RawSaver)
}

// Collector orchestrates repository data collection via a provider plugin.
type Collector struct {
	Provider provider.RepositoryProvider
	Storage  storage.Storage
}

// New creates a Collector from application config and storage.
func New(cfg *config.Config, store storage.Storage, ref domain.RepositoryRef) (*Collector, error) {
	provCfg := cfg.ProviderConfig()
	if cfg.Source.Type == "localgit" {
		provCfg["path"] = filepath.Join(cfg.Workspace.ReposDir, ref.Name)
	}
	prov, err := provider.Get(cfg.Source.Type, provCfg)
	if err != nil {
		return nil, fmt.Errorf("resolve provider: %w", err)
	}
	if rs, ok := prov.(RawSaverSetter); ok {
		rs.SetRawSaver(&httpclient.StorageRawSaver{
			Store:    store,
			Provider: cfg.Source.Type,
		})
	}
	return &Collector{Provider: prov, Storage: store}, nil
}

// ParseRepoRef parses "owner/name" into a RepositoryRef.
func ParseRepoRef(providerName, repo string) (domain.RepositoryRef, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return domain.RepositoryRef{}, fmt.Errorf("invalid repo format %q, expected owner/name", repo)
	}
	return domain.RepositoryRef{
		Provider: providerName,
		Owner:    parts[0],
		Name:     parts[1],
		FullName: repo,
	}, nil
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

	commits, err := c.Provider.GetCommits(ctx, ref, domain.CommitListOptions{ListOptions: domain.ListOptions{PerPage: 100}})
	if err != nil {
		return fmt.Errorf("get commits: %w", err)
	}
	if err := c.Storage.SaveCommits(ctx, ref, commits); err != nil {
		return fmt.Errorf("save commits: %w", err)
	}

	branches, err := c.Provider.GetBranches(ctx, ref)
	if err != nil {
		return fmt.Errorf("get branches: %w", err)
	}
	if err := c.Storage.SaveBranches(ctx, ref, branches); err != nil {
		return fmt.Errorf("save branches: %w", err)
	}

	prs, err := c.Provider.GetPullRequests(ctx, ref, domain.ListOptions{PerPage: 100})
	if err != nil {
		return fmt.Errorf("get pull requests: %w", err)
	}
	if err := c.Storage.SavePullRequests(ctx, ref, prs); err != nil {
		return fmt.Errorf("save pull requests: %w", err)
	}

	issues, err := c.Provider.GetIssues(ctx, ref, domain.ListOptions{PerPage: 100})
	if err != nil {
		return fmt.Errorf("get issues: %w", err)
	}
	if err := c.Storage.SaveIssues(ctx, ref, issues); err != nil {
		return fmt.Errorf("save issues: %w", err)
	}

	contribs, err := c.Provider.GetContributors(ctx, ref)
	if err != nil {
		return fmt.Errorf("get contributors: %w", err)
	}
	if err := c.Storage.SaveContributors(ctx, ref, contribs); err != nil {
		return fmt.Errorf("save contributors: %w", err)
	}

	return nil
}
