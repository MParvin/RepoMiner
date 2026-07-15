package dataset_test

import (
	"strings"
	"testing"

	"github.com/mparvin/repo-miner/internal/dataset"
)

func TestSanitizeName(t *testing.T) {
	if got := dataset.SanitizeName("my go dataset!"); got != "my-go-dataset" {
		t.Errorf("got %q", got)
	}
}

func TestNewPaths(t *testing.T) {
	p := dataset.NewPaths("datasets", "golang-repos")
	if p.Dir != "datasets/golang-repos" {
		t.Errorf("dir: %s", p.Dir)
	}
	if p.JSONL != "datasets/golang-repos/dataset.jsonl" {
		t.Errorf("jsonl: %s", p.JSONL)
	}
}

func TestResolveNameExplicit(t *testing.T) {
	if got := dataset.ResolveName("my-set", "gin"); got != "my-set" {
		t.Errorf("explicit should win: %s", got)
	}
}

func TestResolveNameKeywords(t *testing.T) {
	if got := dataset.ResolveName("", "gin framework"); got != "gin-framework" {
		t.Errorf("keywords fallback: %s", got)
	}
}

func TestResolveNameRandom(t *testing.T) {
	got := dataset.ResolveName("", "")
	if !strings.HasPrefix(got, "dataset-") {
		t.Errorf("expected random prefix, got %q", got)
	}
}

func TestEnsureDir(t *testing.T) {
	dir := t.TempDir()
	paths, err := dataset.EnsureDir(dir, "", "my-keywords")
	if err != nil {
		t.Fatal(err)
	}
	if paths.Name != "my-keywords" {
		t.Errorf("name: %s", paths.Name)
	}
}
