package dedupe

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/mparvin/repo-miner/internal/core/domain"
)

// Filter removes duplicate samples using normalized content hashing.
func Filter(samples []domain.DatasetSample) []domain.DatasetSample {
	seen := make(map[string]struct{})
	result := make([]domain.DatasetSample, 0, len(samples))
	for _, s := range samples {
		key := hash(normalize(s.Instruction) + "|" + normalize(s.Solution))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, s)
	}
	return result
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}
