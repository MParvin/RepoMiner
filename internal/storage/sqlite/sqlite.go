package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/mparvin/repo-miner/internal/core/domain"
)

// Storage implements storage.Storage using SQLite.
type Storage struct {
	db *sql.DB
}

// Open creates a new SQLite storage at the given path.
func Open(path string) (*Storage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &Storage{db: db}, nil
}

// Ping verifies the database connection is alive.
func (s *Storage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Migrate creates the database schema.
func (s *Storage) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			owner TEXT NOT NULL,
			name TEXT NOT NULL,
			full_name TEXT NOT NULL,
			description TEXT,
			url TEXT,
			clone_url TEXT,
			language TEXT,
			stars INTEGER DEFAULT 0,
			forks INTEGER DEFAULT 0,
			is_private INTEGER DEFAULT 0,
			created_at TEXT,
			updated_at TEXT,
			UNIQUE(provider, owner, name)
		)`,
		`CREATE TABLE IF NOT EXISTS commits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			owner TEXT NOT NULL,
			repo_name TEXT NOT NULL,
			hash TEXT NOT NULL,
			short_hash TEXT,
			author TEXT,
			email TEXT,
			message TEXT,
			date TEXT,
			UNIQUE(provider, owner, repo_name, hash)
		)`,
		`CREATE TABLE IF NOT EXISTS pull_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			owner TEXT NOT NULL,
			repo_name TEXT NOT NULL,
			pr_id INTEGER,
			number INTEGER,
			title TEXT,
			description TEXT,
			author TEXT,
			state TEXT,
			base_branch TEXT,
			head_branch TEXT,
			created_at TEXT,
			merged_at TEXT,
			UNIQUE(provider, owner, repo_name, number)
		)`,
		`CREATE TABLE IF NOT EXISTS issues (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			owner TEXT NOT NULL,
			repo_name TEXT NOT NULL,
			issue_id INTEGER,
			number INTEGER,
			title TEXT,
			body TEXT,
			author TEXT,
			state TEXT,
			labels TEXT,
			comments INTEGER DEFAULT 0,
			created_at TEXT,
			closed_at TEXT,
			UNIQUE(provider, owner, repo_name, number)
		)`,
		`CREATE TABLE IF NOT EXISTS raw_responses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			data BLOB NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS branches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			owner TEXT NOT NULL,
			repo_name TEXT NOT NULL,
			name TEXT NOT NULL,
			protected INTEGER DEFAULT 0,
			is_default INTEGER DEFAULT 0,
			sha TEXT,
			UNIQUE(provider, owner, repo_name, name)
		)`,
		`CREATE TABLE IF NOT EXISTS contributors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			owner TEXT NOT NULL,
			repo_name TEXT NOT NULL,
			username TEXT NOT NULL,
			contributions INTEGER DEFAULT 0,
			UNIQUE(provider, owner, repo_name, username)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	return s.db.Close()
}

// SaveRepository persists repository metadata.
func (s *Storage) SaveRepository(ctx context.Context, repo *domain.Repository) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO repositories
			(provider, owner, name, full_name, description, url, clone_url, language, stars, forks, is_private, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		repo.Ref.Provider, repo.Ref.Owner, repo.Ref.Name, repo.Ref.FullName,
		repo.Description, repo.URL, repo.CloneURL, repo.Language,
		repo.Stars, repo.Forks, boolToInt(repo.IsPrivate),
		formatTime(repo.CreatedAt), formatTime(repo.UpdatedAt),
	)
	return err
}

