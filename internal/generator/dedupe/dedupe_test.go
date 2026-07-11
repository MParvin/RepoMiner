package dedupe_test

import (
	"testing"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/generator/dedupe"
)

func TestFilter(t *testing.T) {
	samples := []domain.DatasetSample{
		{Instruction: "Fix bug A", Solution: "patch A"},
		{Instruction: "fix bug a", Solution: "patch a"},
		{Instruction: "Add feature B", Solution: "patch B"},
	}
	result := dedupe.Filter(samples)
	if len(result) != 2 {
		t.Errorf("expected 2 unique samples, got %d", len(result))
	}
}
