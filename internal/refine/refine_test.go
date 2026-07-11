package refine_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mparvin/repo-miner/internal/core/domain"
)

func TestJSONLSampleFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	sample := domain.DatasetSample{
		Instruction: "Fix connection timeout issue",
		Context:     "old code",
		Solution:    "new patch",
		Metadata:    map[string]string{"source": "issue"},
	}
	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed domain.DatasetSample
	if err := json.Unmarshal(content, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Instruction != sample.Instruction {
		t.Errorf("instruction mismatch: %s", parsed.Instruction)
	}
}
