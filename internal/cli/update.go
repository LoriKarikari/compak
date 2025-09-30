package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/core/index"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update compak pak index",
	Long: `Update the local pak index from GitHub.

This command pulls the latest paks from the compak repository.

Examples:
  compak update`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateIndex()
	},
}

func updateIndex() error {
	fmt.Println("Updating compak pak index...")

	client := index.NewClient()
	ctx := context.Background()

	if err := client.Update(ctx); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println("Pak index updated successfully")
	return nil
}
