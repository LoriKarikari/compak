package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/config"
	"github.com/LoriKarikari/compak/internal/core/compose"
	pkg "github.com/LoriKarikari/compak/internal/core/package"
)

var installCmd = &cobra.Command{
	Use:   "install [package]",
	Short: "Install a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageName := args[0]
		version, err := cmd.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("failed to get version flag: %w", err)
		}

		composeCmd, err := compose.DetectComposeCommand()
		if err != nil {
			return fmt.Errorf("failed to detect compose command: %w", err)
		}

		fmt.Printf("Using compose command: %s\n", composeCmd.String())

		stateDir, err := config.GetStateDir()
		if err != nil {
			return fmt.Errorf("failed to get state directory: %w", err)
		}

		client := pkg.NewClient(composeCmd, stateDir)

		mockPackage := pkg.Package{
			Name:        packageName,
			Version:     version,
			Description: "Mock package for testing",
			Parameters:  make(map[string]pkg.Param),
			Values:      make(map[string]string),
		}

		if version == "" {
			mockPackage.Version = "latest"
		}

		fmt.Printf("Installing package: %s@%s\n", packageName, mockPackage.Version)

		return client.Install(mockPackage, nil)
	},
}

func init() {
	installCmd.Flags().String("version", "", "package version to install")
	rootCmd.AddCommand(installCmd)
}
