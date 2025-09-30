package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/config"
	"github.com/LoriKarikari/compak/internal/core/compose"
	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

var statusCmd = &cobra.Command{
	Use:   "status [package]",
	Short: "Show package status",
	Long: `Show the status of an installed package.

This command displays the current status of containers for the specified package
using the underlying compose command (docker compose ps or similar).`,
	Example: `  # Show status of nginx package
  compak status nginx

  # Show status with verbose output
  compak status nginx --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageName := args[0]

		composeClient, err := compose.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create compose client: %w", err)
		}

		stateDir, err := config.GetStateDir()
		if err != nil {
			return fmt.Errorf("failed to get state directory: %w", err)
		}

		client := pkg.NewClient(stateDir)
		manager := pkg.NewManager(client, composeClient, stateDir)

		if _, err := client.GetInstalledPackage(packageName); err != nil {
			return fmt.Errorf("package '%s' is not installed", packageName)
		}

		status, err := manager.Status(packageName)
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		fmt.Printf("Package: %s\n", packageName)
		fmt.Printf("Status:\n%s", status)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