// GetRepository retrieves repository metadata by ref.
func (s *Storage) GetRepository(ctx context.Context, ref domain.RepositoryRef) (*domain.Repository, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT provider, owner, name, full_name, description, url, clone_url, language, stars, forks, is_private, created_at, updated_at
		FROM repositories WHERE provider = ? AND owner = ? AND name = ?`,
		ref.Provider, ref.Owner, ref.Name,
	)
	repo := &domain.Repository{Ref: ref}
	var isPrivate int
	var createdAt, updatedAt sql.NullString
	err := row.Scan(
		&repo.Ref.Provider, &repo.Ref.Owner, &repo.Ref.Name, &repo.Ref.FullName,
		&repo.Description, &repo.URL, &repo.CloneURL, &repo.Language,
		&repo.Stars, &repo.Forks, &isPrivate, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	repo.IsPrivate = isPrivate == 1
	repo.CreatedAt = parseTime(createdAt)
	repo.UpdatedAt = parseTime(updatedAt)
	return repo, nil
}

// SaveCommits persists commits for a repository.
func (s *Storage) SaveCommits(ctx context.Context, ref domain.RepositoryRef, commits []domain.Commit) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO commits
			(provider, owner, repo_name, hash, short_hash, author, email, message, date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range commits {
		if _, err := stmt.ExecContext(ctx,
			ref.Provider, ref.Owner, ref.Name,
			c.Hash, c.ShortHash, c.Author, c.Email, c.Message, formatTime(c.Date),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetCommits retrieves commits for a repository.
func (s *Storage) GetCommits(ctx context.Context, ref domain.RepositoryRef) ([]domain.Commit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT hash, short_hash, author, email, message, date
		FROM commits WHERE provider = ? AND owner = ? AND repo_name = ?
		ORDER BY date DESC`,
		ref.Provider, ref.Owner, ref.Name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []domain.Commit
	for rows.Next() {
		var c domain.Commit
		var date sql.NullString
		if err := rows.Scan(&c.Hash, &c.ShortHash, &c.Author, &c.Email, &c.Message, &date); err != nil {
			return nil, err
		}
		c.Date = parseTime(date)
		commits = append(commits, c)
	}
	return commits, rows.Err()
}

