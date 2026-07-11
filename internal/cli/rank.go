package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/ranking"
	"github.com/mparvin/repo-miner/internal/storage/sqlite"
)

var rankCmd = &cobra.Command{
	Use:   "rank",
	Short: "Rank repositories by quality and AI signals",
	RunE:  runRank,
}

func init() {
	rootCmd.AddCommand(rankCmd)
}

func runRank(_ *cobra.Command, _ []string) error {
	cfg := loadConfigOrExit()
	ctx := context.Background()

	store, err := sqlite.Open(cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	engine := ranking.New(cfg, store)
	results, err := engine.RankAll(ctx)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("No repositories to rank. Run 'collect' first.")
		return nil
	}

	fmt.Printf("%-30s %6s\n", "Repository", "Score")
	fmt.Println(strings.Repeat("-", 38))
	for _, r := range results {
		fmt.Printf("%-30s %6d\n", r.Repository, r.TotalScore)
	}

	fmt.Println()
	for _, r := range results {
		fmt.Printf("=== %s (score: %d, AI: %d, Quality: %d) ===\n",
			r.Repository, r.TotalScore, r.AIScore, r.QualityScore)
		for _, reason := range r.Reasons {
			fmt.Printf("  %s: %s +%d\n", reason.Category, reason.Detail, reason.Points)
		}
		fmt.Println()
	}
	return nil
}
