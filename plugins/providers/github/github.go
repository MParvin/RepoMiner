package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/httpclient"
	"github.com/mparvin/repo-miner/internal/core/provider"
)

func init() {
	provider.Register("github", New)
}

// Provider implements GitHub REST API v3 access.
type Provider struct {
	client *httpclient.Client
}

// New creates a new GitHub provider from config.
func New(cfg map[string]string) (provider.RepositoryProvider, error) {
	baseURL := cfg["url"]
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Provider{
		client: httpclient.New(httpclient.Config{
			BaseURL:  baseURL,
			Token:    cfg["token"],
			Provider: "github",
		}),
	}, nil
}

// SetRawSaver configures raw response persistence.
func (p *Provider) SetRawSaver(saver httpclient.RawSaver) {
	p.client = httpclient.New(httpclient.Config{
		BaseURL:  p.client.BaseURL(),
		Token:    p.client.Token(),
		Provider: "github",
		RawSaver: saver,
	})
}

func (p *Provider) Name() string { return "github" }

func (p *Provider) ListRepositories(ctx context.Context, opts domain.ListOptions) ([]domain.Repository, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 30
	}
	page := opts.Page
	if page == 0 {
		page = 1
	}
	path := fmt.Sprintf("/user/repos?per_page=%d&page=%d&sort=updated", perPage, page)
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var repos []ghRepo
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, err
	}
	result := make([]domain.Repository, 0, len(repos))
	for _, r := range repos {
		result = append(result, mapRepo(r))
	}
	return result, nil
}

func (p *Provider) GetRepository(ctx context.Context, ref domain.RepositoryRef) (*domain.Repository, error) {
	path := fmt.Sprintf("/repos/%s/%s", ref.Owner, ref.Name)
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var r ghRepo
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	repo := mapRepo(r)
	return &repo, nil
}

func (p *Provider) GetCommits(ctx context.Context, ref domain.RepositoryRef, opts domain.CommitListOptions) ([]domain.Commit, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 100
	}
	var all []domain.Commit
	for page := 1; page <= 3; page++ {
		path := fmt.Sprintf("/repos/%s/%s/commits?per_page=%d&page=%d", ref.Owner, ref.Name, perPage, page)
		if opts.Since != "" {
			path += "&since=" + url.QueryEscape(opts.Since)
		}
		data, err := p.client.Get(ctx, path)
		if err != nil {
			return nil, err
		}
		var commits []ghCommit
		if err := json.Unmarshal(data, &commits); err != nil {
			return nil, err
		}
		if len(commits) == 0 {
			break
		}
		for _, c := range commits {
			all = append(all, domain.Commit{
				Hash:      c.SHA,
				ShortHash: c.SHA[:7],
				Author:    c.Commit.Author.Name,
				Email:     c.Commit.Author.Email,
				Message:   c.Commit.Message,
				Date:      parseTime(c.Commit.Author.Date),
			})
		}
		if len(commits) < perPage {
			break
		}
	}
	return all, nil
}

func (p *Provider) GetBranches(ctx context.Context, ref domain.RepositoryRef) ([]domain.Branch, error) {
	path := fmt.Sprintf("/repos/%s/%s/branches?per_page=100", ref.Owner, ref.Name)
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var branches []ghBranch
	if err := json.Unmarshal(data, &branches); err != nil {
		return nil, err
	}
	result := make([]domain.Branch, 0, len(branches))
	for _, b := range branches {
		result = append(result, domain.Branch{
			Name:      b.Name,
			Protected: b.Protected,
			SHA:       b.Commit.SHA,
		})
	}
	return result, nil
}

func (p *Provider) GetContributors(ctx context.Context, ref domain.RepositoryRef) ([]domain.Contributor, error) {
	path := fmt.Sprintf("/repos/%s/%s/contributors?per_page=100", ref.Owner, ref.Name)
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var contribs []ghContributor
	if err := json.Unmarshal(data, &contribs); err != nil {
		return nil, err
	}
	result := make([]domain.Contributor, 0, len(contribs))
	for _, c := range contribs {
		result = append(result, domain.Contributor{
			Username:      c.Login,
			Contributions: c.Contributions,
		})
	}
	return result, nil
}