// SavePullRequests persists pull requests for a repository.
func (s *Storage) SavePullRequests(ctx context.Context, ref domain.RepositoryRef, prs []domain.PullRequest) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO pull_requests
			(provider, owner, repo_name, pr_id, number, title, description, author, state, base_branch, head_branch, created_at, merged_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, pr := range prs {
		if _, err := stmt.ExecContext(ctx,
			ref.Provider, ref.Owner, ref.Name,
			pr.ID, pr.Number, pr.Title, pr.Description, pr.Author, pr.State,
			pr.BaseBranch, pr.HeadBranch, formatTime(pr.CreatedAt), formatTime(pr.MergedAt),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetPullRequests retrieves pull requests for a repository.
func (s *Storage) GetPullRequests(ctx context.Context, ref domain.RepositoryRef) ([]domain.PullRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT pr_id, number, title, description, author, state, base_branch, head_branch, created_at, merged_at
		FROM pull_requests WHERE provider = ? AND owner = ? AND repo_name = ?
		ORDER BY number DESC`,
		ref.Provider, ref.Owner, ref.Name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []domain.PullRequest
	for rows.Next() {
		var pr domain.PullRequest
		var createdAt, mergedAt sql.NullString
		if err := rows.Scan(
			&pr.ID, &pr.Number, &pr.Title, &pr.Description, &pr.Author, &pr.State,
			&pr.BaseBranch, &pr.HeadBranch, &createdAt, &mergedAt,
		); err != nil {
			return nil, err
		}
		pr.CreatedAt = parseTime(createdAt)
		pr.MergedAt = parseTime(mergedAt)
		prs = append(prs, pr)
	}
	return prs, rows.Err()
}

// SaveIssues persists issues for a repository.
func (s *Storage) SaveIssues(ctx context.Context, ref domain.RepositoryRef, issues []domain.Issue) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO issues
			(provider, owner, repo_name, issue_id, number, title, body, author, state, labels, comments, created_at, closed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, issue := range issues {
		labels, _ := json.Marshal(issue.Labels)
		if _, err := stmt.ExecContext(ctx,
			ref.Provider, ref.Owner, ref.Name,
			issue.ID, issue.Number, issue.Title, issue.Body, issue.Author, issue.State,
			string(labels), issue.Comments, formatTime(issue.CreatedAt), formatTime(issue.ClosedAt),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetIssues retrieves issues for a repository.
func (s *Storage) GetIssues(ctx context.Context, ref domain.RepositoryRef) ([]domain.Issue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT issue_id, number, title, body, author, state, labels, comments, created_at, closed_at
		FROM issues WHERE provider = ? AND owner = ? AND repo_name = ?
		ORDER BY number DESC`,
		ref.Provider, ref.Owner, ref.Name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []domain.Issue
	for rows.Next() {
		var issue domain.Issue
		var labelsJSON sql.NullString
		var createdAt, closedAt sql.NullString
		if err := rows.Scan(
			&issue.ID, &issue.Number, &issue.Title, &issue.Body, &issue.Author, &issue.State,
			&labelsJSON, &issue.Comments, &createdAt, &closedAt,
		); err != nil {
			return nil, err
		}
		if labelsJSON.Valid {
			_ = json.Unmarshal([]byte(labelsJSON.String), &issue.Labels)
		}
		issue.CreatedAt = parseTime(createdAt)
		issue.ClosedAt = parseTime(closedAt)
		issues = append(issues, issue)
	}
	return issues, rows.Err()
}

// SaveRawResponse persists a raw API response for debugging and replay.
func (s *Storage) SaveRawResponse(ctx context.Context, prov, endpoint string, data []byte) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO raw_responses (provider, endpoint, data, created_at)
		VALUES (?, ?, ?, ?)`,
		prov, endpoint, data, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// SaveBranches persists branches for a repository.
func (s *Storage) SaveBranches(ctx context.Context, ref domain.RepositoryRef, branches []domain.Branch) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO branches
			(provider, owner, repo_name, name, protected, is_default, sha)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, b := range branches {
		if _, err := stmt.ExecContext(ctx,
			ref.Provider, ref.Owner, ref.Name,
			b.Name, boolToInt(b.Protected), boolToInt(b.Default), b.SHA,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetBranches retrieves branches for a repository.
func (s *Storage) GetBranches(ctx context.Context, ref domain.RepositoryRef) ([]domain.Branch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, protected, is_default, sha
		FROM branches WHERE provider = ? AND owner = ? AND repo_name = ?
		ORDER BY name`,
		ref.Provider, ref.Owner, ref.Name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []domain.Branch
	for rows.Next() {
		var b domain.Branch
		var protected, isDefault int
		if err := rows.Scan(&b.Name, &protected, &isDefault, &b.SHA); err != nil {
			return nil, err
		}
		b.Protected = protected == 1
		b.Default = isDefault == 1
		branches = append(branches, b)
	}
	return branches, rows.Err()
}

// SaveContributors persists contributors for a repository.
func (s *Storage) SaveContributors(ctx context.Context, ref domain.RepositoryRef, contribs []domain.Contributor) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO contributors
			(provider, owner, repo_name, username, contributions)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range contribs {
		if _, err := stmt.ExecContext(ctx,
			ref.Provider, ref.Owner, ref.Name,
			c.Username, c.Contributions,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetContributors retrieves contributors for a repository.
func (s *Storage) GetContributors(ctx context.Context, ref domain.RepositoryRef) ([]domain.Contributor, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT username, contributions
		FROM contributors WHERE provider = ? AND owner = ? AND repo_name = ?
		ORDER BY contributions DESC`,
		ref.Provider, ref.Owner, ref.Name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contribs []domain.Contributor
	for rows.Next() {
		var c domain.Contributor
		if err := rows.Scan(&c.Username, &c.Contributions); err != nil {
			return nil, err
		}
		contribs = append(contribs, c)
	}
	return contribs, rows.Err()
}

func rollbackTx(tx *sql.Tx) {
	_ = tx.Rollback()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s sql.NullString) time.Time {
	if !s.Valid || s.String == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s.String)
	return t
}
