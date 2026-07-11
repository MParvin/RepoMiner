package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/generator"
	"github.com/mparvin/repo-miner/internal/storage/sqlite"
)

var (
	generateRepo   string
	generateOutput string
	generateFormat string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate dataset samples from collected data",
	RunE:  runGenerate,
}

func init() {
	generateCmd.Flags().StringVar(&generateRepo, "repo", "", "repository to generate from (owner/name)")
	generateCmd.Flags().StringVar(&generateOutput, "output", "", "output file or directory")
	generateCmd.Flags().StringVar(&generateFormat, "format", "jsonl", "output format: jsonl or huggingface")
	_ = generateCmd.MarkFlagRequired("repo")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(_ *cobra.Command, _ []string) error {
	cfg := loadConfigOrExit()
	ctx := context.Background()

	store, err := sqlite.Open(cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	ref, err := generator.ParseRepoRef(cfg.Source.Type, generateRepo)
	if err != nil {
		return err
	}

	output := generateOutput
	if output == "" {
		output = filepath.Join(cfg.Workspace.DatasetsDir, ref.Name+".jsonl")
	}

	gen, err := generator.New(cfg, store)
	if err != nil {
		return err
	}

	fmt.Printf("Generating dataset for %s...\n", generateRepo)
	count, err := gen.Generate(ctx, ref, output, generateFormat)
	if err != nil {
		return err
	}

	fmt.Printf("Generated %d samples -> %s\n", count, output)
	return nil
}