func (p *Provider) GetIssues(ctx context.Context, ref domain.RepositoryRef, opts domain.ListOptions) ([]domain.Issue, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 100
	}
	path := fmt.Sprintf("/repos/%s/%s/issues?state=all&per_page=%d", ref.Owner, ref.Name, perPage)
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var issues []ghIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}
	result := make([]domain.Issue, 0)
	for _, i := range issues {
		if i.PullRequest.URL != "" {
			continue // skip PRs
		}
		labels := make([]string, 0, len(i.Labels))
		for _, l := range i.Labels {
			labels = append(labels, l.Name)
		}
		result = append(result, domain.Issue{
			ID:        i.ID,
			Number:    i.Number,
			Title:     i.Title,
			Body:      i.Body,
			Author:    i.User.Login,
			State:     i.State,
			Labels:    labels,
			Comments:  i.Comments,
			CreatedAt: parseTime(i.CreatedAt),
			ClosedAt:  parseTime(i.ClosedAt),
		})
	}
	return result, nil
}

func (p *Provider) GetPullRequests(ctx context.Context, ref domain.RepositoryRef, opts domain.ListOptions) ([]domain.PullRequest, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 100
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=all&per_page=%d", ref.Owner, ref.Name, perPage)
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var prs []ghPR
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, err
	}
	result := make([]domain.PullRequest, 0, len(prs))
	for _, pr := range prs {
		result = append(result, domain.PullRequest{
			ID:          pr.ID,
			Number:      pr.Number,
			Title:       pr.Title,
			Description: pr.Body,
			Author:      pr.User.Login,
			State:       pr.State,
			BaseBranch:  pr.Base.Ref,
			HeadBranch:  pr.Head.Ref,
			CreatedAt:   parseTime(pr.CreatedAt),
			MergedAt:    parseTime(pr.MergedAt),
		})
	}
	return result, nil
}

func (p *Provider) CloneRepository(ctx context.Context, ref domain.RepositoryRef, opts domain.CloneOptions) (string, error) {
	repo, err := p.GetRepository(ctx, ref)
	if err != nil {
		return "", err
	}
	dest := opts.Destination
	if dest == "" {
		dest = filepath.Join("repos", ref.Name)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	args := []string{"clone"}
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}
	args = append(args, repo.CloneURL, dest)
	cmd := exec.CommandContext(ctx, "git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone: %w: %s", err, string(out))
	}
	return dest, nil
}

// GitHub API response types

type ghRepo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	Language    string `json:"language"`
	Stargazers  int    `json:"stargazers_count"`
	Forks       int    `json:"forks_count"`
	Private     bool   `json:"private"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type ghCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type ghBranch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	Commit    struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

type ghContributor struct {
	Login         string `json:"login"`
	Contributions int    `json:"contributions"`
}

type ghIssue struct {
	ID        int64  `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	Comments  int    `json:"comments"`
	CreatedAt string `json:"created_at"`
	ClosedAt  string `json:"closed_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	PullRequest struct {
		URL string `json:"url"`
	} `json:"pull_request"`
}

type ghPR struct {
	ID        int64  `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	MergedAt  string `json:"merged_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

func mapRepo(r ghRepo) domain.Repository {
	parts := strings.SplitN(r.FullName, "/", 2)
	owner := r.Owner.Login
	name := r.Name
	if len(parts) == 2 {
		owner = parts[0]
		name = parts[1]
	}
	return domain.Repository{
		Ref: domain.RepositoryRef{
			Provider: "github",
			Owner:    owner,
			Name:     name,
			FullName: r.FullName,
		},
		Description: r.Description,
		URL:         r.HTMLURL,
		CloneURL:    r.CloneURL,
		Language:    r.Language,
		Stars:       r.Stargazers,
		Forks:       r.Forks,
		IsPrivate:   r.Private,
		CreatedAt:   parseTime(r.CreatedAt),
		UpdatedAt:   parseTime(r.UpdatedAt),
	}
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse("2006-01-02T15:04:05Z", s)
	}
	return t
}
