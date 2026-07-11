package samples

import (
	"fmt"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/generator/clean"
)

// FromIssues builds dataset samples from repository issues.
func FromIssues(issues []domain.Issue, repo string) []domain.DatasetSample {
	var samples []domain.DatasetSample
	for _, issue := range issues {
		instruction := clean.Text(issue.Title)
		context := clean.Text(issue.Body)
		if instruction == "" {
			continue
		}
		samples = append(samples, domain.DatasetSample{
			Instruction: instruction,
			Context:     context,
			Metadata: map[string]string{
				"source":  "issue",
				"repo":    repo,
				"number":  fmt.Sprintf("%d", issue.Number),
				"author":  issue.Author,
				"state":   issue.State,
			},
		})
	}
	return samples
}

// FromPullRequests builds dataset samples from pull requests.
func FromPullRequests(prs []domain.PullRequest, repo string) []domain.DatasetSample {
	var samples []domain.DatasetSample
	for _, pr := range prs {
		instruction := clean.Text(pr.Title)
		context := clean.Text(pr.Description)
		if instruction == "" {
			continue
		}
		solution := fmt.Sprintf("Merged PR #%d from branch %s into %s", pr.Number, pr.HeadBranch, pr.BaseBranch)
		if pr.State != "merged" && pr.State != "closed" {
			solution = fmt.Sprintf("PR #%d: %s branch %s -> %s", pr.Number, pr.State, pr.HeadBranch, pr.BaseBranch)
		}
		samples = append(samples, domain.DatasetSample{
			Instruction: instruction,
			Context:     context,
			Solution:    solution,
			Metadata: map[string]string{
				"source": "pull_request",
				"repo":   repo,
				"number": fmt.Sprintf("%d", pr.Number),
				"author": pr.Author,
				"state":  pr.State,
			},
		})
	}
	return samples
}

// FromCommits builds dataset samples from commit messages.
func FromCommits(commits []domain.Commit, repo string) []domain.DatasetSample {
	var samples []domain.DatasetSample
	for _, c := range commits {
		msg := clean.Text(c.Message)
		if msg == "" || len(msg) < 10 {
			continue
		}
		lines := splitFirstLine(msg)
		samples = append(samples, domain.DatasetSample{
			Instruction: lines,
			Context:     fmt.Sprintf("Commit %s by %s", c.ShortHash, c.Author),
			Solution:    msg,
			Metadata: map[string]string{
				"source": "commit",
				"repo":   repo,
				"hash":   c.Hash,
				"author": c.Author,
			},
		})
	}
	return samples
}

func splitFirstLine(s string) string {
	for i, r := range s {
		if r == '\n' {
			return s[:i]
		}
	}
	return s
}
