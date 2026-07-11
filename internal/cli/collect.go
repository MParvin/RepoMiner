package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/collector"
	"github.com/mparvin/repo-miner/internal/storage/sqlite"
)

var collectRepo string

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect repository data from the configured provider",
	RunE:  runCollect,
}

func init() {
	collectCmd.Flags().StringVar(&collectRepo, "repo", "", "repository to collect (owner/name)")
	_ = collectCmd.MarkFlagRequired("repo")
	rootCmd.AddCommand(collectCmd)
}

func runCollect(_ *cobra.Command, _ []string) error {
	cfg := loadConfigOrExit()
	ctx := context.Background()

	store, err := sqlite.Open(cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	ref, err := collector.ParseRepoRef(cfg.Source.Type, collectRepo)
	if err != nil {
		return err
	}

	c, err := collector.New(cfg, store, ref)
	if err != nil {
		return err
	}

	fmt.Printf("Collecting %s via %s provider...\n", collectRepo, cfg.Source.Type)
	if err := c.Collect(ctx, ref); err != nil {
		return err
	}

	fmt.Println("Collection complete.")
	return nil
}
