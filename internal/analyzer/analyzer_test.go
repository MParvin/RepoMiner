package analyzer_test

import (
	"testing"

	"github.com/mparvin/repo-miner/internal/analyzer"

	_ "github.com/mparvin/repo-miner/plugins/analyzers/golang"
)

func TestAnalyzerRegistry(t *testing.T) {
	registered := analyzer.List()
	found := false
	for _, name := range registered {
		if name == "golang" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("golang analyzer not registered; got %v", registered)
	}
}

func TestAnalyzerGetFromConfig(t *testing.T) {
	cfg := map[string]string{"language": "golang"}
	ana, err := analyzer.Get("golang", cfg)
	if err != nil {
		t.Fatalf("get golang analyzer: %v", err)
	}
	if ana.Name() != "golang" {
		t.Errorf("expected name golang, got %s", ana.Name())
	}
}

func TestAnalyzerUnknown(t *testing.T) {
	_, err := analyzer.Get("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown analyzer")
	}
}
