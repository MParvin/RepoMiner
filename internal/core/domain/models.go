package domain

import "time"

// RepositoryRef identifies a repository within a provider.
type RepositoryRef struct {
	Provider string `json:"provider"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// Repository holds metadata about a source code repository.
type Repository struct {
	Ref         RepositoryRef `json:"ref"`
	Description string        `json:"description,omitempty"`
	URL         string        `json:"url,omitempty"`
	CloneURL    string        `json:"clone_url,omitempty"`
	Language    string        `json:"language,omitempty"`
	Stars       int           `json:"stars,omitempty"`
	Forks       int           `json:"forks,omitempty"`
	IsPrivate   bool          `json:"is_private,omitempty"`
	CreatedAt   time.Time     `json:"created_at,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at,omitempty"`
}

// Commit represents a single version control commit.
type Commit struct {
	Hash      string    `json:"hash"`
	ShortHash string    `json:"short_hash,omitempty"`
	Author    string    `json:"author"`
	Email     string    `json:"email,omitempty"`
	Message   string    `json:"message"`
	Date      time.Time `json:"date"`
}

// Branch represents a repository branch.
type Branch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected,omitempty"`
	Default   bool   `json:"default,omitempty"`
	SHA       string `json:"sha,omitempty"`
}

// PullRequest represents a pull/merge request.
type PullRequest struct {
	ID          int64     `json:"id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Author      string    `json:"author"`
	State       string    `json:"state"`
	BaseBranch  string    `json:"base_branch,omitempty"`
	HeadBranch  string    `json:"head_branch,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	MergedAt    time.Time `json:"merged_at,omitempty"`
}

// Issue represents a repository issue.
type Issue struct {
	ID          int64     `json:"id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body,omitempty"`
	Author      string    `json:"author"`
	State       string    `json:"state"`
	Labels      []string  `json:"labels,omitempty"`
	Comments    int       `json:"comments,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	ClosedAt    time.Time `json:"closed_at,omitempty"`
}

// Contributor represents a repository contributor.
type Contributor struct {
	Username      string `json:"username"`
	Contributions int    `json:"contributions"`
}

// ListOptions controls pagination for list operations.
type ListOptions struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
}

// CommitListOptions controls commit listing.
type CommitListOptions struct {
	ListOptions
	Branch string `json:"branch,omitempty"`
	Since  string `json:"since,omitempty"`
	Until  string `json:"until,omitempty"`
}

// CloneOptions controls repository cloning.
type CloneOptions struct {
	Destination string `json:"destination"`
	Depth       int    `json:"depth,omitempty"`
}

// AnalysisResult holds the output of a language analyzer.
type AnalysisResult struct {
	Language              string   `json:"language"`
	Packages              int      `json:"packages"`
	Functions             int      `json:"functions"`
	Structs               int      `json:"structs"`
	Interfaces            int      `json:"interfaces"`
	Dependencies          []string `json:"dependencies,omitempty"`
	TestFiles             int      `json:"test_files"`
	DocumentedFunctions   int      `json:"documented_functions"`
	HasTests              bool     `json:"tests"`
	TestCoverageSignal    float64  `json:"test_coverage_signal"`
	StructureQualityScore float64  `json:"structure_quality_score"`
	ComplexityScore       float64  `json:"complexity_score"`
}

// DatasetSample represents a single training example.
type DatasetSample struct {
	Instruction string            `json:"instruction"`
	Context     string            `json:"context,omitempty"`
	Solution    string            `json:"solution,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Job represents a unit of work in the processing queue.
type Job struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`
	Payload map[string]string `json:"payload"`
}
