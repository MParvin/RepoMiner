package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rankCmd = &cobra.Command{
	Use:   "rank",
	Short: "Rank repositories by quality and AI signals",
	RunE: func(_ *cobra.Command, _ []string) error {
		return fmt.Errorf("rank not implemented (Phase 4)")
	},
}

func init() {
	rootCmd.AddCommand(rankCmd)
}
