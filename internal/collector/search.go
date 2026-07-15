package collector

import (
	"fmt"
	"strings"

	"github.com/mparvin/repo-miner/internal/core/domain"
)

// BuildSearchQuery composes a provider search query from structured options.
// Example: keywords=gin + language=Go + created_after=2026-01-01
//   => "gin created:>2026-01-01 language:Go"
func BuildSearchQuery(opts domain.SearchOptions) string {
	if q := strings.TrimSpace(opts.Query); q != "" {
		return q
	}

	var parts []string
	if kw := strings.TrimSpace(opts.Keywords); kw != "" {
		parts = append(parts, kw)
	}
	if opts.User != "" {
		parts = append(parts, "user:"+opts.User)
	}
	if opts.Org != "" {
		parts = append(parts, "org:"+opts.Org)
	}
	if opts.Language != "" {
		parts = append(parts, "language:"+opts.Language)
	}
	if opts.CreatedAfter != "" {
		parts = append(parts, "created:>"+opts.CreatedAfter)
	}
	if opts.CreatedBefore != "" {
		parts = append(parts, "created:<"+opts.CreatedBefore)
	}
	if opts.MinStars > 0 {
		parts = append(parts, fmt.Sprintf("stars:>=%d", opts.MinStars))
	}
	if opts.MaxStars > 0 {
		parts = append(parts, fmt.Sprintf("stars:<=%d", opts.MaxStars))
	}
	if opts.Topic != "" {
		parts = append(parts, "topic:"+opts.Topic)
	}
	if opts.Forks != nil {
		parts = append(parts, fmt.Sprintf("fork:%t", *opts.Forks))
	}
	if opts.Archived != nil {
		parts = append(parts, fmt.Sprintf("archived:%t", *opts.Archived))
	}
	return strings.Join(parts, " ")
}

// HasSearchCriteria returns true if any search filter is set.
func HasSearchCriteria(opts domain.SearchOptions) bool {
	return strings.TrimSpace(opts.Query) != "" ||
		strings.TrimSpace(opts.Keywords) != "" ||
		opts.Language != "" ||
		opts.CreatedAfter != "" ||
		opts.CreatedBefore != "" ||
		opts.MinStars > 0 ||
		opts.MaxStars > 0 ||
		opts.Topic != "" ||
		opts.User != "" ||
		opts.Org != "" ||
		opts.Forks != nil ||
		opts.Archived != nil
}
