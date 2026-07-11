package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/analyzer"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze source code at the given path",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(_ *cobra.Command, args []string) error {
	cfg := loadConfigOrExit()
	ana, err := analyzer.Get(cfg.Analyzer.Language, cfg.AnalyzerConfigMap())
	if err != nil {
		return fmt.Errorf("analyzer %q: %w", cfg.Analyzer.Language, err)
	}

	path := args[0]
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path not found: %s", path)
	}

	fmt.Printf("Analyzing %s with %s analyzer...\n", path, ana.Name())
	result, err := ana.Analyze(context.Background(), path)
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
