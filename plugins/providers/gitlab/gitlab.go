package gitlab

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
	provider.Register("gitlab", New)
}

// Provider implements GitLab API v4 access (cloud and self-hosted).
type Provider struct {
	client *httpclient.Client
}

// New creates a new GitLab provider from config.
func New(cfg map[string]string) (provider.RepositoryProvider, error) {
	baseURL := cfg["url"]
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	baseURL = strings.TrimRight(baseURL, "/") + "/api/v4"

	return &Provider{
		client: httpclient.New(httpclient.Config{
			BaseURL:  baseURL,
			Token:    cfg["token"],
			Provider: "gitlab",
		}),
	}, nil
}

// SetRawSaver configures raw response persistence.
func (p *Provider) SetRawSaver(saver httpclient.RawSaver) {
	p.client = httpclient.New(httpclient.Config{
		BaseURL:  p.client.BaseURL(),
		Token:    p.client.Token(),
		Provider: "gitlab",
		RawSaver: saver,
	})
}

func (p *Provider) Name() string { return "gitlab" }

func (p *Provider) projectID(ref domain.RepositoryRef) string {
	return url.PathEscape(ref.Owner + "/" + ref.Name)
}

func (p *Provider) ListRepositories(ctx context.Context, opts domain.ListOptions) ([]domain.Repository, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 30
	}
	page := opts.Page
	if page == 0 {
		page = 1
	}
	path := fmt.Sprintf("/projects?membership=true&per_page=%d&page=%d", perPage, page)
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var projects []glProject
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	result := make([]domain.Repository, 0, len(projects))
	for _, pr := range projects {
		result = append(result, mapProject(pr))
	}
	return result, nil
}

func (p *Provider) GetRepository(ctx context.Context, ref domain.RepositoryRef) (*domain.Repository, error) {
	path := fmt.Sprintf("/projects/%s", p.projectID(ref))
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var pr glProject
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, err
	}
	repo := mapProject(pr)
	return &repo, nil
}

func (p *Provider) GetCommits(ctx context.Context, ref domain.RepositoryRef, opts domain.CommitListOptions) ([]domain.Commit, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 100
	}
	path := fmt.Sprintf("/projects/%s/repository/commits?per_page=%d", p.projectID(ref), perPage)
	if opts.Since != "" {
		path += "&since=" + url.QueryEscape(opts.Since)
	}
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var commits []glCommit
	if err := json.Unmarshal(data, &commits); err != nil {
		return nil, err
	}
	result := make([]domain.Commit, 0, len(commits))
	for _, c := range commits {
		result = append(result, domain.Commit{
			Hash:      c.ID,
			ShortHash: c.ShortID,
			Author:    c.AuthorName,
			Email:     c.AuthorEmail,
			Message:   c.Message,
			Date:      parseTime(c.CreatedAt),
		})
	}
	return result, nil
}

func (p *Provider) GetBranches(ctx context.Context, ref domain.RepositoryRef) ([]domain.Branch, error) {
	path := fmt.Sprintf("/projects/%s/repository/branches?per_page=100", p.projectID(ref))
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var branches []glBranch
	if err := json.Unmarshal(data, &branches); err != nil {
		return nil, err
	}
	result := make([]domain.Branch, 0, len(branches))
	for _, b := range branches {
		result = append(result, domain.Branch{
			Name:      b.Name,
			Protected: b.Protected,
			Default:   b.Default,
			SHA:       b.Commit.ID,
		})
	}
	return result, nil
}

func (p *Provider) GetContributors(ctx context.Context, ref domain.RepositoryRef) ([]domain.Contributor, error) {
	path := fmt.Sprintf("/projects/%s/repository/contributors", p.projectID(ref))
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var contribs []glContributor
	if err := json.Unmarshal(data, &contribs); err != nil {
		return nil, err
	}
	result := make([]domain.Contributor, 0, len(contribs))
	for _, c := range contribs {
		result = append(result, domain.Contributor{
			Username:      c.Name,
			Contributions: c.Commits,
		})
	}
	return result, nil
}

