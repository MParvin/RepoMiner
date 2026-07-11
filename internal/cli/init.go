package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/storage/sqlite"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize workspace (config, folders, database)",
	Long: `Initialize the dataset-builder workspace:
  - Create a sample config.yaml (if absent)
  - Create workspace directories (data/, datasets/, repos/)
  - Open and migrate the SQLite database`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// 1. Write sample config
	if err := config.WriteSample(configPath); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	fmt.Printf("Config: %s\n", configPath)

	// 2. Load config and create directories
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := config.EnsureDirs(cfg); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}
	fmt.Printf("Directories: %s, %s, %s\n",
		cfg.Workspace.DataDir, cfg.Workspace.DatasetsDir, cfg.Workspace.ReposDir)

	// 3. Open database, migrate, ping
	store, err := sqlite.Open(cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	if err := store.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	fmt.Printf("Database: %s (connected)\n", cfg.Storage.Path)

	fmt.Println("\nInitialization complete.")
	fmt.Printf("  Provider:  %s\n", cfg.Source.Type)
	fmt.Printf("  Analyzer:  %s\n", cfg.Analyzer.Language)
	return nil
}

// loadConfigOrExit loads config and prints error to stderr on failure.
func loadConfigOrExit() *config.Config {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}
