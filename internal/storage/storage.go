package storage

import (
	"context"

	"github.com/mparvin/repo-miner/internal/core/domain"
)

// Storage abstracts persistence for collected repository data.
type Storage interface {
	Ping(ctx context.Context) error
	Migrate(ctx context.Context) error
	Close() error

	SaveRepository(ctx context.Context, repo *domain.Repository) error
	GetRepository(ctx context.Context, ref domain.RepositoryRef) (*domain.Repository, error)

	SaveCommits(ctx context.Context, ref domain.RepositoryRef, commits []domain.Commit) error
	GetCommits(ctx context.Context, ref domain.RepositoryRef) ([]domain.Commit, error)

	SavePullRequests(ctx context.Context, ref domain.RepositoryRef, prs []domain.PullRequest) error
	GetPullRequests(ctx context.Context, ref domain.RepositoryRef) ([]domain.PullRequest, error)

	SaveIssues(ctx context.Context, ref domain.RepositoryRef, issues []domain.Issue) error
	GetIssues(ctx context.Context, ref domain.RepositoryRef) ([]domain.Issue, error)

	SaveBranches(ctx context.Context, ref domain.RepositoryRef, branches []domain.Branch) error
	GetBranches(ctx context.Context, ref domain.RepositoryRef) ([]domain.Branch, error)

	SaveContributors(ctx context.Context, ref domain.RepositoryRef, contributors []domain.Contributor) error
	GetContributors(ctx context.Context, ref domain.RepositoryRef) ([]domain.Contributor, error)

	SaveRawResponse(ctx context.Context, provider, endpoint string, data []byte) error
}
