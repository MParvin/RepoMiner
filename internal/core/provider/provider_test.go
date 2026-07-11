package provider_test

import (
	"testing"

	"github.com/mparvin/repo-miner/internal/core/provider"

	// Register plugins for test
	_ "github.com/mparvin/repo-miner/plugins/providers/gitea"
	_ "github.com/mparvin/repo-miner/plugins/providers/github"
	_ "github.com/mparvin/repo-miner/plugins/providers/gitlab"
	_ "github.com/mparvin/repo-miner/plugins/providers/localgit"
)

func TestProviderRegistry(t *testing.T) {
	expected := []string{"github", "gitlab", "gitea", "localgit"}
	registered := provider.List()

	for _, name := range expected {
		found := false
		for _, r := range registered {
			if r == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("provider %q not registered; got %v", name, registered)
		}
	}
}

func TestProviderGet(t *testing.T) {
	cfg := map[string]string{"url": "https://api.github.com", "token": ""}
	p, err := provider.Get("github", cfg)
	if err != nil {
		t.Fatalf("get github provider: %v", err)
	}
	if p.Name() != "github" {
		t.Errorf("expected name github, got %s", p.Name())
	}
}

func TestProviderUnknown(t *testing.T) {
	_, err := provider.Get("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
