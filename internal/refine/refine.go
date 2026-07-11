package refine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/llm"
)

// ReviewScore holds LLM evaluation scores for a sample.
type ReviewScore struct {
	TechnicalCorrectness float64 `json:"technical_correctness"`
	CodeQuality          float64 `json:"code_quality"`
	InstructionClarity   float64 `json:"instruction_clarity"`
	SolutionValidity     float64 `json:"solution_validity"`
	BestPractices        float64 `json:"best_practices"`
	Overall              float64 `json:"overall"`
	Action               string  `json:"action"` // keep | improve | reject
	Feedback             string  `json:"feedback,omitempty"`
}

// ReviewResult holds the outcome of reviewing one sample.
type ReviewResult struct {
	Original  domain.DatasetSample `json:"original"`
	Refined   *domain.DatasetSample `json:"refined,omitempty"`
	Score     ReviewScore          `json:"score"`
}

// Report is the machine-readable refinement report.
type Report struct {
	InputFile    string         `json:"input_file"`
	OutputFile   string         `json:"output_file"`
	Version      string         `json:"version"`
	Model        string         `json:"model"`
	ProcessedAt  time.Time      `json:"processed_at"`
	TotalSamples int            `json:"total_samples"`
	Kept         int            `json:"kept"`
	Improved     int            `json:"improved"`
	Rejected     int            `json:"rejected"`
	Results      []ReviewResult `json:"results"`
}

// Pipeline refines dataset samples using an LLM reviewer.
type Pipeline struct {
	LLM       llm.Provider
	Threshold float64
}

// NewPipeline creates a refinement pipeline.
func NewPipeline(provider llm.Provider, threshold float64) *Pipeline {
	if threshold == 0 {
		threshold = 6.0
	}
	return &Pipeline{LLM: provider, Threshold: threshold}
}

// Refine processes a JSONL dataset file and produces a refined version.
func (p *Pipeline) Refine(ctx context.Context, inputPath, outputPath string, limit int) (*Report, error) {
	samples, err := readJSONL(inputPath)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(samples) > limit {
		samples = samples[:limit]
	}

	version := fmt.Sprintf("v%d", time.Now().Unix())
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		outputPath = base + "-" + version + ext
	}

	report := &Report{
		InputFile:   inputPath,
		OutputFile:  outputPath,
		Version:     version,
		Model:       p.LLM.Name(),
		ProcessedAt: time.Now().UTC(),
		TotalSamples: len(samples),
	}

	var refined []domain.DatasetSample
	for _, sample := range samples {
		result, err := p.reviewSample(ctx, sample)
		if err != nil {
			result = ReviewResult{
				Original: sample,
				Score: ReviewScore{Action: "reject", Feedback: err.Error()},
			}
			report.Rejected++
		} else {
			switch result.Score.Action {
			case "keep":
				report.Kept++
				refined = append(refined, sample)
			case "improve":
				report.Improved++
				if result.Refined != nil {
					refined = append(refined, *result.Refined)
				} else {
					refined = append(refined, sample)
				}
			default:
				report.Rejected++
			}
		}
		report.Results = append(report.Results, result)
	}

	if err := writeJSONL(outputPath, refined); err != nil {
		return nil, err
	}

	reportPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "-report.json"
	reportBytes, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(reportPath, reportBytes, 0o644); err != nil {
		return nil, err
	}

	// Write version manifest
	manifestPath := filepath.Join(filepath.Dir(outputPath), "manifest.json")
	manifest := map[string]string{
		"input":    inputPath,
		"output":   outputPath,
		"report":   reportPath,
		"version":  version,
		"model":    p.LLM.Name(),
		"refined_at": time.Now().UTC().Format(time.RFC3339),
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(manifestPath, manifestBytes, 0o644)

	return report, nil
}

func (p *Pipeline) reviewSample(ctx context.Context, sample domain.DatasetSample) (ReviewResult, error) {
	prompt := fmt.Sprintf(`Review this software engineering training sample. Respond ONLY with valid JSON:
{
  "technical_correctness": <0-10>,
  "code_quality": <0-10>,
  "instruction_clarity": <0-10>,
  "solution_validity": <0-10>,
  "best_practices": <0-10>,
  "overall": <0-10>,
  "action": "keep" | "improve" | "reject",
  "feedback": "<brief feedback>",
  "improved_instruction": "<only if action is improve>",
  "improved_solution": "<only if action is improve>"
}

Sample:
Instruction: %s
Context: %s
Solution: %s`,
		sample.Instruction, truncate(sample.Context, 500), truncate(sample.Solution, 500))

	resp, err := p.LLM.Chat(ctx, []llm.Message{
		{Role: "system", Content: "You are a software engineering dataset quality reviewer. Output strict JSON only."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return ReviewResult{}, err
	}

	resp = extractJSON(resp)
	var scored struct {
		ReviewScore
		ImprovedInstruction string `json:"improved_instruction"`
		ImprovedSolution    string `json:"improved_solution"`
	}
	if err := json.Unmarshal([]byte(resp), &scored); err != nil {
		return ReviewResult{}, fmt.Errorf("parse LLM response: %w (response: %s)", err, truncate(resp, 200))
	}

	if scored.Overall == 0 {
		scored.Overall = (scored.TechnicalCorrectness + scored.CodeQuality +
			scored.InstructionClarity + scored.SolutionValidity + scored.BestPractices) / 5
	}

	if scored.Action == "" {
		if scored.Overall >= p.Threshold {
			scored.Action = "keep"
		} else if scored.Overall >= p.Threshold-2 {
			scored.Action = "improve"
		} else {
			scored.Action = "reject"
		}
	}

	result := ReviewResult{Original: sample, Score: scored.ReviewScore}
	if scored.Action == "improve" {
		improved := sample
		if scored.ImprovedInstruction != "" {
			improved.Instruction = scored.ImprovedInstruction
		}
		if scored.ImprovedSolution != "" {
			improved.Solution = scored.ImprovedSolution
		}
		result.Refined = &improved
	}
	return result, nil
}

func readJSONL(path string) ([]domain.DatasetSample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var samples []domain.DatasetSample
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var s domain.DatasetSample
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			continue
		}
		samples = append(samples, s)
	}
	return samples, scanner.Err()
}

func writeJSONL(path string, samples []domain.DatasetSample) error {
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

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