func (p *Provider) GetIssues(ctx context.Context, ref domain.RepositoryRef, opts domain.ListOptions) ([]domain.Issue, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 100
	}
	path := fmt.Sprintf("/projects/%s/issues?state=all&per_page=%d", p.projectID(ref), perPage)
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var issues []glIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}
	result := make([]domain.Issue, 0, len(issues))
	for _, i := range issues {
		labels := append([]string(nil), i.Labels...)
		result = append(result, domain.Issue{
			ID:        i.ID,
			Number:    i.IID,
			Title:     i.Title,
			Body:      i.Description,
			Author:    i.Author.Username,
			State:     i.State,
			Labels:    labels,
			Comments:  i.UserNotesCount,
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
	path := fmt.Sprintf("/projects/%s/merge_requests?state=all&per_page=%d", p.projectID(ref), perPage)
	data, err := p.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var mrs []glMR
	if err := json.Unmarshal(data, &mrs); err != nil {
		return nil, err
	}
	result := make([]domain.PullRequest, 0, len(mrs))
	for _, mr := range mrs {
		result = append(result, domain.PullRequest{
			ID:          mr.ID,
			Number:      mr.IID,
			Title:       mr.Title,
			Description: mr.Description,
			Author:      mr.Author.Username,
			State:       mr.State,
			BaseBranch:  mr.TargetBranch,
			HeadBranch:  mr.SourceBranch,
			CreatedAt:   parseTime(mr.CreatedAt),
			MergedAt:    parseTime(mr.MergedAt),
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

type glProject struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	Description       string `json:"description"`
	WebURL            string `json:"web_url"`
	HTTPURLToRepo     string `json:"http_url_to_repo"`
	StarCount         int    `json:"star_count"`
	ForksCount        int    `json:"forks_count"`
	Visibility        string `json:"visibility"`
	CreatedAt         string `json:"created_at"`
	LastActivityAt    string `json:"last_activity_at"`
	Namespace         struct {
		Path string `json:"path"`
	} `json:"namespace"`
}

type glCommit struct {
	ID           string `json:"id"`
	ShortID      string `json:"short_id"`
	Title        string `json:"title"`
	Message      string `json:"message"`
	AuthorName   string `json:"author_name"`
	AuthorEmail  string `json:"author_email"`
	CreatedAt    string `json:"created_at"`
}

type glBranch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	Default   bool   `json:"default"`
	Commit    struct {
		ID string `json:"id"`
	} `json:"commit"`
}

type glContributor struct {
	Name    string `json:"name"`
	Commits int    `json:"commits"`
}

type glIssue struct {
	ID             int64    `json:"id"`
	IID            int      `json:"iid"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	State          string   `json:"state"`
	Labels         []string `json:"labels"`
	UserNotesCount int      `json:"user_notes_count"`
	CreatedAt      string   `json:"created_at"`
	ClosedAt       string   `json:"closed_at"`
	Author         struct {
		Username string `json:"username"`
	} `json:"author"`
}

type glMR struct {
	ID            int64  `json:"id"`
	IID           int    `json:"iid"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	State         string `json:"state"`
	SourceBranch  string `json:"source_branch"`
	TargetBranch  string `json:"target_branch"`
	CreatedAt     string `json:"created_at"`
	MergedAt      string `json:"merged_at"`
	Author        struct {
		Username string `json:"username"`
	} `json:"author"`
}

func mapProject(pr glProject) domain.Repository {
	parts := strings.SplitN(pr.PathWithNamespace, "/", 2)
	owner := pr.Namespace.Path
	name := pr.Name
	if len(parts) == 2 {
		owner = parts[0]
		name = parts[1]
	}
	return domain.Repository{
		Ref: domain.RepositoryRef{
			Provider: "gitlab",
			Owner:    owner,
			Name:     name,
			FullName: pr.PathWithNamespace,
		},
		Description: pr.Description,
		URL:         pr.WebURL,
		CloneURL:    pr.HTTPURLToRepo,
		Stars:       pr.StarCount,
		Forks:       pr.ForksCount,
		IsPrivate:   pr.Visibility != "public",
		CreatedAt:   parseTime(pr.CreatedAt),
		UpdatedAt:   parseTime(pr.LastActivityAt),
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
