package collector_test

import (
	"strings"
	"testing"

	"github.com/mparvin/repo-miner/internal/collector"
	"github.com/mparvin/repo-miner/internal/core/domain"
)

func TestBuildSearchQuery(t *testing.T) {
	opts := domain.SearchOptions{
		Keywords:     "gin",
		Language:     "Go",
		CreatedAfter: "2026-01-01",
	}
	got := collector.BuildSearchQuery(opts)
	for _, part := range []string{"gin", "language:Go", "created:>2026-01-01"} {
		if !strings.Contains(got, part) {
			t.Errorf("query %q missing %q", got, part)
		}
	}
}

func TestBuildSearchQueryRaw(t *testing.T) {
	opts := domain.SearchOptions{Query: "gin created:>2026-01-01 language:Go"}
	got := collector.BuildSearchQuery(opts)
	if got != opts.Query {
		t.Errorf("raw query should pass through, got %q", got)
	}
}

func TestHasSearchCriteria(t *testing.T) {
	if !collector.HasSearchCriteria(domain.SearchOptions{Language: "Go"}) {
		t.Error("language should count as search criteria")
	}
	if collector.HasSearchCriteria(domain.SearchOptions{}) {
		t.Error("empty opts should not have criteria")
	}
}
