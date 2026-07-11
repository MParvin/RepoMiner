package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("dataset-builder %s (phase 0 foundation)\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
