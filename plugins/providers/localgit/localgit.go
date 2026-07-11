package localgit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/provider"
)

func init() {
	provider.Register("localgit", New)
}

// Provider reads data from a local Git repository via git CLI.
type Provider struct {
	path string
}

// New creates a new local Git provider from config.
func New(cfg map[string]string) (provider.RepositoryProvider, error) {
	return &Provider{path: cfg["path"]}, nil
}

func (p *Provider) Name() string { return "localgit" }

func (p *Provider) repoPath(ref domain.RepositoryRef) string {
	if p.path != "" {
		return p.path
	}
	return filepath.Join("repos", ref.Name)
}

func (p *Provider) ListRepositories(_ context.Context, _ domain.ListOptions) ([]domain.Repository, error) {
	if p.path == "" {
		return nil, provider.ErrNotImplemented
	}
	repo, err := p.GetRepository(context.Background(), domain.RepositoryRef{
		Provider: "localgit",
		Name:     filepath.Base(p.path),
	})
	if err != nil {
		return nil, err
	}
	return []domain.Repository{*repo}, nil
}

func (p *Provider) GetRepository(_ context.Context, ref domain.RepositoryRef) (*domain.Repository, error) {
	repoPath := p.repoPath(ref)
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}
	lang, _ := p.gitOutput(repoPath, "log", "-1", "--format=%s")
	return &domain.Repository{
		Ref: domain.RepositoryRef{
			Provider: "localgit",
			Owner:    ref.Owner,
			Name:     ref.Name,
			FullName: ref.FullName,
		},
		CloneURL: repoPath,
		Language: lang,
	}, nil
}

func (p *Provider) GetCommits(_ context.Context, ref domain.RepositoryRef, opts domain.CommitListOptions) ([]domain.Commit, error) {
	repoPath := p.repoPath(ref)
	limit := opts.PerPage
	if limit == 0 {
		limit = 100
	}
	args := []string{"-C", repoPath, "log", fmt.Sprintf("-%d", limit),
		"--format=%H|%h|%an|%ae|%s|%aI"}
	if opts.Branch != "" {
		args = append(args, opts.Branch)
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	var commits []domain.Commit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 6 {
			continue
		}
		date, _ := time.Parse(time.RFC3339, parts[5])
		commits = append(commits, domain.Commit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Author:    parts[2],
			Email:     parts[3],
			Message:   parts[4],
			Date:      date,
		})
	}
	return commits, nil
}

func (p *Provider) GetBranches(_ context.Context, ref domain.RepositoryRef) ([]domain.Branch, error) {
	repoPath := p.repoPath(ref)
	out, err := exec.Command("git", "-C", repoPath, "branch", "--format=%(refname:short)|%(objectname)").Output()
	if err != nil {
		return nil, fmt.Errorf("git branch: %w", err)
	}
	defaultBranch, _ := p.gitOutput(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	var branches []domain.Branch
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}
		branches = append(branches, domain.Branch{
			Name:    parts[0],
			Default: parts[0] == defaultBranch,
			SHA:     parts[1],
		})
	}
	return branches, nil
}

func (p *Provider) GetContributors(_ context.Context, ref domain.RepositoryRef) ([]domain.Contributor, error) {
	repoPath := p.repoPath(ref)
	out, err := exec.Command("git", "-C", repoPath, "shortlog", "-sn", "--all").Output()
	if err != nil {
		return nil, fmt.Errorf("git shortlog: %w", err)
	}
	var contribs []domain.Contributor
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var count int
		var name string
		if _, err := fmt.Sscanf(line, "%d\t%s", &count, &name); err != nil {
			if _, err := fmt.Sscanf(line, "%d %s", &count, &name); err != nil {
				continue
			}
		}
		contribs = append(contribs, domain.Contributor{
			Username:      name,
			Contributions: count,
		})
	}
	return contribs, nil
}

func (p *Provider) GetIssues(_ context.Context, _ domain.RepositoryRef, _ domain.ListOptions) ([]domain.Issue, error) {
	return nil, nil
}

func (p *Provider) GetPullRequests(_ context.Context, _ domain.RepositoryRef, _ domain.ListOptions) ([]domain.PullRequest, error) {
	return nil, nil
}

func (p *Provider) CloneRepository(_ context.Context, ref domain.RepositoryRef, _ domain.CloneOptions) (string, error) {
	return p.repoPath(ref), nil
}

func (p *Provider) gitOutput(repoPath string, args ...string) (string, error) {
	full := append([]string{"-C", repoPath}, args...)
	out, err := exec.Command("git", full...).Output()
	return strings.TrimSpace(string(out)), err
}
