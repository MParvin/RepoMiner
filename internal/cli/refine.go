package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/dataset"
	"github.com/mparvin/repo-miner/internal/llm"
	"github.com/mparvin/repo-miner/internal/refine"
)

var (
	refineInput    string
	refineName     string
	refineKeywords string
	refineOutput   string
	refineLimit    int
)

var refineCmd = &cobra.Command{
	Use:   "refine",
	Short: "Refine dataset samples using a local LLM reviewer",
	Long: `Refine a dataset using an LLM quality reviewer.

Dataset directory naming (--name > --keywords > random):
  dataset-builder refine --name golang-gin
  dataset-builder refine --keywords gin
  dataset-builder refine --input datasets/my-set/dataset.jsonl`,
	RunE: runRefine,
}

func init() {
	refineCmd.Flags().StringVar(&refineInput, "input", "", "input JSONL dataset file")
	refineCmd.Flags().StringVar(&refineName, "name", "", "dataset name (resolves paths under datasets/<name>/)")
	refineCmd.Flags().StringVar(&refineKeywords, "keywords", "", "dataset name from keywords if --name omitted")
	refineCmd.Flags().StringVar(&refineOutput, "output", "", "override output refined JSONL file")
	refineCmd.Flags().IntVar(&refineLimit, "limit", 0, "max samples to process (0 = all)")
	rootCmd.AddCommand(refineCmd)
}

func runRefine(_ *cobra.Command, _ []string) error {
	cfg := loadConfigOrExit()

	llmCfg := cfg.LLM
	if llmCfg.Type == "" {
		llmCfg.Type = "ollama"
	}
	if llmCfg.Model == "" {
		llmCfg.Model = "mparvin/Supra-50M-f16:latest"
	}
	if llmCfg.BaseURL == "" {
		llmCfg.BaseURL = "http://localhost:11434"
	}

	provider, err := llm.New(llm.Config{
		Type:    llmCfg.Type,
		BaseURL: llmCfg.BaseURL,
		Model:   llmCfg.Model,
		APIKey:  llmCfg.APIKey,
	})
	if err != nil {
		return err
	}

	input := refineInput
	output := refineOutput
	var reportPath, manifestPath string
	var paths dataset.Paths

	if refineInput == "" || refineName != "" || refineKeywords != "" {
		var err error
		paths, err = dataset.EnsureDir(cfg.Workspace.DatasetsDir, refineName, refineKeywords)
		if err != nil {
			return fmt.Errorf("create dataset dir: %w", err)
		}
		if input == "" {
			input = paths.JSONL
		}
		if output == "" {
			output = paths.RefinedJSONL
		}
		reportPath = paths.Report
		manifestPath = paths.Manifest
	}

	if input == "" {
		return fmt.Errorf("specify --input, --name, or --keywords")
	}

	if output == "" {
		paths = dataset.NewPaths(cfg.Workspace.DatasetsDir, dataset.ResolveName(refineName, refineKeywords))
		output = paths.RefinedJSONL
	}

	pipeline := refine.NewPipeline(provider, llmCfg.Threshold)
	fmt.Printf("Refining %s (dataset: %q) with %s (%s)...\n", input, paths.Name, provider.Name(), llmCfg.Model)

	report, err := pipeline.Refine(context.Background(), input, output, reportPath, manifestPath, refineLimit)
	if err != nil {
		return err
	}

	fmt.Printf("Refinement complete (version %s):\n", report.Version)
	fmt.Printf("  Total:    %d\n", report.TotalSamples)
	fmt.Printf("  Kept:     %d\n", report.Kept)
	fmt.Printf("  Improved: %d\n", report.Improved)
	fmt.Printf("  Rejected: %d\n", report.Rejected)
	fmt.Printf("  Output:   %s\n", report.OutputFile)
	if paths.Dir != "" {
		fmt.Printf("  Dataset:  %s\n", paths.Dir)
	}
	return nil
}
