package clean_test

import (
	"testing"

	"github.com/mparvin/repo-miner/internal/generator/clean"
)

func TestCleanText(t *testing.T) {
	in := "<p>Fix  timeout</p>\n\nissue"
	got := clean.Text(in)
	if got != "Fix timeout issue" {
		t.Errorf("got %q", got)
	}
}

func TestIsQuality(t *testing.T) {
	if !clean.IsQuality("Fix connection timeout issue", "old code", "new patch", 10, 8000) {
		t.Error("expected quality sample to pass")
	}
	if clean.IsQuality("short", "", "", 10, 8000) {
		t.Error("expected short instruction to fail")
	}
}
