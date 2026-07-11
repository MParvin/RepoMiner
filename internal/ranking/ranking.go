package ranking

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/storage"
)

// Reason describes a single score contribution.
type Reason struct {
	Category string `json:"category"`
	Detail   string `json:"detail"`
	Points   int    `json:"points"`
}

// Result holds the ranking result for a repository.
type Result struct {
	Repository string   `json:"repository"`
	TotalScore int      `json:"total_score"`
	AIScore    int      `json:"ai_score"`
	QualityScore int    `json:"quality_score"`
	Reasons    []Reason `json:"reasons"`
}

// Engine scores repositories by AI signals and engineering quality.
type Engine struct {
	Storage storage.Storage
	Weights config.RankingWeights
	ReposDir string
}

// New creates a ranking engine.
func New(cfg *config.Config, store storage.Storage) *Engine {
	weights := cfg.Ranking.Weights
	if weights.AgentsMD == 0 && weights.Tests == 0 {
		weights = config.DefaultRankingWeights()
	}
	return &Engine{
		Storage:  store,
		Weights:  weights,
		ReposDir: cfg.Workspace.ReposDir,
	}
}

// RankAll scores all repositories in storage.
func (e *Engine) RankAll(ctx context.Context) ([]Result, error) {
	refs, err := e.listRepositories(ctx)
	if err != nil {
		return nil, err
	}
	var results []Result
	for _, ref := range refs {
		r, err := e.Score(ctx, ref)
		if err != nil {
			continue
		}
		results = append(results, r)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalScore > results[j].TotalScore
	})
	return results, nil
}

// Score evaluates a single repository.
func (e *Engine) Score(ctx context.Context, ref domain.RepositoryRef) (Result, error) {
	repo, err := e.Storage.GetRepository(ctx, ref)
	if err != nil {
		return Result{}, err
	}

	repoPath := filepath.Join(e.ReposDir, ref.Name)
	var reasons []Reason
	aiScore := 0
	qualityScore := 0

	// AI assistance signals (file-based)
	if fileExists(filepath.Join(repoPath, "AGENTS.md")) {
		reasons = append(reasons, Reason{"AI", "AGENTS.md", e.Weights.AgentsMD})
		aiScore += e.Weights.AgentsMD
	}
	if fileExists(filepath.Join(repoPath, "CLAUDE.md")) {
		reasons = append(reasons, Reason{"AI", "CLAUDE.md", e.Weights.ClaudeMD})
		aiScore += e.Weights.ClaudeMD
	}
	if dirExists(filepath.Join(repoPath, ".cursor")) {
		reasons = append(reasons, Reason{"AI", ".cursor/", e.Weights.CursorDir})
		aiScore += e.Weights.CursorDir
	}
	if hasAIFiles(repoPath) {
		reasons = append(reasons, Reason{"AI", "AI-related files", 5})
		aiScore += 5
	}

	// AI commit messages
	commits, _ := e.Storage.GetCommits(ctx, ref)
	aiCommits := countAICommits(commits)
	if aiCommits > 0 {
		pts := min(e.Weights.AICommits, aiCommits*2)
		reasons = append(reasons, Reason{"AI", fmt.Sprintf("AI commit messages (%d)", aiCommits), pts})
		aiScore += pts
	}

	// Engineering quality
	if hasTestFiles(repoPath) {
		reasons = append(reasons, Reason{"Quality", "Tests", e.Weights.Tests})
		qualityScore += e.Weights.Tests
	}
	if hasCI(repoPath) {
		reasons = append(reasons, Reason{"Quality", "CI configuration", e.Weights.CI})
		qualityScore += e.Weights.CI
	}
	if fileExists(filepath.Join(repoPath, "README.md")) {
		reasons = append(reasons, Reason{"Quality", "README.md", e.Weights.README})
		qualityScore += e.Weights.README
	}
	if dirExists(filepath.Join(repoPath, "docs")) {
		reasons = append(reasons, Reason{"Quality", "docs/", e.Weights.Docs})
		qualityScore += e.Weights.Docs
	}

	// Activity
	if !repo.UpdatedAt.IsZero() && time.Since(repo.UpdatedAt) < 90*24*time.Hour {
		reasons = append(reasons, Reason{"Activity", "Recent activity (<90d)", e.Weights.Activity})
		qualityScore += e.Weights.Activity
	} else if repo.Stars > 1000 {
		reasons = append(reasons, Reason{"Activity", fmt.Sprintf("High stars (%d)", repo.Stars), e.Weights.Activity/2})
		qualityScore += e.Weights.Activity / 2
	}

	// Maintainers
	contribs, _ := e.Storage.GetContributors(ctx, ref)
	if len(contribs) >= 5 {
		reasons = append(reasons, Reason{"Activity", fmt.Sprintf("Maintainers (%d contributors)", len(contribs)), e.Weights.Maintainers})
		qualityScore += e.Weights.Maintainers
	}

	return Result{
		Repository:   repo.Ref.FullName,
		TotalScore:   aiScore + qualityScore,
		AIScore:      aiScore,
		QualityScore: qualityScore,
		Reasons:      reasons,
	}, nil
}

func (e *Engine) listRepositories(ctx context.Context) ([]domain.RepositoryRef, error) {
	// Use SQLite directly via a simple query approach - list from storage
	if lister, ok := e.Storage.(RepositoryLister); ok {
		return lister.ListRepositories(ctx)
	}
	return nil, fmt.Errorf("storage does not support listing repositories")
}

// RepositoryLister lists all stored repository refs.
type RepositoryLister interface {
	ListRepositories(ctx context.Context) ([]domain.RepositoryRef, error)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasTestFiles(repoPath string) bool {
	found := false
	_ = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || strings.Contains(path, "/test/") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func hasCI(repoPath string) bool {
	ciPaths := []string{
		".github/workflows",
		".gitlab-ci.yml",
		"Jenkinsfile",
		".circleci",
	}
	for _, p := range ciPaths {
		if fileExists(filepath.Join(repoPath, p)) || dirExists(filepath.Join(repoPath, p)) {
			return true
		}
	}
	return false
}

func hasAIFiles(repoPath string) bool {
	aiFiles := []string{".copilot", "copilot-instructions.md", ".github/copilot-instructions.md"}
	for _, f := range aiFiles {
		if fileExists(filepath.Join(repoPath, f)) {
			return true
		}
	}
	return false
}

func countAICommits(commits []domain.Commit) int {
	keywords := []string{"copilot", "claude", "cursor", "ai-assisted", "chatgpt", "llm"}
	count := 0
	for _, c := range commits {
		msg := strings.ToLower(c.Message)
		for _, kw := range keywords {
			if strings.Contains(msg, kw) {
				count++
				break
			}
		}
	}
	return count
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
