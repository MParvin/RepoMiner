package generator

import (
	"context"
	"fmt"

	"github.com/mparvin/repo-miner/internal/collector"
	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/generator/clean"
	"github.com/mparvin/repo-miner/internal/generator/dedupe"
	"github.com/mparvin/repo-miner/internal/generator/export"
	"github.com/mparvin/repo-miner/internal/generator/samples"
	"github.com/mparvin/repo-miner/internal/storage"
)

const (
	minInstructionLen = 10
	maxSampleLen      = 8000
)

// Generator orchestrates dataset sample generation from collected data.
type Generator struct {
	Storage storage.Storage
}

// New creates a Generator from application config and storage.
func New(_ *config.Config, store storage.Storage) (*Generator, error) {
	return &Generator{Storage: store}, nil
}

// Generate produces dataset samples for a repository and writes output files.
func (g *Generator) Generate(ctx context.Context, ref domain.RepositoryRef, outputPath, format string) (int, error) {
	repo, err := g.Storage.GetRepository(ctx, ref)
	if err != nil {
		return 0, fmt.Errorf("get repository: %w", err)
	}

	issues, err := g.Storage.GetIssues(ctx, ref)
	if err != nil {
		return 0, fmt.Errorf("get issues: %w", err)
	}
	prs, err := g.Storage.GetPullRequests(ctx, ref)
	if err != nil {
		return 0, fmt.Errorf("get pull requests: %w", err)
	}
	commits, err := g.Storage.GetCommits(ctx, ref)
	if err != nil {
		return 0, fmt.Errorf("get commits: %w", err)
	}

	repoName := repo.Ref.FullName
	var all []domain.DatasetSample
	all = append(all, samples.FromIssues(issues, repoName)...)
	all = append(all, samples.FromPullRequests(prs, repoName)...)
	all = append(all, samples.FromCommits(commits, repoName)...)

	all = filterQuality(all)
	all = dedupe.Filter(all)

	if len(all) == 0 {
		return 0, fmt.Errorf("no samples generated for %s", repoName)
	}

	switch format {
	case "huggingface", "hf":
		dir := outputPath
		name := ref.Name
		if err := export.WriteHuggingFace(dir, name, all); err != nil {
			return 0, err
		}
	default:
		if err := export.WriteJSONL(outputPath, all); err != nil {
			return 0, err
		}
	}

	return len(all), nil
}

func filterQuality(samples []domain.DatasetSample) []domain.DatasetSample {
	result := make([]domain.DatasetSample, 0, len(samples))
	for _, s := range samples {
		s.Instruction = clean.Text(s.Instruction)
		s.Context = clean.Text(s.Context)
		s.Solution = clean.Text(s.Solution)
		if clean.IsQuality(s.Instruction, s.Context, s.Solution, minInstructionLen, maxSampleLen) {
			result = append(result, s)
		}
	}
	return result
}

// ParseRepoRef re-exports collector.ParseRepoRef for CLI use.
func ParseRepoRef(providerName, repo string) (domain.RepositoryRef, error) {
	return collector.ParseRepoRef(providerName, repo)
}
