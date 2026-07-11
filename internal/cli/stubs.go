package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/analyzer"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze source code at the given path",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		cfg := loadConfigOrExit()
		ana, err := analyzer.Get(cfg.Analyzer.Language, cfg.AnalyzerConfigMap())
		if err != nil {
			return fmt.Errorf("analyzer %q: %w", cfg.Analyzer.Language, err)
		}
		fmt.Printf("Using analyzer: %s\n", ana.Name())
		return fmt.Errorf("analyze not implemented (Phase 2)")
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate dataset samples from collected data",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg := loadConfigOrExit()
		if _, err := analyzer.Get(cfg.Analyzer.Language, cfg.AnalyzerConfigMap()); err != nil {
			return fmt.Errorf("analyzer %q: %w", cfg.Analyzer.Language, err)
		}
		return fmt.Errorf("generate not implemented (Phase 3)")
	},
}

var rankCmd = &cobra.Command{
	Use:   "rank",
	Short: "Rank repositories by quality and AI signals",
	RunE: func(_ *cobra.Command, _ []string) error {
		return fmt.Errorf("rank not implemented (Phase 4)")
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(rankCmd)
}
