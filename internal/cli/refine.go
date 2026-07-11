package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/llm"
	"github.com/mparvin/repo-miner/internal/refine"
)

var (
	refineInput  string
	refineOutput string
	refineLimit  int
)

var refineCmd = &cobra.Command{
	Use:   "refine",
	Short: "Refine dataset samples using a local LLM reviewer",
	RunE:  runRefine,
}

func init() {
	refineCmd.Flags().StringVar(&refineInput, "input", "", "input JSONL dataset file")
	refineCmd.Flags().StringVar(&refineOutput, "output", "", "output refined JSONL file")
	refineCmd.Flags().IntVar(&refineLimit, "limit", 0, "max samples to process (0 = all)")
	_ = refineCmd.MarkFlagRequired("input")
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

	pipeline := refine.NewPipeline(provider, llmCfg.Threshold)
	fmt.Printf("Refining %s with %s (%s)...\n", refineInput, provider.Name(), llmCfg.Model)

	report, err := pipeline.Refine(context.Background(), refineInput, refineOutput, refineLimit)
	if err != nil {
		return err
	}

	fmt.Printf("Refinement complete (version %s):\n", report.Version)
	fmt.Printf("  Total:    %d\n", report.TotalSamples)
	fmt.Printf("  Kept:     %d\n", report.Kept)
	fmt.Printf("  Improved: %d\n", report.Improved)
	fmt.Printf("  Rejected: %d\n", report.Rejected)
	fmt.Printf("  Output:   %s\n", report.OutputFile)
	return nil
}
