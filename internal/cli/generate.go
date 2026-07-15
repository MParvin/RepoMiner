package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/dataset"
	"github.com/mparvin/repo-miner/internal/generator"
	"github.com/mparvin/repo-miner/internal/storage/sqlite"
)

var (
	generateRepo     string
	generateName     string
	generateKeywords string
	generateOutput   string
	generateFormat   string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate dataset samples from collected data",
	Long: `Generate a dataset from collected repository data.

Dataset directory naming (--name > --keywords > random):
  dataset-builder generate --repo gin-gonic/gin --name golang-gin
  dataset-builder generate --repo gin-gonic/gin --keywords gin
  dataset-builder generate --repo gin-gonic/gin   # random directory name`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringVar(&generateRepo, "repo", "", "repository to generate from (owner/name)")
	generateCmd.Flags().StringVar(&generateName, "name", "", "dataset name (used for output files and directories)")
	generateCmd.Flags().StringVar(&generateKeywords, "keywords", "", "dataset name from keywords if --name omitted")
	generateCmd.Flags().StringVar(&generateOutput, "output", "", "override output file or directory path")
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
	var paths dataset.Paths
	if output == "" {
		var err error
		paths, err = dataset.EnsureDir(cfg.Workspace.DatasetsDir, generateName, generateKeywords)
		if err != nil {
			return fmt.Errorf("create dataset dir: %w", err)
		}
		output = paths.OutputPath(generateFormat)
	} else {
		paths = dataset.NewPaths(cfg.Workspace.DatasetsDir, dataset.ResolveName(generateName, generateKeywords))
	}

	gen, err := generator.New(cfg, store)
	if err != nil {
		return err
	}

	fmt.Printf("Generating dataset %q for %s...\n", paths.Name, generateRepo)
	count, err := gen.Generate(ctx, ref, output, generateFormat)
	if err != nil {
		return err
	}

	fmt.Printf("Generated %d samples -> %s\n", count, output)
	fmt.Printf("Dataset directory: %s\n", paths.Dir)
	return nil
}
