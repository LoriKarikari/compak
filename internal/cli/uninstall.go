package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/config"
	"github.com/LoriKarikari/compak/internal/core/compose"
	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [package]",
	Short: "Uninstall a package",
	Args:  cobra.ExactArgs(1),
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

		return manager.Stop(packageName)
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
