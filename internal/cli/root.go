package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const defaultConfigPath = "config.yaml"

var configPath string

// rootCmd is the base command for dataset-builder.
var rootCmd = &cobra.Command{
	Use:   "dataset-builder",
	Short: "Software Engineering Dataset Builder",
	Long: `RepoMiner dataset-builder collects, analyzes, and generates
software engineering datasets from multiple source control platforms.

Supported sources: GitHub, GitLab, Gitea, Gerrit, Local Git.`,
}

// Execute runs the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", defaultConfigPath, "path to config file")
}
