package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mparvin/repo-miner/internal/core/domain"
)

// WriteJSONL writes samples to a JSONL file.
func WriteJSONL(path string, samples []domain.DatasetSample) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, s := range samples {
		if err := enc.Encode(s); err != nil {
			return err
		}
	}
	return nil
}

// HuggingFaceMeta describes a HuggingFace-compatible dataset.
type HuggingFaceMeta struct {
	Description string            `json:"description"`
	Features    map[string]string `json:"features"`
	Splits      map[string]int    `json:"splits"`
}

// WriteHuggingFace writes samples in HuggingFace-compatible layout under dir/name/.
func WriteHuggingFace(dir string, name string, samples []domain.DatasetSample) error {
	return WriteHuggingFaceInPlace(filepath.Join(dir, name), samples)
}

// WriteHuggingFaceInPlace writes train.jsonl and dataset_info.json directly in dir.
func WriteHuggingFaceInPlace(dir string, samples []domain.DatasetSample) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if err := WriteJSONL(filepath.Join(dir, "train.jsonl"), samples); err != nil {
		return err
	}

	meta := HuggingFaceMeta{
		Description: fmt.Sprintf("Software engineering dataset (%s)", filepath.Base(dir)),
		Features: map[string]string{
			"instruction": "string",
			"context":     "string",
			"solution":    "string",
			"metadata":    "dict",
		},
		Splits: map[string]int{"train": len(samples)},
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "dataset_info.json"), metaBytes, 0o644)
}
