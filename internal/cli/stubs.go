package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/analyzer"
)

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
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(rankCmd)
}
